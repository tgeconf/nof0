//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"nof0-api/pkg/llm"
)

type TradeDecision struct {
	Action     string  `json:"action"`
	Symbol     string  `json:"symbol"`
	Confidence float64 `json:"confidence"`
	Reasoning  string  `json:"reasoning"`
}

func main() {
	cfg, err := llm.LoadConfig("../../etc/llm.yaml")
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	client, err := llm.NewClient(cfg)
	if err != nil {
		log.Fatalf("create client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &llm.ChatRequest{
		Model: "gpt-5",
		Messages: []llm.Message{
			{Role: "system", Content: "You are an AI crypto trading assistant."},
			{Role: "user", Content: "BTC is trading at $68k with strong momentum. Recommend an action."},
		},
	}

	var decision TradeDecision
	if _, err := client.ChatStructured(ctx, req, &decision); err != nil {
		log.Fatalf("structured chat failed: %v", err)
	}

	fmt.Printf("Action: %s (%.2f) -> %s\n", decision.Action, decision.Confidence, decision.Reasoning)
}
