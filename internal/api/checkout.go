package api

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/money"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/store"
	"github.com/pocketbase/pocketbase/core"
)

var (
	checkoutRequestID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	checkoutPaymentID = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)
)

type checkoutBucket struct {
	start time.Time
	count int
}

type checkoutQuota struct {
	mu                      sync.Mutex
	perIP                   map[string]checkoutBucket
	global                  checkoutBucket
	perIPLimit, globalLimit int
	perIPWindow             time.Duration
	globalWindow            time.Duration
}

type checkoutLimiterSet struct{ create, status checkoutQuota }

func newCheckoutLimiterSet() *checkoutLimiterSet {
	return &checkoutLimiterSet{
		create: checkoutQuota{perIP: map[string]checkoutBucket{}, perIPLimit: 5, globalLimit: 60, perIPWindow: 5 * time.Minute, globalWindow: time.Minute},
		status: checkoutQuota{perIP: map[string]checkoutBucket{}, perIPLimit: 180, globalLimit: 1800, perIPWindow: time.Minute, globalWindow: time.Minute},
	}
}

func refreshCheckoutBucket(bucket checkoutBucket, now time.Time, window time.Duration) checkoutBucket {
	if bucket.start.IsZero() || now.Sub(bucket.start) >= window {
		return checkoutBucket{start: now}
	}
	return bucket
}

func checkoutRetryAfter(bucket checkoutBucket, now time.Time, window time.Duration) int {
	remaining := window - now.Sub(bucket.start)
	seconds := int((remaining + time.Second - 1) / time.Second)
	if seconds < 1 {
		return 1
	}
	return seconds
}

func (q *checkoutQuota) allow(ip string, now time.Time) (bool, int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	per := refreshCheckoutBucket(q.perIP[ip], now, q.perIPWindow)
	global := refreshCheckoutBucket(q.global, now, q.globalWindow)
	if per.count >= q.perIPLimit {
		q.perIP[ip], q.global = per, global
		return false, checkoutRetryAfter(per, now, q.perIPWindow)
	}
	if global.count >= q.globalLimit {
		q.perIP[ip], q.global = per, global
		return false, checkoutRetryAfter(global, now, q.globalWindow)
	}
	per.count++
	global.count++
	q.perIP[ip], q.global = per, global
	return true, 0
}

func (a *API) checkoutLimiters() *checkoutLimiterSet {
	a.checkoutMu.Lock()
	defer a.checkoutMu.Unlock()
	if a.checkoutLimits == nil {
		a.checkoutLimits = newCheckoutLimiterSet()
	}
	return a.checkoutLimits
}

func (a *API) checkoutEnabled() bool { return len(a.Config.CheckoutAllowedOrigins) > 0 }

func (a *API) checkoutOrigin(e *core.RequestEvent) bool {
	if !a.checkoutEnabled() {
		return false
	}
	origin := strings.TrimSpace(strings.TrimRight(e.Request.Header.Get("Origin"), "/"))
	if origin == "" {
		return true
	}
	for _, allowed := range a.Config.CheckoutAllowedOrigins {
		if origin == allowed {
			e.Response.Header().Set("Access-Control-Allow-Origin", origin)
			e.Response.Header().Set("Vary", "Origin")
			return true
		}
	}
	return false
}

func (a *API) checkoutPreflight(e *core.RequestEvent) error {
	if !a.checkoutOrigin(e) {
		return e.NotFoundError("route not found", nil)
	}
	e.Response.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	e.Response.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Idempotency-Key")
	e.Response.Header().Set("Access-Control-Max-Age", "600")
	return e.NoContent(http.StatusNoContent)
}

func checkoutError(e *core.RequestEvent, status int, code, message string) error {
	return e.JSON(status, map[string]any{"code": code, "message": message})
}

func (a *API) checkoutRequireOrigin(e *core.RequestEvent) error {
	if !a.checkoutOrigin(e) {
		return e.NotFoundError("route not found", nil)
	}
	e.Response.Header().Set("Cache-Control", "no-store")
	return nil
}

