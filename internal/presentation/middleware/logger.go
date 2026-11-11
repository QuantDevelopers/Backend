package middleware

import (
	"time"

	"github.com/QuantDevelopers/Backend/internal/domain/logger"
	"github.com/labstack/echo/v4"
)

type LoggerMiddleware struct {
	logger logger.Logger
}

func NewLoggerMiddleware(logger logger.Logger) *LoggerMiddleware {
	return &LoggerMiddleware{
		logger: logger,
	}
}

func (lm *LoggerMiddleware) EchoLoggerMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		start := time.Now()
		path := ctx.Request().URL.Path
		query := ctx.Request().URL.RawQuery

		if path == "/metrics" {
			return next(ctx)
		}

		err := next(ctx)
		latency := time.Since(start)

		if err != nil {
			lm.logger.Error(
				"Request Error",
				logger.Int("status", ctx.Response().Status),
				logger.String("method", ctx.Request().Method),
				logger.String("path", path),
				logger.String("query", query),
				logger.String("ip", ctx.RealIP()),
				logger.Duration("latency", latency),
				logger.String("user-agent", ctx.Request().UserAgent()),
				logger.Error("error", err),
			)
		} else {
			lm.logger.Info(
				"Request",
				logger.Int("status", ctx.Response().Status),
				logger.String("method", ctx.Request().Method),
				logger.String("path", path),
				logger.String("query", query),
				logger.String("ip", ctx.RealIP()),
				logger.Duration("latency", latency),
				logger.String("user-agent", ctx.Request().UserAgent()),
			)
		}

		return err
	}
}
