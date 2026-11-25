package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	crawlerdto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	"github.com/QuantDevelopers/Backend/internal/application/usecase"
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	domainlogger "github.com/QuantDevelopers/Backend/internal/domain/logger"
	domainrepo "github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	domaintsetmc "github.com/QuantDevelopers/Backend/internal/domain/tsetmc"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

const (
	defaultProductionWorkers = 3
	crawlStatusFileName      = "crawl_status.json"
)

// ProductionCrawlerService implements E2-3 queued worker crawling, manifest resume,
// idempotent raw tick persistence, and E2-5 crawl progress reporting.
type ProductionCrawlerService struct {
	crawlerCfg     *bootstrap.Crawler
	tickIngestor   usecase.TickIngestorService
	underlyingRepo domainrepo.UnderlyingRepository
	optionRepo     domainrepo.OptionContractRepository
	dailyRepo      domainrepo.DailyHistoryRepository
	tickRawRepo    domainrepo.TickRawRepository
	db             database.Database
	logger         domainlogger.Logger
}

// NewProductionCrawlerService constructs a ProductionCrawlerService instance.
func NewProductionCrawlerService(
	crawlerCfg *bootstrap.Crawler,
	tickIngestor usecase.TickIngestorService,
	underlyingRepo domainrepo.UnderlyingRepository,
	optionRepo domainrepo.OptionContractRepository,
	dailyRepo domainrepo.DailyHistoryRepository,
	tickRawRepo domainrepo.TickRawRepository,
	db database.Database,
	log domainlogger.Logger,
) *ProductionCrawlerService {
	return &ProductionCrawlerService{
		crawlerCfg:     crawlerCfg,
		tickIngestor:   tickIngestor,
		underlyingRepo: underlyingRepo,
		optionRepo:     optionRepo,
		dailyRepo:      dailyRepo,
		tickRawRepo:    tickRawRepo,
		db:             db,
		logger:         log,
	}
}

func (s *ProductionCrawlerService) BuildQueue(
	opts crawlerdto.ProductionCrawlOptions,
) ([]crawlerdto.TickCrawlTask, int, error) {
	if !opts.IncludeUnderlyings && !opts.IncludeOptions {
		opts.IncludeUnderlyings = true
		opts.IncludeOptions = true
	}
	dataDir := opts.DataDir
	if dataDir == "" {
		dataDir = defaultDataBaseDir
	}

	if err := validateCrawlOptions(opts); err != nil {
		return nil, 0, err
	}
	candidates, err := s.resolveProductionTargets(opts)
	if err != nil {
		return nil, 0, err
	}

	rawExplicit := normalizeDateList(opts.Dates)
	var explicitDates []string
	for _, d := range rawExplicit {
		if opts.StartDate != "" && d < opts.StartDate {
			continue
		}
		if opts.EndDate != "" && d > opts.EndDate {
			continue
		}
		explicitDates = append(explicitDates, d)
	}

	todayStr := marketToday()
	var tasks []crawlerdto.TickCrawlTask
	skippedResume := 0

	for _, target := range candidates {
		dates := explicitDates
		if len(opts.Dates) == 0 {
			dates, err = s.resolveTargetDates(target, opts, todayStr)
			if err != nil {
				return nil, 0, err
			}
		}

		for _, dateStr := range dates {
			if (target.BeginDate != "" && dateStr < target.BeginDate) || (target.EndDate != "" && dateStr > target.EndDate) {
				continue
			}
			if opts.Resume && !opts.Force && !opts.PersistDB && taskReusable(dataDir, target.InsCode, dateStr, opts.UseOfflineFixture) {
				skippedResume++
				continue
			}

			contractSize := target.ContractSize
			if contractSize <= 0 {
				contractSize = 1000
			}
			tasks = append(tasks, crawlerdto.TickCrawlTask{
				InstrumentType: target.InstrumentType,
				InstrumentID:   target.InstrumentID,
				SymbolFA:       target.SymbolFA,
				InsCode:        target.InsCode,
				DateStr:        dateStr,
				ContractSize:   contractSize,
			})
		}
	}

	// Sort tasks deterministically (E2-4 default: newest to oldest dates)
	sortDesc := opts.OrderDesc
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].DateStr != tasks[j].DateStr {
			if sortDesc {
				return tasks[i].DateStr > tasks[j].DateStr
			}
			return tasks[i].DateStr < tasks[j].DateStr
		}
		if tasks[i].SymbolFA != tasks[j].SymbolFA {
			return tasks[i].SymbolFA < tasks[j].SymbolFA
		}
		return tasks[i].InsCode < tasks[j].InsCode
	})

	if opts.MaxTasks > 0 && len(tasks) > opts.MaxTasks {
		tasks = tasks[:opts.MaxTasks]
	}

	return tasks, skippedResume, nil
}

