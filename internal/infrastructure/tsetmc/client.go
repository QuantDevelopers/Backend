package tsetmc

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	domainlogger "github.com/QuantDevelopers/Backend/internal/domain/logger"
	domaintsetmc "github.com/QuantDevelopers/Backend/internal/domain/tsetmc"
	"golang.org/x/time/rate"
)

// ClientOptions configures rate limiting, retries, timeouts, and circuit breaking.
type ClientOptions struct {
	RateLimitRPS         float64
	Burst                int
	MaxRetries           int
	Timeout              time.Duration
	BackoffBase          time.Duration
	BackoffFactor        float64
	MaxJitter            time.Duration
	MaxConsecutiveErrors int
	HTTPClient           *http.Client
}

// HTTPClient is a concurrency-safe, polite TSETMC HTTP client with token-bucket
// rate limiting, exponential backoff + jitter, and a consecutive-error circuit breaker.
type HTTPClient struct {
	httpClient           *http.Client
	limiter              *rate.Limiter
	maxRetries           int
	backoffBase          time.Duration
	backoffFactor        float64
	maxJitter            time.Duration
	maxConsecutiveErrors int
	logger               domainlogger.Logger

	mu                sync.Mutex
	consecutiveErrors int
}

// NewHTTPClient creates an HTTPClient with explicit options and sensible defaults.
func NewHTTPClient(opts ClientOptions, log domainlogger.Logger) *HTTPClient {
	rps := opts.RateLimitRPS
	if rps <= 0 {
		rps = 1.25
	}
	burst := opts.Burst
	if burst <= 0 {
		burst = 1
	}
	maxRetries := opts.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	backoffBase := opts.BackoffBase
	if backoffBase <= 0 {
		backoffBase = 1 * time.Second
	}
	backoffFactor := opts.BackoffFactor
	if backoffFactor <= 0 {
		backoffFactor = 2.0
	}
	maxConsecutiveErrors := opts.MaxConsecutiveErrors
	if maxConsecutiveErrors <= 0 {
		maxConsecutiveErrors = 5
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}

	return &HTTPClient{
		httpClient:           httpClient,
		limiter:              rate.NewLimiter(rate.Limit(rps), burst),
		maxRetries:           maxRetries,
		backoffBase:          backoffBase,
		backoffFactor:        backoffFactor,
		maxJitter:            opts.MaxJitter,
		maxConsecutiveErrors: maxConsecutiveErrors,
		logger:               log,
	}
}

// NewHTTPClientFromConfig adapts bootstrap.Crawler settings into HTTPClient.
func NewHTTPClientFromConfig(cfg *bootstrap.Crawler, log domainlogger.Logger) *HTTPClient {
	var opts ClientOptions
	if cfg != nil {
		opts.RateLimitRPS = cfg.RateLimitRPS
		opts.MaxRetries = cfg.MaxRetries
		opts.BackoffFactor = cfg.BackoffFactor
	}
	opts.MaxJitter = 500 * time.Millisecond
	return NewHTTPClient(opts, log)
}

func (c *HTTPClient) ConsecutiveErrors() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.consecutiveErrors
}

func (c *HTTPClient) ResetCircuitBreaker() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.consecutiveErrors = 0
}

func (c *HTTPClient) recordSuccess() {
	c.mu.Lock()
	c.consecutiveErrors = 0
	c.mu.Unlock()
}

func (c *HTTPClient) recordError() int {
	c.mu.Lock()
	c.consecutiveErrors++
	count := c.consecutiveErrors
	c.mu.Unlock()
	return count
}

func (c *HTTPClient) Get(ctx context.Context, targetURL string) ([]byte, int, error) {
	if c.ConsecutiveErrors() >= c.maxConsecutiveErrors {
		return nil, 0, domaintsetmc.ErrCircuitBreakerOpen
	}

	var lastStatus int
	var lastErr error

	for attempt := 1; attempt <= c.maxRetries; attempt++ {
		if err := c.limiter.Wait(ctx); err != nil {
			return nil, 0, err
		}

		body, status, err := c.doSingleAttempt(ctx, targetURL)
		if err == nil && status == http.StatusOK {
			c.recordSuccess()
			return body, status, nil
		}

		lastStatus = status
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("tsetmc HTTP %d for %s", status, targetURL)
		}

		// Do not retry non-429 4xx client errors.
		if err == nil && status >= 400 && status < 500 && status != http.StatusTooManyRequests {
			return body, status, lastErr
		}

		errCount := c.recordError()
		if errCount >= c.maxConsecutiveErrors {
			if c.logger != nil {
				c.logger.Warn("tsetmc circuit breaker tripped",
					domainlogger.Int("consecutive_errors", errCount),
					domainlogger.String("url", targetURL),
				)
			}
			return nil, lastStatus, fmt.Errorf("%w: %v", domaintsetmc.ErrCircuitBreakerOpen, lastErr)
		}

		if attempt < c.maxRetries {
			delay := c.computeBackoff(attempt)
			if c.logger != nil {
				c.logger.Warn("retrying tsetmc request",
					domainlogger.Int("attempt", attempt),
					domainlogger.Int("max_retries", c.maxRetries),
					domainlogger.Int("status_code", status),
					domainlogger.String("url", targetURL),
				)
			}
			if err := sleepWithContext(ctx, delay); err != nil {
				return nil, lastStatus, err
			}
		}
	}

	return nil, lastStatus, fmt.Errorf("failed fetching %s after %d attempts: %w", targetURL, c.maxRetries, lastErr)
}

func (c *HTTPClient) doSingleAttempt(ctx context.Context, targetURL string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "fa-IR,fa;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Referer", "https://tsetmc.com/")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, resp.StatusCode, readErr
	}
	return body, resp.StatusCode, nil
}

func (c *HTTPClient) computeBackoff(attempt int) time.Duration {
	exp := math.Pow(c.backoffFactor, float64(attempt-1))
	delay := time.Duration(float64(c.backoffBase) * exp)
	if c.maxJitter > 0 {
		delay += time.Duration(rand.Int64N(int64(c.maxJitter)))
	}
	return delay
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
