package manager

import (
	"fmt"

	"nof0-api/pkg/llm"
)

// ManagerPromptInputs encapsulates the data required to render a manager prompt.
type ManagerPromptInputs struct {
	Trader      *TraderConfig
	ContextJSON string
}

// PromptRenderer renders manager prompt templates for a specific trader.
type PromptRenderer struct {
	template *llm.PromptTemplate
}

// NewPromptRenderer parses the template at the provided path.
func NewPromptRenderer(path string) (*PromptRenderer, error) {
	tpl, err := llm.NewPromptTemplate(path, nil)
	if err != nil {
		return nil, err
	}
	return &PromptRenderer{template: tpl}, nil
}

// Render executes the template using the supplied inputs.
func (r *PromptRenderer) Render(inputs ManagerPromptInputs) (string, error) {
	if r == nil || r.template == nil {
		return "", fmt.Errorf("manager prompt renderer not initialised")
	}
	if inputs.Trader == nil {
		return "", fmt.Errorf("manager prompt renderer requires trader data")
	}
	return r.template.Render(inputs)
}

// Digest exposes the template digest for version tracking.
func (r *PromptRenderer) Digest() string {
	if r == nil || r.template == nil {
		return ""
	}
	return r.template.Digest()
}
