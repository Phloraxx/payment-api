package config

import (
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const minPrimarySecretLength = 24

type Config struct {
	DataDir                        string
	DefaultPaymentAccount          string
	KotakUPIID                     string
	KotakUPIPayeeName              string
	SliceUPIID                     string
	SliceUPIPayeeName              string
	PaytmQRPayload                 string
	PaytmPayeeName                 string
	PaytmNotificationWebhookSecret string
	UPIID                          string
	UPIPayeeName                   string
	APIKey                         string
	SMSWebhookSecret               string
	EmailEvidenceEnabled           bool
	EmailWebhookSecret             string
	EmailAllowedSender             string
	EmailAuthServID                string
	EmailSignatureTolerance        time.Duration
	LegacySMSWebhookSecret         string
	LegacySMSWebhookEnabled        bool
	PaymentTTL                     time.Duration
	AmountQuarantine               time.Duration
	OutgoingWebhookURL             string
	OutgoingWebhookSecret          string
	GMessagesEnabled               bool
	GMessagesSessionPath           string
	TestMode                       bool
	RateLimitsEnabled              bool
	RetentionEnabled               bool
	SMSRawRetention                time.Duration
	EmailRawRetention              time.Duration
	ReconciliationRawRetention     time.Duration
	AuditRetention                 time.Duration
	BackupCron                     string
	BackupMaxKeep                  int
	BackupS3Enabled                bool
	BackupS3Bucket                 string
	BackupS3Region                 string
	BackupS3Endpoint               string
	BackupS3AccessKey              string
	BackupS3Secret                 string
	BackupS3ForcePathStyle         bool
	OperatorAlertWebhookURL        string
	OperatorAlertWebhookSecret     string
	StatementTimezone              string
	RazorpayTestEnabled            bool
	RazorpayTestKeyID              string
	RazorpayTestKeySecret          string
	RazorpayTestWebhookSecret      string
	RazorpayTestDisplayName        string
	RazorpayLiveEnabled            bool
	RazorpayLiveKeyID              string
	RazorpayLiveKeySecret          string
	RazorpayLiveWebhookSecret      string
	RazorpayLiveDisplayName        string
}

type PaymentAccount struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	UPIID        string `json:"-"`
	PayeeName    string `json:"-"`
	Verification string `json:"verification"`
	Flow         string `json:"flow"`
	QRPayload    string `json:"-"`
}

