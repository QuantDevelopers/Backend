// TSETMC Active Contract Discovery (E1-3) & Tick Sample Ingestion (E2-2) CLI.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	crawlerdto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	domainlogger "github.com/QuantDevelopers/Backend/internal/domain/logger"
	"github.com/QuantDevelopers/Backend/wire"
)

func main() {
	mode := flag.String("mode", "discover", "crawler mode: discover | tick-sample")
	offline := flag.Bool("offline", false, "use local fixture instead of calling live TSETMC")
	persistDB := flag.Bool("persist-db", false, "persist discovered underlyings and active contracts into PostgreSQL")
	fixturePath := flag.String("fixture", "", "optional override path to offline JSON fixture")
	registryDir := flag.String("registry-dir", "data/registry", "output directory for E1-3 registry JSONs")
	dataDir := flag.String("data-dir", "data", "base output directory for E2-2 raw/manifests/clean ticks")
	insCode := flag.String("ins-code", "42511156565416187", "TSETMC insCode for tick-sample mode")
	dateStr := flag.String("date", "20261028", "trading date (YYYYMMDD) for tick-sample mode")
	contractSize := flag.Int("contract-size", 1000, "contract multiplier for tick-sample notional value")
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

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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
		res, err := app.TickIngestorService.IngestSample(ctx, crawlerdto.TickIngestOptions{
			InsCode:           *insCode,
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
	default:
		fmt.Fprintf(os.Stderr, "unsupported mode %q: expected 'discover' or 'tick-sample'\n", *mode)
		os.Exit(2)
	}
}
