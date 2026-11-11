package optiondto

import (
	"time"

	commondto "github.com/QuantDevelopers/Backend/internal/application/dto/common"
	underlyingdto "github.com/QuantDevelopers/Backend/internal/application/dto/underlying"
)

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

// OptionListPaginatedResponse wraps filtered option contracts with pagination metadata.
type OptionListPaginatedResponse struct {
	Items      []OptionResponse         `json:"items"`
	Pagination commondto.PaginationMeta `json:"pagination"`
}

// OptionChainRow groups matching Call and Put contracts at a single strike & expiry.
type OptionChainRow struct {
	ExpirationDate string          `json:"expiration_date"`
	StrikePrice    float64         `json:"strike_price"`
	ContractSize   int             `json:"contract_size"`
	CallMoneyness  string          `json:"call_moneyness,omitempty"`
	PutMoneyness   string          `json:"put_moneyness,omitempty"`
	Call           *OptionResponse `json:"call,omitempty"`
	Put            *OptionResponse `json:"put,omitempty"`
}

// OptionChainResponse returns an underlying asset's paired Call/Put option matrix.
type OptionChainResponse struct {
	Underlying  underlyingdto.UnderlyingResponse `json:"underlying"`
	SpotPrice   *float64                         `json:"spot_price,omitempty"`
	Expirations []string                         `json:"expirations"`
	TotalRows   int                              `json:"total_rows"`
	Rows        []OptionChainRow                 `json:"rows"`
}
