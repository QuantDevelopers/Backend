package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	healthdto "github.com/QuantDevelopers/Backend/internal/application/dto/health"
	underlyingdto "github.com/QuantDevelopers/Backend/internal/application/dto/underlying"
	"github.com/QuantDevelopers/Backend/internal/domain/exception"
	infraloc "github.com/QuantDevelopers/Backend/internal/infrastructure/localization"
	infralogger "github.com/QuantDevelopers/Backend/internal/infrastructure/logger"
	inframetrics "github.com/QuantDevelopers/Backend/internal/infrastructure/metrics"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller/v1/docs"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller/v1/health"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller/v1/underlying"
	"github.com/QuantDevelopers/Backend/internal/presentation/middleware"
	"github.com/QuantDevelopers/Backend/internal/presentation/routes"
	serviceMocks "github.com/QuantDevelopers/Backend/mocks/service"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type envelope struct {
	StatusCode int             `json:"statusCode"`
	Message    string          `json:"message"`
	Data       json.RawMessage `json:"data"`
}

type RouteTestSuite struct {
	suite.Suite
	config            *bootstrap.Config
	healthService     *serviceMocks.HealthServiceMock
	underlyingService *serviceMocks.UnderlyingServiceMock
	echoEngine        *echo.Echo
}

func (s *RouteTestSuite) SetupTest() {
	s.config = bootstrap.Run()
	s.healthService = serviceMocks.NewHealthServiceMock()
	s.underlyingService = serviceMocks.NewUnderlyingServiceMock()

	log, err := infralogger.NewLogger(&bootstrap.Logger{
		LogLevel:      "error",
		ConsoleOutput: "false",
	}, s.config.Constants)
	s.Require().NoError(err)

	translator := infraloc.NewTranslationService()
	metricsClient := inframetrics.NewPrometheusMetrics(&s.config.Constants.Metrics)

	appRouter := &routes.AppRouter{
		Server:               &s.config.Env.Server,
		MetricsClient:        metricsClient,
		CORS:                 middleware.NewCorsMiddleware(&s.config.Env.CORS),
		Logger:               middleware.NewLoggerMiddleware(log),
		Localization:         middleware.NewLocalization(s.config.Constants, translator),
		Recovery:             middleware.NewRecovery(s.config.Constants, log),
		RateLimit:            middleware.NewRateLimit(&s.config.Env.RateLimit),
		Prometheus:           middleware.NewPrometheusMiddleware(metricsClient),
		HealthController:     health.NewGeneralHealthController(s.healthService),
		UnderlyingController: underlying.NewGeneralUnderlyingController(s.underlyingService),
		DocsController:       docs.NewGeneralDocsController(&s.config.Env.Server),
	}

	s.echoEngine = echo.New()
	routes.Run(s.echoEngine, appRouter)
}

func (s *RouteTestSuite) TestHealthRoutes() {
	s.Run("200 healthy", func() {
		s.healthService.On("CheckHealth", mock.Anything).Return(healthdto.HealthResponse{
			Status:    "healthy",
			App:       "QuantBackend",
			Database:  "connected",
			Timestamp: time.Now().UTC(),
		}, true).Once()

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		s.echoEngine.ServeHTTP(rec, req)

		s.Equal(http.StatusOK, rec.Code)

		var env envelope
		s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &env))
		s.Equal(200, env.StatusCode)

		var payload healthdto.HealthResponse
		s.Require().NoError(json.Unmarshal(env.Data, &payload))
		s.Equal("healthy", payload.Status)
		s.Equal("connected", payload.Database)
		s.healthService.AssertExpectations(s.T())
	})

	s.Run("503 degraded", func() {
		s.healthService.On("CheckHealth", mock.Anything).Return(healthdto.HealthResponse{
			Status:    "degraded",
			App:       "QuantBackend",
			Database:  "disconnected",
			Timestamp: time.Now().UTC(),
		}, false).Once()

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		s.echoEngine.ServeHTTP(rec, req)

		s.Equal(http.StatusServiceUnavailable, rec.Code)
		s.healthService.AssertExpectations(s.T())
	})
}

