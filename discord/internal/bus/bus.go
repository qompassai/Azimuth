// Package bus connects Azimuth's gateways and workers to NATS
// JetStream. Handlers receive raw message payloads; decoding is the
// subscriber's job.
//
// Copyright (C) 2025 Qompass AI, All rights reserved
package bus

import (
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/qompassai/azimuth/internal/config"
	"github.com/rs/zerolog/log"
)

// ConnectTimeout bounds a JetStream connection attempt so a missing
// or unreachable NATS server fails fast instead of blocking startup.
const ConnectTimeout = 5 * time.Second

// Handler processes one message payload. It is called on a NATS
// subscription goroutine and must not block indefinitely.
type Handler func(data []byte)

// ConnectJetStream dials the configured NATS server and returns its
// JetStream context. Errors are returned, not fatalled, so callers
// own the failure policy; MustJetStream is the fatal wrapper.
func ConnectJetStream(cfg *config.Config) (nats.JetStreamContext, error) {
	if cfg == nil {
		return nil, errors.New("bus: nil config")
	}
	if cfg.NATSUrl == "" {
		return nil, errors.New("bus: NATS_URL is empty")
	}
	nc, err := nats.Connect(cfg.NATSUrl, nats.Name("azimuth"), nats.Timeout(ConnectTimeout))
	if err != nil {
		return nil, fmt.Errorf("bus: nats connect: %w", err)
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("bus: jetstream context: %w", err)
	}
	return js, nil
}

// MustJetStream is ConnectJetStream with a fatal exit on error,
// preserving the entry-point behavior the cmd programs use.
func MustJetStream(cfg *config.Config) nats.JetStreamContext {
	js, err := ConnectJetStream(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("jetstream connect")
	}
	return js
}

// Subscribe registers fn for subject. The subscription arguments
// are validated before touching the server: a nil context, an empty
// subject, or a nil handler is a programmer error reported as an
// error value, never a panic or a silent no-op.
func Subscribe(js nats.JetStreamContext, subject string, fn Handler) error {
	if subject == "" {
		return errors.New("bus: empty subject")
	}
	if fn == nil {
		return errors.New("bus: nil handler")
	}
	if js == nil {
		return errors.New("bus: nil jetstream context")
	}
	if _, err := js.Subscribe(subject, func(m *nats.Msg) { fn(m.Data) }); err != nil {
		return fmt.Errorf("bus: subscribe %q: %w", subject, err)
	}
	return nil
}

// MustSubscribe is Subscribe with a fatal exit on error.
func MustSubscribe(js nats.JetStreamContext, subject string, fn Handler) {
	if err := Subscribe(js, subject, fn); err != nil {
		log.Fatal().Err(err).Msg("subscribe")
	}
}
