package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/QuantDevelopers/Backend/bootstrap"
	crawlerdto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	infralogger "github.com/QuantDevelopers/Backend/internal/infrastructure/logger"
	"github.com/stretchr/testify/suite"
)

type BenchmarkServiceTestSuite struct {
	suite.Suite
	service *BenchmarkService
}

func (s *BenchmarkServiceTestSuite) SetupTest() {
	cfg := bootstrap.Run()
	log, err := infralogger.NewLogger(&bootstrap.Logger{
		LogLevel:      "error",
		ConsoleOutput: "false",
	}, cfg.Constants)
	s.Require().NoError(err)

	s.service = NewBenchmarkService(log)
}

func (s *BenchmarkServiceTestSuite) TestRunRateSweepStopsOnThrottleAndComputesSafeRPS() {
	var callNum atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callNum.Add(1)
		// First two RPS tiers (2 * 4 = 8 requests) succeed; third tier hits HTTP 429.
		if n > 8 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"closingPriceInfo":{}}`))
	}))
	defer srv.Close()

	outDir := s.T().TempDir()
	report, err := s.service.RunRateSweep(context.Background(), crawlerdto.BenchmarkOptions{
		TargetURL:       srv.URL,
		Rates:           []float64{20.0, 40.0, 80.0, 120.0},
		RequestsPerRate: 4,
		SafetyFactor:    0.75,
		OutputDir:       outDir,
	})
	s.Require().NoError(err)

	// Stopped right after 80.0 RPS hit 429; 120.0 RPS must never execute.
	s.Require().Len(report.Tiers, 3)
	s.True(report.Tiers[0].IsSafe)
	s.True(report.Tiers[1].IsSafe)
	s.False(report.Tiers[2].IsSafe)
	s.Equal(4, report.Tiers[2].RateLimitedCount)

	// 40.0 * 0.75 = 30.0
	s.InDelta(30.0, report.RecommendedSafeRPS, 0.01)
	s.FileExists(report.JSONPath)
	s.FileExists(report.MarkdownPath)

	md := s.service.FormatMarkdown(report)
	s.Contains(md, "Recommended Safe Crawl Rate")
	s.Contains(md, "| 40.00 |")
}

func (s *BenchmarkServiceTestSuite) TestAllFailingTiersUseConservativeFallback() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	report, err := s.service.RunRateSweep(context.Background(), crawlerdto.BenchmarkOptions{
		TargetURL:       srv.URL,
		Rates:           []float64{25.0, 50.0},
		RequestsPerRate: 2,
	})
	s.Require().NoError(err)
	s.Len(report.Tiers, 1)
	s.False(report.Tiers[0].IsSafe)
	s.InDelta(1.0, report.RecommendedSafeRPS, 0.01)
}

func TestBenchmarkServiceSuite(t *testing.T) {
	suite.Run(t, new(BenchmarkServiceTestSuite))
}
