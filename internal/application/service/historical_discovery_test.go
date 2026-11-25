package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	dto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	clientmock "github.com/QuantDevelopers/Backend/mocks/infrastructure/tsetmc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func historyFixture(t *testing.T, dir, name, raw string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(raw), 0o644))
}
func historyOptions(t *testing.T) dto.FullRegistryOptions {
	t.Helper()
	dir := t.TempDir()
	historyFixture(t, dir, "watch.json", `{"instrumentOptMarketWatch":[]}`)
	return dto.FullRegistryOptions{UseOfflineFixture: true, FixturePath: filepath.Join(dir, "watch.json"), RegistryDir: t.TempDir(), IncludeExpired: true, HistoryDates: []string{"20241027", "20241026"}, HistoryFixtureDir: dir}
}

func TestHistoricalDiscoveryFindsUnknownExpiredOptionsAndBases(t *testing.T) {
	opts := historyOptions(t)
	historyFixture(t, opts.HistoryFixtureDir, "20241027.json", `{"closingPriceDailyHistoryWithInstDetails":[
 {"instrument":{"insCode":9007199254740993,"lVal18AFC":"قدیمی","lVal30":"سهم قدیمی","cIsin":"IRO100001"}},
 {"insCode":9007199254740994,"instrument":{"lVal18AFC":"ضقدیم1","lVal30":"اختيارخ قدیمی-1,000-2024/10/30","cIsin":"IRO900001","contractSize":500}}
 ]}`)
	historyFixture(t, opts.HistoryFixtureDir, "20241026.json", `{"closingPriceDailyHistoryWithInstDetails":[
 {"insCode":"9007199254740994","lVal18AFC":"ضقدیم1","lVal30":"اختیارخ قدیمی-1000-2024/10/30"},
 {"insCode":"9007199254740995","lVal18AFC":"طقدیم1","lVal30":"اختیارف قدیمی-1000-2024/10/30"}
 ]}`)
	s := NewDiscoveryService(nil, nil, nil, nil, nil)
	res, err := s.DiscoverFullRegistry(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, res.Underlyings, 1)
	require.Len(t, res.ExpiredContracts, 2)
	require.Empty(t, res.ActiveContracts)
	require.Equal(t, "9007199254740993", res.Underlyings[0].InsCode)
	require.Equal(t, 2, res.Summary.HistoricalOptionsAdded)
	require.Equal(t, 2, res.Summary.HistoryDatesCompleted)
	require.Equal(t, 500, res.ExpiredContracts[0].ContractSize)
	require.False(t, res.ExpiredContracts[0].ContractSizeEstimated)
	require.True(t, res.ExpiredContracts[1].ContractSizeEstimated)
	require.Equal(t, 1, res.Summary.EstimatedContractSizes)
	require.Equal(t, "9007199254740993", res.ExpiredContracts[1].UaInsCode)
	require.Equal(t, "20241030", res.ExpiredContracts[0].ExpiryDate)
	require.Equal(t, "IRO900001", res.ExpiredContracts[0].ISIN)
}

func TestHistoricalUnderlyingUsesExactSearchAndRejectsAmbiguity(t *testing.T) {
	opts := historyOptions(t)
	opts.HistoryDates = []string{"20241026"}
	historyFixture(t, opts.HistoryFixtureDir, "20241026.json", `{"closingPriceDailyHistoryWithInstDetails":[{"insCode":"123","lVal18AFC":"ضقدیم1","lVal30":"اختیارخ قدیمی-1000-2024/10/30"}]}`)
	historyFixture(t, opts.HistoryFixtureDir, "search_قدیمی.json", `{"instrumentSearch":[{"insCode":"456","lVal18AFC":"قدیمی","lVal30":"سهم قدیمی","cIsin":"IRO10001"},{"insCode":"789","lVal18AFC":"قدیمی2","lVal30":"سهم مشابه"}]}`)
	s := NewDiscoveryService(nil, nil, nil, nil, nil)
	res, err := s.DiscoverFullRegistry(context.Background(), opts)
	require.NoError(t, err)
	require.Equal(t, "456", res.ExpiredContracts[0].UaInsCode)
	historyFixture(t, opts.HistoryFixtureDir, "search_قدیمی.json", `{"instrumentSearch":[]}`)
	opts.RegistryDir = t.TempDir()
	_, err = s.DiscoverFullRegistry(context.Background(), opts)
	require.ErrorContains(t, err, "exact identity")
	_, err = os.Stat(filepath.Join(opts.RegistryDir, "registry_summary.json"))
	require.True(t, os.IsNotExist(err))
}

