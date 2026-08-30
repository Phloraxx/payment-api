package androidrelay

import (
	"context"
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
	"github.com/Phloraxx/payment-api/internal/evidenceshadow"
	"github.com/Phloraxx/payment-api/internal/paytmnotification"
	"github.com/Phloraxx/payment-api/internal/store"
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
	Store store.Database
	Paytm *paytmnotification.Service
	Now   func() time.Time
}

func NewService(app core.App, paytm *paytmnotification.Service) *Service {
	return &Service{Store: store.NewPocketBase(app), Paytm: paytm, Now: time.Now}
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
	_, der, err := parsePublicKey(in.PublicKeyPEM)
	if err != nil {
		return EnrollmentResult{}, domain.New("INVALID_RELAY_PUBLIC_KEY", "public key must be a P-256 ECDSA SubjectPublicKeyInfo PEM", 400)
	}
	sum := sha256.Sum256(der)
	if hex.EncodeToString(sum[:]) != in.DeviceID {
		return EnrollmentResult{}, domain.New("RELAY_DEVICE_ID_MISMATCH", "deviceId does not match public key fingerprint", 400)
	}
	now := s.now()
	enabled := false
	err = s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		repo := uow.Relay()
		existing, findErr := repo.FindByDeviceID(in.DeviceID)
		if findErr == nil {
			if strings.TrimSpace(existing.PublicKeyPEM) != in.PublicKeyPEM {
				return domain.New("RELAY_DEVICE_KEY_CONFLICT", "this deviceId is already enrolled with a different public key", 409)
			}
			existing.Name = in.Name
			existing.AppVersion = trimMax(in.AppVersion, 64)
			existing.AndroidVersion = trimMax(in.AndroidVersion, 64)
			existing.DeviceModel = trimMax(in.DeviceModel, 255)
			if existing.EnrolledAt.IsZero() {
				existing.EnrolledAt = now
			}
			enabled = existing.Enabled
			return repo.Save(existing)
		}
		if !errors.Is(findErr, sql.ErrNoRows) {
			return findErr
		}
		device := &domain.RelayDevice{DeviceID: in.DeviceID, Name: in.Name, PublicKeyPEM: in.PublicKeyPEM, Enabled: true, AppVersion: trimMax(in.AppVersion, 64), AndroidVersion: trimMax(in.AndroidVersion, 64), DeviceModel: trimMax(in.DeviceModel, 255), EnrolledAt: now}
		if err := repo.Create(device); err != nil {
			return err
		}
		enabled = true
		return nil
	})
	if err != nil {
		return EnrollmentResult{}, err
	}
	return EnrollmentResult{DeviceID: in.DeviceID, Enrolled: true, Enabled: enabled}, nil
}

func (s *Service) Verify(deviceID, timestamp, signature, method, path string, body []byte) (*domain.RelayDevice, error) {
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
	var device *domain.RelayDevice
	err = s.Store.View(context.Background(), func(uow store.UnitOfWork) error {
		var findErr error
		device, findErr = uow.Relay().FindByDeviceID(deviceID)
		return findErr
	})
	if err != nil || device == nil || !device.Enabled {
		return nil, domain.New("UNKNOWN_RELAY_DEVICE", "relay device is not enrolled or is disabled", 401)
	}
	pub, _, err := parsePublicKey(device.PublicKeyPEM)
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
	return device, nil
}

