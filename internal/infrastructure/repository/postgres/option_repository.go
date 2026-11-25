package postgres

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	domainrepo "github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OptionContractRepository struct{}

func NewOptionContractRepository() *OptionContractRepository {
	return &OptionContractRepository{}
}

func applyOptionFilters(
	query *gorm.DB,
	uaID *uint,
	contractType *string,
	isActive *bool,
) *gorm.DB {
	if uaID != nil {
		query = query.Where("underlying_asset_id = ?", *uaID)
	}
	if contractType != nil {
		query = query.Where("contract_type = ?", strings.ToUpper(strings.TrimSpace(*contractType)))
	}
	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}
	return query
}

func (repo *OptionContractRepository) GetAll(db database.Database, filter domainrepo.OptionFilter) ([]*entity.OptionContract, error) {
	query := applyOptionFilters(
		db.GetDB().Model(&entity.OptionContract{}),
		filter.UnderlyingAssetID,
		filter.ContractType,
		filter.IsActive,
	)

	var contracts []*entity.OptionContract
	if err := query.Preload("UnderlyingAsset").Order("expiration_date ASC, strike_price ASC").Find(&contracts).Error; err != nil {
		return nil, err
	}
	return contracts, nil
}

func (repo *OptionContractRepository) ListPaginated(
	db database.Database,
	filter domainrepo.OptionQueryFilter,
) ([]*entity.OptionContract, int64, error) {
	query := applyOptionFilters(
		db.GetDB().Model(&entity.OptionContract{}),
		filter.UnderlyingAssetID,
		filter.ContractType,
		filter.IsActive,
	)

	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + q + "%"
		query = query.Where("symbol_fa ILIKE ? OR name_fa ILIKE ? OR tsetmc_id ILIKE ? OR isin ILIKE ?", like, like, like, like)
	}
	if filter.ExpirationDate != nil {
		expDay := normalizeTradeDay(*filter.ExpirationDate)
		query = query.Where("expiration_date = ?", expDay)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	ordered := query.Order("expiration_date ASC, strike_price ASC, id ASC")
	if filter.Offset > 0 {
		ordered = ordered.Offset(filter.Offset)
	}
	if filter.Limit > 0 {
		ordered = ordered.Limit(filter.Limit)
	}

	var contracts []*entity.OptionContract
	if err := ordered.Find(&contracts).Error; err != nil {
		return nil, 0, err
	}
	return contracts, total, nil
}

func (repo *OptionContractRepository) GetBySymbol(db database.Database, symbolFA string) (*entity.OptionContract, error) {
	var contract entity.OptionContract
	err := db.GetDB().Where("symbol_fa = ?", symbolFA).Order("is_active DESC, expiration_date DESC, id DESC").First(&contract).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &contract, nil
}

func (repo *OptionContractRepository) Save(db database.Database, contract *entity.OptionContract) error {
	contract.ContractType = strings.ToUpper(strings.TrimSpace(contract.ContractType))
	active := contract.IsActive
	err := db.GetDB().Transaction(func(tx *gorm.DB) error {
		conflict := clause.OnConflict{
			Columns: []clause.Column{{Name: "tsetmc_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"underlying_asset_id",
				"name_fa",
				"contract_type",
				"strike_price",
				"contract_size",
				"expiration_date",
				"begin_date",
				"tsetmc_id",
				"isin",
				"is_active",
				"status",
				"updated_at",
			}),
		}
		if contract.TSETMCID == nil {
			existing, e := repo.GetBySymbol(&database.PostgresDatabase{DB: tx}, contract.SymbolFA)
			if e != nil {
				return e
			}
			if existing != nil {
				contract.ID = existing.ID
				return tx.Model(&entity.OptionContract{}).Where("id = ?", existing.ID).Select("underlying_asset_id", "name_fa", "contract_type", "strike_price", "contract_size", "expiration_date", "begin_date", "is_active", "status", "updated_at").Updates(contract).Error
			}
			conflict = clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, UpdateAll: true}
		}
		err := tx.Clauses(conflict).Create(contract).Error
		if err != nil {
			return err
		}
		return tx.Model(&entity.OptionContract{}).Where("id = ?", contract.ID).Update("is_active", active).Error
	})
	contract.IsActive = active
	return err
}

func (repo *OptionContractRepository) SaveBulk(db database.Database, contracts []*entity.OptionContract) (int, error) {
	if len(contracts) == 0 {
		return 0, nil
	}
	for _, c := range contracts {
		if c == nil {
			return 0, fmt.Errorf("nil contract")
		}
		c.ContractType = strings.ToUpper(strings.TrimSpace(c.ContractType))
	}
	rows := make([]map[string]interface{}, 0, len(contracts))
	legacyCount := 0
	now := time.Now().UTC()
	for _, c := range contracts {
		if c.TSETMCID == nil {
			if err := repo.Save(db, c); err != nil {
				return 0, err
			}
			legacyCount++
			continue
		}
		rows = append(rows, map[string]interface{}{"underlying_asset_id": c.UnderlyingAssetID, "symbol_fa": c.SymbolFA, "name_fa": c.NameFA, "contract_type": c.ContractType, "strike_price": c.StrikePrice, "contract_size": c.ContractSize, "expiration_date": c.ExpirationDate, "begin_date": c.BeginDate, "tsetmc_id": c.TSETMCID, "isin": c.ISIN, "is_active": c.IsActive, "status": c.Status, "created_at": now, "updated_at": now})
	}
	if len(rows) == 0 {
		return legacyCount, nil
	}
	updates := clause.AssignmentColumns([]string{"symbol_fa", "underlying_asset_id", "name_fa", "contract_type", "strike_price", "contract_size", "expiration_date", "begin_date", "tsetmc_id", "is_active", "status", "updated_at"})
	updates = append(updates, clause.Assignment{Column: clause.Column{Name: "isin"}, Value: gorm.Expr("COALESCE(EXCLUDED.isin, option_contracts.isin)")})
	res := db.GetDB().Model(&entity.OptionContract{}).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "tsetmc_id"}}, DoUpdates: updates}).CreateInBatches(rows, 500)
	if res.Error != nil {
		return 0, res.Error
	}
	return legacyCount + int(res.RowsAffected), nil
}
