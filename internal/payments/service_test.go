package payments

import (
	"errors"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/money"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func paymentTestService(t *testing.T) (*Service, *tests.TestApp, *time.Time) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("create PocketBase test app: %v", err)
	}
	t.Cleanup(app.Cleanup)
	now := time.Date(2026, 7, 25, 12, 0, 0, 0, time.UTC)
	cfg := config.Config{
		PaymentTTL:       5 * time.Minute,
		AmountQuarantine: 24 * time.Hour,
		UPIID:            "operator@bank",
		SliceUPIID:       "operator@slice",
		UPIPayeeName:     "PayGate",
	}
	service := NewService(app, cfg, nil)
	service.Now = func() time.Time { return now }
	service.SuffixStart = func() (int64, error) { return 1, nil }
	return service, app, &now
}

func TestPaymentAccountsAreIsolatedForCreationIdempotencyAndMatching(t *testing.T) {
	service, _, _ := paymentTestService(t)
	kotak, _, err := service.Create(CreateInput{AmountRupees: 100, PaymentAccount: "kotak", IdempotencyKey: "account-choice"})
	if err != nil {
		t.Fatal(err)
	}
	if kotak.Account != domain.PaymentAccountKotak {
		t.Fatalf("Kotak payment account = %q", kotak.Account)
	}
	if _, _, err := service.Create(CreateInput{AmountRupees: 100, PaymentAccount: "slice", IdempotencyKey: "account-choice"}); err == nil {
		t.Fatal("idempotency key reused across accounts without a conflict")
	}
	slice, _, err := service.Create(CreateInput{AmountRupees: 100, PaymentAccount: "slice"})
	if err != nil {
		t.Fatal(err)
	}
	response := CreateResponse(slice, service.Config)
	if response["paymentAccount"] != domain.PaymentAccountSlice || response["verificationMethod"] != "email" ||
		!strings.Contains(response["upiUri"].(string), "pa=operator%40slice") {
		t.Fatalf("Slice create response = %+v", response)
	}
	wrongRail, err := service.Match(domain.ParsedSMS{
		Account: domain.PaymentAccountKotak, AmountPaise: slice.PayablePaise, RRN: "222233334444",
	})
	if err != nil || wrongRail.Action != "unmatched" {
		t.Fatalf("Kotak evidence for Slice amount = %+v, %v", wrongRail, err)
	}
	matched, err := service.Match(domain.ParsedSMS{
		Account: domain.PaymentAccountSlice, AmountPaise: slice.PayablePaise, RRN: "222233334444",
	})
	if err != nil || matched.Action != "marked_paid" || matched.Payment.ID != slice.ID {
		t.Fatalf("Slice evidence = %+v, %v", matched, err)
	}
	storedKotak, _ := service.Get(kotak.ID)
	if storedKotak.Status != domain.StatusPending {
		t.Fatalf("Slice evidence changed Kotak payment to %s", storedKotak.Status)
	}
}

func TestPaytmQROnlyCreateResponseEncodesExactAmountWithoutTransactionNote(t *testing.T) {
	service, _, _ := paymentTestService(t)
	service.Config.PaytmUPIID = "merchant@paytm"
	payment, _, err := service.Create(CreateInput{AmountRupees: 1, PaymentAccount: "paytm"})
	if err != nil {
		t.Fatal(err)
	}
	response := CreateResponse(payment, service.Config)
	if response["paymentFlow"] != "qr_only" || response["verificationMethod"] != "notification" {
		t.Fatalf("Paytm response metadata = %+v", response)
	}
	uri, ok := response["upiUri"].(string)
	if !ok {
		t.Fatalf("Paytm response missing upiUri: %+v", response)
	}
	parsed, err := url.Parse(uri)
	if err != nil {
		t.Fatal(err)
	}
	q := parsed.Query()
	if parsed.Scheme != "upi" || parsed.Host != "pay" || q.Get("pa") != "merchant@paytm" || q.Get("am") != money.FormatPaise(payment.PayablePaise) || q.Get("cu") != "INR" {
		t.Fatalf("Paytm URI = %q", uri)
	}
	if q.Get("tn") != "" || q.Get("pn") != "" {
		t.Fatalf("Paytm QR-only URI contains unwanted metadata: %q", uri)
	}
}

