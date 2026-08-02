package migrations

import (
	"database/sql"

	"github.com/pocketbase/pocketbase/core"
	pbmigrations "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	pbmigrations.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		orders, err := findOrCreateRazorpayTestOrders(app, users.Id)
		if err != nil {
			return err
		}
		_, err = findOrCreateRazorpayTestEvents(app, orders.Id)
		return err
	}, func(app core.App) error {
		for _, name := range []string{"razorpay_test_events", "razorpay_test_orders"} {
			collection, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				if err == sql.ErrNoRows {
					continue
				}
				return err
			}
			if err := app.Delete(collection); err != nil {
				return err
			}
		}
		return nil
	})
}

func findOrCreateRazorpayTestOrders(app core.App, usersID string) (*core.Collection, error) {
	if collection, err := app.FindCollectionByNameOrId("razorpay_test_orders"); err == nil {
		return collection, nil
	}
	collection := core.NewBaseCollection("razorpay_test_orders")
	lockDomainWrites(collection)
	collection.Fields.Add(
		&core.NumberField{Name: "amount", OnlyInt: true, Min: types.Pointer(float64(100)), Required: true},
		&core.TextField{Name: "currency", Max: 3, Required: true},
		&core.SelectField{Name: "status", Values: []string{
			"creating", "create_failed", "created", "verification_pending", "authorized", "captured", "failed", "partially_refunded", "refunded",
		}, Required: true},
		&core.TextField{Name: "external_id", Max: 255},
		&core.TextField{Name: "idempotency_key", Max: 255, Required: true},
		&core.TextField{Name: "razorpay_order_id", Max: 64},
		&core.TextField{Name: "razorpay_payment_id", Max: 64},
		&core.TextField{Name: "provider_status", Max: 64},
		&core.TextField{Name: "payment_method", Max: 64},
		&core.NumberField{Name: "amount_refunded", OnlyInt: true, Min: types.Pointer(float64(0))},
		&core.TextField{Name: "error", Max: 4096},
		&core.RelationField{Name: "created_by", CollectionId: usersID, MaxSelect: 1},
		&core.DateField{Name: "created_at", Required: true},
		&core.DateField{Name: "signature_verified_at"},
		&core.DateField{Name: "captured_at"},
		&core.DateField{Name: "failed_at"},
		&core.DateField{Name: "last_synced_at"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	collection.AddIndex("idx_rzp_test_idempotency", true, "idempotency_key", "")
	collection.AddIndex("idx_rzp_test_order", true, "razorpay_order_id", "razorpay_order_id != ''")
	collection.AddIndex("idx_rzp_test_payment", true, "razorpay_payment_id", "razorpay_payment_id != ''")
	collection.AddIndex("idx_rzp_test_status", false, "status,created_at", "")
	if err := app.Save(collection); err != nil {
		return nil, err
	}
	return collection, nil
}

func findOrCreateRazorpayTestEvents(app core.App, ordersID string) (*core.Collection, error) {
	if collection, err := app.FindCollectionByNameOrId("razorpay_test_events"); err == nil {
		return collection, nil
	}
	collection := core.NewBaseCollection("razorpay_test_events")
	lockDomainWrites(collection)
	collection.Fields.Add(
		&core.TextField{Name: "event_id", Max: 128, Required: true},
		&core.TextField{Name: "event_type", Max: 128, Required: true},
		&core.RelationField{Name: "test_order", CollectionId: ordersID, MaxSelect: 1},
		&core.TextField{Name: "razorpay_order_id", Max: 64},
		&core.TextField{Name: "razorpay_payment_id", Max: 64},
		&core.SelectField{Name: "status", Values: []string{"processed", "ignored", "failed"}, Required: true},
		&core.TextField{Name: "payload_hash", Max: 64, Required: true},
		&core.DateField{Name: "provider_created_at"},
		&core.DateField{Name: "received_at", Required: true},
		&core.TextField{Name: "error", Max: 4096},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	collection.AddIndex("idx_rzp_test_event_id", true, "event_id", "")
	collection.AddIndex("idx_rzp_test_event_order", false, "test_order,received_at", "test_order != ''")
	collection.AddIndex("idx_rzp_test_event_type", false, "event_type,received_at", "")
	if err := app.Save(collection); err != nil {
		return nil, err
	}
	return collection, nil
}
