package mocks

import (
	optiondto "github.com/QuantDevelopers/Backend/internal/application/dto/option"
	"github.com/stretchr/testify/mock"
)

type OptionServiceMock struct {
	mock.Mock
}

func NewOptionServiceMock() *OptionServiceMock {
	return &OptionServiceMock{}
}

func (m *OptionServiceMock) ListOptions(req optiondto.ListOptionsRequest) ([]optiondto.OptionResponse, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]optiondto.OptionResponse), args.Error(1)
}

func (m *OptionServiceMock) ListOptionsPaginated(req optiondto.ListOptionsQueryRequest) (optiondto.OptionListPaginatedResponse, error) {
	args := m.Called(req)
	return args.Get(0).(optiondto.OptionListPaginatedResponse), args.Error(1)
}

func (m *OptionServiceMock) GetOptionBySymbol(symbolFA string) (optiondto.OptionResponse, error) {
	args := m.Called(symbolFA)
	return args.Get(0).(optiondto.OptionResponse), args.Error(1)
}

func (m *OptionServiceMock) GetOptionChain(req optiondto.GetOptionChainRequest) (optiondto.OptionChainResponse, error) {
	args := m.Called(req)
	return args.Get(0).(optiondto.OptionChainResponse), args.Error(1)
}

func (m *OptionServiceMock) CreateOrUpdateOption(req optiondto.SaveOptionRequest) (optiondto.OptionResponse, error) {
	args := m.Called(req)
	return args.Get(0).(optiondto.OptionResponse), args.Error(1)
}
