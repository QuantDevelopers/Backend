package tsetmc

import (
	"context"
	"errors"
)

// ErrCircuitBreakerOpen indicates the crawler halted after too many consecutive upstream errors.
var ErrCircuitBreakerOpen = errors.New("tsetmc circuit breaker open: consecutive errors threshold exceeded")

// Client defines a polite, rate-limited HTTP client for TSETMC CDN requests.
type Client interface {
	Get(ctx context.Context, url string) ([]byte, int, error)
	ConsecutiveErrors() int
	ResetCircuitBreaker()
}
