package payments

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/store"
	"github.com/pocketbase/pocketbase/core"
)

// MatchEvidence is the single automatic payment matcher used by every trusted
// evidence adapter. Source parsing/authentication happens before this boundary;
// persistence is accessed only through typed repositories.
func (s *Service) MatchEvidence(uow store.UnitOfWork, evidence domain.Evidence, now time.Time) (*domain.Payment, domain.MatchOutcome, bool, error) {
	now = now.UTC()
	account, _, err := s.paymentAccount(string(evidence.Account))
	if err != nil {
		return nil, domain.MatchNotMatchable, false, err
	}
	evidence.Account = domain.PaymentAccount(account)
	evidence = evidence.NormalizeWindow(now)
	evidence.Reference = strings.TrimSpace(evidence.Reference)
	if evidence.AmountPaise <= 0 || evidence.Reference == "" {
		return nil, domain.MatchNotMatchable, false, domain.New(
			"EVIDENCE_NOT_MATCHABLE", "payment evidence requires an exact amount and unique reference", http.StatusUnprocessableEntity,
		)
	}

	_, outcomes, codes, err := evidenceIdentity(evidence.ReferenceKind)
	if err != nil {
		return nil, domain.MatchNotMatchable, false, err
	}
	repo := uow.Payments()
	existing, err := repo.FindByEvidenceReference(evidence.ReferenceKind, evidence.Reference)
	if err == nil {
		if existing.Account != evidence.Account {
			return nil, outcomes.accountMismatch, false, domain.New(codes.accountMismatch, codes.accountMessage, http.StatusConflict)
		}
		if existing.PayablePaise != evidence.AmountPaise {
			return nil, outcomes.amountMismatch, false, domain.New(codes.amountMismatch, codes.amountMessage, http.StatusConflict)
		}
		return existing, outcomes.duplicate, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, domain.MatchError, false, err
	}

	createdBefore := evidence.OccurredUntil.Add(EvidenceTimestampTolerance)
	onTime, err := repo.FindOnTimeCandidates(evidence.Account, evidence.AmountPaise, evidence.OccurredFrom, createdBefore, now)
	if err != nil {
		return nil, domain.MatchError, false, err
	}
	if len(onTime) > 1 {
		return nil, domain.MatchAmbiguous, false, domain.AmbiguousMatch()
	}
	if len(onTime) == 1 {
		payment := onTime[0]
		applyNormalizedEvidence(payment, evidence, domain.StatusPaid, now, s.Config.AmountQuarantine)
		if err := repo.Save(payment); err != nil {
			return nil, domain.MatchError, false, err
		}
		if err := s.scheduleTyped(uow, "payment.paid", payment, now); err != nil {
			return nil, domain.MatchError, false, err
		}
		return payment, domain.MatchMarkedPaid, true, nil
	}

	late, err := repo.FindLateCandidates(evidence.Account, evidence.AmountPaise, evidence.OccurredFrom, createdBefore, now)
	if err != nil {
		return nil, domain.MatchError, false, err
	}
	if len(late) > 1 {
		return nil, domain.MatchAmbiguous, false, domain.AmbiguousMatch()
	}
	if len(late) == 1 {
		payment := late[0]
		if payment.Status == domain.StatusPending {
			expiredView := *payment
			expiredView.Status = domain.StatusExpired
			expiredView.ResolvedAt = now
			if err := s.scheduleTyped(uow, "payment.expired", &expiredView, now); err != nil {
				return nil, domain.MatchError, false, err
			}
		}
		applyNormalizedEvidence(payment, evidence, domain.StatusLate, now, s.Config.AmountQuarantine)
		if err := repo.Save(payment); err != nil {
			return nil, domain.MatchError, false, err
		}
		if err := s.scheduleTyped(uow, "payment.late", payment, now); err != nil {
			return nil, domain.MatchError, false, err
		}
		return payment, domain.MatchMarkedLate, true, nil
	}
	return nil, domain.MatchUnmatched, false, nil
}

// MatchEvidenceInApp is a compatibility shim for evidence ingestors that still
// own a PocketBase transaction. The matching algorithm itself no longer sees
// core.Record or collection fields.
func (s *Service) MatchEvidenceInApp(tx core.App, evidence domain.Evidence, now time.Time) (*core.Record, domain.MatchOutcome, bool, error) {
	payment, outcome, queued, err := s.MatchEvidence(store.NewPocketBaseUnit(tx), evidence, now)
	if err != nil || payment == nil {
		return nil, outcome, queued, err
	}
	record, findErr := tx.FindRecordById("payments", payment.ID)
	if findErr != nil {
		return nil, domain.MatchError, queued, findErr
	}
	return record, outcome, queued, nil
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
func applyNormalizedEvidence(payment *domain.Payment, evidence domain.Evidence, status domain.PaymentStatus, now time.Time, quarantine time.Duration) {
	paidAt := evidence.OccurredFrom.UTC()
	if paidAt.IsZero() || paidAt.After(now) {
		paidAt = now.UTC()
	}
	payment.Status = status
	payment.PayerName = strings.TrimSpace(evidence.PayerName)
	if evidence.UPIID != "" {
		payment.UPIId = strings.TrimSpace(evidence.UPIID)
	}
	switch evidence.ReferenceKind {
	case domain.EvidenceReferenceRRN:
		payment.RRN = strings.TrimSpace(evidence.Reference)
	case domain.EvidenceReferenceRelay:
		payment.EvidenceSource = string(evidence.Source)
		payment.EvidenceReference = strings.TrimSpace(evidence.Reference)
	}
	payment.PaidAt = paidAt
	payment.ResolvedAt = now.UTC()
	candidate := now.UTC().Add(quarantine)
	if payment.ReuseAfter.IsZero() || candidate.After(payment.ReuseAfter) {
		payment.ReuseAfter = candidate
	}
}
