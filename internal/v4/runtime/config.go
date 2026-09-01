package runtime

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	DataDir                  string
	ListenAddr               string
	PublicURL                string
	AllowedOrigins           []string
	BootstrapAdminPassword   string
	BootstrapMerchantAPIKey  string
	BootstrapWebhookEndpoint string
	BootstrapWebhookSecret   string
	BackupDir                string
	BackupHourUTC            int
	BackupRetention          int
	ExpiryInterval           time.Duration
}

func (c Config) normalized() (Config, error) {
	if strings.TrimSpace(c.DataDir) == "" {
		return Config{}, errors.New("PayGate v4 data directory is required")
	}
	abs, err := filepath.Abs(c.DataDir)
	if err != nil {
		return Config{}, err
	}
	c.DataDir = abs
	c.ListenAddr = strings.TrimSpace(c.ListenAddr)
	if c.ListenAddr == "" {
		c.ListenAddr = ":8091"
	}
	c.PublicURL = strings.TrimRight(strings.TrimSpace(c.PublicURL), "/")
	if c.PublicURL != "" {
		u, err := url.Parse(c.PublicURL)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return Config{}, errors.New("PayGate public URL must be an https origin")
		}
	}
	if strings.TrimSpace(c.BackupDir) == "" {
		c.BackupDir = filepath.Join(c.DataDir, "backups")
	} else if c.BackupDir, err = filepath.Abs(c.BackupDir); err != nil {
		return Config{}, err
	}
	if c.BackupHourUTC < 0 || c.BackupHourUTC > 23 {
		return Config{}, errors.New("backup hour UTC must be between 0 and 23")
	}
	if c.BackupRetention == 0 {
		c.BackupRetention = 30
	}
	if c.BackupRetention < 1 || c.BackupRetention > 365 {
		return Config{}, errors.New("backup retention must be between 1 and 365")
	}
	if c.ExpiryInterval <= 0 {
		c.ExpiryInterval = 30 * time.Second
	}
	origins := make([]string, 0, len(c.AllowedOrigins))
	seen := map[string]struct{}{}
	for _, origin := range c.AllowedOrigins {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")
		if origin == "" {
			continue
		}
		u, err := url.Parse(origin)
		if err != nil || u.Scheme == "" || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return Config{}, errors.New("allowed origins must be origins only")
		}
		if _, ok := seen[origin]; ok {
			continue
		}
		seen[origin] = struct{}{}
		origins = append(origins, origin)
	}
	c.AllowedOrigins = origins
	return c, nil
}
