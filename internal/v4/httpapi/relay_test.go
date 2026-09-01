package httpapi

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/payments"
	"github.com/Phloraxx/payment-api/internal/v4/relay"
	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

type relayHTTPFixture struct {
	db      *storage.DB
	service *relay.Service
	handler *RelayHandler
	private *ecdsa.PrivateKey
	device  string
	now     time.Time
}

func newRelayHTTPFixture(t *testing.T) relayHTTPFixture {
	t.Helper()
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "paygate.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	paymentService := payments.NewService(db)
	service := relay.NewService(db, paymentService)
	now := time.Date(2026, 9, 1, 6, 30, 0, 0, time.UTC)
	service.Now = func() time.Time { return now }

	private, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&private.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(der)
	deviceID := hex.EncodeToString(sum[:])
	return relayHTTPFixture{db: db, service: service, handler: NewRelayHandler(service), private: private, device: deviceID, now: now}
}
func (f relayHTTPFixture) publicKeyPEM(t *testing.T) string {
	t.Helper()
	der, err := x509.MarshalPKIXPublicKey(&f.private.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
}

func pairRelayHTTP(t *testing.T, f relayHTTPFixture) {
	t.Helper()
	session, err := f.service.CreatePairing(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]any{
		"token": session.Token, "name": "Edge 60 Stylus", "public_key_pem": f.publicKeyPEM(t),
		"app_version": "0.5.0", "device_model": "motorola edge 60 stylus", "android_version": "16",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v4/relay/pair", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), f.device) {
		t.Fatalf("pair status=%d body=%s", rr.Code, rr.Body.String())
	}
}
func signedRelayRequest(t *testing.T, f relayHTTPFixture, path string, body []byte) *http.Request {
	t.Helper()
	timestamp := strconv.FormatInt(f.now.UnixMilli(), 10)
	canonical := relay.CanonicalRequest(http.MethodPost, path, timestamp, body)
	digest := sha256.Sum256([]byte(canonical))
	signature, err := ecdsa.SignASN1(rand.Reader, f.private, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(relayDeviceHeader, f.device)
	req.Header.Set(relayTimeHeader, timestamp)
	req.Header.Set(relaySignatureHeader, base64.StdEncoding.EncodeToString(signature))
	return req
}

func TestRelayPairHeartbeatAndHealthPersistence(t *testing.T) {
	f := newRelayHTTPFixture(t)
	pairRelayHTTP(t, f)
	body := []byte(`{"schema_version":1,"app_version":"0.5.0","android_version":"16","device_model":"motorola edge 60 stylus","notification_access":true,"listener_connected":true,"battery_optimization_exempt":true,"power_save_mode":false,"background_restricted":false,"foreground_service":true,"pending_count":0,"failed_count":2}`)
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, signedRelayRequest(t, f, relay.HeartbeatPath, body))
	if rr.Code != http.StatusOK {
		t.Fatalf("heartbeat status=%d body=%s", rr.Code, rr.Body.String())
	}
	device, err := f.service.ActiveDevice(context.Background())
	if err != nil || device == nil || device.LastHeartbeatAt == nil || device.NotificationAccess == nil || !*device.NotificationAccess || device.FailedCount == nil || *device.FailedCount != 2 {
		t.Fatalf("device=%+v err=%v", device, err)
	}
}
func TestRelaySignedEventAndSignatureFailure(t *testing.T) {
	f := newRelayHTTPFixture(t)
	pairRelayHTTP(t, f)
	body := []byte(`{"schema_version":1,"event_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","package_name":"com.paytm.business","posted_at_ms":1788244200000,"title":"Payment Received on Paytm for Business","text":"₹100.00 Received from Test"}`)
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, signedRelayRequest(t, f, relay.EventPath, body))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"status":"ignored"`) {
		t.Fatalf("event status=%d body=%s", rr.Code, rr.Body.String())
	}

	bad := signedRelayRequest(t, f, relay.EventPath, body)
	bad.Header.Set(relaySignatureHeader, base64.StdEncoding.EncodeToString([]byte("bad")))
	rr = httptest.NewRecorder()
	f.handler.ServeHTTP(rr, bad)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("bad signature status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRelayRejectsQueryParameters(t *testing.T) {
	f := newRelayHTTPFixture(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v4/relay/pair?token=leak", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("query status=%d body=%s", rr.Code, rr.Body.String())
	}
}
