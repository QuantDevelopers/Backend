package optiondto

import "time"

type ListOptionsRequest struct {
	UnderlyingAssetID *uint
	ContractType      *string
	IsActive          *bool
}

// ListOptionsQueryRequest binds REST query filters and pagination for GET /api/v1/options.
type ListOptionsQueryRequest struct {
	UnderlyingID     *uint  `query:"underlying_id"`
	UnderlyingSymbol string `query:"symbol_ua"`
	ContractType     string `query:"contract_type"`
	IsActive         *bool  `query:"is_active"`
	Query            string `query:"q"`
	ExpirationDate   string `query:"expiration_date"`
	Page             int    `query:"page"`
	PageSize         int    `query:"page_size"`
}

// GetOptionBySymbolRequest binds the Persian symbol path parameter for GET /api/v1/options/{symbol_fa}.
type GetOptionBySymbolRequest struct {
	SymbolFA string `param:"symbol_fa" validate:"required"`
}

// GetOptionChainRequest binds parameters for GET /api/v1/underlyings/{symbol_fa}/option-chain.
type GetOptionChainRequest struct {
	SymbolFA       string   `param:"symbol_fa" validate:"required"`
	ExpirationDate string   `query:"expiration_date"`
	SpotPrice      *float64 `query:"spot_price"`
}

type SaveOptionRequest struct {
	UnderlyingAssetID uint
	SymbolFA          string
	NameFA            string
	ContractType      string
	StrikePrice       float64
	ContractSize      int
	ExpirationDate    time.Time
	BeginDate         *time.Time
	TSETMCID          *string
	ISIN              *string
	IsActive          bool
	Status            string
}
