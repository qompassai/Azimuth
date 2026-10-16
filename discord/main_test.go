package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Validation: well-formed "!ask" messages yield their prompt.
func TestParseAskValid(t *testing.T) {
	cases := map[string]string{
		"!ask hello":            "hello",
		"!ask   spaced prompt ": "spaced prompt",
		"!ask\t tabbed":         "tabbed",
		"!ask what is 2+2?":     "what is 2+2?",
	}
	for content, want := range cases {
		got, ok := parseAsk(content)
		if !ok || got != want {
			t.Errorf("parseAsk(%q) = (%q, %v), want (%q, true)", content, got, ok, want)
		}
	}
}

// Adversarial: lookalikes and prompt-less commands must not query rose.
func TestParseAskRejects(t *testing.T) {
	for _, content := range []string{
		"", "!ask", "!ask ", "!ask \t ", "!askfoo bar",
		"!askme", "ask hello", " !ask hello", "hello !ask x",
	} {
		if got, ok := parseAsk(content); ok {
			t.Errorf("parseAsk(%q) = (%q, true), want ok=false", content, got)
		}
	}
}

func roseServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		if _, err := w.Write([]byte(body)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
}

// Validation: a 200 JSON answer is returned verbatim.
func TestQueryRoseFromSuccess(t *testing.T) {
	srv := roseServer(t, http.StatusOK, `{"response":"four","done":true}`)
	defer srv.Close()
	got, err := queryRoseFrom(srv.Client(), srv.URL, "what is 2+2?")
	if err != nil {
		t.Fatalf("queryRoseFrom: %v", err)
	}
	if got != "four" {
		t.Errorf("got %q, want %q", got, "four")
	}
}

// Adversarial: empty and oversized prompts fail before any request.
func TestQueryRoseFromRejectsBadPrompts(t *testing.T) {
	srv := roseServer(t, http.StatusOK, `{"response":"x","done":true}`)
	defer srv.Close()
	if _, err := queryRoseFrom(srv.Client(), srv.URL, ""); err == nil {
		t.Error("empty prompt: want error, got nil")
	}
	big := strings.Repeat("a", maxPromptBytes+1)
	if _, err := queryRoseFrom(srv.Client(), srv.URL, big); err == nil {
		t.Error("oversized prompt: want error, got nil")
	}
}

// Adversarial: non-200 and malformed bodies are errors, not answers.
func TestQueryRoseFromBadResponses(t *testing.T) {
	for name, tc := range map[string]struct {
		status int
		body   string
	}{
		"server error": {http.StatusInternalServerError, `{"response":"x"}`},
		"not json":     {http.StatusOK, `not-json`},
		"truncated":    {http.StatusOK, `{"response":`},
	} {
		srv := roseServer(t, tc.status, tc.body)
		if _, err := queryRoseFrom(srv.Client(), srv.URL, "hi"); err == nil {
			t.Errorf("%s: want error, got nil", name)
		}
		srv.Close()
	}
}

// Adversarial: a response past the byte bound is rejected.
func TestQueryRoseFromOversizedResponse(t *testing.T) {
	big := `{"response":"` + strings.Repeat("a", maxResponseBytes) + `","done":true}`
	srv := roseServer(t, http.StatusOK, big)
	defer srv.Close()
	if _, err := queryRoseFrom(srv.Client(), srv.URL, "hi"); err == nil {
		t.Error("oversized response: want error, got nil")
	}
}

// Adversarial: an unreachable server is an error, promptly.
func TestQueryRoseFromUnreachable(t *testing.T) {
	client := &http.Client{Timeout: 2 * time.Second}
	if _, err := queryRoseFrom(client, "http://127.0.0.1:1/api/generate", "hi"); err == nil {
		t.Error("unreachable server: want error, got nil")
	}
}
