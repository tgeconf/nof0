package llm

import (
	"context"
	"errors"
	"math"
	"net"
	"net/http"
	"time"

	"github.com/openai/openai-go"
)

const (
	defaultInitialBackoff = 200 * time.Millisecond
	defaultMaxBackoff     = 3 * time.Second
	defaultBackoffFactor  = 2.0
)

// RetryConfig encapsulates exponential backoff settings.
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Multiplier     float64
}

// RetryHandler executes retryable operations with backoff.
type RetryHandler struct {
	cfg RetryConfig
}

// NewRetryHandler constructs a handler with sane defaults.
func NewRetryHandler(cfg RetryConfig) *RetryHandler {
	if cfg.InitialBackoff <= 0 {
		cfg.InitialBackoff = defaultInitialBackoff
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = defaultMaxBackoff
	}
	if cfg.Multiplier <= 1 {
		cfg.Multiplier = defaultBackoffFactor
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	return &RetryHandler{cfg: cfg}
}

// Do executes fn with retries until it succeeds or exhausts attempts.
func (r *RetryHandler) Do(ctx context.Context, fn func() error) error {
	var attempt int
	backoff := r.cfg.InitialBackoff

	for {
		err := fn()
		if err == nil {
			return nil
		}

		if !shouldRetry(err) || attempt >= r.cfg.MaxRetries {
			return err
		}
		attempt++

		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}

		backoff = time.Duration(math.Min(
			float64(r.cfg.MaxBackoff),
			float64(backoff)*r.cfg.Multiplier,
		))
	}
}

func shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	switch {
	case errors.Is(err, context.Canceled),
		errors.Is(err, context.DeadlineExceeded):
		return false
	}

	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusTooManyRequests,
			http.StatusRequestTimeout,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout:
			return true
		default:
			return false
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Temporary() {
		return true
	}

	// Treat unknown transport errors as retryable to be safe.
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	return false
}
