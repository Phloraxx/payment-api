package api

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/gmessages"
	"github.com/Phloraxx/payment-api/internal/money"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/sms"
	appweb "github.com/Phloraxx/payment-api/internal/web"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const (
	maxPaymentRequestBytes int64 = (1 << 20) + (64 << 10)
	maxSMSRequestBytes     int64 = 128 << 10
	maxGMessagesPairBytes  int64 = 128 << 10
)

type API struct {
	Config    config.Config
	Payments  *payments.Service
	SMS       *sms.Service
	GMessages *gmessages.Manager
}

func New(cfg config.Config, paymentService *payments.Service, smsService *sms.Service, manager *gmessages.Manager) *API {
	return &API{Config: cfg, Payments: paymentService, SMS: smsService, GMessages: manager}
}

func (a *API) Register(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.POST("/api/payments", a.createPayment).Bind(apis.BodyLimit(maxPaymentRequestBytes))
		e.Router.GET("/api/payments/{id}", a.getPayment)
		e.Router.POST("/api/payments/{id}/cancel", a.cancelPayment)
		e.Router.POST("/api/events/sms", a.ingestSMS).Bind(apis.BodyLimit(maxSMSRequestBytes))
		e.Router.POST("/api/webhook", a.ingestLegacySMS).Bind(apis.BodyLimit(maxSMSRequestBytes))
		e.Router.GET("/api/paygate/health", a.health)
		e.Router.GET("/api/config", a.getConfig)
		e.Router.GET("/api/dashboard", a.dashboard)
		e.Router.GET("/api/connector/gmessages/status", a.gmessagesStatus)
		e.Router.POST("/api/connector/gmessages/pair/google", a.gmessagesGooglePair).Bind(apis.BodyLimit(maxGMessagesPairBytes))
		e.Router.POST("/api/connector/gmessages/pair/qr", a.gmessagesPair)
		e.Router.POST("/api/connector/gmessages/pair/qr/refresh", a.gmessagesPairRefresh)
		// Backward-compatible QR aliases from the first PayGate rebuild.
		e.Router.POST("/api/connector/gmessages/pair", a.gmessagesPair)
		e.Router.POST("/api/connector/gmessages/pair/refresh", a.gmessagesPairRefresh)
		e.Router.POST("/api/connector/gmessages/reconnect", a.gmessagesReconnect)
		e.Router.DELETE("/api/connector/gmessages/pair", a.gmessagesUnpair)

		// Keep API/admin namespaces out of the SPA fallback. Unknown API routes
		// must remain real 404s rather than HTML 200 responses.
		static := apis.Static(appweb.Assets(), true)
		e.Router.GET("/{path...}", func(event *core.RequestEvent) error {
			path := strings.TrimPrefix(event.Request.URL.Path, "/")
			if path == "api" || strings.HasPrefix(path, "api/") || path == "_" || strings.HasPrefix(path, "_/") {
				return event.NotFoundError("route not found", nil)
			}
			return static(event)
		})
		return e.Next()
	})
}

type createPaymentBody struct {
	Amount     json.RawMessage `json:"amount"`
	ExternalID string          `json:"externalId"`
	Metadata   json.RawMessage `json:"metadata"`
}

func (a *API) createPayment(e *core.RequestEvent) error {
	if !a.authorizedWrite(e) {
		return e.UnauthorizedError("API key or dashboard authentication is required", nil)
	}
	var body createPaymentBody
	if err := decodeJSON(e, &body); err != nil {
		return e.BadRequestError("invalid JSON body", err)
	}
	if len(body.Amount) == 0 {
		return writeDomainError(e, domain.InvalidAmount())
	}
	amount, err := money.ParseWholeRupees(body.Amount)
	if err != nil {
		return writeDomainError(e, domain.InvalidAmount())
	}
	var metadata any
	if len(body.Metadata) > 0 && string(body.Metadata) != "null" {
		if err := json.Unmarshal(body.Metadata, &metadata); err != nil {
			return e.BadRequestError("metadata must be valid JSON", err)
		}
	}

	payment, replayed, err := a.Payments.Create(payments.CreateInput{
		AmountRupees:   amount,
		ExternalID:     strings.TrimSpace(body.ExternalID),
		Metadata:       metadata,
		IdempotencyKey: strings.TrimSpace(e.Request.Header.Get("Idempotency-Key")),
	})
	if err != nil {
		return writeDomainError(e, err)
	}
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
		e.Response.Header().Set("X-Idempotent-Replayed", "true")
	}
	return e.JSON(status, payments.CreateResponse(payment, a.Config))
}

func (a *API) getPayment(e *core.RequestEvent) error {
	payment, err := a.Payments.Get(e.Request.PathValue("id"))
	if err != nil {
		return writeDomainError(e, err)
	}
	// Public by design, but intentionally omits RRN, UPI ID, payer data and raw SMS.
	return e.JSON(http.StatusOK, payments.PublicPayment(payment))
}

