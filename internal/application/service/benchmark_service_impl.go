package service

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	crawlerdto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	domainlogger "github.com/QuantDevelopers/Backend/internal/domain/logger"
	"golang.org/x/time/rate"
)

const defaultBenchmarkURL = "https://cdn.tsetmc.com/api/ClosingPrice/GetClosingPriceInfo/17914401175772326"

// BenchmarkService implements E1-2 safe crawl-rate sweeping and latency percentile analysis.
type BenchmarkService struct {
	httpClient *http.Client
	logger     domainlogger.Logger
}

// NewBenchmarkService constructs a BenchmarkService instance.
func NewBenchmarkService(log domainlogger.Logger) *BenchmarkService {
	return &BenchmarkService{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     log,
	}
}

func (s *BenchmarkService) RunRateSweep(
	ctx context.Context,
	opts crawlerdto.BenchmarkOptions,
) (*crawlerdto.RateBenchmarkReport, error) {
	targetURL := strings.TrimSpace(opts.TargetURL)
	if targetURL == "" {
		targetURL = defaultBenchmarkURL
	}
	rates := opts.Rates
	if len(rates) == 0 {
		rates = []float64{1.0, 2.0, 3.0, 5.0, 8.0}
	}
	sortedRates := make([]float64, 0, len(rates))
	for _, r := range rates {
		if r > 0 {
			sortedRates = append(sortedRates, r)
		}
	}
	if len(sortedRates) == 0 {
		sortedRates = []float64{1.0, 2.0, 3.0}
	}
	sort.Float64s(sortedRates)

	reqsPerRate := opts.RequestsPerRate
	if reqsPerRate <= 0 {
		reqsPerRate = 10
	}
	safetyFactor := opts.SafetyFactor
	if safetyFactor <= 0 || safetyFactor > 1 {
		safetyFactor = 0.75
	}
	maxErrorPct := opts.MaxErrorRatePct
	if maxErrorPct <= 0 {
		maxErrorPct = 5.0
	}
	maxP95MS := opts.MaxP95LatencyMS
	if maxP95MS <= 0 {
		maxP95MS = 1500.0
	}

	tiers := make([]crawlerdto.RateTierResult, 0, len(sortedRates))
	highestSafeRPS := 0.0

	for _, rps := range sortedRates {
		tier, err := s.measureTier(ctx, targetURL, rps, reqsPerRate, maxErrorPct, maxP95MS)
		if err != nil {
			return nil, err
		}
		tiers = append(tiers, tier)

		if tier.IsSafe {
			highestSafeRPS = rps
		}

		// Stop escalating immediately once rate limiting or heavy errors occur.
		if tier.RateLimitedCount > 0 || tier.ErrorRatePct > maxErrorPct {
			if s.logger != nil {
				s.logger.Warn("halting rate sweep after upstream throttling/errors",
					domainlogger.Int("rate_limited", tier.RateLimitedCount),
				)
			}
			break
		}
	}

	recommended := 1.0
	rationale := "Fallback conservative floor (1.0 RPS): no candidate tier satisfied strict error/latency bounds."
	if highestSafeRPS > 0 {
		recommended = math.Max(0.5, round2(highestSafeRPS*safetyFactor))
		rationale = fmt.Sprintf(
			"Highest healthy tier %.2f RPS with 0 throttling and P95 <= %.0f ms; applied %.0f%% safety factor -> %.2f RPS.",
			highestSafeRPS, maxP95MS, safetyFactor*100, recommended,
		)
	}

	report := &crawlerdto.RateBenchmarkReport{
		GeneratedAt:        time.Now().UTC(),
		TargetURL:          targetURL,
		SafetyFactor:       safetyFactor,
		RecommendedSafeRPS: recommended,
		Rationale:          rationale,
		Tiers:              tiers,
	}

	if strings.TrimSpace(opts.OutputDir) != "" {
		if err := os.MkdirAll(opts.OutputDir, 0o755); err != nil {
			return nil, fmt.Errorf("create benchmark output dir: %w", err)
		}
		jsonPath := filepath.Join(opts.OutputDir, "e1_2_rate_benchmark.json")
		mdPath := filepath.Join(opts.OutputDir, "e1_2_rate_benchmark.md")
		report.JSONPath = jsonPath
		report.MarkdownPath = mdPath

		if err := writeJSONPretty(jsonPath, report); err != nil {
			return nil, err
		}
		if err := os.WriteFile(mdPath, []byte(s.FormatMarkdown(report)), 0o644); err != nil {
			return nil, fmt.Errorf("write benchmark markdown %s: %w", mdPath, err)
		}
	}

	return report, nil
}

