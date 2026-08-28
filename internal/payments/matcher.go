package payments

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// MatchEvidenceInApp is the single automatic payment matcher used by every
// trusted evidence adapter. Source-specific parsing and authentication happen
// before this boundary; payment invariants are enforced only here.
func (s *Service) MatchEvidenceInApp(tx core.App, evidence domain.Evidence, now time.Time) (*core.Record, domain.MatchOutcome, bool, error) {
	now = now.UTC()
	account, _, err := s.paymentAccount(string(evidence.Account))
	if err != nil {
		return nil, domain.MatchNotMatchable, false, err
	}
	evidence = evidence.NormalizeWindow(now)
	evidence.Reference = strings.TrimSpace(evidence.Reference)
	if evidence.AmountPaise <= 0 || evidence.Reference == "" {
		return nil, domain.MatchNotMatchable, false, domain.New(
			"EVIDENCE_NOT_MATCHABLE",
			"payment evidence requires an exact amount and unique reference",
			http.StatusUnprocessableEntity,
		)
	}

	duplicateField, outcomes, codes, err := evidenceIdentity(evidence.ReferenceKind)
	if err != nil {
		return nil, domain.MatchNotMatchable, false, err
	}
	existing, err := tx.FindFirstRecordByData("payments", duplicateField, evidence.Reference)
	if err == nil {
		if existing.GetString("payment_account") != account {
			return nil, outcomes.accountMismatch, false, domain.New(codes.accountMismatch, codes.accountMessage, http.StatusConflict)
		}
		if int64(existing.GetInt("payable_amount")) != evidence.AmountPaise {
			return nil, outcomes.amountMismatch, false, domain.New(codes.amountMismatch, codes.amountMessage, http.StatusConflict)
		}
		return existing, outcomes.duplicate, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, domain.MatchError, false, err
	}
	createdBefore := evidence.OccurredUntil.Add(EvidenceTimestampTolerance)
	onTime, err := tx.FindRecordsByFilter(
		"payments",
		"payment_account = {:account} && payable_amount = {:amount} && created_at <= {:createdBefore} && ((status = 'pending' && expires_at >= {:evidenceAt} && reuse_after > {:now}) || (status = 'expired' && expires_at >= {:evidenceAt} && reuse_after > {:now}) || (status = 'cancelled' && resolved_at != '' && resolved_at >= {:evidenceAt} && reuse_after > {:now}))",
		"created",
		2,
		0,
		dbx.Params{
			"account":       account,
			"amount":        evidence.AmountPaise,
			"now":           filterDate(now),
			"evidenceAt":    filterDate(evidence.OccurredFrom),
			"createdBefore": filterDate(createdBefore),
		},
	)
	if err != nil {
		return nil, domain.MatchError, false, err
	}
	if len(onTime) > 1 {
		return nil, domain.MatchAmbiguous, false, domain.AmbiguousMatch()
	}
	if len(onTime) == 1 {
		record := onTime[0]
		applyNormalizedEvidence(record, evidence, domain.StatusPaid, now, s.Config.AmountQuarantine)
		if err := tx.Save(record); err != nil {
			return nil, domain.MatchError, false, err
		}
		if err := s.schedule(tx, "payment.paid", record, now); err != nil {
			return nil, domain.MatchError, false, err
		}
		return record, domain.MatchMarkedPaid, true, nil
	}

	late, err := tx.FindRecordsByFilter(
		"payments",
		"payment_account = {:account} && payable_amount = {:amount} && (status = 'expired' || status = 'cancelled' || (status = 'pending' && expires_at < {:evidenceAt})) && reuse_after > {:now} && created_at <= {:createdBefore}",
		"-created",
		2,
		0,
		dbx.Params{
			"account":       account,
			"amount":        evidence.AmountPaise,
			"now":           filterDate(now),
			"evidenceAt":    filterDate(evidence.OccurredFrom),
			"createdBefore": filterDate(createdBefore),
		},
	)
	if err != nil {
		return nil, domain.MatchError, false, err
	}
	if len(late) > 1 {
		return nil, domain.MatchAmbiguous, false, domain.AmbiguousMatch()
	}
	if len(late) == 1 {
		record := late[0]
		if record.GetString("status") == string(domain.StatusPending) {
			expiredView := record.Clone()
			expiredView.Set("status", string(domain.StatusExpired))
			expiredView.Set("resolved_at", now)
			if err := s.schedule(tx, "payment.expired", expiredView, now); err != nil {
				return nil, domain.MatchError, false, err
			}
		}
		applyNormalizedEvidence(record, evidence, domain.StatusLate, now, s.Config.AmountQuarantine)
		if err := tx.Save(record); err != nil {
			return nil, domain.MatchError, false, err
		}
		if err := s.schedule(tx, "payment.late", record, now); err != nil {
			return nil, domain.MatchError, false, err
		}
		return record, domain.MatchMarkedLate, true, nil
	}
	return nil, domain.MatchUnmatched, false, nil
}

