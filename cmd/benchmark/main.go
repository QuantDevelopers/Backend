// TSETMC API Cookbook Benchmark and Sample Ingestion CLI.
//
// Card: E1-1 · API Cookbook
// Empirically tests TSETMC CDN endpoints, measures request latency,
// saves real raw response samples, and benchmarks safe request rates.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	ahramInsCode  = "17914401175772326"
	optionInsCode = "19693832241143005"
	testDate      = "20261027"
)

var samplesDir = filepath.Join("data", "raw", "samples")

var browserHeaders = map[string]string{
	"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
	"Accept":     "application/json,text/plain,*/*",
	"Referer":    "https://www.tsetmc.com/",
	"Origin":     "https://www.tsetmc.com",
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 15 * time.Second}
}

func doGet(client *http.Client, url string) (int, []byte, float64, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, 0, err
	}
	for k, v := range browserHeaders {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsedMS := float64(time.Since(start).Microseconds()) / 1000.0
	if err != nil {
		return 0, nil, elapsedMS, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, elapsedMS, err
	}
	return resp.StatusCode, body, elapsedMS, nil
}

func fetchAndSaveSamples(client *http.Client) map[string]string {
	endpoints := []struct {
		filename string
		url      string
	}{
		{"sample_underlying_identity.json", fmt.Sprintf("https://cdn.tsetmc.com/api/Instrument/GetInstrumentIdentity/%s", ahramInsCode)},
		{"sample_underlying_daily.json", fmt.Sprintf("https://cdn.tsetmc.com/api/ClosingPrice/GetClosingPriceDailyList/%s/10", ahramInsCode)},
		{"sample_underlying_ticks.json", fmt.Sprintf("https://cdn.tsetmc.com/api/Trade/GetTradeHistory/%s/%s/false", ahramInsCode, testDate)},
		{"sample_option_search.json", "https://cdn.tsetmc.com/api/Instrument/GetInstrumentSearch/%D8%B6%D9%87%D8%B1%D9%85"},
		{"sample_option_daily.json", fmt.Sprintf("https://cdn.tsetmc.com/api/ClosingPrice/GetClosingPriceDailyList/%s/10", optionInsCode)},
		{"sample_option_ticks.json", fmt.Sprintf("https://cdn.tsetmc.com/api/Trade/GetTradeHistory/%s/%s/false", optionInsCode, testDate)},
	}

	results := make(map[string]string, len(endpoints))
	fmt.Println("--- 1. Fetching Real Raw Samples from TSETMC CDN ---")
	for _, ep := range endpoints {
		statusCode, body, elapsedMS, err := doGet(client, ep.url)
		if err != nil {
			fmt.Printf(" [ERROR] %s: %v\n", ep.filename, err)
			results[ep.filename] = fmt.Sprintf("Error: %v", err)
			time.Sleep(300 * time.Millisecond)
			continue
		}

		if statusCode != http.StatusOK {
			fmt.Printf(" [FAIL] %s returned status %d\n", ep.filename, statusCode)
			results[ep.filename] = fmt.Sprintf("HTTP %d", statusCode)
			time.Sleep(300 * time.Millisecond)
			continue
		}

		var payload any
		if err := json.Unmarshal(body, &payload); err != nil {
			fmt.Printf(" [ERROR] %s invalid JSON: %v\n", ep.filename, err)
			results[ep.filename] = fmt.Sprintf("Error: %v", err)
			time.Sleep(300 * time.Millisecond)
			continue
		}

		var formatted bytes.Buffer
		enc := json.NewEncoder(&formatted)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(payload); err != nil {
			fmt.Printf(" [ERROR] %s format JSON: %v\n", ep.filename, err)
			results[ep.filename] = fmt.Sprintf("Error: %v", err)
			time.Sleep(300 * time.Millisecond)
			continue
		}

		targetPath := filepath.Join(samplesDir, ep.filename)
		if err := os.WriteFile(targetPath, formatted.Bytes(), 0o644); err != nil {
			fmt.Printf(" [ERROR] %s write file: %v\n", ep.filename, err)
			results[ep.filename] = fmt.Sprintf("Error: %v", err)
			time.Sleep(300 * time.Millisecond)
			continue
		}

		sizeKB := float64(len(body)) / 1024.0
		fmt.Printf(" [OK] Saved %s (%.1f KB, latency: %.1fms)\n", ep.filename, sizeKB, elapsedMS)
		results[ep.filename] = fmt.Sprintf("OK (%.1f KB, %.1fms)", sizeKB, elapsedMS)
		time.Sleep(300 * time.Millisecond)
	}
	return results
}

type latencySummary struct {
	Count    int
	MeanMS   float64
	MedianMS float64
	MinMS    float64
	MaxMS    float64
	P95MS    float64
}

func runLatencyBenchmark(client *http.Client, nRequests int) *latencySummary {
	fmt.Printf("\n--- 2. Measuring Latency Benchmark (%d sample requests) ---\n", nRequests)
	url := fmt.Sprintf("https://cdn.tsetmc.com/api/ClosingPrice/GetClosingPriceInfo/%s", ahramInsCode)
	latencies := make([]float64, 0, nRequests)

	for i := 0; i < nRequests; i++ {
		statusCode, _, dtMS, err := doGet(client, url)
		if err != nil {
			fmt.Printf("  Req %02d: Exception %v\n", i+1, err)
		} else if statusCode == http.StatusOK {
			latencies = append(latencies, dtMS)
			fmt.Printf("  Req %02d: %.1f ms (Status: %d)\n", i+1, dtMS, statusCode)
		} else {
			fmt.Printf("  Req %02d: Failed HTTP %d\n", i+1, statusCode)
		}
		time.Sleep(350 * time.Millisecond) // ~2.8 RPS polite rate
	}

	if len(latencies) == 0 {
		return nil
	}

	sorted := append([]float64(nil), latencies...)
	sort.Float64s(sorted)

	var sum float64
	for _, v := range sorted {
		sum += v
	}
	mean := sum / float64(len(sorted))

	var median float64
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		median = (sorted[mid-1] + sorted[mid]) / 2.0
	} else {
		median = sorted[mid]
	}

	p95Idx := int(math.Ceil(0.95*float64(len(sorted)))) - 1
	if p95Idx < 0 {
		p95Idx = 0
	}
	if p95Idx >= len(sorted) {
		p95Idx = len(sorted) - 1
	}

	summary := &latencySummary{
		Count:    len(sorted),
		MeanMS:   math.Round(mean*10) / 10,
		MedianMS: math.Round(median*10) / 10,
		MinMS:    math.Round(sorted[0]*10) / 10,
		MaxMS:    math.Round(sorted[len(sorted)-1]*10) / 10,
		P95MS:    math.Round(sorted[p95Idx]*10) / 10,
	}
	fmt.Printf(
		"\nLatency Summary: Mean=%.1fms, Median=%.1fms, Min=%.1fms, Max=%.1fms\n",
		summary.MeanMS, summary.MedianMS, summary.MinMS, summary.MaxMS,
	)
	return summary
}

func main() {
	if err := os.MkdirAll(samplesDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create samples directory: %v\n", err)
		os.Exit(1)
	}

	client := newHTTPClient()
	fetchAndSaveSamples(client)
	runLatencyBenchmark(client, 15)
	fmt.Println("\nBenchmark completed successfully.")
}
