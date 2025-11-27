// ~/.GH/Qompass/Go/Azimuth/discord/internal/llm/rose.go
// -----------------------------------------------------
// Copyright (C) 2025 Qompass AI, All rights reserved

package llm

import (
    "bytes"
    "context"
    "encoding/json"
    "io"
    "net/http"
    "time"

    "github.com/qompassai/azimuth/internal/auth"
    "github.com/qompassai/azimuth/internal/config"
    "github.com/rs/zerolog/log"
)

type RoseClient struct {
    url string
    hc  *http.Client
}

type request struct {
    Model string `json:"model"`
    Prompt string `json:"prompt"`
    Stream bool   `json:"stream"`
}

type response struct {
    Answer string `json:"response"`
}

func NewRoseClient(cfg *config.Config) *RoseClient {
    tlsCfg := auth.MustTLS(cfg)
    return &RoseClient{
        url: cfg.RoseURL,
        hc: &http.Client{Timeout: 120 * time.Second, Transport: &http.Transport{TLSClientConfig: tlsCfg}},
    }
}

func (c *RoseClient) Complete(prompt string) (string, error) {
    body, _ := json.Marshal(request{Model: "rose", Prompt: prompt, Stream: false})
    req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, c.url+"/api/generate", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    res, err := c.hc.Do(req)
    if err != nil {
        return "", err
    }
    defer res.Body.Close()
    b, _ := io.ReadAll(res.Body)
    var r response
    if err = json.Unmarshal(b, &r); err != nil {
        return "", err
    }
    return r.Answer, nil
}

