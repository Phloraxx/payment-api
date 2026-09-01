package operator

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/adminpayments"
	"github.com/Phloraxx/payment-api/internal/v4/payments"
	"github.com/Phloraxx/payment-api/internal/v4/profiles"
	"github.com/Phloraxx/payment-api/internal/v4/storage"
	"github.com/Phloraxx/payment-api/internal/v4/webhooks"
)

type operatorFixture struct {
	db       *storage.DB
	payments *payments.Service
	admin    *adminpayments.Service
	operator *Service
	now      *time.Time
}

func newOperatorFixture(t *testing.T) operatorFixture {
	t.Helper()
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "paygate.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	now := time.Date(2026, 9, 1, 5, 0, 0, 0, time.UTC)
	profileService := profiles.NewService(db)
	ctx := context.Background()
	if _, err := profileService.Upsert(ctx, profiles.UpsertInput{
		ID: "paytm", Label: "Paytm", UPIID: "paygate@paytm", PayeeName: "PayGate",
		Parser: "paytm_notification", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := profileService.Activate(ctx, "paytm"); err != nil {
		t.Fatal(err)
	}
	paymentService := payments.NewService(db)
	paymentService.Now = func() time.Time { return now }
	adminService := adminpayments.NewService(db)
	adminService.Now = func() time.Time { return now }
	operatorService := NewService(db)
	operatorService.Now = func() time.Time { return now }
	return operatorFixture{db: db, payments: paymentService, admin: adminService, operator: operatorService, now: &now}
}

func (f operatorFixture) create(t *testing.T, amount int64, idem string) payments.Payment {
	t.Helper()
	result, err := f.payments.Create(context.Background(), payments.CreateInput{
		RequestedAmountPaise: amount * 100, Name: "Sourav P Bijoy", ExternalID: "evt_test",
		IdempotencyScope: "operator-test", IdempotencyKey: idem,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.Payment
}
func TestOverviewUsesIndiaLocalDayAndShowsOperationalSummary(t *testing.T) {
	f := newOperatorFixture(t)
	paidPayment := f.create(t, 100, "paid")
	status := "paid"
	if _, err := f.admin.Edit(context.Background(), paidPayment.ID, adminpayments.EditInput{Status: &status}); err != nil {
		t.Fatal(err)
	}
	_ = f.create(t, 200, "pending")
	if _, err := f.db.SQL.Exec(`INSERT INTO relay_devices(id,name,public_key_pem,enabled,enrolled_at,last_seen_at,app_version)
		VALUES('device-1','Edge 60 Stylus','pem',1,?,?,?)`, f.now.Add(-time.Hour).UnixMilli(), f.now.Add(-time.Minute).UnixMilli(), "0.5.0"); err != nil {
		t.Fatal(err)
	}

	overview, err := f.operator.Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if overview.PaymentsToday != 2 || overview.PaidToday != 1 || overview.Pending != 1 {
		t.Fatalf("overview counts = %+v", overview)
	}
	if overview.CollectedTodayPaise != paidPayment.PayableAmountPaise {
		t.Fatalf("collected=%d want=%d", overview.CollectedTodayPaise, paidPayment.PayableAmountPaise)
	}
	if overview.ActiveProfile == nil || overview.ActiveProfile.ID != "paytm" {
		t.Fatalf("active profile=%+v", overview.ActiveProfile)
	}
	if !overview.Relay.Connected || overview.Relay.Name != "Edge 60 Stylus" || overview.Relay.LastSeenAt == nil {
		t.Fatalf("relay=%+v", overview.Relay)
	}
	if len(overview.Volume) != 7 || overview.Volume[6].Date != "2026-09-01" || overview.Volume[6].Payments != 1 {
		t.Fatalf("volume=%+v", overview.Volume)
	}
}
func TestActivityCombinesPaymentObservationAndWebhookEvents(t *testing.T) {
	f := newOperatorFixture(t)
	payment := f.create(t, 100, "activity")
	if _, err := f.db.SQL.Exec(`INSERT INTO relay_devices(id,name,public_key_pem,enabled,enrolled_at) VALUES('device-1','Phone','pem',1,?)`, f.now.UnixMilli()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.SQL.Exec(`INSERT INTO relay_events(id,device_id,source_event_id,package_name,posted_at,received_at,amount_hint_paise,status)
		VALUES('relay-1','device-1','source-1','com.paytm.business',?,?,?,?)`, f.now.UnixMilli(), f.now.UnixMilli(), payment.PayableAmountPaise, "matched"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.SQL.Exec(`INSERT INTO payment_observations(id,relay_event_id,source,collection_profile_id,amount_paise,payer_name,occurred_at,occurred_at_source,received_at,matched_payment_id,match_result)
		VALUES('obs-1','relay-1','paytm_notification','paytm',?,'Bijoy P',?,'notification_posted_at',?,?,'matched')`,
		payment.PayableAmountPaise, f.now.UnixMilli(), f.now.UnixMilli(), payment.ID); err != nil {
		t.Fatal(err)
	}
	entries, err := f.operator.Activity(context.Background(), 20)
	if err != nil {
		t.Fatal(err)
	}
	var sawPayment, sawObservation, sawWebhook bool
	for _, entry := range entries {
		switch entry.Kind {
		case "payment":
			sawPayment = true
		case "payment_detected":
			sawObservation = true
		case "webhook":
			sawWebhook = true
		}
	}
	if !sawPayment || !sawObservation || !sawWebhook {
		t.Fatalf("activity kinds missing: %+v", entries)
	}
}
func TestWebhookSettingsGenerateHideRotateAndApplyLive(t *testing.T) {
	f := newOperatorFixture(t)
	worker := webhooks.NewService(f.db, webhooks.Config{})
	settings := NewSettingsService(f.db, worker)

	current, err := settings.Webhook(context.Background())
	if err != nil || current.Enabled || current.SecretConfigured {
		t.Fatalf("initial settings=%+v err=%v", current, err)
	}
	configured, secret, err := settings.ConfigureWebhook(context.Background(), "https://example.com/paygate-hook", false)
	if err != nil {
		t.Fatal(err)
	}
	if !configured.Enabled || !configured.SecretConfigured || secret == "" {
		t.Fatalf("configured=%+v secret=%q", configured, secret)
	}
	if got := worker.ConfigSnapshot(); got.Endpoint != configured.Endpoint || got.Secret != secret {
		t.Fatalf("worker config=%+v", got)
	}
	ordinary, err := settings.Webhook(context.Background())
	if err != nil || !ordinary.SecretConfigured {
		t.Fatalf("ordinary settings=%+v err=%v", ordinary, err)
	}

	rotated, newSecret, err := settings.ConfigureWebhook(context.Background(), configured.Endpoint, true)
	if err != nil {
		t.Fatal(err)
	}
	if !rotated.Enabled || newSecret == "" || newSecret == secret {
		t.Fatalf("rotation secret old=%q new=%q settings=%+v", secret, newSecret, rotated)
	}
	if _, _, err := settings.ConfigureWebhook(context.Background(), "", false); err != nil {
		t.Fatal(err)
	}
	if worker.Enabled() {
		t.Fatal("worker remained enabled after webhook was disabled")
	}
}