func TestCreateGuardedSkipsReadinessGateForIdempotentReplay(t *testing.T) {
	service, _, _ := paymentTestService(t)
	input := CreateInput{AmountRupees: 25, PaymentAccount: "kotak", IdempotencyKey: "guarded-replay"}
	gateCalls := 0
	first, replayed, err := service.CreateGuarded(input, func(core.App) error {
		gateCalls++
		return nil
	})
	if err != nil || replayed || gateCalls != 1 {
		t.Fatalf("first guarded create = %+v replayed=%v calls=%d err=%v", first, replayed, gateCalls, err)
	}

	replay, replayed, err := service.CreateGuarded(input, func(core.App) error {
		t.Fatal("readiness gate must not run for an exact idempotency replay")
		return errors.New("unreachable")
	})
	if err != nil || !replayed || replay.ID != first.ID {
		t.Fatalf("guarded replay = %+v replayed=%v err=%v", replay, replayed, err)
	}

	before, err := service.App.CountRecords("payments")
	if err != nil {
		t.Fatal(err)
	}
	_, replayed, err = service.CreateGuarded(CreateInput{AmountRupees: 26, IdempotencyKey: "guarded-denied"}, func(core.App) error {
		return domain.New("PAYMENT_ACCOUNT_UNAVAILABLE", "verification unavailable", 503)
	})
	if err == nil || replayed {
		t.Fatalf("denied guarded create err=%v replayed=%v", err, replayed)
	}
	after, err := service.App.CountRecords("payments")
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("guard failure allocated a payment: before=%d after=%d", before, after)
	}
}

func TestCreateAllocatesAllNinetyNineSlotsAndExhausts(t *testing.T) {
	service, _, _ := paymentTestService(t)
	seen := make(map[int64]bool)
	for i := 0; i < 99; i++ {
		payment, replayed, err := service.Create(CreateInput{AmountRupees: 100})
		if err != nil {
			t.Fatalf("Create #%d: %v", i+1, err)
		}
		if replayed || payment.PayablePaise < 10001 || payment.PayablePaise > 10099 {
			t.Fatalf("Create #%d returned %+v", i+1, payment)
		}
		seen[payment.PayablePaise] = true
	}
	if len(seen) != 99 {
		t.Fatalf("allocated %d unique slots; want 99", len(seen))
	}
	_, _, err := service.Create(CreateInput{AmountRupees: 100})
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "AMOUNT_CAPACITY_EXHAUSTED" {
		t.Fatalf("exhausted Create() error = %v; want AMOUNT_CAPACITY_EXHAUSTED", err)
	}
}

func TestCreateIdempotencyAndExactPaymentMatching(t *testing.T) {
	service, _, now := paymentTestService(t)
	first, replayed, err := service.Create(CreateInput{
		AmountRupees:   100,
		ExternalID:     "order-1",
		Metadata:       map[string]any{"kind": "checkout"},
		IdempotencyKey: "idem-1",
	})
	if err != nil || replayed {
		t.Fatalf("first Create() = %+v, %v, %v", first, replayed, err)
	}
	replay, replayed, err := service.Create(CreateInput{
		AmountRupees:   100,
		ExternalID:     "order-1",
		Metadata:       map[string]any{"kind": "checkout"},
		IdempotencyKey: "idem-1",
	})
	if err != nil || !replayed || replay.ID != first.ID {
		t.Fatalf("replayed Create() = %+v, %v, %v", replay, replayed, err)
	}
	_, _, err = service.Create(CreateInput{AmountRupees: 101, IdempotencyKey: "idem-1"})
	var conflict *domain.Error
	if !errors.As(err, &conflict) || conflict.Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("conflicting Create() error = %v; want IDEMPOTENCY_CONFLICT", err)
	}

	matched, err := service.Match(domain.ParsedSMS{AmountPaise: first.PayablePaise, RRN: "123456789012"})
	if err != nil || matched.Action != "marked_paid" || matched.Payment.Status != domain.StatusPaid {
		t.Fatalf("exact Match() = %+v, %v", matched, err)
	}
	duplicate, err := service.Match(domain.ParsedSMS{AmountPaise: first.PayablePaise, RRN: "123456789012"})
	if err != nil || duplicate.Action != "duplicate_rrn" || duplicate.Payment.ID != first.ID {
		t.Fatalf("duplicate Match() = %+v, %v", duplicate, err)
	}

	// A freshly constructed service still reads the durable record state.
	restarted := NewService(service.App, service.Config, nil)
	restarted.Now = func() time.Time { return *now }
	persisted, err := restarted.Get(first.ID)
	if err != nil || persisted.Status != domain.StatusPaid || persisted.PayablePaise != first.PayablePaise {
		t.Fatalf("persisted payment = %+v, %v", persisted, err)
	}
}

