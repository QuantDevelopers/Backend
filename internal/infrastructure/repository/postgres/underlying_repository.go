package postgres

import (
	"errors"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	"gorm.io/gorm"
)

type UnderlyingRepository struct{}

func NewUnderlyingRepository() *UnderlyingRepository {
	return &UnderlyingRepository{}
}

func (repo *UnderlyingRepository) GetAll(db database.Database) ([]*entity.UnderlyingAsset, error) {
	var assets []*entity.UnderlyingAsset
	if err := db.GetDB().Order("id ASC").Find(&assets).Error; err != nil {
		return nil, err
	}
	return assets, nil
}

func (repo *UnderlyingRepository) GetBySymbol(db database.Database, symbolFA string) (*entity.UnderlyingAsset, error) {
	var asset entity.UnderlyingAsset
	err := db.GetDB().Where("symbol_fa = ?", symbolFA).First(&asset).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &asset, nil
}

func (repo *UnderlyingRepository) Create(db database.Database, asset *entity.UnderlyingAsset) error {
	return db.GetDB().Create(asset).Error
}
