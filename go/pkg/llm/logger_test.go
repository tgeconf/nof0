package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name  string
		level string
	}{
		{"debug level", "debug"},
		{"info level", "info"},
		{"warn level", "warn"},
		{"error level", "error"},
		{"invalid level defaults to info", "invalid"},
		{"empty level defaults to info", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger(tt.level)
			require.NotNil(t, logger)
			// Verify logger implements the Logger interface
			require.Implements(t, (*Logger)(nil), logger)
		})
	}
}

func TestLoggerMethods(t *testing.T) {
	// Test that logger methods can be called without panicking
	logger := NewLogger("debug")
	ctx := context.Background()

	t.Run("Debug", func(t *testing.T) {
		require.NotPanics(t, func() {
			logger.Debug(ctx, "debug message", Fields{"key": "value"})
		})
	})

	t.Run("Info", func(t *testing.T) {
		require.NotPanics(t, func() {
			logger.Info(ctx, "info message", Fields{"key": "value"})
		})
	})

	t.Run("Warn", func(t *testing.T) {
		require.NotPanics(t, func() {
			logger.Warn(ctx, "warning message", Fields{"key": "value"})
		})
	})

	t.Run("Error", func(t *testing.T) {
		require.NotPanics(t, func() {
			err := &testError{msg: "test error"}
			logger.Error(ctx, err, Fields{"key": "value"})
		})
	})

	t.Run("with nil fields", func(t *testing.T) {
		require.NotPanics(t, func() {
			logger.Info(ctx, "message", nil)
		})
	})

	t.Run("with empty fields", func(t *testing.T) {
		require.NotPanics(t, func() {
			logger.Info(ctx, "message", Fields{})
		})
	})
}

func TestParseLevel(t *testing.T) {
	// Test that parseLevel returns consistent values
	t.Run("debug level", func(t *testing.T) {
		level1 := parseLevel("debug")
		level2 := parseLevel("DEBUG")
		require.Equal(t, level1, level2, "case insensitive")
	})

	t.Run("info level", func(t *testing.T) {
		level1 := parseLevel("info")
		level2 := parseLevel("INFO")
		require.Equal(t, level1, level2, "case insensitive")
	})

	t.Run("error level", func(t *testing.T) {
		level := parseLevel("error")
		require.NotZero(t, level)
	})

	t.Run("severe level", func(t *testing.T) {
		level1 := parseLevel("severe")
		level2 := parseLevel("fatal")
		require.Equal(t, level1, level2, "severe and fatal should map to same level")
	})

	t.Run("default level", func(t *testing.T) {
		infoLevel := parseLevel("info")
		defaultLevel := parseLevel("invalid")
		emptyLevel := parseLevel("")
		require.Equal(t, infoLevel, defaultLevel, "invalid should default to info")
		require.Equal(t, infoLevel, emptyLevel, "empty should default to info")
	})

	t.Run("whitespace handling", func(t *testing.T) {
		debugLevel := parseLevel("debug")
		withSpaces := parseLevel("  debug  ")
		require.Equal(t, debugLevel, withSpaces, "should trim whitespace")
	})
}

func TestMsgWithFields(t *testing.T) {
	t.Run("nil fields", func(t *testing.T) {
		result := msgWithFields("test message", nil)
		require.Equal(t, "test message", result)
	})

	t.Run("empty fields", func(t *testing.T) {
		result := msgWithFields("test message", Fields{})
		require.Equal(t, "test message", result)
	})

	t.Run("single field", func(t *testing.T) {
		result := msgWithFields("test message", Fields{"key": "value"})
		require.Contains(t, result, "test message")
		require.Contains(t, result, "key=value")
	})

	t.Run("multiple fields", func(t *testing.T) {
		result := msgWithFields("test message", Fields{
			"key1": "value1",
			"key2": 42,
			"key3": true,
		})
		require.Contains(t, result, "test message")
		require.Contains(t, result, "key1=value1")
		require.Contains(t, result, "key2=42")
		require.Contains(t, result, "key3=true")
	})

	t.Run("fields with special characters", func(t *testing.T) {
		result := msgWithFields("test", Fields{
			"path":  "/api/v1/chat",
			"error": "connection refused",
		})
		require.Contains(t, result, "test")
		require.True(t, strings.Contains(result, "path") || strings.Contains(result, "error"))
	})
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
