package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/QuantDevelopers/Backend/internal/interfaces/http/dto"
	"github.com/labstack/echo/v4"
)

// DBHealthChecker verifies live database connectivity.
type DBHealthChecker interface {
	CheckDB(ctx context.Context) bool
}

// HealthHandler serves service health probes.
type HealthHandler struct {
	appName string
	checker DBHealthChecker
}

// NewHealthHandler constructs a HealthHandler.
func NewHealthHandler(appName string, checker DBHealthChecker) *HealthHandler {
	return &HealthHandler{
		appName: appName,
		checker: checker,
	}
}

// GetHealth handles GET /health and returns 200 when healthy or 503 when database is unreachable.
func (h *HealthHandler) GetHealth(c echo.Context) error {
	isConnected := h.checker != nil && h.checker.CheckDB(c.Request().Context())

	statusStr := "healthy"
	dbStatus := "connected"
	httpCode := http.StatusOK
	if !isConnected {
		statusStr = "degraded"
		dbStatus = "disconnected"
		httpCode = http.StatusServiceUnavailable
	}

	resp := dto.HealthResponse{
		Status:    statusStr,
		App:       h.appName,
		Database:  dbStatus,
		Timestamp: time.Now().UTC(),
	}
	return c.JSON(httpCode, resp)
}
