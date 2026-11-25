// Package marketdata implements PostgreSQL bulk storage and quantitative processing.
package marketdata

import (
	"context"
	"fmt"
	"time"

	domain "github.com/QuantDevelopers/Backend/internal/domain/marketdata"
	"github.com/jackc/pgx/v5"
)

type Store struct{ Conn *pgx.Conn }

// Partition converts existing raw/clean tables atomically, retaining the old
// tables as *_unpartitioned backups. It takes exclusive locks: run in maintenance.
// Monthly partitions cover the requested window and existing rows; a default
// partition accepts dates outside that window.
func (s Store) Partition(ctx context.Context, w domain.Window) error {
	if err := w.Validate(); err != nil {
		return err
	}
	tx, err := s.Conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(32703, 1)"); err != nil {
		return err
	}
	for _, table := range []string{"ticks_raw", "ticks_clean"} {
		var partitioned bool
		if err = tx.QueryRow(ctx, `SELECT relkind = 'p' FROM pg_class WHERE oid = $1::regclass`, table).Scan(&partitioned); err != nil {
			return err
		}
		from, to := w.From, w.To
		if !partitioned {
			if _, err = tx.Exec(ctx, `LOCK TABLE `+table+` IN ACCESS EXCLUSIVE MODE`); err != nil {
				return err
			}
			var earliest, latest *time.Time
			if err = tx.QueryRow(ctx, `SELECT min(trade_date),max(trade_date) FROM `+table).Scan(&earliest, &latest); err != nil {
				return err
			}
			if earliest != nil && earliest.Before(from) {
				from = *earliest
			}
			if latest != nil && !latest.Before(to) {
				to = latest.AddDate(0, 0, 1)
			}
			if _, err = tx.Exec(ctx, `ALTER TABLE `+table+` RENAME TO `+table+`_unpartitioned`); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `CREATE TABLE `+table+` (LIKE `+table+`_unpartitioned INCLUDING DEFAULTS INCLUDING CONSTRAINTS INCLUDING STORAGE) PARTITION BY RANGE (trade_date)`); err != nil {
				return err
			}
			// Partitioned unique constraints must include the partition key.
			if _, err = tx.Exec(ctx, `ALTER TABLE `+table+` ADD PRIMARY KEY (id,trade_date)`); err != nil {
				return err
			}
		}
		first := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
		for month := first; month.Before(to); month = month.AddDate(0, 1, 0) {
			next := month.AddDate(0, 1, 0)
			sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s_%s PARTITION OF %s FOR VALUES FROM ('%s') TO ('%s')", table, month.Format("200601"), table, month.Format("2006-01-02"), next.Format("2006-01-02"))
			if _, err = tx.Exec(ctx, sql); err != nil {
				return fmt.Errorf("create monthly partition (check default partition for overlapping rows): %w", err)
			}
		}
		if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS `+table+`_default PARTITION OF `+table+` DEFAULT`); err != nil {
			return err
		}
		if !partitioned {
			if _, err = tx.Exec(ctx, `INSERT INTO `+table+` SELECT * FROM `+table+`_unpartitioned`); err != nil {
				return err
			}
			// LIKE shares the original sequence. Keep its ownership with the backup,
			// which is retained intentionally; never drop that backup without migrating it.
		}
		if _, err = tx.Exec(ctx, `CREATE INDEX IF NOT EXISTS `+table+`_partition_lookup ON `+table+` (instrument_type,instrument_id,trade_date)`); err != nil {
			return err
		}
		if table == "ticks_clean" {
			if _, err = tx.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS ticks_clean_partition_seq ON ticks_clean (instrument_type,instrument_id,trade_date,sequence_number)`); err != nil {
				return err
			}
			if _, err = tx.Exec(ctx, `CREATE INDEX IF NOT EXISTS ticks_clean_asof ON ticks_clean (instrument_id,trade_date,trade_timestamp DESC,sequence_number DESC) INCLUDE (price) WHERE instrument_type='UNDERLYING' AND NOT is_canceled AND NOT is_outlier AND deleted_at IS NULL`); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS candles (
 instrument_type varchar(20) NOT NULL, instrument_id bigint NOT NULL,
 trade_date date NOT NULL, interval_minutes integer NOT NULL CHECK(interval_minutes IN (1,5)),
 bucket_start timestamptz NOT NULL, open numeric(15,2) NOT NULL, high numeric(15,2) NOT NULL,
 low numeric(15,2) NOT NULL, close numeric(15,2) NOT NULL, volume bigint NOT NULL,
 trade_count bigint NOT NULL,
 PRIMARY KEY(instrument_type,instrument_id,trade_date,interval_minutes,bucket_start)
);
CREATE TABLE IF NOT EXISTS option_tick_asof (
 option_id bigint NOT NULL, trade_date date NOT NULL, sequence_number integer NOT NULL,
 trade_timestamp timestamptz NOT NULL, option_price numeric(15,2) NOT NULL,
 underlying_id bigint NOT NULL, underlying_timestamp timestamptz,
 underlying_sequence integer, underlying_price numeric(15,2),
 PRIMARY KEY(option_id,trade_date,sequence_number),
 CHECK(underlying_timestamp IS NULL OR underlying_timestamp <= trade_timestamp)
);
CREATE INDEX IF NOT EXISTS option_tick_asof_time ON option_tick_asof(option_id,trade_timestamp);
CREATE INDEX IF NOT EXISTS ticks_clean_processing ON ticks_clean(instrument_type,instrument_id,trade_date,trade_timestamp,sequence_number)
 WHERE NOT is_canceled AND NOT is_outlier AND deleted_at IS NULL;
