package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/evidenceshadow"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

func TestGoogleMessagesShadowMetricsAndQRRoutesRetired(t *testing.T) {
	app := apiTestFactoryWithGMessages(t)
	defer app.Cleanup()
	now := time.Now().UTC()
	devices, err := app.FindCollectionByNameOrId("relay_devices")
	if err != nil {
		t.Fatal(err)
	}
	device := core.NewRecord(devices)
	device.Set("device_id", strings.Repeat("a", 64))
	device.Set("name", "Shadow Phone")
	device.Set("public_key_pem", "test-key")
	device.Set("enabled", true)
	if err := app.Save(device); err != nil {
		t.Fatal(err)
	}
	relays, err := app.FindCollectionByNameOrId("relay_events")
	if err != nil {
		t.Fatal(err)
	}
	relay := core.NewRecord(relays)
	relay.Set("device", device.Id)
	relay.Set("event_id", strings.Repeat("b", 64))
	relay.Set("kind", "notification")
	relay.Set("app_package", evidenceshadow.GoogleMessagesPackage)
	relay.Set("notification_when", now)
	relay.Set("processing_status", "shadow_observed")
	relay.Set("provider_result", map[string]any{
		"provider":      evidenceshadow.Provider,
		"parser":        "bank_sms_v1",
		"parseStatus":   "complete",
		"amountPaise":   10001,
		"referenceHash": evidenceshadow.HashReference("123456789012"),
	})
	if err := app.Save(relay); err != nil {
		t.Fatal(err)
	}

	smsCollection, err := app.FindCollectionByNameOrId("sms_events")
	if err != nil {
		t.Fatal(err)
	}
	sms := core.NewRecord(smsCollection)
	sms.Set("source", "gmessages")
	sms.Set("payment_account", "kotak")
	sms.Set("body", "bank credit")
	sms.Set("message_time", now)
	sms.Set("amount", 10001)
	sms.Set("rrn", "123456789012")
	sms.Set("processing_status", "matched")
	if err := app.Save(sms); err != nil {
		t.Fatal(err)
	}

	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	operator := core.NewRecord(users)
	operator.SetEmail("shadow-operator@example.com")
	operator.SetPassword("operator-password-123")
	operator.SetVerified(true)
	if err := app.Save(operator); err != nil {
		t.Fatal(err)
	}
	token, err := operator.NewAuthToken()
	if err != nil {
		t.Fatal(err)
	}

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
	unauth, err := server.Client().Get(server.URL + "/api/operator/v2/evidence-shadow/google-messages")
	if err != nil {
		t.Fatal(err)
	}
	_ = unauth.Body.Close()
	if unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauth status=%d", unauth.StatusCode)
	}

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/api/operator/v2/evidence-shadow/google-messages?days=14", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("metrics status=%d body=%s", res.StatusCode, body)
	}
	payload := string(body)
	for _, want := range []string{`"androidObserved":1`, `"libgmObserved":1`, `"exactMatches":1`, `"removalReady":false`} {
		if !strings.Contains(payload, want) {
			t.Fatalf("metrics missing %q: %s", want, payload)
		}
	}

	for _, path := range []string{"/api/connector/gmessages/pair/qr", "/api/connector/gmessages/pair/qr/refresh", "/api/connector/gmessages/pair", "/api/connector/gmessages/pair/refresh"} {
		post, _ := http.NewRequest(http.MethodPost, server.URL+path, strings.NewReader(`{}`))
		post.Header.Set("Authorization", "Bearer "+token)
		post.Header.Set("Content-Type", "application/json")
		postRes, err := server.Client().Do(post)
		if err != nil {
			t.Fatal(err)
		}
		_ = postRes.Body.Close()
		if postRes.StatusCode != http.StatusNotFound {
			t.Fatalf("retired QR route %s status=%d", path, postRes.StatusCode)
		}
	}
	googlePair, _ := http.NewRequest(http.MethodPost, server.URL+"/api/connector/gmessages/pair/google", strings.NewReader(`{"cookieData":""}`))
	googlePair.Header.Set("Authorization", "Bearer "+token)
	googlePair.Header.Set("Content-Type", "application/json")
	googlePairRes, err := server.Client().Do(googlePair)
	if err != nil {
		t.Fatal(err)
	}
	_ = googlePairRes.Body.Close()
	if googlePairRes.StatusCode == http.StatusNotFound {
		t.Fatalf("Google account pairing route was removed with QR fallback")
	}
}
