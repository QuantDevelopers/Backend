package mocks

import (
	"context"

	healthdto "github.com/QuantDevelopers/Backend/internal/application/dto/health"
	"github.com/stretchr/testify/mock"
)

type HealthServiceMock struct {
	mock.Mock
}

func NewHealthServiceMock() *HealthServiceMock {
	return &HealthServiceMock{}
}

func (m *HealthServiceMock) CheckHealth(ctx context.Context) (healthdto.HealthResponse, bool) {
	args := m.Called(ctx)
	return args.Get(0).(healthdto.HealthResponse), args.Bool(1)
}
