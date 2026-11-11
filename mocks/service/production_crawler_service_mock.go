package mocks

import (
	"context"

	crawlerdto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	"github.com/stretchr/testify/mock"
)

type ProductionCrawlerServiceMock struct {
	mock.Mock
}

func NewProductionCrawlerServiceMock() *ProductionCrawlerServiceMock {
	return &ProductionCrawlerServiceMock{}
}

func (m *ProductionCrawlerServiceMock) BuildQueue(opts crawlerdto.ProductionCrawlOptions) ([]crawlerdto.TickCrawlTask, int, error) {
	args := m.Called(opts)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]crawlerdto.TickCrawlTask), args.Int(1), args.Error(2)
}

func (m *ProductionCrawlerServiceMock) Run(ctx context.Context, opts crawlerdto.ProductionCrawlOptions) (*crawlerdto.ProductionCrawlReport, error) {
	args := m.Called(ctx, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*crawlerdto.ProductionCrawlReport), args.Error(1)
}

func (m *ProductionCrawlerServiceMock) GetStatus(dataDir string) (*crawlerdto.CrawlStatusResponse, error) {
	args := m.Called(dataDir)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*crawlerdto.CrawlStatusResponse), args.Error(1)
}
