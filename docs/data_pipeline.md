# E3-2 / E4-2: bulk storage and quantitative data products

## Build and prerequisites

Run `make build`. The new `bin/quant-data` command uses the same database
configuration as the server. Initialize the existing schema with the normal
`--seed-only` flow first. PostgreSQL 16 is required. There is no network data
fetch in this command.

E4-2 consumes `ticks_clean` from E4-1. It does not populate that table from raw
data, remove duplicates or implement the cleaning backlog. Empty clean input
produces empty output. Option mappings come from `option_contracts`.

## E3-2: COPY and monthly partitions

The crawler's existing `ReplaceDayTicks` path now uses PostgreSQL binary COPY on
its GORM pool's native pgx connection. It deletes and replaces one instrument-day
inside a transaction, with an advisory lock for concurrent replacements of the
same instrument-day. An insert error rolls back the deletion. New IDs are allocated
on replay; the logical instrument-day content is idempotent. Nil input clears
that instrument-day. Other non-cooperating writers must not bypass this protocol.

For large offline deliveries, stream NDJSON (one JSON object per record):

```json
{"instrument_type":"UNDERLYING","instrument_id":1,"trade_date":"2026-10-28","trade_time":"09:00:05","sequence_number":1,"price":72500,"volume":100,"raw_payload":{"nTran":1}}
```

```bash
./bin/quant-data -mode=load-raw -input=ticks.ndjson
```

Each supplied instrument-day must be complete: its existing rows are replaced.
Days absent from the file are untouched. The importer uses COPY into a temporary
staging table, validates the stream, and publishes atomically. Memory use is
bounded by an individual record; PostgreSQL staging and transaction disk usage
scale with the import. Concurrent table writes are blocked during import.
This format is explicitly NDJSON, not a crawler manifest, JSON array or the
TSETMC HTTP payload. Missing raw payload is stored as SQL NULL.

For monthly range partitioning of **both** raw and clean tables:

```bash
./bin/quant-data -mode=partition -from=2026-01-01 -to=2027-01-01
```

Bounds are UTC dates, with an inclusive `from` and exclusive `to`. Run in a
maintenance window with writers stopped. The command takes exclusive table
locks, creates monthly partitions covering both requested dates and existing
rows, copies existing records and indexes them in one transaction. Primary keys
include `trade_date`, as required by PostgreSQL. Clean sequence uniqueness is
preserved. Failed conversion rolls back all DDL and copies.

Original tables are retained as `ticks_raw_unpartitioned` and
`ticks_clean_unpartitioned` backups; conversion therefore needs additional disk
space. Their sequences supply IDs for the new partitioned parents; sequence ownership
is transferred to the new parents during conversion. Backups
are snapshots at conversion time and do not receive subsequent writes. Existing
custom grants, triggers, views, foreign keys and custom indexes are outside this
migration: inspect dependencies before applying it to a customized deployment.

A default partition accepts dates outside the provisioned months. Provision
future months before writing those dates. Re-running partition mode adds missing
months; if overlapping rows already exist in the default partition, PostgreSQL
rejects the attachment and the transaction rolls back. Move those rows during a
separate maintenance operation first. Existing partitioned parents are not copied
again. This is an explicit maintenance command, not a server startup migration.

Indexes support instrument/date reads and backward underlying lookup; validate
actual query performance with `EXPLAIN (ANALYZE, BUFFERS)` on representative data.
No production performance number is claimed by the unit suite.

## E4-2: candles and backward As-of Join

```bash
./bin/quant-data -mode=process -from=2026-10-01 -to=2026-11-01 -max-age=5m
```

The processor creates `candles` and `option_tick_asof` on first use. It rebuilds
only the supplied window in one repeatable-read transaction. Runs are serialized;
a serialization failure can be retried. Reprocessing replaces previous results,
including obsolete outputs if clean source rows were removed.

