package optiondto

import "time"

type OptionResponse struct {
	ID                uint       `json:"id"`
	UnderlyingAssetID uint       `json:"underlying_asset_id"`
	SymbolFA          string     `json:"symbol_fa"`
	NameFA            string     `json:"name_fa"`
	ContractType      string     `json:"contract_type"`
	StrikePrice       float64    `json:"strike_price"`
	ContractSize      int        `json:"contract_size"`
	ExpirationDate    time.Time  `json:"expiration_date"`
	BeginDate         *time.Time `json:"begin_date"`
	TSETMCID          *string    `json:"tsetmc_id"`
	ISIN              *string    `json:"isin"`
	IsActive          bool       `json:"is_active"`
	Status            string     `json:"status"`
	CreatedAt         time.Time  `json:"created_at"`
}
