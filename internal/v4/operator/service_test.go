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
	if _, err := f.db.SQL.Exec(`INSERT INTO relay_devices(id,name,public_key_pem,enabled,enrolled_at,last_seen_at,last_heartbeat_at,
		notification_access,listener_connected,battery_optimization_exempt,background_restricted,foreground_service,app_version)
		VALUES('device-1','Edge 60 Stylus','pem',1,?,?,?,?,?,?,?,?,?)`,
		f.now.Add(-time.Hour).UnixMilli(), f.now.Add(-time.Minute).UnixMilli(), f.now.Add(-time.Minute).UnixMilli(),
		1, 1, 1, 0, 1, "0.5.0"); err != nil {
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
func TestOverviewIncludesAttentionAndLastObservation(t *testing.T) {
	f := newOperatorFixture(t)
	payment := f.create(t, 100, "attention")
	if _, err := f.db.SQL.Exec(`INSERT INTO relay_devices(id,name,public_key_pem,enabled,enrolled_at) VALUES('device-attention','Motorola Edge 60 Stylus','pem',1,?)`, f.now.Add(-time.Hour).UnixMilli()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.SQL.Exec(`INSERT INTO relay_events(id,device_id,source_event_id,package_name,posted_at,received_at,status)
		VALUES('relay-attention','device-attention','source-attention','com.google.android.apps.nbu.paisa.user',?,?, 'unmatched')`, f.now.Add(-time.Second).UnixMilli(), f.now.Add(-time.Second).UnixMilli()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.SQL.Exec(`INSERT INTO payment_observations(id,relay_event_id,source,collection_profile_id,amount_paise,occurred_at,occurred_at_source,received_at,match_result)
		VALUES('obs-attention','relay-attention','android_notification','paytm',?,?, 'notification_posted_at',?,'unmatched')`, payment.PayableAmountPaise, f.now.Add(-time.Second).UnixMilli(), f.now.Add(-time.Second).UnixMilli()); err != nil {
		t.Fatal(err)
	}

	overview, err := f.operator.Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if overview.UnmatchedToday != 1 || overview.ExpiringSoon != 1 {
		t.Fatalf("attention counts = unmatched %d expiring %d", overview.UnmatchedToday, overview.ExpiringSoon)
	}
	if overview.LastObservation == nil || overview.LastObservation.PackageName != "com.google.android.apps.nbu.paisa.user" ||
		overview.LastObservation.DeviceName != "Motorola Edge 60 Stylus" || overview.LastObservation.MatchResult != "unmatched" {
		t.Fatalf("last observation = %+v", overview.LastObservation)
	}
}

func TestOverviewCountsParserLevelAmbiguousRelayEvidence(t *testing.T) {
	f := newOperatorFixture(t)
	if _, err := f.db.SQL.Exec(`INSERT INTO relay_devices(id,name,public_key_pem,enabled,enrolled_at) VALUES('device-parser-ambiguous','Relay Phone','pem',1,?)`, f.now.Add(-time.Hour).UnixMilli()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.SQL.Exec(`INSERT INTO relay_events(id,device_id,source_event_id,package_name,posted_at,received_at,status,error)
		VALUES('relay-parser-ambiguous','device-parser-ambiguous','source-parser-ambiguous','com.google.android.apps.nbu.paisa.user',?,?, 'ambiguous','notification contains multiple monetary amounts')`, f.now.Add(-time.Second).UnixMilli(), f.now.Add(-time.Second).UnixMilli()); err != nil {
		t.Fatal(err)
	}

	overview, err := f.operator.Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if overview.UnmatchedToday != 1 {
		t.Fatalf("evidence to review = %d, want 1", overview.UnmatchedToday)
	}
}

func TestOverviewRelaySummaryUsesAnyHealthyEnabledDevice(t *testing.T) {
	f := newOperatorFixture(t)
	if _, err := f.db.SQL.Exec(`INSERT INTO relay_devices(
		id,name,public_key_pem,enabled,enrolled_at,last_seen_at,last_heartbeat_at,app_version,
		notification_access,listener_connected,battery_optimization_exempt,background_restricted,foreground_service) VALUES
		('stale','Stale phone','pem',1,?,?,?,?,1,1,0,0,1),
		('healthy','Healthy phone','pem',1,?,?,?,?,1,1,1,0,1)`,
		f.now.Add(-3*time.Hour).UnixMilli(), f.now.Add(-3*time.Hour).UnixMilli(), f.now.Add(-3*time.Hour).UnixMilli(), "0.6.1",
		f.now.Add(-3*time.Hour).UnixMilli(), f.now.Add(-time.Hour).UnixMilli(), f.now.Add(-time.Minute).UnixMilli(), "0.7.0"); err != nil {
		t.Fatal(err)
	}
	overview, err := f.operator.Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !overview.Relay.Connected || overview.Relay.EnabledDevices != 2 || overview.Relay.ConnectedDevices != 1 {
		t.Fatalf("relay summary = %+v", overview.Relay)
	}
	if overview.Relay.Name != "Healthy phone" || overview.Relay.AppVersion != "0.7.0" || overview.Relay.LastSeenAt == nil ||
		!overview.Relay.LastSeenAt.Equal(f.now.Add(-time.Hour)) {
		t.Fatalf("representative relay = %+v", overview.Relay)
	}
}

func TestOverviewRelaySummaryRequiresReadyCurrentHeartbeat(t *testing.T) {
	f := newOperatorFixture(t)
	if _, err := f.db.SQL.Exec(`INSERT INTO relay_devices(
		id,name,public_key_pem,enabled,enrolled_at,last_seen_at,last_heartbeat_at,app_version,
		notification_access,listener_connected,battery_optimization_exempt,background_restricted,foreground_service) VALUES
		('restricted','Restricted phone','pem',1,?,?,?,?,1,1,0,1,1),
		('healthy','Healthy phone','pem',1,?,?,?,?,1,1,1,0,1)`,
		f.now.Add(-time.Hour).UnixMilli(), f.now.Add(-time.Minute).UnixMilli(), f.now.Add(-time.Minute).UnixMilli(), "0.7.2",
		f.now.Add(-time.Hour).UnixMilli(), f.now.Add(-2*time.Minute).UnixMilli(), f.now.Add(-2*time.Minute).UnixMilli(), "0.7.2"); err != nil {
		t.Fatal(err)
	}
	overview, err := f.operator.Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !overview.Relay.Connected || overview.Relay.ConnectedDevices != 1 || overview.Relay.Name != "Healthy phone" {
		t.Fatalf("relay summary = %+v", overview.Relay)
	}

	if _, err := f.db.SQL.Exec(`UPDATE relay_devices SET battery_optimization_exempt=0 WHERE id='healthy'`); err != nil {
		t.Fatal(err)
	}
	overview, err = f.operator.Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if overview.Relay.Connected || overview.Relay.ConnectedDevices != 0 {
		t.Fatalf("unready current devices must not be reported online: %+v", overview.Relay)
	}
}
func TestOverviewRelaySummaryKeepsLegacyLastSeenAsDiagnosticOnly(t *testing.T) {
	f := newOperatorFixture(t)
	if _, err := f.db.SQL.Exec(`INSERT INTO relay_devices(id,name,public_key_pem,enabled,enrolled_at,last_seen_at,app_version)
		VALUES('legacy','Migrated phone','pem',1,?,?,?)`, f.now.Add(-time.Hour).UnixMilli(), f.now.Add(-time.Minute).UnixMilli(), "0.4.0"); err != nil {
		t.Fatal(err)
	}
	overview, err := f.operator.Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if overview.Relay.Connected || overview.Relay.ConnectedDevices != 0 || overview.Relay.Name != "Migrated phone" {
		t.Fatalf("legacy last-seen diagnostic = %+v", overview.Relay)
	}
}

func TestOverviewRelaySummaryReadinessMatrix(t *testing.T) {
	boolInt := func(value bool) int {
		if value {
			return 1
		}
		return 0
	}
	tests := []struct {
		name                 string
		heartbeat            bool
		offset               time.Duration
		notificationAccess   bool
		listenerConnected    bool
		batteryExempt        bool
		powerSaveMode        bool
		backgroundRestricted bool
		foregroundService    bool
		wantConnected        bool
	}{
		{name: "current healthy", heartbeat: true, offset: -2 * time.Minute, notificationAccess: true, listenerConnected: true, batteryExempt: true, foregroundService: true, wantConnected: true},
		{name: "battery restricted", heartbeat: true, offset: -2 * time.Minute, notificationAccess: true, listenerConnected: true, batteryExempt: false, foregroundService: true},
		{name: "listener disconnected", heartbeat: true, offset: -2 * time.Minute, notificationAccess: true, listenerConnected: false, batteryExempt: true, foregroundService: true},
		{name: "notification access missing", heartbeat: true, offset: -2 * time.Minute, notificationAccess: false, listenerConnected: true, batteryExempt: true, foregroundService: true},
		{name: "foreground service absent", heartbeat: true, offset: -2 * time.Minute, notificationAccess: true, listenerConnected: true, batteryExempt: true, foregroundService: false},
		{name: "background restricted", heartbeat: true, offset: -2 * time.Minute, notificationAccess: true, listenerConnected: true, batteryExempt: true, backgroundRestricted: true, foregroundService: true},
		{name: "stale heartbeat", heartbeat: true, offset: -61 * time.Minute, notificationAccess: true, listenerConnected: true, batteryExempt: true, foregroundService: true},
		{name: "future within clock tolerance", heartbeat: true, offset: 4 * time.Minute, notificationAccess: true, listenerConnected: true, batteryExempt: true, foregroundService: true, wantConnected: true},
		{name: "future beyond clock tolerance", heartbeat: true, offset: 6 * time.Minute, notificationAccess: true, listenerConnected: true, batteryExempt: true, foregroundService: true},
		{name: "power saver allowed when otherwise ready", heartbeat: true, offset: -2 * time.Minute, notificationAccess: true, listenerConnected: true, batteryExempt: true, powerSaveMode: true, foregroundService: true, wantConnected: true},
		{name: "migrated without heartbeat telemetry", offset: -2 * time.Minute},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newOperatorFixture(t)
			seenAt := f.now.Add(test.offset).UnixMilli()
			var heartbeat any
			if test.heartbeat {
				heartbeat = seenAt
			}
			var notificationAccess, listenerConnected, batteryExempt, powerSaveMode, backgroundRestricted, foregroundService any
			if test.heartbeat {
				notificationAccess = boolInt(test.notificationAccess)
				listenerConnected = boolInt(test.listenerConnected)
				batteryExempt = boolInt(test.batteryExempt)
				powerSaveMode = boolInt(test.powerSaveMode)
				backgroundRestricted = boolInt(test.backgroundRestricted)
				foregroundService = boolInt(test.foregroundService)
			}
			if _, err := f.db.SQL.Exec(`INSERT INTO relay_devices(
				id,name,public_key_pem,enabled,enrolled_at,last_seen_at,last_heartbeat_at,app_version,
				notification_access,listener_connected,battery_optimization_exempt,power_save_mode,background_restricted,foreground_service)
				VALUES('matrix','Matrix phone','pem',1,?,?,?,?,?,?,?,?,?,?)`,
				f.now.Add(-time.Hour).UnixMilli(), seenAt, heartbeat, "0.7.2",
				notificationAccess, listenerConnected, batteryExempt, powerSaveMode, backgroundRestricted, foregroundService); err != nil {
				t.Fatal(err)
			}
			overview, err := f.operator.Overview(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if overview.Relay.Connected != test.wantConnected || overview.Relay.ConnectedDevices != boolInt(test.wantConnected) {
				t.Fatalf("relay summary = %+v, want connected=%v", overview.Relay, test.wantConnected)
			}
		})
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
}