func TestExpiredPaymentReceivesLateExactCredit(t *testing.T) {
	service, _, now := paymentTestService(t)
	payment, _, err := service.Create(CreateInput{AmountRupees: 50})
	if err != nil {
		t.Fatal(err)
	}
	*now = now.Add(6 * time.Minute)
	result, err := service.Match(domain.ParsedSMS{AmountPaise: payment.PayablePaise, RRN: "998877665544"})
	if err != nil || result.Action != "marked_late" || result.Payment.Status != domain.StatusLate {
		t.Fatalf("late Match() = %+v, %v", result, err)
	}
}

func TestConcurrentAllocationsRemainUnique(t *testing.T) {
	service, app, _ := paymentTestService(t)
	const workers = 20
	ids := make(chan string, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			payment, _, err := service.Create(CreateInput{AmountRupees: 200})
			if err != nil {
				errs <- err
				return
			}
			ids <- payment.ID
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent Create() error: %v", err)
	}
	seen := map[string]bool{}
	for id := range ids {
		seen[id] = true
	}
	if len(seen) != workers {
		t.Fatalf("concurrent allocation returned %d unique IDs; want %d", len(seen), workers)
	}
	records, err := app.FindRecordsByFilter("payments", "requested_amount = 20000", "created", 0, 0)
	if err != nil || len(records) != workers {
		t.Fatalf("stored concurrent payments = %d, %v; want %d", len(records), err, workers)
	}
}

func TestPublicPaymentRedactsEvidence(t *testing.T) {
	record := core.NewRecord(core.NewBaseCollection("payments"))
	record.Id = "payment-id"
	record.Set("requested_amount", 10000)
	record.Set("payable_amount", 10001)
	record.Set("status", "paid")
	payment := FromRecord(record)
	public := PublicPayment(payment)
	for _, forbidden := range []string{"rrn", "upiId", "payerName", "rawSms"} {
		if _, ok := public[forbidden]; ok {
			t.Errorf("public response exposed %q", forbidden)
		}
	}
	if public["payableAmount"] != "100.01" {
		t.Errorf("public payableAmount = %v; want 100.01", public["payableAmount"])
	}
}

