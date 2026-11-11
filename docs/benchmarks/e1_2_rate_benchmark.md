# E1-2 · Rate & Latency Benchmark Report

- **Generated At (UTC):** 2026-11-11T19:59:15Z
- **Target Endpoint:** `https://cdn.tsetmc.com/api/ClosingPrice/GetClosingPriceInfo/17914401175772326`
- **Recommended Safe Crawl Rate:** `3.75 RPS` (Safety Factor: `0.75`)
- **Rationale:** Highest healthy tier 5.00 RPS with 0 throttling and P95 <= 1500 ms; applied 75% safety factor -> 3.75 RPS.

| Target RPS | Requests | HTTP 200 | 429/403 | 5xx/Err | Error % | Min (ms) | Mean (ms) | P50 (ms) | P95 (ms) | P99 (ms) | Safe |
| :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| 1.00 | 6 | 6 | 0 | 0 | 0.00% | 34.0 | 266.5 | 42.0 | 1362.0 | 1362.0 | Yes |
| 2.00 | 6 | 6 | 0 | 0 | 0.00% | 43.3 | 77.0 | 83.3 | 95.8 | 95.8 | Yes |
| 3.00 | 6 | 6 | 0 | 0 | 0.00% | 38.2 | 92.0 | 80.4 | 184.3 | 184.3 | Yes |
| 5.00 | 6 | 6 | 0 | 0 | 0.00% | 58.5 | 87.3 | 82.3 | 127.1 | 127.1 | Yes |
