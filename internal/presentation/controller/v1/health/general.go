package health

import (
	"net/http"

	_ "github.com/QuantDevelopers/Backend/internal/application/dto/health"
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

// GetHealth returns application and PostgreSQL connectivity status.
// @Summary Service health check
// @Description Returns application health metadata and PostgreSQL database connectivity state.
// @Tags Health
// @Produce json
// @Success 200 {object} controller.SingleMessageResponse{data=healthdto.HealthResponse}
// @Failure 503 {object} controller.SingleMessageResponse{data=healthdto.HealthResponse}
// @Router /health [get]
func (hc *GeneralHealthController) GetHealth(ctx echo.Context) error {
	resp, isHealthy := hc.healthService.CheckHealth(ctx.Request().Context())
	statusCode := http.StatusOK
	if !isHealthy {
		statusCode = http.StatusServiceUnavailable
	}
	return controller.Response(ctx, statusCode, "", resp)
}
