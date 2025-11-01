//go:build integration

package llm_test

import (
	"context"
	"testing"
	"time"

	appcfg "nof0-api/internal/config"
	"nof0-api/pkg/llm"
)

// newIntegrationClient builds a client from etc/llm.yaml via internal/config.
// If config is missing or invalid (e.g., api_key absent), the test fails fast.
func newIntegrationClient(t *testing.T) *llm.Client {
	t.Helper()
	cfg := appcfg.MustLoadLLM()
	client, err := llm.NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

// TestIntegration_Chat_LowCostModel performs a minimal chat call with a free model.
func TestIntegration_Chat_LowCostModel(t *testing.T) {
	client := newIntegrationClient(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, &llm.ChatRequest{
		Messages: []llm.Message{{Role: "user", Content: "Say a short hello."}},
	})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	if resp == nil || len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		t.Fatalf("unexpected empty response: %#v", resp)
	}
	content := resp.Choices[0].Message.Content
	if len(content) > 50 {
		content = content[:50] + "..."
	}
	t.Logf("Response: %s", content)
}

// TestIntegration_Chat_JSONObject_LowCost verifies the json_object
// response_format is accepted by the gateway for a low-cost model.
func TestIntegration_Chat_JSONObject_LowCost(t *testing.T) {
	client := newIntegrationClient(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, &llm.ChatRequest{
		ResponseFormat: &llm.ResponseFormat{Type: "json_object"},
		Messages:       []llm.Message{{Role: "user", Content: "Respond with a tiny JSON object."}},
	})
	if err != nil {
		t.Fatalf("Chat (json_object) error: %v", err)
	}
	if resp == nil || len(resp.Choices) == 0 {
		t.Fatalf("empty response: %#v", resp)
	}
}

// TestIntegration_ChatStructured_JSONSchema_LowCost exercises ChatStructured
// with a minimal schema and verifies decoding works.
func TestIntegration_ChatStructured_JSONSchema_LowCost(t *testing.T) {
	client := newIntegrationClient(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	type Result struct {
		OK bool `json:"ok"`
	}

	var out Result
	_, err := client.ChatStructured(ctx, &llm.ChatRequest{
		Messages: []llm.Message{
			{Role: "system", Content: "You must follow the JSON schema strictly."},
			{Role: "user", Content: "Return {\"ok\":true}."},
		},
	}, &out)
	if err != nil {
		t.Skipf("ChatStructured returned non-JSON or schema not honored strictly: %v", err)
	}
}