func TestCreateResponseUsesMinimalUPIURI(t *testing.T) {
	payment := &domain.Payment{ID: "pay123", Account: domain.PaymentAccountKotak, RequestedPaise: 10000, PayablePaise: 10094, Status: domain.StatusPending}
	response := CreateResponse(payment, config.Config{DefaultPaymentAccount: "kotak", KotakUPIID: "operator@bank", KotakUPIPayeeName: "PayGate"})
	raw, ok := response["upiUri"].(string)
	if !ok {
		t.Fatalf("upiUri = %#v; want string", response["upiUri"])
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if query.Get("pa") != "operator@bank" || query.Get("pn") != "PayGate" || query.Get("am") != "100.94" || query.Get("cu") != "INR" {
		t.Fatalf("unexpected UPI query: %v", query)
	}
	if query.Has("tn") || query.Has("tr") {
		t.Fatalf("UPI URI must not include tn/tr: %s", raw)
	}
}

func TestCreateRejectsOverflowFromServiceInput(t *testing.T) {
	service, _, _ := paymentTestService(t)
	_, _, err := service.Create(CreateInput{AmountRupees: int64(^uint64(0)>>1)/100 + 1})
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "INVALID_AMOUNT" {
		t.Fatalf("overflow Create() error = %v", err)
	}
}

func TestAmountRemainsUnavailableUntilQuarantineEnds(t *testing.T) {
	service, _, now := paymentTestService(t)
	created := make([]*domain.Payment, 0, 99)
	for i := 0; i < 99; i++ {
		payment, _, err := service.Create(CreateInput{AmountRupees: 300})
		if err != nil {
			t.Fatalf("Create #%d: %v", i+1, err)
		}
		created = append(created, payment)
	}
	for _, payment := range created {
		if payment.PayablePaise%100 == 0 {
			t.Fatalf("allocator used .00 fingerprint: %d", payment.PayablePaise)
		}
		if payment.PayablePaise/100 != 300 {
			t.Fatalf("allocator spilled into another rupee: %d", payment.PayablePaise)
		}
	}

	*now = now.Add(6 * time.Minute) // all expired, but still quarantined
	if _, _, err := service.Create(CreateInput{AmountRupees: 300}); err == nil {
		t.Fatal("allocator reused an expired amount while it was still quarantined")
	}

	*now = now.Add(24*time.Hour + time.Minute) // beyond expires_at + quarantine
	payment, _, err := service.Create(CreateInput{AmountRupees: 300})
	if err != nil {
		t.Fatalf("allocator did not reuse a released amount after quarantine: %v", err)
	}
	if payment.PayablePaise != 30001 {
		t.Fatalf("reused amount = %d; deterministic allocator should reuse 30001", payment.PayablePaise)
	}
}

func TestPaidAmountUsesQuarantineBeforeReuse(t *testing.T) {
	service, _, now := paymentTestService(t)
	first, _, err := service.Create(CreateInput{AmountRupees: 400})
	if err != nil {
		t.Fatal(err)
	}
	if first.PayablePaise != 40001 {
		t.Fatalf("first amount = %d; want 40001", first.PayablePaise)
	}
	if _, err := service.Match(domain.ParsedSMS{AmountPaise: first.PayablePaise, RRN: "111122223333"}); err != nil {
		t.Fatal(err)
	}
	second, _, err := service.Create(CreateInput{AmountRupees: 400})
	if err != nil {
		t.Fatal(err)
	}
	if second.PayablePaise == first.PayablePaise {
		t.Fatal("paid amount was reused before quarantine expired")
	}

	*now = now.Add(24*time.Hour + 6*time.Minute) // beyond original expires_at + quarantine
	third, _, err := service.Create(CreateInput{AmountRupees: 400})
	if err != nil {
		t.Fatal(err)
	}
	if third.PayablePaise != first.PayablePaise {
		t.Fatalf("released paid amount = %d; want deterministic reuse of %d", third.PayablePaise, first.PayablePaise)
	}
}

func TestDuplicateRRNWithDifferentAmountIsRejected(t *testing.T) {
	service, _, _ := paymentTestService(t)
	first, _, err := service.Create(CreateInput{AmountRupees: 500})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Match(domain.ParsedSMS{AmountPaise: first.PayablePaise, RRN: "444455556666"}); err != nil {
		t.Fatal(err)
	}
	_, err = service.Match(domain.ParsedSMS{AmountPaise: first.PayablePaise + 1, RRN: "444455556666"})
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "RRN_AMOUNT_MISMATCH" {
		t.Fatalf("mismatched duplicate RRN error = %v; want RRN_AMOUNT_MISMATCH", err)
	}
}

func TestCreateRejectsOversizedIdentifiersAndMetadata(t *testing.T) {
	service, _, _ := paymentTestService(t)
	cases := []struct {
		name  string
		input CreateInput
		code  string
	}{
		{name: "external id", input: CreateInput{AmountRupees: 10, ExternalID: strings.Repeat("x", 256)}, code: "INVALID_EXTERNAL_ID"},
		{name: "idempotency key", input: CreateInput{AmountRupees: 10, IdempotencyKey: strings.Repeat("k", 256)}, code: "INVALID_IDEMPOTENCY_KEY"},
		{name: "metadata", input: CreateInput{AmountRupees: 10, Metadata: map[string]any{"blob": strings.Repeat("m", (1<<20)+1)}}, code: "INVALID_METADATA"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := service.Create(tc.input)
			var domainErr *domain.Error
			if !errors.As(err, &domainErr) || domainErr.Code != tc.code {
				t.Fatalf("error = %v; want %s", err, tc.code)
			}
		})
	}
}

