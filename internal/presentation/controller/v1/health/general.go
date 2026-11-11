package health

import (
	"net/http"

	"github.com/QuantDevelopers/Backend/internal/application/usecase"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller"
	"github.com/labstack/echo/v4"
)

type GeneralHealthController struct {
	healthService usecase.HealthService
}

func NewGeneralHealthController(healthService usecase.HealthService) *GeneralHealthController {
	return &GeneralHealthController{
		healthService: healthService,
	}
}

func (hc *GeneralHealthController) GetHealth(ctx echo.Context) error {
	resp, isHealthy := hc.healthService.CheckHealth(ctx.Request().Context())
	statusCode := http.StatusOK
	if !isHealthy {
		statusCode = http.StatusServiceUnavailable
	}
	return controller.Response(ctx, statusCode, "", resp)
}
