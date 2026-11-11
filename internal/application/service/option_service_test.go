package service

import (
	"testing"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	optiondto "github.com/QuantDevelopers/Backend/internal/application/dto/option"
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/domain/exception"
	domainrepo "github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	dbMocks "github.com/QuantDevelopers/Backend/mocks/infrastructure/database"
	repoMocks "github.com/QuantDevelopers/Backend/mocks/infrastructure/repository/postgres"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type OptionServiceTestSuite struct {
	suite.Suite
	constants            *bootstrap.Constants
	optionRepository     *repoMocks.OptionContractRepositoryMock
	underlyingRepository *repoMocks.UnderlyingRepositoryMock
	dailyRepository      *repoMocks.DailyHistoryRepositoryMock
	db                   *dbMocks.DatabaseMock
	optionService        *OptionService
}

func (s *OptionServiceTestSuite) SetupTest() {
	cfg := bootstrap.Run()
	s.constants = cfg.Constants
	s.optionRepository = repoMocks.NewOptionContractRepositoryMock()
	s.underlyingRepository = repoMocks.NewUnderlyingRepositoryMock()
	s.dailyRepository = repoMocks.NewDailyHistoryRepositoryMock()
	s.db = dbMocks.NewDatabaseMock()
	s.optionService = NewOptionService(
		s.constants,
		s.optionRepository,
		s.underlyingRepository,
		s.dailyRepository,
		s.db,
	)
}

func (s *OptionServiceTestSuite) TestListOptions() {
	s.Run("success - filtered options list", func() {
		uaID := uint(1)
		cType := "CALL"
		active := true
		expiry := time.Date(2025, 2, 22, 0, 0, 0, 0, time.UTC)

		contracts := []*entity.OptionContract{
			{
				Model:             database.Model{ID: 10},
				UnderlyingAssetID: 1,
				SymbolFA:          "ضهرم1204",
				NameFA:            "اختیار خ اهرم-1200-1403/12/04",
				ContractType:      "CALL",
				StrikePrice:       1200.0,
				ContractSize:      1000,
				ExpirationDate:    expiry,
				IsActive:          true,
				Status:            "ACTIVE",
			},
		}

		expectedFilter := domainrepo.OptionFilter{
			UnderlyingAssetID: &uaID,
			ContractType:      &cType,
			IsActive:          &active,
		}
		s.optionRepository.On("GetAll", s.db, expectedFilter).Return(contracts, nil).Once()

		res, err := s.optionService.ListOptions(optiondto.ListOptionsRequest{
			UnderlyingAssetID: &uaID,
			ContractType:      &cType,
			IsActive:          &active,
		})

		s.NoError(err)
		s.Len(res, 1)
		s.Equal("ضهرم1204", res[0].SymbolFA)
		s.Equal(1200.0, res[0].StrikePrice)
		s.optionRepository.AssertExpectations(s.T())
	})
}

func (s *OptionServiceTestSuite) TestGetOptionBySymbolAndSave() {
	s.Run("error - option not found", func() {
		s.optionRepository.On("GetBySymbol", s.db, "ضناموجود").Return(nil, nil).Once()

		_, err := s.optionService.GetOptionBySymbol("ضناموجود")

		var notFoundErr exception.NotFoundError
		s.ErrorAs(err, &notFoundErr)
		s.Equal(s.constants.Field.OptionContract, notFoundErr.Item)
		s.optionRepository.AssertExpectations(s.T())
	})

	s.Run("success - create or update applies defaults", func() {
		expiry := time.Date(2025, 2, 22, 0, 0, 0, 0, time.UTC)
		s.optionRepository.On("Save", s.db, mock.MatchedBy(func(c *entity.OptionContract) bool {
			return c.SymbolFA == "ضخود1204" && c.ContractSize == 1000 && c.Status == "ACTIVE"
		})).Return(nil).Once()

		res, err := s.optionService.CreateOrUpdateOption(optiondto.SaveOptionRequest{
			UnderlyingAssetID: 1,
			SymbolFA:          "ضخود1204",
			NameFA:            "اختیار خ خودرو",
			ContractType:      "CALL",
			StrikePrice:       2800,
			ExpirationDate:    expiry,
			IsActive:          true,
		})

		s.NoError(err)
		s.Equal(1000, res.ContractSize)
		s.Equal("ACTIVE", res.Status)
		s.optionRepository.AssertExpectations(s.T())
	})
}

func (s *OptionServiceTestSuite) TestListOptionsPaginatedAndOptionChain() {
	s.Run("ListOptionsPaginated resolves symbol_ua and applies pagination", func() {
		ua := &entity.UnderlyingAsset{Model: database.Model{ID: 1}, SymbolFA: "اهرم"}
		s.underlyingRepository.On("GetBySymbol", s.db, "اهرم").Return(ua, nil).Once()

		expiry := time.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC)
		uaID := uint(1)
		ct := "CALL"
		s.optionRepository.On("ListPaginated", s.db, domainrepo.OptionQueryFilter{
			UnderlyingAssetID: &uaID,
			ContractType:      &ct,
			Query:             "ضهرم",
			Offset:            0,
			Limit:             10,
		}).Return([]*entity.OptionContract{
			{
				Model:             database.Model{ID: 11},
				UnderlyingAssetID: 1,
				SymbolFA:          "ضهرم101",
				ContractType:      "CALL",
				StrikePrice:       50000,
				ContractSize:      1000,
				ExpirationDate:    expiry,
				IsActive:          true,
				Status:            "ACTIVE",
			},
		}, 1, nil).Once()

		res, err := s.optionService.ListOptionsPaginated(optiondto.ListOptionsQueryRequest{
			UnderlyingSymbol: "اهرم",
			ContractType:     "call",
			Query:            "ضهرم",
			Page:             1,
			PageSize:         10,
		})
		s.Require().NoError(err)
		s.Len(res.Items, 1)
		s.Equal(int64(1), res.Pagination.TotalItems)
	})

	s.Run("GetOptionChain pairs call and put by strike with spot price moneyness", func() {
		ua := &entity.UnderlyingAsset{Model: database.Model{ID: 1}, SymbolFA: "اهرم", NameFA: "صندوق اهرم", IsActive: true}
		s.underlyingRepository.On("GetBySymbol", s.db, "اهرم").Return(ua, nil).Once()
		s.dailyRepository.On("GetByInstrument", s.db, "UNDERLYING", uint(1), 1).Return([]*entity.DailyHistory{
			{ClosePrice: 72000},
		}, nil).Once()

		expiry := time.Date(2027, 1, 13, 0, 0, 0, 0, time.UTC)
		active := true
		uaID := uint(1)
		s.optionRepository.On("GetAll", s.db, domainrepo.OptionFilter{
			UnderlyingAssetID: &uaID,
			IsActive:          &active,
		}).Return([]*entity.OptionContract{
			{Model: database.Model{ID: 1}, UnderlyingAssetID: 1, SymbolFA: "ضهرم700", ContractType: "CALL", StrikePrice: 70000, ContractSize: 1000, ExpirationDate: expiry, IsActive: true},
			{Model: database.Model{ID: 2}, UnderlyingAssetID: 1, SymbolFA: "طهرم700", ContractType: "PUT", StrikePrice: 70000, ContractSize: 1000, ExpirationDate: expiry, IsActive: true},
		}, nil).Once()

		chain, err := s.optionService.GetOptionChain(optiondto.GetOptionChainRequest{SymbolFA: "اهرم"})
		s.Require().NoError(err)
		s.Require().NotNil(chain.SpotPrice)
		s.InDelta(72000.0, *chain.SpotPrice, 0.01)
		s.Equal([]string{"2027-01-13"}, chain.Expirations)
		s.Require().Len(chain.Rows, 1)
		s.Equal("ITM", chain.Rows[0].CallMoneyness)
		s.Equal("OTM", chain.Rows[0].PutMoneyness)
		s.Require().NotNil(chain.Rows[0].Call)
		s.Require().NotNil(chain.Rows[0].Put)
	})
}

func TestOptionServiceSuite(t *testing.T) {
	suite.Run(t, new(OptionServiceTestSuite))
}