func TestHistoricalArchiveCacheChecksumsAndFailClosed(t *testing.T) {
	c := clientmock.NewTSETMCClientMock()
	day := "20241026"
	dir := t.TempDir()
	c.On("Get", mock.Anything, TSETMCOptionWatchURL).Return([]byte(`{"instrumentOptMarketWatch":[]}`), http.StatusOK, nil)
	archive := []byte(`{"closingPriceDailyHistoryWithInstDetails":[]}`)
	c.On("Get", mock.Anything, fmt.Sprintf(archiveURL, day)).Return(archive, http.StatusOK, nil).Twice()
	s := NewDiscoveryService(c, nil, nil, nil, nil)
	opts := dto.FullRegistryOptions{HistoryDates: []string{day}, HistoryCacheDir: dir, RegistryDir: t.TempDir()}
	_, err := s.DiscoverFullRegistry(context.Background(), opts)
	require.NoError(t, err)
	_, err = s.DiscoverFullRegistry(context.Background(), opts)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, day+".json"), []byte(`{}`), 0o644))
	_, err = s.DiscoverFullRegistry(context.Background(), opts)
	require.NoError(t, err)
	c.AssertExpectations(t)
	opts.HistoryDates = []string{"20241027"}
	opts.RegistryDir = t.TempDir()
	c.On("Get", mock.Anything, fmt.Sprintf(archiveURL, "20241027")).Return([]byte(`{"error":"not available"}`), http.StatusOK, nil)
	_, err = s.DiscoverFullRegistry(context.Background(), opts)
	require.Error(t, err)
	_, err = os.Stat(filepath.Join(opts.RegistryDir, "registry_summary.json"))
	require.True(t, os.IsNotExist(err))
}

func TestJalaliOptionExpiryValidation(t *testing.T) {
	for raw, want := range map[string]string{"1403/01/01": "20240320", "1404/01/01": "20250321", "1405/01/01": "20260321", "1403/12/30": "20250320", "1405/10/30": "20270120"} {
		got, err := optionExpiry(raw)
		require.NoError(t, err, raw)
		require.Equal(t, want, got)
	}
	for _, bad := range []string{"1404/12/30", "1405/07/31", "1405/13/01", "2026/02/30"} {
		_, err := optionExpiry(bad)
		require.Error(t, err, bad)
	}
	_, _, _, _, err := parseHistoricalOption("اختیارخ سهم بدون قیمت")
	require.Error(t, err)
	opts := dto.FullRegistryOptions{HistoryStartDate: "20261028", HistoryEndDate: "20261026"}
	_, err = historicalDates(opts)
	require.Error(t, err)
}

func TestHistoricalMappingUsesSameDayBaseWhenTickerIsReused(t *testing.T) {
	opts := historyOptions(t)
	for day, ids := range map[string][2]string{"20241027": {"111", "222"}, "20241026": {"333", "444"}} {
		historyFixture(t, opts.HistoryFixtureDir, day+".json", fmt.Sprintf(`{"closingPriceDailyHistoryWithInstDetails":[{"insCode":"%s","lVal18AFC":"پایه","lVal30":"پایه","cIsin":"IRO1001"},{"insCode":"%s","lVal18AFC":"ضپایه1","lVal30":"اختیار خ پایه-1000-2024/10/30"}]}`, ids[0], ids[1]))
	}
	res, err := NewDiscoveryService(nil, nil, nil, nil, nil).DiscoverFullRegistry(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, res.Underlyings, 2)
	require.Len(t, res.ExpiredContracts, 2)
	require.Equal(t, "111", res.ExpiredContracts[0].UaInsCode)
	require.Equal(t, "333", res.ExpiredContracts[1].UaInsCode)
}