func Load() (Config, error) {
	legacyTTL, err := legacyMinutesEnv("TICKET_TTL_MINUTES", 5*time.Minute)
	if err != nil {
		return Config{}, err
	}
	paymentTTL, err := durationEnv("PAYMENT_TTL", legacyTTL)
	if err != nil {
		return Config{}, err
	}
	legacyQuarantine, err := legacyHoursEnv("AMOUNT_QUARANTINE_HOURS", 24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	quarantine, err := durationEnv("PAYMENT_QUARANTINE", legacyQuarantine)
	if err != nil {
		return Config{}, err
	}
	gmessagesEnabled, err := boolEnv("GMESSAGES_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	testMode, err := boolEnv("PAYGATE_TEST_MODE", false)
	if err != nil {
		return Config{}, err
	}
	rateLimitsEnabled, err := boolEnv("PAYGATE_RATE_LIMITS_ENABLED", true)
	if err != nil {
		return Config{}, err
	}
	legacyEnabled, err := boolEnv("LEGACY_SMS_WEBHOOK_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	emailEvidenceEnabled, err := boolEnv("PAYMENT_EMAIL_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	retentionEnabled, err := boolEnv("PAYGATE_RETENTION_ENABLED", true)
	if err != nil {
		return Config{}, err
	}
	smsRawRetention, err := durationEnv("SMS_RAW_RETENTION", 90*24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	emailRawRetention, err := durationEnv("EMAIL_RAW_RETENTION", 90*24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	emailSignatureTolerance, err := durationEnv("PAYMENT_EMAIL_SIGNATURE_TOLERANCE", 5*time.Minute)
	if err != nil {
		return Config{}, err
	}
	reconciliationRetention, err := durationEnv("RECONCILIATION_RAW_RETENTION", 365*24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	auditRetention, err := durationEnv("AUDIT_RETENTION", 2*365*24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	backupMaxKeep, err := intEnv("PAYGATE_BACKUP_MAX_KEEP", 14)
	if err != nil {
		return Config{}, err
	}
	backupS3Enabled, err := boolEnv("PAYGATE_BACKUP_S3_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	backupS3ForcePathStyle, err := boolEnv("PAYGATE_BACKUP_S3_FORCE_PATH_STYLE", false)
	if err != nil {
		return Config{}, err
	}
	razorpayTestEnabled, err := boolEnv("RAZORPAY_TEST_ENABLED", false)
	if err != nil {
		return Config{}, err
	}
	razorpayLiveEnabled, err := boolEnv("RAZORPAY_LIVE_ENABLED", false)
	if err != nil {
		return Config{}, err
	}

	dataDir := strings.TrimSpace(env("PB_DATA_DIR", "./pb_data"))
	kotakUPIID := strings.TrimSpace(firstNonEmpty(os.Getenv("KOTAK_UPI_ID"), os.Getenv("UPI_ID")))
	kotakPayeeName := strings.TrimSpace(firstNonEmpty(os.Getenv("KOTAK_UPI_PAYEE_NAME"), os.Getenv("UPI_PAYEE_NAME"), os.Getenv("UPI_NAME"), "PayGate"))
	cfg := Config{
		DataDir:                        dataDir,
		DefaultPaymentAccount:          strings.ToLower(strings.TrimSpace(env("PAYMENT_DEFAULT_ACCOUNT", "kotak"))),
		KotakUPIID:                     kotakUPIID,
		KotakUPIPayeeName:              kotakPayeeName,
		SliceUPIID:                     strings.TrimSpace(os.Getenv("SLICE_UPI_ID")),
		SliceUPIPayeeName:              strings.TrimSpace(firstNonEmpty(os.Getenv("SLICE_UPI_PAYEE_NAME"), os.Getenv("UPI_PAYEE_NAME"), os.Getenv("UPI_NAME"), "PayGate")),
		PaytmQRPayload:                 strings.TrimSpace(os.Getenv("PAYTM_QR_PAYLOAD")),
		PaytmPayeeName:                 strings.TrimSpace(firstNonEmpty(os.Getenv("PAYTM_PAYEE_NAME"), "Paytm for Business")),
		PaytmNotificationWebhookSecret: strings.TrimSpace(os.Getenv("PAYTM_NOTIFICATION_WEBHOOK_SECRET")),
		UPIID:                          kotakUPIID,
		UPIPayeeName:                   kotakPayeeName,
		APIKey:                         strings.TrimSpace(os.Getenv("PAYGATE_API_KEY")),
		SMSWebhookSecret:               strings.TrimSpace(os.Getenv("SMS_WEBHOOK_SECRET")),
		EmailEvidenceEnabled:           emailEvidenceEnabled,
		EmailWebhookSecret:             strings.TrimSpace(os.Getenv("PAYMENT_EMAIL_WEBHOOK_SECRET")),
		EmailAllowedSender:             strings.ToLower(strings.TrimSpace(env("PAYMENT_EMAIL_ALLOWED_SENDER", "noreply@slice.bank.in"))),
		EmailAuthServID:                strings.ToLower(strings.TrimSpace(env("PAYMENT_EMAIL_AUTH_SERV_ID", "mx.cloudflare.net"))),
		EmailSignatureTolerance:        emailSignatureTolerance,
		LegacySMSWebhookSecret:         strings.TrimSpace(os.Getenv("WEBHOOK_SECRET")),
		LegacySMSWebhookEnabled:        legacyEnabled,
		PaymentTTL:                     paymentTTL,
		AmountQuarantine:               quarantine,
		OutgoingWebhookURL:             strings.TrimSpace(firstNonEmpty(os.Getenv("OUTGOING_WEBHOOK_URL"), os.Getenv("PAYMENT_WEBHOOK_URL"))),
		OutgoingWebhookSecret:          strings.TrimSpace(firstNonEmpty(os.Getenv("OUTGOING_WEBHOOK_SECRET"), os.Getenv("PAYMENT_WEBHOOK_SECRET"))),
		GMessagesEnabled:               gmessagesEnabled,
		TestMode:                       testMode,
		RateLimitsEnabled:              rateLimitsEnabled,
		RetentionEnabled:               retentionEnabled,
		SMSRawRetention:                smsRawRetention,
		EmailRawRetention:              emailRawRetention,
		ReconciliationRawRetention:     reconciliationRetention,
		AuditRetention:                 auditRetention,
		BackupCron:                     strings.TrimSpace(env("PAYGATE_BACKUP_CRON", "0 3 * * *")),
		BackupMaxKeep:                  backupMaxKeep,
		BackupS3Enabled:                backupS3Enabled,
		BackupS3Bucket:                 strings.TrimSpace(os.Getenv("PAYGATE_BACKUP_S3_BUCKET")),
		BackupS3Region:                 strings.TrimSpace(os.Getenv("PAYGATE_BACKUP_S3_REGION")),
		BackupS3Endpoint:               strings.TrimSpace(os.Getenv("PAYGATE_BACKUP_S3_ENDPOINT")),
		BackupS3AccessKey:              strings.TrimSpace(os.Getenv("PAYGATE_BACKUP_S3_ACCESS_KEY")),
		BackupS3Secret:                 strings.TrimSpace(os.Getenv("PAYGATE_BACKUP_S3_SECRET")),
		BackupS3ForcePathStyle:         backupS3ForcePathStyle,
		OperatorAlertWebhookURL:        strings.TrimSpace(os.Getenv("OPERATOR_ALERT_WEBHOOK_URL")),
		OperatorAlertWebhookSecret:     strings.TrimSpace(os.Getenv("OPERATOR_ALERT_WEBHOOK_SECRET")),
		StatementTimezone:              strings.TrimSpace(env("STATEMENT_TIMEZONE", "Asia/Kolkata")),
		RazorpayTestEnabled:            razorpayTestEnabled,
		RazorpayTestKeyID:              strings.TrimSpace(os.Getenv("RAZORPAY_TEST_KEY_ID")),
		RazorpayTestKeySecret:          strings.TrimSpace(os.Getenv("RAZORPAY_TEST_KEY_SECRET")),
		RazorpayTestWebhookSecret:      strings.TrimSpace(os.Getenv("RAZORPAY_TEST_WEBHOOK_SECRET")),
		RazorpayTestDisplayName:        strings.TrimSpace(env("RAZORPAY_TEST_DISPLAY_NAME", "PayGate Razorpay Test")),
		RazorpayLiveEnabled:            razorpayLiveEnabled,
		RazorpayLiveKeyID:              strings.TrimSpace(os.Getenv("RAZORPAY_LIVE_KEY_ID")),
		RazorpayLiveKeySecret:          strings.TrimSpace(os.Getenv("RAZORPAY_LIVE_KEY_SECRET")),
		RazorpayLiveWebhookSecret:      strings.TrimSpace(os.Getenv("RAZORPAY_LIVE_WEBHOOK_SECRET")),
		RazorpayLiveDisplayName:        strings.TrimSpace(env("RAZORPAY_LIVE_DISPLAY_NAME", "IEEE Sahrdaya Razorpay Live")),
	}
	cfg.GMessagesSessionPath = strings.TrimSpace(os.Getenv("GMESSAGES_SESSION_PATH"))
	if cfg.GMessagesSessionPath == "" {
		cfg.GMessagesSessionPath = filepath.Join(cfg.DataDir, "gmessages", "session.json")
	}
	return cfg, nil
}

func (c Config) ValidateServe() error {
	defaultPaymentAccount := strings.ToLower(strings.TrimSpace(c.DefaultPaymentAccount))
	if defaultPaymentAccount == "" {
		defaultPaymentAccount = "kotak"
	}
	if !c.TestMode {
		var missing []string
		if c.KotakUPIID == "" && c.UPIID == "" {
			missing = append(missing, "KOTAK_UPI_ID (or legacy UPI_ID)")
		}
		if c.APIKey == "" {
			missing = append(missing, "PAYGATE_API_KEY")
		}
		if c.SMSWebhookSecret == "" {
			missing = append(missing, "SMS_WEBHOOK_SECRET")
		}
		if len(missing) > 0 {
			return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
		}
		if len(c.APIKey) < minPrimarySecretLength {
			return fmt.Errorf("PAYGATE_API_KEY must be at least %d characters", minPrimarySecretLength)
		}
		if len(c.SMSWebhookSecret) < minPrimarySecretLength {
			return fmt.Errorf("SMS_WEBHOOK_SECRET must be at least %d characters", minPrimarySecretLength)
		}
	}
	if defaultPaymentAccount != "kotak" && defaultPaymentAccount != "slice" && defaultPaymentAccount != "paytm" {
		return errors.New("PAYMENT_DEFAULT_ACCOUNT must be kotak, slice, or paytm")
	}
	if strings.TrimSpace(c.PaytmNotificationWebhookSecret) != "" && len(c.PaytmNotificationWebhookSecret) < minPrimarySecretLength {
		return fmt.Errorf("PAYTM_NOTIFICATION_WEBHOOK_SECRET must be at least %d characters when configured", minPrimarySecretLength)
	}
	if strings.TrimSpace(c.PaytmQRPayload) != "" && len(c.PaytmNotificationWebhookSecret) < minPrimarySecretLength {
		return errors.New("PAYTM_NOTIFICATION_WEBHOOK_SECRET is required when PAYTM_QR_PAYLOAD is configured")
	}
	if _, ok := c.PaymentAccount(defaultPaymentAccount); !ok && !c.TestMode {
		return fmt.Errorf("%s payment account configuration is required for PAYMENT_DEFAULT_ACCOUNT", strings.ToUpper(defaultPaymentAccount))
	}
	if c.EmailEvidenceEnabled && strings.TrimSpace(c.SliceUPIID) == "" {
		return errors.New("SLICE_UPI_ID is required when PAYMENT_EMAIL_ENABLED=true")
	}
	if c.PaymentTTL <= 0 {
		return errors.New("PAYMENT_TTL must be positive")
	}
	if c.AmountQuarantine < 0 {
		return errors.New("PAYMENT_QUARANTINE cannot be negative")
	}
	if c.LegacySMSWebhookEnabled {
		if c.LegacySMSWebhookSecret == "" {
			return errors.New("WEBHOOK_SECRET is required when LEGACY_SMS_WEBHOOK_ENABLED=true")
		}
		if len(c.LegacySMSWebhookSecret) < minPrimarySecretLength {
			return fmt.Errorf("WEBHOOK_SECRET must be at least %d characters when LEGACY_SMS_WEBHOOK_ENABLED=true", minPrimarySecretLength)
		}
	}
	if c.EmailEvidenceEnabled {
		if len(c.EmailWebhookSecret) < minPrimarySecretLength {
			return fmt.Errorf("PAYMENT_EMAIL_WEBHOOK_SECRET must be at least %d characters when PAYMENT_EMAIL_ENABLED=true", minPrimarySecretLength)
		}
		address, err := mail.ParseAddress(c.EmailAllowedSender)
		if err != nil || !strings.EqualFold(address.Address, c.EmailAllowedSender) || !strings.Contains(address.Address, "@") {
			return errors.New("PAYMENT_EMAIL_ALLOWED_SENDER must be one exact email address")
		}
		if c.EmailAuthServID == "" {
			return errors.New("PAYMENT_EMAIL_AUTH_SERV_ID is required when PAYMENT_EMAIL_ENABLED=true")
		}
		if c.EmailSignatureTolerance <= 0 {
			return errors.New("PAYMENT_EMAIL_SIGNATURE_TOLERANCE must be positive")
		}
	}
	if c.OutgoingWebhookURL != "" {
		if err := validateHTTPURL("OUTGOING_WEBHOOK_URL", c.OutgoingWebhookURL); err != nil {
			return err
		}
		if len(c.OutgoingWebhookSecret) < minPrimarySecretLength {
			return fmt.Errorf("OUTGOING_WEBHOOK_SECRET must be at least %d characters when OUTGOING_WEBHOOK_URL is configured", minPrimarySecretLength)
		}
	}
	if c.StatementTimezone == "" {
		return errors.New("STATEMENT_TIMEZONE is required")
	}
	if _, err := time.LoadLocation(c.StatementTimezone); err != nil {
		return fmt.Errorf("STATEMENT_TIMEZONE is invalid: %w", err)
	}
	if c.RetentionEnabled && (c.SMSRawRetention <= 0 || c.EmailRawRetention <= 0 || c.ReconciliationRawRetention <= 0 || c.AuditRetention <= 0) {
		return errors.New("retention durations must be positive when PAYGATE_RETENTION_ENABLED=true")
	}
	if c.BackupCron != "" && c.BackupMaxKeep < 1 {
		return errors.New("PAYGATE_BACKUP_MAX_KEEP must be at least 1 when backups are enabled")
	}
	if (c.OperatorAlertWebhookURL == "") != (c.OperatorAlertWebhookSecret == "") {
		return errors.New("OPERATOR_ALERT_WEBHOOK_URL and OPERATOR_ALERT_WEBHOOK_SECRET must be configured together")
	}
	if c.OperatorAlertWebhookURL != "" {
		if err := validateHTTPURL("OPERATOR_ALERT_WEBHOOK_URL", c.OperatorAlertWebhookURL); err != nil {
			return err
		}
		if len(c.OperatorAlertWebhookSecret) < minPrimarySecretLength {
			return fmt.Errorf("OPERATOR_ALERT_WEBHOOK_SECRET must be at least %d characters", minPrimarySecretLength)
		}
	}
	if c.RazorpayTestEnabled {
		if !strings.HasPrefix(c.RazorpayTestKeyID, "rzp_test_") {
			return errors.New("RAZORPAY_TEST_KEY_ID must be a Test Mode key beginning with rzp_test_")
		}
		if len(c.RazorpayTestKeySecret) < 16 {
			return errors.New("RAZORPAY_TEST_KEY_SECRET must be at least 16 characters")
		}
		if len(c.RazorpayTestWebhookSecret) < minPrimarySecretLength {
			return fmt.Errorf("RAZORPAY_TEST_WEBHOOK_SECRET must be at least %d characters", minPrimarySecretLength)
		}
		if c.RazorpayTestDisplayName == "" || len(c.RazorpayTestDisplayName) > 128 {
			return errors.New("RAZORPAY_TEST_DISPLAY_NAME must be between 1 and 128 characters")
		}
	}
	if c.RazorpayLiveEnabled {
		if !strings.HasPrefix(c.RazorpayLiveKeyID, "rzp_live_") {
			return errors.New("RAZORPAY_LIVE_KEY_ID must be a Live Mode key beginning with rzp_live_")
		}
		if len(c.RazorpayLiveKeySecret) < 16 {
			return errors.New("RAZORPAY_LIVE_KEY_SECRET must be at least 16 characters")
		}
		if len(c.RazorpayLiveWebhookSecret) < minPrimarySecretLength {
			return fmt.Errorf("RAZORPAY_LIVE_WEBHOOK_SECRET must be at least %d characters", minPrimarySecretLength)
		}
		if c.RazorpayLiveDisplayName == "" || len(c.RazorpayLiveDisplayName) > 128 {
			return errors.New("RAZORPAY_LIVE_DISPLAY_NAME must be between 1 and 128 characters")
		}
	}
	if c.BackupS3Enabled {
		if c.BackupS3Bucket == "" || c.BackupS3Region == "" || c.BackupS3Endpoint == "" || c.BackupS3AccessKey == "" || c.BackupS3Secret == "" {
			return errors.New("all PAYGATE_BACKUP_S3_* values are required when S3 backup storage is enabled")
		}
		if err := validateHTTPURL("PAYGATE_BACKUP_S3_ENDPOINT", c.BackupS3Endpoint); err != nil {
			return err
		}
	}
	return nil
}

func (c Config) PaymentAccount(id string) (PaymentAccount, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	switch id {
	case "kotak":
		upiID := strings.TrimSpace(firstNonEmpty(c.KotakUPIID, c.UPIID))
		if upiID == "" && !c.TestMode {
			return PaymentAccount{}, false
		}
		return PaymentAccount{ID: "kotak", Label: "Kotak", UPIID: upiID, PayeeName: firstNonEmpty(c.KotakUPIPayeeName, c.UPIPayeeName, "PayGate"), Verification: "sms", Flow: "upi_intent"}, true
	case "slice":
		if strings.TrimSpace(c.SliceUPIID) == "" && !c.TestMode {
			return PaymentAccount{}, false
		}
		return PaymentAccount{ID: "slice", Label: "Slice", UPIID: strings.TrimSpace(c.SliceUPIID), PayeeName: firstNonEmpty(c.SliceUPIPayeeName, c.UPIPayeeName, "PayGate"), Verification: "email", Flow: "upi_intent"}, true
	case "paytm":
		if (strings.TrimSpace(c.PaytmQRPayload) == "" || strings.TrimSpace(c.PaytmNotificationWebhookSecret) == "") && !c.TestMode {
			return PaymentAccount{}, false
		}
		return PaymentAccount{ID: "paytm", Label: "Paytm", PayeeName: firstNonEmpty(c.PaytmPayeeName, "Paytm for Business"), Verification: "notification", Flow: "merchant_qr", QRPayload: strings.TrimSpace(c.PaytmQRPayload)}, true
	default:
		return PaymentAccount{}, false
	}
}

func (c Config) PaymentAccounts() []PaymentAccount {
	accounts := make([]PaymentAccount, 0, 3)
	for _, id := range []string{"kotak", "slice", "paytm"} {
		if account, ok := c.PaymentAccount(id); ok {
			accounts = append(accounts, account)
		}
	}
	return accounts
}

func validateHTTPURL(name, value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%s must be an absolute http(s) URL", name)
	}
	return nil
}

func env(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func boolEnv(name string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s must be true or false: %w", name, err)
	}
	return parsed, nil
}

func intEnv(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", name, err)
	}
	return parsed, nil
}

func durationEnv(name string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s is not a valid duration: %w", name, err)
	}
	return parsed, nil
}

func legacyMinutesEnv(name string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer number of minutes", name)
	}
	return time.Duration(n) * time.Minute, nil
}

func legacyHoursEnv(name string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer number of hours", name)
	}
	return time.Duration(n) * time.Hour, nil
}
