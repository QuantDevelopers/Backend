package mocks

import (
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	"github.com/stretchr/testify/mock"
)

type UnderlyingRepositoryMock struct {
	mock.Mock
}

func NewUnderlyingRepositoryMock() *UnderlyingRepositoryMock {
	return &UnderlyingRepositoryMock{}
}

func (m *UnderlyingRepositoryMock) GetAll(db database.Database) ([]*entity.UnderlyingAsset, error) {
	args := m.Called(db)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.UnderlyingAsset), args.Error(1)
}

func (m *UnderlyingRepositoryMock) Search(db database.Database, query string, limit int) ([]*entity.UnderlyingAsset, error) {
	args := m.Called(db, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.UnderlyingAsset), args.Error(1)
}

func (m *UnderlyingRepositoryMock) GetBySymbol(db database.Database, symbolFA string) (*entity.UnderlyingAsset, error) {
	args := m.Called(db, symbolFA)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.UnderlyingAsset), args.Error(1)
}

func (m *UnderlyingRepositoryMock) GetByTSETMCID(db database.Database, tsetmcID string) (*entity.UnderlyingAsset, error) {
	args := m.Called(db, tsetmcID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.UnderlyingAsset), args.Error(1)
}

func (m *UnderlyingRepositoryMock) Create(db database.Database, asset *entity.UnderlyingAsset) error {
	args := m.Called(db, asset)
	return args.Error(0)
}

func (m *UnderlyingRepositoryMock) Upsert(db database.Database, asset *entity.UnderlyingAsset) error {
	args := m.Called(db, asset)
	return args.Error(0)
}
