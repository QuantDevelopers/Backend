package postgres

import (
	"errors"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func (repo *UnderlyingRepository) Search(db database.Database, query string, limit int) ([]*entity.UnderlyingAsset, error) {
	tx := db.GetDB().Model(&entity.UnderlyingAsset{})
	if query != "" {
		like := "%" + query + "%"
		tx = tx.Where("symbol_fa ILIKE ? OR name_fa ILIKE ? OR tsetmc_id ILIKE ? OR isin ILIKE ?", like, like, like, like)
	}
	tx = tx.Order("id ASC")
	if limit > 0 {
		tx = tx.Limit(limit)
	}
	var assets []*entity.UnderlyingAsset
	if err := tx.Find(&assets).Error; err != nil {
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

func (repo *UnderlyingRepository) GetByTSETMCID(db database.Database, tsetmcID string) (*entity.UnderlyingAsset, error) {
	var asset entity.UnderlyingAsset
	err := db.GetDB().Where("tsetmc_id = ?", tsetmcID).First(&asset).Error
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

func (repo *UnderlyingRepository) Upsert(db database.Database, asset *entity.UnderlyingAsset) error {
	return db.GetDB().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "symbol_fa"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"name_fa":     gorm.Expr("EXCLUDED.name_fa"),
			"tsetmc_id":   gorm.Expr("COALESCE(EXCLUDED.tsetmc_id, underlying_assets.tsetmc_id)"),
			"isin":        gorm.Expr("COALESCE(EXCLUDED.isin, underlying_assets.isin)"),
			"market_type": gorm.Expr("EXCLUDED.market_type"),
			"is_active":   gorm.Expr("EXCLUDED.is_active"),
			"updated_at":  gorm.Expr("NOW()"),
		}),
	}).Create(asset).Error
}
