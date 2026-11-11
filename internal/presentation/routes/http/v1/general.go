package httpv1

import (
	"github.com/QuantDevelopers/Backend/internal/presentation/controller/v1/crawl"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller/v1/marketdata"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller/v1/option"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller/v1/underlying"
	"github.com/labstack/echo/v4"
)

func SetupGeneralRoutes(
	routerGroup *echo.Group,
	underlyingController *underlying.GeneralUnderlyingController,
	optionController *option.GeneralOptionController,
	marketDataController *marketdata.GeneralMarketDataController,
	crawlController *crawl.GeneralCrawlController,
) {
	underlyings := routerGroup.Group("/underlyings")
	{
		underlyings.GET("", underlyingController.GetUnderlyings)
		underlyings.GET("/:symbol_fa", underlyingController.GetUnderlyingBySymbol)
		underlyings.GET("/:symbol_fa/option-chain", optionController.GetOptionChain)
	}

	options := routerGroup.Group("/options")
	{
		options.GET("", optionController.GetOptions)
		options.GET("/:symbol_fa", optionController.GetOptionBySymbol)
	}

	routerGroup.GET("/instruments/search", marketDataController.SearchInstruments)
	routerGroup.GET("/daily-history/:instrument_type/:symbol_fa", marketDataController.GetDailyHistory)
	routerGroup.GET("/ticks/:instrument_type/:symbol_fa", marketDataController.GetTicks)
	routerGroup.GET("/crawl/status", crawlController.GetCrawlStatus)
}
