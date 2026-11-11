package crawlerdto

import "time"

// MarketWatchPayload matches TSETMC GetInstrumentOptionMarketWatch/0 JSON response.
type MarketWatchPayload struct {
	InstrumentOptMarketWatch []MarketWatchItem `json:"instrumentOptMarketWatch"`
}

// MarketWatchItem represents a single Call/Put strike row from TSETMC option watch.
type MarketWatchItem struct {
	UaInsCode        string  `json:"uaInsCode"`
	LVal30UA         string  `json:"lval30_UA"`
	PClosingUA       float64 `json:"pClosing_UA"`
	PDrCotValUA      float64 `json:"pDrCotVal_UA"`
	PriceYesterdayUA float64 `json:"priceYesterday_UA"`
	BeginDate        string  `json:"beginDate"`
	EndDate          string  `json:"endDate"`
	StrikePrice      float64 `json:"strikePrice"`
	ContractSize     int     `json:"contractSize"`
	RemainedDay      int     `json:"remainedDay"`

	InsCodeC        string  `json:"insCode_C"`
	LVal18AFCC      string  `json:"lVal18AFC_C"`
	LVal30C         string  `json:"lVal30_C"`
	PDrCotValC      float64 `json:"pDrCotVal_C"`
	PClosingC       float64 `json:"pClosing_C"`
	PriceYesterdayC float64 `json:"priceYesterday_C"`
	OPC             int     `json:"oP_C"`
	ZTotTranC       int     `json:"zTotTran_C"`
	QTotTran5JC     int64   `json:"qTotTran5J_C"`
	QTotCapC        float64 `json:"qTotCap_C"`

	InsCodeP        string  `json:"insCode_P"`
	LVal18AFCP      string  `json:"lVal18AFC_P"`
	LVal30P         string  `json:"lVal30_P"`
	PDrCotValP      float64 `json:"pDrCotVal_P"`
	PClosingP       float64 `json:"pClosing_P"`
	PriceYesterdayP float64 `json:"priceYesterday_P"`
	OPP             int     `json:"oP_P"`
	ZTotTranP       int     `json:"zTotTran_P"`
	QTotTran5JP     int64   `json:"qTotTran5J_P"`
	QTotCapP        float64 `json:"qTotCap_P"`
}

// DiscoveredUnderlying matches E1-3 underlyings.json registry schema.
type DiscoveredUnderlying struct {
	InsCode            string  `json:"insCode"`
	Symbol             string  `json:"symbol"`
	Name               string  `json:"name"`
	Market             string  `json:"market"`
	LastPrice          float64 `json:"last_price"`
	ClosePrice         float64 `json:"close_price"`
	YesterdayPrice     float64 `json:"yesterday_price"`
	ActiveOptionsCount int     `json:"active_options_count"`
	CallCount          int     `json:"call_count"`
	PutCount           int     `json:"put_count"`
	DiscoveredBy       string  `json:"discovered_by"`
	PriceChange        float64 `json:"price_change"`
}

// DiscoveredContract matches E1-3 active_contracts.json registry schema.
type DiscoveredContract struct {
	InsCode          string  `json:"insCode"`
	Symbol           string  `json:"symbol"`
	Name             string  `json:"name"`
	Side             string  `json:"side"`
	UaInsCode        string  `json:"uaInsCode"`
	UnderlyingSymbol string  `json:"underlying_symbol"`
	StrikePrice      float64 `json:"strike_price"`
	ContractSize     int     `json:"contract_size"`
	BeginDate        string  `json:"begin_date"`
	ExpiryDate       string  `json:"expiry_date"`
	RemainedDays     int     `json:"remained_days"`
	LastPrice        float64 `json:"last_price"`
	ClosePrice       float64 `json:"close_price"`
	OpenPositions    int     `json:"open_positions"`
	TradeCount       int     `json:"trade_count"`
	TradeVolume      int64   `json:"trade_volume"`
	TradeValue       float64 `json:"trade_value"`
	DiscoveredBy     string  `json:"discovered_by"`
}

// DiscoveryOptions controls E1-3 contract discovery execution.
type DiscoveryOptions struct {
	UseOfflineFixture bool
	FixturePath       string
	RegistryDir       string
	PersistDB         bool
	DisableFallback   bool
}

// DiscoveryResult summarizes E1-3 discovery output and written artifacts.
type DiscoveryResult struct {
	Underlyings     []DiscoveredUnderlying
	Contracts       []DiscoveredContract
	UnderlyingsPath string
	ContractsPath   string
	PersistedDBRows int
}

// TradePayload matches TSETMC GetTrade / GetTradeHistory JSON responses.
type TradePayload struct {
	TradeHistory []TradeTickRaw `json:"tradeHistory"`
	Trade        []TradeTickRaw `json:"trade"`
}

// TradeTickRaw represents a single raw trade row from TSETMC CDN.
type TradeTickRaw struct {
	InsCode  *string `json:"insCode"`
	DEven    int     `json:"dEven"`
	NTran    int     `json:"nTran"`
	HEven    int     `json:"hEven"`
	QTitTran int64   `json:"qTitTran"`
	PTran    float64 `json:"pTran"`
	Canceled int     `json:"canceled"`
}

// TickManifest matches E2-2 data/manifests/ticks/{insCode}_{date}_manifest.json.
type TickManifest struct {
	InsCode      string    `json:"insCode"`
	Date         string    `json:"date"`
	Status       string    `json:"status"`
	RecordsCount int       `json:"records_count"`
	RawSizeBytes int       `json:"raw_size_bytes"`
	SHA256       string    `json:"sha256"`
	FetchedAt    time.Time `json:"fetched_at"`
	RawPath      string    `json:"raw_path"`
	Source       string    `json:"source"`
}

// CleanTickRecord represents a normalized tick trade row produced by E2-2.
type CleanTickRecord struct {
	InsCode       string  `json:"insCode"`
	Date          string  `json:"date"`
	NTran         int     `json:"nTran"`
	HEven         int     `json:"hEven"`
	Time          string  `json:"time"`
	Price         float64 `json:"price"`
	Quantity      int64   `json:"quantity"`
	ContractSize  int     `json:"contract_size"`
	NotionalValue float64 `json:"notional_value"`
	IsCanceled    bool    `json:"is_canceled"`
}

// TickIngestOptions controls E2-2 sample tick ingestion execution.
type TickIngestOptions struct {
	InsCode           string
	DateStr           string
	ContractSize      int
	UseOfflineFixture bool
	FixturePath       string
	DataDir           string
	DisableFallback   bool
}

// TickIngestResult summarizes E2-2 tick ingestion artifacts and records.
type TickIngestResult struct {
	Manifest     TickManifest
	Records      []CleanTickRecord
	RawPath      string
	ManifestPath string
	CleanPath    string
}
