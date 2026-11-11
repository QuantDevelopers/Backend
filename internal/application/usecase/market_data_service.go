package usecase

import marketdatadto "github.com/QuantDevelopers/Backend/internal/application/dto/marketdata"

// MarketDataService serves search, daily OHLCV history, and raw tick queries for REST API v1.
type MarketDataService interface {
	SearchInstruments(req marketdatadto.SearchInstrumentsRequest) (marketdatadto.InstrumentSearchResponse, error)
	GetDailyHistory(req marketdatadto.GetDailyHistoryRequest) (marketdatadto.DailyHistoryResponse, error)
	GetTicks(req marketdatadto.GetTicksRequest) (marketdatadto.TicksPaginatedResponse, error)
}
