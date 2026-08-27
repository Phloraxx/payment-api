package androidrelay

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/paytmnotification"
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	PaytmBusinessPackage  = "com.paytm.business"
	GPayPersonalPackage   = "com.google.android.apps.nbu.paisa.user"
	GPayBusinessPackage   = "com.google.android.apps.nbu.paisa.merchant"
	GoogleMessagesPackage = "com.google.android.apps.messaging"
	signatureTolerance    = 5 * time.Minute
)

var allowedPackages = map[string]bool{
	PaytmBusinessPackage:  true,
	GPayPersonalPackage:   true,
	GPayBusinessPackage:   true,
	GoogleMessagesPackage: true,
}

type Service struct {
	App   core.App
	Paytm *paytmnotification.Service
	Now   func() time.Time
}

func NewService(app core.App, paytm *paytmnotification.Service) *Service {
	return &Service{App: app, Paytm: paytm, Now: time.Now}
}

type EnrollmentInput struct {
	DeviceID       string `json:"deviceId"`
	Name           string `json:"name"`
	PublicKeyPEM   string `json:"publicKeyPem"`
	AppVersion     string `json:"appVersion"`
	AndroidVersion string `json:"androidVersion"`
	DeviceModel    string `json:"deviceModel"`
}

type EnrollmentResult struct {
	DeviceID string `json:"deviceId"`
	Enrolled bool   `json:"enrolled"`
	Enabled  bool   `json:"enabled"`
}

type Notification struct {
	PackageName    string   `json:"packageName"`
	AppName        string   `json:"appName"`
	Key            string   `json:"key"`
	ID             int      `json:"id"`
	Tag            string   `json:"tag"`
	GroupKey       string   `json:"groupKey"`
	IsGroupSummary bool     `json:"isGroupSummary"`
	PostTimeMs     int64    `json:"postTimeMs"`
	WhenMs         int64    `json:"whenMs"`
	ChannelID      string   `json:"channelId"`
	Category       string   `json:"category"`
	Title          string   `json:"title"`
	Text           string   `json:"text"`
	BigText        string   `json:"bigText"`
	SubText        string   `json:"subText"`
	SummaryText    string   `json:"summaryText"`
	TextLines      []string `json:"textLines"`
	CustomTexts    []string `json:"customTexts"`
}

type EventInput struct {
	SchemaVersion int          `json:"schemaVersion"`
	EventID       string       `json:"eventId"`
	Kind          string       `json:"kind"`
	CapturedAtMs  int64        `json:"capturedAtMs"`
	Notification  Notification `json:"notification"`
}

type EventResult struct {
	EventID   string `json:"eventId"`
	Status    string `json:"status"`
	Action    string `json:"action"`
	PaymentID string `json:"paymentId,omitempty"`
	Duplicate bool   `json:"duplicate,omitempty"`
}

