package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/alerts"
	"github.com/Phloraxx/payment-api/internal/audit"
	"github.com/Phloraxx/payment-api/internal/backups"
	"github.com/Phloraxx/payment-api/internal/config"
	"github.com/Phloraxx/payment-api/internal/domain"
	"github.com/Phloraxx/payment-api/internal/gmessages"
	"github.com/Phloraxx/payment-api/internal/money"
	"github.com/Phloraxx/payment-api/internal/paymentemail"
	"github.com/Phloraxx/payment-api/internal/payments"
	"github.com/Phloraxx/payment-api/internal/razorpaylive"
	"github.com/Phloraxx/payment-api/internal/razorpaytest"
	"github.com/Phloraxx/payment-api/internal/reconciliation"
	"github.com/Phloraxx/payment-api/internal/refunds"
	"github.com/Phloraxx/payment-api/internal/reviews"
	"github.com/Phloraxx/payment-api/internal/sms"
	appweb "github.com/Phloraxx/payment-api/internal/web"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const (
	maxPaymentRequestBytes      int64 = (1 << 20) + (64 << 10)
	maxSMSRequestBytes          int64 = 128 << 10
	maxEmailRequestBytes        int64 = ((paymentemail.MaxRawBytes + 2) / 3 * 4) + (128 << 10)
	maxGMessagesPairBytes       int64 = 128 << 10
	maxReviewRequestBytes       int64 = 16 << 10
	maxRefundRequestBytes       int64 = (1 << 20) + (64 << 10)
	maxStatementRequestBytes    int64 = reconciliation.MaxFileBytes + (1 << 20)
	maxRazorpayTestRequestBytes int64 = 1 << 20
	maxRazorpayLiveRequestBytes int64 = 1 << 20
	robotsTagValue                    = "noindex, nofollow, noarchive, nosnippet, noimageindex"
)

type API struct {
	Config         config.Config
	Payments       *payments.Service
	SMS            *sms.Service
	Email          *paymentemail.Service
	GMessages      *gmessages.Manager
	Reviews        *reviews.Service
	Reconciliation *reconciliation.Service
	Alerts         *alerts.Service
	Refunds        *refunds.Service
	Backups        *backups.Service
	RazorpayTest   *razorpaytest.Service
	RazorpayLive   *razorpaylive.Service
}

func New(cfg config.Config, paymentService *payments.Service, smsService *sms.Service, manager *gmessages.Manager) *API {
	return &API{Config: cfg, Payments: paymentService, SMS: smsService, GMessages: manager}
}

