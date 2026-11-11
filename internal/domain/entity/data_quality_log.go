package entity

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

type DataQualityLog struct {
	database.Model
	InstrumentType string          `gorm:"column:instrument_type;type:varchar(20);not null;index;index:ix_quality_inst_date,priority:1"`
	InstrumentID   uint            `gorm:"column:instrument_id;not null;index;index:ix_quality_inst_date,priority:2"`
	TradeDate      time.Time       `gorm:"column:trade_date;type:date;not null;index;index:ix_quality_inst_date,priority:3"`
	CheckCategory  string          `gorm:"column:check_category;type:varchar(50);not null;index"`
	Status         string          `gorm:"column:status;type:varchar(20);not null;index;check:check_quality_status,status IN ('PASSED', 'WARNING', 'FAILED')"`
	GapCount       int             `gorm:"column:gap_count;default:0;not null"`
	Details        json.RawMessage `gorm:"column:details;type:jsonb"`
	Notes          *string         `gorm:"column:notes;type:text"`
}

func (DataQualityLog) TableName() string {
	return "data_quality_logs"
}

func (q DataQualityLog) String() string {
	return fmt.Sprintf(
		"<DataQualityLog(id=%d, type='%s', check='%s', status='%s')>",
		q.ID, q.InstrumentType, q.CheckCategory, q.Status,
	)
}
