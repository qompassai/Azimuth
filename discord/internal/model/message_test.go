package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// Validation: every defined Kind names its gateway.
func TestKindStringValid(t *testing.T) {
	cases := map[Kind]string{
		KindDiscord: "discord",
		KindEmail:   "email",
		KindSMS:     "sms",
	}
	for kind, want := range cases {
		if got := kind.String(); got != want {
			t.Errorf("Kind(%d).String() = %q, want %q", int(kind), got, want)
		}
	}
}

// Adversarial: out-of-range Kind values (possible from wire data)
// must not panic and must not impersonate a real gateway.
func TestKindStringOutOfRange(t *testing.T) {
	for _, kind := range []Kind{-1, 3, 99} {
		if got := kind.String(); got != "unknown" {
			t.Errorf("Kind(%d).String() = %q, want %q", int(kind), got, "unknown")
		}
	}
}

// Validation: a Discord event maps field-for-field into a Message.
func TestNewDiscordMessageValid(t *testing.T) {
	stamp := time.Unix(1700000000, 0).UTC()
	ev := &discordgo.MessageCreate{Message: &discordgo.Message{
		ChannelID: "chan-1",
		Content:   "hello",
		Timestamp: stamp,
		Author:    &discordgo.User{ID: "user-9"},
	}}
	msg := NewDiscordMessage(ev)
	if msg.ID == uuid.Nil {
		t.Error("ID is nil UUID")
	}
	if msg.Kind != KindDiscord || msg.ChannelID != "chan-1" ||
		msg.UserID != "user-9" || msg.Content != "hello" || !msg.CreatedAt.Equal(stamp) {
		t.Errorf("unexpected mapping: %+v", msg)
	}
}

// Adversarial: nil and author-less events must not panic.
func TestNewDiscordMessageNilSafe(t *testing.T) {
	if msg := NewDiscordMessage(nil); msg.ID == uuid.Nil || msg.Kind != KindDiscord {
		t.Errorf("nil event: %+v", msg)
	}
	ev := &discordgo.MessageCreate{Message: &discordgo.Message{Content: "x"}}
	if msg := NewDiscordMessage(ev); msg.UserID != "" || msg.Content != "x" {
		t.Errorf("author-less event: %+v", msg)
	}
}

// Validation: Message and Response survive a JSON round-trip,
// which is the bus wire format.
func TestMessageJSONRoundTrip(t *testing.T) {
	in := Message{
		ID: uuid.New(), Kind: KindEmail, ChannelID: "c",
		UserID: "u", Content: "body", CreatedAt: time.Unix(1700000000, 0).UTC(),
	}
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var out Message
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Errorf("round-trip mismatch:\n in %+v\nout %+v", in, out)
	}
}
