package service

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

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

type DailyHistoryCrawlerServiceTestSuite struct {
	suite.Suite
	client         *tsetmcMocks.TSETMCClientMock
	underlyingRepo *repoMocks.UnderlyingRepositoryMock
	optionRepo     *repoMocks.OptionContractRepositoryMock
	dailyRepo      *repoMocks.DailyHistoryRepositoryMock
	db             *dbMocks.DatabaseMock
	service        *DailyHistoryCrawlerService
}

func (s *DailyHistoryCrawlerServiceTestSuite) SetupTest() {
	cfg := bootstrap.Run()
	log, err := infralogger.NewLogger(&bootstrap.Logger{
		LogLevel:      "error",
		ConsoleOutput: "false",
	}, cfg.Constants)
	s.Require().NoError(err)

	s.client = tsetmcMocks.NewTSETMCClientMock()
	s.underlyingRepo = repoMocks.NewUnderlyingRepositoryMock()
	s.optionRepo = repoMocks.NewOptionContractRepositoryMock()
	s.dailyRepo = repoMocks.NewDailyHistoryRepositoryMock()
	s.db = dbMocks.NewDatabaseMock()

	s.service = NewDailyHistoryCrawlerService(
		s.client,
		s.underlyingRepo,
		s.optionRepo,
		s.dailyRepo,
		s.db,
		log,
	)
}

func (s *DailyHistoryCrawlerServiceTestSuite) TestParseDailySampleFixturesAndSanitizesOHLC() {
	rawSample := `{
		"closingPriceDaily": [
			{"dEven": 20261028, "priceFirst": 0, "priceMax": 0, "priceMin": 0, "pClosing": 71666, "pDrCotVal": 72740, "priceYesterday": 70744, "zTotTran": 20749, "qTotTran5J": 162591305, "qTotCap": 11652324145553},
			{"dEven": 20261028, "priceFirst": 69869, "priceMax": 73070, "priceMin": 69205, "pClosing": 71666, "pDrCotVal": 72740, "priceYesterday": 70744, "zTotTran": 20749, "qTotTran5J": 162591305, "qTotCap": 11652324145553},
			{"dEven": 20261027, "priceFirst": 74000, "priceMax": 74420, "priceMin": 69843, "pClosing": 70744, "pDrCotVal": 69871, "priceYesterday": 72753, "zTotTran": 22928, "qTotTran5J": 212430401, "qTotCap": 15028131015896},
			{"dEven": 0}
		]
	}`

	records, err := s.service.ParseDailyPayload([]byte(rawSample), "underlying", 5)
	s.Require().NoError(err)
	// Deduplicated duplicate 20261028 + ignored dEven=0
	s.Require().Len(records, 2)
	s.Equal("UNDERLYING", records[0].InstrumentType)
	s.Equal(uint(5), records[0].InstrumentID)
	s.Equal("2026-10-28", records[0].TradeDate.Format("2006-01-02"))
	s.InDelta(69869.0, records[0].OpenPrice, 0.01)
	s.InDelta(71666.0, records[0].ClosePrice, 0.01)
	s.Equal(int64(162591305), records[0].Volume)
	s.Equal(20749, records[0].TradeCount)
	s.Require().NotNil(records[0].YesterdayPrice)
	s.InDelta(70744.0, *records[0].YesterdayPrice, 0.01)
}

func (s *DailyHistoryCrawlerServiceTestSuite) TestCrawlOfflineFixturesWithPostgresMocks() {
	uaIDStr := "17914401175772326"
	optIDStr := "19693832241143005"
	missingIDAsset := &entity.UnderlyingAsset{
		Model:    database.Model{ID: 2},
		SymbolFA: "فولاد",
		TSETMCID: nil,
		IsActive: true,
	}

	s.underlyingRepo.On("GetAll", s.db).Return([]*entity.UnderlyingAsset{
		{Model: database.Model{ID: 1}, SymbolFA: "اهرم", TSETMCID: &uaIDStr, IsActive: true},
		missingIDAsset,
	}, nil).Once()

	active := true
	s.optionRepo.On("GetAll", s.db, domainrepo.OptionFilter{IsActive: &active}).Return([]*entity.OptionContract{
		{Model: database.Model{ID: 10}, SymbolFA: "ضهرم1033", TSETMCID: &optIDStr, IsActive: true},
	}, nil).Once()

	s.dailyRepo.On("SaveBulk", s.db, mock.MatchedBy(func(items []*entity.DailyHistory) bool {
		return len(items) == 10 && items[0].InstrumentType == "UNDERLYING" && items[0].InstrumentID == 1
	})).Return(10, nil).Once()

	s.dailyRepo.On("SaveBulk", s.db, mock.MatchedBy(func(items []*entity.DailyHistory) bool {
		return len(items) == 6 && items[0].InstrumentType == "OPTION" && items[0].InstrumentID == 10
	})).Return(6, nil).Once()

	res, err := s.service.Crawl(context.Background(), crawlerdto.DailyHistoryCrawlOptions{
		IncludeUnderlyings:    true,
		IncludeOptions:        true,
		SaveRaw:               true,
		RawDir:                s.T().TempDir(),
		UseOfflineFixture:     true,
		UnderlyingFixturePath: filepath.Join("..", "..", "..", "data", "raw", "samples", "sample_underlying_daily.json"),
		OptionFixturePath:     filepath.Join("..", "..", "..", "data", "raw", "samples", "sample_option_daily.json"),
	})
	s.Require().NoError(err)
	s.Equal(2, res.InstrumentsTotal)
	s.Equal(2, res.InstrumentsSucceeded)
	s.Equal(0, res.InstrumentsFailed)
	s.Equal(16, res.RowsParsed)
	s.Equal(16, res.RowsSaved)
	s.False(res.CircuitBroken)
	s.FileExists(res.Items[0].RawPath)
	s.FileExists(res.Items[1].RawPath)

	s.underlyingRepo.AssertExpectations(s.T())
	s.optionRepo.AssertExpectations(s.T())
	s.dailyRepo.AssertExpectations(s.T())
}

func (s *DailyHistoryCrawlerServiceTestSuite) TestCircuitBreakerStopsCrawlCleanly() {
	ua1 := "17914401175772326"
	ua2 := "65883838195688438"

	s.underlyingRepo.On("GetAll", s.db).Return([]*entity.UnderlyingAsset{
		{Model: database.Model{ID: 1}, SymbolFA: "اهرم", TSETMCID: &ua1, IsActive: true},
		{Model: database.Model{ID: 2}, SymbolFA: "خودرو", TSETMCID: &ua2, IsActive: true},
	}, nil).Once()

	s.client.On("Get", mock.Anything, fmt.Sprintf(TSETMCDailyHistoryURL, ua1, 0)).
		Return([]byte(nil), http.StatusServiceUnavailable, domaintsetmc.ErrCircuitBreakerOpen).Once()

	res, err := s.service.Crawl(context.Background(), crawlerdto.DailyHistoryCrawlOptions{
		IncludeUnderlyings: true,
		IncludeOptions:     false,
		DisableFallback:    true,
	})
	s.Require().NoError(err)
	s.True(res.CircuitBroken)
	s.Equal(2, res.InstrumentsTotal)
	s.Equal(0, res.InstrumentsSucceeded)
	s.Equal(1, res.InstrumentsFailed)
	s.Len(res.Items, 1)
	s.client.AssertExpectations(s.T())
}

func TestDailyHistoryCrawlerServiceSuite(t *testing.T) {
	suite.Run(t, new(DailyHistoryCrawlerServiceTestSuite))
}
