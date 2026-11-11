package postgres

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/domain/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type mockRow struct {
	values []any
	err    error
}

func (r mockRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return fmt.Errorf("scan dest len %d != row values len %d", len(dest), len(r.values))
	}
	for i := range dest {
		if err := assignDest(dest[i], r.values[i]); err != nil {
			return err
		}
	}
	return nil
}

type mockRows struct {
	data [][]any
	idx  int
	err  error
}

func (r *mockRows) Close()                                       {}
func (r *mockRows) Err() error                                   { return r.err }
func (r *mockRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *mockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *mockRows) Values() ([]any, error)                       { return nil, nil }
func (r *mockRows) RawValues() [][]byte                          { return nil }
func (r *mockRows) Conn() *pgx.Conn                              { return nil }
func (r *mockRows) TypeMap() *pgtype.Map                         { return nil }

func (r *mockRows) Next() bool {
	if r.idx < len(r.data) {
		r.idx++
		return true
	}
	return false
}

func (r *mockRows) Scan(dest ...any) error {
	row := r.data[r.idx-1]
	return mockRow{values: row}.Scan(dest...)
}

func assignDest(dest, val any) error {
	dv := reflect.ValueOf(dest)
	if dv.Kind() != reflect.Ptr || dv.IsNil() {
		return fmt.Errorf("dest %T must be non-nil pointer", dest)
	}
	elem := dv.Elem()
	if val == nil {
		elem.Set(reflect.Zero(elem.Type()))
		return nil
	}
	vv := reflect.ValueOf(val)
	if vv.Type().AssignableTo(elem.Type()) {
		elem.Set(vv)
		return nil
	}
	return fmt.Errorf("cannot assign %T to %s", val, elem.Type())
}

type recordedCall struct {
	sql  string
	args []any
}

type mockQuerier struct {
	execCalls     []recordedCall
	queryCalls    []recordedCall
	queryRowCalls []recordedCall

	execFn     func(sql string, args ...any) (pgconn.CommandTag, error)
	queryFn    func(sql string, args ...any) (pgx.Rows, error)
	queryRowFn func(sql string, args ...any) pgx.Row
}

func (m *mockQuerier) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	m.execCalls = append(m.execCalls, recordedCall{sql: sql, args: args})
	if m.execFn != nil {
		return m.execFn(sql, args...)
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (m *mockQuerier) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	m.queryCalls = append(m.queryCalls, recordedCall{sql: sql, args: args})
	if m.queryFn != nil {
		return m.queryFn(sql, args...)
	}
	return &mockRows{}, nil
}

func (m *mockQuerier) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	m.queryRowCalls = append(m.queryRowCalls, recordedCall{sql: sql, args: args})
	if m.queryRowFn != nil {
		return m.queryRowFn(sql, args...)
	}
	return mockRow{err: pgx.ErrNoRows}
}

func TestDefault25UnderlyingsList(t *testing.T) {
	if len(Default25Underlyings) != 25 {
		t.Fatalf("len(Default25Underlyings) = %d, want 25", len(Default25Underlyings))
	}
	seen := make(map[string]bool, 25)
	for _, item := range Default25Underlyings {
		if item.SymbolFA == "" || item.NameFA == "" {
			t.Fatalf("empty field in default underlying: %+v", item)
		}
		if seen[item.SymbolFA] {
			t.Fatalf("duplicate symbol_fa %q", item.SymbolFA)
		}
		seen[item.SymbolFA] = true
	}
}

