"""TSETMC API Cookbook Benchmark and Sample Ingestion Script.

Card: E1-1 · API Cookbook
Empirically tests TSETMC CDN endpoints, measures request latency,
saves real raw response samples, and benchmarks safe request rates.
"""
import json
import statistics
import time
from pathlib import Path

import httpx

HEADERS = {
    "User-Agent": (
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) "
        "Chrome/126.0.0.0 Safari/537.36"
    ),
    "Accept": "application/json,text/plain,*/*",
    "Referer": "https://www.tsetmc.com/",
    "Origin": "https://www.tsetmc.com",
}

SAMPLES_DIR = Path("data/raw/samples")
SAMPLES_DIR.mkdir(parents=True, exist_ok=True)

# Key test instruments:
# Ahram Underlying insCode: 17914401175772326
# Ahram Call Option (ضهرم1033) insCode: 19693832241143005
# Test trading date: 20261027 (Tuesday) / 20261028 (Wednesday)
AHRAM_INS_CODE = "17914401175772326"
OPTION_INS_CODE = "19693832241143005"
TEST_DATE = "20261027"


def fetch_and_save_samples(client: httpx.Client) -> dict[str, str]:
    endpoints = {
        "sample_underlying_identity.json": f"https://cdn.tsetmc.com/api/Instrument/GetInstrumentIdentity/{AHRAM_INS_CODE}",
        "sample_underlying_daily.json": f"https://cdn.tsetmc.com/api/ClosingPrice/GetClosingPriceDailyList/{AHRAM_INS_CODE}/10",
        "sample_underlying_ticks.json": f"https://cdn.tsetmc.com/api/Trade/GetTradeHistory/{AHRAM_INS_CODE}/{TEST_DATE}/false",
        "sample_option_search.json": "https://cdn.tsetmc.com/api/Instrument/GetInstrumentSearch/%D8%B6%D9%87%D8%B1%D9%85",
        "sample_option_daily.json": f"https://cdn.tsetmc.com/api/ClosingPrice/GetClosingPriceDailyList/{OPTION_INS_CODE}/10",
        "sample_option_ticks.json": f"https://cdn.tsetmc.com/api/Trade/GetTradeHistory/{OPTION_INS_CODE}/{TEST_DATE}/false",
    }

    results = {}
    print("--- 1. Fetching Real Raw Samples from TSETMC CDN ---")
    for filename, url in endpoints.items():
        try:
            start_t = time.perf_counter()
            r = client.get(url)
            elapsed_ms = (time.perf_counter() - start_t) * 1000
            if r.status_code == 200:
                payload = r.json()
                filepath = SAMPLES_DIR / filename
                with open(filepath, "w", encoding="utf-8") as f:
                    json.dump(payload, f, ensure_ascii=False, indent=2)
                size_kb = len(r.text) / 1024
                print(f" [OK] Saved {filename} ({size_kb:.1f} KB, latency: {elapsed_ms:.1f}ms)")
                results[filename] = f"OK ({size_kb:.1f} KB, {elapsed_ms:.1f}ms)"
            else:
                print(f" [FAIL] {filename} returned status {r.status_code}")
                results[filename] = f"HTTP {r.status_code}"
        except Exception as e:
            print(f" [ERROR] {filename}: {e}")
            results[filename] = f"Error: {e}"
        time.sleep(0.3)  # Gentle delay
    return results


def run_latency_benchmark(client: httpx.Client, n_requests: int = 15) -> dict:
    print(f"\n--- 2. Measuring Latency Benchmark ({n_requests} sample requests) ---")
    url = f"https://cdn.tsetmc.com/api/ClosingPrice/GetClosingPriceInfo/{AHRAM_INS_CODE}"
    latencies = []

    for i in range(n_requests):
        t0 = time.perf_counter()
        try:
            resp = client.get(url)
            t1 = time.perf_counter()
            if resp.status_code == 200:
                dt_ms = (t1 - t0) * 1000
                latencies.append(dt_ms)
                print(f"  Req {i+1:02d}: {dt_ms:.1f} ms (Status: {resp.status_code})")
            else:
                print(f"  Req {i+1:02d}: Failed HTTP {resp.status_code}")
        except Exception as ex:
            print(f"  Req {i+1:02d}: Exception {ex}")
        time.sleep(0.35)  # ~2.8 RPS (Safe polite rate)

    if not latencies:
        return {}

    summary = {
        "count": len(latencies),
        "mean_ms": round(statistics.mean(latencies), 1),
        "median_ms": round(statistics.median(latencies), 1),
        "min_ms": round(min(latencies), 1),
        "max_ms": round(max(latencies), 1),
        "p95_ms": round(statistics.quantiles(latencies, n=20)[18], 1) if len(latencies) >= 20 else round(max(latencies), 1),
    }
    print(f"\nLatency Summary: Mean={summary['mean_ms']}ms, Median={summary['median_ms']}ms, Min={summary['min_ms']}ms, Max={summary['max_ms']}ms")
    return summary


def main():
    with httpx.Client(headers=HEADERS, timeout=15.0, follow_redirects=True) as client:
        fetch_and_save_samples(client)
        run_latency_benchmark(client, n_requests=15)
        print("\nBenchmark completed successfully.")


if __name__ == "__main__":
    main()
