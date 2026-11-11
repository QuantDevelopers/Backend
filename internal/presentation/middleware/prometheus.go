package middleware

import (
	"strconv"
	"time"

	"github.com/QuantDevelopers/Backend/internal/domain/metrics"
	"github.com/labstack/echo/v4"
)

type PrometheusMiddleware struct {
	metricsClient metrics.MetricsClient
}

func NewPrometheusMiddleware(metricsClient metrics.MetricsClient) *PrometheusMiddleware {
	return &PrometheusMiddleware{
		metricsClient: metricsClient,
	}
}

func (pm *PrometheusMiddleware) Prometheus(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		start := time.Now()

		err := next(ctx)

		duration := time.Since(start).Seconds()
		status := ctx.Response().Status
		route := ctx.Path()
		if route == "" {
			route = ctx.Request().URL.Path
		}
		method := ctx.Request().Method

		pm.metricsClient.IncHTTPRequest(method, route, strconv.Itoa(status))
		pm.metricsClient.ObserveHTTPRequestDuration(method, route, duration)

		return err
	}
}
