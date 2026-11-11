package entity

import (
	"fmt"
	"time"

	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

type DailyHistory struct {
	database.Model
	InstrumentType  string    `gorm:"column:instrument_type;type:varchar(20);not null;index;uniqueIndex:uq_daily_instrument_date,priority:1;index:ix_daily_inst_date,priority:1;check:check_daily_instrument_type,instrument_type IN ('UNDERLYING', 'OPTION')"`
	InstrumentID    uint      `gorm:"column:instrument_id;not null;index;uniqueIndex:uq_daily_instrument_date,priority:2;index:ix_daily_inst_date,priority:2"`
	TradeDate       time.Time `gorm:"column:trade_date;type:date;not null;index;uniqueIndex:uq_daily_instrument_date,priority:3;index:ix_daily_inst_date,priority:3"`
	OpenPrice       float64   `gorm:"column:open_price;type:numeric(15,2);not null"`
	HighPrice       float64   `gorm:"column:high_price;type:numeric(15,2);not null"`
	LowPrice        float64   `gorm:"column:low_price;type:numeric(15,2);not null"`
	ClosePrice      float64   `gorm:"column:close_price;type:numeric(15,2);not null"`
	YesterdayPrice  *float64  `gorm:"column:yesterday_price;type:numeric(15,2)"`
	SettlementPrice *float64  `gorm:"column:settlement_price;type:numeric(15,2)"`
	Volume          int64     `gorm:"column:volume;default:0;not null"`
	Value           float64   `gorm:"column:value;type:numeric(20,2);default:0.0;not null"`
	TradeCount      int       `gorm:"column:trade_count;default:0;not null"`
	OpenInterest    *int      `gorm:"column:open_interest"`
}

func (DailyHistory) TableName() string {
	return "daily_history"
}

func (d DailyHistory) String() string {
	return fmt.Sprintf(
		"<DailyHistory(id=%d, type='%s', inst_id=%d, date='%s', close=%.2f)>",
		d.ID, d.InstrumentType, d.InstrumentID, d.TradeDate.Format("2006-01-02"), d.ClosePrice,
	)
}
