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

	for _, name := range []string{"payments", "sms_events", "webhook_deliveries"} {
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
}
