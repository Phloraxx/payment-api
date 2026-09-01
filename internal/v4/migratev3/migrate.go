package migratev3

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

func Run(ctx context.Context, opts Options) (report Report, err error) {
	opts, err = normalizeOptions(opts)
	if err != nil {
		return report, err
	}
	now := time.Now().UTC()
	if opts.Now != nil {
		now = opts.Now().UTC()
	}
	source, err := openVerifiedSource(ctx, opts.SourceZIP)
	if err != nil {
		return report, err
	}
	defer source.Close()
	legacy, err := loadPayments(ctx, source.DB)
	if err != nil {
		return report, err
	}
	device, err := loadEnabledDevice(ctx, source.DB)
	if err != nil {
		return report, err
	}
	archivedOnly, err := archivedOnlyCounts(ctx, source.DB)
	if err != nil {
		return report, err
	}
	mapped, report, err := preflight(legacy, opts, now)
	if err != nil {
		return report, err
	}
	report.SourceSHA256, report.SourcePayments = source.SHA256, len(legacy)
	report.ArchivedOnly = archivedOnly
	db, err := storage.Open(ctx, opts.Destination)
	if err != nil {
		return report, fmt.Errorf("create v4 destination: %w", err)
	}
	committed := false
	defer func() {
		_ = db.Close()
		if err != nil || !committed {
			removeSQLiteFiles(opts.Destination)
		}
	}()
	if err = importAll(ctx, db, mapped, device, opts, now); err != nil {
		return report, err
	}
	if err = verifyDestination(ctx, db, len(mapped)); err != nil {
		return report, err
	}
	committed = true
	report.MigratedPayments = len(mapped)
	report.MigratedDevice = device != nil
	report.CompletedAt = now
	return report, nil
}
func normalizeOptions(opts Options) (Options, error) {
	var err error
	opts.SourceZIP = strings.TrimSpace(opts.SourceZIP)
	opts.Destination = strings.TrimSpace(opts.Destination)
	opts.ActiveProfile = strings.ToLower(strings.TrimSpace(opts.ActiveProfile))
	opts.KotakUPIID, opts.PaytmUPIID = strings.TrimSpace(opts.KotakUPIID), strings.TrimSpace(opts.PaytmUPIID)
	opts.KotakPayee, opts.PaytmPayee = strings.TrimSpace(opts.KotakPayee), strings.TrimSpace(opts.PaytmPayee)
	if opts.SourceZIP == "" || opts.Destination == "" {
		return opts, errors.New("source ZIP and destination are required")
	}
	if opts.ActiveProfile != "kotak" && opts.ActiveProfile != "paytm" {
		return opts, errors.New("active profile must be kotak or paytm")
	}
	for label, value := range map[string]string{"kotak UPI ID": opts.KotakUPIID, "paytm UPI ID": opts.PaytmUPIID} {
		if len(value) < 3 || len(value) > 255 || !strings.Contains(value, "@") {
			return opts, fmt.Errorf("%s is required and must look like a UPI ID", label)
		}
	}
	if opts.Destination, err = filepath.Abs(opts.Destination); err != nil {
		return opts, err
	}
	if _, statErr := os.Stat(opts.Destination); statErr == nil {
		return opts, errors.New("destination already exists")
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return opts, statErr
	}
	return opts, nil
}

func preflight(rows []legacyPayment, opts Options, now time.Time) ([]migratedPayment, Report, error) {
	report := Report{MigratedByStatus: map[string]int{}, MigratedByProfile: map[string]int{}}
	out := make([]migratedPayment, 0, len(rows))
	ids, idem := map[string]struct{}{}, map[string]struct{}{}
	activeReservations := map[string]string{}
	for _, row := range rows {
		mapped, filled, late, err := normalizePayment(row, opts, now)
		if err != nil {
			return nil, report, err
		}
		if _, exists := ids[mapped.ID]; exists {
			return nil, report, fmt.Errorf("duplicate legacy payment id %s", mapped.ID)
		}
		ids[mapped.ID] = struct{}{}
		if mapped.LegacyIdempotencyKey != "" {
			if _, exists := idem[mapped.LegacyIdempotencyKey]; exists {
				return nil, report, fmt.Errorf("duplicate legacy idempotency key on payment %s", mapped.ID)
			}
			idem[mapped.LegacyIdempotencyKey] = struct{}{}
		}
		if now.Before(mapped.ReuseAfter) {
			key := fmt.Sprintf("%s:%d", mapped.ProfileID, mapped.Payable)
			if previous, exists := activeReservations[key]; exists {
				return nil, report, fmt.Errorf("active reservation collision %s between %s and %s", key, previous, mapped.ID)
			}
			activeReservations[key] = mapped.ID
		}
		if filled {
			report.LegacyNamesFilled++
		}
		if late {
			report.LateNormalizedPaid++
		}
		if mapped.ExternalID != "" {
			report.EventIDsRecovered++
		}
		report.MigratedByStatus[mapped.Status]++
		report.MigratedByProfile[mapped.ProfileID]++
		out = append(out, mapped)
	}
	if report.LegacyNamesFilled > 0 {
		report.Warnings = append(report.Warnings, "historical payments without a trustworthy person name use Unknown (legacy)")
	}
	return out, report, nil
}
func verifyDestination(ctx context.Context, db *storage.DB, wantPayments int) error {
	var integrity string
	if err := db.SQL.QueryRowContext(ctx, `PRAGMA integrity_check`).Scan(&integrity); err != nil {
		return fmt.Errorf("destination integrity check: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(integrity), "ok") {
		return fmt.Errorf("destination integrity check returned %q", integrity)
	}
	rows, err := db.SQL.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return fmt.Errorf("destination foreign key check: %w", err)
	}
	defer rows.Close()
	if rows.Next() {
		return errors.New("destination has foreign key violations")
	}
	var payments, reservations int
	if err := db.SQL.QueryRowContext(ctx, `SELECT COUNT(*) FROM payments`).Scan(&payments); err != nil {
		return err
	}
	if err := db.SQL.QueryRowContext(ctx, `SELECT COUNT(*) FROM amount_reservations`).Scan(&reservations); err != nil {
		return err
	}
	if payments != wantPayments || reservations != wantPayments {
		return fmt.Errorf("destination count mismatch payments=%d reservations=%d want=%d", payments, reservations, wantPayments)
	}
	return nil
}

func removeSQLiteFiles(path string) {
	for _, suffix := range []string{"", "-wal", "-shm"} {
		_ = os.Remove(path + suffix)
	}
}
