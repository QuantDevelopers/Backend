package routes

import (
	"net/http"

	"github.com/QuantDevelopers/Backend/bootstrap"
	swaggerDocs "github.com/QuantDevelopers/Backend/docs/swagger"
	domainmetrics "github.com/QuantDevelopers/Backend/internal/domain/metrics"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller/v1/health"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller/v1/underlying"
	"github.com/QuantDevelopers/Backend/internal/presentation/middleware"
	httpv1 "github.com/QuantDevelopers/Backend/internal/presentation/routes/http/v1"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

type AppRouter struct {
	Server               *bootstrap.Server
	MetricsClient        domainmetrics.MetricsClient
	CORS                 *middleware.CORSMiddleware
	Logger               *middleware.LoggerMiddleware
	Localization         *middleware.LocalizationMiddleware
	Recovery             *middleware.RecoveryMiddleware
	RateLimit            *middleware.RateLimitMiddleware
	Prometheus           *middleware.PrometheusMiddleware
	HealthController     *health.GeneralHealthController
	UnderlyingController *underlying.GeneralUnderlyingController
}

func Run(echoEngine *echo.Echo, app *AppRouter) {
	echoEngine.Pre(echomw.RemoveTrailingSlash())
	echoEngine.Use(app.CORS.CORS())
	echoEngine.Use(app.Logger.EchoLoggerMiddleware)
	echoEngine.Use(app.Localization.Localization)
	echoEngine.Use(app.Recovery.Recovery)
	echoEngine.Use(app.RateLimit.RateLimit)
	echoEngine.Use(app.Prometheus.Prometheus)

	if app.Server != nil && app.Server.RootPath != "" {
		swaggerDocs.SwaggerInfo.BasePath = app.Server.RootPath
	}

	root := echoEngine.Group(app.Server.RootPath)
	root.GET("/metrics", echo.WrapHandler(app.MetricsClient.Handler()))
	root.GET("/health", app.HealthController.GetHealth)
	root.GET("/swagger", func(ctx echo.Context) error {
		return ctx.Redirect(http.StatusMovedPermanently, app.Server.RootPath+"/swagger/index.html")
	})
	root.GET("/docs", func(ctx echo.Context) error {
		return ctx.Redirect(http.StatusMovedPermanently, app.Server.RootPath+"/swagger/index.html")
	})
	root.GET("/swagger/*", echoSwagger.WrapHandler)

	v1 := root.Group("/api/v1")
	httpv1.SetupGeneralRoutes(v1, app.UnderlyingController)
}
