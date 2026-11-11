package usecase

import (
	"context"

	crawlerdto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
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
