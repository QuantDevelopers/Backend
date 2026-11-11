package container

import (
	"context"
	"log/slog"

	"github.com/QuantDevelopers/Backend/internal/application/service"
	"github.com/QuantDevelopers/Backend/internal/config"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/persistence/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Container wires configuration, database pools, repositories, and application services.
type Container struct {
	Settings          config.Settings
	Logger            *slog.Logger
	DBPool            *pgxpool.Pool
	UnderlyingService *service.UnderlyingService
	OptionService     *service.OptionService
}

// New builds the dependency container using an established PostgreSQL connection pool.
func New(settings config.Settings, logger *slog.Logger, pool *pgxpool.Pool) *Container {
	underlyingRepo := postgres.NewUnderlyingRepository(pool)
	optionRepo := postgres.NewOptionContractRepository(pool)

	return &Container{
		Settings:          settings,
		Logger:            logger,
		DBPool:            pool,
		UnderlyingService: service.NewUnderlyingService(underlyingRepo),
		OptionService:     service.NewOptionService(optionRepo),
	}
}

// CheckDB verifies live PostgreSQL connectivity for the health endpoint.
func (c *Container) CheckDB(ctx context.Context) bool {
	return database.CheckConnection(ctx, c.DBPool)
}
