package mocks

import (
	"time"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	domainrepo "github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	"github.com/stretchr/testify/mock"
)

type TickRawRepositoryMock struct {
	mock.Mock
}

func NewTickRawRepositoryMock() *TickRawRepositoryMock {
	return &TickRawRepositoryMock{}
}

func (m *TickRawRepositoryMock) ReplaceDayTicks(
	db database.Database,
	instrumentType string,
	instrumentID uint,
	tradeDate time.Time,
	ticks []*entity.TickRaw,
) (int, error) {
	args := m.Called(db, instrumentType, instrumentID, tradeDate, ticks)
	return args.Int(0), args.Error(1)
}

func (m *TickRawRepositoryMock) GetByInstrumentAndDate(
	db database.Database,
	instrumentType string,
	instrumentID uint,
	tradeDate time.Time,
	offset int,
	limit int,
) ([]*entity.TickRaw, int64, error) {
	args := m.Called(db, instrumentType, instrumentID, tradeDate, offset, limit)
	var rows []*entity.TickRaw
	if args.Get(0) != nil {
		rows = args.Get(0).([]*entity.TickRaw)
	}
	return rows, int64(args.Int(1)), args.Error(2)
}

func (m *TickRawRepositoryMock) GetLatestDate(
	db database.Database,
	instrumentType string,
	instrumentID uint,
) (*time.Time, error) {
	args := m.Called(db, instrumentType, instrumentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*time.Time), args.Error(1)
}

func (m *TickRawRepositoryMock) Count(db database.Database, filter domainrepo.TickRawFilter) (int64, error) {
	args := m.Called(db, filter)
	return int64(args.Int(0)), args.Error(1)
}
