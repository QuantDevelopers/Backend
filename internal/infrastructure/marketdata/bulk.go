package marketdata

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/stdlib"
)

var rawColumns = []string{"instrument_type", "instrument_id", "trade_date", "trade_time", "sequence_number", "price", "volume", "raw_payload", "created_at", "updated_at"}

// ReplaceRawDay uses the existing GORM pool's native pgx connection, retaining
// DELETE + COPY atomicity. Advisory locks serialize replacement of the same day.
func ReplaceRawDay(ctx context.Context, db database.Database, kind string, id uint, day time.Time, ticks []*entity.TickRaw) (int, error) {
	kind = strings.ToUpper(strings.TrimSpace(kind))
	if (kind != "UNDERLYING" && kind != "OPTION") || id == 0 {
		return 0, fmt.Errorf("invalid instrument")
	}
	pool, err := db.GetDB().DB()
	if err != nil {
		return 0, err
	}
	conn, err := pool.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = conn.Close() }()
	var count int64
	err = conn.Raw(func(driverConn interface{}) error {
		native, ok := driverConn.(*stdlib.Conn)
		if !ok {
			return fmt.Errorf("COPY requires pgx stdlib driver")
		}
		tx, beginErr := native.Conn().Begin(ctx)
		if beginErr != nil {
			return beginErr
		}
		defer func() { _ = tx.Rollback(ctx) }()
		key := fmt.Sprintf("%s/%d/%s", kind, id, day.Format("2006-01-02"))
		if _, e := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, key); e != nil {
			return e
		}
		if _, e := tx.Exec(ctx, `DELETE FROM ticks_raw WHERE instrument_type=$1 AND instrument_id=$2 AND trade_date=$3`, kind, id, day); e != nil {
			return e
		}
		now := time.Now().UTC()
		index := 0
		source := pgx.CopyFromFunc(func() ([]interface{}, error) {
			for index < len(ticks) {
				t := ticks[index]
				index++
				if t == nil {
					continue
				}
				if _, e := time.Parse("15:04:05", t.TradeTime); e != nil {
					return nil, e
				}
				var payload interface{}
				if len(t.RawPayload) > 0 {
					payload = t.RawPayload
				}
				return []interface{}{kind, int64(id), day, pgtype.Time{Microseconds: clockMicros(t.TradeTime), Valid: true}, t.SequenceNumber, t.Price, t.Volume, payload, now, now}, nil
			}
			return nil, nil
		})
		var copyErr error
		count, copyErr = tx.CopyFrom(ctx, pgx.Identifier{"ticks_raw"}, rawColumns, source)
		if copyErr != nil {
			return copyErr
		}
		return tx.Commit(ctx)
	})
	return int(count), err
}

// RawRecord is the newline-delimited JSON interchange format for the bulk loader.
// Each record represents one original trade; raw_payload remains available for audit.
type RawRecord struct {
	InstrumentType string          `json:"instrument_type"`
	InstrumentID   int64           `json:"instrument_id"`
	TradeDate      string          `json:"trade_date"`
	TradeTime      string          `json:"trade_time"`
	SequenceNumber *int            `json:"sequence_number"`
	Price          float64         `json:"price"`
	Volume         int64           `json:"volume"`
	RawPayload     json.RawMessage `json:"raw_payload"`
}

// ImportRaw streams NDJSON through COPY into a temporary staging table, then
// replaces only instrument-days present in the input. Replays are idempotent;
// malformed records roll back the entire import. It holds no input-sized buffer.
func (s Store) ImportRaw(ctx context.Context, r io.Reader) (int64, error) {
	tx, err := s.Conn.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// Prevent overlapping imports; block other writers during replacement.
	if _, err = tx.Exec(ctx, `LOCK TABLE ticks_raw IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(ctx, `CREATE TEMP TABLE raw_stage (LIKE ticks_raw INCLUDING DEFAULTS) ON COMMIT DROP`); err != nil {
		return 0, err
	}
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	now := time.Now().UTC()
	line := 0
	count, err := tx.CopyFrom(ctx, pgx.Identifier{"raw_stage"}, rawColumns, pgx.CopyFromFunc(func() ([]interface{}, error) {
		var t RawRecord
		if e := decoder.Decode(&t); e != nil {
			if e == io.EOF {
				return nil, nil
			}
			return nil, fmt.Errorf("record %d: %w", line+1, e)
		}
		line++
		day, e := time.Parse("2006-01-02", t.TradeDate)
		if e != nil {
			return nil, fmt.Errorf("record %d trade_date: %w", line, e)
		}
		if _, e = time.Parse("15:04:05", t.TradeTime); e != nil {
			return nil, fmt.Errorf("record %d trade_time: %w", line, e)
		}
		if t.InstrumentID <= 0 || (t.InstrumentType != "UNDERLYING" && t.InstrumentType != "OPTION") {
			return nil, fmt.Errorf("record %d: invalid instrument", line)
		}
		var payload interface{}
		if len(t.RawPayload) > 0 {
			payload = t.RawPayload
		}
		return []interface{}{t.InstrumentType, t.InstrumentID, day, pgtype.Time{Microseconds: clockMicros(t.TradeTime), Valid: true}, t.SequenceNumber, t.Price, t.Volume, payload, now, now}, nil
	}))
	if err != nil {
		return 0, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM ticks_raw t USING (SELECT DISTINCT instrument_type,instrument_id,trade_date FROM raw_stage) s WHERE t.instrument_type=s.instrument_type AND t.instrument_id=s.instrument_id AND t.trade_date=s.trade_date`); err != nil {
		return 0, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO ticks_raw (`+strings.Join(rawColumns, ",")+`) SELECT `+strings.Join(rawColumns, ",")+` FROM raw_stage`); err != nil {
		return 0, err
	}
	return count, tx.Commit(ctx)
}

func clockMicros(clock string) int64 {
	t, _ := time.Parse("15:04:05", clock)
	return int64(t.Hour()*3600+t.Minute()*60+t.Second()) * 1000000
}
