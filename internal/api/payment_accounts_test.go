package api

import (
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/androidrelay"
	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/paymentemail"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestPaytmReadinessFailsClosedWithoutActiveRelay(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	cfg := config.Config{PaytmUPIID: "merchant@paytm", AndroidRelayStaleAfter: time.Hour}
	relay := androidrelay.NewService(app, nil)
	apiService := &API{Config: cfg, AndroidRelay: relay}

	options, err := apiService.paymentAccountOptions()
	if err != nil {
		t.Fatal(err)
	}
	var paytm paymentAccountOption
	for _, option := range options {
		if option.ID == "paytm" {
			paytm = option
		}
	}
	if paytm.ID != "paytm" || paytm.Ready || paytm.Flow != "qr_only" || paytm.UnavailableReason == "" {
		t.Fatalf("paytm option = %+v", paytm)
	}
	if err := apiService.ensurePaymentAccountReady("paytm"); err == nil {
		t.Fatal("expected unavailable Paytm account to fail closed")
	}
}

func TestPaytmReadinessAcceptsHealthyHeartbeat(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	cfg := config.Config{PaytmUPIID: "merchant@paytm", AndroidRelayStaleAfter: time.Hour}
	relay := androidrelay.NewService(app, nil)
	relay.Now = func() time.Time { return time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC) }
	collection, _ := app.FindCollectionByNameOrId("relay_devices")
	device := core.NewRecord(collection)
	device.Set("device_id", "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
	device.Set("name", "Phone")
	device.Set("public_key_pem", "test-key")
	device.Set("enabled", true)
	if err := app.Save(device); err != nil {
		t.Fatal(err)
	}
	if _, err := relay.Heartbeat(device, androidrelay.HeartbeatInput{
		SchemaVersion: 1, NotificationAccess: true, ListenerConnected: true,
	}); err != nil {
		t.Fatal(err)
	}
	apiService := &API{Config: cfg, AndroidRelay: relay}
	if err := apiService.ensurePaymentAccountReady("paytm"); err != nil {
		t.Fatalf("active relay should make Paytm ready: %v", err)
	}
}

func TestPaytmReadinessAcceptsMigratedLegacyGrace(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	now := time.Date(2026, 8, 28, 1, 0, 0, 0, time.UTC)
	cfg := config.Config{PaytmUPIID: "merchant@paytm", AndroidRelayStaleAfter: time.Hour}
	relay := androidrelay.NewService(app, nil)
	relay.Now = func() time.Time { return now }
	collection, _ := app.FindCollectionByNameOrId("relay_devices")
	device := core.NewRecord(collection)
	device.Set("device_id", "abababababababababababababababababababababababababababababababab")
	device.Set("name", "Migrated v0.2 phone")
	device.Set("public_key_pem", "test-key")
	device.Set("enabled", true)
	device.Set("last_seen_at", now.Add(-24*time.Hour))
	device.Set("heartbeat_grace_until", now.Add(time.Hour))
	if err := app.Save(device); err != nil {
		t.Fatal(err)
	}
	apiService := &API{Config: cfg, AndroidRelay: relay}
	if err := apiService.ensurePaymentAccountReady("paytm"); err != nil {
		t.Fatalf("migrated relay should remain ready during bounded heartbeat grace: %v", err)
	}
}

func TestSliceReadinessRequiresEmailEvidence(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	cfg := config.Config{SliceUPIID: "operator@slice"}
	apiService := &API{Config: cfg}
	if err := apiService.ensurePaymentAccountReady("slice"); err == nil {
		t.Fatal("Slice without enabled email evidence must fail closed")
	}
	cfg.EmailEvidenceEnabled = true
	apiService.Config = cfg
	apiService.Email = paymentemail.NewService(app, nil, "noreply@slice.bank.in", "mx.cloudflare.net")
	if err := apiService.ensurePaymentAccountReady("slice"); err != nil {
		t.Fatalf("enabled Slice email evidence should be ready: %v", err)
	}
}
