package retention

import (
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

const (
	redactedMarker     = "[redacted after retention period]"
	retentionBatchSize = 250
)

type Result struct {
	SMSEventsRedacted             int `json:"smsEventsRedacted"`
	EmailEventsRedacted           int `json:"emailEventsRedacted"`
	ReconciliationEntriesRedacted int `json:"reconciliationEntriesRedacted"`
	AuditEventsDeleted            int `json:"auditEventsDeleted"`
}

type Service struct {
	App    core.App
	Config config.Config
	Now    func() time.Time
}

func NewService(app core.App, cfg config.Config) *Service {
	return &Service{App: app, Config: cfg, Now: time.Now}
}

func (s *Service) Run() (Result, error) {
	if !s.Config.RetentionEnabled {
		return Result{}, nil
	}
	now := s.now()
	result := Result{}

	for {
		count, err := s.redactSMSBatch(now.Add(-s.Config.SMSRawRetention))
		if err != nil {
			return result, err
		}
		result.SMSEventsRedacted += count
		if count < retentionBatchSize {
			break
		}
	}
	for {
		count, err := s.redactEmailBatch(now.Add(-s.Config.EmailRawRetention))
		if err != nil {
			return result, err
		}
		result.EmailEventsRedacted += count
		if count < retentionBatchSize {
			break
		}
	}
	for {
		count, err := s.redactReconciliationBatch(now.Add(-s.Config.ReconciliationRawRetention))
		if err != nil {
			return result, err
		}
		result.ReconciliationEntriesRedacted += count
		if count < retentionBatchSize {
			break
		}
	}
	for {
		count, err := s.deleteAuditBatch(now.Add(-s.Config.AuditRetention))
		if err != nil {
			return result, err
		}
		result.AuditEventsDeleted += count
		if count < retentionBatchSize {
			break
		}
	}
	return result, nil
}

func (s *Service) redactEmailBatch(cutoff time.Time) (int, error) {
	count := 0
	err := s.App.RunInTransaction(func(tx core.App) error {
		records, err := tx.FindRecordsByFilter(
			"email_events",
			"((message_time != '' && message_time < {:cutoff}) || (message_time = '' && created < {:cutoff})) && body != {:redacted}",
			"created", retentionBatchSize, 0,
			dbx.Params{"cutoff": filterDate(cutoff), "redacted": redactedMarker},
		)
		if err != nil {
			return err
		}
		for _, record := range records {
			record.Set("subject", redactedMarker)
			record.Set("body", redactedMarker)
			record.Set("raw_payload", nil)
			record.Set("envelope_sender", "")
			record.Set("recipient", "")
			record.Set("sender", "")
			record.Set("auth_result", "")
			record.Set("upi_id", "")
			record.Set("payer_name", "")
			if err := tx.Save(record); err != nil {
				return err
			}
		}
		count = len(records)
		return nil
	})
	return count, err
}

func (s *Service) redactSMSBatch(cutoff time.Time) (int, error) {
	count := 0
	err := s.App.RunInTransaction(func(tx core.App) error {
		records, err := tx.FindRecordsByFilter(
			"sms_events",
			"((message_time != '' && message_time < {:cutoff}) || (message_time = '' && created < {:cutoff})) && body != {:redacted}",
			"created", retentionBatchSize, 0,
			dbx.Params{"cutoff": filterDate(cutoff), "redacted": redactedMarker},
		)
		if err != nil {
			return err
		}
		for _, record := range records {
			record.Set("body", redactedMarker)
			record.Set("raw_payload", nil)
			record.Set("sender", "")
			record.Set("upi_id", "")
			record.Set("payer_name", "")
			if err := tx.Save(record); err != nil {
				return err
			}
		}
		count = len(records)
		return nil
	})
	return count, err
}

func (s *Service) redactReconciliationBatch(cutoff time.Time) (int, error) {
	count := 0
	err := s.App.RunInTransaction(func(tx core.App) error {
		records, err := tx.FindRecordsByFilter(
			"reconciliation_entries",
			"((transaction_time != '' && transaction_time < {:cutoff}) || (transaction_time = '' && created < {:cutoff})) && description != {:redacted}",
			"created", retentionBatchSize, 0,
			dbx.Params{"cutoff": filterDate(cutoff), "redacted": redactedMarker},
		)
		if err != nil {
			return err
		}
		for _, record := range records {
			record.Set("description", redactedMarker)
			record.Set("raw_row", nil)
			if err := tx.Save(record); err != nil {
				return err
			}
		}
		count = len(records)
		return nil
	})
	return count, err
}

func (s *Service) deleteAuditBatch(cutoff time.Time) (int, error) {
	count := 0
	err := s.App.RunInTransaction(func(tx core.App) error {
		records, err := tx.FindRecordsByFilter(
			"audit_events", "occurred_at < {:cutoff}", "occurred_at", retentionBatchSize, 0,
			dbx.Params{"cutoff": filterDate(cutoff)},
		)
		if err != nil {
			return err
		}
		for _, record := range records {
			if err := tx.Delete(record); err != nil {
				return err
			}
		}
		count = len(records)
		return nil
	})
	return count, err
}

func (s *Service) now() time.Time {
	if s.Now == nil {
		return time.Now().UTC()
	}
	return s.Now().UTC()
}

func filterDate(t time.Time) string {
	value, err := types.ParseDateTime(t.UTC())
	if err != nil {
		return t.UTC().Format(time.RFC3339Nano)
	}
	return value.String()
}
