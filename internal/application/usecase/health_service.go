package usecase

import (
	"context"

	healthdto "github.com/QuantDevelopers/Backend/internal/application/dto/health"
)

type HealthService interface {
	CheckHealth(ctx context.Context) (healthdto.HealthResponse, bool)
}
