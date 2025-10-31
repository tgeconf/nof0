package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

// Fields represents structured logging fields.
type Fields map[string]interface{}

// Logger wraps logging behaviour used by the client.
type Logger interface {
	Debug(ctx context.Context, msg string, fields Fields)
	Info(ctx context.Context, msg string, fields Fields)
	Warn(ctx context.Context, msg string, fields Fields)
	Error(ctx context.Context, err error, fields Fields)
}

type logxLogger struct{}

// NewLogger returns a Logger backed by go-zero's logx.
func NewLogger(level string) Logger {
	logx.SetLevel(parseLevel(level))
	return &logxLogger{}
}

func (l *logxLogger) Debug(ctx context.Context, msg string, fields Fields) {
	logx.WithContext(ctx).Debug(msgWithFields(msg, fields))
}

func (l *logxLogger) Info(ctx context.Context, msg string, fields Fields) {
	logx.WithContext(ctx).Info(msgWithFields(msg, fields))
}

func (l *logxLogger) Warn(ctx context.Context, msg string, fields Fields) {
	logx.WithContext(ctx).Slow(msgWithFields(msg, fields))
}

func (l *logxLogger) Error(ctx context.Context, err error, fields Fields) {
	payload := msgWithFields(err.Error(), fields)
	logx.WithContext(ctx).Error(payload)
}

func parseLevel(level string) uint32 {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return logx.DebugLevel
	case "info":
		return logx.InfoLevel
	case "error":
		return logx.ErrorLevel
	case "severe", "fatal":
		return logx.SevereLevel
	default:
		return logx.InfoLevel
	}
}

func msgWithFields(msg string, fields Fields) string {
	if len(fields) == 0 {
		return msg
	}

	parts := make([]string, 0, len(fields))
	for k, v := range fields {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return fmt.Sprintf("%s | %s", msg, strings.Join(parts, " "))
}
