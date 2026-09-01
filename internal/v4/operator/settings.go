package operator

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Phloraxx/payment-api/internal/v4/storage"
	"github.com/Phloraxx/payment-api/internal/v4/webhooks"
)

const (
	webhookEndpointKey = "webhook_endpoint"
	webhookSecretKey   = "webhook_secret"
)

type SettingsService struct {
	DB     *storage.DB
	Worker *webhooks.Service
	Random io.Reader
	Now    func() time.Time
}

type WebhookSettings struct {
	Enabled          bool
	Endpoint         string
	SecretConfigured bool
}

func NewSettingsService(db *storage.DB, worker *webhooks.Service) *SettingsService {
	return &SettingsService{DB: db, Worker: worker, Random: rand.Reader, Now: time.Now}
}
func (s *SettingsService) Webhook(ctx context.Context) (WebhookSettings, error) {
	cfg, err := s.loadWebhookConfig(ctx)
	if err != nil {
		return WebhookSettings{}, err
	}
	return WebhookSettings{
		Enabled:          strings.TrimSpace(cfg.Endpoint) != "" && strings.TrimSpace(cfg.Secret) != "",
		Endpoint:         cfg.Endpoint,
		SecretConfigured: strings.TrimSpace(cfg.Secret) != "",
	}, nil
}

func (s *SettingsService) ConfigureWebhook(ctx context.Context, endpoint string, rotateSecret bool) (WebhookSettings, string, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return WebhookSettings{}, "", errors.New("settings storage is required")
	}
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		if err := s.storeWebhookConfig(ctx, webhooks.Config{}); err != nil {
			return WebhookSettings{}, "", err
		}
		if s.Worker != nil {
			_ = s.Worker.UpdateConfig(webhooks.Config{})
		}
		return WebhookSettings{}, "", nil
	}
	existing, err := s.loadWebhookConfig(ctx)
	if err != nil {
		return WebhookSettings{}, "", err
	}
	secret := existing.Secret
	newSecret := ""
	if secret == "" || rotateSecret {
		secret, err = s.generateWebhookSecret()
		if err != nil {
			return WebhookSettings{}, "", err
		}
		newSecret = secret
	}
	cfg := webhooks.Config{Endpoint: endpoint, Secret: secret}
	if err := webhooks.ValidateConfig(cfg); err != nil {
		return WebhookSettings{}, "", err
	}
	if err := s.storeWebhookConfig(ctx, cfg); err != nil {
		return WebhookSettings{}, "", err
	}
	if s.Worker != nil {
		if err := s.Worker.UpdateConfig(cfg); err != nil {
			return WebhookSettings{}, "", err
		}
	}
	return WebhookSettings{Enabled: true, Endpoint: endpoint, SecretConfigured: true}, newSecret, nil
}
func (s *SettingsService) ApplyPersistedWebhook(ctx context.Context) error {
	if s == nil || s.Worker == nil {
		return nil
	}
	cfg, err := s.loadWebhookConfig(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.Endpoint) == "" && strings.TrimSpace(cfg.Secret) == "" {
		return s.Worker.UpdateConfig(webhooks.Config{})
	}
	return s.Worker.UpdateConfig(cfg)
}

func (s *SettingsService) loadWebhookConfig(ctx context.Context) (webhooks.Config, error) {
	if s == nil || s.DB == nil || s.DB.SQL == nil {
		return webhooks.Config{}, errors.New("settings storage is required")
	}
	endpoint, err := readSetting(ctx, s.DB.SQL, webhookEndpointKey)
	if err != nil {
		return webhooks.Config{}, err
	}
	secret, err := readSetting(ctx, s.DB.SQL, webhookSecretKey)
	if err != nil {
		return webhooks.Config{}, err
	}
	return webhooks.Config{Endpoint: endpoint, Secret: secret}, nil
}
func (s *SettingsService) storeWebhookConfig(ctx context.Context, cfg webhooks.Config) error {
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	return s.DB.WithImmediateTx(ctx, func(tx *storage.ImmediateTx) error {
		if err := writeSetting(ctx, tx, webhookEndpointKey, strings.TrimSpace(cfg.Endpoint), now); err != nil {
			return err
		}
		if err := writeSetting(ctx, tx, webhookSecretKey, strings.TrimSpace(cfg.Secret), now); err != nil {
			return err
		}
		return nil
	})
}

func (s *SettingsService) generateWebhookSecret() (string, error) {
	r := s.Random
	if r == nil {
		r = rand.Reader
	}
	raw := make([]byte, 32)
	if _, err := io.ReadFull(r, raw); err != nil {
		return "", fmt.Errorf("generate webhook secret: %w", err)
	}
	return "whsec_" + base64.RawURLEncoding.EncodeToString(raw), nil
}

type settingQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type settingExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func readSetting(ctx context.Context, db settingQuerier, key string) (string, error) {
	var value string
	err := db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key=?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read setting %s: %w", key, err)
	}
	return value, nil
}

func writeSetting(ctx context.Context, db settingExecer, key, value string, at time.Time) error {
	_, err := db.ExecContext(ctx, `INSERT INTO settings(key,value,updated_at) VALUES(?,?,?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value,updated_at=excluded.updated_at`, key, value, at.UnixMilli())
	if err != nil {
		return fmt.Errorf("write setting %s: %w", key, err)
	}
	return nil
}
