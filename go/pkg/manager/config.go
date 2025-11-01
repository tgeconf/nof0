package manager

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config defines the overall manager configuration schema.
type Config struct {
	Manager    ManagerConfig             `yaml:"manager"`
	Traders    []TraderConfig            `yaml:"traders"`
	Exchanges  map[string]ExchangeConfig `yaml:"exchanges"`
	Monitoring MonitoringConfig          `yaml:"monitoring"`

	baseDir string
}

type ManagerConfig struct {
	TotalEquityUSD      float64       `yaml:"total_equity_usd"`
	ReserveEquityPct    float64       `yaml:"reserve_equity_pct"`
	AllocationStrategy  string        `yaml:"allocation_strategy"`
	RebalanceInterval   time.Duration `yaml:"-"`
	StateStorageBackend string        `yaml:"state_storage_backend"`
	StateStoragePath    string        `yaml:"state_storage_path"`

	RebalanceIntervalRaw string `yaml:"rebalance_interval"`
}

type TraderConfig struct {
	ID               string         `yaml:"id"`
	Name             string         `yaml:"name"`
	Exchange         string         `yaml:"exchange"`
	PromptTemplate   string         `yaml:"prompt_template"`
	DecisionInterval time.Duration  `yaml:"-"`
	RiskParams       RiskParameters `yaml:"risk_params"`
	AllocationPct    float64        `yaml:"allocation_pct"`
	AutoStart        bool           `yaml:"auto_start"`

	DecisionIntervalRaw string `yaml:"decision_interval"`
}

type RiskParameters struct {
	MaxPositions       int     `yaml:"max_positions"`
	MaxPositionSizeUSD float64 `yaml:"max_position_size_usd"`
	MaxMarginUsagePct  float64 `yaml:"max_margin_usage_pct"`
	BTCETHLeverage     int     `yaml:"btc_eth_leverage"`
	AltcoinLeverage    int     `yaml:"altcoin_leverage"`
	MinRiskRewardRatio float64 `yaml:"min_risk_reward_ratio"`
	MinConfidence      int     `yaml:"min_confidence"`
	StopLossEnabled    bool    `yaml:"stop_loss_enabled"`
	TakeProfitEnabled  bool    `yaml:"take_profit_enabled"`
}

type ExchangeConfig struct {
	Type       string        `yaml:"type"`
	APIKey     string        `yaml:"api_key"`
	Secret     string        `yaml:"api_secret"`
	Passphrase string        `yaml:"passphrase"`
	Testnet    bool          `yaml:"testnet"`
	Timeout    time.Duration `yaml:"-"`

	TimeoutRaw string `yaml:"timeout"`
}

type MonitoringConfig struct {
	UpdateInterval  time.Duration `yaml:"-"`
	AlertWebhook    string        `yaml:"alert_webhook"`
	MetricsExporter string        `yaml:"metrics_exporter"`

	UpdateIntervalRaw string `yaml:"update_interval"`
}

// LoadConfig reads configuration from disk.
func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open manager config: %w", err)
	}
	defer file.Close()
	return LoadConfigFromReader(file, filepath.Dir(path))
}

// LoadConfigFromReader constructs a Config from a reader with the provided base directory.
func LoadConfigFromReader(r io.Reader, baseDir string) (*Config, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read manager config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal manager config: %w", err)
	}
	cfg.baseDir = baseDir

	cfg.applyDefaults()
	if err := cfg.parseDurations(); err != nil {
		return nil, err
	}
	cfg.expandFields()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if strings.TrimSpace(c.Manager.RebalanceIntervalRaw) == "" {
		c.Manager.RebalanceIntervalRaw = "1h"
	}
	for i := range c.Traders {
		if strings.TrimSpace(c.Traders[i].DecisionIntervalRaw) == "" {
			c.Traders[i].DecisionIntervalRaw = "3m"
		}
	}
	if strings.TrimSpace(c.Monitoring.UpdateIntervalRaw) == "" {
		c.Monitoring.UpdateIntervalRaw = "30s"
	}
	for key, ex := range c.Exchanges {
		if strings.TrimSpace(ex.TimeoutRaw) == "" {
			ex.TimeoutRaw = "30s"
			c.Exchanges[key] = ex
		}
	}
}

