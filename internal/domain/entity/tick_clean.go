package entity

import (
	"fmt"
	"time"

	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

type TickClean struct {
	database.Model
	InstrumentType string    `gorm:"column:instrument_type;type:varchar(20);not null;index;uniqueIndex:uq_clean_tick_seq,priority:1;index:ix_clean_ticks_ts,priority:1;check:check_clean_instrument_type,instrument_type IN ('UNDERLYING', 'OPTION')"`
	InstrumentID   uint      `gorm:"column:instrument_id;not null;index;uniqueIndex:uq_clean_tick_seq,priority:2;index:ix_clean_ticks_ts,priority:2"`
	TradeDate      time.Time `gorm:"column:trade_date;type:date;not null;index;uniqueIndex:uq_clean_tick_seq,priority:3"`
	TradeTimestamp time.Time `gorm:"column:trade_timestamp;not null;index;index:ix_clean_ticks_ts,priority:3"`
	SequenceNumber int       `gorm:"column:sequence_number;not null;index;uniqueIndex:uq_clean_tick_seq,priority:4"`
	Price          float64   `gorm:"column:price;type:numeric(15,2);not null"`
	Volume         int64     `gorm:"column:volume;not null"`
	IsCanceled     bool      `gorm:"column:is_canceled;default:false;not null"`
	IsOutlier      bool      `gorm:"column:is_outlier;default:false;not null"`
}

func (TickClean) TableName() string {
	return "ticks_clean"
}

func (t TickClean) String() string {
	return fmt.Sprintf(
		"<TickClean(id=%d, type='%s', inst_id=%d, ts='%s', price=%.2f, vol=%d)>",
		t.ID, t.InstrumentType, t.InstrumentID, t.TradeTimestamp.Format(time.RFC3339), t.Price, t.Volume,
	)
}
