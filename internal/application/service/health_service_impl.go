package service

import (
	"context"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	healthdto "github.com/QuantDevelopers/Backend/internal/application/dto/health"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

type HealthService struct {
	server *bootstrap.Server
	db     database.Database
}

func NewHealthService(server *bootstrap.Server, db database.Database) *HealthService {
	return &HealthService{
		server: server,
		db:     db,
	}
}

func (s *HealthService) CheckHealth(ctx context.Context) (healthdto.HealthResponse, bool) {
	isHealthy := s.db != nil && s.db.Ping(ctx) == nil

	status := "healthy"
	dbStatus := "connected"
	if !isHealthy {
		status = "degraded"
		dbStatus = "disconnected"
	}

	appName := "QuantBackend"
	if s.server != nil && s.server.AppName != "" {
		appName = s.server.AppName
	}

	return healthdto.HealthResponse{
		Status:    status,
		App:       appName,
		Database:  dbStatus,
		Timestamp: time.Now().UTC(),
	}, isHealthy
}
