package httpv1

import (
	"github.com/QuantDevelopers/Backend/internal/presentation/controller/v1/underlying"
	"github.com/labstack/echo/v4"
)

func SetupGeneralRoutes(routerGroup *echo.Group, underlyingController *underlying.GeneralUnderlyingController) {
	underlyings := routerGroup.Group("/underlyings")
	{
		underlyings.GET("", underlyingController.GetUnderlyings)
		underlyings.GET("/:symbol_fa", underlyingController.GetUnderlyingBySymbol)
	}
}
