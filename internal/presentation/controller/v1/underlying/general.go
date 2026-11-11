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

// GetUnderlyings lists all underlying assets configured in the system.
// @Summary List underlying assets
// @Description Returns all underlying assets seeded or discovered in PostgreSQL.
// @Tags Underlyings
// @Produce json
// @Success 200 {object} controller.SingleMessageResponse{data=underlyingdto.UnderlyingListResponse}
// @Failure 500 {object} controller.SingleMessageResponse
// @Router /api/v1/underlyings [get]
func (uc *GeneralUnderlyingController) GetUnderlyings(ctx echo.Context) error {
	underlyings, err := uc.underlyingService.GetUnderlyings()
	if err != nil {
		panic(err)
	}
	return controller.Response(ctx, http.StatusOK, "", underlyings)
}

// GetUnderlyingBySymbol returns a single underlying asset matching the Persian symbol.
// @Summary Get underlying asset by Persian symbol
// @Description Fetches a single underlying asset identified by its normalized Persian symbol_fa.
// @Tags Underlyings
// @Produce json
// @Param symbol_fa path string true "Persian underlying symbol (e.g. اهرم)"
// @Success 200 {object} controller.SingleMessageResponse{data=underlyingdto.UnderlyingResponse}
// @Failure 400 {object} controller.SingleMessageResponse
// @Failure 404 {object} controller.SingleMessageResponse
// @Failure 500 {object} controller.SingleMessageResponse
// @Router /api/v1/underlyings/{symbol_fa} [get]
func (uc *GeneralUnderlyingController) GetUnderlyingBySymbol(ctx echo.Context) error {
	params := controller.Validated[underlyingdto.GetUnderlyingBySymbolRequest](ctx)
	asset, err := uc.underlyingService.GetUnderlyingBySymbol(params.SymbolFA)
	if err != nil {
		panic(err)
	}
	return controller.Response(ctx, http.StatusOK, "", asset)
}
