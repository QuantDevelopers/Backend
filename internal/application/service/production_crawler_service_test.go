package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	crawlerdto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	domainrepo "github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	domaintsetmc "github.com/QuantDevelopers/Backend/internal/domain/tsetmc"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	infralogger "github.com/QuantDevelopers/Backend/internal/infrastructure/logger"
	dbMocks "github.com/QuantDevelopers/Backend/mocks/infrastructure/database"
	repoMocks "github.com/QuantDevelopers/Backend/mocks/infrastructure/repository/postgres"
	tsetmcMocks "github.com/QuantDevelopers/Backend/mocks/infrastructure/tsetmc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ProductionCrawlerServiceTestSuite struct {
	suite.Suite
	cfg            *bootstrap.Config
	client         *tsetmcMocks.TSETMCClientMock
	underlyingRepo *repoMocks.UnderlyingRepositoryMock
	optionRepo     *repoMocks.OptionContractRepositoryMock
	dailyRepo      *repoMocks.DailyHistoryRepositoryMock
	tickRawRepo    *repoMocks.TickRawRepositoryMock
	db             *dbMocks.DatabaseMock
	service        *ProductionCrawlerService
	tickIngestor   *TickIngestorService
}

func (s *ProductionCrawlerServiceTestSuite) SetupTest() {
	s.cfg = bootstrap.Run()
	log, err := infralogger.NewLogger(&bootstrap.Logger{
		LogLevel:      "error",
		ConsoleOutput: "false",
	}, s.cfg.Constants)
	s.Require().NoError(err)

	s.client = tsetmcMocks.NewTSETMCClientMock()
	s.underlyingRepo = repoMocks.NewUnderlyingRepositoryMock()
	s.optionRepo = repoMocks.NewOptionContractRepositoryMock()
	s.dailyRepo = repoMocks.NewDailyHistoryRepositoryMock()
	s.tickRawRepo = repoMocks.NewTickRawRepositoryMock()
	s.db = dbMocks.NewDatabaseMock()
	s.tickIngestor = NewTickIngestorService(s.client, log)

	s.service = NewProductionCrawlerService(
		&s.cfg.Env.Crawler,
		s.tickIngestor,
		s.underlyingRepo,
		s.optionRepo,
		s.dailyRepo,
		s.tickRawRepo,
		s.db,
		log,
	)
}

func (s *ProductionCrawlerServiceTestSuite) TestConcurrentWorkerPoolPersistDBAndResumeSkip() {
	uaID := "17914401175772326"
	optID := "42511156565416187"
	day1 := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC)

	s.underlyingRepo.On("GetAll", s.db).Return([]*entity.UnderlyingAsset{
		{Model: database.Model{ID: 1}, SymbolFA: "اهرم", TSETMCID: &uaID, IsActive: true},
	}, nil)
	active := true
	s.optionRepo.On("GetAll", s.db, domainrepo.OptionFilter{IsActive: &active}).Return([]*entity.OptionContract{
		{Model: database.Model{ID: 10}, SymbolFA: "ضهرم1033", TSETMCID: &optID, ContractSize: 1000, IsActive: true},
	}, nil)
	s.dailyRepo.On("GetTradingDates", s.db, "UNDERLYING", uint(1), true).Return([]time.Time{day1, day2}, nil)
	s.dailyRepo.On("GetTradingDates", s.db, "OPTION", uint(10), true).Return([]time.Time{day1}, nil)

	s.tickRawRepo.On("ReplaceDayTicks", s.db, mock.AnythingOfType("string"), mock.AnythingOfType("uint"), mock.Anything, mock.Anything).
		Return(10, nil)

	dataDir := s.T().TempDir()
	fixturePath := filepath.Join("..", "..", "..", "tests", "fixtures", "sample_trade_history.json")

	// First run: 3 tasks queued concurrently (2 underlying days + 1 option day).
	firstReport, err := s.service.Run(context.Background(), crawlerdto.ProductionCrawlOptions{
		Workers:            3,
		DataDir:            dataDir,
		IncludeUnderlyings: true,
		IncludeOptions:     true,
		Resume:             true,
		PersistDB:          true,
		UseOfflineFixture:  true,
		FixturePath:        fixturePath,
	})
	s.Require().NoError(err)
	s.Equal(3, firstReport.TotalQueued)
	s.Equal(0, firstReport.SkippedResume)
	s.Equal(3, firstReport.Succeeded)
	s.Equal(0, firstReport.Failed)
	s.Equal(30, firstReport.TotalRecords)
	s.Equal(30, firstReport.DBRowsSaved)

	// Second run with Resume=true: all 3 valid manifests + raw payloads already exist -> skipped cleanly.
	secondReport, err := s.service.Run(context.Background(), crawlerdto.ProductionCrawlOptions{
		Workers:            2,
		DataDir:            dataDir,
		IncludeUnderlyings: true,
		IncludeOptions:     true,
		Resume:             true,
		PersistDB:          true,
		UseOfflineFixture:  true,
		FixturePath:        fixturePath,
	})
	s.Require().NoError(err)
	s.Equal(0, secondReport.TotalQueued)
	s.Equal(3, secondReport.SkippedResume)

	// Corrupt one raw file -> resume detects checksum mismatch and queues only that 1 task.
	corruptPath := filepath.Join(dataDir, "raw", "ticks", optID, "20261028.json")
	s.Require().NoError(os.WriteFile(corruptPath, []byte(`corrupted`), 0o644))

	thirdReport, err := s.service.Run(context.Background(), crawlerdto.ProductionCrawlOptions{
		Workers:            2,
		DataDir:            dataDir,
		IncludeUnderlyings: true,
		IncludeOptions:     true,
		Resume:             true,
		PersistDB:          true,
		UseOfflineFixture:  true,
		FixturePath:        fixturePath,
	})
	s.Require().NoError(err)
	s.Equal(1, thirdReport.TotalQueued)
	s.Equal(2, thirdReport.SkippedResume)
	s.Equal(1, thirdReport.Succeeded)
}

