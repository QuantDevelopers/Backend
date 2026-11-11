package service

import (
	"context"
	"errors"
	"testing"

	"github.com/QuantDevelopers/Backend/bootstrap"
	dbMocks "github.com/QuantDevelopers/Backend/mocks/infrastructure/database"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type HealthServiceTestSuite struct {
	suite.Suite
	server        *bootstrap.Server
	db            *dbMocks.DatabaseMock
	healthService *HealthService
}

func (s *HealthServiceTestSuite) SetupTest() {
	cfg := bootstrap.Run()
	s.server = &cfg.Env.Server
	s.db = dbMocks.NewDatabaseMock()
	s.healthService = NewHealthService(s.server, s.db)
}

func (s *HealthServiceTestSuite) TestCheckHealth() {
	s.Run("success - database connected", func() {
		s.db.On("Ping", mock.Anything).Return(nil).Once()

		resp, healthy := s.healthService.CheckHealth(context.Background())

		s.True(healthy)
		s.Equal("healthy", resp.Status)
		s.Equal("connected", resp.Database)
		s.Equal(s.server.AppName, resp.App)
		s.False(resp.Timestamp.IsZero())
		s.db.AssertExpectations(s.T())
	})

	s.Run("degraded - database unreachable", func() {
		s.db.On("Ping", mock.Anything).Return(errors.New("connection refused")).Once()

		resp, healthy := s.healthService.CheckHealth(context.Background())

		s.False(healthy)
		s.Equal("degraded", resp.Status)
		s.Equal("disconnected", resp.Database)
		s.db.AssertExpectations(s.T())
	})
}

func TestHealthServiceSuite(t *testing.T) {
	suite.Run(t, new(HealthServiceTestSuite))
}
