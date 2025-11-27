// ~/.GH/Qompass/Go/Azimuth/discord/cmd/llmworker/main.go
// ------------------------------------------------------
// Copyright (C) 2025 Qompass AI, All rights reserved

package main

import (
    "encoding/json"
    "github.com/qompassai/azimuth/internal/bus"
    "github.com/qompassai/azimuth/internal/config"
    "github.com/qompassai/azimuth/internal/llm"
    "github.com/qompassai/azimuth/internal/model"
    "github.com/rs/zerolog/log"
)

func main() {
    cfg := config.MustLoad()
    js := bus.MustJetStream(cfg)
    client := llm.NewRoseClient(cfg)

    bus.MustSubscribe(js, "msg.in.*", func(data []byte) {
        var msg model.Message
        if err := json.Unmarshal(data, &msg); err != nil {
            log.Error().Err(err).Msg("json unmarshal")
            return
        }
        respText, err := client.Complete(msg.Content)
        if err != nil {
            log.Error().Err(err).Msg("rose complete")
            return
        }
        out := model.Response{MessageID: msg.ID, ChannelID: msg.ChannelID, Content: respText}
        if b, _ := json.Marshal(out); b != nil {
            js.PublishAsync("msg.out."+msg.Kind.String(), b)
        }
    })

    select {} // block forever
}