func (s *ProductionCrawlerService) Run(
	ctx context.Context,
	opts crawlerdto.ProductionCrawlOptions,
) (*crawlerdto.ProductionCrawlReport, error) {
	if opts.PersistDB && (s.db == nil || s.tickRawRepo == nil) {
		return nil, errors.New("database and tick repository are required for persistence")
	}
	if s.tickIngestor == nil {
		return nil, errors.New("tick ingestor service is required")
	}
	dataDir := opts.DataDir
	if dataDir == "" {
		dataDir = defaultDataBaseDir
	}
	opts.DataDir = dataDir

	startedAt := time.Now().UTC()
	tasks, skippedResume, err := s.BuildQueue(opts)
	if err != nil {
		return nil, err
	}

	workers := opts.Workers
	if workers <= 0 {
		workers = defaultProductionWorkers
	}
	if len(tasks) > 0 && workers > len(tasks) {
		workers = len(tasks)
	}
	if len(tasks) == 0 {
		workers = 0
	}

	newestDate := ""
	oldestDate := ""
	for _, t := range tasks {
		if newestDate == "" || t.DateStr > newestDate {
			newestDate = t.DateStr
		}
		if oldestDate == "" || t.DateStr < oldestDate {
			oldestDate = t.DateStr
		}
	}
	orderLabel := "desc"
	if !opts.OrderDesc {
		orderLabel = "asc"
	}

	report := &crawlerdto.ProductionCrawlReport{
		JobID:         fmt.Sprintf("crawl-%d", startedAt.UnixNano()),
		StartedAt:     startedAt,
		Order:         orderLabel,
		NewestDate:    newestDate,
		OldestDate:    oldestDate,
		Workers:       workers,
		TotalQueued:   len(tasks),
		SkippedResume: skippedResume,
	}

	report.State = "running"
	report.Pending = len(tasks)
	if err := writeStatusReport(dataDir, report); err != nil {
		return report, err
	}
	var statusErr error
	lastStatusWrite := time.Now()
	if len(tasks) > 0 {
		workerCtx, cancelWorkers := context.WithCancel(ctx)
		defer cancelWorkers()

		jobs := make(chan crawlerdto.TickCrawlTask, len(tasks))
		for _, task := range tasks {
			jobs <- task
		}
		close(jobs)

		var mu sync.Mutex
		var wg sync.WaitGroup

		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for task := range jobs {
					if workerCtx.Err() != nil {
						return
					}

					cachePath := ""
					if opts.Resume && !opts.Force && opts.PersistDB && taskReusable(dataDir, task.InsCode, task.DateStr, opts.UseOfflineFixture) {
						cachePath = filepath.Join(dataDir, "raw", "ticks", task.InsCode, task.DateStr+".json")
					}
					res, ingestErr := s.tickIngestor.IngestSample(workerCtx, crawlerdto.TickIngestOptions{
						CachePath:         cachePath,
						InsCode:           task.InsCode,
						DateStr:           task.DateStr,
						ContractSize:      task.ContractSize,
						UseOfflineFixture: opts.UseOfflineFixture,
						FixturePath:       opts.FixturePath,
						DataDir:           dataDir,
						DisableFallback:   opts.DisableFallback,
					})
					if ingestErr != nil {
						mu.Lock()
						report.Failed++
						if len(report.Errors) < 25 {
							report.Errors = append(report.Errors, fmt.Sprintf("%s/%s: %v", task.InsCode, task.DateStr, ingestErr))
						}
						if errors.Is(ingestErr, domaintsetmc.ErrCircuitBreakerOpen) {
							report.CircuitBroken = true
							cancelWorkers()
						}
						report.Pending = report.TotalQueued - report.Succeeded - report.Failed
						if time.Since(lastStatusWrite) >= time.Second {
							statusErr = writeStatusReport(dataDir, report)
							lastStatusWrite = time.Now()
							if statusErr != nil {
								cancelWorkers()
							}
						}
						mu.Unlock()
						continue
					}

					dbSaved := 0
					var dbErr error
					if opts.PersistDB && s.db != nil && s.tickRawRepo != nil {
						tradeDay, parseErr := parseTSETMCDate(task.DateStr)
						if parseErr != nil {
							dbErr = parseErr
						} else {
							rawEntities := make([]*entity.TickRaw, 0, len(res.Records))
							for _, rec := range res.Records {
								seq := rec.NTran
								rawJSON, _ := json.Marshal(crawlerdto.TradeTickRaw{NTran: rec.NTran, HEven: rec.HEven, PTran: rec.Price, QTitTran: rec.Quantity, Canceled: boolInt(rec.IsCanceled)})
								rawEntities = append(rawEntities, &entity.TickRaw{
									InstrumentType: task.InstrumentType,
									InstrumentID:   task.InstrumentID,
									TradeDate:      tradeDay,
									TradeTime:      rec.Time,
									SequenceNumber: &seq,
									Price:          rec.Price,
									Volume:         rec.Quantity,
									RawPayload:     rawJSON,
								})
							}
							dbSaved, dbErr = s.tickRawRepo.ReplaceDayTicks(s.db, task.InstrumentType, task.InstrumentID, tradeDay, rawEntities)
						}
					}

					mu.Lock()
					if cachePath != "" {
						report.ReplayedCache++
					}
					if dbErr != nil {
						report.Failed++
						if len(report.Errors) < 25 {
							report.Errors = append(report.Errors, fmt.Sprintf("%s/%s db: %v", task.InsCode, task.DateStr, dbErr))
						}
					} else {
						report.Succeeded++
						if res.Manifest.Status == "empty" {
							report.EmptyDays++
						}
						report.TotalRecords += res.Manifest.RecordsCount
						report.DBRowsSaved += dbSaved
					}
					report.Pending = report.TotalQueued - report.Succeeded - report.Failed
					if time.Since(lastStatusWrite) >= time.Second {
						statusErr = writeStatusReport(dataDir, report)
						lastStatusWrite = time.Now()
						if statusErr != nil {
							cancelWorkers()
						}
					}
					mu.Unlock()
				}
			}()
		}

		wg.Wait()
	}

	report.FinishedAt = time.Now().UTC()
	duration := report.FinishedAt.Sub(startedAt)
	report.Duration = duration.Round(time.Millisecond).String()
	if duration.Seconds() > 0 {
		report.ThroughputRPS = math.Round(float64(report.Succeeded+report.Failed)/duration.Seconds()*100) / 100
		report.ThroughputTicks = math.Round(float64(report.TotalRecords)/duration.Seconds()*100) / 100
	}
	report.Pending = report.TotalQueued - report.Succeeded - report.Failed
	report.State = "completed"
	if report.Failed > 0 || report.Pending > 0 || ctx.Err() != nil || statusErr != nil {
		report.State = "failed"
	}
	if err := writeStatusReport(dataDir, report); err != nil {
		return report, err
	}
	if statusErr != nil {
		return report, statusErr
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	if report.Failed > 0 || report.Pending > 0 {
		return report, fmt.Errorf("crawl incomplete: failed=%d pending=%d", report.Failed, report.Pending)
	}
	return report, nil
}

