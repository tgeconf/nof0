package executor

import (
	"context"
	"path/filepath"
	"testing"

	"nof0-api/pkg/llm"
)

// fakeLLM returns a fixed structured decision matching the contract.
type fakeLLM struct{}

func (f *fakeLLM) Chat(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	return nil, nil
}
func (f *fakeLLM) ChatStream(ctx context.Context, req *llm.ChatRequest) (<-chan llm.StreamResponse, error) {
	return nil, nil
}

func (f *fakeLLM) ChatStructured(_ context.Context, _ *llm.ChatRequest, target interface{}) (interface{}, error) {
	// Fill target via llm.ParseStructured-compatible JSON
	jsonStr := `{
      "signal":"buy_to_enter",
      "symbol":"BTC",
      "leverage":5,
      "position_size_usd":200,
      "entry_price":100,
      "stop_loss":95,
      "take_profit":115,
      "risk_usd":10,
      "confidence":90,
      "invalidation_condition":"below EMA20",
      "reasoning":"clear uptrend"
    }`
	_ = llm.ParseStructured(jsonStr, target)
	return nil, nil
}

func (f *fakeLLM) GetConfig() *llm.Config { return &llm.Config{} }
func (f *fakeLLM) Close() error           { return nil }

func TestExecutor_GetFullDecision(t *testing.T) {
	cfg := &Config{
		BTCETHLeverage:         20,
		AltcoinLeverage:        10,
		MinConfidence:          75,
		MinRiskReward:          3.0,
		MaxPositions:           4,
		DecisionIntervalRaw:    "3m",
		DecisionTimeoutRaw:     "60s",
		MaxConcurrentDecisions: 1,
	}
	client := &fakeLLM{}
	templatePath := filepath.Join("..", "..", "etc", "prompts", "executor", "default_prompt.tmpl")

	exec, err := NewExecutor(cfg, client, templatePath)
	if err != nil {
		t.Fatalf("NewExecutor error: %v", err)
	}
	ctx := &Context{CurrentTime: "2025-01-01T00:00:00Z"}
	out, err := exec.GetFullDecision(ctx)
	if err != nil {
		t.Fatalf("GetFullDecision error: %v", err)
	}
	if out == nil || len(out.Decisions) != 1 {
		t.Fatalf("unexpected decisions: %+v", out)
	}
	d := out.Decisions[0]
	if d.Action != "open_long" || d.Symbol != "BTC" || d.Confidence < 75 {
		t.Fatalf("mapped decision invalid: %+v", d)
	}
	if out.UserPrompt == "" {
		t.Fatal("expected UserPrompt to be populated")
	}
}
