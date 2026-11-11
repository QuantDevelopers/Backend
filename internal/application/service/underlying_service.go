package service

import (
	"context"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/domain/repository"
)

// UnderlyingService orchestrates use cases for underlying assets.
type UnderlyingService struct {
	repo repository.UnderlyingRepository
}

// NewUnderlyingService constructs an UnderlyingService with its repository dependency.
func NewUnderlyingService(repo repository.UnderlyingRepository) *UnderlyingService {
	return &UnderlyingService{repo: repo}
}

// ListUnderlyings ensures the 25 default TSETMC underlying assets exist and returns all records.
func (s *UnderlyingService) ListUnderlyings(ctx context.Context) ([]entity.UnderlyingAsset, error) {
	if _, err := s.repo.SeedDefaults(ctx); err != nil {
		return nil, err
	}
	return s.repo.GetAll(ctx)
}

// GetUnderlyingBySymbol returns a single underlying asset matching its Persian symbol, or nil if absent.
func (s *UnderlyingService) GetUnderlyingBySymbol(ctx context.Context, symbolFA string) (*entity.UnderlyingAsset, error) {
	return s.repo.GetBySymbol(ctx, symbolFA)
}
