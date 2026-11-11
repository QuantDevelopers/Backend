//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/QuantDevelopers/Backend/bootstrap"
	"github.com/QuantDevelopers/Backend/internal/application/service"
	"github.com/QuantDevelopers/Backend/internal/application/usecase"
	domainloc "github.com/QuantDevelopers/Backend/internal/domain/localization"
	domainlogger "github.com/QuantDevelopers/Backend/internal/domain/logger"
	domainmetrics "github.com/QuantDevelopers/Backend/internal/domain/metrics"
	domainpostgres "github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	domaintsetmc "github.com/QuantDevelopers/Backend/internal/domain/tsetmc"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	infraloc "github.com/QuantDevelopers/Backend/internal/infrastructure/localization"
	infralogger "github.com/QuantDevelopers/Backend/internal/infrastructure/logger"
	inframetrics "github.com/QuantDevelopers/Backend/internal/infrastructure/metrics"
	infrapostgres "github.com/QuantDevelopers/Backend/internal/infrastructure/repository/postgres"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/seed"
	infratsetmc "github.com/QuantDevelopers/Backend/internal/infrastructure/tsetmc"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller/v1/health"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller/v1/underlying"
	"github.com/QuantDevelopers/Backend/internal/presentation/middleware"
	"github.com/QuantDevelopers/Backend/internal/presentation/routes"
	"github.com/google/wire"
)

var DatabaseProviderSet = wire.NewSet(
	database.NewPostgresDatabase,
	wire.Bind(new(database.Database), new(*database.PostgresDatabase)),
	wire.Struct(new(Database), "*"),
)

var RepositoryProviderSet = wire.NewSet(
	infrapostgres.NewUnderlyingRepository,
	infrapostgres.NewOptionContractRepository,
	infrapostgres.NewDailyHistoryRepository,
	wire.Bind(new(domainpostgres.UnderlyingRepository), new(*infrapostgres.UnderlyingRepository)),
	wire.Bind(new(domainpostgres.OptionContractRepository), new(*infrapostgres.OptionContractRepository)),
	wire.Bind(new(domainpostgres.DailyHistoryRepository), new(*infrapostgres.DailyHistoryRepository)),
)

var ServiceProviderSet = wire.NewSet(
	service.NewHealthService,
	service.NewUnderlyingService,
	service.NewOptionService,
	service.NewDiscoveryService,
	service.NewTickIngestorService,
	service.NewBenchmarkService,
	service.NewDailyHistoryCrawlerService,
	wire.Bind(new(usecase.HealthService), new(*service.HealthService)),
	wire.Bind(new(usecase.UnderlyingService), new(*service.UnderlyingService)),
	wire.Bind(new(usecase.OptionService), new(*service.OptionService)),
	wire.Bind(new(usecase.DiscoveryService), new(*service.DiscoveryService)),
	wire.Bind(new(usecase.TickIngestorService), new(*service.TickIngestorService)),
	wire.Bind(new(usecase.BenchmarkService), new(*service.BenchmarkService)),
	wire.Bind(new(usecase.DailyHistoryCrawlerService), new(*service.DailyHistoryCrawlerService)),
)

var AdapterProviderSet = wire.NewSet(
	infraloc.NewTranslationService,
	infralogger.NewLogger,
	inframetrics.NewPrometheusMetrics,
	infratsetmc.NewHTTPClientFromConfig,
	wire.Bind(new(domainlogger.Logger), new(*infralogger.Logger)),
	wire.Bind(new(domainmetrics.MetricsClient), new(*inframetrics.PrometheusMetrics)),
	wire.Bind(new(domaintsetmc.Client), new(*infratsetmc.HTTPClient)),
)

var GeneralControllerProviderSet = wire.NewSet(
	health.NewGeneralHealthController,
	underlying.NewGeneralUnderlyingController,
	wire.Struct(new(GeneralControllers), "*"),
)

var ControllersProviderSet = wire.NewSet(
	wire.Struct(new(Controllers), "*"),
)

var MiddlewareProviderSet = wire.NewSet(
	middleware.NewCorsMiddleware,
	middleware.NewLocalization,
	middleware.NewLoggerMiddleware,
	middleware.NewPrometheusMiddleware,
	middleware.NewRateLimit,
	middleware.NewRecovery,
	wire.Struct(new(Middlewares), "*"),
)

var SeederProviderSet = wire.NewSet(
	seed.NewUnderlyingSeeder,
	wire.Struct(new(Seeds), "*"),
)

func ProvideConstants(container *bootstrap.Config) *bootstrap.Constants {
	return container.Constants
}

func ProvideServerConfig(container *bootstrap.Config) *bootstrap.Server {
	return &container.Env.Server
}

func ProvideLoggerConfig(container *bootstrap.Config) *bootstrap.Logger {
	return &container.Env.Logger
}

func ProvideRateLimitConfig(container *bootstrap.Config) *bootstrap.RateLimit {
	return &container.Env.RateLimit
}

func ProvideDBConfig(container *bootstrap.Config) *bootstrap.Database {
	return &container.Env.PrimaryDB
}

func ProvideCORSConfig(container *bootstrap.Config) *bootstrap.CORS {
	return &container.Env.CORS
}

func ProvideMetrics(container *bootstrap.Config) *bootstrap.Metrics {
	return &container.Constants.Metrics
}

func ProvideCrawlerConfig(container *bootstrap.Config) *bootstrap.Crawler {
	return &container.Env.Crawler
}

func ProvideNilDatabase() database.Database {
	return nil
}

func ProvideNilUnderlyingRepository() domainpostgres.UnderlyingRepository {
	return nil
}

