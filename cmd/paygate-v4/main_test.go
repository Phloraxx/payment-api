package main

import "testing"

func TestConfigFromEnvUsesExplicitV4BootstrapValues(t *testing.T) {
	t.Setenv("PAYGATE_V4_DATA_DIR", t.TempDir())
	t.Setenv("PAYGATE_V4_MERCHANT_API_KEY", "v4-merchant-key-0123456789abcdef012345678901")
	t.Setenv("PAYGATE_V4_WEBHOOK_ENDPOINT", "https://v4.example/paygate")
	t.Setenv("PAYGATE_V4_WEBHOOK_SECRET", "v4-webhook-secret-0123456789abcdef012345")
	cfg, err := configFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BootstrapMerchantAPIKey != "v4-merchant-key-0123456789abcdef012345678901" {
		t.Fatal("explicit v4 merchant key missing")
	}
	if cfg.BootstrapWebhookEndpoint != "https://v4.example/paygate" {
		t.Fatal("explicit v4 webhook endpoint missing")
	}
	if cfg.BootstrapWebhookSecret != "v4-webhook-secret-0123456789abcdef012345" {
		t.Fatal("explicit v4 webhook secret missing")
	}
}

func TestConfigFromEnvIgnoresRemovedLegacyBootstrapAliases(t *testing.T) {
	t.Setenv("PAYGATE_V4_DATA_DIR", t.TempDir())
	t.Setenv("PAYGATE_API_KEY", "retired-merchant-key")
	t.Setenv("OUTGOING_WEBHOOK_URL", "https://legacy.example/paygate")
	t.Setenv("OUTGOING_WEBHOOK_SECRET", "retired-webhook-secret")
	cfg, err := configFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BootstrapMerchantAPIKey != "" || cfg.BootstrapWebhookEndpoint != "" || cfg.BootstrapWebhookSecret != "" {
		t.Fatal("retired v3 environment aliases must not bootstrap v4")
	}
}
