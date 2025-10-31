//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"nof0-api/pkg/llm"
)

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

	resp, err := client.Chat(ctx, &llm.ChatRequest{
		Model: "gpt-5",
		Messages: []llm.Message{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Summarise the latest BTC price action."},
		},
	})
	if err != nil {
		log.Fatalf("chat failed: %v", err)
	}

	fmt.Println(resp.Choices[0].Message.Content)
}
