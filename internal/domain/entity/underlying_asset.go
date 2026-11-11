package entity

import (
	"fmt"

	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

type UnderlyingAsset struct {
	database.Model
	SymbolFA        string           `gorm:"column:symbol_fa;type:varchar(50);uniqueIndex:ix_underlying_assets_symbol_fa;not null"`
	NameFA          string           `gorm:"column:name_fa;type:varchar(150);not null"`
	TSETMCID        *string          `gorm:"column:tsetmc_id;type:varchar(50);uniqueIndex:ix_underlying_assets_tsetmc_id"`
	ISIN            *string          `gorm:"column:isin;type:varchar(50);uniqueIndex:ix_underlying_assets_isin"`
	MarketType      string           `gorm:"column:market_type;type:varchar(50);default:'Bourse';not null"`
	IsActive        bool             `gorm:"column:is_active;default:true;not null"`
	OptionContracts []OptionContract `gorm:"foreignKey:UnderlyingAssetID;constraint:OnDelete:CASCADE"`
}

func (UnderlyingAsset) TableName() string {
	return "underlying_assets"
}

func (u UnderlyingAsset) String() string {
	return fmt.Sprintf("<UnderlyingAsset(id=%d, symbol_fa='%s', name_fa='%s')>", u.ID, u.SymbolFA, u.NameFA)
}