`

// Process rebuilds an entire date window in one repeatable-read transaction.
// Unmatched option ticks are retained with NULL underlying fields. No forward
// fill from another day or future tick is allowed. maxAge=0 permits any same-day age.
func (s Store) Process(ctx context.Context, w domain.Window, maxAge time.Duration) (domain.Result, error) {
	var result domain.Result
	if err := w.Validate(); err != nil {
		return result, err
	}
	if maxAge < 0 {
		return result, fmt.Errorf("max age cannot be negative")
	}
	tx, err := s.Conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return result, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Serialize rebuilds, including schema initialization and overlapping ranges.
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(32703,2)`); err != nil {
		return result, err
	}
	if _, err = tx.Exec(ctx, schemaSQL); err != nil {
		return result, err
	}
	var invalid bool
	// A mismatched day or missing option mapping must not silently lose input.
	if err = tx.QueryRow(ctx, `SELECT EXISTS (
 SELECT 1 FROM ticks_clean t LEFT JOIN option_contracts c ON c.id=t.instrument_id AND c.deleted_at IS NULL
 WHERE t.trade_date >= $1 AND t.trade_date < $2 AND NOT t.is_canceled AND NOT t.is_outlier AND t.deleted_at IS NULL
 AND ((t.trade_timestamp AT TIME ZONE 'Asia/Tehran')::date <> t.trade_date
 OR (t.instrument_type='OPTION' AND c.id IS NULL)))`, w.From, w.To).Scan(&invalid); err != nil {
		return result, err
	}
	if invalid {
		return result, fmt.Errorf("clean input contains a mismatched Tehran trading day or an unmapped option")
	}
	for _, table := range []string{"candles", "option_tick_asof"} {
		if _, err = tx.Exec(ctx, `DELETE FROM `+table+` WHERE trade_date >= $1 AND trade_date < $2`, w.From, w.To); err != nil {
			return result, err
		}
	}
	tag, err := tx.Exec(ctx, candleSQL, w.From, w.To)
	if err != nil {
		return result, err
	}
	result.Candles = tag.RowsAffected()
	tag, err = tx.Exec(ctx, asofSQL, w.From, w.To, maxAge.Seconds())
	if err != nil {
		return result, err
	}
	result.OptionTicks = tag.RowsAffected()
	return result, tx.Commit(ctx)
}

const candleSQL = `
INSERT INTO candles
SELECT instrument_type,instrument_id,trade_date,minutes,bucket,
 max(price) FILTER (WHERE first_row=1),max(price),min(price),
 max(price) FILTER (WHERE last_row=1),sum(volume),count(*)
FROM (
 SELECT *,
 row_number() OVER (PARTITION BY instrument_type,instrument_id,trade_date,minutes,bucket ORDER BY trade_timestamp,sequence_number,id) first_row,
 row_number() OVER (PARTITION BY instrument_type,instrument_id,trade_date,minutes,bucket ORDER BY trade_timestamp DESC,sequence_number DESC,id DESC) last_row
 FROM (
  SELECT t.*,m.minutes,
   date_bin(make_interval(mins=>m.minutes),trade_timestamp AT TIME ZONE 'Asia/Tehran',timestamp '2000-01-01') AT TIME ZONE 'Asia/Tehran' bucket
  FROM ticks_clean t CROSS JOIN (VALUES(1),(5)) m(minutes)
  WHERE trade_date >= $1 AND trade_date < $2 AND NOT is_canceled AND NOT is_outlier AND deleted_at IS NULL
 ) buckets
) ranked
GROUP BY instrument_type,instrument_id,trade_date,minutes,bucket`

const asofSQL = `
INSERT INTO option_tick_asof
SELECT o.instrument_id,o.trade_date,o.sequence_number,o.trade_timestamp,o.price,
 c.underlying_asset_id,u.trade_timestamp,u.sequence_number,u.price
FROM ticks_clean o JOIN option_contracts c ON c.id=o.instrument_id AND c.deleted_at IS NULL
LEFT JOIN LATERAL (
 SELECT b.trade_timestamp,b.sequence_number,b.price FROM ticks_clean b
 WHERE b.instrument_type='UNDERLYING' AND b.instrument_id=c.underlying_asset_id
 AND b.trade_date=o.trade_date AND b.trade_timestamp <= o.trade_timestamp
 AND ($3::double precision=0 OR b.trade_timestamp >= o.trade_timestamp - $3::double precision * interval '1 second')
 AND NOT b.is_canceled AND NOT b.is_outlier AND b.deleted_at IS NULL
 ORDER BY b.trade_timestamp DESC,b.sequence_number DESC,b.id DESC LIMIT 1
) u ON true
WHERE o.instrument_type='OPTION' AND o.trade_date >= $1 AND o.trade_date < $2
AND NOT o.is_canceled AND NOT o.is_outlier AND o.deleted_at IS NULL`

var _ domain.Processor = Store{}
