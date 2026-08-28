package store

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/domain"
	_ "github.com/Phloraxx/payment-api/migrations"
	"github.com/pocketbase/pocketbase/tests"
)

func testDatabase(t *testing.T) (*tests.TestApp, Database) {
	t.Helper()
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Cleanup)
	return app, NewPocketBase(app)
}

func TestEvidenceRepositoriesRoundTrip(t *testing.T) {
	_, db := testDatabase(t)
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	var smsID, emailID string
	err := db.Write(context.Background(), func(uow UnitOfWork) error {
		sms := &domain.SMSEvent{Source: "manual", SourceEventID: "sms-1", Body: "credit", Account: domain.PaymentAccountKotak, MessageTime: now, ProcessingStatus: "received"}
		if err := uow.SMSEvents().Create(sms); err != nil {
			return err
		}
		sms.ProcessingStatus = "matched"
		sms.RRN = "123456789012"
		if err := uow.SMSEvents().Save(sms); err != nil {
			return err
		}
		email := &domain.EmailEvent{Source: "manual", SourceEventID: "mail-1", Sender: "bank@example.com", Account: domain.PaymentAccountSlice, MessageTime: now, ReceivedAt: now, ProcessingStatus: "received"}
		if err := uow.EmailEvents().Create(email); err != nil {
			return err
		}
		smsID, emailID = sms.ID, email.ID
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if smsID == "" || emailID == "" {
		t.Fatal("repository did not assign record ids")
	}
	if err := db.View(context.Background(), func(uow UnitOfWork) error {
		sms, err := uow.SMSEvents().FindBySourceEvent("manual", "sms-1")
		if err != nil {
			return err
		}
		if sms.ID != smsID || sms.ProcessingStatus != "matched" || sms.RRN != "123456789012" {
			t.Fatalf("sms=%+v", sms)
		}
		email, err := uow.EmailEvents().FindBySourceEvent("manual", "mail-1")
		if err != nil {
			return err
		}
		if email.ID != emailID || email.Account != domain.PaymentAccountSlice {
			t.Fatalf("email=%+v", email)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestUnitOfWorkRollsBackEvidenceAndReview(t *testing.T) {
	app, db := testDatabase(t)
	sentinel := errors.New("rollback")
	err := db.Write(context.Background(), func(uow UnitOfWork) error {
		sms := &domain.SMSEvent{Source: "manual", SourceEventID: "rollback-sms", Body: "credit", Account: domain.PaymentAccountKotak, ProcessingStatus: "received", MessageTime: time.Now().UTC()}
		if err := uow.SMSEvents().Create(sms); err != nil {
			return err
		}
		review := &domain.ReviewCase{Kind: "unmatched", Status: "open", Severity: "warning", SMSEventID: sms.ID, Reason: "rollback test", OpenedAt: time.Now().UTC()}
		if err := uow.Reviews().Create(review); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("error=%v", err)
	}
	if count, _ := app.CountRecords("sms_events"); count != 0 {
		t.Fatalf("sms count=%d after rollback", count)
	}
	if count, _ := app.CountRecords("review_cases"); count != 0 {
		t.Fatalf("review count=%d after rollback", count)
	}
}

func TestReviewRepositoryFindsEvidenceCaseAndCountsOpen(t *testing.T) {
	_, db := testDatabase(t)
	var smsID, reviewID string
	if err := db.Write(context.Background(), func(uow UnitOfWork) error {
		sms := &domain.SMSEvent{Source: "manual", SourceEventID: "review-sms", Body: "credit", Account: domain.PaymentAccountKotak, ProcessingStatus: "received", MessageTime: time.Now().UTC()}
		if err := uow.SMSEvents().Create(sms); err != nil {
			return err
		}
		smsID = sms.ID
		review := &domain.ReviewCase{Kind: "unmatched", Status: "open", Severity: "warning", SMSEventID: sms.ID, Reason: "needs review", OpenedAt: time.Now().UTC()}
		if err := uow.Reviews().Create(review); err != nil {
			return err
		}
		reviewID = review.ID
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.View(context.Background(), func(uow UnitOfWork) error {
		review, err := uow.Reviews().FindByEvidence(smsID, "", "")
		if err != nil {
			return err
		}
		if review.ID != reviewID {
			t.Fatalf("review id=%s want %s", review.ID, reviewID)
		}
		count, err := uow.Reviews().OpenCount()
		if err != nil {
			return err
		}
		if count != 1 {
			t.Fatalf("open count=%d", count)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRelayDeviceAndEventRoundTrip(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	db := NewPocketBase(app)
	now := time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
	var device *domain.RelayDevice
	if err := db.Write(context.Background(), func(uow UnitOfWork) error {
		device = &domain.RelayDevice{DeviceID: strings.Repeat("a", 64), Name: "Phone", PublicKeyPEM: "key", Enabled: true, EnrolledAt: now, LastSeenAt: now}
		if err := uow.Relay().Create(device); err != nil {
			return err
		}
		event := &domain.RelayEvent{DeviceRecordID: device.ID, EventID: strings.Repeat("b", 64), Kind: "notification", AppPackage: "com.paytm.business", ProcessingStatus: "received", CapturedAt: now}
		return uow.RelayEvents().Create(event)
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.View(context.Background(), func(uow UnitOfWork) error {
		stored, err := uow.Relay().FindByDeviceID(device.DeviceID)
		if err != nil {
			return err
		}
		if stored.ID != device.ID || stored.Name != "Phone" || !stored.Enabled {
			t.Fatalf("stored device=%+v", stored)
		}
		event, err := uow.RelayEvents().FindByDeviceEvent(device.ID, strings.Repeat("b", 64))
		if err != nil {
			return err
		}
		if event.ProcessingStatus != "received" || event.DeviceRecordID != device.ID {
			t.Fatalf("event=%+v", event)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRelayEventRollsBackWithUnitOfWork(t *testing.T) {
	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()
	db := NewPocketBase(app)
	rollback := errors.New("rollback relay")
	err = db.Write(context.Background(), func(uow UnitOfWork) error {
		device := &domain.RelayDevice{DeviceID: strings.Repeat("c", 64), Name: "Phone", PublicKeyPEM: "key", Enabled: true}
		if err := uow.Relay().Create(device); err != nil {
			return err
		}
		if err := uow.RelayEvents().Create(&domain.RelayEvent{DeviceRecordID: device.ID, EventID: strings.Repeat("d", 64), Kind: "notification", AppPackage: "com.paytm.business", ProcessingStatus: "received"}); err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("err=%v", err)
	}
	if count, _ := app.CountRecords("relay_devices"); count != 0 {
		t.Fatalf("devices=%d", count)
	}
	if count, _ := app.CountRecords("relay_events"); count != 0 {
		t.Fatalf("events=%d", count)
	}
}
