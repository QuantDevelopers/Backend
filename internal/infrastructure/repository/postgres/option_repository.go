package postgres

import (
	"errors"
	"strings"

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
	if err := query.Order("expiration_date ASC, strike_price ASC").Find(&contracts).Error; err != nil {
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
	err := db.GetDB().Where("symbol_fa = ?", symbolFA).First(&contract).Error
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
	return db.GetDB().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "symbol_fa"}},
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
	}).Create(contract).Error
}

func (repo *OptionContractRepository) SaveBulk(db database.Database, contracts []*entity.OptionContract) (int, error) {
	if len(contracts) == 0 {
		return 0, nil
	}
	for _, c := range contracts {
		c.ContractType = strings.ToUpper(strings.TrimSpace(c.ContractType))
	}
	res := db.GetDB().Clauses(clause.OnConflict{DoNothing: true}).Create(&contracts)
	if res.Error != nil {
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
}
