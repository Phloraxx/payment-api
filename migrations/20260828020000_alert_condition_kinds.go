package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	pbmigrations "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	pbmigrations.Register(func(app core.App) error {
		alerts, err := app.FindCollectionByNameOrId("alerts")
		if err != nil {
			return err
		}
		kind, ok := alerts.Fields.GetByName("kind").(*core.SelectField)
		if !ok || kind == nil {
			return fmt.Errorf("alerts.kind is not a select field")
		}
		if !containsSelectValue(kind.Values, "relay_unavailable") {
			kind.Values = append(kind.Values, "relay_unavailable")
		}
		return app.Save(alerts)
	}, func(app core.App) error {
		alerts, err := app.FindCollectionByNameOrId("alerts")
		if err != nil {
			return nil
		}
		kind, ok := alerts.Fields.GetByName("kind").(*core.SelectField)
		if !ok || kind == nil {
			return nil
		}
		values := kind.Values[:0]
		for _, value := range kind.Values {
			if value != "relay_unavailable" {
				values = append(values, value)
			}
		}
		kind.Values = values
		return app.Save(alerts)
	})
}

func containsSelectValue(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
