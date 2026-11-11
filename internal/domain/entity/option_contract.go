package entity

import (
	"fmt"
	"time"

	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

type OptionContract struct {
	database.Model
	UnderlyingAssetID uint             `gorm:"column:underlying_asset_id;not null;index;index:ix_option_ua_expiry,priority:1;index:ix_option_ua_type_strike,priority:1"`
	SymbolFA          string           `gorm:"column:symbol_fa;type:varchar(50);uniqueIndex:ix_option_contracts_symbol_fa;not null"`
	NameFA            string           `gorm:"column:name_fa;type:varchar(150);not null"`
	ContractType      string           `gorm:"column:contract_type;type:varchar(10);not null;index;index:ix_option_ua_type_strike,priority:2;check:check_contract_type,contract_type IN ('CALL', 'PUT')"`
	StrikePrice       float64          `gorm:"column:strike_price;type:numeric(15,2);not null;index;index:ix_option_ua_type_strike,priority:3"`
	ContractSize      int              `gorm:"column:contract_size;default:1000;not null"`
	ExpirationDate    time.Time        `gorm:"column:expiration_date;type:date;not null;index;index:ix_option_ua_expiry,priority:2"`
	BeginDate         *time.Time       `gorm:"column:begin_date;type:date"`
	TSETMCID          *string          `gorm:"column:tsetmc_id;type:varchar(50);uniqueIndex:ix_option_contracts_tsetmc_id"`
	ISIN              *string          `gorm:"column:isin;type:varchar(50);uniqueIndex:ix_option_contracts_isin"`
	IsActive          bool             `gorm:"column:is_active;default:true;not null;index"`
	Status            string           `gorm:"column:status;type:varchar(20);default:'ACTIVE';not null"`
	UnderlyingAsset   *UnderlyingAsset `gorm:"foreignKey:UnderlyingAssetID"`
}

func (OptionContract) TableName() string {
	return "option_contracts"
}

func (o OptionContract) String() string {
	return fmt.Sprintf(
		"<OptionContract(id=%d, symbol_fa='%s', type='%s', strike=%.2f, expiry='%s')>",
		o.ID, o.SymbolFA, o.ContractType, o.StrikePrice, o.ExpirationDate.Format("2006-01-02"),
	)
}
