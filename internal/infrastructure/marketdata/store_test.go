package marketdata

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	domain "github.com/QuantDevelopers/Backend/internal/domain/marketdata"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Tests own an isolated schema; they never truncate shared application tables.
func testStore(t *testing.T) Store {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	require.NoError(t, err)
	schema := fmt.Sprintf("pipeline_%d", time.Now().UnixNano())
	_, err = conn.Exec(ctx, "CREATE SCHEMA "+schema)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = conn.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); _ = conn.Close(ctx) })
	_, err = conn.Exec(ctx, "SET search_path TO "+schema)
	require.NoError(t, err)
	cfg, err := pgx.ParseConfig(dsn)
	require.NoError(t, err)
	cfg.RuntimeParams["search_path"] = schema
	// Use the same GORM entities as production, including defaults and indexes.
	nativePool := stdlib.OpenDB(*cfg)
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: nativePool}), &gorm.Config{})
	require.NoError(t, err)
	pool, err := gdb.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, gdb.AutoMigrate(&entity.UnderlyingAsset{}, &entity.OptionContract{}, &entity.TickRaw{}, &entity.TickClean{}))
	return Store{Conn: conn}
}

func window() domain.Window {
	return domain.Window{From: time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC)}
}

func TestBulkPartitionReplayAndRollback(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	input := `{"instrument_type":"UNDERLYING","instrument_id":1,"trade_date":"2026-10-28","trade_time":"09:00:00","sequence_number":1,"price":100,"volume":20,"raw_payload":{"canceled":0}}` + "\n" +
		`{"instrument_type":"OPTION","instrument_id":2,"trade_date":"2026-11-01","trade_time":"09:01:00","sequence_number":1,"price":20,"volume":5}`
	n, err := s.ImportRaw(ctx, strings.NewReader(input))
	require.NoError(t, err)
	require.EqualValues(t, 2, n)
	require.NoError(t, s.Partition(ctx, window()))
	require.NoError(t, s.Partition(ctx, window()))
	n, err = s.ImportRaw(ctx, strings.NewReader(input))
	require.NoError(t, err)
	require.EqualValues(t, 2, n)
	var count int
	require.NoError(t, s.Conn.QueryRow(ctx, `SELECT count(*) FROM ticks_raw`).Scan(&count))
	require.Equal(t, 2, count)
	var children int
	require.NoError(t, s.Conn.QueryRow(ctx, `SELECT count(*) FROM pg_inherits WHERE inhparent='ticks_raw'::regclass`).Scan(&children))
	require.Equal(t, 3, children) // October, November, default
	_, err = s.ImportRaw(ctx, strings.NewReader(input+"\n{broken"))
	require.Error(t, err)
	require.NoError(t, s.Conn.QueryRow(ctx, `SELECT count(*) FROM ticks_raw`).Scan(&count))
	require.Equal(t, 2, count)
	// Exercise the actual crawler COPY adapter against GORM's pgx pool.
	var schema string
	require.NoError(t, s.Conn.QueryRow(ctx, `SELECT current_schema()`).Scan(&schema))
	cfg := s.Conn.Config().Copy()
	cfg.RuntimeParams["search_path"] = schema
	nativePool := stdlib.OpenDB(*cfg)
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: nativePool}), &gorm.Config{})
	require.NoError(t, err)
	pool, err := gdb.DB()
	require.NoError(t, err)
	defer func() { _ = pool.Close() }()
	db := &database.PostgresDatabase{DB: gdb}
	seq := 1
	ticks := []*entity.TickRaw{{TradeTime: "09:02:00", SequenceNumber: &seq, Price: 110, Volume: 50, RawPayload: json.RawMessage(`{"test":1}`)}}
	saved, err := ReplaceRawDay(ctx, db, "UNDERLYING", 1, window().From, ticks)
	require.NoError(t, err)
	require.Equal(t, 1, saved)
	ticks[0].TradeTime = "invalid"
	_, err = ReplaceRawDay(ctx, db, "UNDERLYING", 1, window().From, ticks)
	require.Error(t, err)
	var price float64
	require.NoError(t, s.Conn.QueryRow(ctx, `SELECT price FROM ticks_raw WHERE instrument_type='UNDERLYING'`).Scan(&price))
	require.Equal(t, 110.0, price)
}

