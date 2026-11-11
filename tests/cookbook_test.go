package tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func repoPath(parts ...string) string {
	all := append([]string{".."}, parts...)
	return filepath.Join(all...)
}

func TestCookbookSampleFilesExistAndValid(t *testing.T) {
	samplesDir := repoPath("data", "raw", "samples")
	info, err := os.Stat(samplesDir)
	if err != nil || !info.IsDir() {
		t.Fatalf("data/raw/samples directory must exist: %v", err)
	}

	expectedFiles := []string{
		"sample_underlying_identity.json",
		"sample_underlying_daily.json",
		"sample_underlying_ticks.json",
		"sample_option_search.json",
		"sample_option_daily.json",
		"sample_option_ticks.json",
	}

	for _, filename := range expectedFiles {
		fp := filepath.Join(samplesDir, filename)
		st, err := os.Stat(fp)
		if err != nil {
			t.Fatalf("sample file %s must exist: %v", filename, err)
		}
		if st.Size() == 0 {
			t.Fatalf("sample file %s must not be empty", filename)
		}

		raw, err := os.ReadFile(fp)
		if err != nil {
			t.Fatalf("read %s: %v", filename, err)
		}
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Fatalf("sample file %s must be valid JSON object: %v", filename, err)
		}
	}
}

func TestSampleTicksAndDailyStructures(t *testing.T) {
	ticksRaw, err := os.ReadFile(repoPath("data", "raw", "samples", "sample_underlying_ticks.json"))
	if err != nil {
		t.Fatalf("read sample_underlying_ticks.json: %v", err)
	}

	var ticksData struct {
		TradeHistory []map[string]any `json:"tradeHistory"`
	}
	if err := json.Unmarshal(ticksRaw, &ticksData); err != nil {
		t.Fatalf("unmarshal ticks JSON: %v", err)
	}
	if len(ticksData.TradeHistory) == 0 {
		t.Fatal("expected non-empty tradeHistory in sample_underlying_ticks.json")
	}

	firstTick := ticksData.TradeHistory[0]
	for _, field := range []string{"pTran", "qTitTran", "hEven"} {
		if _, ok := firstTick[field]; !ok {
			t.Fatalf("expected field %q in first tick", field)
		}
	}

	searchRaw, err := os.ReadFile(repoPath("data", "raw", "samples", "sample_option_search.json"))
	if err != nil {
		t.Fatalf("read sample_option_search.json: %v", err)
	}

	var searchData struct {
		InstrumentSearch []map[string]any `json:"instrumentSearch"`
	}
	if err := json.Unmarshal(searchRaw, &searchData); err != nil {
		t.Fatalf("unmarshal search JSON: %v", err)
	}
	if len(searchData.InstrumentSearch) == 0 {
		t.Fatal("expected non-empty instrumentSearch in sample_option_search.json")
	}

	firstOpt := searchData.InstrumentSearch[0]
	for _, field := range []string{"insCode", "lVal18AFC"} {
		if _, ok := firstOpt[field]; !ok {
			t.Fatalf("expected field %q in first option search item", field)
		}
	}
}