func (a *API) checkoutRateLimit(e *core.RequestEvent, create bool) error {
	limits := a.checkoutLimiters()
	quota := &limits.status
	message := "Too many payment status requests. Please wait and try again."
	if create {
		quota = &limits.create
		message = "Too many payment requests. Please wait and try again."
	}
	allowed, retry := quota.allow(e.RealIP(), time.Now().UTC())
	if !allowed {
		e.Response.Header().Set("Retry-After", strconv.Itoa(retry))
		return checkoutError(e, 429, "RATE_LIMITED", message)
	}
	return nil
}

type checkoutCreateBody struct {
	Amount         json.RawMessage `json:"amount"`
	PaymentAccount string          `json:"paymentAccount"`
}

func (a *API) checkoutPaymentAccounts(e *core.RequestEvent) error {
	if err := a.checkoutRequireOrigin(e); err != nil {
		return err
	}
	if err := a.checkoutRateLimit(e, false); err != nil {
		return err
	}
	defaultAccount := strings.ToLower(strings.TrimSpace(a.Config.DefaultPaymentAccount))
	if defaultAccount == "" {
		defaultAccount = "kotak"
	}
	accounts, err := a.paymentAccountOptions()
	if err != nil {
		return checkoutError(e, 502, "PAYGATE_UNAVAILABLE", "Payment service is temporarily unavailable.")
	}
	return e.JSON(http.StatusOK, map[string]any{"default": defaultAccount, "accounts": accounts})
}

func (a *API) checkoutCreatePayment(e *core.RequestEvent) error {
	if err := a.checkoutRequireOrigin(e); err != nil {
		return err
	}
	if ct := strings.ToLower(strings.TrimSpace(strings.Split(e.Request.Header.Get("Content-Type"), ";")[0])); ct != "application/json" {
		return checkoutError(e, 415, "INVALID_CONTENT_TYPE", "Content-Type must be application/json.")
	}
	requestID := strings.ToLower(strings.TrimSpace(e.Request.Header.Get("Idempotency-Key")))
	if !checkoutRequestID.MatchString(requestID) {
		return checkoutError(e, 400, "INVALID_REQUEST", "requestId must be a UUID.")
	}
	var body checkoutCreateBody
	if err := decodeJSON(e, &body); err != nil {
		return checkoutError(e, 400, "INVALID_JSON", "Request body must be valid JSON.")
	}
	amount, err := money.ParseWholeRupees(body.Amount)
	if err != nil {
		return checkoutError(e, 400, "INVALID_REQUEST", "Amount must be a positive whole number of rupees.")
	}
	account := strings.ToLower(strings.TrimSpace(body.PaymentAccount))
	if account != "kotak" && account != "slice" && account != "paytm" {
		return checkoutError(e, 400, "INVALID_REQUEST", "paymentAccount must be kotak, slice, or paytm.")
	}
	if err := a.checkoutRateLimit(e, true); err != nil {
		return err
	}
	payment, replayed, err := a.Payments.CreateGuarded(payments.CreateInput{AmountRupees: amount, PaymentAccount: account, IdempotencyKey: requestID}, func(uow store.UnitOfWork) error { return a.ensurePaymentAccountReadyUoW(uow, account) })
	if err != nil {
		return a.checkoutDomainError(e, err)
	}
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
		e.Response.Header().Set("X-Idempotent-Replayed", "true")
	}
	return e.JSON(status, payments.CreateResponse(payment, a.Config))
}

func (a *API) checkoutGetPayment(e *core.RequestEvent) error {
	if err := a.checkoutRequireOrigin(e); err != nil {
		return err
	}
	id := strings.TrimSpace(e.Request.PathValue("id"))
	if !checkoutPaymentID.MatchString(id) {
		return checkoutError(e, 400, "INVALID_PAYMENT_ID", "Invalid payment ID.")
	}
	if err := a.checkoutRateLimit(e, false); err != nil {
		return err
	}
	payment, err := a.Payments.Get(id)
	if err != nil {
		return a.checkoutDomainError(e, err)
	}
	return e.JSON(http.StatusOK, payments.PublicPayment(payment))
}

func (a *API) checkoutDomainError(e *core.RequestEvent, err error) error {
	if de, ok := err.(*domain.Error); ok {
		return checkoutError(e, de.Status, de.Code, de.Message)
	}
	return checkoutError(e, 502, "PAYGATE_UNAVAILABLE", "Payment service is temporarily unavailable.")
}
