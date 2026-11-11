package option

import (
	"net/http"

	optiondto "github.com/QuantDevelopers/Backend/internal/application/dto/option"
	"github.com/QuantDevelopers/Backend/internal/application/usecase"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller"
	"github.com/labstack/echo/v4"
)

type GeneralOptionController struct {
	optionService usecase.OptionService
}

func NewGeneralOptionController(optionService usecase.OptionService) *GeneralOptionController {
	return &GeneralOptionController{
		optionService: optionService,
	}
}

// GetOptions returns filtered and paginated active/historical option contracts.
// @Summary List option contracts
// @Description Lists option contracts with optional underlying, contract_type, active status, expiry, search query, and pagination filters.
// @Tags Options
// @Produce json
// @Param underlying_id query int false "Filter by underlying asset ID"
// @Param symbol_ua query string false "Filter by underlying Persian symbol (e.g. اهرم)"
// @Param contract_type query string false "CALL or PUT"
// @Param is_active query bool false "Active contracts only"
// @Param expiration_date query string false "Expiration date (YYYY-MM-DD or YYYYMMDD)"
// @Param q query string false "Search symbol, name, TSETMC ID, or ISIN"
// @Param page query int false "Page number (default 1)"
// @Param page_size query int false "Items per page (default 25, max 200)"
// @Success 200 {object} controller.SingleMessageResponse{data=optiondto.OptionListPaginatedResponse}
// @Failure 400 {object} controller.SingleMessageResponse
// @Failure 404 {object} controller.SingleMessageResponse
// @Router /api/v1/options [get]
func (oc *GeneralOptionController) GetOptions(ctx echo.Context) error {
	params := controller.Validated[optiondto.ListOptionsQueryRequest](ctx)
	res, err := oc.optionService.ListOptionsPaginated(params)
	if err != nil {
		panic(err)
	}
	return controller.Response(ctx, http.StatusOK, "", res)
}

// GetOptionBySymbol returns detailed metadata for a single option contract.
// @Summary Get option contract by Persian symbol
// @Description Fetches detailed contract metadata identified by normalized Persian symbol_fa.
// @Tags Options
// @Produce json
// @Param symbol_fa path string true "Persian option symbol (e.g. ضهرم1033)"
// @Success 200 {object} controller.SingleMessageResponse{data=optiondto.OptionResponse}
// @Failure 400 {object} controller.SingleMessageResponse
// @Failure 404 {object} controller.SingleMessageResponse
// @Router /api/v1/options/{symbol_fa} [get]
func (oc *GeneralOptionController) GetOptionBySymbol(ctx echo.Context) error {
	params := controller.Validated[optiondto.GetOptionBySymbolRequest](ctx)
	res, err := oc.optionService.GetOptionBySymbol(params.SymbolFA)
	if err != nil {
		panic(err)
	}
	return controller.Response(ctx, http.StatusOK, "", res)
}

// GetOptionChain returns Call/Put strike pairs grouped by expiration date for an underlying asset.
// @Summary Get underlying option chain matrix
// @Description Returns paired Call/Put contracts grouped by expiration date and strike price, including moneyness relative to spot price.
// @Tags Options
// @Produce json
// @Param symbol_fa path string true "Persian underlying symbol (e.g. اهرم)"
// @Param expiration_date query string false "Filter chain to one expiration date (YYYY-MM-DD or YYYYMMDD)"
// @Param spot_price query number false "Optional override underlying spot price (defaults to latest daily close)"
// @Success 200 {object} controller.SingleMessageResponse{data=optiondto.OptionChainResponse}
// @Failure 400 {object} controller.SingleMessageResponse
// @Failure 404 {object} controller.SingleMessageResponse
// @Router /api/v1/underlyings/{symbol_fa}/option-chain [get]
func (oc *GeneralOptionController) GetOptionChain(ctx echo.Context) error {
	params := controller.Validated[optiondto.GetOptionChainRequest](ctx)
	res, err := oc.optionService.GetOptionChain(params)
	if err != nil {
		panic(err)
	}
	return controller.Response(ctx, http.StatusOK, "", res)
}
