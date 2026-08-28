package migrations

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/tests"
)

func TestDomainCollectionsOnlyExposeReadsToOperatorUsers(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	for _, name := range []string{"payments", "sms_events", "email_events", "notification_events", "webhook_deliveries", "audit_events", "review_cases", "reconciliation_runs", "reconciliation_entries", "alerts", "refunds", "razorpay_test_orders", "razorpay_test_events"} {
		collection, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			t.Fatalf("find %s: %v", name, err)
		}
		for ruleName, rule := range map[string]*string{"list": collection.ListRule, "view": collection.ViewRule} {
			if rule == nil || !strings.Contains(*rule, "@request.auth.collectionName = 'users'") {
				t.Errorf("%s %s rule = %v; expected users-only auth restriction", name, ruleName, rule)
			}
		}
		if collection.CreateRule != nil || collection.UpdateRule != nil || collection.DeleteRule != nil {
			t.Errorf("%s direct write rules must remain locked", name)
		}
	}

	payments, err := app.FindCollectionByNameOrId("payments")
	if err != nil {
		t.Fatal(err)
	}
	if payments.Fields.GetByName("resolved_at") == nil {
		t.Fatal("payments.resolved_at migration field is missing")
	}
	if payments.Fields.GetByName("payment_account") == nil {
		t.Fatal("payments.payment_account migration field is missing")
	}
	if payments.Fields.GetByName("evidence_source") == nil || payments.Fields.GetByName("evidence_reference") == nil {
		t.Fatal("payments notification evidence fields are missing")
	}
	for _, name := range []string{"sms_events", "email_events"} {
		collection, err := app.FindCollectionByNameOrId(name)
		if err != nil || collection.Fields.GetByName("payment_account") == nil {
			t.Fatalf("%s.payment_account migration field is missing", name)
		}
	}
	reviews, err := app.FindCollectionByNameOrId("review_cases")
	if err != nil {
		t.Fatal(err)
	}
	if reviews.Fields.GetByName("reconciliation_entry") == nil {
		t.Fatal("review_cases.reconciliation_entry migration field is missing")
	}
	if reviews.Fields.GetByName("email_event") == nil {
		t.Fatal("review_cases.email_event migration field is missing")
	}
	relayDevices, err := app.FindCollectionByNameOrId("relay_devices")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"power_health_reported", "battery_optimization_exempt", "power_save_mode", "background_restricted", "foreground_service_active"} {
		if relayDevices.Fields.GetByName(name) == nil {
			t.Fatalf("relay_devices.%s migration field is missing", name)
		}
	}
}
