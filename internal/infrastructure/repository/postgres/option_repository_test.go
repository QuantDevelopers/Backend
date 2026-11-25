package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	"github.com/stretchr/testify/require"
	pg "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// This driver runs GORM's real create/scan callbacks without a TCP server.
// It reproduces PostgreSQL RETURNING id, including the []map scan failure.
type bulkDriver struct{ state *bulkState }
type bulkState struct {
	batches, falseFlags, commits, rollbacks int
	failBatch                               int
}
type bulkConn struct{ state *bulkState }
type bulkTx struct{ state *bulkState }
type bulkRows struct{ remaining int64 }

func (d bulkDriver) Open(string) (driver.Conn, error) { return &bulkConn{state: d.state}, nil }
func (c *bulkConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unexpected prepared statement")
}
func (c *bulkConn) Close() error              { return nil }
func (c *bulkConn) Begin() (driver.Tx, error) { return bulkTx{c.state}, nil }
func (t bulkTx) Commit() error                { t.state.commits++; return nil }
func (t bulkTx) Rollback() error              { t.state.rollbacks++; return nil }
func (c *bulkConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if !strings.HasPrefix(query, "INSERT INTO") {
		return nil, fmt.Errorf("unexpected SQL: %s", query)
	}
	c.state.batches++
	if c.state.failBatch == c.state.batches {
		return nil, errors.New("injected insert failure")
	}
	count := 0
	for _, a := range args {
		if value, ok := a.Value.(bool); ok && !value {
			count++
		}
	}
	c.state.falseFlags += count
	return driver.RowsAffected(count), nil
}
func (c *bulkConn) QueryContext(_ context.Context, _ string, args []driver.NamedValue) (driver.Rows, error) {
	count := int64(0)
	for _, a := range args {
		if _, ok := a.Value.(bool); ok {
			count++
		}
	}
	return &bulkRows{remaining: count}, nil
}
func (r *bulkRows) Columns() []string { return []string{"id"} }
func (r *bulkRows) Close() error      { return nil }
func (r *bulkRows) Next(dest []driver.Value) error {
	if r.remaining == 0 {
		return io.EOF
	}
	dest[0] = r.remaining
	r.remaining--
	return nil
}

func TestSaveBulkMapBatchesUseAffectedRowsWithoutReturningScan(t *testing.T) {
	for _, size := range []int{1, 2, 501} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			state := &bulkState{}
			name := fmt.Sprintf("quant_bulk_%d_%d", size, time.Now().UnixNano())
			sql.Register(name, bulkDriver{state: state})
			pool, err := sql.Open(name, "")
			require.NoError(t, err)
			t.Cleanup(func() { _ = pool.Close() })
			gdb, err := gorm.Open(pg.New(pg.Config{Conn: pool}), &gorm.Config{DisableAutomaticPing: true})
			require.NoError(t, err)
			records := make([]*entity.OptionContract, size)
			for i := range records {
				code := fmt.Sprint(1000 + i)
				records[i] = &entity.OptionContract{UnderlyingAssetID: 1, SymbolFA: "reused", NameFA: "expired", ContractType: "CALL", TSETMCID: &code, StrikePrice: 100, ContractSize: 1000, ExpirationDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), IsActive: false, Status: "EXPIRED"}
			}
			count, err := NewOptionContractRepository().SaveBulk(&database.PostgresDatabase{DB: gdb}, records)
			require.NoError(t, err)
			require.Equal(t, size, count)
			require.Equal(t, size, state.falseFlags)
			require.Equal(t, (size+499)/500, state.batches)
			// A later failed batch must roll back the whole non-legacy batch import.
			if size > 500 {
				state.batches = 0
				state.failBatch = 2
				state.commits = 0
				state.rollbacks = 0
				count, err = NewOptionContractRepository().SaveBulk(&database.PostgresDatabase{DB: gdb}, records)
				require.Error(t, err)
				require.Zero(t, count)
				require.Zero(t, state.commits)
				require.Equal(t, 1, state.rollbacks)
			}
		})
	}
}
