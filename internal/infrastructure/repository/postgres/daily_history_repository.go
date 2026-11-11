package postgres

import (
	"fmt"
	"strings"
	"time"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	domainrepo "github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	"gorm.io/gorm/clause"
)

type DailyHistoryRepository struct{}

func NewDailyHistoryRepository() *DailyHistoryRepository {
	return &DailyHistoryRepository{}
}

func (repo *DailyHistoryRepository) SaveBulk(db database.Database, records []*entity.DailyHistory) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}

	dedupMap := make(map[string]*entity.DailyHistory, len(records))
	order := make([]string, 0, len(records))
	for _, rec := range records {
		if rec == nil {
			continue
		}
		cloned := *rec
		cloned.InstrumentType = strings.ToUpper(strings.TrimSpace(cloned.InstrumentType))
		cloned.TradeDate = time.Date(
			cloned.TradeDate.Year(),
			cloned.TradeDate.Month(),
			cloned.TradeDate.Day(),
			0, 0, 0, 0, time.UTC,
		)
		key := fmt.Sprintf("%s:%d:%s", cloned.InstrumentType, cloned.InstrumentID, cloned.TradeDate.Format("2006-01-02"))
		if _, exists := dedupMap[key]; !exists {
			order = append(order, key)
		}
		dedupMap[key] = &cloned
	}

	if len(order) == 0 {
		return 0, nil
	}

	batch := make([]*entity.DailyHistory, 0, len(order))
	for _, key := range order {
		batch = append(batch, dedupMap[key])
	}

	res := db.GetDB().Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "instrument_type"},
			{Name: "instrument_id"},
			{Name: "trade_date"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"open_price",
			"high_price",
			"low_price",
			"close_price",
			"yesterday_price",
			"settlement_price",
			"volume",
			"value",
			"trade_count",
			"open_interest",
			"updated_at",
		}),
	}).CreateInBatches(&batch, 500)
	if res.Error != nil {
		return 0, res.Error
	}
	return int(res.RowsAffected), nil
}

func (repo *DailyHistoryRepository) GetByInstrument(
	db database.Database,
	instrumentType string,
	instrumentID uint,
	limit int,
) ([]*entity.DailyHistory, error) {
	normType := strings.ToUpper(strings.TrimSpace(instrumentType))
	return repo.List(db, domainrepo.DailyHistoryFilter{
		InstrumentType: &normType,
		InstrumentID:   &instrumentID,
	}, limit)
}

func (repo *DailyHistoryRepository) List(
	db database.Database,
	filter domainrepo.DailyHistoryFilter,
	limit int,
) ([]*entity.DailyHistory, error) {
	query := db.GetDB().Model(&entity.DailyHistory{})
	if filter.InstrumentType != nil {
		query = query.Where("instrument_type = ?", strings.ToUpper(strings.TrimSpace(*filter.InstrumentType)))
	}
	if filter.InstrumentID != nil {
		query = query.Where("instrument_id = ?", *filter.InstrumentID)
	}
	if filter.FromDate != nil {
		query = query.Where("trade_date >= ?", normalizeTradeDay(*filter.FromDate))
	}
	if filter.ToDate != nil {
		query = query.Where("trade_date <= ?", normalizeTradeDay(*filter.ToDate))
	}
	if filter.OnlyWithTrades {
		query = query.Where("trade_count > 0 OR volume > 0")
	}

	query = query.Order("trade_date DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	var rows []*entity.DailyHistory
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (repo *DailyHistoryRepository) GetTradingDates(
	db database.Database,
	instrumentType string,
	instrumentID uint,
	onlyWithTrades bool,
) ([]time.Time, error) {
	query := db.GetDB().Model(&entity.DailyHistory{}).
		Where("instrument_type = ? AND instrument_id = ?", strings.ToUpper(strings.TrimSpace(instrumentType)), instrumentID)
	if onlyWithTrades {
		query = query.Where("trade_count > 0 OR volume > 0")
	}

	var dates []time.Time
	if err := query.Order("trade_date DESC").Pluck("trade_date", &dates).Error; err != nil {
		return nil, err
	}
	return dates, nil
}

func (repo *DailyHistoryRepository) Count(db database.Database, filter domainrepo.DailyHistoryFilter) (int64, error) {
	query := db.GetDB().Model(&entity.DailyHistory{})
	if filter.InstrumentType != nil {
		query = query.Where("instrument_type = ?", strings.ToUpper(strings.TrimSpace(*filter.InstrumentType)))
	}
	if filter.InstrumentID != nil {
		query = query.Where("instrument_id = ?", *filter.InstrumentID)
	}
	if filter.FromDate != nil {
		query = query.Where("trade_date >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil {
		query = query.Where("trade_date <= ?", *filter.ToDate)
	}
	if filter.OnlyWithTrades {
		query = query.Where("trade_count > 0 OR volume > 0")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}
