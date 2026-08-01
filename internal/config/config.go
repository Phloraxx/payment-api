package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const minPrimarySecretLength = 24

type Config struct {
	DataDir                    string
	UPIID                      string
	UPIPayeeName               string
	APIKey                     string
	SMSWebhookSecret           string
	LegacySMSWebhookSecret     string
	LegacySMSWebhookEnabled    bool
	PaymentTTL                 time.Duration
	AmountQuarantine           time.Duration
	OutgoingWebhookURL         string
	OutgoingWebhookSecret      string
	GMessagesEnabled           bool
	GMessagesSessionPath       string
	TestMode                   bool
	RateLimitsEnabled          bool
	RetentionEnabled           bool
	SMSRawRetention            time.Duration
	ReconciliationRawRetention time.Duration
	AuditRetention             time.Duration
	BackupCron                 string
	BackupMaxKeep              int
	BackupS3Enabled            bool
	BackupS3Bucket             string
	BackupS3Region             string
	BackupS3Endpoint           string
	BackupS3AccessKey          string
	BackupS3Secret             string
	BackupS3ForcePathStyle     bool
	OperatorAlertWebhookURL    string
	OperatorAlertWebhookSecret string
	StatementTimezone          string
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
	retentionEnabled, err := boolEnv("PAYGATE_RETENTION_ENABLED", true)
	if err != nil {
		return Config{}, err
	}
	smsRawRetention, err := durationEnv("SMS_RAW_RETENTION", 90*24*time.Hour)
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

	dataDir := strings.TrimSpace(env("PB_DATA_DIR", "./pb_data"))
	cfg := Config{
		DataDir:                    dataDir,
		UPIID:                      strings.TrimSpace(os.Getenv("UPI_ID")),
		UPIPayeeName:               strings.TrimSpace(firstNonEmpty(os.Getenv("UPI_PAYEE_NAME"), os.Getenv("UPI_NAME"), "PayGate")),
		APIKey:                     strings.TrimSpace(os.Getenv("PAYGATE_API_KEY")),
		SMSWebhookSecret:           strings.TrimSpace(os.Getenv("SMS_WEBHOOK_SECRET")),
		LegacySMSWebhookSecret:     strings.TrimSpace(os.Getenv("WEBHOOK_SECRET")),
		LegacySMSWebhookEnabled:    legacyEnabled,
		PaymentTTL:                 paymentTTL,
		AmountQuarantine:           quarantine,
		OutgoingWebhookURL:         strings.TrimSpace(firstNonEmpty(os.Getenv("OUTGOING_WEBHOOK_URL"), os.Getenv("PAYMENT_WEBHOOK_URL"))),
		OutgoingWebhookSecret:      strings.TrimSpace(firstNonEmpty(os.Getenv("OUTGOING_WEBHOOK_SECRET"), os.Getenv("PAYMENT_WEBHOOK_SECRET"))),
		GMessagesEnabled:           gmessagesEnabled,
		TestMode:                   testMode,
		RateLimitsEnabled:          rateLimitsEnabled,
		RetentionEnabled:           retentionEnabled,
		SMSRawRetention:            smsRawRetention,
		ReconciliationRawRetention: reconciliationRetention,
		AuditRetention:             auditRetention,
		BackupCron:                 strings.TrimSpace(env("PAYGATE_BACKUP_CRON", "0 3 * * *")),
		BackupMaxKeep:              backupMaxKeep,
		BackupS3Enabled:            backupS3Enabled,
		BackupS3Bucket:             strings.TrimSpace(os.Getenv("PAYGATE_BACKUP_S3_BUCKET")),
		BackupS3Region:             strings.TrimSpace(os.Getenv("PAYGATE_BACKUP_S3_REGION")),
		BackupS3Endpoint:           strings.TrimSpace(os.Getenv("PAYGATE_BACKUP_S3_ENDPOINT")),
		BackupS3AccessKey:          strings.TrimSpace(os.Getenv("PAYGATE_BACKUP_S3_ACCESS_KEY")),
		BackupS3Secret:             strings.TrimSpace(os.Getenv("PAYGATE_BACKUP_S3_SECRET")),
		BackupS3ForcePathStyle:     backupS3ForcePathStyle,
		OperatorAlertWebhookURL:    strings.TrimSpace(os.Getenv("OPERATOR_ALERT_WEBHOOK_URL")),
		OperatorAlertWebhookSecret: strings.TrimSpace(os.Getenv("OPERATOR_ALERT_WEBHOOK_SECRET")),
		StatementTimezone:          strings.TrimSpace(env("STATEMENT_TIMEZONE", "Asia/Kolkata")),
	}
	cfg.GMessagesSessionPath = strings.TrimSpace(os.Getenv("GMESSAGES_SESSION_PATH"))
	if cfg.GMessagesSessionPath == "" {
		cfg.GMessagesSessionPath = filepath.Join(cfg.DataDir, "gmessages", "session.json")
	}
	return cfg, nil
}

func (c Config) ValidateServe() error {
	if !c.TestMode {
		var missing []string
		if c.UPIID == "" {
			missing = append(missing, "UPI_ID")
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
	if c.RetentionEnabled && (c.SMSRawRetention <= 0 || c.ReconciliationRawRetention <= 0 || c.AuditRetention <= 0) {
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