func (s *Service) Enroll(in EnrollmentInput) (EnrollmentResult, error) {
	in.DeviceID = strings.ToLower(strings.TrimSpace(in.DeviceID))
	in.Name = strings.TrimSpace(in.Name)
	in.PublicKeyPEM = strings.TrimSpace(in.PublicKeyPEM)
	if len(in.DeviceID) != 64 {
		return EnrollmentResult{}, domain.New("INVALID_RELAY_DEVICE_ID", "deviceId must be a SHA-256 hex fingerprint", 400)
	}
	if _, err := hex.DecodeString(in.DeviceID); err != nil {
		return EnrollmentResult{}, domain.New("INVALID_RELAY_DEVICE_ID", "deviceId must be a SHA-256 hex fingerprint", 400)
	}
	if in.Name == "" || utf8.RuneCountInString(in.Name) > 120 {
		return EnrollmentResult{}, domain.New("INVALID_RELAY_DEVICE_NAME", "device name is required and must be at most 120 characters", 400)
	}
	pub, der, err := parsePublicKey(in.PublicKeyPEM)
	if err != nil {
		return EnrollmentResult{}, domain.New("INVALID_RELAY_PUBLIC_KEY", "public key must be a P-256 ECDSA SubjectPublicKeyInfo PEM", 400)
	}
	_ = pub
	sum := sha256.Sum256(der)
	if hex.EncodeToString(sum[:]) != in.DeviceID {
		return EnrollmentResult{}, domain.New("RELAY_DEVICE_ID_MISMATCH", "deviceId does not match public key fingerprint", 400)
	}
	now := s.now()
	var enabled bool
	err = s.App.RunInTransaction(func(tx core.App) error {
		existing, findErr := tx.FindFirstRecordByFilter("relay_devices", "device_id = {:id}", dbx.Params{"id": in.DeviceID})
		if findErr == nil {
			if strings.TrimSpace(existing.GetString("public_key_pem")) != in.PublicKeyPEM {
				return domain.New("RELAY_DEVICE_KEY_CONFLICT", "this deviceId is already enrolled with a different public key", 409)
			}
			existing.Set("name", in.Name)
			existing.Set("app_version", trimMax(in.AppVersion, 64))
			existing.Set("android_version", trimMax(in.AndroidVersion, 64))
			existing.Set("device_model", trimMax(in.DeviceModel, 255))
			existing.Set("last_seen_at", now)
			enabled = existing.GetBool("enabled")
			return tx.Save(existing)
		}
		if !errors.Is(findErr, sql.ErrNoRows) {
			return findErr
		}
		c, err := tx.FindCollectionByNameOrId("relay_devices")
		if err != nil {
			return err
		}
		r := core.NewRecord(c)
		r.Set("device_id", in.DeviceID)
		r.Set("name", in.Name)
		r.Set("public_key_pem", in.PublicKeyPEM)
		r.Set("enabled", true)
		r.Set("app_version", trimMax(in.AppVersion, 64))
		r.Set("android_version", trimMax(in.AndroidVersion, 64))
		r.Set("device_model", trimMax(in.DeviceModel, 255))
		r.Set("last_seen_at", now)
		enabled = true
		return tx.Save(r)
	})
	if err != nil {
		return EnrollmentResult{}, err
	}
	return EnrollmentResult{DeviceID: in.DeviceID, Enrolled: true, Enabled: enabled}, nil
}

func (s *Service) Verify(deviceID, timestamp, signature, method, path string, body []byte) (*core.Record, error) {
	deviceID = strings.ToLower(strings.TrimSpace(deviceID))
	tsMs, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
	if err != nil || tsMs <= 0 {
		return nil, domain.New("INVALID_RELAY_TIMESTAMP", "invalid relay timestamp", 401)
	}
	now := s.now()
	requestTime := time.UnixMilli(tsMs).UTC()
	delta := now.Sub(requestTime)
	if delta < 0 {
		delta = -delta
	}
	if delta > signatureTolerance {
		return nil, domain.New("STALE_RELAY_REQUEST", "relay request timestamp is outside the allowed window", 401)
	}
	device, err := s.App.FindFirstRecordByFilter("relay_devices", "device_id = {:id}", dbx.Params{"id": deviceID})
	if err != nil || !device.GetBool("enabled") {
		return nil, domain.New("UNKNOWN_RELAY_DEVICE", "relay device is not enrolled or is disabled", 401)
	}
	pub, _, err := parsePublicKey(device.GetString("public_key_pem"))
	if err != nil {
		return nil, domain.New("INVALID_RELAY_DEVICE_KEY", "stored relay device key is invalid", 500)
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature))
	if err != nil {
		return nil, domain.New("INVALID_RELAY_SIGNATURE", "invalid relay signature", 401)
	}
	hash := sha256.Sum256(body)
	canonical := strings.Join([]string{strings.ToUpper(method), path, strings.TrimSpace(timestamp), hex.EncodeToString(hash[:])}, "\n")
	digest := sha256.Sum256([]byte(canonical))
	if !ecdsa.VerifyASN1(pub, digest[:], sig) {
		return nil, domain.New("INVALID_RELAY_SIGNATURE", "invalid relay signature", 401)
	}
	device.Set("last_seen_at", now)
	if err := s.App.Save(device); err != nil {
		return nil, err
	}
	return device, nil
}

