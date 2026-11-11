package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	crawlerdto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	domainlogger "github.com/QuantDevelopers/Backend/internal/domain/logger"
	domaintsetmc "github.com/QuantDevelopers/Backend/internal/domain/tsetmc"
)

const (
	// TSETMCTradeURLToday fetches intraday trades for today.
	TSETMCTradeURLToday = "https://cdn.tsetmc.com/api/Trade/GetTrade/%s"
	// TSETMCTradeURLHistory fetches full historical tick trades for an insCode on YYYYMMDD.
	TSETMCTradeURLHistory = "https://cdn.tsetmc.com/api/Trade/GetTradeHistory/%s/%s/true"

	defaultTickFixture = "tests/fixtures/sample_trade_history.json"
	defaultDataBaseDir = "data"
	tickSourceLabel    = "tsetmc_cdn_trade_api"
)

// TickIngestorService implements E2-2 raw-first tick ingestion, manifest metadata, and cleaning.
type TickIngestorService struct {
	client domaintsetmc.Client
	logger domainlogger.Logger
}

// NewTickIngestorService constructs a TickIngestorService instance.
func NewTickIngestorService(client domaintsetmc.Client, log domainlogger.Logger) *TickIngestorService {
	return &TickIngestorService{
		client: client,
		logger: log,
	}
}

// ComputeSHA256 returns the lowercase hexadecimal SHA-256 checksum of raw payload bytes.
func ComputeSHA256(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// FormatHEven formats a TSETMC integer hEven (e.g. 90005 or 103806) into zero-padded HH:MM:SS.
func FormatHEven(heven int) string {
	if heven <= 0 {
		return "00:00:00"
	}
	padded := fmt.Sprintf("%06d", heven)
	if len(padded) > 6 {
		padded = padded[:6]
	}
	return padded[0:2] + ":" + padded[2:4] + ":" + padded[4:6]
}

func (s *TickIngestorService) ParseTickPayload(
	raw []byte,
	insCode string,
	dateStr string,
	contractSize int,
) ([]crawlerdto.CleanTickRecord, error) {
	var payload crawlerdto.TradePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode trade tick JSON: %w", err)
	}

	if contractSize <= 0 {
		contractSize = 1000
	}

	items := payload.TradeHistory
	if len(items) == 0 {
		items = payload.Trade
	}

	clean := make([]crawlerdto.CleanTickRecord, 0, len(items))
	for _, item := range items {
		notional := item.PTran * float64(item.QTitTran) * float64(contractSize)
		clean = append(clean, crawlerdto.CleanTickRecord{
			InsCode:       insCode,
			Date:          dateStr,
			NTran:         item.NTran,
			HEven:         item.HEven,
			Time:          FormatHEven(item.HEven),
			Price:         item.PTran,
			Quantity:      item.QTitTran,
			ContractSize:  contractSize,
			NotionalValue: notional,
			IsCanceled:    item.Canceled != 0,
		})
	}

	sort.Slice(clean, func(i, j int) bool {
		return clean[i].NTran < clean[j].NTran
	})

	return clean, nil
}

func (s *TickIngestorService) IngestSample(
	ctx context.Context,
	opts crawlerdto.TickIngestOptions,
) (*crawlerdto.TickIngestResult, error) {
	insCode := strings.TrimSpace(opts.InsCode)
	if insCode == "" {
		insCode = "42511156565416187"
	}
	dateStr := strings.TrimSpace(opts.DateStr)
	if dateStr == "" {
		dateStr = time.Now().UTC().Format("20060102")
	}
	contractSize := opts.ContractSize
	if contractSize <= 0 {
		contractSize = 1000
	}
	fixturePath := opts.FixturePath
	if fixturePath == "" {
		fixturePath = defaultTickFixture
	}
	dataDir := opts.DataDir
	if dataDir == "" {
		dataDir = defaultDataBaseDir
	}

	rawBytes, err := s.fetchRawTicks(ctx, insCode, opts.DateStr, opts.UseOfflineFixture, fixturePath, opts.DisableFallback)
	if err != nil {
		return nil, err
	}

	records, err := s.ParseTickPayload(rawBytes, insCode, dateStr, contractSize)
	if err != nil {
		return nil, err
	}

	rawInsDir := filepath.Join(dataDir, "raw", "ticks", insCode)
	manifestDir := filepath.Join(dataDir, "manifests", "ticks")
	cleanDir := filepath.Join(dataDir, "clean", "ticks")
	for _, dir := range []string{rawInsDir, manifestDir, cleanDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create output dir %s: %w", dir, err)
		}
	}

	rawPath := filepath.Join(rawInsDir, dateStr+".json")
	if err := os.WriteFile(rawPath, rawBytes, 0o644); err != nil {
		return nil, fmt.Errorf("write raw tick payload %s: %w", rawPath, err)
	}

	status := "ok"
	if len(records) == 0 {
		status = "empty"
	}

	relRawPath := rawPath
	if rel, relErr := filepath.Rel(dataDir, rawPath); relErr == nil {
		relRawPath = filepath.ToSlash(filepath.Join("data", rel))
	}

	manifest := crawlerdto.TickManifest{
		InsCode:      insCode,
		Date:         dateStr,
		Status:       status,
		RecordsCount: len(records),
		RawSizeBytes: len(rawBytes),
		SHA256:       ComputeSHA256(rawBytes),
		FetchedAt:    time.Now().UTC(),
		RawPath:      relRawPath,
		Source:       tickSourceLabel,
	}

	manifestPath := filepath.Join(manifestDir, fmt.Sprintf("%s_%s_manifest.json", insCode, dateStr))
	if err := writeJSONPretty(manifestPath, manifest); err != nil {
		return nil, err
	}

	cleanPath := filepath.Join(cleanDir, fmt.Sprintf("%s_%s.json", insCode, dateStr))
	if err := writeJSONPretty(cleanPath, records); err != nil {
		return nil, err
	}

	return &crawlerdto.TickIngestResult{
		Manifest:     manifest,
		Records:      records,
		RawPath:      rawPath,
		ManifestPath: manifestPath,
		CleanPath:    cleanPath,
	}, nil
}

func (s *TickIngestorService) fetchRawTicks(
	ctx context.Context,
	insCode string,
	explicitDate string,
	useOffline bool,
	fixturePath string,
	disableFallback bool,
) ([]byte, error) {
	if useOffline {
		raw, err := os.ReadFile(fixturePath)
		if err != nil {
			return nil, fmt.Errorf("read offline tick fixture %s: %w", fixturePath, err)
		}
		return raw, nil
	}

	targetURL := fmt.Sprintf(TSETMCTradeURLToday, insCode)
	if strings.TrimSpace(explicitDate) != "" {
		targetURL = fmt.Sprintf(TSETMCTradeURLHistory, insCode, strings.TrimSpace(explicitDate))
	}

	if s.client != nil {
		body, status, err := s.client.Get(ctx, targetURL)
		if err == nil && status == http.StatusOK {
			return body, nil
		}
		if s.logger != nil {
			s.logger.Warn("live tick fetch failed",
				domainlogger.String("ins_code", insCode),
				domainlogger.Int("status_code", status),
				domainlogger.Error("error", err),
			)
		}
	}

	if !disableFallback {
		if raw, err := os.ReadFile(fixturePath); err == nil {
			if s.logger != nil {
				s.logger.Info("falling back to cached tick fixture",
					domainlogger.String("fixture_path", fixturePath),
				)
			}
			return raw, nil
		}
	}

	return nil, fmt.Errorf("failed fetching ticks for insCode %s and no offline fixture found at %s", insCode, fixturePath)
}
