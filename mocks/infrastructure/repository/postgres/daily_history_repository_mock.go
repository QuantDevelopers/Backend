package mocks

import (
	"time"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	domainrepo "github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	"github.com/stretchr/testify/mock"
)

type DailyHistoryRepositoryMock struct {
	mock.Mock
}

func NewDailyHistoryRepositoryMock() *DailyHistoryRepositoryMock {
	return &DailyHistoryRepositoryMock{}
}

func (m *DailyHistoryRepositoryMock) SaveBulk(db database.Database, records []*entity.DailyHistory) (int, error) {
	args := m.Called(db, records)
	return args.Int(0), args.Error(1)
}

func (m *DailyHistoryRepositoryMock) GetByInstrument(
	db database.Database,
	instrumentType string,
	instrumentID uint,
	limit int,
) ([]*entity.DailyHistory, error) {
	args := m.Called(db, instrumentType, instrumentID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.DailyHistory), args.Error(1)
}

func (m *DailyHistoryRepositoryMock) List(
	db database.Database,
	filter domainrepo.DailyHistoryFilter,
	limit int,
) ([]*entity.DailyHistory, error) {
	args := m.Called(db, filter, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.DailyHistory), args.Error(1)
}

func (m *DailyHistoryRepositoryMock) GetTradingDates(
	db database.Database,
	instrumentType string,
	instrumentID uint,
	onlyWithTrades bool,
) ([]time.Time, error) {
	args := m.Called(db, instrumentType, instrumentID, onlyWithTrades)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]time.Time), args.Error(1)
}

func (m *DailyHistoryRepositoryMock) Count(db database.Database, filter domainrepo.DailyHistoryFilter) (int64, error) {
	args := m.Called(db, filter)
	return int64(args.Int(0)), args.Error(1)
}
