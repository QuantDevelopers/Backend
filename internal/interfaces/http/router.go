package http

import (
	"log/slog"
	stdhttp "net/http"

	"github.com/QuantDevelopers/Backend/internal/config"
	"github.com/QuantDevelopers/Backend/internal/interfaces/http/handler"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// RouterDeps bundles handlers and dependencies required to construct the HTTP router.
type RouterDeps struct {
	HealthChecker     handler.DBHealthChecker
	UnderlyingUseCase handler.UnderlyingUseCase
}

// NewRouter creates and configures an Echo HTTP server matching the REST API v1 contract.
func NewRouter(cfg config.Settings, logger *slog.Logger, deps RouterDeps) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())

	if logger != nil {
		e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
			LogStatus:  true,
			LogURI:     true,
			LogMethod:  true,
			LogLatency: true,
			LogValuesFunc: func(_ echo.Context, v middleware.RequestLoggerValues) error {
				logger.Info("http_request",
					"method", v.Method,
					"uri", v.URI,
					"status", v.Status,
					"latency_ms", v.Latency.Milliseconds(),
				)
				return nil
			},
		}))
	}

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     cfg.CORSOrigins,
		AllowCredentials: true,
		AllowMethods: []string{
			stdhttp.MethodGet,
			stdhttp.MethodPost,
			stdhttp.MethodPut,
			stdhttp.MethodPatch,
			stdhttp.MethodDelete,
			stdhttp.MethodOptions,
		},
		AllowHeaders: []string{"*"},
	}))

	healthHandler := handler.NewHealthHandler(cfg.AppName, deps.HealthChecker)
	underlyingHandler := handler.NewUnderlyingHandler(deps.UnderlyingUseCase, logger)
	docsHandler := handler.NewDocsHandler(cfg.RootPath)

	root := e.Group(cfg.RootPath)
	root.GET("/health", healthHandler.GetHealth)
	root.GET("/openapi.json", docsHandler.OpenAPISpec)
	root.GET("/docs", docsHandler.SwaggerUI)
	root.GET("/redoc", docsHandler.Redoc)

	v1 := root.Group("/api/v1")
	v1.GET("/underlyings", underlyingHandler.ListUnderlyings)
	v1.GET("/underlyings/:symbol_fa", underlyingHandler.GetUnderlyingBySymbol)

	return e
}