func (s *ProductionCrawlerService) GetStatus(dataDir string) (*crawlerdto.CrawlStatusResponse, error) {
	if dataDir == "" {
		dataDir = defaultDataBaseDir
	}

	safeRPS := 3.0
	if s.crawlerCfg != nil && s.crawlerCfg.RateLimitRPS > 0 {
		safeRPS = s.crawlerCfg.RateLimitRPS
	}

	resp := &crawlerdto.CrawlStatusResponse{
		CheckedAt:  time.Now().UTC(),
		SafeMaxRPS: safeRPS,
		Manifests:  scanManifestDirectory(dataDir),
	}

	if s.db != nil {
		if s.underlyingRepo != nil {
			assets, err := s.underlyingRepo.GetAll(s.db)
			if err != nil {
				return nil, fmt.Errorf("count underlyings for status: %w", err)
			}
			resp.Database.UnderlyingsTotal = len(assets)
			for _, a := range assets {
				if a.TSETMCID != nil && strings.TrimSpace(*a.TSETMCID) != "" {
					resp.Database.UnderlyingsWithTSETMCID++
				}
			}
		}

		if s.optionRepo != nil {
			contracts, err := s.optionRepo.GetAll(s.db, domainrepo.OptionFilter{})
			if err != nil {
				return nil, fmt.Errorf("count options for status: %w", err)
			}
			resp.Database.OptionsTotal = len(contracts)
			for _, c := range contracts {
				if c.IsActive {
					resp.Database.OptionsActive++
				}
			}
		}

		if s.dailyRepo != nil {
			totalDaily, err := s.dailyRepo.Count(s.db, domainrepo.DailyHistoryFilter{})
			if err != nil {
				return nil, fmt.Errorf("count daily history for status: %w", err)
			}
			resp.Database.DailyHistoryRows = totalDaily
		}

		if s.tickRawRepo != nil {
			totalTicks, err := s.tickRawRepo.Count(s.db, domainrepo.TickRawFilter{})
			if err != nil {
				return nil, fmt.Errorf("count tick_raw for status: %w", err)
			}
			resp.Database.TickRawRows = totalTicks
		}
	}

	statusPath := filepath.Join(dataDir, "manifests", crawlStatusFileName)
	if raw, err := os.ReadFile(statusPath); err == nil {
		var latest crawlerdto.ProductionCrawlReport
		if jsonErr := json.Unmarshal(raw, &latest); jsonErr == nil {
			resp.LatestJob = &latest
		}
	}

	return resp, nil
}

