package executor

import (
	"path/filepath"
	"testing"
	"time"

	"context"
	"nof0-api/pkg/llm"
)

type slowLLM struct{}

func (s *slowLLM) Chat(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func (s *slowLLM) ChatStream(ctx context.Context, req *llm.ChatRequest) (<-chan llm.StreamResponse, error) {
	ch := make(chan llm.StreamResponse)
	go func() { <-ctx.Done(); close(ch) }()
	return ch, ctx.Err()
}
func (s *slowLLM) ChatStructured(ctx context.Context, req *llm.ChatRequest, target interface{}) (interface{}, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func (s *slowLLM) GetConfig() *llm.Config { return &llm.Config{} }
func (s *slowLLM) Close() error           { return nil }

func TestExecutor_TimeoutHonored(t *testing.T) {
	cfg := &Config{
		BTCETHLeverage:         20,
		AltcoinLeverage:        10,
		MinConfidence:          75,
		MinRiskReward:          3.0,
		MaxPositions:           4,
		DecisionIntervalRaw:    "3m",
		DecisionTimeoutRaw:     "20ms",
		MaxConcurrentDecisions: 1,
	}
	if err := cfg.parseDurations(); err != nil {
		t.Fatalf("parseDurations: %v", err)
	}

	client := &slowLLM{}
	templatePath := filepath.Join("..", "..", "etc", "prompts", "executor", "default_prompt.tmpl")
	exec, err := NewExecutor(cfg, client, templatePath)
	if err != nil {
		t.Fatalf("NewExecutor error: %v", err)
	}
	start := time.Now()
	_, err = exec.GetFullDecision(&Context{CurrentTime: "2025-01-01T00:00:00Z"})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if time.Since(start) < 15*time.Millisecond {
		t.Fatal("timeout not enforced with sufficient delay")
	}
}
