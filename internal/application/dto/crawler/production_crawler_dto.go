package crawlerdto

import "time"

// TickCrawlTask represents a single (instrument, date) unit of work in the E2-3 queue.
type TickCrawlTask struct {
	InstrumentType string `json:"instrument_type"`
	InstrumentID   uint   `json:"instrument_id"`
	SymbolFA       string `json:"symbol_fa"`
	InsCode        string `json:"ins_code"`
	DateStr        string `json:"date"`
	ContractSize   int    `json:"contract_size"`
}

// ProductionCrawlOptions configures E2-3/E2-4 concurrent queue, worker pool, resume, and DB persistence.
type ProductionCrawlOptions struct {
	Workers            int
	DataDir            string
	Dates              []string
	DaysBack           int
	OrderDesc          bool   // E2-4: Crawl from newest to oldest dates
	StartDate          string // YYYYMMDD lower bound
	EndDate            string // YYYYMMDD upper bound
	CalendarPath       string // path to trading calendar JSON
	FullHistory        bool   // crawl all available history without capping
	IncludeUnderlyings bool
	IncludeOptions     bool
	InsCode            string
	MaxTasks           int
	Resume             bool
	Force              bool
	PersistDB          bool
	UseOfflineFixture  bool
	FixturePath        string
	DisableFallback    bool
}

// ProductionCrawlReport summarizes an E2-3/E2-4 production crawl batch run.
type ProductionCrawlReport struct {
	State           string    `json:"state"`
	Pending         int       `json:"pending"`
	ReplayedCache   int       `json:"replayed_cache"`
	JobID           string    `json:"job_id"`
	StartedAt       time.Time `json:"started_at"`
	FinishedAt      time.Time `json:"finished_at"`
	Duration        string    `json:"duration,omitempty"`
	Order           string    `json:"order,omitempty"` // "desc" (newest to oldest) or "asc"
	NewestDate      string    `json:"newest_date,omitempty"`
	OldestDate      string    `json:"oldest_date,omitempty"`
	Workers         int       `json:"workers"`
	TotalQueued     int       `json:"total_queued"`
	SkippedResume   int       `json:"skipped_resume"`
	Succeeded       int       `json:"succeeded"`
	EmptyDays       int       `json:"empty_days"`
	Failed          int       `json:"failed"`
	TotalRecords    int       `json:"total_records"`
	DBRowsSaved     int       `json:"db_rows_saved"`
	ThroughputRPS   float64   `json:"throughput_rps,omitempty"`
	ThroughputTicks float64   `json:"throughput_ticks_per_sec,omitempty"`
	CircuitBroken   bool      `json:"circuit_broken"`
	Errors          []string  `json:"errors,omitempty"`
}

// DBCoverageSummary captures row counts across core database tables for status monitoring.
type DBCoverageSummary struct {
	UnderlyingsTotal        int   `json:"underlyings_total"`
	UnderlyingsWithTSETMCID int   `json:"underlyings_with_tsetmc_id"`
	OptionsTotal            int   `json:"options_total"`
	OptionsActive           int   `json:"options_active"`
	DailyHistoryRows        int64 `json:"daily_history_rows"`
	TickRawRows             int64 `json:"tick_raw_rows"`
}

// ManifestDiskSummary summarizes raw & manifest files persisted on disk by the crawler.
type ManifestDiskSummary struct {
	ManifestsTotal int        `json:"manifests_total"`
	ManifestsOK    int        `json:"manifests_ok"`
	ManifestsEmpty int        `json:"manifests_empty"`
	RawBytesTotal  int64      `json:"raw_bytes_total"`
	LastFetchedAt  *time.Time `json:"last_fetched_at,omitempty"`
}

// CrawlStatusResponse exposes combined crawler progress and DB coverage for E2-3 / E2-5 / E5-2.
type CrawlStatusResponse struct {
	CheckedAt  time.Time              `json:"checked_at"`
	Database   DBCoverageSummary      `json:"database"`
	Manifests  ManifestDiskSummary    `json:"manifests"`
	LatestJob  *ProductionCrawlReport `json:"latest_job,omitempty"`
	SafeMaxRPS float64                `json:"safe_max_rps"`
}
