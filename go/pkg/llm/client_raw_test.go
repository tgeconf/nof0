package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientChat_ZenmuxAuto_PreservesFields_And_Retries(t *testing.T) {
	var callCount int32
	var captured map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		c := atomic.AddInt32(&callCount, 1)
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)

		if c == 1 {
			// Force a retry on first attempt
			http.Error(w, "temporary error", http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "id":"chatcmpl-auto-1",
            "object":"chat.completion",
            "created":1730366400,
            "model":"zenmux/auto",
            "choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}],
            "usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}
        }`))
	}))
	defer server.Close()

	cfg := &Config{
		BaseURL:      server.URL,
		APIKey:       "test-key",
		DefaultModel: "zenmux/auto",
		Timeout:      2 * time.Second,
		MaxRetries:   1, // one retry after the initial failure
		LogLevel:     "error",
	}

	client, err := NewClient(cfg, WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req := &ChatRequest{
		Model:          "zenmux/auto",
		Routing:        &RoutingConfig{AvailableModels: []string{"openai/gpt-5-nano"}, Preference: "balanced"},
		ResponseFormat: &ResponseFormat{Type: "json_object"},
		Messages: []Message{
			{Role: "system", Name: "sysA", Content: "policy"},
			{Role: "tool", ToolCallID: "call-1", Content: "tool-output"},
			{Role: "function", Name: "fnA", Content: "{}"},
			{Role: "user", Name: "u1", Content: "hi"},
		},
	}

	_, err = client.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}

	if got := atomic.LoadInt32(&callCount); got != 2 {
		t.Fatalf("expected 2 calls (1 retry), got %d", got)
	}

	// Validate preserved fields
	msgs, _ := captured["messages"].([]any)
	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(msgs))
	}

	// system name
	m0 := msgs[0].(map[string]any)
	if m0["role"] != "system" || m0["name"] != "sysA" {
		t.Fatalf("system name not preserved: %#v", m0)
	}

	// tool tool_call_id
	m1 := msgs[1].(map[string]any)
	if m1["role"] != "tool" || m1["tool_call_id"] != "call-1" {
		t.Fatalf("tool_call_id not preserved: %#v", m1)
	}

	// function name
	m2 := msgs[2].(map[string]any)
	if m2["role"] != "function" || m2["name"] != "fnA" {
		t.Fatalf("function name not preserved: %#v", m2)
	}

	// json_object format included
	rf := captured["response_format"].(map[string]any)
	if rf["type"] != "json_object" {
		t.Fatalf("response_format not json_object: %#v", rf)
	}

	// routing present
	routing := captured["model_routing_config"].(map[string]any)
	if routing == nil {
		t.Fatalf("missing model_routing_config")
	}
}

func TestClientChat_ZenmuxAuto_JSONSchemaFormat(t *testing.T) {
	var captured map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &captured)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
            "id":"chatcmpl-auto-2",
            "object":"chat.completion",
            "created":1730366400,
            "model":"zenmux/auto",
            "choices":[{"index":0,"finish_reason":"stop","message":{"role":"assistant","content":"ok"}}],
            "usage":{"prompt_tokens":3,"completion_tokens":2,"total_tokens":5}
        }`))
	}))
	defer server.Close()

	cfg := &Config{
		BaseURL:      server.URL,
		APIKey:       "test-key",
		DefaultModel: "zenmux/auto",
		Timeout:      2 * time.Second,
		MaxRetries:   0,
		LogLevel:     "error",
	}

	client, err := NewClient(cfg, WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	strict := true
	req := &ChatRequest{
		Model:   "zenmux/auto",
		Routing: &RoutingConfig{AvailableModels: []string{"openai/gpt-5-nano"}},
		ResponseFormat: &ResponseFormat{
			Type:        "json_schema",
			Name:        "Decision",
			Description: "test schema",
			Schema:      map[string]any{"type": "object", "properties": map[string]any{"ok": map[string]any{"type": "boolean"}}},
			Strict:      &strict,
		},
		Messages: []Message{{Role: "user", Content: "hi"}},
	}

	if _, err := client.Chat(ctx, req); err != nil {
		t.Fatalf("Chat error: %v", err)
	}

	rf := captured["response_format"].(map[string]any)
	if rf["type"] != "json_schema" {
		t.Fatalf("expected json_schema, got %#v", rf)
	}
	js := rf["json_schema"].(map[string]any)
	if js["name"] != "Decision" || js["description"] != "test schema" || js["strict"] != true {
		t.Fatalf("json_schema fields not preserved: %#v", js)
	}
}
