package executor

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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
		MajorCoinLeverage:      20,
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
	assert.NoError(t, err, "NewExecutor should not error")
	assert.NotNil(t, exec, "executor should not be nil")

	ctx := &Context{CurrentTime: "2025-01-01T00:00:00Z"}
	out, err := exec.GetFullDecision(ctx)
	assert.NoError(t, err, "GetFullDecision should not error")
	assert.NotNil(t, out, "decision output should not be nil")
	assert.Len(t, out.Decisions, 1, "should have exactly one decision")

	d := out.Decisions[0]
	assert.Equal(t, "open_long", d.Action, "action should be open_long")
	assert.Equal(t, "BTC", d.Symbol, "symbol should be BTC")
	assert.GreaterOrEqual(t, d.Confidence, 75, "confidence should be >= 75")
	assert.NotEmpty(t, out.UserPrompt, "UserPrompt should be populated")
}
