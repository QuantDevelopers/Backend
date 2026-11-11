package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	domainrepo "github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	infrapostgres "github.com/QuantDevelopers/Backend/internal/infrastructure/repository/postgres"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/seed"
)

func TestPostgresAutoMigrateSeedAndRepositories(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping live PostgreSQL integration test")
	}

	dbCfg := &bootstrap.Database{
		URL:         bootstrap.NormalizeDatabaseURL(dsn),
		PoolSize:    5,
		MaxOverflow: 5,
	}

	pgDB, err := database.NewPostgresDatabase(dbCfg)
	if err != nil {
		t.Fatalf("connect test postgres: %v", err)
	}

	if err := pgDB.Ping(context.Background()); err != nil {
		t.Fatalf("ping test postgres: %v", err)
	}

	if err := pgDB.GetDB().AutoMigrate(
		&entity.UnderlyingAsset{},
		&entity.OptionContract{},
		&entity.DailyHistory{},
		&entity.TickRaw{},
		&entity.TickClean{},
		&entity.DataQualityLog{},
	); err != nil {
		t.Fatalf("auto-migrate schema: %v", err)
	}

	uaRepo := infrapostgres.NewUnderlyingRepository()
	seeder := seed.NewUnderlyingSeeder(uaRepo, pgDB)

	// Seeding twice must stay idempotent at 25 default underlyings.
	seeder.SeedDefaultUnderlyings()
	seeder.SeedDefaultUnderlyings()

	assets, err := uaRepo.GetAll(pgDB)
	if err != nil {
		t.Fatalf("GetAll underlyings: %v", err)
	}
	if len(assets) < 25 {
		t.Fatalf("expected at least 25 seeded underlyings, got %d", len(assets))
	}

	ahram, err := uaRepo.GetBySymbol(pgDB, "اهرم")
	if err != nil || ahram == nil {
		t.Fatalf("expected underlying 'اهرم', got asset=%+v err=%v", ahram, err)
	}

	optRepo := infrapostgres.NewOptionContractRepository()
	opt := &entity.OptionContract{
		UnderlyingAssetID: ahram.ID,
		SymbolFA:          "ضهرم_تست_CI",
		NameFA:            "اختیار تست CI",
		ContractType:      "CALL",
		StrikePrice:       50000,
		ContractSize:      1000,
		ExpirationDate:    time.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC),
		IsActive:          true,
		Status:            "ACTIVE",
	}
	if err := optRepo.Save(pgDB, opt); err != nil {
		t.Fatalf("save option contract: %v", err)
	}

	cType := "CALL"
	active := true
	opts, err := optRepo.GetAll(pgDB, domainrepo.OptionFilter{
		UnderlyingAssetID: &ahram.ID,
		ContractType:      &cType,
		IsActive:          &active,
	})
	if err != nil {
		t.Fatalf("GetAll option contracts: %v", err)
	}
	if len(opts) == 0 {
		t.Fatal("expected at least 1 saved option contract")
	}
}
