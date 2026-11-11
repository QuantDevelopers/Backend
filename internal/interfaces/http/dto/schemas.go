package dto

import (
	"time"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
)

// HealthResponse represents the payload returned by GET /health.
type HealthResponse struct {
	Status    string    `json:"status"`
	App       string    `json:"app"`
	Database  string    `json:"database"`
	Timestamp time.Time `json:"timestamp"`
}

// UnderlyingAssetResponse represents a serialized underlying asset returned by the API.
type UnderlyingAssetResponse struct {
	ID         int        `json:"id"`
	SymbolFA   string     `json:"symbol_fa"`
	NameFA     string     `json:"name_fa"`
	TSETMCID   *string    `json:"tsetmc_id"`
	MarketType string     `json:"market_type"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  *time.Time `json:"created_at"`
}

// UnderlyingListResponse wraps the list of underlying assets with total count.
type UnderlyingListResponse struct {
	Total int                       `json:"total"`
	Items []UnderlyingAssetResponse `json:"items"`
}

// HTTPErrorResponse represents an API error payload compatible with FastAPI's detail convention.
type HTTPErrorResponse struct {
	Detail string `json:"detail"`
}

// FromUnderlyingEntity maps a domain UnderlyingAsset to an HTTP response DTO.
func FromUnderlyingEntity(a entity.UnderlyingAsset) UnderlyingAssetResponse {
	return UnderlyingAssetResponse{
		ID:         a.ID,
		SymbolFA:   a.SymbolFA,
		NameFA:     a.NameFA,
		TSETMCID:   a.TSETMCID,
		MarketType: a.MarketType,
		IsActive:   a.IsActive,
		CreatedAt:  a.CreatedAt,
	}
}