func TestEarlyResolutionNeverShortensOriginalQuarantine(t *testing.T) {
	service, _, now := paymentTestService(t)
	payment, _, err := service.Create(CreateInput{AmountRupees: 321})
	if err != nil {
		t.Fatal(err)
	}
	original := payment.ReuseAfter

	paid, err := service.Match(domain.ParsedSMS{AmountPaise: payment.PayablePaise, RRN: "321321321321", OccurredAt: *now})
	if err != nil {
		t.Fatal(err)
	}
	if paid.Payment.ReuseAfter.Before(original) {
		t.Fatalf("paid reuse_after shortened: original=%s paid=%s", original, paid.Payment.ReuseAfter)
	}

	second, _, err := service.Create(CreateInput{AmountRupees: 322})
	if err != nil {
		t.Fatal(err)
	}
	originalSecond := second.ReuseAfter
	cancelled, err := service.Cancel(second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.ReuseAfter.Before(originalSecond) {
		t.Fatalf("cancelled reuse_after shortened: original=%s cancelled=%s", originalSecond, cancelled.ReuseAfter)
	}
}

func TestDelayedOnTimeEvidenceRemainsPaidAfterExpiry(t *testing.T) {
	service, _, now := paymentTestService(t)
	payment, _, err := service.Create(CreateInput{AmountRupees: 700})
	if err != nil {
		t.Fatal(err)
	}
	evidenceAt := now.Add(2 * time.Minute)
	*now = now.Add(8 * time.Minute)
	result, err := service.Match(domain.ParsedSMS{
		AmountPaise: payment.PayablePaise,
		RRN:         "700700700700",
		OccurredAt:  evidenceAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "marked_paid" || result.Payment.Status != domain.StatusPaid {
		t.Fatalf("delayed on-time evidence = %+v", result)
	}
	if !result.Payment.PaidAt.Equal(evidenceAt) {
		t.Fatalf("paidAt=%s want %s", result.Payment.PaidAt, evidenceAt)
	}
}

func TestManualMatchKeepsExactAmountAndRRNInvariants(t *testing.T) {
	service, _, now := paymentTestService(t)
	payment, _, err := service.Create(CreateInput{AmountRupees: 800})
	if err != nil {
		t.Fatal(err)
	}
	var matched *core.Record
	err = service.App.RunInTransaction(func(tx core.App) error {
		var action string
		var queued bool
		var matchErr error
		matched, action, queued, matchErr = service.ManualMatchInApp(tx, payment.ID, domain.ParsedSMS{
			AmountPaise: payment.PayablePaise,
			RRN:         "800800800800",
			OccurredAt:  *now,
		}, *now)
		if matchErr != nil {
			return matchErr
		}
		if action != "marked_paid" || !queued {
			t.Fatalf("action=%s queued=%v", action, queued)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if matched.GetString("status") != "paid" {
		t.Fatalf("status=%s", matched.GetString("status"))
	}

	other, _, err := service.Create(CreateInput{AmountRupees: 801})
	if err != nil {
		t.Fatal(err)
	}
	err = service.App.RunInTransaction(func(tx core.App) error {
		_, _, _, err := service.ManualMatchInApp(tx, other.ID, domain.ParsedSMS{
			AmountPaise: other.PayablePaise,
			RRN:         "800800800800",
		}, *now)
		return err
	})
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "RRN_ALREADY_ASSIGNED" {
		t.Fatalf("duplicate manual RRN error=%v", err)
	}
}

func TestCapacitySnapshotReportsBlockedFingerprintPools(t *testing.T) {
	service, _, _ := paymentTestService(t)
	for i := 0; i < 70; i++ {
		if _, _, err := service.Create(CreateInput{AmountRupees: 900}); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := service.Capacity()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Pools) != 1 {
		t.Fatalf("pools=%d", len(snapshot.Pools))
	}
	pool := snapshot.Pools[0]
	if pool.Blocked != 70 || pool.Available != 29 || pool.Level != "warning" {
		t.Fatalf("pool=%+v", pool)
	}
	if snapshot.WarningPools != 1 {
		t.Fatalf("warningPools=%d", snapshot.WarningPools)
	}
}

func TestManualMatchUsesEvidenceTimeForHistoricalReconciliation(t *testing.T) {
	service, _, now := paymentTestService(t)
	payment, _, err := service.Create(CreateInput{AmountRupees: 850})
	if err != nil {
		t.Fatal(err)
	}
	evidenceAt := payment.ExpiresAt.Add(time.Hour)
	*now = payment.ReuseAfter.Add(time.Hour)
	if _, err := service.ExpireDue(); err != nil {
		t.Fatal(err)
	}
	var matched *core.Record
	err = service.App.RunInTransaction(func(tx core.App) error {
		var action string
		var matchErr error
		matched, action, _, matchErr = service.ManualMatchInApp(tx, payment.ID, domain.ParsedSMS{
			AmountPaise: payment.PayablePaise,
			RRN:         "850850850850",
			OccurredAt:  evidenceAt,
		}, *now)
		if matchErr == nil && action != "marked_late" {
			t.Fatalf("action=%s", action)
		}
		return matchErr
	})
	if err != nil {
		t.Fatalf("historical evidence inside original quarantine was rejected: %v", err)
	}
	if matched.GetString("status") != "late" {
		t.Fatalf("status=%s", matched.GetString("status"))
	}
}

func TestManualMatchRejectsTransactionAfterFingerprintReuseBoundary(t *testing.T) {
	service, _, now := paymentTestService(t)
	payment, _, err := service.Create(CreateInput{AmountRupees: 851})
	if err != nil {
		t.Fatal(err)
	}
	*now = payment.ReuseAfter.Add(2 * time.Hour)
	err = service.App.RunInTransaction(func(tx core.App) error {
		_, _, _, err := service.ManualMatchInApp(tx, payment.ID, domain.ParsedSMS{
			AmountPaise: payment.PayablePaise,
			RRN:         "851851851851",
			OccurredAt:  payment.ReuseAfter.Add(time.Minute),
		}, *now)
		return err
	})
	var domainErr *domain.Error
	if !errors.As(err, &domainErr) || domainErr.Code != "PAYMENT_QUARANTINE_ELAPSED" {
		t.Fatalf("error=%v", err)
	}
}

func TestMatchAllowsSecondPrecisionBankTimestampNearCreation(t *testing.T) {
	service, _, now := paymentTestService(t)
	*now = time.Date(2026, 8, 1, 13, 35, 29, 819_000_000, time.UTC)
	payment, _, err := service.Create(CreateInput{AmountRupees: 860})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Match(domain.ParsedSMS{
		AmountPaise: payment.PayablePaise,
		RRN:         "860860860860",
		OccurredAt:  now.Truncate(time.Second),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "marked_paid" {
		t.Fatalf("action=%s", result.Action)
	}
}

func TestMatchRejectsEvidenceBeyondCreationTolerance(t *testing.T) {
	service, _, now := paymentTestService(t)
	payment, _, err := service.Create(CreateInput{AmountRupees: 861})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Match(domain.ParsedSMS{
		AmountPaise: payment.PayablePaise,
		RRN:         "861861861861",
		OccurredAt:  now.Add(-EvidenceTimestampTolerance - time.Millisecond),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "unmatched" || result.Payment != nil {
		t.Fatalf("result=%+v", result)
	}
}
