package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"

	"nof0-api/pkg/confkit"
	exchangepkg "nof0-api/pkg/exchange"
	executorpkg "nof0-api/pkg/executor"
	llmpkg "nof0-api/pkg/llm"
	managerpkg "nof0-api/pkg/manager"
	marketpkg "nof0-api/pkg/market"
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

type Config struct {
	rest.RestConf
	// Env indicates the running environment: test | dev | prod
	// Defaults to test. In test mode we prefer low-cost LLM routing.
	Env      string          `json:",default=test"`
	DataPath string          `json:",default=../../mcp/data"`
	Postgres PostgresConf    `json:",optional"`
	Redis    redis.RedisConf `json:",optional"`
	TTL      CacheTTL        `json:",optional"`

	LLM      confkit.Section[llmpkg.Config]      `json:",optional"`
	Executor confkit.Section[executorpkg.Config] `json:",optional"`
	Manager  confkit.Section[managerpkg.Config]  `json:",optional"`
	Exchange confkit.Section[exchangepkg.Config] `json:",optional"`
	Market   confkit.Section[marketpkg.Config]   `json:",optional"`

	mainPath string
	baseDir  string
}

func (c *Config) IsTestEnv() bool {
	return c.Env == "test" || c.Env == ""
}

func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		panic(err)
	}
	return cfg
}

func Load(path string) (*Config, error) {
	confkit.LoadDotenvOnce()

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve config path %s: %w", path, err)
	}

	var cfg Config
	if err := conf.Load(absPath, &cfg, conf.UseEnv()); err != nil {
		return nil, fmt.Errorf("load config %s: %w", absPath, err)
	}

	cfg.mainPath = absPath
	cfg.baseDir = filepath.Dir(absPath)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := cfg.hydrateSections(); err != nil {
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
	return c.validateTTL()
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

func (c *Config) hydrateSections() error {
	base := c.baseDir

	if err := c.LLM.Hydrate(base, llmpkg.LoadConfig); err != nil {
		return fmt.Errorf("load llm config: %w", err)
	}
	if err := c.Executor.Hydrate(base, executorpkg.LoadConfig); err != nil {
		return fmt.Errorf("load executor config: %w", err)
	}
	if err := c.Manager.Hydrate(base, managerpkg.LoadConfig); err != nil {
		return fmt.Errorf("load manager config: %w", err)
	}
	if err := c.Exchange.Hydrate(base, exchangepkg.LoadConfig); err != nil {
		return fmt.Errorf("load exchange config: %w", err)
	}
	if err := c.Market.Hydrate(base, marketpkg.LoadConfig); err != nil {
		return fmt.Errorf("load market config: %w", err)
	}

	return nil
}

func (c *Config) MainPath() string {
	return c.mainPath
}

func (c *Config) BaseDir() string {
	return c.baseDir
}