func (c *Config) parseDurations() error {
	var err error
	c.Manager.RebalanceInterval, err = parsePositiveDuration("manager.rebalance_interval", c.Manager.RebalanceIntervalRaw)
	if err != nil {
		return err
	}
	for i := range c.Traders {
		d, err := parsePositiveDuration(fmt.Sprintf("traders[%d].decision_interval", i), c.Traders[i].DecisionIntervalRaw)
		if err != nil {
			return err
		}
		c.Traders[i].DecisionInterval = d
	}
	c.Monitoring.UpdateInterval, err = parsePositiveDuration("monitoring.update_interval", c.Monitoring.UpdateIntervalRaw)
	if err != nil {
		return err
	}
	for key, ex := range c.Exchanges {
		d, err := parsePositiveDuration(fmt.Sprintf("exchanges.%s.timeout", key), ex.TimeoutRaw)
		if err != nil {
			return err
		}
		ex.Timeout = d
		c.Exchanges[key] = ex
	}
	return nil
}

func (c *Config) expandFields() {
	c.Manager.StateStoragePath = c.resolvePath(c.Manager.StateStoragePath)
	c.Manager.AllocationStrategy = strings.TrimSpace(c.Manager.AllocationStrategy)
	c.Manager.StateStorageBackend = strings.TrimSpace(c.Manager.StateStorageBackend)
	for i := range c.Traders {
		c.Traders[i].ID = strings.TrimSpace(c.Traders[i].ID)
		c.Traders[i].Name = strings.TrimSpace(c.Traders[i].Name)
		c.Traders[i].Exchange = strings.TrimSpace(c.Traders[i].Exchange)
		c.Traders[i].PromptTemplate = c.resolvePath(c.Traders[i].PromptTemplate)
	}
	for key, ex := range c.Exchanges {
		ex.Type = strings.TrimSpace(ex.Type)
		ex.APIKey = strings.TrimSpace(os.ExpandEnv(ex.APIKey))
		ex.Secret = strings.TrimSpace(os.ExpandEnv(ex.Secret))
		ex.Passphrase = strings.TrimSpace(os.ExpandEnv(ex.Passphrase))
		c.Exchanges[key] = ex
	}
	c.Monitoring.AlertWebhook = strings.TrimSpace(os.ExpandEnv(c.Monitoring.AlertWebhook))
	c.Monitoring.MetricsExporter = strings.TrimSpace(c.Monitoring.MetricsExporter)
}

func (c *Config) resolvePath(path string) string {
	path = strings.TrimSpace(os.ExpandEnv(path))
	if path == "" || filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(c.baseDir, path)
}

