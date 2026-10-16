// Azimuth Discord bot: answers "!ask <question>" messages by
// querying the local rose server and posting its answer back to the
// channel. All other messages are ignored.
//
// Copyright (C) 2025 Qompass AI, All rights reserved
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

const roseEndpoint = "http://localhost:11434/api/generate"

// Bounds for one rose query: a chat question and its answer are
// small, and an unbounded read of a local server response is still
// an unbounded read.
const (
	maxPromptBytes   = 32 * 1024
	maxResponseBytes = 1 << 20
	roseTimeout      = 30 * time.Second
)

// errEmptyPrompt is returned when a query is attempted with no text.
var errEmptyPrompt = errors.New("empty prompt")

// RoseRequest is the JSON body sent to rose's /api/generate.
type RoseRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// RoseResponse is the JSON body rose returns from /api/generate.
type RoseResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func main() {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		log.Fatal("DISCORD_BOT_TOKEN not set")
	}

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatalf("error creating Discord session: %v", err)
	}

	dg.AddHandler(messageCreate)

	if err := dg.Open(); err != nil {
		log.Fatalf("error opening connection: %v", err)
	}
	log.Println("Bot is running. Press CTRL-C to exit.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop

	if err := dg.Close(); err != nil {
		log.Printf("error closing Discord session: %v", err)
	}
}

// parseAsk extracts the prompt from a message of the exact form
// "!ask <prompt>". Anything else — another command, "!ask" with no
// prompt, or a lookalike such as "!askfoo" — reports ok=false so
// the caller stays silent instead of querying rose with garbage.
func parseAsk(content string) (prompt string, ok bool) {
	rest, found := strings.CutPrefix(content, "!ask")
	if !found || rest == "" || (rest[0] != ' ' && rest[0] != '\t') {
		return "", false
	}
	prompt = strings.TrimSpace(rest)
	if prompt == "" {
		return "", false
	}
	return prompt, true
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m == nil || m.Author == nil || s == nil || s.State == nil || s.State.User == nil {
		return
	}
	if m.Author.ID == s.State.User.ID {
		return
	}
	prompt, ok := parseAsk(m.Content)
	if !ok {
		return
	}
	resp, err := queryRose(prompt)
	if err != nil {
		if _, sendErr := s.ChannelMessageSend(m.ChannelID, "❌ Error querying Rose: "+err.Error()); sendErr != nil {
			log.Printf("error sending error reply: %v", sendErr)
		}
		return
	}
	if _, err := s.ChannelMessageSend(m.ChannelID, resp); err != nil {
		log.Printf("error sending reply: %v", err)
	}
}

// queryRose queries the bot's own rose endpoint (roseEndpoint) with
// a bounded HTTP client.
func queryRose(prompt string) (string, error) {
	client := &http.Client{Timeout: roseTimeout}
	return queryRoseFrom(client, roseEndpoint, prompt)
}

// queryRoseFrom posts prompt to endpoint and returns rose's answer
// text. An empty or oversized prompt, a non-200 status, or a
// malformed or oversized response is an error, never a partial
// answer. endpoint and client are parameters so tests can use an
// httptest server.
func queryRoseFrom(client *http.Client, endpoint, prompt string) (string, error) {
	if prompt == "" {
		return "", errEmptyPrompt
	}
	if len(prompt) > maxPromptBytes {
		return "", fmt.Errorf("prompt is %d bytes, max %d", len(prompt), maxPromptBytes)
	}
	jsonData, err := json.Marshal(RoseRequest{Model: "rose", Prompt: prompt, Stream: false})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}
	resp, err := client.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("post to rose: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing rose response body: %v", err)
		}
		return "", fmt.Errorf("rose returned status %s", resp.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if closeErr := resp.Body.Close(); err == nil && closeErr != nil {
		err = closeErr
	}
	if err != nil {
		return "", fmt.Errorf("read rose response: %w", err)
	}
	if len(raw) > maxResponseBytes {
		return "", fmt.Errorf("rose response exceeds %d bytes", maxResponseBytes)
	}
	var roseResp RoseResponse
	if err := json.Unmarshal(raw, &roseResp); err != nil {
		return "", fmt.Errorf("decode rose response: %w", err)
	}
	return roseResp.Response, nil
}
