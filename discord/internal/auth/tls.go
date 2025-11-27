// ~/.GH/Qompass/Go/Azimuth/discord/internal/auth/tls.go
// -----------------------------------------------------
// Copyright (C) 2025 Qompass AI, All rights reserved

package auth

import (
    "crypto/tls"
    "crypto/x509"
    "os"

    "github.com/qompassai/azimuth/internal/config"
    "github.com/rs/zerolog/log"
)

func MustTLS(cfg *config.Config) *tls.Config {
    cert, err := tls.LoadX509KeyPair(cfg.TLSCert, cfg.TLSKey)
    if err != nil {
        log.Fatal().Err(err).Msg("load cert")
    }
    ca, err := os.ReadFile(cfg.TLSCA)
    if err != nil {
        log.Fatal().Err(err).Msg("read ca")
    }
    pool := x509.NewCertPool()
    pool.AppendCertsFromPEM(ca)

    return &tls.Config{
        Certificates:       []tls.Certificate{cert},
        RootCAs:            pool,
        MinVersion:         tls.VersionTLS13,
        CurvePreferences:   []tls.CurveID{tls.X25519}, // replace once Go supports Kyber hybrid
        InsecureSkipVerify: false,
    }
}

