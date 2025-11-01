package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"

	"nof0-api/pkg/exchange"
	_ "nof0-api/pkg/exchange/hyperliquid"
	"nof0-api/pkg/executor"
	"nof0-api/pkg/llm"
	"nof0-api/pkg/manager"
	"nof0-api/pkg/market"
	_ "nof0-api/pkg/market/exchanges/hyperliquid"
)

type PostgresConf struct {
	// DSN example: postgres://user:pass@localhost:5432/nof0?sslmode=disable
	DSN     string `json:",optional"`
	MaxOpen int    `json:",default=10"`
	MaxIdle int    `json:",default=5"`
}

type CacheTTL struct {
	Short  int `json:",default=10"` // seconds
	Medium int `json:",default=60"`
	Long   int `json:",default=300"`
}

type LLMSection struct {
	File   string      `json:",optional"`
	Config *llm.Config `json:"-"`
}

type ExecutorSection struct {
	File   string           `json:",optional"`
	Config *executor.Config `json:"-"`
}

type ManagerSection struct {
	File   string          `json:",optional"`
	Config *manager.Config `json:"-"`
}

type ExchangeSection struct {
	File   string           `json:",optional"`
	Config *exchange.Config `json:"-"`
}

type MarketSection struct {
	File   string         `json:",optional"`
	Config *market.Config `json:"-"`
}

type Config struct {
	rest.RestConf
	// Env indicates the running environment: test | dev | prod
	// Defaults to test. In test mode we prefer low-cost LLM routing.
	Env      string          `json:",default=test"`
	DataPath string          `json:",default=../../mcp/data"`
	Postgres PostgresConf    `json:",optional"`
	Redis    redis.RedisConf `json:",optional"`
	TTL      CacheTTL        `json:",optional"`

	LLM      LLMSection      `json:",optional"`
	Executor ExecutorSection `json:",optional"`
	Manager  ManagerSection  `json:",optional"`
	Exchange ExchangeSection `json:",optional"`
	Market   MarketSection   `json:",optional"`
}

func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		panic(err)
	}
	return cfg
}

func Load(path string) (*Config, error) {
	var cfg Config
	if err := conf.Load(path, &cfg, conf.UseEnv()); err != nil {
		return nil, fmt.Errorf("load config %s: %w", path, err)
	}
	if err := cfg.hydrateSections(path); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	switch strings.ToLower(strings.TrimSpace(c.Env)) {
	case "", "test", "dev", "prod":
		if strings.TrimSpace(c.Env) == "" {
			c.Env = "test"
		}
	default:
		return errors.New("config: env must be one of test|dev|prod")
	}
	if strings.TrimSpace(c.DataPath) == "" {
		return errors.New("config: dataPath is required")
	}
	if err := c.validateTTL(); err != nil {
		return err
	}
	if err := c.validateLLM(); err != nil {
		return err
	}
	if err := c.validateExecutor(); err != nil {
		return err
	}
	if err := c.validateManager(); err != nil {
		return err
	}
	if err := c.validateExchange(); err != nil {
		return err
	}
	if err := c.validateMarket(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateTTL() error {
	if c.TTL.Short <= 0 {
		return errors.New("config: ttl.short must be positive")
	}
	if c.TTL.Medium <= 0 {
		return errors.New("config: ttl.medium must be positive")
	}
	if c.TTL.Long <= 0 {
		return errors.New("config: ttl.long must be positive")
	}
	return nil
}

func (c *Config) validateLLM() error {
	if c.LLM.File == "" {
		return nil
	}
	if c.LLM.Config == nil {
		return fmt.Errorf("config: llm file %q loaded without config", c.LLM.File)
	}
	return nil
}

func (c *Config) validateExecutor() error {
	if c.Executor.File == "" {
		return nil
	}
	if c.Executor.Config == nil {
		return fmt.Errorf("config: executor file %q loaded without config", c.Executor.File)
	}
	return nil
}

func (c *Config) validateManager() error {
	if c.Manager.File == "" {
		return nil
	}
	if c.Manager.Config == nil {
		return fmt.Errorf("config: manager file %q loaded without config", c.Manager.File)
	}
	return nil
}

func (c *Config) validateExchange() error {
	if c.Exchange.File == "" {
		return nil
	}
	if c.Exchange.Config == nil {
		return fmt.Errorf("config: exchange file %q loaded without config", c.Exchange.File)
	}
	return nil
}

func (c *Config) validateMarket() error {
	if c.Market.File == "" {
		return nil
	}
	if c.Market.Config == nil {
		return fmt.Errorf("config: market file %q loaded without config", c.Market.File)
	}
	return nil
}

func (c *Config) hydrateSections(mainPath string) error {
	baseDir := filepath.Dir(mainPath)
	if err := c.loadLLM(baseDir); err != nil {
		return err
	}
	if err := c.loadExecutor(baseDir); err != nil {
		return err
	}
	if err := c.loadManager(baseDir); err != nil {
		return err
	}
	if err := c.loadExchange(baseDir); err != nil {
		return err
	}
	if err := c.loadMarket(baseDir); err != nil {
		return err
	}
	return nil
}

func (c *Config) loadLLM(baseDir string) error {
	if c.LLM.File == "" {
		return nil
	}
	path := resolvePath(baseDir, c.LLM.File)
	cfg, err := llm.LoadConfig(path)
	if err != nil {
		return fmt.Errorf("load llm config %s: %w", path, err)
	}
	c.LLM.File = path
	// If running in test environment, force zenmux/auto by default to enable
	// low-cost routing (can be further refined by llm routing_defaults).
	if strings.EqualFold(c.Env, "test") {
		cfg.DefaultModel = "zenmux/auto"
	}
	c.LLM.Config = cfg
	return nil
}

func (c *Config) loadExecutor(baseDir string) error {
	if c.Executor.File == "" {
		return nil
	}
	path := resolvePath(baseDir, c.Executor.File)
	cfg, err := executor.LoadConfig(path)
	if err != nil {
		return fmt.Errorf("load executor config %s: %w", path, err)
	}
	c.Executor.File = path
	c.Executor.Config = cfg
	return nil
}

func (c *Config) loadManager(baseDir string) error {
	if c.Manager.File == "" {
		return nil
	}
	path := resolvePath(baseDir, c.Manager.File)
	cfg, err := manager.LoadConfig(path)
	if err != nil {
		return fmt.Errorf("load manager config %s: %w", path, err)
	}
	c.Manager.File = path
	c.Manager.Config = cfg
	return nil
}

func (c *Config) loadExchange(baseDir string) error {
	if c.Exchange.File == "" {
		return nil
	}
	path := resolvePath(baseDir, c.Exchange.File)
	cfg, err := exchange.LoadConfig(path)
	if err != nil {
		return fmt.Errorf("load exchange config %s: %w", path, err)
	}
	c.Exchange.File = path
	c.Exchange.Config = cfg
	return nil
}

func (c *Config) loadMarket(baseDir string) error {
	if c.Market.File == "" {
		return nil
	}
	path := resolvePath(baseDir, c.Market.File)
	cfg, err := market.LoadConfig(path)
	if err != nil {
		return fmt.Errorf("load market config %s: %w", path, err)
	}
	c.Market.File = path
	c.Market.Config = cfg
	return nil
}

func resolvePath(baseDir, file string) string {
	file = os.ExpandEnv(file)
	if filepath.IsAbs(file) {
		return file
	}
	return filepath.Join(baseDir, file)
}
