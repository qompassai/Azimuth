// Package config loads Azimuth's runtime configuration from the
// environment. Configuration is immutable after Load returns.
//
// Copyright (C) 2025 Qompass AI, All rights reserved
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Default endpoints, applied when the environment does not set them.
// These are the deployment defaults recorded in the layout blueprint.
const (
	DefaultNATSURL = "nats://nats:4222"
	DefaultRoseURL = "https://rose:11434"
)

// ErrMissingDiscordToken is returned by Load when DISCORD_TOKEN is
// unset or empty; a bot without a token cannot authenticate, so this
// is a hard configuration error rather than a default.
var ErrMissingDiscordToken = errors.New("config: DISCORD_TOKEN is required")

// Config is the resolved runtime configuration. Field tags name the
// environment variable each field is read from.
type Config struct {
	DiscordToken string `mapstructure:"DISCORD_TOKEN"`
	NATSUrl      string `mapstructure:"NATS_URL"`
	RoseURL      string `mapstructure:"ROSE_URL"`
	TLSCert      string `mapstructure:"TLS_CERT"`
	TLSKey       string `mapstructure:"TLS_KEY"`
	TLSCA        string `mapstructure:"TLS_CA"`
}

// Load resolves the configuration from the environment and applies
// the documented defaults. It returns an error instead of exiting so
// callers (and tests) own the failure policy; MustLoad is the fatal
// wrapper used by the cmd entry points.
func Load() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetDefault("NATS_URL", DefaultNATSURL)
	v.SetDefault("ROSE_URL", DefaultRoseURL)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal environment: %w", err)
	}
	// Viper's AutomaticEnv does not reliably bind keys that have no
	// explicit default during Unmarshal; read the exact names as a
	// fallback so the documented variables always win.
	cfg.DiscordToken = envOr("DISCORD_TOKEN", cfg.DiscordToken)
	cfg.NATSUrl = envOr("NATS_URL", cfg.NATSUrl)
	cfg.RoseURL = envOr("ROSE_URL", cfg.RoseURL)
	cfg.TLSCert = envOr("TLS_CERT", cfg.TLSCert)
	cfg.TLSKey = envOr("TLS_KEY", cfg.TLSKey)
	cfg.TLSCA = envOr("TLS_CA", cfg.TLSCA)
	if cfg.NATSUrl == "" {
		cfg.NATSUrl = DefaultNATSURL
	}
	if cfg.RoseURL == "" {
		cfg.RoseURL = DefaultRoseURL
	}
	if cfg.DiscordToken == "" {
		return nil, ErrMissingDiscordToken
	}
	return &cfg, nil
}

// MustLoad is Load with a fatal exit on error, preserving the
// entry-point behavior the cmd programs were written against.
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}
	return cfg
}

func envOr(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