type evidenceOutcomes struct {
	duplicate       domain.MatchOutcome
	accountMismatch domain.MatchOutcome
	amountMismatch  domain.MatchOutcome
}

type evidenceCodes struct {
	accountMismatch string
	accountMessage  string
	amountMismatch  string
	amountMessage   string
}

func evidenceIdentity(kind domain.EvidenceReferenceKind) (string, evidenceOutcomes, evidenceCodes, error) {
	switch kind {
	case domain.EvidenceReferenceRRN:
		return "rrn", evidenceOutcomes{
				duplicate:       domain.MatchDuplicateRRN,
				accountMismatch: domain.MatchRRNAccountMismatch,
				amountMismatch:  domain.MatchRRNAmountMismatch,
			}, evidenceCodes{
				accountMismatch: "RRN_ACCOUNT_MISMATCH",
				accountMessage:  "the UPI reference was already recorded for a different payment account",
				amountMismatch:  "RRN_AMOUNT_MISMATCH",
				amountMessage:   "the UPI reference was already recorded with a different amount",
			}, nil
	case domain.EvidenceReferenceRelay:
		return "evidence_reference", evidenceOutcomes{
				duplicate:       domain.MatchDuplicateEvidence,
				accountMismatch: domain.MatchEvidenceAccountMismatch,
				amountMismatch:  domain.MatchEvidenceAmountMismatch,
			}, evidenceCodes{
				accountMismatch: "EVIDENCE_ACCOUNT_MISMATCH",
				accountMessage:  "the notification evidence was already recorded for a different payment account",
				amountMismatch:  "EVIDENCE_AMOUNT_MISMATCH",
				amountMessage:   "the notification evidence was already recorded with a different amount",
			}, nil
	default:
		return "", evidenceOutcomes{}, evidenceCodes{}, domain.New(
			"EVIDENCE_REFERENCE_INVALID",
			"payment evidence has an unsupported reference kind",
			http.StatusUnprocessableEntity,
		)
	}
}
func applyNormalizedEvidence(record *core.Record, evidence domain.Evidence, status domain.PaymentStatus, now time.Time, quarantine time.Duration) {
	paidAt := evidence.OccurredFrom.UTC()
	if paidAt.IsZero() || paidAt.After(now) {
		paidAt = now.UTC()
	}
	record.Set("status", string(status))
	record.Set("payer_name", strings.TrimSpace(evidence.PayerName))
	if evidence.UPIID != "" {
		record.Set("upi_id", strings.TrimSpace(evidence.UPIID))
	}
	switch evidence.ReferenceKind {
	case domain.EvidenceReferenceRRN:
		record.Set("rrn", strings.TrimSpace(evidence.Reference))
	case domain.EvidenceReferenceRelay:
		record.Set("evidence_source", string(evidence.Source))
		record.Set("evidence_reference", strings.TrimSpace(evidence.Reference))
	}
	record.Set("paid_at", paidAt)
	record.Set("resolved_at", now.UTC())
	extendReuseAfter(record, now.UTC().Add(quarantine))
}
