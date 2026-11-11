package service

import (
	"context"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/domain/repository"
)

// OptionService orchestrates use cases for option contracts.
type OptionService struct {
	repo repository.OptionContractRepository
}

// NewOptionService constructs an OptionService with its repository dependency.
func NewOptionService(repo repository.OptionContractRepository) *OptionService {
	return &OptionService{repo: repo}
}

// ListOptions lists option contracts optionally filtered by underlying, contract type, and active status.
func (s *OptionService) ListOptions(ctx context.Context, filter repository.OptionFilter) ([]entity.OptionContract, error) {
	return s.repo.GetAll(ctx, filter)
}

// GetBySymbol fetches a single option contract by its Persian symbol, or nil if absent.
func (s *OptionService) GetBySymbol(ctx context.Context, symbolFA string) (*entity.OptionContract, error) {
	return s.repo.GetBySymbol(ctx, symbolFA)
}

// CreateOrUpdate persists an option contract and returns the stored record.
func (s *OptionService) CreateOrUpdate(ctx context.Context, contract entity.OptionContract) (*entity.OptionContract, error) {
	return s.repo.Save(ctx, contract)
}