func TestCandlesAndBackwardAsOf(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	_, err := s.Conn.Exec(ctx, `INSERT INTO underlying_assets (id,symbol_fa,name_fa) VALUES(1,'base','Base');
 INSERT INTO option_contracts(id,underlying_asset_id,symbol_fa,name_fa,contract_type,strike_price,expiration_date) VALUES(2,1,'call','Call','CALL',100,'2026-12-01')`)
	require.NoError(t, err)
	// Deliberately shuffled input; sequence breaks ties at equal timestamps.
	_, err = s.Conn.Exec(ctx, `INSERT INTO ticks_clean(instrument_type,instrument_id,trade_date,trade_timestamp,sequence_number,price,volume,is_canceled,is_outlier) VALUES
 ('UNDERLYING',1,'2026-10-28','2026-10-28 09:00:50+03:30',3,110,30,false,false),
 ('UNDERLYING',1,'2026-10-28','2026-10-28 09:00:00+03:30',1,100,10,false,false),
 ('UNDERLYING',1,'2026-10-28','2026-10-28 09:00:50+03:30',2,90,20,false,false),
 ('UNDERLYING',1,'2026-10-28','2026-10-28 09:01:00+03:30',4,120,40,false,false),
 ('UNDERLYING',1,'2026-10-28','2026-10-28 09:00:55+03:30',5,999,999,true,false),
 ('UNDERLYING',1,'2026-10-28','2026-10-28 09:00:56+03:30',6,999,999,false,true),
 ('UNDERLYING',1,'2026-10-27','2026-10-27 12:00:00+03:30',1,888,1,false,false),
 ('OPTION',2,'2026-10-28','2026-10-28 08:59:59+03:30',1,10,1,false,false),
 ('OPTION',2,'2026-10-28','2026-10-28 09:00:50+03:30',2,11,2,false,false),
 ('OPTION',2,'2026-10-28','2026-10-28 09:05:00+03:30',3,12,3,false,false)`)
	require.NoError(t, err)
	require.NoError(t, s.Partition(ctx, window()))
	result, err := s.Process(ctx, window(), 0)
	require.NoError(t, err)
	require.EqualValues(t, 3, result.OptionTicks)
	var o, h, l, c float64
	var volume, trades int64
	require.NoError(t, s.Conn.QueryRow(ctx, `SELECT open,high,low,close,volume,trade_count FROM candles WHERE instrument_type='UNDERLYING' AND interval_minutes=1 AND bucket_start='2026-10-28 09:00:00+03:30'`).Scan(&o, &h, &l, &c, &volume, &trades))
	require.Equal(t, []float64{100, 110, 90, 110}, []float64{o, h, l, c})
	require.EqualValues(t, 60, volume)
	require.EqualValues(t, 3, trades)
	require.NoError(t, s.Conn.QueryRow(ctx, `SELECT close,volume FROM candles WHERE instrument_type='UNDERLYING' AND interval_minutes=5`).Scan(&c, &volume))
	require.Equal(t, 120.0, c)
	require.EqualValues(t, 100, volume)
	var matched *float64
	require.NoError(t, s.Conn.QueryRow(ctx, `SELECT underlying_price FROM option_tick_asof WHERE sequence_number=1`).Scan(&matched))
	require.Nil(t, matched)
	require.NoError(t, s.Conn.QueryRow(ctx, `SELECT underlying_price FROM option_tick_asof WHERE sequence_number=2`).Scan(&matched))
	require.NotNil(t, matched)
	require.Equal(t, 110.0, *matched)
	again, err := s.Process(ctx, window(), 0)
	require.NoError(t, err)
	require.Equal(t, result, again)
	_, err = s.Process(ctx, window(), 30*time.Second)
	require.NoError(t, err)
	require.NoError(t, s.Conn.QueryRow(ctx, `SELECT underlying_price FROM option_tick_asof WHERE sequence_number=3`).Scan(&matched))
	require.Nil(t, matched)
	// Invalid clean dates must not erase the previously published window.
	_, err = s.Conn.Exec(ctx, `UPDATE ticks_clean SET trade_timestamp='2026-10-29 09:00:50+03:30' WHERE instrument_type='OPTION' AND sequence_number=2`)
	require.NoError(t, err)
	_, err = s.Process(ctx, window(), 0)
	require.Error(t, err)
	var count int
	require.NoError(t, s.Conn.QueryRow(ctx, `SELECT count(*) FROM option_tick_asof`).Scan(&count))
	require.Equal(t, 3, count)
}
