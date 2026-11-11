// TSETMC Active Contract Discovery (E1-3) & Tick Sample Ingestion (E2-2) CLI.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	crawlerdto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	domainlogger "github.com/QuantDevelopers/Backend/internal/domain/logger"
	"github.com/QuantDevelopers/Backend/wire"
)

func main() {
	mode := flag.String("mode", "discover", "crawler mode: discover | tick-sample | daily-history | production")
	offline := flag.Bool("offline", false, "use local fixture instead of calling live TSETMC")
	persistDB := flag.Bool("persist-db", false, "persist crawled records into PostgreSQL")
	fixturePath := flag.String("fixture", "", "optional override path to offline JSON fixture")
	registryDir := flag.String("registry-dir", "data/registry", "output directory for E1-3 registry JSONs")
	dataDir := flag.String("data-dir", "data", "base output directory for raw/manifests/clean artifacts")
	insCode := flag.String("ins-code", "", "TSETMC insCode filter (defaults to 42511156565416187 in tick-sample mode)")
	dateStr := flag.String("date", "20261028", "trading date (YYYYMMDD) for tick-sample mode")
	datesCSV := flag.String("dates", "", "comma-separated YYYYMMDD dates for production mode (empty = derive from daily_history)")
	daysBack := flag.Int("days-back", 5, "max recent trading days per instrument when deriving dates in production mode")
	workers := flag.Int("workers", 3, "concurrent worker goroutines for production mode")
	resume := flag.Bool("resume", true, "skip completed (insCode, date) tasks with valid manifests in production mode")
	force := flag.Bool("force", false, "force re-crawling even when valid manifest exists in production mode")
	maxTasks := flag.Int("max-tasks", 0, "optional cap on queued tasks in production mode (0 = unlimited)")
	contractSize := flag.Int("contract-size", 1000, "contract multiplier for tick-sample notional value")
	days := flag.Int("days", 0, "number of daily candles for daily-history mode (0 = full available history)")
	includeUnderlyings := flag.Bool("underlyings", true, "include underlying assets in daily-history/production mode")
	includeOptions := flag.Bool("options", true, "include active options in daily-history/production mode")
	limit := flag.Int("limit", 0, "optional cap on number of instruments in daily-history mode")
	saveRaw := flag.Bool("save-raw", true, "save raw daily payloads under <data-dir>/raw/daily")
	timeout := flag.Duration("timeout", 15*time.Minute, "crawler execution timeout")
	flag.Parse()

	config := bootstrap.Run()
	var (
		app *wire.CrawlerApplication
		err error
	)
	if *persistDB {
		app, err = wire.InitializeCrawlerApplication(config)
	} else {
		app, err = wire.InitializeOfflineCrawlerApplication(config)
	}
	if err != nil {
		panic(err)
	}
	defer app.Logger.Close()

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	switch *mode {
	case "discover":
		res, err := app.DiscoveryService.Discover(ctx, crawlerdto.DiscoveryOptions{
			UseOfflineFixture: *offline,
			FixturePath:       *fixturePath,
			RegistryDir:       *registryDir,
			PersistDB:         *persistDB,
		})
		if err != nil {
			app.Logger.Fatal("E1-3 contract discovery failed", domainlogger.Error("error", err))
		}
		fmt.Printf(
			"E1-3 Discovery complete: underlyings=%d, active_contracts=%d, db_rows=%d\n  -> %s\n  -> %s\n",
			len(res.Underlyings),
			len(res.Contracts),
			res.PersistedDBRows,
			res.UnderlyingsPath,
			res.ContractsPath,
		)
	case "tick-sample":
		targetIns := *insCode
		if targetIns == "" {
			targetIns = "42511156565416187"
		}
		res, err := app.TickIngestorService.IngestSample(ctx, crawlerdto.TickIngestOptions{
			InsCode:           targetIns,
			DateStr:           *dateStr,
			ContractSize:      *contractSize,
			UseOfflineFixture: *offline,
			FixturePath:       *fixturePath,
			DataDir:           *dataDir,
		})
		if err != nil {
			app.Logger.Fatal("E2-2 tick sample ingestion failed", domainlogger.Error("error", err))
		}
		fmt.Printf(
			"E2-2 Tick Ingestion complete: insCode=%s, date=%s, status=%s, records=%d, sha256=%s\n  raw:      %s\n  manifest: %s\n  clean:    %s\n",
			res.Manifest.InsCode,
			res.Manifest.Date,
			res.Manifest.Status,
			res.Manifest.RecordsCount,
			res.Manifest.SHA256,
			res.RawPath,
			res.ManifestPath,
			res.CleanPath,
		)
	case "daily-history":
		res, err := app.DailyHistoryCrawler.Crawl(ctx, crawlerdto.DailyHistoryCrawlOptions{
			TopDays:            *days,
			IncludeUnderlyings: *includeUnderlyings,
			IncludeOptions:     *includeOptions,
			InsCode:            *insCode,
			Limit:              *limit,
			SaveRaw:            *saveRaw,
			RawDir:             filepath.Join(*dataDir, "raw", "daily"),
			UseOfflineFixture:  *offline,
		})
		if err != nil {
			app.Logger.Fatal("E2-1 daily history crawl failed", domainlogger.Error("error", err))
		}
		fmt.Printf(
			"E2-1 Daily History complete: targets=%d, succeeded=%d, failed=%d, rows_parsed=%d, rows_saved=%d, circuit_broken=%v\n",
			res.InstrumentsTotal,
			res.InstrumentsSucceeded,
			res.InstrumentsFailed,
			res.RowsParsed,
			res.RowsSaved,
			res.CircuitBroken,
		)
	case "production":
		var parsedDates []string
		for _, d := range strings.Split(*datesCSV, ",") {
			if trimmed := strings.TrimSpace(d); trimmed != "" {
				parsedDates = append(parsedDates, trimmed)
			}
		}
		report, err := app.ProductionCrawler.Run(ctx, crawlerdto.ProductionCrawlOptions{
			Workers:            *workers,
			DataDir:            *dataDir,
			Dates:              parsedDates,
			DaysBack:           *daysBack,
			IncludeUnderlyings: *includeUnderlyings,
			IncludeOptions:     *includeOptions,
			InsCode:            *insCode,
			MaxTasks:           *maxTasks,
			Resume:             *resume,
			Force:              *force,
			PersistDB:          *persistDB,
			UseOfflineFixture:  *offline,
			FixturePath:        *fixturePath,
		})
		if err != nil {
			app.Logger.Fatal("E2-3 production crawl failed", domainlogger.Error("error", err))
		}
		fmt.Printf(
			"E2-3 Production Crawl complete: job=%s, workers=%d, queued=%d, skipped_resume=%d, succeeded=%d, empty=%d, failed=%d, records=%d, db_saved=%d, circuit_broken=%v\n",
			report.JobID,
			report.Workers,
			report.TotalQueued,
			report.SkippedResume,
			report.Succeeded,
			report.EmptyDays,
			report.Failed,
			report.TotalRecords,
			report.DBRowsSaved,
			report.CircuitBroken,
		)
	default:
		fmt.Fprintf(os.Stderr, "unsupported mode %q: expected 'discover', 'tick-sample', 'daily-history', or 'production'\n", *mode)
		os.Exit(2)
	}
}
