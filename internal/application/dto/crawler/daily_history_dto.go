package crawlerdto

// ClosingPriceDailyPayload matches TSETMC GetClosingPriceDailyList/{insCode}/{Top}.
type ClosingPriceDailyPayload struct {
	ClosingPriceDaily []ClosingPriceDailyItem `json:"closingPriceDaily"`
}

// ClosingPriceDailyItem captures a single trading day's OHLCV statistics from TSETMC.
type ClosingPriceDailyItem struct {
	InsCode         string   `json:"insCode"`
	DEven           int      `json:"dEven"`
	HEven           int      `json:"hEven"`
	PriceFirst      float64  `json:"priceFirst"`
	PriceMax        float64  `json:"priceMax"`
	PriceMin        float64  `json:"priceMin"`
	PClosing        float64  `json:"pClosing"`
	PDrCotVal       float64  `json:"pDrCotVal"`
	PriceYesterday  float64  `json:"priceYesterday"`
	ZTotTran        float64  `json:"zTotTran"`
	QTotTran5J      float64  `json:"qTotTran5J"`
	QTotCap         float64  `json:"qTotCap"`
	SettlementPrice *float64 `json:"settlementPrice,omitempty"`
	OpenInterest    *int     `json:"openInterest,omitempty"`
}

// DailyHistoryCrawlOptions configures E2-1 daily OHLCV crawling.
type DailyHistoryCrawlOptions struct {
	TopDays               int
	IncludeUnderlyings    bool
	IncludeOptions        bool
	InsCode               string
	Limit                 int
	SaveRaw               bool
	RawDir                string
	UseOfflineFixture     bool
	UnderlyingFixturePath string
	OptionFixturePath     string
	DisableFallback       bool
}

// DailyHistoryInstrumentSummary summarizes crawl results for one instrument.
type DailyHistoryInstrumentSummary struct {
	InstrumentType string `json:"instrument_type"`
	InstrumentID   uint   `json:"instrument_id"`
	SymbolFA       string `json:"symbol_fa"`
	InsCode        string `json:"ins_code"`
	RowsParsed     int    `json:"rows_parsed"`
	RowsSaved      int    `json:"rows_saved"`
	RawPath        string `json:"raw_path,omitempty"`
	Error          string `json:"error,omitempty"`
}

// DailyHistoryCrawlResult aggregates E2-1 crawl progress across all queued instruments.
type DailyHistoryCrawlResult struct {
	InstrumentsTotal     int                             `json:"instruments_total"`
	InstrumentsSucceeded int                             `json:"instruments_succeeded"`
	InstrumentsFailed    int                             `json:"instruments_failed"`
	RowsParsed           int                             `json:"rows_parsed"`
	RowsSaved            int                             `json:"rows_saved"`
	CircuitBroken        bool                            `json:"circuit_broken"`
	Items                []DailyHistoryInstrumentSummary `json:"items"`
}
