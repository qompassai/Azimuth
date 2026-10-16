package llm

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testClient(t *testing.T, status int, body string) (*RoseClient, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("path = %q, want /api/generate", r.URL.Path)
		}
		w.WriteHeader(status)
		if _, err := w.Write([]byte(body)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	return NewRoseClientWithHTTPClient(srv.URL, srv.Client()), srv.Close
}

// Validation: a 200 JSON answer is returned verbatim.
func TestCompleteValid(t *testing.T) {
	client, closeFn := testClient(t, http.StatusOK, `{"response":"pong"}`)
	defer closeFn()
	got, err := client.Complete("ping")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got != "pong" {
		t.Errorf("got %q, want %q", got, "pong")
	}
}

// Adversarial: empty and oversized prompts fail before any request.
func TestCompleteRejectsBadPrompts(t *testing.T) {
	client, closeFn := testClient(t, http.StatusOK, `{"response":"x"}`)
	defer closeFn()
	if _, err := client.Complete(""); !errors.Is(err, ErrEmptyPrompt) {
		t.Errorf("empty prompt err = %v, want ErrEmptyPrompt", err)
	}
	big := strings.Repeat("a", MaxPromptBytes+1)
	if _, err := client.Complete(big); err == nil {
		t.Error("oversized prompt: want error, got nil")
	}
}

// Adversarial: non-200, malformed, and oversized responses are errors.
func TestCompleteBadResponses(t *testing.T) {
	for name, tc := range map[string]struct {
		status int
		body   string
	}{
		"server error": {http.StatusInternalServerError, `{"response":"x"}`},
		"not json":     {http.StatusOK, `garbage`},
		"oversized": {http.StatusOK,
			`{"response":"` + strings.Repeat("a", MaxResponseBytes) + `"}`},
	} {
		client, closeFn := testClient(t, tc.status, tc.body)
		if _, err := client.Complete("hi"); err == nil {
			t.Errorf("%s: want error, got nil", name)
		}
		closeFn()
	}
}