type productionTarget struct {
	InstrumentType     string
	InstrumentID       uint
	SymbolFA           string
	InsCode            string
	ContractSize       int
	BeginDate, EndDate string
}

func (s *ProductionCrawlerService) resolveProductionTargets(opts crawlerdto.ProductionCrawlOptions) ([]productionTarget, error) {
	insFilter := strings.TrimSpace(opts.InsCode)

	if s.db == nil {
		if !opts.UseOfflineFixture {
			return nil, errors.New("live production crawl requires persisted instrument registry (-persist-db)")
		}
		var synthetic []productionTarget
		if opts.IncludeUnderlyings {
			synthetic = append(synthetic, productionTarget{
				InstrumentType: "UNDERLYING",
				InstrumentID:   1,
				SymbolFA:       "اهرم",
				InsCode:        "17914401175772326",
				ContractSize:   1,
			})
		}
		if opts.IncludeOptions {
			synthetic = append(synthetic, productionTarget{
				InstrumentType: "OPTION",
				InstrumentID:   1,
				SymbolFA:       "ضهرم1033",
				InsCode:        "42511156565416187",
				ContractSize:   1000,
			})
		}
		return filterProductionTargets(synthetic, insFilter), nil
	}

	var out []productionTarget
	if opts.IncludeUnderlyings && s.underlyingRepo != nil {
		assets, err := s.underlyingRepo.GetAll(s.db)
		if err != nil {
			return nil, fmt.Errorf("list underlyings for crawl queue: %w", err)
		}
		for _, a := range assets {
			if a.TSETMCID == nil || strings.TrimSpace(*a.TSETMCID) == "" {
				continue
			}
			out = append(out, productionTarget{
				InstrumentType: "UNDERLYING",
				InstrumentID:   a.ID,
				SymbolFA:       a.SymbolFA,
				InsCode:        strings.TrimSpace(*a.TSETMCID),
				ContractSize:   1,
			})
		}
	}

	if opts.IncludeOptions && s.optionRepo != nil {
		filter := domainrepo.OptionFilter{}
		if !opts.FullHistory {
			active := true
			filter.IsActive = &active
		}
		contracts, err := s.optionRepo.GetAll(s.db, filter)
		if err != nil {
			return nil, fmt.Errorf("list options for crawl queue: %w", err)
		}
		for _, c := range contracts {
			begin, end := "", ""
			if c.BeginDate != nil {
				begin = c.BeginDate.Format("20060102")
			}
			if !c.ExpirationDate.IsZero() {
				end = c.ExpirationDate.Format("20060102")
			}
			if c.TSETMCID == nil || strings.TrimSpace(*c.TSETMCID) == "" {
				continue
			}
			cSize := c.ContractSize
			if cSize <= 0 {
				cSize = 1000
			}
			out = append(out, productionTarget{
				InstrumentType: "OPTION",
				InstrumentID:   c.ID,
				SymbolFA:       c.SymbolFA,
				InsCode:        strings.TrimSpace(*c.TSETMCID),
				ContractSize:   cSize,
				BeginDate:      begin, EndDate: end,
			})
		}
	}

	return filterProductionTargets(out, insFilter), nil
}

