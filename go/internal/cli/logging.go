package cli

import (
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"nof0-api/internal/config"
	"nof0-api/pkg/confkit"
)

// ConfigSummaryLines returns human readable lines describing the loaded app config.
func ConfigSummaryLines(cfg *config.Config) []string {
	if cfg == nil {
		return []string{"Configuration: <nil>"}
	}

	lines := []string{
		fmt.Sprintf("Environment: %s", cfg.Env),
		fmt.Sprintf("Data path: %s", cfg.DataPath),
		fmt.Sprintf("Postgres: %s", presence(cfg.Postgres.DSN != "")),
		fmt.Sprintf("Redis: %s", presence(strings.TrimSpace(cfg.Redis.Host) != "")),
		fmt.Sprintf("TTL (short/medium/long): %ds / %ds / %ds", cfg.TTL.Short, cfg.TTL.Medium, cfg.TTL.Long),
		sectionLine("LLM config", cfg.LLM),
		sectionLine("Executor config", cfg.Executor),
		sectionLine("Manager config", cfg.Manager),
		sectionLine("Exchange config", cfg.Exchange),
		sectionLine("Market config", cfg.Market),
	}

	return lines
}

// LogConfigSummary emits the configuration summary using logx.
func LogConfigSummary(cfg *config.Config) {
	lines := ConfigSummaryLines(cfg)
	if len(lines) == 0 {
		return
	}
	logx.Info("configuration summary")
	for _, line := range lines {
		logx.Infof("config • %s", line)
	}
}

func presence(ok bool) string {
	if ok {
		return "configured"
	}
	return "not configured"
}

func sectionLine[T any](name string, section confkit.Section[T]) string {
	switch {
	case strings.TrimSpace(section.File) != "":
		return fmt.Sprintf("%s: %s", name, section.File)
	case section.Value != nil:
		return fmt.Sprintf("%s: inline", name)
	default:
		return fmt.Sprintf("%s: not configured", name)
	}
}
