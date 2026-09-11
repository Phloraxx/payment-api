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

func pairRelayHTTP(t *testing.T, f relayHTTPFixture) int64 {
	t.Helper()
	session, err := f.service.CreatePairing(context.Background())
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
	var response struct {
		EnrolledAtMS int64 `json:"enrolled_at_ms"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	var dbEpoch int64
	if err := f.db.SQL.QueryRow(`SELECT enrolled_at FROM relay_devices WHERE id=?`, f.device).Scan(&dbEpoch); err != nil {
		t.Fatal(err)
	}
	if response.EnrolledAtMS != dbEpoch {
		t.Fatalf("pair response epoch=%d db epoch=%d", response.EnrolledAtMS, dbEpoch)
	}
	return response.EnrolledAtMS
}
func signedRelayRequest(t *testing.T, f relayHTTPFixture, path string, body []byte, enrollmentEpoch int64) *http.Request {
	t.Helper()
	timestamp := strconv.FormatInt(f.now.UnixMilli(), 10)
	canonical := relay.CanonicalRequestWithEpoch(http.MethodPost, path, timestamp, enrollmentEpoch, body)
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
	req.Header.Set(relayEpochHeader, strconv.FormatInt(enrollmentEpoch, 10))
	return req
}

func TestRelayPairHeartbeatAndHealthPersistence(t *testing.T) {
	f := newRelayHTTPFixture(t)
	epoch := pairRelayHTTP(t, f)
	body := []byte(`{"schema_version":1,"app_version":"0.5.0","android_version":"16","device_model":"motorola edge 60 stylus","notification_access":true,"listener_connected":true,"battery_optimization_exempt":true,"power_save_mode":false,"background_restricted":false,"foreground_service":true,"pending_count":0,"failed_count":2}`)
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, signedRelayRequest(t, f, relay.HeartbeatPath, body, epoch))
	if rr.Code != http.StatusOK {
		t.Fatalf("heartbeat status=%d body=%s", rr.Code, rr.Body.String())
	}
	devices, err := f.service.Devices(context.Background())
	if err != nil || len(devices) != 1 || devices[0].LastHeartbeatAt == nil || devices[0].NotificationAccess == nil || !*devices[0].NotificationAccess || devices[0].FailedCount == nil || *devices[0].FailedCount != 2 {
		t.Fatalf("devices=%+v err=%v", devices, err)
	}
}
func TestRelayHeartbeatDatabaseBusyIsRetryable(t *testing.T) {
	f := newRelayHTTPFixture(t)
	epoch := pairRelayHTTP(t, f)
	ctx := context.Background()
	f.db.SQL.SetMaxOpenConns(2)
	for i := 0; i < 2; i++ {
		conn, err := f.db.SQL.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := conn.ExecContext(ctx, `PRAGMA busy_timeout=1`); err != nil {
			conn.Close()
			t.Fatal(err)
		}
		conn.Close()
	}

	locked := make(chan struct{})
	release := make(chan struct{})
	txErr := make(chan error, 1)
	go func() {
		txErr <- f.db.WithImmediateTx(ctx, func(*storage.ImmediateTx) error {
			close(locked)
			<-release
			return nil
		})
	}()
	<-locked

	body := []byte(`{"schema_version":1,"app_version":"0.5.0","android_version":"16","device_model":"motorola edge 60 stylus","notification_access":true,"listener_connected":true,"battery_optimization_exempt":true,"power_save_mode":false,"background_restricted":false,"foreground_service":true,"pending_count":0,"failed_count":2}`)
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, signedRelayRequest(t, f, relay.HeartbeatPath, body, epoch))
	close(release)
	if err := <-txErr; err != nil {
		t.Fatal(err)
	}
	if rr.Code != http.StatusServiceUnavailable || rr.Header().Get("Retry-After") != "1" {
		t.Fatalf("busy status=%d retry=%q body=%s", rr.Code, rr.Header().Get("Retry-After"), rr.Body.String())
	}
	response := rr.Body.String()
	if !strings.Contains(response, `"code":"retryable_busy"`) || strings.Contains(strings.ToLower(response), "sqlite") {
		t.Fatalf("unexpected busy response: %s", response)
	}
}
func TestRelaySignedEventAndSignatureFailure(t *testing.T) {
	f := newRelayHTTPFixture(t)
	epoch := pairRelayHTTP(t, f)
	body := []byte(`{"schema_version":1,"event_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","package_name":"com.paytm.business","posted_at_ms":1788244200000,"title":"Payment Received on Paytm for Business","text":"₹100.00 Received from Test"}`)
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, signedRelayRequest(t, f, relay.EventPath, body, epoch))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"status":"ignored"`) {
		t.Fatalf("event status=%d body=%s", rr.Code, rr.Body.String())
	}

	bad := signedRelayRequest(t, f, relay.EventPath, body, epoch)
	bad.Header.Set(relaySignatureHeader, base64.StdEncoding.EncodeToString([]byte("bad")))
	rr = httptest.NewRecorder()
	f.handler.ServeHTTP(rr, bad)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("bad signature status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRelaySignedBatchAcceptsAndDeduplicatesEvents(t *testing.T) {
	f := newRelayHTTPFixture(t)
	epoch := pairRelayHTTP(t, f)
	eventA := map[string]any{
		"schema_version": 1, "event_id": strings.Repeat("a", 64),
		"package_name": "com.paytm.business", "posted_at_ms": f.now.UnixMilli(),
		"title": "Payment Received on Paytm for Business", "text": "Received Rs. 100.00 from Alice",
	}
	eventB := map[string]any{
		"schema_version": 1, "event_id": strings.Repeat("b", 64),
		"package_name": "com.paytm.business", "posted_at_ms": f.now.UnixMilli(),
		"title": "Payment Received on Paytm for Business", "text": "Received Rs. 100.00 from Bob",
	}
	body, _ := json.Marshal(map[string]any{"schema_version": 1, "events": []any{eventA, eventB}})

	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, signedRelayRequest(t, f, relay.EventBatchPath, body, epoch))
	if rr.Code != http.StatusOK {
		t.Fatalf("batch status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"event_id":"`+strings.Repeat("a", 64)+`"`) ||
		!strings.Contains(rr.Body.String(), `"event_id":"`+strings.Repeat("b", 64)+`"`) {
		t.Fatalf("batch response missing source ids: %s", rr.Body.String())
	}
	var count int
	if err := f.db.SQL.QueryRow(`SELECT COUNT(*) FROM relay_events`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("relay events=%d err=%v", count, err)
	}

	rr = httptest.NewRecorder()
	f.handler.ServeHTTP(rr, signedRelayRequest(t, f, relay.EventBatchPath, body, epoch))
	if rr.Code != http.StatusOK || strings.Count(rr.Body.String(), `"duplicate":true`) != 2 {
		t.Fatalf("batch retry status=%d body=%s", rr.Code, rr.Body.String())
	}
	if err := f.db.SQL.QueryRow(`SELECT COUNT(*) FROM relay_events`).Scan(&count); err != nil || count != 2 {
		t.Fatalf("relay events after retry=%d err=%v", count, err)
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

func TestRelayDeviceCannotSelfRevoke(t *testing.T) {
	f := newRelayHTTPFixture(t)
	req := httptest.NewRequest(http.MethodDelete, relay.DevicePath, nil)
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("self-revoke status=%d body=%s", rr.Code, rr.Body.String())
	}
}
