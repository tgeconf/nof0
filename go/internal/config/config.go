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
	"gopkg.in/yaml.v3"

	"nof0-api/pkg/llm"
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
	File string         `json:",optional"`
	Raw  map[string]any `json:"-"`
}

type ManagerSection struct {
	File string         `json:",optional"`
	Raw  map[string]any `json:"-"`
}

type Config struct {
	rest.RestConf
	DataPath string          `json:",default=../../mcp/data"`
	Postgres PostgresConf    `json:",optional"`
	Redis    redis.RedisConf `json:",optional"`
	TTL      CacheTTL        `json:",optional"`

	LLM      LLMSection      `json:",optional"`
	Executor ExecutorSection `json:",optional"`
	Manager  ManagerSection  `json:",optional"`
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
	if strings.TrimSpace(c.DataPath) == "" {
		return errors.New("config: dataPath is required")
	}
	if err := c.validateTTL(); err != nil {
		return err
	}
	if err := c.validateLLM(); err != nil {
		return err
	}
	if err := c.validateFileSection("executor", c.Executor.File, c.Executor.Raw); err != nil {
		return err
	}
	if err := c.validateFileSection("manager", c.Manager.File, c.Manager.Raw); err != nil {
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

func (c *Config) validateFileSection(name, file string, data map[string]any) error {
	if file == "" {
		return nil
	}
	if len(data) == 0 {
		return fmt.Errorf("config: %s file %q loaded empty or failed to parse", name, file)
	}
	return nil
}

func (c *Config) hydrateSections(mainPath string) error {
	baseDir := filepath.Dir(mainPath)
	if err := c.loadLLM(baseDir); err != nil {
		return err
	}
	if err := c.loadYAMLSection(baseDir, &c.Executor.File, &c.Executor.Raw); err != nil {
		return fmt.Errorf("load executor config: %w", err)
	}
	if err := c.loadYAMLSection(baseDir, &c.Manager.File, &c.Manager.Raw); err != nil {
		return fmt.Errorf("load manager config: %w", err)
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
	c.LLM.Config = cfg
	return nil
}

func (c *Config) loadYAMLSection(baseDir string, file *string, target *map[string]any) error {
	if file == nil || *file == "" {
		return nil
	}
	path := resolvePath(baseDir, *file)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var out map[string]any
	if err := yaml.Unmarshal(data, &out); err != nil {
		return fmt.Errorf("unmarshal %s: %w", path, err)
	}
	*file = path
	if out == nil {
		out = make(map[string]any)
	}
	*target = out
	return nil
}

func resolvePath(baseDir, file string) string {
	file = os.ExpandEnv(file)
	if filepath.IsAbs(file) {
		return file
	}
	return filepath.Join(baseDir, file)
}
