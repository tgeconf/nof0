package llm

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultBaseURL    = "https://zenmux.ai/api/v1"
	defaultTimeout    = 60 * time.Second
	defaultMaxRetries = 3
	defaultLogLevel   = "info"

	envAPIKey       = "ZENMUX_API_KEY"
	envBaseURL      = "ZENMUX_BASE_URL"
	envDefaultModel = "ZENMUX_DEFAULT_MODEL"
	envTimeout      = "ZENMUX_TIMEOUT"
	envMaxRetries   = "ZENMUX_MAX_RETRIES"
)

// Config holds runtime settings for the LLM client.
type Config struct {
	BaseURL      string                 `yaml:"base_url"`
	APIKey       string                 `yaml:"api_key"`
	DefaultModel string                 `yaml:"default_model"`
	Timeout      time.Duration          `yaml:"-"`
	MaxRetries   int                    `yaml:"max_retries"`
	LogLevel     string                 `yaml:"log_level"`
	Models       map[string]ModelConfig `yaml:"models"`
	// Optional defaults for Zenmux auto-routing
	RoutingDefaults *RoutingConfig `yaml:"routing_defaults,omitempty"`

	timeoutRaw string `yaml:"timeout"`
}

// ModelConfig defines defaults for a particular model alias.
type ModelConfig struct {
	Provider    string   `yaml:"provider"`
	ModelName   string   `yaml:"model_name"`
	Temperature *float64 `yaml:"temperature,omitempty"`
	MaxTokens   *int     `yaml:"max_tokens,omitempty"`
	TopP        *float64 `yaml:"top_p,omitempty"`
}

// LoadConfig reads configuration from disk.
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open llm config: %w", err)
	}
	defer file.Close()
	return LoadConfigFromReader(file)
}

// LoadConfigFromReader constructs a Config from a reader.
func LoadConfigFromReader(r io.Reader) (*Config, error) {
	var raw struct {
		BaseURL         string                 `yaml:"base_url"`
		APIKey          string                 `yaml:"api_key"`
		DefaultModel    string                 `yaml:"default_model"`
		Timeout         string                 `yaml:"timeout"`
		MaxRetries      int                    `yaml:"max_retries"`
		LogLevel        string                 `yaml:"log_level"`
		Models          map[string]ModelConfig `yaml:"models"`
		RoutingDefaults *RoutingConfig         `yaml:"routing_defaults"`
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read llm config: %w", err)
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal llm config: %w", err)
	}

	cfg := &Config{
		BaseURL:         raw.BaseURL,
		APIKey:          raw.APIKey,
		DefaultModel:    raw.DefaultModel,
		MaxRetries:      raw.MaxRetries,
		LogLevel:        raw.LogLevel,
		Models:          raw.Models,
		RoutingDefaults: raw.RoutingDefaults,
		timeoutRaw:      raw.Timeout,
	}

	cfg.applyDefaults()
	cfg.applyEnvOverrides()
	if err := cfg.parseTimeout(); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Validate checks that required configuration is present.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.APIKey) == "" {
		return errors.New("llm config: api_key is required")
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		return errors.New("llm config: base_url is required")
	}
	if strings.TrimSpace(c.DefaultModel) == "" {
		return errors.New("llm config: default_model is required")
	}
	if c.Timeout <= 0 {
		return errors.New("llm config: timeout must be positive")
	}
	if c.MaxRetries < 0 {
		return errors.New("llm config: max_retries cannot be negative")
	}
	return nil
}

// Model returns the configuration for the given model alias.
func (c *Config) Model(name string) (ModelConfig, bool) {
	if c.Models == nil {
		return ModelConfig{}, false
	}
	modelCfg, ok := c.Models[name]
	return modelCfg, ok
}

// Clone returns a shallow copy of the configuration.
func (c *Config) Clone() *Config {
	if c == nil {
		return nil
	}
	cp := *c
	cp.timeoutRaw = c.timeoutRaw
	if c.Models != nil {
		cp.Models = make(map[string]ModelConfig, len(c.Models))
		for k, v := range c.Models {
			cp.Models[k] = v
		}
	}
	return &cp
}

func (c *Config) applyDefaults() {
	if strings.TrimSpace(c.BaseURL) == "" {
		c.BaseURL = defaultBaseURL
	}
	if strings.TrimSpace(c.LogLevel) == "" {
		c.LogLevel = defaultLogLevel
	}
	if c.MaxRetries <= 0 {
		c.MaxRetries = defaultMaxRetries
	}
}

func (c *Config) applyEnvOverrides() {
	c.BaseURL = expandAndOverride(c.BaseURL, envBaseURL)
	c.APIKey = expandAndOverride(c.APIKey, envAPIKey)
	c.DefaultModel = expandAndOverride(c.DefaultModel, envDefaultModel)

	if raw := os.Getenv(envTimeout); raw != "" {
		c.timeoutRaw = raw
	} else {
		c.timeoutRaw = os.ExpandEnv(c.timeoutRaw)
	}

	if raw := os.Getenv(envMaxRetries); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			c.MaxRetries = v
		}
	}
}

func (c *Config) parseTimeout() error {
	if strings.TrimSpace(c.timeoutRaw) == "" {
		c.Timeout = defaultTimeout
		return nil
	}

	d, err := time.ParseDuration(c.timeoutRaw)
	if err != nil {
		return fmt.Errorf("llm config: invalid timeout %q: %w", c.timeoutRaw, err)
	}
	if d <= 0 {
		return fmt.Errorf("llm config: timeout must be positive, got %s", d)
	}
	c.Timeout = d
	return nil
}

func expandAndOverride(current, envKey string) string {
	current = os.ExpandEnv(current)
	if envVal := os.Getenv(envKey); envVal != "" {
		return envVal
	}
	return current
}

// Note: test-mode routing now controlled by top-level Config.Env in internal/config.
