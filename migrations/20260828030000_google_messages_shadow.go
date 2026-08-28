package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	pbmigrations "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	pbmigrations.Register(func(app core.App) error {
		events, err := app.FindCollectionByNameOrId("relay_events")
		if err != nil {
			return err
		}
		status, ok := events.Fields.GetByName("processing_status").(*core.SelectField)
		if !ok || status == nil {
			return fmt.Errorf("relay_events.processing_status is not a select field")
		}
		if !containsSelectValue(status.Values, "shadow_observed") {
			status.Values = append(status.Values, "shadow_observed")
		}
		return app.Save(events)
	}, func(app core.App) error {
		events, err := app.FindCollectionByNameOrId("relay_events")
		if err != nil {
			return nil
		}
		status, ok := events.Fields.GetByName("processing_status").(*core.SelectField)
		if !ok || status == nil {
			return nil
		}
		values := status.Values[:0]
		for _, value := range status.Values {
			if value != "shadow_observed" {
				values = append(values, value)
			}
		}
		status.Values = values
		return app.Save(events)
	})
}