func (s *ProductionCrawlerService) resolveTargetDates(target productionTarget, opts crawlerdto.ProductionCrawlOptions, today string) ([]string, error) {
	var dates []string
	if opts.CalendarPath != "" {
		var err error
		dates, err = loadTradingCalendarDates(opts.CalendarPath, opts.StartDate, opts.EndDate)
		if err != nil {
			return nil, err
		}
	}
	// DB history is authoritative; a sample calendar must never truncate it.
	if s.db != nil && s.dailyRepo != nil && opts.CalendarPath == "" {
		days, err := s.dailyRepo.GetTradingDates(s.db, target.InstrumentType, target.InstrumentID, true)
		if err != nil {
			return nil, err
		}
		for _, d := range days {
			dates = append(dates, d.Format("20060102"))
		}
	} else if s.db == nil && opts.CalendarPath == "" && opts.UseOfflineFixture {
		var err error
		dates, err = loadTradingCalendarDates("data/trading_calendar.json", opts.StartDate, opts.EndDate)
		if err != nil {
			dates = []string{today}
		}
	}
	out := make([]string, 0, len(dates))
	seen := map[string]bool{}
	for _, d := range dates {
		if !seen[d] && (opts.StartDate == "" || d >= opts.StartDate) && (opts.EndDate == "" || d <= opts.EndDate) && (target.BeginDate == "" || d >= target.BeginDate) && (target.EndDate == "" || d <= target.EndDate) {
			out = append(out, d)
			seen[d] = true
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(out)))
	if opts.DaysBack > 0 && len(out) > opts.DaysBack {
		out = out[:opts.DaysBack]
	}
	return out, nil
}

func loadTradingCalendarDates(path, start, end string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var entries []struct {
		Date         string `json:"date"`
		IsTradingDay bool   `json:"is_trading_day"`
	}
	if err = json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	dates := []string{}
	seen := map[string]bool{}
	for _, e := range entries {
		d := strings.TrimSpace(e.Date)
		if _, err = parseTSETMCDate(d); err != nil {
			return nil, err
		}
		if e.IsTradingDay && !seen[d] && (start == "" || d >= start) && (end == "" || d <= end) {
			dates = append(dates, d)
			seen[d] = true
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))
	return dates, nil
}

