package service

import (
	"github.com/QuantDevelopers/Backend/bootstrap"
	underlyingdto "github.com/QuantDevelopers/Backend/internal/application/dto/underlying"
	"github.com/QuantDevelopers/Backend/internal/domain/exception"
	"github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

type UnderlyingService struct {
	constants            *bootstrap.Constants
	underlyingRepository postgres.UnderlyingRepository
	db                   database.Database
}

func NewUnderlyingService(
	constants *bootstrap.Constants,
	underlyingRepository postgres.UnderlyingRepository,
	db database.Database,
) *UnderlyingService {
	return &UnderlyingService{
		constants:            constants,
		underlyingRepository: underlyingRepository,
		db:                   db,
	}
}

func (s *UnderlyingService) GetUnderlyings() (underlyingdto.UnderlyingListResponse, error) {
	assets, err := s.underlyingRepository.GetAll(s.db)
	if err != nil {
		return underlyingdto.UnderlyingListResponse{}, err
	}

	items := make([]underlyingdto.UnderlyingResponse, len(assets))
	for i, asset := range assets {
		items[i] = underlyingdto.UnderlyingResponse{
			ID:         asset.ID,
			SymbolFA:   asset.SymbolFA,
			NameFA:     asset.NameFA,
			TSETMCID:   asset.TSETMCID,
			MarketType: asset.MarketType,
			IsActive:   asset.IsActive,
			CreatedAt:  asset.CreatedAt,
		}
	}

	return underlyingdto.UnderlyingListResponse{
		Total: len(items),
		Items: items,
	}, nil
}

func (s *UnderlyingService) GetUnderlyingBySymbol(symbolFA string) (underlyingdto.UnderlyingResponse, error) {
	asset, err := s.underlyingRepository.GetBySymbol(s.db, symbolFA)
	if err != nil {
		return underlyingdto.UnderlyingResponse{}, err
	}
	if asset == nil {
		return underlyingdto.UnderlyingResponse{}, exception.NotFoundError{Item: s.constants.Field.UnderlyingAsset}
	}

	return underlyingdto.UnderlyingResponse{
		ID:         asset.ID,
		SymbolFA:   asset.SymbolFA,
		NameFA:     asset.NameFA,
		TSETMCID:   asset.TSETMCID,
		MarketType: asset.MarketType,
		IsActive:   asset.IsActive,
		CreatedAt:  asset.CreatedAt,
	}, nil
}
