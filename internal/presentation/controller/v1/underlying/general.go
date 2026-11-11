package underlying

import (
	"net/http"

	underlyingdto "github.com/QuantDevelopers/Backend/internal/application/dto/underlying"
	"github.com/QuantDevelopers/Backend/internal/application/usecase"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller"
	"github.com/labstack/echo/v4"
)

type GeneralUnderlyingController struct {
	underlyingService usecase.UnderlyingService
}

func NewGeneralUnderlyingController(underlyingService usecase.UnderlyingService) *GeneralUnderlyingController {
	return &GeneralUnderlyingController{
		underlyingService: underlyingService,
	}
}

func (uc *GeneralUnderlyingController) GetUnderlyings(ctx echo.Context) error {
	underlyings, err := uc.underlyingService.GetUnderlyings()
	if err != nil {
		panic(err)
	}
	return controller.Response(ctx, http.StatusOK, "", underlyings)
}

func (uc *GeneralUnderlyingController) GetUnderlyingBySymbol(ctx echo.Context) error {
	params := controller.Validated[underlyingdto.GetUnderlyingBySymbolRequest](ctx)
	asset, err := uc.underlyingService.GetUnderlyingBySymbol(params.SymbolFA)
	if err != nil {
		panic(err)
	}
	return controller.Response(ctx, http.StatusOK, "", asset)
}
