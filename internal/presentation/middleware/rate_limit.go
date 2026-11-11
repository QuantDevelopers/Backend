package middleware

import (
	"github.com/QuantDevelopers/Backend/bootstrap"
	"github.com/QuantDevelopers/Backend/internal/domain/exception"
	"github.com/labstack/echo/v4"
	"golang.org/x/time/rate"
)

type RateLimitMiddleware struct {
	limit   int
	limiter *rate.Limiter
}

func NewRateLimit(rateLimit *bootstrap.RateLimit) *RateLimitMiddleware {
	limit := 60
	burst := 120
	if rateLimit != nil {
		if rateLimit.Limit > 0 {
			limit = rateLimit.Limit
		}
		if rateLimit.Burst > 0 {
			burst = rateLimit.Burst
		}
	}
	return &RateLimitMiddleware{
		limit:   limit,
		limiter: rate.NewLimiter(rate.Limit(limit), burst),
	}
}

func (rl *RateLimitMiddleware) RateLimit(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		if !rl.limiter.Allow() {
			panic(exception.NewRequestRateLimitError("Rate limit exceeded.", rl.limit, nil))
		}
		return next(ctx)
	}
}
