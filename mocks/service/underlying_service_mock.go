package mocks

import (
	underlyingdto "github.com/QuantDevelopers/Backend/internal/application/dto/underlying"
	"github.com/stretchr/testify/mock"
)

type UnderlyingServiceMock struct {
	mock.Mock
}

func NewUnderlyingServiceMock() *UnderlyingServiceMock {
	return &UnderlyingServiceMock{}
}

func (m *UnderlyingServiceMock) GetUnderlyings() (underlyingdto.UnderlyingListResponse, error) {
	args := m.Called()
	return args.Get(0).(underlyingdto.UnderlyingListResponse), args.Error(1)
}

func (m *UnderlyingServiceMock) GetUnderlyingBySymbol(symbolFA string) (underlyingdto.UnderlyingResponse, error) {
	args := m.Called(symbolFA)
	return args.Get(0).(underlyingdto.UnderlyingResponse), args.Error(1)
}
