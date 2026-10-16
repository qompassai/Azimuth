// Package auth builds Azimuth's TLS configuration: mutual TLS with
// a TLS 1.3 floor, from certificate files named in the config.
//
// Copyright (C) 2025 Qompass AI, All rights reserved
package auth

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"github.com/qompassai/azimuth/internal/config"
	"github.com/rs/zerolog/log"
)

// LoadTLS loads the client certificate pair and CA bundle named by
// cfg and returns a TLS configuration with a TLS 1.3 minimum and
// certificate verification enabled. Every failure is an error
// value — a missing file, an unparsable pair, or a CA bundle with
// no valid certificates — never a partial configuration.
func LoadTLS(cfg *config.Config) (*tls.Config, error) {
	if cfg == nil {
		return nil, errors.New("auth: nil config")
	}
	if cfg.TLSCert == "" || cfg.TLSKey == "" || cfg.TLSCA == "" {
		return nil, errors.New("auth: TLS_CERT, TLS_KEY and TLS_CA must all be set")
	}
	cert, err := tls.LoadX509KeyPair(cfg.TLSCert, cfg.TLSKey)
	if err != nil {
		return nil, fmt.Errorf("auth: load cert pair: %w", err)
	}
	ca, err := os.ReadFile(cfg.TLSCA)
	if err != nil {
		return nil, fmt.Errorf("auth: read ca: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return nil, errors.New("auth: ca bundle contains no valid certificates")
	}
	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            pool,
		MinVersion:         tls.VersionTLS13,
		CurvePreferences:   []tls.CurveID{tls.X25519}, // replace once Go supports Kyber hybrid
		InsecureSkipVerify: false,
	}, nil
}

// MustTLS is LoadTLS with a fatal exit on error, preserving the
// entry-point behavior existing callers were written against.
func MustTLS(cfg *config.Config) *tls.Config {
	tlsCfg, err := LoadTLS(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("load tls config")
	}
	return tlsCfg
}
