//go:build integration

package llm

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

// TestMain loads .env so ZENMUX_API_KEY can be injected easily in local/CI.
func TestMain(m *testing.M) {
	// Walk up from this file to find repo root and load .env
	if _, file, _, ok := runtime.Caller(0); ok {
		dir := filepath.Dir(file)
		for i := 0; i < 10; i++ {
			_ = godotenv.Load(filepath.Join(dir, ".env"))
			if exists(filepath.Join(dir, "go.mod")) || exists(filepath.Join(dir, ".git")) {
				break
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	} else {
		_ = godotenv.Load(".env")
	}
	os.Exit(m.Run())
}

func exists(p string) bool { _, err := os.Stat(p); return err == nil }

// newIntegrationClient builds a client targeting Zenmux with auto-routing.
func newIntegrationClient(t *testing.T) *Client {
	t.Helper()

	apiKey := os.Getenv("ZENMUX_API_KEY")
	if apiKey == "" {
		t.Skip("ZENMUX_API_KEY not set; skipping integration test")
	}
	baseURL := os.Getenv("ZENMUX_BASE_URL")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	cfg := &Config{
		BaseURL:      baseURL,
		APIKey:       apiKey,
		DefaultModel: "zenmux/auto",
		Timeout:      15 * time.Second,
		MaxRetries:   2,
		LogLevel:     "error",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

// TestIntegration_Chat_AutoRouting_Basic performs a minimal routed chat call.
func TestIntegration_Chat_AutoRouting_Basic(t *testing.T) {
	client := newIntegrationClient(t)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, &ChatRequest{
		Model: "zenmux/auto",
		Messages: []Message{
			{Role: "user", Content: "Say a short hello."},
		},
	})
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	if resp == nil || len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		t.Fatalf("unexpected empty response: %#v", resp)
	}
}
