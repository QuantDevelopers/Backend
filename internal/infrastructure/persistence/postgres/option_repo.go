package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/domain/repository"
	"github.com/jackc/pgx/v5"
)

// OptionContractRepository implements repository.OptionContractRepository on PostgreSQL.
type OptionContractRepository struct {
	db Querier
}

// NewOptionContractRepository creates a PostgreSQL-backed OptionContractRepository.
func NewOptionContractRepository(db Querier) *OptionContractRepository {
	return &OptionContractRepository{db: db}
}

var _ repository.OptionContractRepository = (*OptionContractRepository)(nil)

func scanOptionContract(scanner interface{ Scan(dest ...any) error }) (entity.OptionContract, error) {
	var item entity.OptionContract
	err := scanner.Scan(
		&item.ID,
		&item.UnderlyingAssetID,
		&item.SymbolFA,
		&item.NameFA,
		&item.ContractType,
		&item.StrikePrice,
		&item.ContractSize,
		&item.ExpirationDate,
		&item.BeginDate,
		&item.TSETMCID,
		&item.ISIN,
		&item.IsActive,
		&item.Status,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

// GetAll fetches option contracts matching the given optional filters.
func (r *OptionContractRepository) GetAll(ctx context.Context, filter repository.OptionFilter) ([]entity.OptionContract, error) {
	var conditions []string
	var args []any

	if filter.UnderlyingAssetID != nil {
		args = append(args, *filter.UnderlyingAssetID)
		conditions = append(conditions, fmt.Sprintf("underlying_asset_id = $%d", len(args)))
	}
	if filter.ContractType != nil {
		args = append(args, strings.ToUpper(strings.TrimSpace(*filter.ContractType)))
		conditions = append(conditions, fmt.Sprintf("contract_type = $%d", len(args)))
	}
	if filter.IsActive != nil {
		args = append(args, *filter.IsActive)
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", len(args)))
	}

	query := `
		SELECT id, underlying_asset_id, symbol_fa, name_fa, contract_type,
		       strike_price::float8, contract_size, expiration_date, begin_date,
		       tsetmc_id, isin, is_active, status, created_at, updated_at
		FROM option_contracts`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY expiration_date, strike_price"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query option_contracts: %w", err)
	}
	defer rows.Close()

	contracts := make([]entity.OptionContract, 0)
	for rows.Next() {
		item, err := scanOptionContract(rows)
		if err != nil {
			return nil, fmt.Errorf("scan option_contract: %w", err)
		}
		contracts = append(contracts, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate option_contracts: %w", err)
	}
	return contracts, nil
}

// GetBySymbol fetches a single option contract by its Persian symbol, returning nil if not found.
func (r *OptionContractRepository) GetBySymbol(ctx context.Context, symbolFA string) (*entity.OptionContract, error) {
	const query = `
		SELECT id, underlying_asset_id, symbol_fa, name_fa, contract_type,
		       strike_price::float8, contract_size, expiration_date, begin_date,
		       tsetmc_id, isin, is_active, status, created_at, updated_at
		FROM option_contracts
		WHERE symbol_fa = $1
		LIMIT 1`

	item, err := scanOptionContract(r.db.QueryRow(ctx, query, symbolFA))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query option_contract by symbol %q: %w", symbolFA, err)
	}
	return &item, nil
}

const upsertOptionContractQuery = `
	INSERT INTO option_contracts (
		underlying_asset_id, symbol_fa, name_fa, contract_type,
		strike_price, contract_size, expiration_date, begin_date,
		tsetmc_id, isin, is_active, status
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	ON CONFLICT (symbol_fa) DO UPDATE SET
		underlying_asset_id = EXCLUDED.underlying_asset_id,
		name_fa = EXCLUDED.name_fa,
		contract_type = EXCLUDED.contract_type,
		strike_price = EXCLUDED.strike_price,
		contract_size = EXCLUDED.contract_size,
		expiration_date = EXCLUDED.expiration_date,
		begin_date = EXCLUDED.begin_date,
		tsetmc_id = EXCLUDED.tsetmc_id,
		isin = EXCLUDED.isin,
		is_active = EXCLUDED.is_active,
		status = EXCLUDED.status,
		updated_at = NOW()
	RETURNING id, underlying_asset_id, symbol_fa, name_fa, contract_type,
	          strike_price::float8, contract_size, expiration_date, begin_date,
	          tsetmc_id, isin, is_active, status, created_at, updated_at`

func optionContractArgs(c entity.OptionContract) []any {
	contractSize := c.ContractSize
	if contractSize == 0 {
		contractSize = 1000
	}
	status := c.Status
	if status == "" {
		status = entity.ContractStatusActive
	}
	return []any{
		c.UnderlyingAssetID,
		c.SymbolFA,
		c.NameFA,
		strings.ToUpper(strings.TrimSpace(c.ContractType)),
		c.StrikePrice,
		contractSize,
		c.ExpirationDate,
		c.BeginDate,
		c.TSETMCID,
		c.ISIN,
		c.IsActive,
		status,
	}
}

// Save creates or updates an option contract and returns the persisted record.
func (r *OptionContractRepository) Save(ctx context.Context, contract entity.OptionContract) (*entity.OptionContract, error) {
	saved, err := scanOptionContract(r.db.QueryRow(ctx, upsertOptionContractQuery, optionContractArgs(contract)...))
	if err != nil {
		return nil, fmt.Errorf("upsert option_contract %q: %w", contract.SymbolFA, err)
	}
	return &saved, nil
}

// SaveBulk persists a slice of option contracts and returns the affected row count.
func (r *OptionContractRepository) SaveBulk(ctx context.Context, contracts []entity.OptionContract) (int, error) {
	if len(contracts) == 0 {
		return 0, nil
	}

	const insertQuery = `
		INSERT INTO option_contracts (
			underlying_asset_id, symbol_fa, name_fa, contract_type,
			strike_price, contract_size, expiration_date, begin_date,
			tsetmc_id, isin, is_active, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (symbol_fa) DO NOTHING`

	inserted := 0
	for _, c := range contracts {
		tag, err := r.db.Exec(ctx, insertQuery, optionContractArgs(c)...)
		if err != nil {
			return inserted, fmt.Errorf("bulk insert option_contract %q: %w", c.SymbolFA, err)
		}
		inserted += int(tag.RowsAffected())
	}
	return inserted, nil
}
