package config

import (
	"strings"
	"testing"
	"time"
)

const (
	testAPISecret = "api-secret-that-is-long-enough"
	testSMSSecret = "sms-secret-that-is-long-enough"
)

func TestValidateServeRequiresCoreSecrets(t *testing.T) {
	cfg := Config{PaymentTTL: 5 * time.Minute, AmountQuarantine: 24 * time.Hour, StatementTimezone: "Asia/Kolkata"}
	err := cfg.ValidateServe()
	if err == nil {
		t.Fatal("ValidateServe accepted missing core configuration")
	}
	for _, field := range []string{"UPI_ID", "PAYGATE_API_KEY", "SMS_WEBHOOK_SECRET"} {
		if !strings.Contains(err.Error(), field) {
			t.Errorf("error %q does not mention %s", err, field)
		}
	}
}

func TestValidateServeRejectsWeakPrimarySecrets(t *testing.T) {
	base := Config{UPIID: "operator@bank", APIKey: "short", SMSWebhookSecret: testSMSSecret, PaymentTTL: time.Minute, AmountQuarantine: time.Hour, StatementTimezone: "Asia/Kolkata"}
	if err := base.ValidateServe(); err == nil || !strings.Contains(err.Error(), "PAYGATE_API_KEY") {
		t.Fatalf("weak API key error = %v", err)
	}
	base.APIKey = testAPISecret
	base.SMSWebhookSecret = "short"
	if err := base.ValidateServe(); err == nil || !strings.Contains(err.Error(), "SMS_WEBHOOK_SECRET") {
		t.Fatalf("weak SMS secret error = %v", err)
	}
}

func TestValidateServeWebhookSecretIsConditional(t *testing.T) {
	base := Config{UPIID: "operator@bank", APIKey: testAPISecret, SMSWebhookSecret: testSMSSecret, PaymentTTL: time.Minute, AmountQuarantine: time.Hour, StatementTimezone: "Asia/Kolkata"}
	if err := base.ValidateServe(); err != nil {
		t.Fatalf("base config invalid: %v", err)
	}
	base.OutgoingWebhookURL = "https://example.test/webhook"
	if err := base.ValidateServe(); err == nil {
		t.Fatal("webhook URL without signing secret was accepted")
	}
	base.OutgoingWebhookSecret = "signing-secret-that-is-long-enough"
	if err := base.ValidateServe(); err != nil {
		t.Fatalf("complete webhook config invalid: %v", err)
	}
	base.OutgoingWebhookURL = "not-a-url"
	if err := base.ValidateServe(); err == nil || !strings.Contains(err.Error(), "OUTGOING_WEBHOOK_URL") {
		t.Fatalf("malformed webhook URL error = %v", err)
	}
}

func TestLoadSupportsLegacyPrototypeVariablesWithoutPromotingLegacySecret(t *testing.T) {
	t.Setenv("UPI_ID", "legacy@bank")
	t.Setenv("UPI_NAME", "Legacy Name")
	t.Setenv("PAYGATE_API_KEY", testAPISecret)
	t.Setenv("WEBHOOK_SECRET", "legacy-webhook")
	t.Setenv("TICKET_TTL_MINUTES", "7")
	t.Setenv("AMOUNT_QUARANTINE_HOURS", "12")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.UPIPayeeName != "Legacy Name" || cfg.LegacySMSWebhookSecret != "legacy-webhook" || cfg.SMSWebhookSecret != "" {
		t.Fatalf("legacy mapping failed: %+v", cfg)
	}
	if cfg.PaymentTTL != 7*time.Minute || cfg.AmountQuarantine != 12*time.Hour {
		t.Fatalf("legacy durations failed: ttl=%s quarantine=%s", cfg.PaymentTTL, cfg.AmountQuarantine)
	}
}

