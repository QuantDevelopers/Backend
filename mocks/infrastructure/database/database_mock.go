package mocks

import (
	"context"

	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type DatabaseMock struct {
	mock.Mock
}

func NewDatabaseMock() *DatabaseMock {
	return &DatabaseMock{}
}

func (d *DatabaseMock) GetDB() *gorm.DB {
	args := d.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*gorm.DB)
}

func (d *DatabaseMock) WithTransaction(fn func(database.Database) error) error {
	args := d.Called(fn)
	return args.Error(0)
}

func (d *DatabaseMock) Ping(ctx context.Context) error {
	args := d.Called(ctx)
	return args.Error(0)
}
