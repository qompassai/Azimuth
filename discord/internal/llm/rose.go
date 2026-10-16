// Package llm is Azimuth's client for the rose LLM server's
// /api/generate endpoint.
//
// Copyright (C) 2025 Qompass AI, All rights reserved
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/qompassai/azimuth/internal/auth"
	"github.com/qompassai/azimuth/internal/config"
)

// Bounds for one completion: a prompt and a response are single
// chat messages, so both are capped well below any transport limit.
const (
	MaxPromptBytes   = 32 * 1024
	MaxResponseBytes = 1 << 20
	RequestTimeout   = 120 * time.Second
)

// ErrEmptyPrompt is returned when a completion is requested with no
// prompt text; the server would only waste a generation on it.
var ErrEmptyPrompt = errors.New("llm: empty prompt")

// RoseClient calls one rose server. It is safe for concurrent use;
// the underlying http.Client owns connection pooling.
type RoseClient struct {
	url string
	hc  *http.Client
}

type request struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type response struct {
	Answer string `json:"response"`
}

// NewRoseClient builds a client for cfg.RoseURL using mutual TLS
// from cfg's certificate paths (see package auth).
func NewRoseClient(cfg *config.Config) *RoseClient {
	tlsCfg := auth.MustTLS(cfg)
	return &RoseClient{
		url: cfg.RoseURL,
		hc: &http.Client{
			Timeout:   RequestTimeout,
			Transport: &http.Transport{TLSClientConfig: tlsCfg},
		},
	}
}

// NewRoseClientWithHTTPClient builds a client over a caller-owned
// http.Client. It exists for tests (httptest servers) and for
// callers that manage their own transport; production code uses
// NewRoseClient.
func NewRoseClientWithHTTPClient(url string, hc *http.Client) *RoseClient {
	if hc == nil {
		hc = &http.Client{Timeout: RequestTimeout}
	}
	return &RoseClient{url: url, hc: hc}
}

// Complete asks rose to complete prompt and returns its answer text.
// It is CompleteContext with a background context.
func (c *RoseClient) Complete(prompt string) (string, error) {
	return c.CompleteContext(context.Background(), prompt)
}

// CompleteContext is Complete with caller cancellation. Failures —
// an empty or oversized prompt, a transport error, a non-200
// status, or a malformed or oversized response body — are returned
// as errors; no partial answer is ever returned alongside one.
func (c *RoseClient) CompleteContext(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		return "", ErrEmptyPrompt
	}
	if len(prompt) > MaxPromptBytes {
		return "", fmt.Errorf("llm: prompt is %d bytes, max %d", len(prompt), MaxPromptBytes)
	}
	body, err := json.Marshal(request{Model: "rose", Prompt: prompt, Stream: false})
	if err != nil {
		return "", fmt.Errorf("llm: marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.url+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("llm: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm: do request: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		if err := res.Body.Close(); err != nil {
			return "", fmt.Errorf("llm: rose returned status %s (close: %v)", res.Status, err)
		}
		return "", fmt.Errorf("llm: rose returned status %s", res.Status)
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, MaxResponseBytes+1))
	if closeErr := res.Body.Close(); err == nil && closeErr != nil {
		err = closeErr
	}
	if err != nil {
		return "", fmt.Errorf("llm: read response: %w", err)
	}
	if len(raw) > MaxResponseBytes {
		return "", fmt.Errorf("llm: response exceeds %d bytes", MaxResponseBytes)
	}
	var r response
	if err := json.Unmarshal(raw, &r); err != nil {
		return "", fmt.Errorf("llm: decode response: %w", err)
	}
	return r.Answer, nil
}