func TestValidateServeLegacyWebhookIsExplicit(t *testing.T) {
	base := Config{UPIID: "operator@bank", APIKey: testAPISecret, SMSWebhookSecret: testSMSSecret, PaymentTTL: time.Minute, AmountQuarantine: time.Hour, StatementTimezone: "Asia/Kolkata"}
	base.LegacySMSWebhookEnabled = true
	if err := base.ValidateServe(); err == nil || !strings.Contains(err.Error(), "WEBHOOK_SECRET") {
		t.Fatalf("legacy route without legacy secret error = %v", err)
	}
	base.LegacySMSWebhookSecret = "short"
	if err := base.ValidateServe(); err == nil || !strings.Contains(err.Error(), "at least") {
		t.Fatalf("weak legacy route secret accepted: %v", err)
	}
	base.LegacySMSWebhookSecret = "legacy-secret-that-is-long-enough"
	if err := base.ValidateServe(); err != nil {
		t.Fatalf("explicit legacy route config rejected: %v", err)
	}
}

func TestLoadRejectsMalformedEnvironmentValues(t *testing.T) {
	t.Setenv("PAYMENT_TTL", "five-minutes")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "PAYMENT_TTL") {
		t.Fatalf("invalid duration error = %v", err)
	}

	t.Setenv("PAYMENT_TTL", "5m")
	t.Setenv("PAYGATE_RATE_LIMITS_ENABLED", "sometimes")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "PAYGATE_RATE_LIMITS_ENABLED") {
		t.Fatalf("invalid bool error = %v", err)
	}
}

func TestLoadDefaultsEmailEvidenceOff(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.EmailEvidenceEnabled {
		t.Fatal("email evidence must be opt-in")
	}
	if cfg.EmailAllowedSender != "noreply@slice.bank.in" || cfg.EmailAuthServID != "mx.cloudflare.net" {
		t.Fatalf("email defaults = sender %q authserv %q", cfg.EmailAllowedSender, cfg.EmailAuthServID)
	}
	if cfg.EmailSignatureTolerance != 5*time.Minute || cfg.EmailRawRetention != 90*24*time.Hour {
		t.Fatalf("email durations = tolerance %s retention %s", cfg.EmailSignatureTolerance, cfg.EmailRawRetention)
	}
}

func TestValidateServeEmailEvidenceRequiresStrongExactConfiguration(t *testing.T) {
	base := Config{
		TestMode: true, PaymentTTL: time.Minute, AmountQuarantine: time.Hour, StatementTimezone: "Asia/Kolkata",
		SliceUPIID:           "operator@slice",
		EmailEvidenceEnabled: true, EmailAllowedSender: "noreply@slice.bank.in", EmailAuthServID: "mx.cloudflare.net",
		EmailSignatureTolerance: 5 * time.Minute,
	}
	if err := base.ValidateServe(); err == nil || !strings.Contains(err.Error(), "PAYMENT_EMAIL_WEBHOOK_SECRET") {
		t.Fatalf("missing email secret error = %v", err)
	}
	base.EmailWebhookSecret = "email-webhook-secret-long-enough"
	base.EmailAllowedSender = "Slice <noreply@slice.bank.in>"
	if err := base.ValidateServe(); err == nil || !strings.Contains(err.Error(), "PAYMENT_EMAIL_ALLOWED_SENDER") {
		t.Fatalf("display-name sender accepted: %v", err)
	}
	base.EmailAllowedSender = "noreply@slice.bank.in"
	if err := base.ValidateServe(); err != nil {
		t.Fatalf("valid email configuration rejected: %v", err)
	}
}

func TestValidateServeRejectsInvalidStatementTimezone(t *testing.T) {
	cfg := Config{
		UPIID: "operator@bank", APIKey: testAPISecret, SMSWebhookSecret: testSMSSecret,
		PaymentTTL: time.Minute, AmountQuarantine: time.Hour, StatementTimezone: "Not/A-Timezone",
	}
	if err := cfg.ValidateServe(); err == nil || !strings.Contains(err.Error(), "STATEMENT_TIMEZONE") {
		t.Fatalf("invalid timezone error=%v", err)
	}
}