func (a *API) cancelPayment(e *core.RequestEvent) error {
	if !a.authorizedWrite(e) {
		return e.UnauthorizedError("API key or dashboard authentication is required", nil)
	}
	payment, err := a.Payments.Cancel(e.Request.PathValue("id"))
	if err != nil {
		return writeDomainError(e, err)
	}
	return e.JSON(http.StatusOK, payments.PublicPayment(payment))
}

type smsBody struct {
	SMS       string `json:"sms"`
	Body      string `json:"body"`
	Sender    string `json:"sender"`
	Source    string `json:"source"`
	SourceID  string `json:"sourceId"`
	Timestamp string `json:"timestamp"`
}

func (a *API) ingestSMS(e *core.RequestEvent) error {
	if !constantTimeEqual(a.Config.SMSWebhookSecret, e.Request.Header.Get("X-Webhook-Secret")) {
		return e.UnauthorizedError("invalid webhook secret", nil)
	}
	return a.ingestSMSBody(e, false)
}

func (a *API) ingestLegacySMS(e *core.RequestEvent) error {
	if !a.Config.LegacySMSWebhookEnabled {
		return e.NotFoundError("route not found", nil)
	}
	if !constantTimeEqual(a.Config.LegacySMSWebhookSecret, e.Request.Header.Get("X-Webhook-Secret")) {
		return e.UnauthorizedError("invalid webhook secret", nil)
	}
	return a.ingestSMSBody(e, true)
}

func (a *API) ingestSMSBody(e *core.RequestEvent, legacy bool) error {
	var body smsBody
	if err := decodeJSON(e, &body); err != nil {
		return e.BadRequestError("invalid JSON body", err)
	}
	text := body.SMS
	if text == "" {
		text = body.Body
	}
	messageTime := time.Time{}
	if strings.TrimSpace(body.Timestamp) != "" {
		parsed, err := time.Parse(time.RFC3339, body.Timestamp)
		if err != nil {
			return e.BadRequestError("timestamp must be RFC3339", err)
		}
		messageTime = parsed
	}
	source := strings.TrimSpace(body.Source)
	if source == "" {
		source = "android_webhook"
	}
	if legacy {
		// The compatibility route always identifies itself as the Android relay;
		// callers cannot forge a different connector source through it.
		source = "android_webhook"
	}
	result, err := a.SMS.Ingest(sms.Input{
		Source:        source,
		SourceEventID: body.SourceID,
		Sender:        body.Sender,
		Body:          text,
		MessageTime:   messageTime,
		RawPayload: map[string]any{
			"sender":    body.Sender,
			"source":    source,
			"sourceId":  body.SourceID,
			"timestamp": body.Timestamp,
		},
	})
	if err != nil {
		// A domain parsing/matching error is persisted in sms_events before this
		// response is returned, so it remains debuggable and does not vanish.
		if _, ok := err.(*domain.Error); ok {
			return writeDomainErrorWithData(e, err, map[string]any{"event": result})
		}
		return writeDomainError(e, err)
	}
	status := http.StatusAccepted
	if result.Duplicate {
		status = http.StatusOK
	}
	return e.JSON(status, result)
}

func (a *API) health(e *core.RequestEvent) error {
	var one int
	if err := e.App.DB().NewQuery("SELECT 1").Row(&one); err != nil || one != 1 {
		return e.JSON(http.StatusServiceUnavailable, map[string]any{
			"status": "unhealthy", "ready": false, "db": "error", "connector": a.publicConnectorStatus(),
		})
	}
	return e.JSON(http.StatusOK, map[string]any{
		"status": "healthy", "ready": true, "db": "ok", "connector": a.publicConnectorStatus(),
	})
}

func (a *API) getConfig(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	return e.JSON(http.StatusOK, map[string]any{
		"upiId":                   a.Config.UPIID,
		"upiPayeeName":            a.Config.UPIPayeeName,
		"paymentTtlSeconds":       int64(a.Config.PaymentTTL / time.Second),
		"quarantineSeconds":       int64(a.Config.AmountQuarantine / time.Second),
		"webhookConfigured":       a.Config.OutgoingWebhookURL != "",
		"rateLimitsEnabled":       a.Config.RateLimitsEnabled,
		"legacySMSWebhookEnabled": a.Config.LegacySMSWebhookEnabled,
		"connector":               a.connectorStatus(),
	})
}

func (a *API) dashboard(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	stats, err := a.Payments.Stats()
	if err != nil {
		return e.InternalServerError("failed to load dashboard", err)
	}
	return e.JSON(http.StatusOK, map[string]any{"stats": stats, "connector": a.connectorStatus()})
}

func (a *API) gmessagesStatus(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	return e.JSON(http.StatusOK, a.connectorStatus())
}

