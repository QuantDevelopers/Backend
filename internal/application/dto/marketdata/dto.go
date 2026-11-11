package marketdatadto

import commondto "github.com/QuantDevelopers/Backend/internal/application/dto/common"

// SearchInstrumentsRequest binds query parameters for GET /api/v1/instruments/search.
type SearchInstrumentsRequest struct {
	Query string `query:"q" validate:"required"`
	Limit int    `query:"limit"`
}

// InstrumentSearchItem represents either an underlying asset or an option contract in search results.
type InstrumentSearchItem struct {
	InstrumentType string   `json:"instrument_type"`
	ID             uint     `json:"id"`
	SymbolFA       string   `json:"symbol_fa"`
	NameFA         string   `json:"name_fa"`
	TSETMCID       *string  `json:"tsetmc_id,omitempty"`
	ISIN           *string  `json:"isin,omitempty"`
	MarketType     string   `json:"market_type,omitempty"`
	ContractType   string   `json:"contract_type,omitempty"`
	StrikePrice    *float64 `json:"strike_price,omitempty"`
	IsActive       bool     `json:"is_active"`
}

// InstrumentSearchResponse returns unified matching underlyings and option contracts.
type InstrumentSearchResponse struct {
	Query string                 `json:"query"`
	Total int                    `json:"total"`
	Items []InstrumentSearchItem `json:"items"`
}

// GetDailyHistoryRequest binds parameters for GET /api/v1/daily-history/{instrument_type}/{symbol_fa}.
type GetDailyHistoryRequest struct {
	InstrumentType string `param:"instrument_type" validate:"required"`
	SymbolFA       string `param:"symbol_fa" validate:"required"`
	From           string `query:"from"`
	To             string `query:"to"`
	Limit          int    `query:"limit"`
}

// DailyHistoryItem serializes a single OHLCV trading day bar.
type DailyHistoryItem struct {
	ID              uint     `json:"id"`
	TradeDate       string   `json:"trade_date"`
	OpenPrice       float64  `json:"open_price"`
	HighPrice       float64  `json:"high_price"`
	LowPrice        float64  `json:"low_price"`
	ClosePrice      float64  `json:"close_price"`
	YesterdayPrice  *float64 `json:"yesterday_price,omitempty"`
	SettlementPrice *float64 `json:"settlement_price,omitempty"`
	Volume          int64    `json:"volume"`
	Value           float64  `json:"value"`
	TradeCount      int      `json:"trade_count"`
	OpenInterest    *int     `json:"open_interest,omitempty"`
}

// DailyHistoryResponse returns daily bars for the requested underlying or option.
type DailyHistoryResponse struct {
	InstrumentType string             `json:"instrument_type"`
	InstrumentID   uint               `json:"instrument_id"`
	SymbolFA       string             `json:"symbol_fa"`
	TSETMCID       *string            `json:"tsetmc_id,omitempty"`
	Total          int                `json:"total"`
	Items          []DailyHistoryItem `json:"items"`
}

// GetTicksRequest binds parameters for GET /api/v1/ticks/{instrument_type}/{symbol_fa}.
type GetTicksRequest struct {
	InstrumentType string `param:"instrument_type" validate:"required"`
	SymbolFA       string `param:"symbol_fa" validate:"required"`
	Date           string `query:"date"`
	Page           int    `query:"page"`
	PageSize       int    `query:"page_size"`
}

// TickItem serializes one raw intraday trade tick row.
type TickItem struct {
	ID             uint    `json:"id"`
	TradeDate      string  `json:"trade_date"`
	TradeTime      string  `json:"trade_time"`
	SequenceNumber *int    `json:"sequence_number,omitempty"`
	Price          float64 `json:"price"`
	Volume         int64   `json:"volume"`
}

// TicksPaginatedResponse returns paginated intraday trade ticks for an instrument and date.
type TicksPaginatedResponse struct {
	InstrumentType string                   `json:"instrument_type"`
	InstrumentID   uint                     `json:"instrument_id"`
	SymbolFA       string                   `json:"symbol_fa"`
	TradeDate      string                   `json:"trade_date,omitempty"`
	Items          []TickItem               `json:"items"`
	Pagination     commondto.PaginationMeta `json:"pagination"`
}