func (s *BenchmarkService) measureTier(
	ctx context.Context,
	targetURL string,
	targetRPS float64,
	requests int,
	maxErrorPct float64,
	maxP95MS float64,
) (crawlerdto.RateTierResult, error) {
	limiter := rate.NewLimiter(rate.Limit(targetRPS), 1)
	latencies := make([]float64, 0, requests)

	var okCount, rateLimitedCount, serverErrCount, transportErrCount int
	tierStart := time.Now()

	for i := 0; i < requests; i++ {
		if err := limiter.Wait(ctx); err != nil {
			return crawlerdto.RateTierResult{}, err
		}

		status, elapsedMS, err := s.probeOnce(ctx, targetURL)
		if err != nil {
			transportErrCount++
			continue
		}

		switch {
		case status == http.StatusOK:
			okCount++
			latencies = append(latencies, elapsedMS)
		case status == http.StatusTooManyRequests || status == http.StatusForbidden:
			rateLimitedCount++
		case status >= http.StatusInternalServerError:
			serverErrCount++
		default:
			serverErrCount++
		}
	}

	totalDuration := time.Since(tierStart).Seconds()
	effectiveRPS := 0.0
	if totalDuration > 0 {
		effectiveRPS = round2(float64(requests) / totalDuration)
	}

	failCount := rateLimitedCount + serverErrCount + transportErrCount
	errorRatePct := round2((float64(failCount) / float64(requests)) * 100.0)

	minMS, meanMS, p50MS, p95MS, p99MS, maxMS := summarizeLatencies(latencies)
	isSafe := rateLimitedCount == 0 && errorRatePct <= maxErrorPct && okCount > 0 && p95MS <= maxP95MS

	var notes []string
	if rateLimitedCount > 0 {
		notes = append(notes, fmt.Sprintf("throttled (%d x 429/403)", rateLimitedCount))
	}
	if p95MS > maxP95MS && okCount > 0 {
		notes = append(notes, fmt.Sprintf("high tail latency P95=%.1fms", p95MS))
	}
	if okCount == 0 {
		notes = append(notes, "no HTTP 200 responses")
	}

	return crawlerdto.RateTierResult{
		TargetRPS:           targetRPS,
		Requests:            requests,
		SuccessCount:        okCount,
		RateLimitedCount:    rateLimitedCount,
		ServerErrorCount:    serverErrCount,
		TransportErrorCount: transportErrCount,
		ErrorRatePct:        errorRatePct,
		EffectiveRPS:        effectiveRPS,
		MinMS:               minMS,
		MeanMS:              meanMS,
		P50MS:               p50MS,
		P95MS:               p95MS,
		P99MS:               p99MS,
		MaxMS:               maxMS,
		IsSafe:              isSafe,
		Notes:               strings.Join(notes, "; "),
	}, nil
}

func (s *BenchmarkService) probeOnce(ctx context.Context, targetURL string) (int, float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://www.tsetmc.com/")
	req.Header.Set("Origin", "https://www.tsetmc.com")

	start := time.Now()
	resp, err := s.httpClient.Do(req)
	elapsedMS := float64(time.Since(start).Microseconds()) / 1000.0
	if err != nil {
		return 0, elapsedMS, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode, elapsedMS, nil
}

func (s *BenchmarkService) FormatMarkdown(report *crawlerdto.RateBenchmarkReport) string {
	var b strings.Builder
	b.WriteString("# E1-2 · Rate & Latency Benchmark Report\n\n")
	fmt.Fprintf(&b, "- **Generated At (UTC):** %s\n", report.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "- **Target Endpoint:** `%s`\n", report.TargetURL)
	fmt.Fprintf(&b, "- **Recommended Safe Crawl Rate:** `%.2f RPS` (Safety Factor: `%.2f`)\n", report.RecommendedSafeRPS, report.SafetyFactor)
	fmt.Fprintf(&b, "- **Rationale:** %s\n\n", report.Rationale)
	b.WriteString("| Target RPS | Requests | HTTP 200 | 429/403 | 5xx/Err | Error % | Min (ms) | Mean (ms) | P50 (ms) | P95 (ms) | P99 (ms) | Safe |\n")
	b.WriteString("| :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |\n")
	for _, t := range report.Tiers {
		safeStr := "No"
		if t.IsSafe {
			safeStr = "Yes"
		}
		fmt.Fprintf(&b,
			"| %.2f | %d | %d | %d | %d | %.2f%% | %.1f | %.1f | %.1f | %.1f | %.1f | %s |\n",
			t.TargetRPS, t.Requests, t.SuccessCount, t.RateLimitedCount,
			t.ServerErrorCount+t.TransportErrorCount, t.ErrorRatePct,
			t.MinMS, t.MeanMS, t.P50MS, t.P95MS, t.P99MS, safeStr,
		)
	}
	return b.String()
}

func summarizeLatencies(samples []float64) (minMS, meanMS, p50MS, p95MS, p99MS, maxMS float64) {
	if len(samples) == 0 {
		return 0, 0, 0, 0, 0, 0
	}
	sorted := append([]float64(nil), samples...)
	sort.Float64s(sorted)

	var sum float64
	for _, v := range sorted {
		sum += v
	}

	return round1(sorted[0]),
		round1(sum / float64(len(sorted))),
		round1(percentile(sorted, 0.50)),
		round1(percentile(sorted, 0.95)),
		round1(percentile(sorted, 0.99)),
		round1(sorted[len(sorted)-1])
}

func percentile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(q*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
