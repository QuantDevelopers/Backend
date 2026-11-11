package entity

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

type ClockTime struct {
	Hour   int
	Minute int
	Second int
}

func (c ClockTime) String() string {
	return fmt.Sprintf("%02d:%02d:%02d", c.Hour, c.Minute, c.Second)
}

type TickRaw struct {
	database.Model
	InstrumentType string          `gorm:"column:instrument_type;type:varchar(20);not null;index;index:ix_raw_ticks_lookup,priority:1;check:check_raw_instrument_type,instrument_type IN ('UNDERLYING', 'OPTION')"`
	InstrumentID   uint            `gorm:"column:instrument_id;not null;index;index:ix_raw_ticks_lookup,priority:2"`
	TradeDate      time.Time       `gorm:"column:trade_date;type:date;not null;index;index:ix_raw_ticks_lookup,priority:3"`
	TradeTime      string          `gorm:"column:trade_time;type:time without time zone;not null;index;index:ix_raw_ticks_lookup,priority:4"`
	SequenceNumber *int            `gorm:"column:sequence_number"`
	Price          float64         `gorm:"column:price;type:numeric(15,2);not null"`
	Volume         int64           `gorm:"column:volume;not null"`
	RawPayload     json.RawMessage `gorm:"column:raw_payload;type:jsonb"`
}

func (TickRaw) TableName() string {
	return "ticks_raw"
}

func (t TickRaw) String() string {
	return fmt.Sprintf(
		"<TickRaw(id=%d, type='%s', inst_id=%d, time='%s %s', price=%.2f)>",
		t.ID, t.InstrumentType, t.InstrumentID, t.TradeDate.Format("2006-01-02"), t.TradeTime, t.Price,
	)
}
