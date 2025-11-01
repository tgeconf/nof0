package exchange

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Config captures configuration for one or more exchange providers.
type Config struct {
	Default   string                     `yaml:"default"`
	Providers map[string]*ProviderConfig `yaml:"providers"`
}

// ProviderConfig describes how to construct a specific exchange provider instance.
type ProviderConfig struct {
	Type         string `yaml:"type"`
	PrivateKey   string `yaml:"private_key"`
	APIKey       string `yaml:"api_key"`
	APISecret    string `yaml:"api_secret"`
	Passphrase   string `yaml:"passphrase"`
	VaultAddress string `yaml:"vault_address"`
	MainAddress  string `yaml:"main_address"` // Main account address (for API wallet scenarios)
	Testnet      bool   `yaml:"testnet"`

	TimeoutRaw string        `yaml:"timeout"`
	Timeout    time.Duration `yaml:"-"`
}

// ProviderBuilder constructs a Provider from configuration.
type ProviderBuilder func(name string, cfg *ProviderConfig) (Provider, error)

var (
	providerRegistry   = make(map[string]ProviderBuilder)
	providerRegistryMu sync.RWMutex
)

// RegisterProvider associates a builder with an exchange provider type.
func RegisterProvider(typeName string, builder ProviderBuilder) {
	providerRegistryMu.Lock()
	defer providerRegistryMu.Unlock()
	providerRegistry[strings.ToLower(strings.TrimSpace(typeName))] = builder
}

func lookupProviderBuilder(typeName string) (ProviderBuilder, bool) {
	providerRegistryMu.RLock()
	defer providerRegistryMu.RUnlock()
	builder, ok := providerRegistry[strings.ToLower(strings.TrimSpace(typeName))]
	return builder, ok
}

// GetProvider constructs a single provider instance for the given type using
// the provided configuration. This is a convenience for tests and callers that
// want to instantiate a provider without building a full config map.
func GetProvider(typeName string, cfg *ProviderConfig) (Provider, error) {
	if cfg == nil {
		cfg = &ProviderConfig{}
	}
	// Ensure the type is set and valid for validation.
	cfgCopy := *cfg
	cfgCopy.Type = typeName
	if err := cfgCopy.validate("inline"); err != nil {
		return nil, err
	}
	builder, ok := lookupProviderBuilder(cfgCopy.Type)
	if !ok {
		return nil, fmt.Errorf("exchange provider: unsupported type %q", cfgCopy.Type)
	}
	return builder("inline", &cfgCopy)
}

// LoadConfig reads configuration from disk.
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open exchange config: %w", err)
	}
	defer file.Close()
	return LoadConfigFromReader(file)
}

// LoadConfigFromReader constructs a Config from an io.Reader.
func LoadConfigFromReader(r io.Reader) (*Config, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read exchange config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal exchange config: %w", err)
	}
	if err := cfg.normalise(); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) normalise() error {
	if c.Providers == nil {
		c.Providers = make(map[string]*ProviderConfig)
	}
	for name, provider := range c.Providers {
		if provider == nil {
			provider = &ProviderConfig{}
			c.Providers[name] = provider
		}
		provider.expandEnv()
		if err := provider.parseDurations(name); err != nil {
			return err
		}
	}
	return nil
}

func (p *ProviderConfig) expandEnv() {
	p.Type = strings.TrimSpace(os.ExpandEnv(p.Type))
	p.PrivateKey = strings.TrimSpace(os.ExpandEnv(p.PrivateKey))
	p.APIKey = strings.TrimSpace(os.ExpandEnv(p.APIKey))
	p.APISecret = strings.TrimSpace(os.ExpandEnv(p.APISecret))
	p.Passphrase = strings.TrimSpace(os.ExpandEnv(p.Passphrase))
	p.VaultAddress = strings.TrimSpace(os.ExpandEnv(p.VaultAddress))
	p.MainAddress = strings.TrimSpace(os.ExpandEnv(p.MainAddress))
	p.TimeoutRaw = strings.TrimSpace(os.ExpandEnv(p.TimeoutRaw))
}

func (p *ProviderConfig) parseDurations(name string) error {
	if p.TimeoutRaw == "" {
		p.Timeout = 0
		return nil
	}
	d, err := time.ParseDuration(p.TimeoutRaw)
	if err != nil {
		return fmt.Errorf("exchange provider %s: invalid timeout %q: %w", name, p.TimeoutRaw, err)
	}
	if d <= 0 {
		return fmt.Errorf("exchange provider %s: timeout must be positive, got %s", name, d)
	}
	p.Timeout = d
	return nil
}

// Validate ensures all providers have sane configuration.
func (c *Config) Validate() error {
	if len(c.Providers) == 0 {
		return fmt.Errorf("exchange config: providers cannot be empty")
	}
	if c.Default != "" {
		if _, ok := c.Providers[c.Default]; !ok {
			return fmt.Errorf("exchange config: default provider %q not defined", c.Default)
		}
	}

	for name, provider := range c.Providers {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("exchange config: provider name cannot be empty")
		}
		if err := provider.validate(name); err != nil {
			return err
		}
	}
	return nil
}

func (p *ProviderConfig) validate(name string) error {
	if p == nil {
		return fmt.Errorf("exchange config: provider %s is nil", name)
	}
	if strings.TrimSpace(p.Type) == "" {
		return fmt.Errorf("exchange config: provider %s must specify type", name)
	}

	if _, ok := lookupProviderBuilder(p.Type); !ok {
		return fmt.Errorf("exchange config: provider %s has unsupported type %q", name, p.Type)
	}

	if strings.ToLower(p.Type) == "hyperliquid" && p.PrivateKey == "" {
		return fmt.Errorf("exchange config: provider %s requires private_key", name)
	}
	return nil
}

// BuildProviders instantiates exchange providers according to the configuration.
func (c *Config) BuildProviders() (map[string]Provider, error) {
	result := make(map[string]Provider, len(c.Providers))
	for name, providerCfg := range c.Providers {
		builder, ok := lookupProviderBuilder(providerCfg.Type)
		if !ok {
			return nil, fmt.Errorf("exchange provider %s: unsupported type %q", name, providerCfg.Type)
		}
		provider, err := builder(name, providerCfg)
		if err != nil {
			return nil, fmt.Errorf("exchange provider %s: %w", name, err)
		}
		result[name] = provider
	}
	return result, nil
}
