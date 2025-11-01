package executor

import (
	"context"
	"errors"
	"time"

	"nof0-api/pkg/llm"
)

// Executor defines the decision engine interface.
type Executor interface {
	// GetFullDecision builds prompts from input context, calls LLM and returns a validated decision bundle.
	GetFullDecision(input *Context) (*FullDecision, error)
	// UpdatePerformance refreshes the cached performance view used in prompts.
	UpdatePerformance(view *PerformanceView)
	// GetConfig exposes the immutable executor configuration.
	GetConfig() *Config
}

// BasicExecutor is a minimal implementation wiring configuration, prompt rendering and the LLM client.
type BasicExecutor struct {
	cfg         *Config
	llm         llm.LLMClient
	renderer    *PromptRenderer
	performance *PerformanceView
}

// NewExecutor constructs a BasicExecutor. The templatePath is the executor prompt template provided by caller.
func NewExecutor(cfg *Config, client llm.LLMClient, templatePath string) (*BasicExecutor, error) {
	if cfg == nil {
		return nil, errors.New("executor: config is required")
	}
	if client == nil {
		return nil, errors.New("executor: llm client is required")
	}
	renderer, err := NewPromptRenderer(cfg, templatePath)
	if err != nil {
		return nil, err
	}
	return &BasicExecutor{cfg: cfg, llm: client, renderer: renderer}, nil
}

// GetConfig returns the underlying configuration.
func (e *BasicExecutor) GetConfig() *Config { return e.cfg }

// UpdatePerformance stores the latest performance snapshot.
func (e *BasicExecutor) UpdatePerformance(view *PerformanceView) { e.performance = view }

// GetFullDecision implements the end-to-end flow (MVP skeleton).
func (e *BasicExecutor) GetFullDecision(input *Context) (*FullDecision, error) {
	if e == nil || e.renderer == nil {
		return nil, errors.New("executor: not initialised")
	}
	if input == nil {
		return nil, errors.New("executor: input context is required")
	}

	// Render prompt from template with dynamic sections.
	inputs := buildPromptInputs(e.cfg, &Context{
		CurrentTime:     input.CurrentTime,
		RuntimeMinutes:  input.RuntimeMinutes,
		CallCount:       input.CallCount,
		Account:         input.Account,
		Positions:       input.Positions,
		CandidateCoins:  input.CandidateCoins,
		MarketDataMap:   input.MarketDataMap,
		OpenInterestMap: input.OpenInterestMap,
		Performance:     e.performance,
		BTCETHLeverage:  e.cfg.BTCETHLeverage,
		AltcoinLeverage: e.cfg.AltcoinLeverage,
	})

	promptStr, err := e.renderer.Render(inputs)
	if err != nil {
		return nil, err
	}

	// Phase 2: Call LLM with structured output request.
	req := &llm.ChatRequest{
		// Leave Model empty to use client's default model.
		Messages: []llm.Message{
			{Role: "system", Content: promptStr},
		},
	}

	// Use package-level contract type for structured response.
	var out decisionContract
	callCtx, cancel := context.WithTimeout(context.Background(), e.cfg.DecisionTimeout)
	defer cancel()
	_, err = e.llm.ChatStructured(callCtx, req, &out)
	if err != nil {
		return &FullDecision{UserPrompt: promptStr, CoTTrace: "", Decisions: nil, Timestamp: time.Now()}, err
	}

	// Phase 3: Map & validate.
	mapped := mapDecisionContract(out, input.Positions)
	if err := ValidateDecisions(e.cfg, input, []Decision{mapped}); err != nil {
		return &FullDecision{UserPrompt: promptStr, CoTTrace: "", Decisions: []Decision{mapped}, Timestamp: time.Now()}, err
	}

	return &FullDecision{
		UserPrompt: promptStr,
		CoTTrace:   "",
		Decisions:  []Decision{mapped},
		Timestamp:  time.Now(),
	}, nil
}

func condPerf(p *PerformanceView) *PerformanceView {
	if p != nil {
		return p
	}
	return &PerformanceView{}
}
