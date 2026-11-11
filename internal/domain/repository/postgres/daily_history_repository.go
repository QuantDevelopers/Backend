package postgres

import (
	"time"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

// DailyHistoryFilter narrows daily_history queries and counts.
type DailyHistoryFilter struct {
	InstrumentType *string
	InstrumentID   *uint
	FromDate       *time.Time
	ToDate         *time.Time
	OnlyWithTrades bool
}

// DailyHistoryRepository manages OHLCV + volume daily_history persistence in PostgreSQL.
type DailyHistoryRepository interface {
	SaveBulk(db database.Database, records []*entity.DailyHistory) (int, error)
	GetByInstrument(db database.Database, instrumentType string, instrumentID uint, limit int) ([]*entity.DailyHistory, error)
	List(db database.Database, filter DailyHistoryFilter, limit int) ([]*entity.DailyHistory, error)
	GetTradingDates(db database.Database, instrumentType string, instrumentID uint, onlyWithTrades bool) ([]time.Time, error)
	Count(db database.Database, filter DailyHistoryFilter) (int64, error)
}
