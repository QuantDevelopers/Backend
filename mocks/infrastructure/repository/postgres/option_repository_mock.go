package mocks

import (
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	domainrepo "github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	"github.com/stretchr/testify/mock"
)

type OptionContractRepositoryMock struct {
	mock.Mock
}

func NewOptionContractRepositoryMock() *OptionContractRepositoryMock {
	return &OptionContractRepositoryMock{}
}

func (m *OptionContractRepositoryMock) GetAll(db database.Database, filter domainrepo.OptionFilter) ([]*entity.OptionContract, error) {
	args := m.Called(db, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.OptionContract), args.Error(1)
}

func (m *OptionContractRepositoryMock) ListPaginated(db database.Database, filter domainrepo.OptionQueryFilter) ([]*entity.OptionContract, int64, error) {
	args := m.Called(db, filter)
	var rows []*entity.OptionContract
	if args.Get(0) != nil {
		rows = args.Get(0).([]*entity.OptionContract)
	}
	return rows, int64(args.Int(1)), args.Error(2)
}

func (m *OptionContractRepositoryMock) GetBySymbol(db database.Database, symbolFA string) (*entity.OptionContract, error) {
	args := m.Called(db, symbolFA)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.OptionContract), args.Error(1)
}

func (m *OptionContractRepositoryMock) Save(db database.Database, contract *entity.OptionContract) error {
	args := m.Called(db, contract)
	return args.Error(0)
}

func (m *OptionContractRepositoryMock) SaveBulk(db database.Database, contracts []*entity.OptionContract) (int, error) {
	args := m.Called(db, contracts)
	return args.Int(0), args.Error(1)
}