func (s *Service) Ingest(device *domain.RelayDevice, in EventInput, raw any) (EventResult, error) {
	if device == nil {
		return EventResult{}, domain.New("UNKNOWN_RELAY_DEVICE", "relay device is not enrolled or is disabled", 401)
	}
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
	enrolledAt := device.EnrolledAt
	if enrolledAt.IsZero() {
		enrolledAt = device.CreatedAt
	}
	// Preserve v1 behavior: valid allowlisted traffic refreshes last_seen even if downstream matching later fails.
	if err := s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		current, err := uow.Relay().Get(device.ID)
		if err != nil {
			return err
		}
		if !current.Enabled {
			return domain.New("UNKNOWN_RELAY_DEVICE", "relay device is not enrolled or is disabled", 401)
		}
		current.LastSeenAt = now
		return uow.Relay().Save(current)
	}); err != nil {
		return EventResult{}, err
	}
	var result EventResult
	var queued bool
	err := s.Store.Write(context.Background(), func(uow store.UnitOfWork) error {
		repo := uow.RelayEvents()
		existing, findErr := repo.FindByDeviceEvent(device.ID, in.EventID)
		if findErr == nil {
			result = resultFromRelayEvent(existing)
			result.Duplicate = true
			result.Action = "duplicate_event"
			return nil
		}
		if !errors.Is(findErr, sql.ErrNoRows) {
			return findErr
		}
		event := &domain.RelayEvent{DeviceRecordID: device.ID, EventID: in.EventID, Kind: in.Kind, AppPackage: n.PackageName, AppName: trimMax(n.AppName, 255), NotificationKey: trimMax(n.Key, 512), NotificationID: n.ID, NotificationTag: trimMax(n.Tag, 255), GroupKey: trimMax(n.GroupKey, 512), IsGroupSummary: n.IsGroupSummary, PostTime: postTime, NotificationWhen: whenTime, CapturedAt: capturedAt, ChannelID: trimMax(n.ChannelID, 255), Category: trimMax(n.Category, 255), Title: n.Title, Body: n.Text, BigText: n.BigText, SubText: n.SubText, SummaryText: n.SummaryText, TextLines: append([]string(nil), n.TextLines...), CustomTexts: append([]string(nil), n.CustomTexts...), ProcessingStatus: "received", RawPayload: raw}
		if err := repo.Create(event); err != nil {
			return err
		}
		result.EventID = event.ID
		if !enrolledAt.IsZero() && postTime.Before(enrolledAt.Add(-2*time.Minute)) {
			event.ProcessingStatus = "ignored"
			event.Error = "notification predates relay enrollment"
			result.Status = "ignored"
			result.Action = "ignored_pre_enrollment"
			return repo.Save(event)
		}
		if n.IsGroupSummary {
			event.ProcessingStatus = "ignored"
			event.Error = "group summary notification"
			result.Status = "ignored"
			result.Action = "ignored_group_summary"
			return repo.Save(event)
		}
		if n.PackageName == GoogleMessagesPackage {
			custom := strings.Join(n.CustomTexts, "\n")
			shadowText := strings.TrimSpace(strings.Join([]string{n.Title, n.Text, n.BigText, custom, strings.Join(n.TextLines, "\n")}, "\n"))
			annotation := evidenceshadow.Annotate(event, shadowText)
			event.ProcessingStatus = "shadow_observed"
			result.Status = "observed"
			result.Action = "shadow_" + annotation.ParseStatus
			return repo.Save(event)
		}
		if n.PackageName != PaytmBusinessPackage || s.Paytm == nil {
			event.ProcessingStatus = "observed"
			result.Status = "observed"
			result.Action = "observed_only"
			return repo.Save(event)
		}
		custom := strings.Join(n.CustomTexts, "\n")
		downstream, matchQueued, err := s.Paytm.IngestUoW(uow, paytmnotification.Input{Source: "android_relay", SourceEventID: "android:" + device.DeviceID + ":" + in.EventID, AppPackage: n.PackageName, AppName: n.AppName, Title: n.Title, Body: strings.TrimSpace(strings.Join([]string{n.Text, custom, strings.Join(n.TextLines, "\n")}, "\n")), BigText: n.BigText, Channel: n.ChannelID, NotificationTime: whenTime, RawPayload: map[string]any{"relayEventId": in.EventID, "deviceId": device.DeviceID}})
		if err != nil {
			event.ProcessingStatus = "error"
			event.Error = err.Error()
			result.Status = "error"
			result.Action = "downstream_error"
			_ = repo.Save(event)
			return err
		}
		queued = queued || matchQueued
		event.ProcessingStatus = "forwarded"
		event.DownstreamEventID = downstream.EventID
		event.MatchedPaymentID = downstream.PaymentID
		event.ProviderResult = map[string]any{"status": downstream.Status, "action": downstream.Action, "duplicate": downstream.Duplicate}
		result.Status = downstream.Status
		result.Action = downstream.Action
		result.PaymentID = downstream.PaymentID
		result.Duplicate = downstream.Duplicate
		return repo.Save(event)
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
func resultFromRelayEvent(r *domain.RelayEvent) EventResult {
	if r == nil {
		return EventResult{}
	}
	return EventResult{EventID: r.ID, Status: r.ProcessingStatus, PaymentID: r.MatchedPaymentID}
}

func CanonicalRequest(method, path, timestamp string, body []byte) string {
	sum := sha256.Sum256(body)
	return fmt.Sprintf("%s\n%s\n%s\n%s", strings.ToUpper(method), path, strings.TrimSpace(timestamp), hex.EncodeToString(sum[:]))
}
