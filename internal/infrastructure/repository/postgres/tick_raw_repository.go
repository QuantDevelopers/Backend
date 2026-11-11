package postgres

import (
	"strings"
	"time"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	domainrepo "github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	"gorm.io/gorm"
)

type TickRawRepository struct{}

func NewTickRawRepository() *TickRawRepository {
	return &TickRawRepository{}
}

func normalizeTradeDay(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
}

func (repo *TickRawRepository) ReplaceDayTicks(
	db database.Database,
	instrumentType string,
	instrumentID uint,
	tradeDate time.Time,
	ticks []*entity.TickRaw,
) (int, error) {
	normType := strings.ToUpper(strings.TrimSpace(instrumentType))
	day := normalizeTradeDay(tradeDate)

	var inserted int
	err := db.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().
			Where("instrument_type = ? AND instrument_id = ? AND trade_date = ?", normType, instrumentID, day).
			Delete(&entity.TickRaw{}).Error; err != nil {
			return err
		}

		if len(ticks) == 0 {
			return nil
		}

		batch := make([]*entity.TickRaw, 0, len(ticks))
		for _, t := range ticks {
			if t == nil {
				continue
			}
			cloned := *t
			cloned.ID = 0
			cloned.InstrumentType = normType
			cloned.InstrumentID = instrumentID
			cloned.TradeDate = day
			batch = append(batch, &cloned)
		}
		if len(batch) == 0 {
			return nil
		}

		res := tx.CreateInBatches(&batch, 500)
		if res.Error != nil {
			return res.Error
		}
		inserted = int(res.RowsAffected)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return inserted, nil
}

func (repo *TickRawRepository) GetByInstrumentAndDate(
	db database.Database,
	instrumentType string,
	instrumentID uint,
	tradeDate time.Time,
	offset int,
	limit int,
) ([]*entity.TickRaw, int64, error) {
	normType := strings.ToUpper(strings.TrimSpace(instrumentType))
	day := normalizeTradeDay(tradeDate)

	baseQuery := db.GetDB().Model(&entity.TickRaw{}).
		Where("instrument_type = ? AND instrument_id = ? AND trade_date = ?", normType, instrumentID, day)

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := baseQuery.Order("COALESCE(sequence_number, 0) ASC, trade_time ASC, id ASC")
	if offset > 0 {
		query = query.Offset(offset)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}

	var rows []*entity.TickRaw
	if err := query.Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (repo *TickRawRepository) GetLatestDate(
	db database.Database,
	instrumentType string,
	instrumentID uint,
) (*time.Time, error) {
	normType := strings.ToUpper(strings.TrimSpace(instrumentType))
	var latest *time.Time
	err := db.GetDB().Model(&entity.TickRaw{}).
		Select("MAX(trade_date)").
		Where("instrument_type = ? AND instrument_id = ?", normType, instrumentID).
		Scan(&latest).Error
	if err != nil {
		return nil, err
	}
	return latest, nil
}

func (repo *TickRawRepository) Count(db database.Database, filter domainrepo.TickRawFilter) (int64, error) {
	query := db.GetDB().Model(&entity.TickRaw{})
	if filter.InstrumentType != nil {
		query = query.Where("instrument_type = ?", strings.ToUpper(strings.TrimSpace(*filter.InstrumentType)))
	}
	if filter.InstrumentID != nil {
		query = query.Where("instrument_id = ?", *filter.InstrumentID)
	}
	if filter.TradeDate != nil {
		query = query.Where("trade_date = ?", normalizeTradeDay(*filter.TradeDate))
	}
	if filter.FromDate != nil {
		query = query.Where("trade_date >= ?", normalizeTradeDay(*filter.FromDate))
	}
	if filter.ToDate != nil {
		query = query.Where("trade_date <= ?", normalizeTradeDay(*filter.ToDate))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}
