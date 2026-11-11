package postgres

import (
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

type UnderlyingRepository interface {
	GetAll(db database.Database) ([]*entity.UnderlyingAsset, error)
	GetBySymbol(db database.Database, symbolFA string) (*entity.UnderlyingAsset, error)
	GetByTSETMCID(db database.Database, tsetmcID string) (*entity.UnderlyingAsset, error)
	Create(db database.Database, asset *entity.UnderlyingAsset) error
	Upsert(db database.Database, asset *entity.UnderlyingAsset) error
}
