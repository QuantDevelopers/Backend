package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/domain/repository"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querier abstracts pgxpool.Pool, pgx.Tx, and test mocks.
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Default25Underlyings lists the 25 primary underlying assets in the Tehran Stock Exchange options market.
var Default25Underlyings = []entity.UnderlyingAsset{
	{SymbolFA: "اهرم", NameFA: "صندوق س. ا. کاریسما-شتاب", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "خودرو", NameFA: "ایران خودرو", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "خساپا", NameFA: "سایپا", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "شستا", NameFA: "سرمایه گذاری تامین اجتماعی", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "فملی", NameFA: "ملی صنایع مس ایران", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "فولاد", NameFA: "فولاد مبارکه اصفهان", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "طلا", NameFA: "صندوق پشتوانه طلای لوتوس", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "عیار", NameFA: "صندوق طلای عیار مفید", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "کهربا", NameFA: "صندوق طلای کهربا", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "زر", NameFA: "صندوق طلای زرین ترنج", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "گوهر", NameFA: "صندوق طلای گوهر نفیس", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "زرفام", NameFA: "صندوق طلای زرفام ترنج", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "ناب", NameFA: "صندوق طلای ناب مفید", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "وبملت", NameFA: "بانک ملت", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "وتجارت", NameFA: "بانک تجارت", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "دی", NameFA: "بانک دی", MarketType: "Farabourse", IsActive: true},
	{SymbolFA: "خگستر", NameFA: "گسترش سرمایه گذاری ایران خودرو", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "شتران", NameFA: "پالایش نفت تهران", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "شپنا", NameFA: "پالایش نفت اصفهان", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "شبندر", NameFA: "پالایش نفت بندرعباس", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "تاپیکو", NameFA: "س. نفت و گاز و پتروشیمی تامین", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "فارس", NameFA: "صنایع پتروشیمی خلیج فارس", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "وغدیر", NameFA: "سرمایه گذاری غدیر", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "کگل", NameFA: "معدنی و صنعتی گل گهر", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "کچاد", NameFA: "معدنی و صنعتی چادرملو", MarketType: "Bourse", IsActive: true},
}

// UnderlyingRepository implements repository.UnderlyingRepository on PostgreSQL.
type UnderlyingRepository struct {
	db Querier
}

// NewUnderlyingRepository creates a PostgreSQL-backed UnderlyingRepository.
func NewUnderlyingRepository(db Querier) *UnderlyingRepository {
	return &UnderlyingRepository{db: db}
}

var _ repository.UnderlyingRepository = (*UnderlyingRepository)(nil)

func scanUnderlying(scanner interface{ Scan(dest ...any) error }) (entity.UnderlyingAsset, error) {
	var item entity.UnderlyingAsset
	err := scanner.Scan(
		&item.ID,
		&item.SymbolFA,
		&item.NameFA,
		&item.TSETMCID,
		&item.ISIN,
		&item.MarketType,
		&item.IsActive,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

// GetAll returns all underlying assets ordered by ID.
func (r *UnderlyingRepository) GetAll(ctx context.Context) ([]entity.UnderlyingAsset, error) {
	const query = `
		SELECT id, symbol_fa, name_fa, tsetmc_id, isin, market_type, is_active, created_at, updated_at
		FROM underlying_assets
		ORDER BY id`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query underlying_assets: %w", err)
	}
	defer rows.Close()

	assets := make([]entity.UnderlyingAsset, 0)
	for rows.Next() {
		item, err := scanUnderlying(rows)
		if err != nil {
			return nil, fmt.Errorf("scan underlying_asset: %w", err)
		}
		assets = append(assets, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate underlying_assets: %w", err)
	}
	return assets, nil
}

// GetBySymbol fetches a single underlying asset by its Persian symbol, returning nil if not found.
func (r *UnderlyingRepository) GetBySymbol(ctx context.Context, symbolFA string) (*entity.UnderlyingAsset, error) {
	const query = `
		SELECT id, symbol_fa, name_fa, tsetmc_id, isin, market_type, is_active, created_at, updated_at
		FROM underlying_assets
		WHERE symbol_fa = $1
		LIMIT 1`

	item, err := scanUnderlying(r.db.QueryRow(ctx, query, symbolFA))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query underlying_asset by symbol %q: %w", symbolFA, err)
	}
	return &item, nil
}

// SeedDefaults inserts any missing records from the 25 default underlying assets.
func (r *UnderlyingRepository) SeedDefaults(ctx context.Context) (int, error) {
	var count int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM underlying_assets").Scan(&count); err != nil {
		return 0, fmt.Errorf("count underlying_assets: %w", err)
	}
	if count >= len(Default25Underlyings) {
		return 0, nil
	}

	const insertQuery = `
		INSERT INTO underlying_assets (symbol_fa, name_fa, market_type, is_active)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (symbol_fa) DO NOTHING`

	inserted := 0
	for _, item := range Default25Underlyings {
		tag, err := r.db.Exec(ctx, insertQuery, item.SymbolFA, item.NameFA, item.MarketType, item.IsActive)
		if err != nil {
			return inserted, fmt.Errorf("seed underlying %q: %w", item.SymbolFA, err)
		}
		inserted += int(tag.RowsAffected())
	}
	return inserted, nil
}