func TestOptionRepositoryGetAll(t *testing.T) {
	expiry := time.Date(2025, 2, 22, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)

	mq := &mockQuerier{
		queryFn: func(_ string, _ ...any) (pgx.Rows, error) {
			return &mockRows{
				data: [][]any{
					{
						10, 1, "ضهرم1204", "اختیار خ اهرم-1200-1403/12/04", "CALL",
						1200.0, 1000, expiry, (*time.Time)(nil),
						(*string)(nil), (*string)(nil), true, "ACTIVE", &now, &now,
					},
				},
			}, nil
		},
	}

	repo := NewOptionContractRepository(mq)
	uaID := 1
	cType := "call"
	active := true
	contracts, err := repo.GetAll(context.Background(), repository.OptionFilter{
		UnderlyingAssetID: &uaID,
		ContractType:      &cType,
		IsActive:          &active,
	})
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(contracts) != 1 {
		t.Fatalf("len(contracts) = %d, want 1", len(contracts))
	}
	if contracts[0].SymbolFA != "ضهرم1204" || contracts[0].StrikePrice != 1200.0 || contracts[0].ContractType != "CALL" {
		t.Fatalf("unexpected contract: %+v", contracts[0])
	}

	if len(mq.queryCalls) != 1 {
		t.Fatalf("expected 1 SQL query, got %d", len(mq.queryCalls))
	}
	call := mq.queryCalls[0]
	if !strings.Contains(call.sql, "underlying_asset_id = $1") ||
		!strings.Contains(call.sql, "contract_type = $2") ||
		!strings.Contains(call.sql, "is_active = $3") {
		t.Fatalf("unexpected SQL filter clause: %s", call.sql)
	}
	if len(call.args) != 3 || call.args[1] != "CALL" {
		t.Fatalf("unexpected query args: %#v", call.args)
	}
}

func TestOptionRepositoryGetBySymbolAndSave(t *testing.T) {
	mq := &mockQuerier{}
	repo := NewOptionContractRepository(mq)

	notFound, err := repo.GetBySymbol(context.Background(), "ناشناس")
	if err != nil {
		t.Fatalf("GetBySymbol not found returned error: %v", err)
	}
	if notFound != nil {
		t.Fatalf("expected nil for missing option, got %+v", notFound)
	}

	expiry := time.Date(2025, 2, 22, 0, 0, 0, 0, time.UTC)
	now := time.Now().UTC()
	mq.queryRowFn = func(_ string, _ ...any) pgx.Row {
		return mockRow{
			values: []any{
				11, 1, "ضخود1204", "اختیار خ خودرو", "CALL",
				2800.0, 1000, expiry, (*time.Time)(nil),
				(*string)(nil), (*string)(nil), true, "ACTIVE", &now, &now,
			},
		}
	}

	saved, err := repo.Save(context.Background(), entity.OptionContract{
		UnderlyingAssetID: 1,
		SymbolFA:          "ضخود1204",
		NameFA:            "اختیار خ خودرو",
		ContractType:      "call",
		StrikePrice:       2800.0,
		ExpirationDate:    expiry,
		IsActive:          true,
	})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if saved.ID != 11 || saved.ContractSize != 1000 || saved.Status != "ACTIVE" {
		t.Fatalf("unexpected saved contract: %+v", saved)
	}
}

func TestUnderlyingRepositorySeedAndLookup(t *testing.T) {
	// Case 1: Empty table -> seeds 25 assets.
	mq := &mockQuerier{
		queryRowFn: func(sql string, _ ...any) pgx.Row {
			if strings.Contains(sql, "COUNT(*)") {
				return mockRow{values: []any{0}}
			}
			return mockRow{err: pgx.ErrNoRows}
		},
	}

	repo := NewUnderlyingRepository(mq)
	inserted, err := repo.SeedDefaults(context.Background())
	if err != nil {
		t.Fatalf("SeedDefaults failed: %v", err)
	}
	if inserted != 25 || len(mq.execCalls) != 25 {
		t.Fatalf("inserted=%d execCalls=%d, want 25/25", inserted, len(mq.execCalls))
	}

	// Case 2: Already populated -> skips seeding.
	mqSkip := &mockQuerier{
		queryRowFn: func(_ string, _ ...any) pgx.Row {
			return mockRow{values: []any{25}}
		},
	}
	repoSkip := NewUnderlyingRepository(mqSkip)
	insertedSkip, err := repoSkip.SeedDefaults(context.Background())
	if err != nil {
		t.Fatalf("SeedDefaults skip failed: %v", err)
	}
	if insertedSkip != 0 || len(mqSkip.execCalls) != 0 {
		t.Fatalf("expected 0 inserts when already seeded, got %d", insertedSkip)
	}
}
