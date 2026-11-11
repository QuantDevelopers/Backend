package repository

import (
	"context"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
)

// OptionFilter defines optional filters for querying option contracts.
type OptionFilter struct {
	UnderlyingAssetID *int
	ContractType      *string
	IsActive          *bool
}

// UnderlyingRepository abstracts persistence operations for underlying assets.
type UnderlyingRepository interface {
	GetAll(ctx context.Context) ([]entity.UnderlyingAsset, error)
	GetBySymbol(ctx context.Context, symbolFA string) (*entity.UnderlyingAsset, error)
	SeedDefaults(ctx context.Context) (int, error)
}

// OptionContractRepository abstracts persistence operations for option contracts.
type OptionContractRepository interface {
	GetAll(ctx context.Context, filter OptionFilter) ([]entity.OptionContract, error)
	GetBySymbol(ctx context.Context, symbolFA string) (*entity.OptionContract, error)
	Save(ctx context.Context, contract entity.OptionContract) (*entity.OptionContract, error)
	SaveBulk(ctx context.Context, contracts []entity.OptionContract) (int, error)
}