func (a *API) Register(app core.App) {
	app.OnServe().BindFunc(func(e *core.ServeEvent) error {
		e.Router.BindFunc(func(event *core.RequestEvent) error {
			event.Response.Header().Set("X-Robots-Tag", robotsTagValue)
			return event.Next()
		})
		e.Router.GET("/robots.txt", func(event *core.RequestEvent) error {
			event.Response.Header().Set("Cache-Control", "no-store")
			return event.String(http.StatusOK, "User-agent: *\nContent-Signal: search=no, ai-input=no, ai-train=no, use=immediate\nDisallow:\n")
		})
		e.Router.POST("/api/payments", a.createPayment).Bind(apis.BodyLimit(maxPaymentRequestBytes))
		e.Router.GET("/api/payment-accounts", a.paymentAccounts)
		e.Router.GET("/api/payments/{id}", a.getPayment)
		e.Router.POST("/api/payments/{id}/cancel", a.cancelPayment)
		e.Router.POST("/api/events/sms", a.ingestSMS).Bind(apis.BodyLimit(maxSMSRequestBytes))
		e.Router.POST("/api/events/email", a.ingestEmail).Bind(apis.BodyLimit(maxEmailRequestBytes))
		e.Router.POST("/api/webhook", a.ingestLegacySMS).Bind(apis.BodyLimit(maxSMSRequestBytes))
		e.Router.GET("/api/paygate/health", a.health)
		e.Router.GET("/api/config", a.getConfig)
		e.Router.GET("/api/dashboard", a.dashboard)
		e.Router.GET("/api/capacity", a.capacity)
		e.Router.POST("/api/review-cases/{id}/resolve", a.resolveReview).Bind(apis.BodyLimit(maxReviewRequestBytes))
		e.Router.POST("/api/reconciliation/import", a.importReconciliation).Bind(apis.BodyLimit(maxStatementRequestBytes))
		e.Router.POST("/api/refunds", a.requestRefund).Bind(apis.BodyLimit(maxRefundRequestBytes))
		e.Router.POST("/api/refunds/{id}/status", a.updateRefund).Bind(apis.BodyLimit(maxReviewRequestBytes))
		e.Router.GET("/api/paygate/backups/status", a.backupStatus)
		e.Router.POST("/api/paygate/backups", a.createBackup)
		e.Router.POST("/api/paygate/backups/verify", a.verifyBackup)
		e.Router.POST("/api/paygate/backups/restore-drill", a.restoreDrill)
		e.Router.GET("/api/razorpay/test/config", a.razorpayTestConfig)
		e.Router.POST("/api/razorpay/test/orders", a.razorpayTestCreateOrder).Bind(apis.BodyLimit(maxRazorpayTestRequestBytes))
		e.Router.GET("/api/razorpay/test/orders/{id}", a.razorpayTestGetOrder)
		e.Router.POST("/api/razorpay/test/orders/{id}/verify", a.razorpayTestVerify).Bind(apis.BodyLimit(maxRazorpayTestRequestBytes))
		e.Router.POST("/api/razorpay/test/orders/{id}/refresh", a.razorpayTestRefresh)
		e.Router.POST("/api/razorpay/test/webhook", a.razorpayTestWebhook).Bind(apis.BodyLimit(maxRazorpayTestRequestBytes))
		e.Router.GET("/api/razorpay/live/config", a.razorpayLiveConfig)
		e.Router.POST("/api/razorpay/live/orders", a.razorpayLiveCreateOrder).Bind(apis.BodyLimit(maxRazorpayLiveRequestBytes))
		e.Router.GET("/api/razorpay/live/orders/{id}", a.razorpayLiveGetOrder)
		e.Router.POST("/api/razorpay/live/orders/{id}/verify", a.razorpayLiveVerify).Bind(apis.BodyLimit(maxRazorpayLiveRequestBytes))
		e.Router.POST("/api/razorpay/live/orders/{id}/refresh", a.razorpayLiveRefresh)
		e.Router.POST("/api/razorpay/live/webhook", a.razorpayLiveWebhook).Bind(apis.BodyLimit(maxRazorpayLiveRequestBytes))
		e.Router.GET("/api/connector/gmessages/status", a.gmessagesStatus)
		e.Router.POST("/api/connector/gmessages/pair/google", a.gmessagesGooglePair).Bind(apis.BodyLimit(maxGMessagesPairBytes))
		e.Router.POST("/api/connector/gmessages/reauth/google", a.gmessagesGoogleReauth).Bind(apis.BodyLimit(maxGMessagesPairBytes))
		e.Router.POST("/api/connector/gmessages/pair/qr", a.gmessagesPair)
		e.Router.POST("/api/connector/gmessages/pair/qr/refresh", a.gmessagesPairRefresh)
		// Backward-compatible QR aliases from the first PayGate rebuild.
		e.Router.POST("/api/connector/gmessages/pair", a.gmessagesPair)
		e.Router.POST("/api/connector/gmessages/pair/refresh", a.gmessagesPairRefresh)
		e.Router.POST("/api/connector/gmessages/reconnect", a.gmessagesReconnect)
		e.Router.DELETE("/api/connector/gmessages/pair", a.gmessagesUnpair)

		// The operator app uses hash routing, so only the root document and its
		// fingerprinted assets are valid browser paths. Everything else is a real 404.
		static := apis.Static(appweb.Assets(), true)
		e.Router.GET("/{path...}", func(event *core.RequestEvent) error {
			path := strings.TrimPrefix(event.Request.URL.Path, "/")
			if path != "" && path != "index.html" && !strings.HasPrefix(path, "assets/") {
				return event.NotFoundError("route not found", nil)
			}
			a.setOperatorSecurityHeaders(event)
			return static(event)
		})
		return e.Next()
	})
}

