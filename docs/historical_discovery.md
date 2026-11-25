# E1-5: independent historical instrument discovery

Full discovery now combines the current option watch, archived whole-market daily
snapshots and the existing registry. The historical scan can discover an expired
option AND a base instrument that are absent from both current watch and database.
It does not restrict historical requests to today's underlying names.

## Live execution

First run the server's normal `--seed-only` initialization. It also migrates the
legacy unique ticker/ISIN indexes to ordinary indexes; `tsetmc_id` remains unique.
Symbols can be reused across instruments/expiries. Upserts use exchange identity,
and symbol-only API reads prefer the active/latest record. Database rows and
historical ticks keep their internal instrument IDs.

```bash
go run ./cmd/app --seed-only
go run ./cmd/crawler -mode=full-discovery -persist-db \
  -start-date=20241028 -end-date=20261028 -timeout=30m
```

- Range bounds are Gregorian YYYYMMDD, inclusive. Without bounds, live mode scans
  two years through yesterday, plus today's current option watch. A full range
  scans **every calendar day**, including empty snapshots; it does not treat a
  short sample calendar as historical coverage.
- Alternatively, `-dates=20241026,20241027` scans exactly those snapshots. Do not
  combine explicit dates with range bounds. Coverage of selected snapshots is
  not evidence of coverage of every day between them.
- `-history-cache=PATH` overrides the default `<registry-dir>/history_cache`.
  Each fetched response has a URL-bound SHA-256 manifest. Restarting reuses
  validated responses; `-force` refreshes history. Corrupt cache is downloaded
  again. Offline snapshots are never reused as live cache.
- The shared TSETMC client enforces the existing rate/retry policy for archive
  and exact underlying searches too. Any failed day, invalid option metadata or
  ambiguous base identity fails the run; the final registry is not published
  from a partial archive scan. Valid daily cache survives for restart.

The archive endpoint is
`ClosingPrice/GetInstrmentsHistoryInDay/{YYYYMMDD}` and the response array is
`closingPriceDailyHistoryWithInstDetails`. Supported entries have flat instrument
identity fields or an `instrument` object plus the top-level `insCode`. Codes
are decoded without floating-point conversion. These adapters reject unexpected
shapes rather than treating an upstream error as an empty market.

## Metadata and audit

Option names are parsed from the right into base symbol, strike and expiry.
Persian/Arabic digits, CALL/PUT names and Gregorian/Jalali expiry dates are
supported. Jalali conversion validates leap days, rather than replacing the
calendar year arithmetically. The base is resolved from that day's market
snapshot first, then known identities, then an exact instrument search.
An ambiguous search is an error; its first result is never accepted implicitly.

Registry JSON includes `insCode`, `isin`, `instrument_id` (the upstream identifier),
asset class, option side, underlying code, strike, expiry and provenance.
`begin_date` is not invented from the first snapshot containing an instrument.
Current-watch metadata takes precedence over archive-derived metadata.

Some historical rows do not provide contract size. Such contracts explicitly
carry `contract_size_estimated:true` with the existing project default of 1000;
they are also counted in `estimated_contract_sizes`. Supply verified overrides:

```json
{"19693832241143005":1000,"123456789":500}
```

```bash
go run ./cmd/crawler -mode=full-discovery -persist-db \
  -start-date=20241028 -end-date=20261028 -contract-sizes=contract_sizes.json
```

A completed registry scan does not certify missing upstream metadata or actual
tick coverage. Resolve estimated sizes before relying on notional calculations.
The upstream may omit contracts that never appear in a daily snapshot; the
summary describes the requested snapshot scan, not an exchange-certified list
of every contract ever issued.

`registry_summary.json` now records `history_dates_completed`, range bounds,
`historical_options_added` and `estimated_contract_sizes`. A failed run leaves
any previously published summary unchanged. Check its range/time before treating
it as the result of the attempted run.

## Offline execution and tests

```bash
go run ./cmd/crawler -mode=full-discovery -offline \
  -fixture=watch.json -dates=20241026,20241027 \
  -history-fixtures=fixtures/history -registry-dir=/tmp/history-registry
```

Provide `<YYYYMMDD>.json` for each requested day. If a base is absent from all
snapshots, provide `search_<normalized-base-symbol>.json` with `instrumentSearch`.
Offline mode without explicit historical dates/range retains the watch-only demo
and reports zero archive dates; it cannot imply historical completion.

Regression tests cover unknown expired contracts/bases with no database,
large numeric codes, snapshot deduplication, exact search, Jalali leap years,
cache reuse/corruption, malformed responses and failure before final publication.
PostgreSQL integration tests cover two exchange identities sharing a ticker and
idempotent reimport. Run the full suite with PostgreSQL via `TEST_DATABASE_URL`.
Live TSETMC/schema verification and PostgreSQL integration remain explicit
validation steps when those services are unavailable in the development sandbox.
