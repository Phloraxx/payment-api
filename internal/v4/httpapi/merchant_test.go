package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/auth"
	"github.com/Phloraxx/payment-api/internal/v4/payments"
	"github.com/Phloraxx/payment-api/internal/v4/profiles"
	"github.com/Phloraxx/payment-api/internal/v4/storage"
)

type merchantFixture struct {
	handler *MerchantHandler
	auth    *auth.Service
	pay     *payments.Service
	key     auth.APIKey
	db      *storage.DB
}

func newMerchantFixture(t *testing.T) merchantFixture {
	t.Helper()
	db, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "paygate.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	profileService := profiles.NewService(db)
	if _, err := profileService.Upsert(ctx, profiles.UpsertInput{
		ID: "paytm", Label: "Paytm", UPIID: "paygate@paytm", PayeeName: "PayGate",
		Parser: "paytm_notification", Enabled: true,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := profileService.Activate(ctx, "paytm"); err != nil {
		t.Fatal(err)
	}
	authService := auth.NewService(db)
	key, err := authService.CreateAPIKey(ctx, "test merchant")
	if err != nil {
		t.Fatal(err)
	}
	paymentService := payments.NewService(db)
	paymentService.Now = func() time.Time { return time.Date(2026, 9, 1, 7, 30, 0, 0, time.UTC) }
	return merchantFixture{
		handler: NewMerchantHandler(authService, paymentService), auth: authService,
		pay: paymentService, key: key, db: db,
	}
}

func merchantRequest(t *testing.T, f merchantFixture, method, path, body, idem string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.key.Secret)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if idem != "" {
		req.Header.Set("Idempotency-Key", idem)
	}
	recorder := httptest.NewRecorder()
	f.handler.ServeHTTP(recorder, req)
	return recorder
}

func decodePaymentResponse(t *testing.T, recorder *httptest.ResponseRecorder) merchantPaymentResponse {
	t.Helper()
	var response merchantPaymentResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
	return response
}

func TestMerchantCreatePaymentContract(t *testing.T) {
	f := newMerchantFixture(t)
	body := `{"amount":100,"name":"Sourav P Bijoy","external_id":"evt_hardware_security_2026","metadata":{"registration_id":"reg_284"}}`
	recorder := merchantRequest(t, f, http.MethodPost, "/v1/payments", body, "create-reg-284")
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	response := decodePaymentResponse(t, recorder)
	if response.Name != "Sourav P Bijoy" || response.ExternalID != "evt_hardware_security_2026" || response.Status != "pending" {
		t.Fatalf("response = %+v", response)
	}
	if response.RequestedAmount != "100.00" || response.Adjustment == "0.00" || response.Currency != "INR" {
		t.Fatalf("money response = %+v", response)
	}
	parsed, err := url.Parse(response.UPIURI)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme != "upi" || parsed.Host != "pay" || parsed.Query().Get("pa") != "paygate@paytm" ||
		parsed.Query().Get("am") != response.PayableAmount || parsed.Query().Get("tn") != response.TransactionNote {
		t.Fatalf("upi_uri = %q", response.UPIURI)
	}
	if response.TransactionNote != "PayGate "+response.ID ||
		strings.Contains(response.TransactionNote, response.Name) ||
		strings.Contains(response.TransactionNote, "reg_284") {
		t.Fatalf("unsafe transaction note = %q", response.TransactionNote)
	}
	if bytes.Contains(recorder.Body.Bytes(), []byte(`"collection_profile"`)) || bytes.Contains(recorder.Body.Bytes(), []byte(`"paymentAccount"`)) {
		t.Fatalf("internal routing leaked: %s", recorder.Body.String())
	}
	if response.Payer != nil || response.PaidAt != nil {
		t.Fatalf("pending payment unexpectedly contains payer/paid_at: %+v", response)
	}
}

func TestMerchantCreateReplayAndConflict(t *testing.T) {
	f := newMerchantFixture(t)
	body := `{"amount":100,"name":"Sourav P Bijoy","external_id":"evt_1"}`
	first := merchantRequest(t, f, http.MethodPost, "/v1/payments", body, "idem-1")
	if first.Code != http.StatusCreated {
		t.Fatalf("first status=%d body=%s", first.Code, first.Body.String())
	}
	one := decodePaymentResponse(t, first)
	second := merchantRequest(t, f, http.MethodPost, "/v1/payments", body, "idem-1")
	if second.Code != http.StatusOK {
		t.Fatalf("replay status=%d body=%s", second.Code, second.Body.String())
	}
	two := decodePaymentResponse(t, second)
	if one.ID != two.ID || one.PayableAmount != two.PayableAmount || one.UPIURI != two.UPIURI || one.TransactionNote != two.TransactionNote {
		t.Fatalf("replay changed payment: one=%+v two=%+v", one, two)
	}
	conflict := merchantRequest(t, f, http.MethodPost, "/v1/payments", `{"amount":200,"name":"Sourav P Bijoy","external_id":"evt_1"}`, "idem-1")
	if conflict.Code != http.StatusConflict || !strings.Contains(conflict.Body.String(), "idempotency_conflict") {
		t.Fatalf("conflict status=%d body=%s", conflict.Code, conflict.Body.String())
	}
}

func TestMerchantIdempotencySurvivesAPIKeyRotation(t *testing.T) {
	f := newMerchantFixture(t)
	body := `{"amount":100,"name":"Sourav P Bijoy","external_id":"evt_1"}`
	first := merchantRequest(t, f, http.MethodPost, "/v1/payments", body, "same-request")
	one := decodePaymentResponse(t, first)
	rotated, err := f.auth.CreateAPIKey(context.Background(), "rotated merchant key")
	if err != nil {
		t.Fatal(err)
	}
	f.key = rotated
	second := merchantRequest(t, f, http.MethodPost, "/v1/payments", body, "same-request")
	if second.Code != http.StatusOK {
		t.Fatalf("rotated replay status=%d body=%s", second.Code, second.Body.String())
	}
	two := decodePaymentResponse(t, second)
	if one.ID != two.ID {
		t.Fatalf("API key rotation broke idempotency: %s != %s", one.ID, two.ID)
	}
}

func TestMerchantExternalEventIDIsNotUnique(t *testing.T) {
	f := newMerchantFixture(t)
	body1 := `{"amount":100,"name":"Sourav P Bijoy","external_id":"evt_shared"}`
	body2 := `{"amount":100,"name":"Another Person","external_id":"evt_shared"}`
	one := merchantRequest(t, f, http.MethodPost, "/v1/payments", body1, "person-1")
	two := merchantRequest(t, f, http.MethodPost, "/v1/payments", body2, "person-2")
	if one.Code != http.StatusCreated || two.Code != http.StatusCreated {
		t.Fatalf("statuses one=%d two=%d", one.Code, two.Code)
	}
	first := decodePaymentResponse(t, one)
	second := decodePaymentResponse(t, two)
	if first.ID == second.ID || first.ExternalID != "evt_shared" || second.ExternalID != "evt_shared" {
		t.Fatalf("unexpected payments first=%+v second=%+v", first, second)
	}
}

func TestMerchantGetAndCancel(t *testing.T) {
	f := newMerchantFixture(t)
	create := merchantRequest(t, f, http.MethodPost, "/v1/payments", `{"amount":100,"name":"Sourav","external_id":"evt_1"}`, "idem-get")
	payment := decodePaymentResponse(t, create)
	got := merchantRequest(t, f, http.MethodGet, "/v1/payments/"+payment.ID, "", "")
	gotPayment := decodePaymentResponse(t, got)
	if got.Code != http.StatusOK || gotPayment.ID != payment.ID || gotPayment.TransactionNote != payment.TransactionNote || gotPayment.UPIURI != payment.UPIURI {
		t.Fatalf("get status=%d body=%s", got.Code, got.Body.String())
	}
	cancelled := merchantRequest(t, f, http.MethodPost, "/v1/payments/"+payment.ID+"/cancel", "", "")
	cancelledPayment := decodePaymentResponse(t, cancelled)
	if cancelled.Code != http.StatusOK || cancelledPayment.Status != "cancelled" ||
		cancelledPayment.TransactionNote != payment.TransactionNote || cancelledPayment.UPIURI != payment.UPIURI {
		t.Fatalf("cancel status=%d body=%s", cancelled.Code, cancelled.Body.String())
	}
	again := merchantRequest(t, f, http.MethodPost, "/v1/payments/"+payment.ID+"/cancel", "", "")
	againPayment := decodePaymentResponse(t, again)
	if again.Code != http.StatusOK || againPayment.Status != "cancelled" ||
		againPayment.TransactionNote != payment.TransactionNote || againPayment.UPIURI != payment.UPIURI {
		t.Fatalf("idempotent cancel status=%d body=%s", again.Code, again.Body.String())
	}
}
func TestMerchantRejectsInternalRoutingAndMissingAuth(t *testing.T) {
	f := newMerchantFixture(t)
	request := httptest.NewRequest(http.MethodPost, "/v1/payments", strings.NewReader(`{"amount":100,"name":"Sourav"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "unauthorized")
	recorder := httptest.NewRecorder()
	f.handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	body := `{"amount":100,"name":"Sourav","paymentAccount":"paytm"}`
	internal := merchantRequest(t, f, http.MethodPost, "/v1/payments", body, "internal-field")
	if internal.Code != http.StatusBadRequest || !strings.Contains(internal.Body.String(), "invalid_request") {
		t.Fatalf("internal field status=%d body=%s", internal.Code, internal.Body.String())
	}
	missingIdem := merchantRequest(t, f, http.MethodPost, "/v1/payments", `{"amount":100,"name":"Sourav"}`, "")
	if missingIdem.Code != http.StatusBadRequest || !strings.Contains(missingIdem.Body.String(), "idempotency_key_required") {
		t.Fatalf("missing idempotency status=%d body=%s", missingIdem.Code, missingIdem.Body.String())
	}
}

func TestMerchantRejectsMalformedBodiesAndRevokedKey(t *testing.T) {
	f := newMerchantFixture(t)
	bad := merchantRequest(t, f, http.MethodPost, "/v1/payments", `{"amount":100.5,"name":"Sourav"}`, "bad-amount")
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("fractional amount status=%d body=%s", bad.Code, bad.Body.String())
	}
	trailing := merchantRequest(t, f, http.MethodPost, "/v1/payments", `{"amount":100,"name":"Sourav"} {}`, "trailing")
	if trailing.Code != http.StatusBadRequest {
		t.Fatalf("trailing JSON status=%d body=%s", trailing.Code, trailing.Body.String())
	}
	if err := f.auth.RevokeAPIKey(context.Background(), f.key.ID); err != nil {
		t.Fatal(err)
	}
	revoked := merchantRequest(t, f, http.MethodGet, "/v1/payments/not-real", "", "")
	if revoked.Code != http.StatusUnauthorized {
		t.Fatalf("revoked key status=%d body=%s", revoked.Code, revoked.Body.String())
	}
}

func TestMerchantMissingPaymentIs404(t *testing.T) {
	f := newMerchantFixture(t)
	recorder := merchantRequest(t, f, http.MethodGet, "/v1/payments/pay_missing", "", "")
	if recorder.Code != http.StatusNotFound || !strings.Contains(recorder.Body.String(), "payment_not_found") {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
func TestMerchantPaidResponseSeparatesRequestedNameFromActualPayer(t *testing.T) {
	f := newMerchantFixture(t)
	create := merchantRequest(t, f, http.MethodPost, "/v1/payments", `{"amount":100,"name":"Sourav P Bijoy","external_id":"evt_1"}`, "paid-response")
	payment := decodePaymentResponse(t, create)
	paidAt := time.Date(2026, 9, 1, 7, 32, 10, 0, time.UTC)
	if _, err := f.db.SQL.Exec(`UPDATE payments SET status='paid',paid_at=?,payer_name=?,payer_upi_id=? WHERE id=?`,
		paidAt.UnixMilli(), "Bijoy P", "bijoy@okaxis", payment.ID); err != nil {
		t.Fatal(err)
	}
	got := merchantRequest(t, f, http.MethodGet, "/v1/payments/"+payment.ID, "", "")
	if got.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", got.Code, got.Body.String())
	}
	response := decodePaymentResponse(t, got)
	if response.Name != "Sourav P Bijoy" || response.Payer == nil || response.Payer.Name == nil || *response.Payer.Name != "Bijoy P" {
		t.Fatalf("payer/name response = %+v", response)
	}
	if response.Payer.UPIID == nil || *response.Payer.UPIID != "bijoy@okaxis" || response.PaidAt == nil || !response.PaidAt.Equal(paidAt) {
		t.Fatalf("paid response = %+v", response)
	}
}
