package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/interfaces/http/dto"
	"github.com/labstack/echo/v4"
)

// UnderlyingUseCase defines the application operations used by UnderlyingHandler.
type UnderlyingUseCase interface {
	ListUnderlyings(ctx context.Context) ([]entity.UnderlyingAsset, error)
	GetUnderlyingBySymbol(ctx context.Context, symbolFA string) (*entity.UnderlyingAsset, error)
}

// UnderlyingHandler serves /api/v1/underlyings endpoints.
type UnderlyingHandler struct {
	useCase UnderlyingUseCase
	logger  *slog.Logger
}

// NewUnderlyingHandler constructs an UnderlyingHandler.
func NewUnderlyingHandler(useCase UnderlyingUseCase, logger *slog.Logger) *UnderlyingHandler {
	return &UnderlyingHandler{
		useCase: useCase,
		logger:  logger,
	}
}

// ListUnderlyings handles GET /api/v1/underlyings.
func (h *UnderlyingHandler) ListUnderlyings(c echo.Context) error {
	assets, err := h.useCase.ListUnderlyings(c.Request().Context())
	if err != nil {
		if h.logger != nil {
			h.logger.Error("failed to list underlyings", "error", err)
		}
		return c.JSON(http.StatusInternalServerError, dto.HTTPErrorResponse{
			Detail: "Failed to fetch underlying assets",
		})
	}

	items := make([]dto.UnderlyingAssetResponse, 0, len(assets))
	for _, asset := range assets {
		items = append(items, dto.FromUnderlyingEntity(asset))
	}

	return c.JSON(http.StatusOK, dto.UnderlyingListResponse{
		Total: len(items),
		Items: items,
	})
}

// GetUnderlyingBySymbol handles GET /api/v1/underlyings/:symbol_fa.
func (h *UnderlyingHandler) GetUnderlyingBySymbol(c echo.Context) error {
	symbolFA := c.Param("symbol_fa")
	if symbolFA == "" {
		return c.JSON(http.StatusBadRequest, dto.HTTPErrorResponse{
			Detail: "Underlying symbol must not be empty",
		})
	}

	asset, err := h.useCase.GetUnderlyingBySymbol(c.Request().Context(), symbolFA)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("failed to get underlying by symbol", "symbol_fa", symbolFA, "error", err)
		}
		return c.JSON(http.StatusInternalServerError, dto.HTTPErrorResponse{
			Detail: "Failed to fetch underlying asset",
		})
	}
	if asset == nil {
		return c.JSON(http.StatusNotFound, dto.HTTPErrorResponse{
			Detail: fmt.Sprintf("Underlying asset with symbol '%s' not found", symbolFA),
		})
	}

	return c.JSON(http.StatusOK, dto.FromUnderlyingEntity(*asset))
}
