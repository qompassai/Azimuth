// Package model defines the message and response values that move
// between Azimuth's gateways (Discord, email, SMS) and its workers
// over the bus. Values are plain data; JSON tags are the wire format.
//
// Copyright (C) 2025 Qompass AI, All rights reserved
package model

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// Kind identifies which gateway a Message arrived through. The zero
// value is KindDiscord.
type Kind int

const (
	KindDiscord Kind = iota
	KindEmail
	KindSMS
)

// String returns the lowercase gateway name used in bus subjects
// ("msg.out.<kind>"). Out-of-range values return "unknown" instead
// of panicking: a Kind can arrive from untrusted wire data.
func (k Kind) String() string {
	switch k {
	case KindDiscord:
		return "discord"
	case KindEmail:
		return "email"
	case KindSMS:
		return "sms"
	default:
		return "unknown"
	}
}

// Message is one inbound unit of work for the LLM worker.
type Message struct {
	ID        uuid.UUID `json:"id"`
	Kind      Kind      `json:"kind"`
	ChannelID string    `json:"channel_id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// Response is the worker's reply, routed back to the originating
// gateway by ChannelID.
type Response struct {
	MessageID uuid.UUID `json:"message_id"`
	ChannelID string    `json:"channel_id"`
	Content   string    `json:"content"`
}

// NewDiscordMessage converts a Discord message event into a Message.
// A nil event (or one without an author) yields a Message carrying
// only a fresh ID and KindDiscord, rather than panicking: gateway
// events are external input and may be incomplete.
func NewDiscordMessage(m *discordgo.MessageCreate) Message {
	msg := Message{ID: uuid.New(), Kind: KindDiscord}
	if m == nil || m.Message == nil {
		return msg
	}
	msg.ChannelID = m.ChannelID
	msg.Content = m.Content
	msg.CreatedAt = m.Timestamp
	if m.Author != nil {
		msg.UserID = m.Author.ID
	}
	return msg
}