// Validate ensures configuration sanity.
func (c *Config) Validate() error {
	if c.Manager.TotalEquityUSD < 0 {
		return errors.New("manager config: manager.total_equity_usd cannot be negative")
	}
	if c.Manager.ReserveEquityPct < 0 || c.Manager.ReserveEquityPct > 100 {
		return errors.New("manager config: manager.reserve_equity_pct must be between 0 and 100")
	}
	if strings.TrimSpace(c.Manager.StateStorageBackend) == "" {
		return errors.New("manager config: manager.state_storage_backend is required")
	}
	if strings.TrimSpace(c.Manager.StateStoragePath) == "" {
		return errors.New("manager config: manager.state_storage_path is required")
	}
	if len(c.Traders) == 0 {
		return errors.New("manager config: at least one trader must be defined")
	}

	idSeen := make(map[string]struct{}, len(c.Traders))
	totalAllocation := 0.0
	for i, trader := range c.Traders {
		if trader.ID == "" {
			return fmt.Errorf("manager config: traders[%d].id is required", i)
		}
		if _, ok := idSeen[trader.ID]; ok {
			return fmt.Errorf("manager config: duplicate trader id %q", trader.ID)
		}
		idSeen[trader.ID] = struct{}{}
		if trader.Name == "" {
			return fmt.Errorf("manager config: traders[%d].name is required", i)
		}
		if trader.Exchange == "" {
			return fmt.Errorf("manager config: traders[%d].exchange is required", i)
		}
		if _, ok := c.Exchanges[trader.Exchange]; !ok {
			return fmt.Errorf("manager config: traders[%d] references undefined exchange %q", i, trader.Exchange)
		}
		if trader.PromptTemplate == "" {
			return fmt.Errorf("manager config: traders[%d].prompt_template is required", i)
		}
		if _, err := os.Stat(trader.PromptTemplate); err != nil {
			return fmt.Errorf("manager config: traders[%d].prompt_template %q not accessible: %w", i, trader.PromptTemplate, err)
		}
		if trader.AllocationPct < 0 {
			return fmt.Errorf("manager config: traders[%d].allocation_pct cannot be negative", i)
		}
		totalAllocation += trader.AllocationPct
		if err := trader.RiskParams.Validate(i); err != nil {
			return err
		}
	}
	if totalAllocation > 100+1e-6 {
		return fmt.Errorf("manager config: trader allocation sum %.2f exceeds 100", totalAllocation)
	}
	if totalAllocation > 100-c.Manager.ReserveEquityPct+1e-6 {
		return fmt.Errorf("manager config: trader allocation %.2f exceeds available equity after reserve %.2f", totalAllocation, c.Manager.ReserveEquityPct)
	}

	if len(c.Exchanges) == 0 {
		return errors.New("manager config: exchanges section cannot be empty")
	}
	for key, ex := range c.Exchanges {
		if ex.Type == "" {
			return fmt.Errorf("manager config: exchanges.%s.type is required", key)
		}
	}

	if c.Monitoring.MetricsExporter == "" {
		return errors.New("manager config: monitoring.metrics_exporter is required")
	}
	return nil
}

// Validate ensures risk parameters are within expected ranges.
func (r RiskParameters) Validate(index int) error {
	if r.MaxPositions <= 0 {
		return fmt.Errorf("manager config: traders[%d].risk_params.max_positions must be positive", index)
	}
	if r.MaxPositionSizeUSD <= 0 {
		return fmt.Errorf("manager config: traders[%d].risk_params.max_position_size_usd must be positive", index)
	}
	if r.MaxMarginUsagePct < 0 || r.MaxMarginUsagePct > 100 {
		return fmt.Errorf("manager config: traders[%d].risk_params.max_margin_usage_pct must be between 0 and 100", index)
	}
	if r.BTCETHLeverage <= 0 {
		return fmt.Errorf("manager config: traders[%d].risk_params.btc_eth_leverage must be positive", index)
	}
	if r.AltcoinLeverage <= 0 {
		return fmt.Errorf("manager config: traders[%d].risk_params.altcoin_leverage must be positive", index)
	}
	if r.MinRiskRewardRatio <= 0 {
		return fmt.Errorf("manager config: traders[%d].risk_params.min_risk_reward_ratio must be positive", index)
	}
	if r.MinConfidence < 0 || r.MinConfidence > 100 {
		return fmt.Errorf("manager config: traders[%d].risk_params.min_confidence must be between 0 and 100", index)
	}
	return nil
}

func parsePositiveDuration(field, value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("manager config: %s is required", field)
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("manager config: invalid %s %q: %w", field, value, err)
	}
	if d <= 0 {
		return 0, fmt.Errorf("manager config: %s must be positive, got %s", field, d)
	}
	return d, nil
}

// TraderIDs returns a stable ordered list of trader IDs.
func (c *Config) TraderIDs() []string {
	ids := make([]string, 0, len(c.Traders))
	for _, t := range c.Traders {
		ids = append(ids, t.ID)
	}
	sort.Strings(ids)
	return ids
}
