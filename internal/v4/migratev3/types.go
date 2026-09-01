package migratev3

import (
	"encoding/json"
	"time"
)

type Options struct {
	SourceZIP     string
	Destination   string
	ActiveProfile string
	KotakUPIID    string
	KotakPayee    string
	PaytmUPIID    string
	PaytmPayee    string
	Now           func() time.Time
}

type Report struct {
	SourceSHA256       string         `json:"source_sha256"`
	SourcePayments     int            `json:"source_payments"`
	MigratedPayments   int            `json:"migrated_payments"`
	MigratedByStatus   map[string]int `json:"migrated_by_status"`
	MigratedByProfile  map[string]int `json:"migrated_by_profile"`
	MigratedDevice     bool           `json:"migrated_device"`
	LegacyNamesFilled  int            `json:"legacy_names_filled"`
	LateNormalizedPaid int            `json:"late_normalized_paid"`
	EventIDsRecovered  int            `json:"event_ids_recovered"`
	ArchivedOnly       map[string]int `json:"archived_only_rows,omitempty"`
	Warnings           []string       `json:"warnings,omitempty"`
	CompletedAt        time.Time      `json:"completed_at"`
}

type legacyPayment struct {
	ID, ExternalID, Status, Account                        string
	CustomerName, PayerName, PayerUPIID                    string
	DisplayName, CustomerEmail, CustomerPhone, Description string
	AdminNote, IdempotencyKey                              string
	Metadata, Tags, CustomFields                           json.RawMessage
	Requested, Payable                                     int64
	CreatedAt, ExpiresAt, ReuseAfter                       time.Time
	PaidAt, ResolvedAt                                     *time.Time
}
type legacyDevice struct {
	DeviceID, Name, PublicKeyPEM                   string
	AppVersion, DeviceModel, AndroidVersion        string
	LastClientError                                string
	Enabled                                        bool
	EnrolledAt                                     time.Time
	LastSeenAt, LastHeartbeatAt, LastDeliveryAt    *time.Time
	NotificationAccess, ListenerConnected          bool
	BatteryExempt, PowerSave, BackgroundRestricted bool
	ForegroundService                              bool
	PendingCount, FailedCount                      int64
}

type migratedPayment struct {
	ID, Name, ExternalID, MetadataJSON           string
	Requested, Payable, Adjustment               int64
	ProfileID, UPIIDSnapshot, PayeeSnapshot      string
	Status                                       string
	CreatedAt, ExpiresAt, GraceUntil, ReuseAfter time.Time
	PaidAt                                       *time.Time
	PayerName, PayerUPIID, InternalNote          string
	LegacyStatus, LegacyIdempotencyKey           string
	LegacyResolvedAt                             *time.Time
}
