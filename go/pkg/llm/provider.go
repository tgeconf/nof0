package llm

import "strings"

const modelSeparator = "/"

// ResolveModelID returns the fully qualified model identifier in provider/model form.
func ResolveModelID(alias string, cfg ModelConfig) string {
	model := strings.TrimSpace(alias)
	if strings.Contains(model, modelSeparator) {
		return model
	}

	name := strings.TrimSpace(cfg.ModelName)
	if name == "" {
		name = model
	}

	provider := strings.TrimSpace(cfg.Provider)
	if provider == "" || strings.Contains(name, modelSeparator) {
		return name
	}
	return provider + modelSeparator + name
}

// ParseModelID splits a fully qualified model string into provider and model name.
func ParseModelID(model string) (provider, name string) {
	parts := strings.SplitN(model, modelSeparator, 2)
	if len(parts) != 2 {
		return "", model
	}
	return parts[0], parts[1]
}
