package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	_ "github.com/QuantDevelopers/Backend/docs/swagger"
	commondto "github.com/QuantDevelopers/Backend/internal/application/dto/common"
	crawlerdto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	healthdto "github.com/QuantDevelopers/Backend/internal/application/dto/health"
	marketdatadto "github.com/QuantDevelopers/Backend/internal/application/dto/marketdata"
	optiondto "github.com/QuantDevelopers/Backend/internal/application/dto/option"
	underlyingdto "github.com/QuantDevelopers/Backend/internal/application/dto/underlying"
	"github.com/QuantDevelopers/Backend/internal/domain/exception"
	infraloc "github.com/QuantDevelopers/Backend/internal/infrastructure/localization"
	infralogger "github.com/QuantDevelopers/Backend/internal/infrastructure/logger"
	inframetrics "github.com/QuantDevelopers/Backend/internal/infrastructure/metrics"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller/v1/crawl"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller/v1/health"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller/v1/marketdata"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller/v1/option"
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
	optionService     *serviceMocks.OptionServiceMock
	marketDataService *serviceMocks.MarketDataServiceMock
	crawlerService    *serviceMocks.ProductionCrawlerServiceMock
	echoEngine        *echo.Echo
}

func (s *RouteTestSuite) SetupTest() {
	s.config = bootstrap.Run()
	s.healthService = serviceMocks.NewHealthServiceMock()
	s.underlyingService = serviceMocks.NewUnderlyingServiceMock()
	s.optionService = serviceMocks.NewOptionServiceMock()
	s.marketDataService = serviceMocks.NewMarketDataServiceMock()
	s.crawlerService = serviceMocks.NewProductionCrawlerServiceMock()

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
		OptionController:     option.NewGeneralOptionController(s.optionService),
		MarketDataController: marketdata.NewGeneralMarketDataController(s.marketDataService),
		CrawlController:      crawl.NewGeneralCrawlController(&s.config.Env.Crawler, s.crawlerService),
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
	for _, path := range []string{"/swagger/index.html", "/swagger/doc.json", "/metrics"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		s.echoEngine.ServeHTTP(rec, req)
		s.Equal(http.StatusOK, rec.Code, "expected 200 for %s", path)
	}

	for _, redirectPath := range []string{"/docs", "/swagger"} {
		req := httptest.NewRequest(http.MethodGet, redirectPath, nil)
		rec := httptest.NewRecorder()
		s.echoEngine.ServeHTTP(rec, req)
		s.Equal(http.StatusMovedPermanently, rec.Code, "expected 301 redirect for %s", redirectPath)
		s.Equal("/swagger/index.html", rec.Header().Get("Location"))
	}
}

