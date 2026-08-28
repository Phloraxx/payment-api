package domain

import "time"

type EvidenceSource string
type EvidenceReferenceKind string
type MatchOutcome string

const (
	EvidenceSourceBankSMS           EvidenceSource = "bank_sms"
	EvidenceSourceBankEmail         EvidenceSource = "bank_email"
	EvidenceSourcePaytmNotification EvidenceSource = "paytm_notification"
	EvidenceSourceReconciliation    EvidenceSource = "reconciliation"
	EvidenceSourceManual            EvidenceSource = "manual"
)

const (
	EvidenceReferenceRRN   EvidenceReferenceKind = "rrn"
	EvidenceReferenceRelay EvidenceReferenceKind = "relay_reference"
)
const (
	MatchMarkedPaid              MatchOutcome = "marked_paid"
	MatchMarkedLate              MatchOutcome = "marked_late"
	MatchDuplicateRRN            MatchOutcome = "duplicate_rrn"
	MatchDuplicateEvidence       MatchOutcome = "duplicate_evidence"
	MatchRRNAccountMismatch      MatchOutcome = "rrn_account_mismatch"
	MatchRRNAmountMismatch       MatchOutcome = "rrn_amount_mismatch"
	MatchEvidenceAccountMismatch MatchOutcome = "evidence_account_mismatch"
	MatchEvidenceAmountMismatch  MatchOutcome = "evidence_amount_mismatch"
	MatchAmbiguous               MatchOutcome = "ambiguous"
	MatchUnmatched               MatchOutcome = "unmatched"
	MatchNotMatchable            MatchOutcome = "not_matchable"
	MatchError                   MatchOutcome = "error"
)

type Evidence struct {
	Account       PaymentAccount
	AmountPaise   int64
	OccurredFrom  time.Time
	OccurredUntil time.Time

	Reference     string
	ReferenceKind EvidenceReferenceKind
	Source        EvidenceSource
	PayerName     string
	UPIID         string
}

func (e Evidence) NormalizeWindow(now time.Time) Evidence {
	now = now.UTC()
	e.OccurredFrom = e.OccurredFrom.UTC()
	e.OccurredUntil = e.OccurredUntil.UTC()
	if e.OccurredFrom.IsZero() || e.OccurredFrom.After(now) {
		e.OccurredFrom = now
		e.OccurredUntil = now
	}
	if e.OccurredUntil.IsZero() || e.OccurredUntil.Before(e.OccurredFrom) {
		e.OccurredUntil = e.OccurredFrom
	}
	if e.OccurredUntil.After(now) {
		e.OccurredUntil = now
	}
	return e
}
