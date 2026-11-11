package postgres

import (
	"time"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

// TickRawFilter narrows ticks_raw queries and counts.
type TickRawFilter struct {
	InstrumentType *string
	InstrumentID   *uint
	TradeDate      *time.Time
	FromDate       *time.Time
	ToDate         *time.Time
}

// TickRawRepository manages raw intraday tick persistence and queries in PostgreSQL.
type TickRawRepository interface {
	ReplaceDayTicks(db database.Database, instrumentType string, instrumentID uint, tradeDate time.Time, ticks []*entity.TickRaw) (int, error)
	GetByInstrumentAndDate(db database.Database, instrumentType string, instrumentID uint, tradeDate time.Time, offset, limit int) ([]*entity.TickRaw, int64, error)
	GetLatestDate(db database.Database, instrumentType string, instrumentID uint) (*time.Time, error)
	Count(db database.Database, filter TickRawFilter) (int64, error)
}