- Both 1-minute and 5-minute OHLCV candles are persisted. Buckets are half-open,
  aligned to local midnight in `Asia/Tehran`, with `bucket_start` stored as
  `timestamptz`. They do not cross trading days. Empty buckets are absent;
  prices are not forward-filled.
- Open/close use timestamp then sequence then ID ordering, making equal-time
  ticks deterministic. Volume is the sum of input quantities, not quantity
  multiplied by contract size. Trade count counts eligible ticks.
- Canceled, outlier and soft-deleted clean ticks are excluded from both products.
- Each option tick looks up its mapped underlying's latest eligible tick on the
  same `trade_date`, with timestamp **less than or equal to** the option timestamp.
  Equal-time underlying ties select the highest sequence then ID. With second
  precision this is a timestamp rule, not proof of exchange event ordering.
- `-max-age=0` (default) allows any prior same-day tick; a positive duration rejects
  stale matches. Unmatched option ticks remain present with NULL underlying
  fields. No future match or previous-day carry is permitted. Options without a
  live contract mapping are not processed.

Output keys: `candles` is the number of candle rows written; `matches` counts
option output rows, including unmatched rows. Query tables directly for analysis;
this task adds no new public HTTP endpoints.

```sql
SELECT * FROM candles
WHERE instrument_type = 'UNDERLYING' AND instrument_id = 1
  AND trade_date = DATE '2026-10-28' AND interval_minutes = 5
ORDER BY bucket_start;

SELECT option_id, trade_timestamp, option_price,
       underlying_timestamp, underlying_price
FROM option_tick_asof
WHERE trade_date = DATE '2026-10-28'
ORDER BY option_id, trade_timestamp, sequence_number;
```

## Verification

```bash
make test
make lint
make build
golangci-lint run ./...
TEST_DATABASE_URL='postgresql://postgres:postgres@localhost:5432/quant_db?sslmode=disable' \
  go test -race -v ./internal/infrastructure/marketdata
```

Database tests create and remove an isolated schema. They cover conversion with
existing rows in multiple months, repeated conversion/import, malformed-input
rollback, the crawler's native COPY adapter, 1/5-minute candle boundaries and
ordering, canceled/outlier exclusion, backward matching, equal-time ties,
previous-day rejection, staleness and replay. They skip without
`TEST_DATABASE_URL`; the existing CI PostgreSQL job runs them.

`candles` is keyed by `(instrument_type, instrument_id, trade_date,
interval_minutes, bucket_start)` and contains `open`, `high`, `low`, `close`,
`volume` and `trade_count`. Equal-time ticks are ordered by sequence and ID.
Canceled, outlier and soft-deleted rows are excluded.

`option_tick_asof` is keyed by `(option_id, trade_date, sequence_number)` and
contains `trade_timestamp`, `option_price`, `underlying_id`,
`underlying_timestamp`, `underlying_sequence` and `underlying_price`. A missing
or stale same-day underlying match leaves the last three fields NULL. The
reported `option_ticks` count includes such unmatched rows. A database CHECK
also prevents an underlying timestamp later than its option timestamp.
Mismatched Tehran trading dates or missing option mappings fail the transaction
rather than silently discarding clean input.

## Validation

The PostgreSQL integration suite uses unique temporary schemas and covers
migration with existing rows, replay, rollback after malformed COPY input,
the crawler adapter, 1/5-minute boundaries, equal timestamps, excluded trades,
missing/past/future matches, stale matches and transactional rebuild rollback.
CI supplies `TEST_DATABASE_URL` and runs it with the full race suite. Locally:

```bash
TEST_DATABASE_URL='postgresql://postgres:postgres@localhost:5432/quant_db?sslmode=disable' \
  go test -race -v ./internal/infrastructure/marketdata
```

The test account needs permission to create schemas. Without that variable,
integration cases are explicitly skipped. A full live database run and
representative-volume EXPLAIN/throughput measurements are required before
production rollout; an offline fixture run is not a database benchmark.