type emailBody struct {
	SourceID       string `json:"sourceId"`
	EnvelopeFrom   string `json:"envelopeFrom"`
	EnvelopeTo     string `json:"envelopeTo"`
	ReceivedAt     string `json:"receivedAt"`
	RawEmailBase64 string `json:"rawEmailBase64"`
}

func (a *API) ingestEmail(e *core.RequestEvent) error {
	if !a.Config.EmailEvidenceEnabled || a.Email == nil {
		return e.NotFoundError("route not found", nil)
	}
	body, err := io.ReadAll(e.Request.Body)
	if err != nil {
		return e.BadRequestError("could not read request body", err)
	}
	timestamp := strings.TrimSpace(e.Request.Header.Get("X-PayGate-Timestamp"))
	signature := strings.TrimSpace(e.Request.Header.Get("X-PayGate-Signature"))
	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return e.UnauthorizedError("invalid email webhook timestamp", nil)
	}
	signedAt := time.Unix(seconds, 0)
	if delta := time.Since(signedAt); delta > a.Config.EmailSignatureTolerance || delta < -a.Config.EmailSignatureTolerance {
		return e.UnauthorizedError("email webhook timestamp is outside the allowed window", nil)
	}
	if !validEmailSignature(a.Config.EmailWebhookSecret, timestamp, body, signature) {
		return e.UnauthorizedError("invalid email webhook signature", nil)
	}

	var request emailBody
	if err := decodeJSONBytes(body, &request); err != nil {
		return e.BadRequestError("invalid JSON body", err)
	}
	raw, err := base64.StdEncoding.Strict().DecodeString(strings.TrimSpace(request.RawEmailBase64))
	if err != nil {
		return e.BadRequestError("rawEmailBase64 must be valid standard base64", err)
	}
	message, err := paymentemail.ParseRaw(raw)
	if err != nil {
		return e.BadRequestError("raw email could not be parsed", err)
	}
	sourceID := strings.TrimSpace(request.SourceID)
	if sourceID == "" {
		sourceID = message.MessageID
	}
	if sourceID == "" {
		return e.BadRequestError("sourceId or Message-ID is required for deduplication", nil)
	}
	if message.MessageID != "" && sourceID != message.MessageID {
		return e.BadRequestError("sourceId must match the email Message-ID", nil)
	}
	receivedAt := signedAt
	if strings.TrimSpace(request.ReceivedAt) != "" {
		receivedAt, err = time.Parse(time.RFC3339, request.ReceivedAt)
		if err != nil {
			return e.BadRequestError("receivedAt must be RFC3339", err)
		}
	}
	result, err := a.Email.Ingest(paymentemail.Input{
		Source: "cloudflare_email", SourceEventID: sourceID,
		EnvelopeSender: request.EnvelopeFrom, Recipient: request.EnvelopeTo,
		Message: message, ReceivedAt: receivedAt,
		RawPayload: map[string]any{
			"sourceId": sourceID, "messageId": message.MessageID,
			"envelopeFrom": request.EnvelopeFrom, "envelopeTo": request.EnvelopeTo,
			"receivedAt": receivedAt.UTC().Format(time.RFC3339),
		},
	})
	if err != nil {
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

func validEmailSignature(secret, timestamp string, body []byte, signature string) bool {
	if secret == "" || timestamp == "" {
		return false
	}
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	signature = strings.TrimSpace(strings.TrimPrefix(signature, "sha256="))
	provided, err := hex.DecodeString(signature)
	if err != nil || len(provided) != sha256.Size {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(body)
	return hmac.Equal(mac.Sum(nil), provided)
}

type createPaymentBody struct {
	Amount         json.RawMessage `json:"amount"`
	PaymentAccount string          `json:"paymentAccount"`
	ExternalID     string          `json:"externalId"`
	Metadata       json.RawMessage `json:"metadata"`
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
		PaymentAccount: strings.TrimSpace(body.PaymentAccount),
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

func (a *API) paymentAccounts(e *core.RequestEvent) error {
	if !a.authorizedWrite(e) {
		return e.UnauthorizedError("API key or dashboard authentication is required", nil)
	}
	defaultAccount := strings.ToLower(strings.TrimSpace(a.Config.DefaultPaymentAccount))
	if defaultAccount == "" {
		defaultAccount = "kotak"
	}
	return e.JSON(http.StatusOK, map[string]any{
		"default":  defaultAccount,
		"accounts": a.Config.PaymentAccounts(),
	})
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
	// Read real application tables from both SQLite databases instead of using
	// SELECT 1. A constant expression can succeed even when the database/WAL
	// backing the PayGate tables is returning I/O errors.
	var primaryRows int
	if err := e.App.DB().NewQuery("SELECT COUNT(*) FROM payments").Row(&primaryRows); err != nil {
		return e.JSON(http.StatusServiceUnavailable, map[string]any{
			"status": "unhealthy", "ready": false, "db": "error", "connector": a.publicConnectorStatus(),
		})
	}

	var auxiliaryRows int
	if err := e.App.AuxDB().NewQuery("SELECT COUNT(*) FROM _logs").Row(&auxiliaryRows); err != nil {
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
	defaultAccount := strings.ToLower(strings.TrimSpace(a.Config.DefaultPaymentAccount))
	if defaultAccount == "" {
		defaultAccount = "kotak"
	}
	return e.JSON(http.StatusOK, map[string]any{
		"upiId":                             a.Config.UPIID,
		"upiPayeeName":                      a.Config.UPIPayeeName,
		"defaultPaymentAccount":             defaultAccount,
		"paymentAccounts":                   a.Config.PaymentAccounts(),
		"paymentTtlSeconds":                 int64(a.Config.PaymentTTL / time.Second),
		"quarantineSeconds":                 int64(a.Config.AmountQuarantine / time.Second),
		"webhookConfigured":                 a.Config.OutgoingWebhookURL != "",
		"rateLimitsEnabled":                 a.Config.RateLimitsEnabled,
		"legacySMSWebhookEnabled":           a.Config.LegacySMSWebhookEnabled,
		"emailEvidenceEnabled":              a.Config.EmailEvidenceEnabled,
		"emailAllowedSender":                a.Config.EmailAllowedSender,
		"retentionEnabled":                  a.Config.RetentionEnabled,
		"smsRawRetentionSeconds":            int64(a.Config.SMSRawRetention / time.Second),
		"emailRawRetentionSeconds":          int64(a.Config.EmailRawRetention / time.Second),
		"reconciliationRawRetentionSeconds": int64(a.Config.ReconciliationRawRetention / time.Second),
		"auditRetentionSeconds":             int64(a.Config.AuditRetention / time.Second),
		"backupEnabled":                     a.Config.BackupCron != "",
		"backupCron":                        a.Config.BackupCron,
		"backupMaxKeep":                     a.Config.BackupMaxKeep,
		"backupOffsite":                     a.Config.BackupS3Enabled,
		"operatorAlertWebhookConfigured":    a.Config.OperatorAlertWebhookURL != "",
		"statementTimezone":                 a.Config.StatementTimezone,
		"razorpayTestEnabled":               a.Config.RazorpayTestEnabled,
		"connector":                         a.connectorStatus(),
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
	payload := map[string]any{"stats": stats, "connector": a.connectorStatus()}
	if a.Payments != nil {
		capacity, err := a.Payments.Capacity()
		if err != nil {
			return e.InternalServerError("failed to calculate capacity", err)
		}
		payload["capacity"] = capacity
	}
	if a.Reviews != nil {
		count, err := a.Reviews.OpenCount()
		if err != nil {
			return e.InternalServerError("failed to count reviews", err)
		}
		payload["openReviewCount"] = count
	}
	if a.Alerts != nil {
		count, err := a.Alerts.OpenCount()
		if err != nil {
			return e.InternalServerError("failed to count alerts", err)
		}
		payload["openAlertCount"] = count
	}
	if a.Backups != nil {
		status, err := a.Backups.GetStatus(e.Request.Context(), false)
		if err != nil {
			payload["backup"] = map[string]any{"enabled": a.Config.BackupCron != "", "error": err.Error()}
		} else {
			payload["backup"] = status
		}
	}
	return e.JSON(http.StatusOK, payload)
}

func (a *API) capacity(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	capacity, err := a.Payments.Capacity()
	if err != nil {
		return e.InternalServerError("failed to calculate payment capacity", err)
	}
	return e.JSON(http.StatusOK, capacity)
}

type reviewResolutionBody struct {
	Action        string `json:"action"`
	PaymentID     string `json:"paymentId"`
	BankReference string `json:"bankReference"`
	Note          string `json:"note"`
}

func (a *API) resolveReview(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	if a.Reviews == nil {
		return e.NotFoundError("review service is unavailable", nil)
	}
	var body reviewResolutionBody
	if err := decodeJSON(e, &body); err != nil {
		return e.BadRequestError("invalid JSON body", err)
	}
	result, err := a.Reviews.Resolve(reviews.ResolveInput{
		CaseID: e.Request.PathValue("id"), Action: body.Action,
		PaymentID: body.PaymentID, BankReference: body.BankReference,
		Note: body.Note, Actor: a.actor(e),
	})
	if err != nil {
		return writeDomainError(e, err)
	}
	return e.JSON(http.StatusOK, result)
}

func (a *API) importReconciliation(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	if a.Reconciliation == nil {
		return e.NotFoundError("reconciliation service is unavailable", nil)
	}
	if err := e.Request.ParseMultipartForm(reconciliation.MaxFileBytes); err != nil {
		return e.BadRequestError("invalid multipart statement upload", err)
	}
	file, header, err := e.Request.FormFile("statement")
	if err != nil {
		return e.BadRequestError("multipart field 'statement' is required", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, reconciliation.MaxFileBytes+1))
	if err != nil {
		return e.BadRequestError("failed to read statement", err)
	}
	if len(data) > reconciliation.MaxFileBytes {
		return e.JSON(http.StatusRequestEntityTooLarge, map[string]any{"error": map[string]any{"code": "STATEMENT_TOO_LARGE", "message": "statement exceeds 10 MiB"}})
	}
	result, err := a.Reconciliation.Import(reconciliation.ImportInput{Filename: header.Filename, Data: data, Actor: a.actor(e)})
	if err != nil {
		return writeDomainError(e, err)
	}
	return e.JSON(http.StatusCreated, result)
}

type refundRequestBody struct {
	PaymentID   string          `json:"paymentId"`
	AmountPaise int64           `json:"amountPaise"`
	Reason      string          `json:"reason"`
	ExternalID  string          `json:"externalId"`
	Metadata    json.RawMessage `json:"metadata"`
}

func (a *API) requestRefund(e *core.RequestEvent) error {
	if !a.authorizedWrite(e) {
		return e.UnauthorizedError("API key or dashboard authentication is required", nil)
	}
	if a.Refunds == nil {
		return e.NotFoundError("refund service is unavailable", nil)
	}
	var body refundRequestBody
	if err := decodeJSON(e, &body); err != nil {
		return e.BadRequestError("invalid JSON body", err)
	}
	var metadata any
	if len(body.Metadata) > 0 && string(body.Metadata) != "null" {
		if err := json.Unmarshal(body.Metadata, &metadata); err != nil {
			return e.BadRequestError("metadata must be valid JSON", err)
		}
	}
	record, replayed, err := a.Refunds.Request(refunds.RequestInput{
		PaymentID: body.PaymentID, AmountPaise: body.AmountPaise, Reason: body.Reason,
		ExternalID: body.ExternalID, IdempotencyKey: strings.TrimSpace(e.Request.Header.Get("Idempotency-Key")),
		Metadata: metadata, Actor: a.actor(e),
	})
	if err != nil {
		return writeDomainError(e, err)
	}
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
		e.Response.Header().Set("X-Idempotent-Replayed", "true")
	}
	return e.JSON(status, refundResponse(record))
}

type refundUpdateBody struct {
	Status    string `json:"status"`
	Reference string `json:"reference"`
	Note      string `json:"note"`
}

func (a *API) updateRefund(e *core.RequestEvent) error {
	if !a.authorizedWrite(e) {
		return e.UnauthorizedError("API key or dashboard authentication is required", nil)
	}
	if a.Refunds == nil {
		return e.NotFoundError("refund service is unavailable", nil)
	}
	var body refundUpdateBody
	if err := decodeJSON(e, &body); err != nil {
		return e.BadRequestError("invalid JSON body", err)
	}
	record, err := a.Refunds.Update(refunds.UpdateInput{
		RefundID: e.Request.PathValue("id"), Status: body.Status,
		Reference: body.Reference, Note: body.Note, Actor: a.actor(e),
	})
	if err != nil {
		return writeDomainError(e, err)
	}
	return e.JSON(http.StatusOK, refundResponse(record))
}

func (a *API) backupStatus(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	if a.Backups == nil {
		return e.NotFoundError("backup service is unavailable", nil)
	}
	status, err := a.Backups.GetStatus(e.Request.Context(), false)
	if err != nil {
		return e.InternalServerError("failed to inspect backups", err)
	}
	return e.JSON(http.StatusOK, status)
}

func (a *API) createBackup(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	if a.Backups == nil {
		return e.NotFoundError("backup service is unavailable", nil)
	}
	name, err := a.Backups.Create(e.Request.Context())
	if err != nil {
		return e.InternalServerError("failed to create backup", err)
	}
	return e.JSON(http.StatusCreated, map[string]any{"name": name})
}

func (a *API) verifyBackup(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	if a.Backups == nil {
		return e.NotFoundError("backup service is unavailable", nil)
	}
	status, err := a.Backups.GetStatus(e.Request.Context(), true)
	if err != nil {
		return e.InternalServerError("failed to verify backup", err)
	}
	code := http.StatusOK
	if !status.LatestVerified {
		code = http.StatusUnprocessableEntity
	}
	return e.JSON(code, status)
}

func (a *API) restoreDrill(e *core.RequestEvent) error {
	if !a.dashboardAuth(e) {
		return e.UnauthorizedError("dashboard authentication is required", nil)
	}
	if a.Backups == nil {
		return e.NotFoundError("backup service is unavailable", nil)
	}
	result, err := a.Backups.RestoreDrill(e.Request.Context())
	if err != nil {
		return e.InternalServerError("backup restore drill failed", err)
	}
	return e.JSON(http.StatusOK, result)
}

func refundResponse(record *core.Record) map[string]any {
	if record == nil {
		return nil
	}
	return map[string]any{
		"id": record.Id, "paymentId": record.GetString("payment"),
		"amountPaise": record.GetInt("amount"), "status": record.GetString("status"),
		"reason": record.GetString("reason"), "reference": record.GetString("reference"),
		"externalId":  record.GetString("external_id"),
		"requestedAt": record.GetDateTime("requested_at").String(),
		"completedAt": record.GetDateTime("completed_at").String(),
	}
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

func (a *API) gmessagesGoogleReauth(e *core.RequestEvent) error {
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
	if err := a.GMessages.ReauthenticateGoogle(strings.TrimSpace(body.CookieData)); err != nil {
		return e.BadRequestError(err.Error(), nil)
	}
	return e.JSON(http.StatusOK, a.GMessages.Status())
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

func (a *API) actor(e *core.RequestEvent) audit.Actor {
	if e.Auth != nil {
		return audit.Actor{ID: e.Auth.Id, Email: e.Auth.Email()}
	}
	if bearerMatches(a.Config.APIKey, e.Request.Header.Get("Authorization")) {
		return audit.Actor{Email: "api-key"}
	}
	return audit.Actor{}
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

	return decodeJSONBytes(body, dst)
}

func decodeJSONBytes(body []byte, dst any) error {
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
