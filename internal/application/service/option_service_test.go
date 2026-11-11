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
	constants        *bootstrap.Constants
	optionRepository *repoMocks.OptionContractRepositoryMock
	db               *dbMocks.DatabaseMock
	optionService    *OptionService
}

func (s *OptionServiceTestSuite) SetupTest() {
	cfg := bootstrap.Run()
	s.constants = cfg.Constants
	s.optionRepository = repoMocks.NewOptionContractRepositoryMock()
	s.db = dbMocks.NewDatabaseMock()
	s.optionService = NewOptionService(s.constants, s.optionRepository, s.db)
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

func TestOptionServiceSuite(t *testing.T) {
	suite.Run(t, new(OptionServiceTestSuite))
}
