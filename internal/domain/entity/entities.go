package entity

import (
	"fmt"
	"time"
)

const (
	InstrumentTypeUnderlying = "UNDERLYING"
	InstrumentTypeOption     = "OPTION"

	ContractStatusActive   = "ACTIVE"
	ContractStatusExpired  = "EXPIRED"
	ContractStatusDelisted = "DELISTED"

	QualityStatusPassed  = "PASSED"
	QualityStatusWarning = "WARNING"
	QualityStatusFailed  = "FAILED"
)

// ClockTime represents a wall-clock time of day (HH:MM:SS) without a date component.
type ClockTime struct {
	Hour   int
	Minute int
	Second int
}

// String formats ClockTime as HH:MM:SS.
func (c ClockTime) String() string {
	return fmt.Sprintf("%02d:%02d:%02d", c.Hour, c.Minute, c.Second)
}

// UnderlyingAsset represents a Tehran Stock Exchange underlying equity or ETF.
type UnderlyingAsset struct {
	ID         int
	SymbolFA   string
	NameFA     string
	TSETMCID   *string
	ISIN       *string
	MarketType string
	IsActive   bool
	CreatedAt  *time.Time
	UpdatedAt  *time.Time
}

func (u UnderlyingAsset) String() string {
	return fmt.Sprintf("<UnderlyingAsset(id=%d, symbol_fa='%s', name_fa='%s')>", u.ID, u.SymbolFA, u.NameFA)
}

// OptionContract represents a Call or Put option contract listed on TSETMC.
type OptionContract struct {
	ID                int
	UnderlyingAssetID int
	SymbolFA          string
	NameFA            string
	ContractType      string // "CALL" or "PUT"
	StrikePrice       float64
	ContractSize      int
	ExpirationDate    time.Time
	BeginDate         *time.Time
	TSETMCID          *string
	ISIN              *string
	IsActive          bool
	Status            string
	CreatedAt         *time.Time
	UpdatedAt         *time.Time
}

func (o OptionContract) String() string {
	return fmt.Sprintf(
		"<OptionContract(id=%d, symbol_fa='%s', type='%s', strike=%.2f, expiry='%s')>",
		o.ID, o.SymbolFA, o.ContractType, o.StrikePrice, o.ExpirationDate.Format("2006-01-02"),
	)
}

// DailyHistory represents daily OHLCV and open interest summary for an instrument.
type DailyHistory struct {
	ID              int64
	InstrumentType  string // "UNDERLYING" or "OPTION"
	InstrumentID    int
	TradeDate       time.Time
	OpenPrice       float64
	HighPrice       float64
	LowPrice        float64
	ClosePrice      float64
	YesterdayPrice  *float64
	SettlementPrice *float64
	Volume          int64
	Value           float64
	TradeCount      int
	OpenInterest    *int
	CreatedAt       *time.Time
	UpdatedAt       *time.Time
}

func (d DailyHistory) String() string {
	return fmt.Sprintf(
		"<DailyHistory(id=%d, type='%s', inst_id=%d, date='%s', close=%.2f)>",
		d.ID, d.InstrumentType, d.InstrumentID, d.TradeDate.Format("2006-01-02"), d.ClosePrice,
	)
}

// TickRaw stores unmodified tick trade data received from TSETMC.
type TickRaw struct {
	ID             int64
	InstrumentType string
	InstrumentID   int
	TradeDate      time.Time
	TradeTime      ClockTime
	Price          float64
	Volume         int64
	SequenceNumber *int
	RawPayload     map[string]any
	CreatedAt      *time.Time
}

func (t TickRaw) String() string {
	return fmt.Sprintf(
		"<TickRaw(id=%d, type='%s', inst_id=%d, time='%s %s', price=%.2f)>",
		t.ID, t.InstrumentType, t.InstrumentID, t.TradeDate.Format("2006-01-02"), t.TradeTime.String(), t.Price,
	)
}

// TickClean stores deduplicated, ordered, and normalized intraday tick data.
type TickClean struct {
	ID             int64
	InstrumentType string
	InstrumentID   int
	TradeDate      time.Time
	TradeTimestamp time.Time
	SequenceNumber int
	Price          float64
	Volume         int64
	IsCanceled     bool
	IsOutlier      bool
	CreatedAt      *time.Time
}

func (t TickClean) String() string {
	return fmt.Sprintf(
		"<TickClean(id=%d, type='%s', inst_id=%d, ts='%s', price=%.2f, vol=%d)>",
		t.ID, t.InstrumentType, t.InstrumentID, t.TradeTimestamp.Format(time.RFC3339), t.Price, t.Volume,
	)
}

// DataQualityLog records automated quality checks, gap detection, and coverage diagnostics.
type DataQualityLog struct {
	ID             int64
	InstrumentType string
	InstrumentID   int
	TradeDate      time.Time
	CheckCategory  string
	Status         string
	GapCount       int
	Details        map[string]any
	Notes          *string
	CreatedAt      *time.Time
}

func (q DataQualityLog) String() string {
	return fmt.Sprintf(
		"<DataQualityLog(id=%d, type='%s', check='%s', status='%s')>",
		q.ID, q.InstrumentType, q.CheckCategory, q.Status,
	)
}
