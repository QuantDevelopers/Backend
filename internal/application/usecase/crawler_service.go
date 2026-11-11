package usecase

import (
	"context"

	crawlerdto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
)

// DiscoveryService executes E1-3 active option contract and underlying discovery.
type DiscoveryService interface {
	ParseMarketWatch(raw []byte) ([]crawlerdto.DiscoveredUnderlying, []crawlerdto.DiscoveredContract, error)
	Discover(ctx context.Context, opts crawlerdto.DiscoveryOptions) (*crawlerdto.DiscoveryResult, error)
}

// TickIngestorService executes E2-2 raw-first tick ingestion, manifest writing, and cleaning.
type TickIngestorService interface {
	ParseTickPayload(raw []byte, insCode, dateStr string, contractSize int) ([]crawlerdto.CleanTickRecord, error)
	IngestSample(ctx context.Context, opts crawlerdto.TickIngestOptions) (*crawlerdto.TickIngestResult, error)
}

// BenchmarkService executes E1-2 multi-rate latency and safe crawl rate benchmarking.
type BenchmarkService interface {
	RunRateSweep(ctx context.Context, opts crawlerdto.BenchmarkOptions) (*crawlerdto.RateBenchmarkReport, error)
	FormatMarkdown(report *crawlerdto.RateBenchmarkReport) string
}

// DailyHistoryCrawlerService executes E2-1 daily OHLCV crawling and PostgreSQL persistence.
type DailyHistoryCrawlerService interface {
	ParseDailyPayload(raw []byte, instrumentType string, instrumentID uint) ([]*entity.DailyHistory, error)
	Crawl(ctx context.Context, opts crawlerdto.DailyHistoryCrawlOptions) (*crawlerdto.DailyHistoryCrawlResult, error)
}
