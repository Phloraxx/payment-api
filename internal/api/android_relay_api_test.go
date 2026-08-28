package api

import (
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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/androidrelay"
	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestAndroidRelayEnrollmentAndSignedEventAPI(t *testing.T) {
	app := apiTestFactoryWithConfig(t, func(cfg *config.Config) {
		cfg.AndroidRelayPairingSecret = "android-relay-pairing-secret-long-enough"
		cfg.AndroidRelayEnrollmentEnabled = true
	}, nil)
	defer app.Cleanup()
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error { return nil }); err != nil {
		t.Fatal(err)
	}
	mux, err := serveEvent.Router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(mux)
	defer server.Close()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	pemValue := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	sum := sha256.Sum256(der)
	deviceID := hex.EncodeToString(sum[:])
	enrollBody, _ := json.Marshal(androidrelay.EnrollmentInput{DeviceID: deviceID, Name: "API Test Phone", PublicKeyPEM: pemValue, AppVersion: "0.3.0"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/relay/v1/enroll", strings.NewReader(string(enrollBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Pairing-Secret", "android-relay-pairing-secret-long-enough")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("enroll status=%d", resp.StatusCode)
	}

	heartbeatBody := []byte(`{"schemaVersion":1,"appVersion":"0.3.0","notificationAccess":true,"listenerConnected":true,"pendingCount":0,"failedCount":0}`)
	heartbeatTs := strconv.FormatInt(time.Now().UnixMilli(), 10)
	heartbeatCanonical := androidrelay.CanonicalRequest(http.MethodPost, "/api/relay/v1/heartbeat", heartbeatTs, heartbeatBody)
	heartbeatDigest := sha256.Sum256([]byte(heartbeatCanonical))
	heartbeatSig, err := ecdsa.SignASN1(rand.Reader, priv, heartbeatDigest[:])
	if err != nil {
		t.Fatal(err)
	}
	req, _ = http.NewRequest(http.MethodPost, server.URL+"/api/relay/v1/heartbeat", strings.NewReader(string(heartbeatBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PayGate-Relay-Device", deviceID)
	req.Header.Set("X-PayGate-Relay-Time", heartbeatTs)
	req.Header.Set("X-PayGate-Relay-Signature", base64.StdEncoding.EncodeToString(heartbeatSig))
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("heartbeat status=%d", resp.StatusCode)
	}

	eventBody := []byte(`{"schemaVersion":1,"eventId":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd","kind":"notification","capturedAtMs":1787839690718,"notification":{"packageName":"com.google.android.apps.nbu.paisa.user","appName":"Google Pay","key":"test","id":1,"postTimeMs":1787839690356,"whenMs":1787839690313,"title":"You received money"}}`)
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	canonical := androidrelay.CanonicalRequest(http.MethodPost, "/api/relay/v1/events", ts, eventBody)
	digest := sha256.Sum256([]byte(canonical))
	sig, err := ecdsa.SignASN1(rand.Reader, priv, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	req, _ = http.NewRequest(http.MethodPost, server.URL+"/api/relay/v1/events", strings.NewReader(string(eventBody)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-PayGate-Relay-Device", deviceID)
	req.Header.Set("X-PayGate-Relay-Time", ts)
	req.Header.Set("X-PayGate-Relay-Signature", base64.StdEncoding.EncodeToString(sig))
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("event status=%d", resp.StatusCode)
	}
	if count, _ := app.CountRecords("relay_events"); count != 1 {
		t.Fatalf("relay events=%d", count)
	}
}

func TestAndroidRelayEnrollmentDisabledEvenWithStoredSecret(t *testing.T) {
	app := apiTestFactoryWithConfig(t, func(cfg *config.Config) {
		cfg.AndroidRelayPairingSecret = "android-relay-pairing-secret-long-enough"
		cfg.AndroidRelayEnrollmentEnabled = false
	}, nil)
	defer app.Cleanup()
	router, err := apis.NewRouter(app)
	if err != nil {
		t.Fatal(err)
	}
	serveEvent := &core.ServeEvent{App: app, Router: router}
	if err := app.OnServe().Trigger(serveEvent, func(e *core.ServeEvent) error { return nil }); err != nil {
		t.Fatal(err)
	}
	mux, err := serveEvent.Router.BuildMux()
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(mux)
	defer server.Close()

	req, _ := http.NewRequest(http.MethodPost, server.URL+"/api/relay/v1/enroll", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Pairing-Secret", "android-relay-pairing-secret-long-enough")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("disabled enrollment status=%d; want 404", resp.StatusCode)
	}
}
