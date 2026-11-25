package service

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	crawlerdto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	clientmock "github.com/QuantDevelopers/Backend/mocks/infrastructure/tsetmc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLiveFailuresCannotSubstituteFixture(t *testing.T) {
	c := clientmock.NewTSETMCClientMock()
	c.On("Get", mock.Anything, mock.Anything).Return([]byte(nil), http.StatusServiceUnavailable, errors.New("upstream failed"))
	fixture := filepath.Join(t.TempDir(), "valid.json")
	require.NoError(t, os.WriteFile(fixture, []byte(`{"tradeHistory":[]}`), 0o644))
	ingestor := NewTickIngestorService(c, nil)
	_, err := ingestor.IngestSample(context.Background(), crawlerdto.TickIngestOptions{InsCode: "123", DateStr: "20261028", FixturePath: fixture, DataDir: t.TempDir()})
	require.Error(t, err)
	d := NewDiscoveryService(c, nil, nil, nil, nil)
	_, err = d.Discover(context.Background(), crawlerdto.DiscoveryOptions{FixturePath: fixture, RegistryDir: t.TempDir()})
	require.Error(t, err)
	daily := NewDailyHistoryCrawlerService(c, nil, nil, nil, nil, nil)
	_, _, err = daily.fetchDailyPayload(context.Background(), dailyTarget{InsCode: "123", InstrumentType: "UNDERLYING"}, crawlerdto.DailyHistoryCrawlOptions{UnderlyingFixturePath: fixture})
	require.Error(t, err)
}

func TestMalformedPayloadsAndUnsafeIdentifiersFail(t *testing.T) {
	ingestor := NewTickIngestorService(nil, nil)
	for _, raw := range []string{`{}`, `null`, `{"tradeHistory":null}`, `{"error":"unavailable"}`, `{"tradeHistory":[{"nTran":1,"hEven":246001}]}`} {
		_, err := ingestor.ParseTickPayload([]byte(raw), "123", "20261028", 1)
		require.Error(t, err, raw)
	}
	for _, opts := range []crawlerdto.TickIngestOptions{{InsCode: "../escape", DateStr: "20261028"}, {InsCode: "123", DateStr: "20260230"}} {
		_, err := ingestor.IngestSample(context.Background(), opts)
		require.Error(t, err)
	}
	records, err := ingestor.ParseTickPayload([]byte(`{"tradeHistory":[]}`), "123", "20261028", 1)
	require.NoError(t, err)
	require.Empty(t, records)
}

func TestManifestIdentityAndCalendarValidation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "calendar.json")
	require.NoError(t, os.WriteFile(path, []byte(`[{"date":"20260230","is_trading_day":true}]`), 0o644))
	_, err := loadTradingCalendarDates(path, "", "")
	require.Error(t, err)
	require.NoError(t, os.WriteFile(path, []byte(`[{"date":"20261028","is_trading_day":true},{"date":"20261028","is_trading_day":true},{"date":"20261027","is_trading_day":false}]`), 0o644))
	dates, err := loadTradingCalendarDates(path, "", "")
	require.NoError(t, err)
	require.Equal(t, []string{"20261028"}, dates)
	ingestor := NewTickIngestorService(nil, nil)
	fixture := filepath.Join(dir, "ticks.json")
	require.NoError(t, os.WriteFile(fixture, []byte(`{"tradeHistory":[]}`), 0o644))
	result, err := ingestor.IngestSample(context.Background(), crawlerdto.TickIngestOptions{InsCode: "123", DateStr: "20261028", UseOfflineFixture: true, FixturePath: fixture, DataDir: dir})
	require.NoError(t, err)
	require.True(t, taskReusable(dir, "123", "20261028", true))
	require.False(t, taskReusable(dir, "123", "20261028", false))
	result.Manifest.InsCode = "456"
	require.NoError(t, writeJSONPretty(result.ManifestPath, result.Manifest))
	require.False(t, isTaskCompletedOnDisk(dir, "123", "20261028"))
}
