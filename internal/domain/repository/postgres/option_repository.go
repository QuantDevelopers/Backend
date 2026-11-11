package postgres

import (
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

type OptionFilter struct {
	UnderlyingAssetID *uint
	ContractType      *string
	IsActive          *bool
}

type OptionContractRepository interface {
	GetAll(db database.Database, filter OptionFilter) ([]*entity.OptionContract, error)
	GetBySymbol(db database.Database, symbolFA string) (*entity.OptionContract, error)
	Save(db database.Database, contract *entity.OptionContract) error
	SaveBulk(db database.Database, contracts []*entity.OptionContract) (int, error)
}
