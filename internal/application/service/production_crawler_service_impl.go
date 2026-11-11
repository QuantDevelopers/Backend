package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

	candidates, err := s.resolveProductionTargets(opts)
	if err != nil {
		return nil, 0, err
	}

	explicitDates := normalizeDateList(opts.Dates)
	todayStr := time.Now().UTC().Format("20060102")

	var tasks []crawlerdto.TickCrawlTask
	skippedResume := 0

	for _, target := range candidates {
		dates := explicitDates
		if len(dates) == 0 {
			dates = s.resolveTargetDates(target, opts.DaysBack, todayStr)
		}

		for _, dateStr := range dates {
			if opts.Resume && !opts.Force && isTaskCompletedOnDisk(dataDir, target.InsCode, dateStr) {
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
			if opts.MaxTasks > 0 && len(tasks) >= opts.MaxTasks {
				return tasks, skippedResume, nil
			}
		}
	}

	return tasks, skippedResume, nil
}

func (s *ProductionCrawlerService) Run(
	ctx context.Context,
	opts crawlerdto.ProductionCrawlOptions,
) (*crawlerdto.ProductionCrawlReport, error) {
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

	report := &crawlerdto.ProductionCrawlReport{
		JobID:         fmt.Sprintf("crawl-%d", startedAt.Unix()),
		StartedAt:     startedAt,
		Workers:       workers,
		TotalQueued:   len(tasks),
		SkippedResume: skippedResume,
	}

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

					res, ingestErr := s.tickIngestor.IngestSample(workerCtx, crawlerdto.TickIngestOptions{
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
								rawJSON, _ := json.Marshal(rec)
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
					mu.Unlock()
				}
			}()
		}

		wg.Wait()
	}

	report.FinishedAt = time.Now().UTC()
	_ = writeStatusReport(dataDir, report)
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
	InstrumentType string
	InstrumentID   uint
	SymbolFA       string
	InsCode        string
	ContractSize   int
}

func (s *ProductionCrawlerService) resolveProductionTargets(opts crawlerdto.ProductionCrawlOptions) ([]productionTarget, error) {
	insFilter := strings.TrimSpace(opts.InsCode)

	if s.db == nil {
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
			if a.TSETMCID == nil || strings.TrimSpace(*a.TSETMCID) == "" || !a.IsActive {
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
		active := true
		contracts, err := s.optionRepo.GetAll(s.db, domainrepo.OptionFilter{IsActive: &active})
		if err != nil {
			return nil, fmt.Errorf("list options for crawl queue: %w", err)
		}
		for _, c := range contracts {
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
			})
		}
	}

	return filterProductionTargets(out, insFilter), nil
}

func (s *ProductionCrawlerService) resolveTargetDates(target productionTarget, daysBack int, fallbackToday string) []string {
	if s.db != nil && s.dailyRepo != nil {
		dates, err := s.dailyRepo.GetTradingDates(s.db, target.InstrumentType, target.InstrumentID, true)
		if err == nil && len(dates) > 0 {
			limit := len(dates)
			if daysBack > 0 && daysBack < limit {
				limit = daysBack
			}
			out := make([]string, 0, limit)
			for i := 0; i < limit; i++ {
				out = append(out, dates[i].Format("20060102"))
			}
			return out
		}
	}
	return []string{fallbackToday}
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
	if (m.Status != "ok" && m.Status != "empty") || m.SHA256 == "" {
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
