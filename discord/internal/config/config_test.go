package config

import (
	"errors"
	"testing"
)

// setEnv pins the config-relevant environment for one test.
func setEnv(t *testing.T, values map[string]string) {
	t.Helper()
	for _, key := range []string{
		"DISCORD_TOKEN", "NATS_URL", "ROSE_URL", "TLS_CERT", "TLS_KEY", "TLS_CA",
	} {
		t.Setenv(key, values[key]) // missing keys become empty
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
}

// Validation: a token plus explicit endpoints load verbatim.
func TestLoadValid(t *testing.T) {
	setEnv(t, map[string]string{
		"DISCORD_TOKEN": "token-123",
		"NATS_URL":      "nats://example:4222",
		"ROSE_URL":      "https://rose.example:11434",
		"TLS_CERT":      "/tmp/cert.pem",
		"TLS_KEY":       "/tmp/key.pem",
		"TLS_CA":        "/tmp/ca.pem",
	})
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DiscordToken != "token-123" || cfg.NATSUrl != "nats://example:4222" ||
		cfg.RoseURL != "https://rose.example:11434" || cfg.TLSCert != "/tmp/cert.pem" ||
		cfg.TLSKey != "/tmp/key.pem" || cfg.TLSCA != "/tmp/ca.pem" {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

// Validation: with only a token set, the documented defaults apply.
func TestLoadDefaults(t *testing.T) {
	setEnv(t, map[string]string{"DISCORD_TOKEN": "token-123"})
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.NATSUrl != DefaultNATSURL {
		t.Errorf("NATSUrl = %q, want default %q", cfg.NATSUrl, DefaultNATSURL)
	}
	if cfg.RoseURL != DefaultRoseURL {
		t.Errorf("RoseURL = %q, want default %q", cfg.RoseURL, DefaultRoseURL)
	}
}

// Adversarial: no token is a hard error naming the cause, and no
// half-built config is returned alongside it.
func TestLoadMissingToken(t *testing.T) {
	setEnv(t, map[string]string{})
	cfg, err := Load()
	if !errors.Is(err, ErrMissingDiscordToken) {
		t.Fatalf("err = %v, want ErrMissingDiscordToken", err)
	}
	if cfg != nil {
		t.Errorf("cfg = %+v, want nil on error", cfg)
	}
}
