package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/domain/enum"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/seed"
)

func strPtr(s string) *string     { return &s }
func floatPtr(f float64) *float64 { return &f }
func intPtr(i int) *int           { return &i }

func TestDomainEntitiesCreationAndFormatting(t *testing.T) {
	ua := entity.UnderlyingAsset{
		Model:      database.Model{ID: 1},
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
		Model:             database.Model{ID: 10},
		UnderlyingAssetID: 1,
		SymbolFA:          "ضهرم1204",
		NameFA:            "اختیار خ اهرم-1200-1403/12/04",
		ContractType:      string(enum.ContractTypeCall),
		StrikePrice:       1200.0,
		ContractSize:      1000,
		ExpirationDate:    time.Date(2025, 2, 22, 0, 0, 0, 0, time.UTC),
		IsActive:          true,
		Status:            string(enum.ContractStatusActive),
	}
	if opt.ContractType != "CALL" || opt.StrikePrice != 1200.0 || opt.Status != "ACTIVE" {
		t.Fatalf("unexpected OptionContract: %+v", opt)
	}
	if !strings.Contains(opt.String(), "ضهرم1204") {
		t.Fatalf("unexpected OptionContract.String(): %s", opt.String())
	}

	daily := entity.DailyHistory{
		Model:           database.Model{ID: 100},
		InstrumentType:  string(enum.InstrumentTypeOption),
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
		Model:          database.Model{ID: 1000},
		InstrumentType: string(enum.InstrumentTypeOption),
		InstrumentID:   10,
		TradeDate:      time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC),
		TradeTime:      entity.ClockTime{Hour: 9, Minute: 30, Second: 15}.String(),
		Price:          175.0,
		Volume:         500,
		SequenceNumber: intPtr(1),
	}
	if tickRaw.TradeTime != "09:30:15" || !strings.Contains(tickRaw.String(), "TickRaw") {
		t.Fatalf("unexpected TickRaw: %+v", tickRaw)
	}

	tickClean := entity.TickClean{
		Model:          database.Model{ID: 1000},
		InstrumentType: string(enum.InstrumentTypeOption),
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
		Model:          database.Model{ID: 50},
		InstrumentType: string(enum.InstrumentTypeOption),
		InstrumentID:   10,
		TradeDate:      time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC),
		CheckCategory:  "DUPLICATE_CHECK",
		Status:         string(enum.QualityStatusPassed),
		GapCount:       0,
	}
	if quality.Status != "PASSED" || !strings.Contains(quality.String(), "DataQualityLog") {
		t.Fatalf("unexpected DataQualityLog: %+v", quality)
	}
}

func TestDefault25UnderlyingsSeedData(t *testing.T) {
	if len(seed.Default25Underlyings) != 25 {
		t.Fatalf("len(Default25Underlyings) = %d, want 25", len(seed.Default25Underlyings))
	}
	seen := make(map[string]bool, 25)
	for _, item := range seed.Default25Underlyings {
		if item.SymbolFA == "" || item.NameFA == "" {
			t.Fatalf("empty field in default underlying: %+v", item)
		}
		if seen[item.SymbolFA] {
			t.Fatalf("duplicate symbol_fa %q", item.SymbolFA)
		}
		seen[item.SymbolFA] = true
	}
}
