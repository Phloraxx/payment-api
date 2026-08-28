package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestOperatorV2RequiresOperatorAuthAndReturnsTypedViews(t *testing.T) {
	var paymentID string
	app := apiTestFactory(t, func(app *tests.TestApp, paymentService *payments.Service) {
		payment, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 250, PaymentAccount: "kotak"})
		if err != nil {
			t.Fatal(err)
		}
		paymentID = payment.ID
		record, err := app.FindRecordById("payments", payment.ID)
		if err != nil {
			t.Fatal(err)
		}
		record.Set("payer_name", "Sensitive Payer")
		record.Set("rrn", "123456789012")
		if err := app.Save(record); err != nil {
			t.Fatal(err)
		}
	})
	defer app.Cleanup()
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	operator := core.NewRecord(users)
	operator.SetEmail("operator@example.com")
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

	unauth, err := server.Client().Get(server.URL + "/api/operator/v2/overview")
	if err != nil {
		t.Fatal(err)
	}
	_ = unauth.Body.Close()
	if unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauth status=%d", unauth.StatusCode)
	}
	overview := operatorGet(t, server, token, "/api/operator/v2/overview")
	for _, want := range []string{`"paymentCounts"`, `"recentPayments"`, paymentID} {
		if !strings.Contains(overview, want) {
			t.Fatalf("overview missing %q: %s", want, overview)
		}
	}
	if strings.Contains(overview, "Sensitive Payer") || strings.Contains(overview, "123456789012") {
		t.Fatalf("overview leaked payment evidence: %s", overview)
	}

	list := operatorGet(t, server, token, "/api/operator/v2/payments?status=pending")
	if !strings.Contains(list, paymentID) || strings.Contains(list, "Sensitive Payer") || strings.Contains(list, "123456789012") {
		t.Fatalf("payment list contract=%s", list)
	}

	detail := operatorGet(t, server, token, "/api/operator/v2/payments/"+paymentID)
	if !strings.Contains(detail, "Sensitive Payer") || !strings.Contains(detail, "123456789012") {
		t.Fatalf("operator detail missing evidence: %s", detail)
	}

	smsCollection, err := app.FindCollectionByNameOrId("sms_events")
	if err != nil {
		t.Fatal(err)
	}
	smsRecord := core.NewRecord(smsCollection)
	smsRecord.Set("source", "android_webhook")
	smsRecord.Set("payment_account", "kotak")
	smsRecord.Set("body", "RAW SECRET SMS BODY MUST NOT LEAK")
	smsRecord.Set("message_time", time.Now().UTC())
	smsRecord.Set("amount", 25001)
	smsRecord.Set("rrn", "998877665544")
	smsRecord.Set("payer_name", "Review Payer")
	smsRecord.Set("processing_status", "unmatched")
	if err := app.Save(smsRecord); err != nil {
		t.Fatal(err)
	}
	reviewCollection, err := app.FindCollectionByNameOrId("review_cases")
	if err != nil {
		t.Fatal(err)
	}
	reviewRecord := core.NewRecord(reviewCollection)
	reviewRecord.Set("kind", "unmatched")
	reviewRecord.Set("status", "open")
	reviewRecord.Set("severity", "warning")
	reviewRecord.Set("sms_event", smsRecord.Id)
	reviewRecord.Set("reason", "Evidence requires operator review")
	reviewRecord.Set("opened_at", time.Now().UTC())
	if err := app.Save(reviewRecord); err != nil {
		t.Fatal(err)
	}
	reviewDetail := operatorGet(t, server, token, "/api/operator/v2/reviews/"+reviewRecord.Id)
	for _, want := range []string{`"kind":"sms"`, `"reference":"998877665544"`, `"payerName":"Review Payer"`} {
		if !strings.Contains(reviewDetail, want) {
			t.Fatalf("review detail missing %q: %s", want, reviewDetail)
		}
	}
	if strings.Contains(reviewDetail, "RAW SECRET SMS BODY MUST NOT LEAK") {
		t.Fatalf("review detail leaked raw body: %s", reviewDetail)
	}

	badKindReq, _ := http.NewRequest(http.MethodGet, server.URL+"/api/operator/v2/records/not-real", nil)
	badKindReq.Header.Set("Authorization", "Bearer "+token)
	badKindRes, err := server.Client().Do(badKindReq)
	if err != nil {
		t.Fatal(err)
	}
	_ = badKindRes.Body.Close()
	if badKindRes.StatusCode != http.StatusNotFound {
		t.Fatalf("invalid operational kind status=%d", badKindRes.StatusCode)
	}

	unauthCancel, _ := http.NewRequest(http.MethodPost, server.URL+"/api/operator/v2/payments/"+paymentID+"/cancel", strings.NewReader(`{}`))
	unauthCancel.Header.Set("Content-Type", "application/json")
	unauthCancelRes, err := server.Client().Do(unauthCancel)
	if err != nil {
		t.Fatal(err)
	}
	_ = unauthCancelRes.Body.Close()
	if unauthCancelRes.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauth cancel status=%d", unauthCancelRes.StatusCode)
	}
	cancelled := operatorPost(t, server, token, "/api/operator/v2/payments/"+paymentID+"/cancel", `{}`)
	if !strings.Contains(cancelled, `"status":"cancelled"`) {
		t.Fatalf("cancel contract=%s", cancelled)
	}
}

func operatorGet(t *testing.T, server *httptest.Server, token, path string) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status=%d body=%s", path, res.StatusCode, body)
	}
	return string(body)
}

func operatorPost(t *testing.T, server *httptest.Server, token, path, body string) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, server.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	payload, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST %s status=%d body=%s", path, res.StatusCode, payload)
	}
	return string(payload)
}
