package retention

import (
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

const redactedMarker = "[redacted after retention period]"

type Result struct {
	SMSEventsRedacted             int `json:"smsEventsRedacted"`
	ReconciliationEntriesRedacted int `json:"reconciliationEntriesRedacted"`
	AuditEventsDeleted            int `json:"auditEventsDeleted"`
}

type Service struct {
	App    core.App
	Config config.Config
	Now    func() time.Time
}

func NewService(app core.App, cfg config.Config) *Service {
	return &Service{App: app, Config: cfg, Now: time.Now}
}

func (s *Service) Run() (Result, error) {
	if !s.Config.RetentionEnabled {
		return Result{}, nil
	}
	now := s.now()
	result := Result{}
	err := s.App.RunInTransaction(func(tx core.App) error {
		smsCutoff := now.Add(-s.Config.SMSRawRetention)
		smsRecords, err := tx.FindRecordsByFilter(
			"sms_events",
			"((message_time != '' && message_time < {:cutoff}) || (message_time = '' && created < {:cutoff})) && body != {:redacted}",
			"created", 250, 0,
			dbx.Params{"cutoff": filterDate(smsCutoff), "redacted": redactedMarker},
		)
		if err != nil {
			return err
		}
		for _, record := range smsRecords {
			record.Set("body", redactedMarker)
			record.Set("raw_payload", nil)
			record.Set("sender", "")
			record.Set("upi_id", "")
			record.Set("payer_name", "")
			if err := tx.Save(record); err != nil {
				return err
			}
			result.SMSEventsRedacted++
		}

		reconciliationCutoff := now.Add(-s.Config.ReconciliationRawRetention)
		entries, err := tx.FindRecordsByFilter(
			"reconciliation_entries",
			"((transaction_time != '' && transaction_time < {:cutoff}) || (transaction_time = '' && created < {:cutoff})) && description != {:redacted}",
			"created", 250, 0,
			dbx.Params{"cutoff": filterDate(reconciliationCutoff), "redacted": redactedMarker},
		)
		if err != nil {
			return err
		}
		for _, record := range entries {
			record.Set("description", redactedMarker)
			record.Set("raw_row", nil)
			if err := tx.Save(record); err != nil {
				return err
			}
			result.ReconciliationEntriesRedacted++
		}

		auditCutoff := now.Add(-s.Config.AuditRetention)
		auditRecords, err := tx.FindRecordsByFilter(
			"audit_events", "occurred_at < {:cutoff}", "occurred_at", 250, 0,
			dbx.Params{"cutoff": filterDate(auditCutoff)},
		)
		if err != nil {
			return err
		}
		for _, record := range auditRecords {
			if err := tx.Delete(record); err != nil {
				return err
			}
			result.AuditEventsDeleted++
		}
		return nil
	})
	return result, err
}

func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}

func filterDate(t time.Time) string {
	value, err := types.ParseDateTime(t.UTC())
	if err != nil {
		return t.UTC().Format(time.RFC3339Nano)
	}
	return value.String()
}