type googleMessagesPairBody struct {
	CookieData string `json:"cookieData"`
}

func (a *API) gmessagesGooglePair(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	if a.GMessages == nil {
		return e.BadRequestError("Google Messages connector is unavailable", nil)
	}
	var body googleMessagesPairBody
	if err := decodeJSON(e, &body); err != nil {
		return e.BadRequestError("invalid JSON body", err)
	}
	emoji, accountEmail, err := a.GMessages.BeginGooglePair(strings.TrimSpace(body.CookieData))
	if err != nil {
		return e.BadRequestError(err.Error(), nil)
	}
	return e.JSON(http.StatusOK, map[string]any{
		"emoji":        emoji,
		"accountEmail": accountEmail,
		"status":       a.GMessages.Status(),
	})
}

func (a *API) gmessagesPair(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	if a.GMessages == nil {
		return e.BadRequestError("Google Messages connector is unavailable", nil)
	}
	qrURL, err := a.GMessages.BeginPair()
	if err != nil {
		return e.BadRequestError("failed to start Google Messages pairing", err)
	}
	return e.JSON(http.StatusOK, map[string]any{"qrUrl": qrURL, "status": a.GMessages.Status()})
}

func (a *API) gmessagesPairRefresh(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	if a.GMessages == nil {
		return e.BadRequestError("Google Messages connector is unavailable", nil)
	}
	qrURL, err := a.GMessages.RefreshPair()
	if err != nil {
		return e.BadRequestError("failed to refresh Google Messages pairing", err)
	}
	return e.JSON(http.StatusOK, map[string]any{"qrUrl": qrURL, "status": a.GMessages.Status()})
}

func (a *API) gmessagesReconnect(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	if a.GMessages == nil {
		return e.BadRequestError("Google Messages connector is unavailable", nil)
	}
	if err := a.GMessages.Reconnect(); err != nil {
		return e.BadRequestError("failed to reconnect Google Messages", err)
	}
	return e.JSON(http.StatusOK, a.GMessages.Status())
}

func (a *API) gmessagesUnpair(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	if a.GMessages == nil {
		return e.BadRequestError("Google Messages connector is unavailable", nil)
	}
	if err := a.GMessages.Unpair(); err != nil {
		return e.InternalServerError("failed to unpair Google Messages", err)
	}
	return e.JSON(http.StatusOK, a.GMessages.Status())
}

func (a *API) authorizedWrite(e *core.RequestEvent) bool {
	return a.dashboardAuth(e) || bearerMatches(a.Config.APIKey, e.Request.Header.Get("Authorization"))
}

func (a *API) dashboardAuth(e *core.RequestEvent) bool {
	return e.Auth != nil && e.Auth.Collection() != nil && e.Auth.Collection().Name == "users"
}

func (a *API) connectorStatus() gmessages.Status {
	if a.GMessages == nil {
		return gmessages.Status{Enabled: false, State: "disabled"}
	}
	return a.GMessages.Status()
}

func (a *API) publicConnectorStatus() map[string]any {
	status := a.connectorStatus()
	return map[string]any{
		"enabled":         status.Enabled,
		"state":           status.State,
		"paired":          status.Paired,
		"connected":       status.Connected,
		"phoneResponsive": status.PhoneResponsive,
		"lastConnectedAt": status.LastConnectedAt,
		"lastMessageAt":   status.LastMessageAt,
	}
}

func bearerMatches(expected, header string) bool {
	if expected == "" {
		return false
	}
	parts := strings.SplitN(strings.TrimSpace(header), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return false
	}
	return constantTimeEqual(expected, strings.TrimSpace(parts[1]))
}

func constantTimeEqual(expected, actual string) bool {
	if expected == "" || actual == "" || len(expected) != len(actual) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}

func decodeJSON(e *core.RequestEvent, dst any) error {
	// PocketBase wraps request bodies in a rereadable reader so middleware can
	// inspect them more than once. Decode from an immutable snapshot here;
	// otherwise a second Decode used to prove EOF can observe the rewound body
	// as a second JSON value on real network requests.
	body, err := io.ReadAll(e.Request.Body)
	if err != nil {
		return err
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain exactly one JSON value")
		}
		return err
	}
	return nil
}

func writeDomainError(e *core.RequestEvent, err error) error {
	return writeDomainErrorWithData(e, err, nil)
}

func writeDomainErrorWithData(e *core.RequestEvent, err error, extra map[string]any) error {
	var domainErr *domain.Error
	if errors.As(err, &domainErr) {
		payload := map[string]any{
			"error": map[string]any{
				"code":    domainErr.Code,
				"message": domainErr.Message,
				"details": domainErr.Details,
			},
		}
		for key, value := range extra {
			payload[key] = value
		}
		return e.JSON(domainErr.Status, payload)
	}
	return e.InternalServerError("internal server error", err)
}
