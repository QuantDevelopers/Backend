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

	if err := infrapostgres.MigrateInstrumentIdentity(pgDB.GetDB()); err != nil {
		t.Fatal(err)
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

	// Bulk status transitions must update existing rows and preserve false,
	// even though the entity has a database/default GORM value of true.
	opt.IsActive = false
	opt.Status = "EXPIRED"
	if _, err := optRepo.SaveBulk(pgDB, []*entity.OptionContract{opt}); err != nil {
		t.Fatal(err)
	}
	expired, err := optRepo.GetBySymbol(pgDB, opt.SymbolFA)
	if err != nil || expired == nil || expired.IsActive || expired.Status != "EXPIRED" {
		t.Fatalf("bulk expiry transition failed: %+v %v", expired, err)
	}
	if err = optRepo.Save(pgDB, opt); err != nil {
		t.Fatal(err)
	}
	expired, err = optRepo.GetBySymbol(pgDB, opt.SymbolFA)
	if err != nil || expired == nil || expired.IsActive {
		t.Fatalf("single save changed false active flag: %+v %v", expired, err)
	}
	// Reused symbols remain separate identities across historical expiries.
	oldCode, newCode := "991000000000001", "991000000000002"
	oldContract := &entity.OptionContract{UnderlyingAssetID: ahram.ID, SymbolFA: "reused_history_test", NameFA: "old", ContractType: "CALL", StrikePrice: 100, ContractSize: 1000, ExpirationDate: time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC), TSETMCID: &oldCode, IsActive: false, Status: "EXPIRED"}
	newContract := &entity.OptionContract{UnderlyingAssetID: ahram.ID, SymbolFA: "reused_history_test", NameFA: "new", ContractType: "CALL", StrikePrice: 200, ContractSize: 1000, ExpirationDate: time.Date(2027, 12, 1, 0, 0, 0, 0, time.UTC), TSETMCID: &newCode, IsActive: true, Status: "ACTIVE"}
	for i := 0; i < 2; i++ {
		if _, err := optRepo.SaveBulk(pgDB, []*entity.OptionContract{oldContract, newContract}); err != nil {
			t.Fatal(err)
		}
	}
	var reused int64
	if err := pgDB.GetDB().Model(&entity.OptionContract{}).Where("symbol_fa = ?", "reused_history_test").Count(&reused).Error; err != nil || reused != 2 {
		t.Fatalf("reused symbol identities: count=%d err=%v", reused, err)
	}
	latest, err := optRepo.GetBySymbol(pgDB, "reused_history_test")
	if err != nil || latest == nil || latest.TSETMCID == nil || *latest.TSETMCID != newCode {
		t.Fatalf("symbol query must prefer active identity: %+v %v", latest, err)
	}
	dailyRepo := infrapostgres.NewDailyHistoryRepository()
	yPrice := 70744.0
	dailyRecords := []*entity.DailyHistory{
		{
			InstrumentType: "UNDERLYING",
			InstrumentID:   ahram.ID,
			TradeDate:      time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC),
			OpenPrice:      69869,
			HighPrice:      73070,
			LowPrice:       69205,
			ClosePrice:     71666,
			YesterdayPrice: &yPrice,
			Volume:         162591305,
			Value:          11652324145553,
			TradeCount:     20749,
		},
		{
			InstrumentType: "UNDERLYING",
			InstrumentID:   ahram.ID,
			TradeDate:      time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC),
			OpenPrice:      69900,
			HighPrice:      73070,
			LowPrice:       69205,
			ClosePrice:     71700,
			YesterdayPrice: &yPrice,
			Volume:         162591305,
			Value:          11652324145553,
			TradeCount:     20749,
		},
	}
	if _, err := dailyRepo.SaveBulk(pgDB, dailyRecords); err != nil {
		t.Fatalf("save daily history bulk: %v", err)
	}

	dates, err := dailyRepo.GetTradingDates(pgDB, "UNDERLYING", ahram.ID, true)
	if err != nil {
		t.Fatalf("GetTradingDates: %v", err)
	}
	if len(dates) == 0 {
		t.Fatal("expected at least 1 persisted trading date")
	}

	tickRawRepo := infrapostgres.NewTickRawRepository()
	seq1 := 1
	day := time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC)
	ticks := []*entity.TickRaw{
		{
			InstrumentType: "UNDERLYING",
			InstrumentID:   ahram.ID,
			TradeDate:      day,
			TradeTime:      "09:00:05",
			SequenceNumber: &seq1,
			Price:          72500,
			Volume:         100,
			RawPayload:     []byte(`{"nTran":1}`),
		},
	}
	if _, err := tickRawRepo.ReplaceDayTicks(pgDB, "UNDERLYING", ahram.ID, day, ticks); err != nil {
		t.Fatalf("ReplaceDayTicks initial: %v", err)
	}
	if _, err := tickRawRepo.ReplaceDayTicks(pgDB, "UNDERLYING", ahram.ID, day, ticks); err != nil {
		t.Fatalf("ReplaceDayTicks idempotent: %v", err)
	}
	loadedTicks, totalTicks, err := tickRawRepo.GetByInstrumentAndDate(pgDB, "UNDERLYING", ahram.ID, day, 0, 10)
	if err != nil {
		t.Fatalf("GetByInstrumentAndDate: %v", err)
	}
	if totalTicks != 1 || len(loadedTicks) != 1 {
		t.Fatalf("expected 1 idempotent tick_raw row, got total=%d len=%d", totalTicks, len(loadedTicks))
	}
}
