// ~/.GH/Qompass/Go/Azimuth/discord/main.go
// ----------------------------------------
// Copyright (C) 2025 Qompass AI, All rights reserved

package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

const roseEndpoint = "http://localhost:11434/api/generate"

type RoseRequest struct {
	Model string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

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

	err = dg.Open()
	if err != nil {
		log.Fatalf("error opening connection: %v", err)
	}
	log.Println("Bot is running. Press CTRL-C to exit.")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-stop

	dg.Close()
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!ask") {
		prompt := strings.TrimPrefix(m.Content, "!ask ")
		resp, err := queryRose(prompt)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "❌ Error querying Rose: "+err.Error())
			return
		}
		s.ChannelMessageSend(m.ChannelID, resp)
	}
}

func queryRose(prompt string) (string, error) {
	reqBody := RoseRequest{
		Model:  "rose", //
		Prompt: prompt,
		Stream: false, // 
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(roseEndpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var roseResp RoseResponse
	err = json.NewDecoder(resp.Body).Decode(&roseResp)
	if err != nil {
		return "", err
	}
	return roseResp.Response, nil
}

