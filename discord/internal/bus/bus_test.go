package bus

import (
	"testing"

	"github.com/qompassai/azimuth/internal/config"
)

// Adversarial: Subscribe validates its arguments before touching
// the server — empty subject, nil handler, and nil context all
// error instead of panicking or silently dropping messages.
func TestSubscribeValidatesArgs(t *testing.T) {
	handler := func([]byte) {}
	if err := Subscribe(nil, "", handler); err == nil {
		t.Error("empty subject: want error, got nil")
	}
	if err := Subscribe(nil, "msg.in.discord", nil); err == nil {
		t.Error("nil handler: want error, got nil")
	}
	if err := Subscribe(nil, "msg.in.discord", handler); err == nil {
		t.Error("nil jetstream: want error, got nil")
	}
}

// Adversarial: a nil or URL-less config cannot connect.
func TestConnectJetStreamValidatesConfig(t *testing.T) {
	if _, err := ConnectJetStream(nil); err == nil {
		t.Error("nil config: want error, got nil")
	}
	if _, err := ConnectJetStream(&config.Config{}); err == nil {
		t.Error("empty NATS_URL: want error, got nil")
	}
}

// Adversarial: an unreachable NATS server errors within the
// connect timeout instead of hanging startup forever.
func TestConnectJetStreamUnreachable(t *testing.T) {
	cfg := &config.Config{NATSUrl: "nats://127.0.0.1:1"}
	if _, err := ConnectJetStream(cfg); err == nil {
		t.Error("unreachable server: want error, got nil")
	}
}
