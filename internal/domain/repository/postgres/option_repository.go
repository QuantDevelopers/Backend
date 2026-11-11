package postgres

import (
	"time"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

type OptionFilter struct {
	UnderlyingAssetID *uint
	ContractType      *string
	IsActive          *bool
}

// OptionQueryFilter supports filtered, searched, and paginated E5-2 option listing.
type OptionQueryFilter struct {
	UnderlyingAssetID *uint
	ContractType      *string
	IsActive          *bool
	Query             string
	ExpirationDate    *time.Time
	Offset            int
	Limit             int
}

type OptionContractRepository interface {
	GetAll(db database.Database, filter OptionFilter) ([]*entity.OptionContract, error)
	ListPaginated(db database.Database, filter OptionQueryFilter) ([]*entity.OptionContract, int64, error)
	GetBySymbol(db database.Database, symbolFA string) (*entity.OptionContract, error)
	Save(db database.Database, contract *entity.OptionContract) error
	SaveBulk(db database.Database, contracts []*entity.OptionContract) (int, error)
}
