// ~/.GH/Qompass/Go/Azimuth/discord/cmd/bot/main.go
// ------------------------------------------------
// Copyright (C) 2025 Qompass AI, All rights reserved

package main

import (
    "context"
    "encoding/json"
    "os"
    "os/signal"
    "syscall"

    "github.com/bwmarrin/discordgo"
    "github.com/qompassai/azimuth/internal/bus"
    "github.com/qompassai/azimuth/internal/config"
    "github.com/qompassai/azimuth/internal/model"
    "github.com/rs/zerolog/log"
)

func main() {
    cfg := config.MustLoad()

    dg, err := discordgo.New("Bot " + cfg.DiscordToken)
    if err != nil {
        log.Fatal().Err(err).Msg("create discord session")
    }
    dg.Identify.Intents = discordgo.IntentGuildMessages

    // Connect to NATS JetStream
    js := bus.MustJetStream(cfg)

    // Discord → NATS
    dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
        if m.Author.Bot {
            return
        }
        msg := model.NewDiscordMessage(m)
        if data, err := json.Marshal(msg); err == nil {
            js.PublishAsync("msg.in.discord", data)
        }
    })

    // NATS → Discord
    bus.MustSubscribe(js, "msg.out.discord", func(data []byte) {
        var r model.Response
        if err := json.Unmarshal(data, &r); err == nil {
            dg.ChannelMessageSend(r.ChannelID, r.Content)
        }
    })

    if err = dg.Open(); err != nil {
        log.Fatal().Err(err).Msg("discord open")
    }
    defer dg.Close()

    // graceful shutdown
    sig := make(chan os.Signal, 1)
    signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
    <-sig
    dg.Close()
}
