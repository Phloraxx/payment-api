package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
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
	var payablePaise int64
	app := apiTestFactory(t, func(app *tests.TestApp, paymentService *payments.Service) {
		payment, _, err := paymentService.Create(payments.CreateInput{AmountRupees: 250, PaymentAccount: "kotak", ExternalID: "ORIGINAL-ORDER", Metadata: map[string]any{"origin": "create"}})
		if err != nil {
			t.Fatal(err)
		}
		paymentID = payment.ID
		payablePaise = payment.PayablePaise
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

	paged := operatorGet(t, server, token, "/api/operator/v2/payments?status=pending&limit=1&offset=0")
	for _, want := range []string{`"total":1`, `"limit":1`, `"offset":0`} {
		if !strings.Contains(paged, want) {
			t.Fatalf("paged list missing %q: %s", want, paged)
		}
	}

	badFilterReq, _ := http.NewRequest(http.MethodGet, server.URL+"/api/operator/v2/payments?account=unknown", nil)
	badFilterReq.Header.Set("Authorization", "Bearer "+token)
	badFilterRes, err := server.Client().Do(badFilterReq)
	if err != nil {
		t.Fatal(err)
	}
	badFilterBody, _ := io.ReadAll(badFilterRes.Body)
	_ = badFilterRes.Body.Close()
	if badFilterRes.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad filter status=%d body=%s", badFilterRes.StatusCode, badFilterBody)
	}

	profileBody := `{"displayName":"Workshop registration","customerName":"Sourav P Bijoy","customerEmail":"sourav@example.com","customerPhone":"+91 9000000000","description":"IEEE workshop","adminNote":"private operator note","tags":["event","S7"],"customFields":{"semester":"S7"}}`
	unauthPut, _ := http.NewRequest(http.MethodPut, server.URL+"/api/operator/v2/payments/"+paymentID+"/details", strings.NewReader(profileBody))
	unauthPut.Header.Set("Content-Type", "application/json")
	unauthPutRes, err := server.Client().Do(unauthPut)
	if err != nil {
		t.Fatal(err)
	}
	_ = unauthPutRes.Body.Close()
	if unauthPutRes.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauth put status=%d", unauthPutRes.StatusCode)
	}

	updated := operatorPut(t, server, token, "/api/operator/v2/payments/"+paymentID+"/details", profileBody)
	for _, want := range []string{`"displayName":"Workshop registration"`, `"customerName":"Sourav P Bijoy"`, `"status":"pending"`, `"rrn":"123456789012"`, `"externalId":"ORIGINAL-ORDER"`, `"origin":"create"`} {
		if !strings.Contains(updated, want) {
			t.Fatalf("updated detail missing %q: %s", want, updated)
		}
	}
	if !strings.Contains(updated, `"payableAmountPaise":`+strconv.FormatInt(payablePaise, 10)) {
		t.Fatalf("updated detail changed amount: %s", updated)
	}
	searched := operatorGet(t, server, token, "/api/operator/v2/payments?q=Sourav&account=kotak&status=pending")
	if !strings.Contains(searched, paymentID) || !strings.Contains(searched, `"total":1`) {
		t.Fatalf("search contract=%s", searched)
	}

	for _, malicious := range []string{`{"status":"paid"}`, `{"payableAmountPaise":1}`, `{"rrn":"000000000000"}`, `{"paymentAccount":"slice"}`, `{"externalId":"tampered"}`, `{"metadata":{"tampered":true}}`} {
		req, _ := http.NewRequest(http.MethodPut, server.URL+"/api/operator/v2/payments/"+paymentID+"/details", strings.NewReader(malicious))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		res, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		_ = res.Body.Close()
		if res.StatusCode != http.StatusBadRequest {
			t.Fatalf("protected update %s status=%d body=%s", malicious, res.StatusCode, body)
		}
	}
	protected, err := app.FindRecordById("payments", paymentID)
	if err != nil {
		t.Fatal(err)
	}
	if protected.GetString("status") != "pending" || int64(protected.GetInt("payable_amount")) != payablePaise || protected.GetString("rrn") != "123456789012" || protected.GetString("payment_account") != "kotak" {
		t.Fatalf("protected financial fields mutated: status=%q amount=%d rrn=%q account=%q", protected.GetString("status"), protected.GetInt("payable_amount"), protected.GetString("rrn"), protected.GetString("payment_account"))
	}
	audits, err := app.FindRecordsByFilter("audit_events", "entity_id = {:id} && action = 'payment.profile.updated'", "created", 10, 0, map[string]any{"id": paymentID})
	if err != nil {
		t.Fatal(err)
	}
	if len(audits) != 1 || audits[0].GetString("actor_email") != "operator@example.com" {
		t.Fatalf("profile audit=%v", audits)
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

func operatorPut(t *testing.T, server *httptest.Server, token, path, body string) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, server.URL+path, strings.NewReader(body))
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
		t.Fatalf("PUT %s status=%d body=%s", path, res.StatusCode, payload)
	}
	return string(payload)
}