func (s *Service) Ingest(device *core.Record, in EventInput, raw any) (EventResult, error) {
	in.EventID = strings.ToLower(strings.TrimSpace(in.EventID))
	in.Kind = strings.ToLower(strings.TrimSpace(in.Kind))
	n := &in.Notification
	n.PackageName = strings.TrimSpace(n.PackageName)
	if in.SchemaVersion != 1 {
		return EventResult{}, domain.New("UNSUPPORTED_RELAY_SCHEMA", "schemaVersion must be 1", 400)
	}
	if in.Kind != "notification" {
		return EventResult{}, domain.New("UNSUPPORTED_RELAY_EVENT", "only notification events are supported", 400)
	}
	if len(in.EventID) != 64 {
		return EventResult{}, domain.New("INVALID_RELAY_EVENT_ID", "eventId must be a SHA-256 hex string", 400)
	}
	if _, err := hex.DecodeString(in.EventID); err != nil {
		return EventResult{}, domain.New("INVALID_RELAY_EVENT_ID", "eventId must be a SHA-256 hex string", 400)
	}
	if !allowedPackages[n.PackageName] {
		return EventResult{}, domain.New("UNSUPPORTED_RELAY_APP", "notification app is not allowlisted", 400)
	}
	if totalTextBytes(*n) > 128*1024 {
		return EventResult{}, domain.New("RELAY_EVENT_TOO_LARGE", "notification text is too large", 400)
	}
	now := s.now()
	capturedAt := millisTime(in.CapturedAtMs, now, now)
	postTime := millisTime(n.PostTimeMs, capturedAt, now)
	whenTime := millisTime(n.WhenMs, postTime, now)
	var result EventResult
	var queued bool
	err := s.App.RunInTransaction(func(tx core.App) error {
		existing, findErr := tx.FindFirstRecordByFilter("relay_events", "device = {:device} && event_id = {:event}", dbx.Params{"device": device.Id, "event": in.EventID})
		if findErr == nil {
			result = resultFromRelayEvent(existing)
			result.Duplicate = true
			result.Action = "duplicate_event"
			return nil
		}
		if !errors.Is(findErr, sql.ErrNoRows) {
			return findErr
		}
		c, err := tx.FindCollectionByNameOrId("relay_events")
		if err != nil {
			return err
		}
		event := core.NewRecord(c)
		event.Set("device", device.Id)
		event.Set("event_id", in.EventID)
		event.Set("kind", in.Kind)
		event.Set("app_package", n.PackageName)
		event.Set("app_name", trimMax(n.AppName, 255))
		event.Set("notification_key", trimMax(n.Key, 512))
		event.Set("notification_id", n.ID)
		event.Set("notification_tag", trimMax(n.Tag, 255))
		event.Set("group_key", trimMax(n.GroupKey, 512))
		event.Set("is_group_summary", n.IsGroupSummary)
		event.Set("post_time", postTime)
		event.Set("notification_when", whenTime)
		event.Set("captured_at", capturedAt)
		event.Set("channel_id", trimMax(n.ChannelID, 255))
		event.Set("category", trimMax(n.Category, 255))
		event.Set("title", n.Title)
		event.Set("body", n.Text)
		event.Set("big_text", n.BigText)
		event.Set("sub_text", n.SubText)
		event.Set("summary_text", n.SummaryText)
		event.Set("text_lines", n.TextLines)
		event.Set("custom_texts", n.CustomTexts)
		event.Set("processing_status", "received")
		if raw != nil {
			event.Set("raw_payload", raw)
		}
		if err := tx.Save(event); err != nil {
			return err
		}
		result.EventID = event.Id
		if n.IsGroupSummary {
			event.Set("processing_status", "ignored")
			event.Set("error", "group summary notification")
			result.Status = "ignored"
			result.Action = "ignored_group_summary"
			return tx.Save(event)
		}
		if n.PackageName != PaytmBusinessPackage || s.Paytm == nil {
			event.Set("processing_status", "observed")
			result.Status = "observed"
			result.Action = "observed_only"
			return tx.Save(event)
		}
		custom := strings.Join(n.CustomTexts, "\n")
		downstream, matchQueued, err := s.Paytm.IngestInApp(tx, paytmnotification.Input{
			Source:        "android_relay",
			SourceEventID: "android:" + device.GetString("device_id") + ":" + in.EventID,
			AppPackage:    n.PackageName, AppName: n.AppName,
			Title: n.Title, Body: strings.TrimSpace(strings.Join([]string{n.Text, custom, strings.Join(n.TextLines, "\n")}, "\n")),
			BigText: n.BigText, Channel: n.ChannelID, NotificationTime: whenTime, RawPayload: raw,
		})
		if err != nil {
			event.Set("processing_status", "error")
			event.Set("error", err.Error())
			result.Status = "error"
			result.Action = "downstream_error"
			_ = tx.Save(event)
			return err
		}
		queued = queued || matchQueued
		event.Set("processing_status", "forwarded")
		event.Set("downstream_event_id", downstream.EventID)
		event.Set("matched_payment", downstream.PaymentID)
		event.Set("provider_result", map[string]any{"status": downstream.Status, "action": downstream.Action, "duplicate": downstream.Duplicate})
		result.Status = downstream.Status
		result.Action = downstream.Action
		result.PaymentID = downstream.PaymentID
		result.Duplicate = downstream.Duplicate
		return tx.Save(event)
	})
	if err == nil && queued && s.Paytm != nil && s.Paytm.Payments != nil {
		s.Paytm.Payments.WakeWebhooks()
	}
	return result, err
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func parsePublicKey(value string) (*ecdsa.PublicKey, []byte, error) {
	block, _ := pem.Decode([]byte(value))
	if block == nil {
		return nil, nil, errors.New("invalid pem")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, nil, err
	}
	pub, ok := parsed.(*ecdsa.PublicKey)
	if !ok || pub.Curve != elliptic.P256() {
		return nil, nil, errors.New("not p256 ecdsa")
	}
	return pub, block.Bytes, nil
}
func trimMax(v string, max int) string {
	v = strings.TrimSpace(v)
	r := []rune(v)
	if len(r) > max {
		return string(r[:max])
	}
	return v
}
func millisTime(ms int64, fallback, now time.Time) time.Time {
	if ms <= 0 {
		return fallback.UTC()
	}
	t := time.UnixMilli(ms).UTC()
	if t.Year() < 2020 || t.After(now.Add(24*time.Hour)) {
		return fallback.UTC()
	}
	return t
}
func totalTextBytes(n Notification) int {
	total := len(n.Title) + len(n.Text) + len(n.BigText) + len(n.SubText) + len(n.SummaryText)
	for _, v := range n.TextLines {
		total += len(v)
	}
	for _, v := range n.CustomTexts {
		total += len(v)
	}
	return total
}
func resultFromRelayEvent(r *core.Record) EventResult {
	return EventResult{EventID: r.Id, Status: r.GetString("processing_status"), PaymentID: r.GetString("matched_payment")}
}

func CanonicalRequest(method, path, timestamp string, body []byte) string {
	sum := sha256.Sum256(body)
	return fmt.Sprintf("%s\n%s\n%s\n%s", strings.ToUpper(method), path, strings.TrimSpace(timestamp), hex.EncodeToString(sum[:]))
}
