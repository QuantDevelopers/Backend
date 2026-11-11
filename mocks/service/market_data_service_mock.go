package mocks

import (
	marketdatadto "github.com/QuantDevelopers/Backend/internal/application/dto/marketdata"
	"github.com/stretchr/testify/mock"
)

type MarketDataServiceMock struct {
	mock.Mock
}

func NewMarketDataServiceMock() *MarketDataServiceMock {
	return &MarketDataServiceMock{}
}

func (m *MarketDataServiceMock) SearchInstruments(req marketdatadto.SearchInstrumentsRequest) (marketdatadto.InstrumentSearchResponse, error) {
	args := m.Called(req)
	return args.Get(0).(marketdatadto.InstrumentSearchResponse), args.Error(1)
}

func (m *MarketDataServiceMock) GetDailyHistory(req marketdatadto.GetDailyHistoryRequest) (marketdatadto.DailyHistoryResponse, error) {
	args := m.Called(req)
	return args.Get(0).(marketdatadto.DailyHistoryResponse), args.Error(1)
}

func (m *MarketDataServiceMock) GetTicks(req marketdatadto.GetTicksRequest) (marketdatadto.TicksPaginatedResponse, error) {
	args := m.Called(req)
	return args.Get(0).(marketdatadto.TicksPaginatedResponse), args.Error(1)
}
