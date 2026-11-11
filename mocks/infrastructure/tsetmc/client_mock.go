package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type TSETMCClientMock struct {
	mock.Mock
}

func NewTSETMCClientMock() *TSETMCClientMock {
	return &TSETMCClientMock{}
}

func (m *TSETMCClientMock) Get(ctx context.Context, url string) ([]byte, int, error) {
	args := m.Called(ctx, url)
	var body []byte
	if args.Get(0) != nil {
		body = args.Get(0).([]byte)
	}
	return body, args.Int(1), args.Error(2)
}

func (m *TSETMCClientMock) ConsecutiveErrors() int {
	return m.Called().Int(0)
}

func (m *TSETMCClientMock) ResetCircuitBreaker() {
	m.Called()
}