func (s *RouteTestSuite) TestExtendedV1Routes() {
	s.Run("GET /api/v1/options paginated", func() {
		s.optionService.On("ListOptionsPaginated", optiondto.ListOptionsQueryRequest{
			ContractType: "CALL",
			Page:         1,
			PageSize:     10,
		}).Return(optiondto.OptionListPaginatedResponse{
			Items: []optiondto.OptionResponse{{ID: 1, SymbolFA: "ضهرم1033", ContractType: "CALL", StrikePrice: 50000}},
			Pagination: commondto.PaginationMeta{
				CurrentPage: 1,
				PageSize:    10,
				TotalItems:  1,
				TotalPages:  1,
			},
		}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/options?contract_type=CALL&page=1&page_size=10", nil)
		rec := httptest.NewRecorder()
		s.echoEngine.ServeHTTP(rec, req)
		s.Equal(http.StatusOK, rec.Code)
		s.optionService.AssertExpectations(s.T())
	})

	s.Run("GET /api/v1/options/:symbol_fa and /underlyings/:symbol_fa/option-chain", func() {
		s.optionService.On("GetOptionBySymbol", "ضهرم1033").Return(optiondto.OptionResponse{
			ID:           1,
			SymbolFA:     "ضهرم1033",
			ContractType: "CALL",
		}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/api/v1/options/"+url.PathEscape("ضهرم1033"), nil)
		rec := httptest.NewRecorder()
		s.echoEngine.ServeHTTP(rec, req)
		s.Equal(http.StatusOK, rec.Code)

		s.optionService.On("GetOptionChain", optiondto.GetOptionChainRequest{SymbolFA: "اهرم"}).
			Return(optiondto.OptionChainResponse{
				Underlying:  underlyingdto.UnderlyingResponse{ID: 1, SymbolFA: "اهرم"},
				Expirations: []string{"2027-01-13"},
				TotalRows:   1,
			}, nil).Once()

		reqChain := httptest.NewRequest(http.MethodGet, "/api/v1/underlyings/"+url.PathEscape("اهرم")+"/option-chain", nil)
		recChain := httptest.NewRecorder()
		s.echoEngine.ServeHTTP(recChain, reqChain)
		s.Equal(http.StatusOK, recChain.Code)
		s.optionService.AssertExpectations(s.T())
	})

	s.Run("GET /api/v1/instruments/search, /daily-history, /ticks, and /crawl/status", func() {
		s.marketDataService.On("SearchInstruments", marketdatadto.SearchInstrumentsRequest{Query: "اهرم"}).
			Return(marketdatadto.InstrumentSearchResponse{Query: "اهرم", Total: 1}, nil).Once()

		reqSearch := httptest.NewRequest(http.MethodGet, "/api/v1/instruments/search?q="+url.QueryEscape("اهرم"), nil)
		recSearch := httptest.NewRecorder()
		s.echoEngine.ServeHTTP(recSearch, reqSearch)
		s.Equal(http.StatusOK, recSearch.Code)

		s.marketDataService.On("GetDailyHistory", marketdatadto.GetDailyHistoryRequest{
			InstrumentType: "underlying",
			SymbolFA:       "اهرم",
		}).Return(marketdatadto.DailyHistoryResponse{InstrumentType: "UNDERLYING", SymbolFA: "اهرم", Total: 0}, nil).Once()

		reqDaily := httptest.NewRequest(http.MethodGet, "/api/v1/daily-history/underlying/"+url.PathEscape("اهرم"), nil)
		recDaily := httptest.NewRecorder()
		s.echoEngine.ServeHTTP(recDaily, reqDaily)
		s.Equal(http.StatusOK, recDaily.Code)

		s.marketDataService.On("GetTicks", marketdatadto.GetTicksRequest{
			InstrumentType: "underlying",
			SymbolFA:       "اهرم",
		}).Return(marketdatadto.TicksPaginatedResponse{InstrumentType: "UNDERLYING", SymbolFA: "اهرم"}, nil).Once()

		reqTicks := httptest.NewRequest(http.MethodGet, "/api/v1/ticks/underlying/"+url.PathEscape("اهرم"), nil)
		recTicks := httptest.NewRecorder()
		s.echoEngine.ServeHTTP(recTicks, reqTicks)
		s.Equal(http.StatusOK, recTicks.Code)

		s.crawlerService.On("GetStatus", "data").Return(&crawlerdto.CrawlStatusResponse{
			SafeMaxRPS: 3.0,
			Database:   crawlerdto.DBCoverageSummary{UnderlyingsTotal: 25},
		}, nil).Once()

		reqStatus := httptest.NewRequest(http.MethodGet, "/api/v1/crawl/status", nil)
		recStatus := httptest.NewRecorder()
		s.echoEngine.ServeHTTP(recStatus, reqStatus)
		s.Equal(http.StatusOK, recStatus.Code)

		s.marketDataService.AssertExpectations(s.T())
		s.crawlerService.AssertExpectations(s.T())
	})
}

func TestRouteSuite(t *testing.T) {
	suite.Run(t, new(RouteTestSuite))
}
