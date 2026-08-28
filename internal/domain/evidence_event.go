package domain

import "time"

type SMSEvent struct {
	ID               string
	Source           string
	SourceEventID    string
	Sender           string
	Body             string
	Account          PaymentAccount
	MessageTime      time.Time
	AmountPaise      int64
	RRN              string
	UPIID            string
	PayerName        string
	ProcessingStatus string
	MatchedPaymentID string
	Error            string
	RawPayload       any
}

type EmailEvent struct {
	ID               string
	Source           string
	SourceEventID    string
	EnvelopeSender   string
	Recipient        string
	Sender           string
	Subject          string
	Body             string
	Account          PaymentAccount
	MessageTime      time.Time
	ReceivedAt       time.Time
	AuthResult       string
	AmountPaise      int64
	RRN              string
	UPIID            string
	PayerName        string
	ProcessingStatus string
	MatchedPaymentID string
	Error            string
	RawPayload       any
}

type ReviewCase struct {
	ID                    string
	Kind                  string
	Status                string
	Severity              string
	SMSEventID            string
	EmailEventID          string
	ReconciliationEntryID string
	PaymentID             string
	CandidatePaymentIDs   []string
	Reason                string
	Resolution            string
	ResolutionNote        string
	ResolvedBy            string
	OpenedAt              time.Time
	ResolvedAt            time.Time
}

type NotificationEvent struct {
	ID               string
	Source           string
	SourceEventID    string
	AppPackage       string
	AppName          string
	Title            string
	Body             string
	BigText          string
	Channel          string
	NotificationTime time.Time
	Account          PaymentAccount
	AmountPaise      int64
	PayerName        string
	ProcessingStatus string
	MatchedPaymentID string
	Error            string
	RawPayload       any
}