func (s *ProductionCrawlerServiceTestSuite) TestCircuitBreakerHaltsWorkersAndStatusReadsDiskAndDB() {
	uaID := "17914401175772326"
	s.underlyingRepo.On("GetAll", s.db).Return([]*entity.UnderlyingAsset{
		{Model: database.Model{ID: 1}, SymbolFA: "اهرم", TSETMCID: &uaID, IsActive: true},
	}, nil)

	s.client.On("Get", mock.Anything, mock.AnythingOfType("string")).
		Return([]byte(nil), http.StatusServiceUnavailable, domaintsetmc.ErrCircuitBreakerOpen)

	dataDir := s.T().TempDir()
	report, err := s.service.Run(context.Background(), crawlerdto.ProductionCrawlOptions{
		Workers:            2,
		DataDir:            dataDir,
		Dates:              []string{"20261028", "20261027"},
		IncludeUnderlyings: true,
		IncludeOptions:     false,
		DisableFallback:    true,
	})
	s.Require().NoError(err)
	s.True(report.CircuitBroken)
	s.GreaterOrEqual(report.Failed, 1)

	s.optionRepo.On("GetAll", s.db, domainrepo.OptionFilter{}).Return([]*entity.OptionContract{}, nil)
	s.dailyRepo.On("Count", s.db, domainrepo.DailyHistoryFilter{}).Return(10, nil)
	s.tickRawRepo.On("Count", s.db, domainrepo.TickRawFilter{}).Return(45, nil)

	status, err := s.service.GetStatus(dataDir)
	s.Require().NoError(err)
	s.Equal(1, status.Database.UnderlyingsTotal)
	s.Equal(1, status.Database.UnderlyingsWithTSETMCID)
	s.Equal(int64(10), status.Database.DailyHistoryRows)
	s.Equal(int64(45), status.Database.TickRawRows)
	s.Require().NotNil(status.LatestJob)
	s.True(status.LatestJob.CircuitBroken)
}

func (s *ProductionCrawlerServiceTestSuite) TestLiveWorkerPoolAgainstMockHTTPServer() {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"tradeHistory":[{"nTran":1,"hEven":90501,"pTran":1500,"qTitTran":4,"canceled":0}]}`))
	}))
	defer srv.Close()
	s.NotZero(srv.URL)

	uaID := "17914401175772326"
	s.underlyingRepo.On("GetAll", s.db).Return([]*entity.UnderlyingAsset{
		{Model: database.Model{ID: 1}, SymbolFA: "اهرم", TSETMCID: &uaID, IsActive: true},
	}, nil)

	s.client.On("Get", mock.Anything, mock.AnythingOfType("string")).
		Run(func(args mock.Arguments) {
			resp, err := http.Get(srv.URL)
			if err == nil {
				_ = resp.Body.Close()
			}
		}).
		Return([]byte(`{"tradeHistory":[{"nTran":1,"hEven":90501,"pTran":1500,"qTitTran":4,"canceled":0}]}`), http.StatusOK, nil)

	report, err := s.service.Run(context.Background(), crawlerdto.ProductionCrawlOptions{
		Workers:            3,
		DataDir:            s.T().TempDir(),
		Dates:              []string{"20261028", "20261027", "20261026", "20261025", "20261024"},
		IncludeUnderlyings: true,
		IncludeOptions:     false,
		DisableFallback:    true,
	})
	s.Require().NoError(err)
	s.Equal(3, report.Workers)
	s.Equal(5, report.TotalQueued)
	s.Equal(5, report.Succeeded)
	s.Equal(int32(5), hits.Load())
}

func TestProductionCrawlerServiceSuite(t *testing.T) {
	suite.Run(t, new(ProductionCrawlerServiceTestSuite))
}
