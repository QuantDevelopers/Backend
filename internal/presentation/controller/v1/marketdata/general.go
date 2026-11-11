package marketdata

import (
	"net/http"

	marketdatadto "github.com/QuantDevelopers/Backend/internal/application/dto/marketdata"
	"github.com/QuantDevelopers/Backend/internal/application/usecase"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller"
	"github.com/labstack/echo/v4"
)

type GeneralMarketDataController struct {
	marketDataService usecase.MarketDataService
}

func NewGeneralMarketDataController(marketDataService usecase.MarketDataService) *GeneralMarketDataController {
	return &GeneralMarketDataController{
		marketDataService: marketDataService,
	}
}

// SearchInstruments searches across underlying assets and option contracts.
// @Summary Unified instrument search
// @Description Searches underlying assets and option contracts by normalized Persian symbol, name, TSETMC insCode, or ISIN.
// @Tags MarketData
// @Produce json
// @Param q query string true "Search text (Persian symbol, name, or insCode)"
// @Param limit query int false "Max results (default 20, max 100)"
// @Success 200 {object} controller.SingleMessageResponse{data=marketdatadto.InstrumentSearchResponse}
// @Failure 400 {object} controller.SingleMessageResponse
// @Router /api/v1/instruments/search [get]
func (mc *GeneralMarketDataController) SearchInstruments(ctx echo.Context) error {
	params := controller.Validated[marketdatadto.SearchInstrumentsRequest](ctx)
	res, err := mc.marketDataService.SearchInstruments(params)
	if err != nil {
		panic(err)
	}
	return controller.Response(ctx, http.StatusOK, "", res)
}

// GetDailyHistory returns historical daily OHLCV bars for an underlying or option contract.
// @Summary Get daily OHLCV history
// @Description Fetches persisted daily OHLCV and volume records for an underlying or option symbol.
// @Tags MarketData
// @Produce json
// @Param instrument_type path string true "Instrument type (underlying or option)"
// @Param symbol_fa path string true "Persian symbol"
// @Param from query string false "Start trade date inclusive (YYYY-MM-DD or YYYYMMDD)"
// @Param to query string false "End trade date inclusive (YYYY-MM-DD or YYYYMMDD)"
// @Param limit query int false "Max daily bars (default 100, max 1000)"
// @Success 200 {object} controller.SingleMessageResponse{data=marketdatadto.DailyHistoryResponse}
// @Failure 400 {object} controller.SingleMessageResponse
// @Failure 404 {object} controller.SingleMessageResponse
// @Router /api/v1/daily-history/{instrument_type}/{symbol_fa} [get]
func (mc *GeneralMarketDataController) GetDailyHistory(ctx echo.Context) error {
	params := controller.Validated[marketdatadto.GetDailyHistoryRequest](ctx)
	res, err := mc.marketDataService.GetDailyHistory(params)
	if err != nil {
		panic(err)
	}
	return controller.Response(ctx, http.StatusOK, "", res)
}

// GetTicks returns paginated intraday trade ticks for an underlying or option contract.
// @Summary Get raw intraday ticks
// @Description Fetches paginated intraday trade ticks ordered chronologically for a symbol and trading day.
// @Tags MarketData
// @Produce json
// @Param instrument_type path string true "Instrument type (underlying or option)"
// @Param symbol_fa path string true "Persian symbol"
// @Param date query string false "Trading date (YYYY-MM-DD or YYYYMMDD; defaults to latest persisted day)"
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Ticks per page (default 100, max 500)"
// @Success 200 {object} controller.SingleMessageResponse{data=marketdatadto.TicksPaginatedResponse}
// @Failure 400 {object} controller.SingleMessageResponse
// @Failure 404 {object} controller.SingleMessageResponse
// @Router /api/v1/ticks/{instrument_type}/{symbol_fa} [get]
func (mc *GeneralMarketDataController) GetTicks(ctx echo.Context) error {
	params := controller.Validated[marketdatadto.GetTicksRequest](ctx)
	res, err := mc.marketDataService.GetTicks(params)
	if err != nil {
		panic(err)
	}
	return controller.Response(ctx, http.StatusOK, "", res)
}
