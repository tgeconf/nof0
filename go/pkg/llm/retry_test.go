package llm

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/openai/openai-go"
	"github.com/stretchr/testify/require"
)

func TestNewRetryHandler(t *testing.T) {
	t.Run("with all config", func(t *testing.T) {
		cfg := RetryConfig{
			MaxRetries:     5,
			InitialBackoff: 100 * time.Millisecond,
			MaxBackoff:     2 * time.Second,
			Multiplier:     2.5,
		}
		handler := NewRetryHandler(cfg)
		require.NotNil(t, handler)
		require.Equal(t, 5, handler.cfg.MaxRetries)
		require.Equal(t, 100*time.Millisecond, handler.cfg.InitialBackoff)
		require.Equal(t, 2*time.Second, handler.cfg.MaxBackoff)
		require.Equal(t, 2.5, handler.cfg.Multiplier)
	})

	t.Run("with defaults", func(t *testing.T) {
		cfg := RetryConfig{}
		handler := NewRetryHandler(cfg)
		require.NotNil(t, handler)
		require.Equal(t, defaultInitialBackoff, handler.cfg.InitialBackoff)
		require.Equal(t, defaultMaxBackoff, handler.cfg.MaxBackoff)
		require.Equal(t, defaultBackoffFactor, handler.cfg.Multiplier)
		require.Equal(t, 0, handler.cfg.MaxRetries)
	})

	t.Run("negative values use defaults", func(t *testing.T) {
		cfg := RetryConfig{
			MaxRetries:     -1,
			InitialBackoff: -100 * time.Millisecond,
			MaxBackoff:     -2 * time.Second,
			Multiplier:     0.5,
		}
		handler := NewRetryHandler(cfg)
		require.NotNil(t, handler)
		require.Equal(t, 0, handler.cfg.MaxRetries)
		require.Equal(t, defaultInitialBackoff, handler.cfg.InitialBackoff)
		require.Equal(t, defaultMaxBackoff, handler.cfg.MaxBackoff)
		require.Equal(t, defaultBackoffFactor, handler.cfg.Multiplier)
	})
}

func TestRetryHandlerDo(t *testing.T) {
	t.Run("success on first try", func(t *testing.T) {
		handler := NewRetryHandler(RetryConfig{MaxRetries: 3})
		ctx := context.Background()

		callCount := 0
		err := handler.Do(ctx, func() error {
			callCount++
			return nil
		})

		require.NoError(t, err)
		require.Equal(t, 1, callCount)
	})

	t.Run("success on retry", func(t *testing.T) {
		handler := NewRetryHandler(RetryConfig{
			MaxRetries:     3,
			InitialBackoff: 10 * time.Millisecond,
		})
		ctx := context.Background()

		callCount := 0
		err := handler.Do(ctx, func() error {
			callCount++
			if callCount < 3 {
				return &openai.Error{StatusCode: http.StatusTooManyRequests}
			}
			return nil
		})

		require.NoError(t, err)
		require.Equal(t, 3, callCount)
	})

	t.Run("exhausted retries", func(t *testing.T) {
		handler := NewRetryHandler(RetryConfig{
			MaxRetries:     2,
			InitialBackoff: 10 * time.Millisecond,
		})
		ctx := context.Background()

		callCount := 0
		err := handler.Do(ctx, func() error {
			callCount++
			return &openai.Error{StatusCode: http.StatusTooManyRequests}
		})

		require.Error(t, err)
		require.Equal(t, 3, callCount) // initial + 2 retries
	})

	t.Run("context canceled", func(t *testing.T) {
		handler := NewRetryHandler(RetryConfig{
			MaxRetries:     3,
			InitialBackoff: 100 * time.Millisecond,
		})
		ctx, cancel := context.WithCancel(context.Background())

		callCount := 0
		err := handler.Do(ctx, func() error {
			callCount++
			if callCount == 1 {
				cancel()
			}
			return &openai.Error{StatusCode: http.StatusTooManyRequests}
		})

		require.Error(t, err)
		require.Equal(t, context.Canceled, err)
	})

	t.Run("context timeout during retry", func(t *testing.T) {
		handler := NewRetryHandler(RetryConfig{
			MaxRetries:     5,
			InitialBackoff: 50 * time.Millisecond,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		callCount := 0
		err := handler.Do(ctx, func() error {
			callCount++
			return &openai.Error{StatusCode: http.StatusTooManyRequests}
		})

		require.Error(t, err)
		require.True(t, errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled))
	})

	t.Run("non-retryable error", func(t *testing.T) {
		handler := NewRetryHandler(RetryConfig{MaxRetries: 3})
		ctx := context.Background()

		callCount := 0
		err := handler.Do(ctx, func() error {
			callCount++
			return &openai.Error{StatusCode: http.StatusBadRequest}
		})

		require.Error(t, err)
		require.Equal(t, 1, callCount) // no retries
	})
}

func TestShouldRetry(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		require.False(t, shouldRetry(nil))
	})

	t.Run("context canceled", func(t *testing.T) {
		require.False(t, shouldRetry(context.Canceled))
	})

	t.Run("context deadline exceeded", func(t *testing.T) {
		require.False(t, shouldRetry(context.DeadlineExceeded))
	})

	t.Run("openai retryable status codes", func(t *testing.T) {
		retryableCodes := []int{
			http.StatusTooManyRequests,
			http.StatusRequestTimeout,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
		}

		for _, code := range retryableCodes {
			err := &openai.Error{StatusCode: code}
			require.True(t, shouldRetry(err), "status code %d should be retryable", code)
		}
	})

	t.Run("openai non-retryable status codes", func(t *testing.T) {
		nonRetryableCodes := []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusMethodNotAllowed,
		}

		for _, code := range nonRetryableCodes {
			err := &openai.Error{StatusCode: code}
			require.False(t, shouldRetry(err), "status code %d should not be retryable", code)
		}
	})

	t.Run("temporary network error", func(t *testing.T) {
		err := &temporaryError{msg: "temporary network error"}
		require.True(t, shouldRetry(err))
	})

	t.Run("non-temporary network error", func(t *testing.T) {
		err := &nonTemporaryError{msg: "permanent network error"}
		require.False(t, shouldRetry(err))
	})

	t.Run("net.OpError is retryable", func(t *testing.T) {
		err := &net.OpError{
			Op:  "dial",
			Net: "tcp",
			Err: errors.New("connection refused"),
		}
		require.True(t, shouldRetry(err))
	})

	t.Run("generic error is not retryable", func(t *testing.T) {
		err := errors.New("generic error")
		require.False(t, shouldRetry(err))
	})

	t.Run("wrapped context canceled", func(t *testing.T) {
		wrapped := errors.Join(errors.New("wrapper"), context.Canceled)
		require.False(t, shouldRetry(wrapped))
	})

	t.Run("wrapped openai error", func(t *testing.T) {
		apiErr := &openai.Error{StatusCode: http.StatusTooManyRequests}
		wrapped := errors.Join(errors.New("wrapper"), apiErr)
		require.True(t, shouldRetry(wrapped))
	})
}

// Mock types for testing net.Error interface
type temporaryError struct {
	msg string
}

func (e *temporaryError) Error() string   { return e.msg }
func (e *temporaryError) Temporary() bool { return true }
func (e *temporaryError) Timeout() bool   { return false }

type nonTemporaryError struct {
	msg string
}

func (e *nonTemporaryError) Error() string   { return e.msg }
func (e *nonTemporaryError) Temporary() bool { return false }
func (e *nonTemporaryError) Timeout() bool   { return false }