func TestTestModeStillValidatesStructuralConfiguration(t *testing.T) {
	cfg := Config{TestMode: true, PaymentTTL: -time.Second, AmountQuarantine: time.Hour, StatementTimezone: "Asia/Kolkata"}
	if err := cfg.ValidateServe(); err == nil || !strings.Contains(err.Error(), "PAYMENT_TTL") {
		t.Fatalf("test mode accepted invalid TTL: %v", err)
	}
	cfg.PaymentTTL = time.Minute
	cfg.OperatorAlertWebhookURL = "not-a-url"
	cfg.OperatorAlertWebhookSecret = "operator-alert-secret-long-enough"
	if err := cfg.ValidateServe(); err == nil || !strings.Contains(err.Error(), "OPERATOR_ALERT_WEBHOOK_URL") {
		t.Fatalf("test mode accepted invalid webhook URL: %v", err)
	}
}

func TestValidateServeRazorpayTestRequiresTestModeCredentials(t *testing.T) {
	base := Config{
		TestMode: true, PaymentTTL: time.Minute, AmountQuarantine: time.Hour,
		StatementTimezone: "Asia/Kolkata", BackupMaxKeep: 1,
		RazorpayTestEnabled: true, RazorpayTestDisplayName: "PayGate Test",
	}
	cases := []struct {
		name string
		edit func(*Config)
	}{
		{"live key rejected", func(c *Config) {
			c.RazorpayTestKeyID = "rzp_live_example"
			c.RazorpayTestKeySecret = "1234567890123456"
			c.RazorpayTestWebhookSecret = "123456789012345678901234"
		}},
		{"short key secret", func(c *Config) {
			c.RazorpayTestKeyID = "rzp_test_example"
			c.RazorpayTestKeySecret = "short"
			c.RazorpayTestWebhookSecret = "123456789012345678901234"
		}},
		{"short webhook secret", func(c *Config) {
			c.RazorpayTestKeyID = "rzp_test_example"
			c.RazorpayTestKeySecret = "1234567890123456"
			c.RazorpayTestWebhookSecret = "short"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			tc.edit(&cfg)
			if err := cfg.ValidateServe(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
	valid := base
	valid.RazorpayTestKeyID = "rzp_test_example"
	valid.RazorpayTestKeySecret = "1234567890123456"
	valid.RazorpayTestWebhookSecret = "123456789012345678901234"
	if err := valid.ValidateServe(); err != nil {
		t.Fatalf("valid Razorpay test config: %v", err)
	}
}

func TestPaytmAccountRequiresStaticQRAndStrongNotificationSecret(t *testing.T) {
	base := Config{UPIID: "operator@bank", APIKey: testAPISecret, SMSWebhookSecret: testSMSSecret, PaymentTTL: time.Minute, AmountQuarantine: time.Hour, StatementTimezone: "Asia/Kolkata"}
	base.PaytmNotificationWebhookSecret = "short"
	if err := base.ValidateServe(); err == nil || !strings.Contains(err.Error(), "PAYTM_NOTIFICATION_WEBHOOK_SECRET") {
		t.Fatalf("weak Paytm secret accepted: %v", err)
	}
	base.PaytmNotificationWebhookSecret = "paytm-notification-secret-long-enough"
	if _, ok := base.PaymentAccount("paytm"); ok {
		t.Fatal("Paytm account exposed without a static merchant QR payload")
	}
	base.PaytmQRPayload = "paytm-issued-static-merchant-qr-payload"
	account, ok := base.PaymentAccount("paytm")
	if !ok || account.Verification != "notification" || account.Flow != "merchant_qr" || account.QRPayload != base.PaytmQRPayload {
		t.Fatalf("Paytm account = %+v ok=%v", account, ok)
	}
	if err := base.ValidateServe(); err != nil {
		t.Fatalf("valid Paytm configuration rejected: %v", err)
	}
}
