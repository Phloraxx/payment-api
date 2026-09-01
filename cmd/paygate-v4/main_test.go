package main

import "testing"

func TestConfigFromEnvUsesCutoverCompatibilityFallbacks(t *testing.T) {
	t.Setenv("PAYGATE_V4_DATA_DIR", t.TempDir())
	t.Setenv("PAYGATE_V4_MERCHANT_API_KEY", "")
	t.Setenv("PAYGATE_V4_WEBHOOK_ENDPOINT", "")
	t.Setenv("PAYGATE_V4_WEBHOOK_SECRET", "")
	t.Setenv("PAYGATE_API_KEY", "legacy-merchant-key-0123456789abcdef0123456789")
	t.Setenv("OUTGOING_WEBHOOK_URL", "https://merchant.example/paygate")
	t.Setenv("OUTGOING_WEBHOOK_SECRET", "legacy-webhook-secret-0123456789abcdef")
	cfg, err := configFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BootstrapMerchantAPIKey != "legacy-merchant-key-0123456789abcdef0123456789" {
		t.Fatal("legacy merchant key fallback missing")
	}
	if cfg.BootstrapWebhookEndpoint != "https://merchant.example/paygate" {
		t.Fatal("legacy webhook endpoint fallback missing")
	}
	if cfg.BootstrapWebhookSecret != "legacy-webhook-secret-0123456789abcdef" {
		t.Fatal("legacy webhook secret fallback missing")
	}
}

func TestConfigFromEnvPrefersExplicitV4CutoverValues(t *testing.T) {
	t.Setenv("PAYGATE_V4_DATA_DIR", t.TempDir())
	t.Setenv("PAYGATE_API_KEY", "legacy-merchant-key-0123456789abcdef0123456789")
	t.Setenv("OUTGOING_WEBHOOK_URL", "https://legacy.example/paygate")
	t.Setenv("OUTGOING_WEBHOOK_SECRET", "legacy-webhook-secret-0123456789abcdef")
	t.Setenv("PAYGATE_V4_MERCHANT_API_KEY", "v4-merchant-key-0123456789abcdef012345678901")
	t.Setenv("PAYGATE_V4_WEBHOOK_ENDPOINT", "https://v4.example/paygate")
	t.Setenv("PAYGATE_V4_WEBHOOK_SECRET", "v4-webhook-secret-0123456789abcdef012345")
	cfg, err := configFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BootstrapMerchantAPIKey != "v4-merchant-key-0123456789abcdef012345678901" {
		t.Fatal("explicit v4 merchant key not preferred")
	}
	if cfg.BootstrapWebhookEndpoint != "https://v4.example/paygate" {
		t.Fatal("explicit v4 webhook endpoint not preferred")
	}
	if cfg.BootstrapWebhookSecret != "v4-webhook-secret-0123456789abcdef012345" {
		t.Fatal("explicit v4 webhook secret not preferred")
	}
}