func validateCrawlOptions(opts crawlerdto.ProductionCrawlOptions) error {
	if opts.DaysBack < 0 || opts.Workers < 0 || opts.MaxTasks < 0 {
		return errors.New("crawl limits cannot be negative")
	}
	if opts.InsCode != "" {
		if err := validateInsCode(opts.InsCode); err != nil {
			return err
		}
	}
	for _, d := range append(append([]string{}, opts.Dates...), opts.StartDate, opts.EndDate) {
		if d != "" {
			if _, err := parseTSETMCDate(d); err != nil {
				return err
			}
		}
	}
	if opts.StartDate != "" && opts.EndDate != "" && opts.StartDate > opts.EndDate {
		return errors.New("start date exceeds end date")
	}
	return nil
}

func taskReusable(dir, code, date string, offline bool) bool {
	if !isTaskCompletedOnDisk(dir, code, date) {
		return false
	}
	raw, err := os.ReadFile(filepath.Join(dir, "manifests", "ticks", code+"_"+date+"_manifest.json"))
	if err != nil {
		return false
	}
	var m crawlerdto.TickManifest
	return json.Unmarshal(raw, &m) == nil && m.Version == 2 && m.Offline == offline
}

func filterProductionTargets(items []productionTarget, insFilter string) []productionTarget {
	if insFilter == "" {
		return items
	}
	out := make([]productionTarget, 0, len(items))
	for _, it := range items {
		if it.InsCode == insFilter {
			out = append(out, it)
		}
	}
	return out
}

func normalizeDateList(dates []string) []string {
	if len(dates) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(dates))
	out := make([]string, 0, len(dates))
	for _, d := range dates {
		trimmed := strings.TrimSpace(d)
		if len(trimmed) != 8 {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func isTaskCompletedOnDisk(dataDir, insCode, dateStr string) bool {
	manifestPath := filepath.Join(dataDir, "manifests", "ticks", fmt.Sprintf("%s_%s_manifest.json", insCode, dateStr))
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return false
	}

	var m crawlerdto.TickManifest
	if err := json.Unmarshal(manifestBytes, &m); err != nil {
		return false
	}
	if m.InsCode != insCode || m.Date != dateStr {
		return false
	}
	if m.InsCode != insCode || m.Date != dateStr || m.RecordsCount < 0 || (m.Status != "ok" && m.Status != "empty") || m.SHA256 == "" {
		return false
	}

	rawPath := filepath.Join(dataDir, "raw", "ticks", insCode, dateStr+".json")
	rawBytes, err := os.ReadFile(rawPath)
	if err != nil {
		return false
	}
	if len(rawBytes) != m.RawSizeBytes {
		return false
	}
	return ComputeSHA256(rawBytes) == m.SHA256
}

func scanManifestDirectory(dataDir string) crawlerdto.ManifestDiskSummary {
	var summary crawlerdto.ManifestDiskSummary
	manifestDir := filepath.Join(dataDir, "manifests", "ticks")
	entries, err := os.ReadDir(manifestDir)
	if err != nil {
		return summary
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_manifest.json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(manifestDir, entry.Name()))
		if err != nil {
			continue
		}
		var m crawlerdto.TickManifest
		if err := json.Unmarshal(raw, &m); err != nil {
			continue
		}
		summary.ManifestsTotal++
		summary.RawBytesTotal += int64(m.RawSizeBytes)
		switch m.Status {
		case "ok":
			summary.ManifestsOK++
		case "empty":
			summary.ManifestsEmpty++
		}
		if !m.FetchedAt.IsZero() && (summary.LastFetchedAt == nil || m.FetchedAt.After(*summary.LastFetchedAt)) {
			ts := m.FetchedAt
			summary.LastFetchedAt = &ts
		}
	}
	return summary
}

func writeStatusReport(dataDir string, report *crawlerdto.ProductionCrawlReport) error {
	dir := filepath.Join(dataDir, "manifests")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return writeJSONPretty(filepath.Join(dir, crawlStatusFileName), report)
}
