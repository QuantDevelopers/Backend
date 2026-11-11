package service

import (
	"errors"
	"testing"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/domain/exception"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	dbMocks "github.com/QuantDevelopers/Backend/mocks/infrastructure/database"
	repoMocks "github.com/QuantDevelopers/Backend/mocks/infrastructure/repository/postgres"
	"github.com/stretchr/testify/suite"
)

type UnderlyingServiceTestSuite struct {
	suite.Suite
	constants            *bootstrap.Constants
	underlyingRepository *repoMocks.UnderlyingRepositoryMock
	db                   *dbMocks.DatabaseMock
	underlyingService    *UnderlyingService
}

func (s *UnderlyingServiceTestSuite) SetupTest() {
	cfg := bootstrap.Run()
	s.constants = cfg.Constants
	s.underlyingRepository = repoMocks.NewUnderlyingRepositoryMock()
	s.db = dbMocks.NewDatabaseMock()
	s.underlyingService = NewUnderlyingService(s.constants, s.underlyingRepository, s.db)
}

func (s *UnderlyingServiceTestSuite) TestGetUnderlyings() {
	s.Run("success - returns underlying assets list", func() {
		tsetmc1 := "24000305886470390"
		assets := []*entity.UnderlyingAsset{
			{
				Model:      database.Model{ID: 1, CreatedAt: time.Now().UTC()},
				SymbolFA:   "اهرم",
				NameFA:     "صندوق س. ا. کاریسما-شتاب",
				TSETMCID:   &tsetmc1,
				MarketType: "Bourse",
				IsActive:   true,
			},
			{
				Model:      database.Model{ID: 2, CreatedAt: time.Now().UTC()},
				SymbolFA:   "خودرو",
				NameFA:     "ایران خودرو",
				MarketType: "Bourse",
				IsActive:   true,
			},
		}

		s.underlyingRepository.On("GetAll", s.db).Return(assets, nil).Once()

		res, err := s.underlyingService.GetUnderlyings()

		s.NoError(err)
		s.Equal(2, res.Total)
		s.Len(res.Items, 2)
		s.Equal("اهرم", res.Items[0].SymbolFA)
		s.Equal("خودرو", res.Items[1].SymbolFA)
		s.underlyingRepository.AssertExpectations(s.T())
	})

	s.Run("error - repository failure", func() {
		s.underlyingRepository.On("GetAll", s.db).Return(nil, errors.New("db failure")).Once()

		_, err := s.underlyingService.GetUnderlyings()

		s.Error(err)
		s.underlyingRepository.AssertExpectations(s.T())
	})
}

func (s *UnderlyingServiceTestSuite) TestGetUnderlyingBySymbol() {
	s.Run("success - symbol found", func() {
		asset := &entity.UnderlyingAsset{
			Model:      database.Model{ID: 1},
			SymbolFA:   "اهرم",
			NameFA:     "صندوق س. ا. کاریسما-شتاب",
			MarketType: "Bourse",
			IsActive:   true,
		}
		s.underlyingRepository.On("GetBySymbol", s.db, "اهرم").Return(asset, nil).Once()

		res, err := s.underlyingService.GetUnderlyingBySymbol("اهرم")

		s.NoError(err)
		s.Equal("اهرم", res.SymbolFA)
		s.Equal("صندوق س. ا. کاریسما-شتاب", res.NameFA)
		s.underlyingRepository.AssertExpectations(s.T())
	})

	s.Run("error - symbol not found returns NotFoundError", func() {
		s.underlyingRepository.On("GetBySymbol", s.db, "ناموجود").Return(nil, nil).Once()

		_, err := s.underlyingService.GetUnderlyingBySymbol("ناموجود")

		s.Error(err)
		var notFoundErr exception.NotFoundError
		s.ErrorAs(err, &notFoundErr)
		s.Equal(s.constants.Field.UnderlyingAsset, notFoundErr.Item)
		s.underlyingRepository.AssertExpectations(s.T())
	})
}

func TestUnderlyingServiceSuite(t *testing.T) {
	suite.Run(t, new(UnderlyingServiceTestSuite))
}
