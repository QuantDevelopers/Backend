package service

import (
	"testing"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	marketdatadto "github.com/QuantDevelopers/Backend/internal/application/dto/marketdata"
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/domain/exception"
	domainrepo "github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	dbMocks "github.com/QuantDevelopers/Backend/mocks/infrastructure/database"
	repoMocks "github.com/QuantDevelopers/Backend/mocks/infrastructure/repository/postgres"
	"github.com/stretchr/testify/suite"
)

type MarketDataServiceTestSuite struct {
	suite.Suite
	constants      *bootstrap.Constants
	underlyingRepo *repoMocks.UnderlyingRepositoryMock
	optionRepo     *repoMocks.OptionContractRepositoryMock
	dailyRepo      *repoMocks.DailyHistoryRepositoryMock
	tickRawRepo    *repoMocks.TickRawRepositoryMock
	db             *dbMocks.DatabaseMock
	service        *MarketDataService
}

func (s *MarketDataServiceTestSuite) SetupTest() {
	cfg := bootstrap.Run()
	s.constants = cfg.Constants
	s.underlyingRepo = repoMocks.NewUnderlyingRepositoryMock()
	s.optionRepo = repoMocks.NewOptionContractRepositoryMock()
	s.dailyRepo = repoMocks.NewDailyHistoryRepositoryMock()
	s.tickRawRepo = repoMocks.NewTickRawRepositoryMock()
	s.db = dbMocks.NewDatabaseMock()

	s.service = NewMarketDataService(
		s.constants,
		s.underlyingRepo,
		s.optionRepo,
		s.dailyRepo,
		s.tickRawRepo,
		s.db,
	)
}

func (s *MarketDataServiceTestSuite) TestSearchInstrumentsCombinesUnderlyingsAndOptions() {
	s.underlyingRepo.On("Search", s.db, "اهرم", 10).Return([]*entity.UnderlyingAsset{
		{Model: database.Model{ID: 1}, SymbolFA: "اهرم", NameFA: "صندوق اهرم", MarketType: "Bourse", IsActive: true},
	}, nil).Once()
	s.optionRepo.On("ListPaginated", s.db, domainrepo.OptionQueryFilter{Query: "اهرم", Limit: 9}).
		Return([]*entity.OptionContract{
			{Model: database.Model{ID: 12}, SymbolFA: "ضهرم101", NameFA: "اختیار خ اهرم", ContractType: "CALL", StrikePrice: 50000, IsActive: true},
		}, 1, nil).Once()

	res, err := s.service.SearchInstruments(marketdatadto.SearchInstrumentsRequest{Query: "  اهرم ", Limit: 10})
	s.Require().NoError(err)
	s.Equal(2, res.Total)
	s.Equal("UNDERLYING", res.Items[0].InstrumentType)
	s.Equal("OPTION", res.Items[1].InstrumentType)
}

func (s *MarketDataServiceTestSuite) TestGetDailyHistoryAndGetTicks() {
	s.underlyingRepo.On("GetBySymbol", s.db, "اهرم").Return(&entity.UnderlyingAsset{
		Model:    database.Model{ID: 1},
		SymbolFA: "اهرم",
	}, nil)

	instType := "UNDERLYING"
	instID := uint(1)
	day := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	s.dailyRepo.On("List", s.db, domainrepo.DailyHistoryFilter{
		InstrumentType: &instType,
		InstrumentID:   &instID,
		FromDate:       &day,
	}, 100).Return([]*entity.DailyHistory{
		{Model: database.Model{ID: 9}, InstrumentType: "UNDERLYING", InstrumentID: 1, TradeDate: day, ClosePrice: 71666, Volume: 162591305},
	}, nil).Once()

	dailyRes, err := s.service.GetDailyHistory(marketdatadto.GetDailyHistoryRequest{
		InstrumentType: "underlying",
		SymbolFA:       "اهرم",
		From:           "2026-10-28",
	})
	s.Require().NoError(err)
	s.Equal(1, dailyRes.Total)
	s.Equal("2026-10-28", dailyRes.Items[0].TradeDate)

	seq := 1
	s.tickRawRepo.On("GetLatestDate", s.db, "UNDERLYING", uint(1)).Return(&day, nil).Once()
	s.tickRawRepo.On("GetByInstrumentAndDate", s.db, "UNDERLYING", uint(1), day, 0, 100).
		Return([]*entity.TickRaw{
			{Model: database.Model{ID: 5}, InstrumentType: "UNDERLYING", InstrumentID: 1, TradeDate: day, TradeTime: "09:00:05", SequenceNumber: &seq, Price: 71000, Volume: 100},
		}, 1, nil).Once()

	ticksRes, err := s.service.GetTicks(marketdatadto.GetTicksRequest{
		InstrumentType: "underlying",
		SymbolFA:       "اهرم",
	})
	s.Require().NoError(err)
	s.Equal("2026-10-28", ticksRes.TradeDate)
	s.Len(ticksRes.Items, 1)
	s.Equal("09:00:05", ticksRes.Items[0].TradeTime)

	// Invalid instrument type yields ValidationErrors.
	_, err = s.service.GetTicks(marketdatadto.GetTicksRequest{
		InstrumentType: "futures",
		SymbolFA:       "اهرم",
	})
	var vErr exception.ValidationErrors
	s.ErrorAs(err, &vErr)
}

func TestMarketDataServiceSuite(t *testing.T) {
	suite.Run(t, new(MarketDataServiceTestSuite))
}
