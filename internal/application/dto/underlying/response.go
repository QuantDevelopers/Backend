package underlyingdto

import "time"

type UnderlyingResponse struct {
	ID         uint      `json:"id"`
	SymbolFA   string    `json:"symbol_fa"`
	NameFA     string    `json:"name_fa"`
	TSETMCID   *string   `json:"tsetmc_id"`
	MarketType string    `json:"market_type"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

type UnderlyingListResponse struct {
	Total int                  `json:"total"`
	Items []UnderlyingResponse `json:"items"`
}
