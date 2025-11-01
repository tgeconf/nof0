package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
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

	LLM      struct{ File string } `json:",optional"`
	Executor struct{ File string } `json:",optional"`
	Manager  struct{ File string } `json:",optional"`
	Exchange struct{ File string } `json:",optional"`
	Market   struct{ File string } `json:",optional"`
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
	var cfg Config
	if err := conf.Load(path, &cfg, conf.UseEnv()); err != nil {
		return nil, fmt.Errorf("load config %s: %w", path, err)
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
