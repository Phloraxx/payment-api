package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	pbmigrations "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	pbmigrations.Register(func(app core.App) error {
		devices, err := app.FindCollectionByNameOrId("relay_devices")
		if err != nil {
			return err
		}
		for _, name := range []string{
			"power_health_reported",
			"battery_optimization_exempt",
			"power_save_mode",
			"background_restricted",
			"foreground_service_active",
		} {
			if devices.Fields.GetByName(name) == nil {
				devices.Fields.Add(&core.BoolField{Name: name})
			}
		}
		return app.Save(devices)
	}, func(app core.App) error {
		devices, err := app.FindCollectionByNameOrId("relay_devices")
		if err != nil {
			return nil
		}
		for _, name := range []string{
			"power_health_reported",
			"battery_optimization_exempt",
			"power_save_mode",
			"background_restricted",
			"foreground_service_active",
		} {
			devices.Fields.RemoveByName(name)
		}
		return app.Save(devices)
	})
}