func (s *RouteTestSuite) TestUnderlyingRoutes() {
	s.Run("GET /api/v1/underlyings returns list", func() {
		s.underlyingService.On("GetUnderlyings").Return(underlyingdto.UnderlyingListResponse{
			Total: 2,
			Items: []underlyingdto.UnderlyingResponse{
				{ID: 1, SymbolFA: "اهرم", NameFA: "صندوق س. ا. کاریسما-شتاب", MarketType: "Bourse", IsActive: true},
				{ID: 2, SymbolFA: "خودرو", NameFA: "ایران خودرو", MarketType: "Bourse", IsActive: true},
			},
		}, nil).Once()

		for _, path := range []string{"/api/v1/underlyings", "/api/v1/underlyings/"} {
			if path == "/api/v1/underlyings/" {
				s.underlyingService.On("GetUnderlyings").Return(underlyingdto.UnderlyingListResponse{
					Total: 2,
					Items: []underlyingdto.UnderlyingResponse{
						{ID: 1, SymbolFA: "اهرم", NameFA: "صندوق س. ا. کاریسما-شتاب", MarketType: "Bourse", IsActive: true},
						{ID: 2, SymbolFA: "خودرو", NameFA: "ایران خودرو", MarketType: "Bourse", IsActive: true},
					},
				}, nil).Once()
			}
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			s.echoEngine.ServeHTTP(rec, req)

			s.Equal(http.StatusOK, rec.Code)
			var env envelope
			s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &env))
			var list underlyingdto.UnderlyingListResponse
			s.Require().NoError(json.Unmarshal(env.Data, &list))
			s.Equal(2, list.Total)
			s.Equal("اهرم", list.Items[0].SymbolFA)
		}
		s.underlyingService.AssertExpectations(s.T())
	})

	s.Run("GET /api/v1/underlyings/:symbol_fa found", func() {
		s.underlyingService.On("GetUnderlyingBySymbol", "اهرم").Return(underlyingdto.UnderlyingResponse{
			ID:         1,
			SymbolFA:   "اهرم",
			NameFA:     "صندوق س. ا. کاریسما-شتاب",
			MarketType: "Bourse",
			IsActive:   true,
		}, nil).Times(2)

		for _, path := range []string{
			"/api/v1/underlyings/" + url.PathEscape("اهرم"),
			"/api/v1/underlyings/اهرم",
		} {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			s.echoEngine.ServeHTTP(rec, req)

			s.Equal(http.StatusOK, rec.Code)
			var env envelope
			s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &env))
			var item underlyingdto.UnderlyingResponse
			s.Require().NoError(json.Unmarshal(env.Data, &item))
			s.Equal("اهرم", item.SymbolFA)
		}
		s.underlyingService.AssertExpectations(s.T())
	})

	s.Run("GET /api/v1/underlyings/:symbol_fa not found localized in fa_IR and en_US", func() {
		s.underlyingService.On("GetUnderlyingBySymbol", "ناموجود").Return(
			underlyingdto.UnderlyingResponse{},
			exception.NotFoundError{Item: s.config.Constants.Field.UnderlyingAsset},
		).Times(2)

		reqEN := httptest.NewRequest(http.MethodGet, "/api/v1/underlyings/"+url.PathEscape("ناموجود"), nil)
		reqEN.Header.Set("Accept-Language", "en_US")
		recEN := httptest.NewRecorder()
		s.echoEngine.ServeHTTP(recEN, reqEN)

		s.Equal(http.StatusNotFound, recEN.Code)
		var envEN envelope
		s.Require().NoError(json.Unmarshal(recEN.Body.Bytes(), &envEN))
		s.Contains(envEN.Message, "not found")

		reqFA := httptest.NewRequest(http.MethodGet, "/api/v1/underlyings/"+url.PathEscape("ناموجود"), nil)
		reqFA.Header.Set("Accept-Language", "fa_IR")
		recFA := httptest.NewRecorder()
		s.echoEngine.ServeHTTP(recFA, reqFA)

		s.Equal(http.StatusNotFound, recFA.Code)
		var envFA envelope
		s.Require().NoError(json.Unmarshal(recFA.Body.Bytes(), &envFA))
		s.Contains(envFA.Message, "دارایی پایه")
		s.underlyingService.AssertExpectations(s.T())
	})
}

func (s *RouteTestSuite) TestDocsAndMetricsRoutes() {
	for _, path := range []string{"/openapi.json", "/docs", "/redoc", "/metrics"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		s.echoEngine.ServeHTTP(rec, req)
		s.Equal(http.StatusOK, rec.Code, "expected 200 for %s", path)
	}
}

func TestRouteSuite(t *testing.T) {
	suite.Run(t, new(RouteTestSuite))
}
