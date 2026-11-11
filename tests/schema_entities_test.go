package tests

import (
	"io/fs"
	"strings"
	"testing"
	"time"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/migrations"
)

func strPtr(s string) *string     { return &s }
func floatPtr(f float64) *float64 { return &f }
func intPtr(i int) *int           { return &i }

func TestDomainEntitiesCreationAndFormatting(t *testing.T) {
	ua := entity.UnderlyingAsset{
		ID:         1,
		SymbolFA:   "اهرم",
		NameFA:     "صندوق س. ا. کاریسما-شتاب",
		TSETMCID:   strPtr("24000305886470390"),
		ISIN:       strPtr("IRO1AHRM0001"),
		MarketType: "Bourse",
		IsActive:   true,
	}
	if ua.SymbolFA != "اهرم" || !ua.IsActive || !strings.Contains(ua.String(), "اهرم") {
		t.Fatalf("unexpected UnderlyingAsset: %+v", ua)
	}

	opt := entity.OptionContract{
		ID:                10,
		UnderlyingAssetID: 1,
		SymbolFA:          "ضهرم1204",
		NameFA:            "اختیار خ اهرم-1200-1403/12/04",
		ContractType:      "CALL",
		StrikePrice:       1200.0,
		ContractSize:      1000,
		ExpirationDate:    time.Date(2025, 2, 22, 0, 0, 0, 0, time.UTC),
		IsActive:          true,
		Status:            entity.ContractStatusActive,
	}
	if opt.ContractType != "CALL" || opt.StrikePrice != 1200.0 || opt.Status != "ACTIVE" {
		t.Fatalf("unexpected OptionContract: %+v", opt)
	}
	if !strings.Contains(opt.String(), "ضهرم1204") {
		t.Fatalf("unexpected OptionContract.String(): %s", opt.String())
	}

	daily := entity.DailyHistory{
		ID:              100,
		InstrumentType:  entity.InstrumentTypeOption,
		InstrumentID:    10,
		TradeDate:       time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC),
		OpenPrice:       150.0,
		HighPrice:       180.0,
		LowPrice:        140.0,
		ClosePrice:      175.0,
		SettlementPrice: floatPtr(170.0),
		Volume:          50000,
		OpenInterest:    intPtr(1200),
	}
	if *daily.SettlementPrice != 170.0 || *daily.OpenInterest != 1200 || !strings.Contains(daily.String(), "DailyHistory") {
		t.Fatalf("unexpected DailyHistory: %+v", daily)
	}

	tickRaw := entity.TickRaw{
		ID:             1000,
		InstrumentType: entity.InstrumentTypeOption,
		InstrumentID:   10,
		TradeDate:      time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC),
		TradeTime:      entity.ClockTime{Hour: 9, Minute: 30, Second: 15},
		Price:          175.0,
		Volume:         500,
		SequenceNumber: intPtr(1),
	}
	if tickRaw.TradeTime.String() != "09:30:15" || !strings.Contains(tickRaw.String(), "TickRaw") {
		t.Fatalf("unexpected TickRaw: %+v", tickRaw)
	}

	tickClean := entity.TickClean{
		ID:             1000,
		InstrumentType: entity.InstrumentTypeOption,
		InstrumentID:   10,
		TradeDate:      time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC),
		TradeTimestamp: time.Date(2024, 11, 1, 9, 30, 15, 0, time.UTC),
		SequenceNumber: 1,
		Price:          175.0,
		Volume:         500,
	}
	if tickClean.IsCanceled || !strings.Contains(tickClean.String(), "TickClean") {
		t.Fatalf("unexpected TickClean: %+v", tickClean)
	}

	quality := entity.DataQualityLog{
		ID:             50,
		InstrumentType: entity.InstrumentTypeOption,
		InstrumentID:   10,
		TradeDate:      time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC),
		CheckCategory:  "DUPLICATE_CHECK",
		Status:         entity.QualityStatusPassed,
		GapCount:       0,
	}
	if quality.Status != "PASSED" || !strings.Contains(quality.String(), "DataQualityLog") {
		t.Fatalf("unexpected DataQualityLog: %+v", quality)
	}
}

func TestEmbeddedInitialSQLMigrationContainsAllTables(t *testing.T) {
	sqlBytes, err := fs.ReadFile(migrations.FS, "000001_initial_complete_schema.up.sql")
	if err != nil {
		t.Fatalf("failed to read embedded initial migration: %v", err)
	}
	sqlText := string(sqlBytes)

	requiredTables := []string{
		"underlying_assets",
		"option_contracts",
		"daily_history",
		"ticks_raw",
		"ticks_clean",
		"data_quality_logs",
	}
	for _, tbl := range requiredTables {
		if !strings.Contains(sqlText, "CREATE TABLE IF NOT EXISTS "+tbl) {
			t.Fatalf("embedded migration missing table %q", tbl)
		}
	}

	requiredConstraints := []string{
		"check_contract_type",
		"check_daily_instrument_type",
		"uq_daily_instrument_date",
		"check_raw_instrument_type",
		"check_clean_instrument_type",
		"uq_clean_tick_seq",
		"check_quality_status",
	}
	for _, c := range requiredConstraints {
		if !strings.Contains(sqlText, c) {
			t.Fatalf("embedded migration missing constraint %q", c)
		}
	}
}
