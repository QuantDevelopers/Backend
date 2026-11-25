# Backend stabilization

This pass fixes defects in the existing backend. It is not evidence of complete
market data coverage or completion of every project backlog item.

## Corrected behavior

- Live discovery, daily crawling and tick crawling no longer substitute test
  fixtures after an upstream error. Fixtures require explicit offline mode.
- JSON responses without the expected array fail rather than count as empty
  trading days. Tick dates, times, sequences and numeric instrument paths are
  validated before completion markers are written.
- Files are published through temporary files and atomic rename. The tick
  manifest is published last. Version-2 manifests record offline provenance;
  legacy manifests are not trusted for resume.
- Disk-only resume skips validated completed days. With `-persist-db`, resume
  replays verified raw cache through the transactional loader, avoiding network
  requests while repairing missing/previously failed database writes. An offline
  cache is never accepted as evidence for a live crawl. `replayed_cache` reports
  such replays; they are not counted as skipped tasks.
- Production status publishes running/completed/failed state and pending tasks.
  Cancellation, failed tasks, circuit breaking and status-write failures return
  errors instead of reporting successful completion. Status is updated during
  long runs (at most once per second, plus start/end).
- Full-history target selection includes expired contracts and inactive
  underlyings. Explicit dates and derived dates are limited to contract lifetime.
  The requested date order is honored. Filtering all explicit dates out produces
  an empty queue, not an unexpected fallback crawl.
- Instrument-specific database history is the default date source. A calendar
  is used only when explicitly supplied, or for the offline example. The checked-in
  calendar is a short example, not a comprehensive market calendar. Invalid
  calendar dates and database history errors fail visibly. `-days-back` remains
  a cap in full mode when explicitly supplied.
- Live daily/production modes require a persisted instrument registry. They no
  longer silently select two hard-coded example instruments without a database.
- Expiry is determined by date, not absence in a snapshot or zero remaining days.
  Expiry-day contracts remain active for that day. Known future contracts absent
  from the current watch are retained; their underlying mapping is preloaded.
  Historical registry query errors propagate. `include-expired=false` excludes
  expired output. Missing persistence mappings fail visibly.
- Bulk option persistence is a real upsert. Explicit boolean values are preserved
  despite the GORM `default:true` tag; single-save false values are preserved too.
- Database construction no longer caches an initialization failure globally or
  shares the wrong connection configuration. It retries bounded pings on one
  pool and closes that pool on failure.
- Pagination overflow and inverted daily date ranges are rejected. Caller
  cancellation is not counted as an upstream circuit-breaker failure; retry
  backoff is bounded.
- Native COPY rejects use inside an enclosing GORM transaction instead of
  silently committing on another pooled connection. Rollback uses an independent
  bounded context, including after caller cancellation. Failed COPY reports zero
  committed rows. Partition conversion transfers sequence ownership to the new
  parents, so retained backups no longer own the live ID sequences.

## Running the corrected flow

Initialize schema normally, then discover instruments and fetch daily histories:

```bash
go run ./cmd/crawler -mode=full-discovery -persist-db
go run ./cmd/crawler -mode=daily-history -persist-db
go run ./cmd/crawler -mode=full-crawl -persist-db -resume=true
```

Use `-start-date=YYYYMMDD`, `-end-date=YYYYMMDD`, `-days-back=N`, or `-max-tasks=N`
to bound operational runs. Do not use `-offline` to claim real coverage.
The processing command consumes the clean data pipeline described in
[data_pipeline.md](data_pipeline.md).

## Validation and remaining release gates

The repository contains regression tests for live-fetch failures, malformed
payloads, unsafe identifiers, calendar validation, explicit-date filtering,
expiry-day classification, absent future contracts, historical target lifetime,
database-error propagation, cache replay and cancellation. PostgreSQL integration
coverage additionally checks active-to-expired upserts and COPY/partition/As-of
behavior.

Local validation passes build, vet, lint and the race suite excluding tests that
must bind a local TCP port. PostgreSQL cases skip without TEST_DATABASE_URL. The
CI workflow provides PostgreSQL and runs the unrestricted suite; that CI run must
pass before merging. No successful live PostgreSQL or real-market volume run is
claimed by this local pass.

Independent archive discovery has since been implemented; see
[historical_discovery.md](historical_discovery.md). Remaining project deliverables
must be assessed separately: verification of actual archive coverage;
comprehensive calendar/coverage validation; E4-1 clean-data production; large-scale
quality reconciliation and performance measurements; frontend and final reporting.
They cannot be established by fixing backend code or by replaying fixtures.
