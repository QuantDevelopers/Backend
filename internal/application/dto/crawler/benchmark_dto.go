package crawlerdto

import "time"

// BenchmarkOptions configures an E1-2 multi-rate latency and safety sweep.
type BenchmarkOptions struct {
	TargetURL       string
	Rates           []float64
	RequestsPerRate int
	SafetyFactor    float64
	MaxErrorRatePct float64
	MaxP95LatencyMS float64
	OutputDir       string
}

// RateTierResult captures latency percentiles, throughput, and error rates at a candidate RPS.
type RateTierResult struct {
	TargetRPS           float64 `json:"target_rps"`
	Requests            int     `json:"requests"`
	SuccessCount        int     `json:"success_count"`
	RateLimitedCount    int     `json:"rate_limited_count"`
	ServerErrorCount    int     `json:"server_error_count"`
	TransportErrorCount int     `json:"transport_error_count"`
	ErrorRatePct        float64 `json:"error_rate_pct"`
	EffectiveRPS        float64 `json:"effective_rps"`
	MinMS               float64 `json:"min_ms"`
	MeanMS              float64 `json:"mean_ms"`
	P50MS               float64 `json:"p50_ms"`
	P95MS               float64 `json:"p95_ms"`
	P99MS               float64 `json:"p99_ms"`
	MaxMS               float64 `json:"max_ms"`
	IsSafe              bool    `json:"is_safe"`
	Notes               string  `json:"notes,omitempty"`
}

// RateBenchmarkReport summarizes E1-2 sweep results and the recommended safe crawl rate.
type RateBenchmarkReport struct {
	GeneratedAt        time.Time        `json:"generated_at"`
	TargetURL          string           `json:"target_url"`
	SafetyFactor       float64          `json:"safety_factor"`
	RecommendedSafeRPS float64          `json:"recommended_safe_rps"`
	Rationale          string           `json:"rationale"`
	Tiers              []RateTierResult `json:"tiers"`
	JSONPath           string           `json:"json_path,omitempty"`
	MarkdownPath       string           `json:"markdown_path,omitempty"`
}