func ProvideNilOptionContractRepository() domainpostgres.OptionContractRepository {
	return nil
}

func ProvideNilDailyHistoryRepository() domainpostgres.DailyHistoryRepository {
	return nil
}

var ProviderSet = wire.NewSet(
	DatabaseProviderSet,
	RepositoryProviderSet,
	ServiceProviderSet,
	AdapterProviderSet,
	GeneralControllerProviderSet,
	ControllersProviderSet,
	MiddlewareProviderSet,
	SeederProviderSet,
	ProvideConstants,
	ProvideServerConfig,
	ProvideLoggerConfig,
	ProvideRateLimitConfig,
	ProvideDBConfig,
	ProvideCORSConfig,
	ProvideMetrics,
	ProvideCrawlerConfig,
)

var OfflineCrawlerProviderSet = wire.NewSet(
	infralogger.NewLogger,
	infratsetmc.NewHTTPClientFromConfig,
	wire.Bind(new(domainlogger.Logger), new(*infralogger.Logger)),
	wire.Bind(new(domaintsetmc.Client), new(*infratsetmc.HTTPClient)),
	service.NewDiscoveryService,
	service.NewTickIngestorService,
	service.NewBenchmarkService,
	service.NewDailyHistoryCrawlerService,
	wire.Bind(new(usecase.DiscoveryService), new(*service.DiscoveryService)),
	wire.Bind(new(usecase.TickIngestorService), new(*service.TickIngestorService)),
	wire.Bind(new(usecase.BenchmarkService), new(*service.BenchmarkService)),
	wire.Bind(new(usecase.DailyHistoryCrawlerService), new(*service.DailyHistoryCrawlerService)),
	ProvideConstants,
	ProvideLoggerConfig,
	ProvideCrawlerConfig,
	ProvideNilDatabase,
	ProvideNilUnderlyingRepository,
	ProvideNilOptionContractRepository,
	ProvideNilDailyHistoryRepository,
)

type Database struct {
	DB database.Database
}

type GeneralControllers struct {
	HealthController     *health.GeneralHealthController
	UnderlyingController *underlying.GeneralUnderlyingController
}

type Controllers struct {
	General *GeneralControllers
}

type Middlewares struct {
	CORS         *middleware.CORSMiddleware
	Localization *middleware.LocalizationMiddleware
	Logger       *middleware.LoggerMiddleware
	Prometheus   *middleware.PrometheusMiddleware
	RateLimit    *middleware.RateLimitMiddleware
	Recovery     *middleware.RecoveryMiddleware
}

type Seeds struct {
	UnderlyingSeeder *seed.UnderlyingSeeder
}

type Application struct {
	Server        *bootstrap.Server
	Database      *Database
	Controllers   *Controllers
	Middlewares   *Middlewares
	Seeds         *Seeds
	Logger        domainlogger.Logger
	Translator    domainloc.Translator
	MetricsClient domainmetrics.MetricsClient
}

func NewApplication(
	server *bootstrap.Server,
	database *Database,
	controllers *Controllers,
	middlewares *Middlewares,
	seeds *Seeds,
	log domainlogger.Logger,
	translator domainloc.Translator,
	metricsClient domainmetrics.MetricsClient,
) *Application {
	return &Application{
		Server:        server,
		Database:      database,
		Controllers:   controllers,
		Middlewares:   middlewares,
		Seeds:         seeds,
		Logger:        log,
		Translator:    translator,
		MetricsClient: metricsClient,
	}
}

func (a *Application) ToAppRouter() *routes.AppRouter {
	return &routes.AppRouter{
		Server:               a.Server,
		MetricsClient:        a.MetricsClient,
		CORS:                 a.Middlewares.CORS,
		Logger:               a.Middlewares.Logger,
		Localization:         a.Middlewares.Localization,
		Recovery:             a.Middlewares.Recovery,
		RateLimit:            a.Middlewares.RateLimit,
		Prometheus:           a.Middlewares.Prometheus,
		HealthController:     a.Controllers.General.HealthController,
		UnderlyingController: a.Controllers.General.UnderlyingController,
	}
}

func InitializeApplication(container *bootstrap.Config) (*Application, error) {
	wire.Build(
		ProviderSet,
		NewApplication,
	)
	return &Application{}, nil
}

type CrawlerApplication struct {
	Logger              domainlogger.Logger
	DiscoveryService    usecase.DiscoveryService
	TickIngestorService usecase.TickIngestorService
	BenchmarkService    usecase.BenchmarkService
	DailyHistoryCrawler usecase.DailyHistoryCrawlerService
}

func NewCrawlerApplication(
	log domainlogger.Logger,
	discoveryService usecase.DiscoveryService,
	tickIngestorService usecase.TickIngestorService,
	benchmarkService usecase.BenchmarkService,
	dailyHistoryCrawler usecase.DailyHistoryCrawlerService,
) *CrawlerApplication {
	return &CrawlerApplication{
		Logger:              log,
		DiscoveryService:    discoveryService,
		TickIngestorService: tickIngestorService,
		BenchmarkService:    benchmarkService,
		DailyHistoryCrawler: dailyHistoryCrawler,
	}
}

func InitializeCrawlerApplication(container *bootstrap.Config) (*CrawlerApplication, error) {
	wire.Build(
		ProviderSet,
		NewCrawlerApplication,
	)
	return &CrawlerApplication{}, nil
}

func InitializeOfflineCrawlerApplication(container *bootstrap.Config) (*CrawlerApplication, error) {
	wire.Build(
		OfflineCrawlerProviderSet,
		NewCrawlerApplication,
	)
	return &CrawlerApplication{}, nil
}
