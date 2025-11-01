package llm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseModelID(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		expectedProvider string
		expectedModel    string
	}{
		{
			name:             "full model ID with provider",
			input:            "openai/gpt-4",
			expectedProvider: "openai",
			expectedModel:    "gpt-4",
		},
		{
			name:             "model without provider",
			input:            "gpt-4",
			expectedProvider: "",
			expectedModel:    "gpt-4",
		},
		{
			name:             "anthropic model",
			input:            "anthropic/claude-3-opus",
			expectedProvider: "anthropic",
			expectedModel:    "claude-3-opus",
		},
		{
			name:             "google model",
			input:            "google/gemini-pro",
			expectedProvider: "google",
			expectedModel:    "gemini-pro",
		},
		{
			name:             "model with slash in name",
			input:            "provider/model/version",
			expectedProvider: "provider",
			expectedModel:    "model/version",
		},
		{
			name:             "empty string",
			input:            "",
			expectedProvider: "",
			expectedModel:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, model := ParseModelID(tt.input)
			require.Equal(t, tt.expectedProvider, provider)
			require.Equal(t, tt.expectedModel, model)
		})
	}
}

func TestResolveModelID(t *testing.T) {
	tests := []struct {
		name     string
		alias    string
		cfg      ModelConfig
		expected string
	}{
		{
			name:  "alias already contains provider",
			alias: "openai/gpt-4",
			cfg: ModelConfig{
				Provider:  "anthropic",
				ModelName: "claude-3",
			},
			expected: "openai/gpt-4",
		},
		{
			name:  "use provider from config",
			alias: "gpt-4",
			cfg: ModelConfig{
				Provider:  "openai",
				ModelName: "gpt-4-turbo",
			},
			expected: "openai/gpt-4-turbo",
		},
		{
			name:  "model name empty, use alias",
			alias: "gpt-4",
			cfg: ModelConfig{
				Provider:  "openai",
				ModelName: "",
			},
			expected: "openai/gpt-4",
		},
		{
			name:  "provider empty, use model name only",
			alias: "gpt-4",
			cfg: ModelConfig{
				Provider:  "",
				ModelName: "gpt-4-turbo",
			},
			expected: "gpt-4-turbo",
		},
		{
			name:  "model name contains separator",
			alias: "claude",
			cfg: ModelConfig{
				Provider:  "anthropic",
				ModelName: "anthropic/claude-3-opus",
			},
			expected: "anthropic/claude-3-opus",
		},
		{
			name:  "alias with whitespace",
			alias: "  gpt-4  ",
			cfg: ModelConfig{
				Provider:  "openai",
				ModelName: "gpt-4-turbo",
			},
			expected: "openai/gpt-4-turbo",
		},
		{
			name:     "empty config",
			alias:    "gpt-4",
			cfg:      ModelConfig{},
			expected: "gpt-4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveModelID(tt.alias, tt.cfg)
			require.Equal(t, tt.expected, result)
		})
	}
}
