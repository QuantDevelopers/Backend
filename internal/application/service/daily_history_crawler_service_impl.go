package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	crawlerdto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	domainlogger "github.com/QuantDevelopers/Backend/internal/domain/logger"
	domainrepo "github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	domaintsetmc "github.com/QuantDevelopers/Backend/internal/domain/tsetmc"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

const (
	// TSETMCDailyHistoryURL fetches daily OHLCV bars for an instrument (`top=0` returns full history).
	TSETMCDailyHistoryURL = "https://cdn.tsetmc.com/api/ClosingPrice/GetClosingPriceDailyList/%s/%d"

	defaultUnderlyingDailyFixture = "data/raw/samples/sample_underlying_daily.json"
	defaultOptionDailyFixture     = "data/raw/samples/sample_option_daily.json"
	defaultDailyRawDir            = "data/raw/daily"
)

type dailyTarget struct {
	InstrumentType string
	InstrumentID   uint
	SymbolFA       string
	InsCode        string
}

// DailyHistoryCrawlerService implements E2-1 daily history crawling and Postgres persistence.
type DailyHistoryCrawlerService struct {
	client         domaintsetmc.Client
	underlyingRepo domainrepo.UnderlyingRepository
	optionRepo     domainrepo.OptionContractRepository
	dailyRepo      domainrepo.DailyHistoryRepository
	db             database.Database
	logger         domainlogger.Logger
}

// NewDailyHistoryCrawlerService constructs a DailyHistoryCrawlerService instance.
func NewDailyHistoryCrawlerService(
	client domaintsetmc.Client,
	underlyingRepo domainrepo.UnderlyingRepository,
	optionRepo domainrepo.OptionContractRepository,
	dailyRepo domainrepo.DailyHistoryRepository,
	db database.Database,
	log domainlogger.Logger,
) *DailyHistoryCrawlerService {
	return &DailyHistoryCrawlerService{
		client:         client,
		underlyingRepo: underlyingRepo,
		optionRepo:     optionRepo,
		dailyRepo:      dailyRepo,
		db:             db,
		logger:         log,
	}
}

func (s *DailyHistoryCrawlerService) ParseDailyPayload(
	raw []byte,
	instrumentType string,
	instrumentID uint,
) ([]*entity.DailyHistory, error) {
	var payload crawlerdto.ClosingPriceDailyPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("decode closingPriceDaily JSON: %w", err)
	}

	normType := strings.ToUpper(strings.TrimSpace(instrumentType))
	if normType != "UNDERLYING" && normType != "OPTION" {
		return nil, fmt.Errorf("unsupported instrument type %q", instrumentType)
	}

	byDate := make(map[string]*entity.DailyHistory, len(payload.ClosingPriceDaily))
	for _, item := range payload.ClosingPriceDaily {
		if item.DEven <= 0 {
			continue
		}
		tradeDate, err := parseTSETMCDate(fmt.Sprintf("%08d", item.DEven))
		if err != nil {
			continue
		}

		closePrice := item.PClosing
		if closePrice == 0 && item.PDrCotVal > 0 {
			closePrice = item.PDrCotVal
		}
		openPrice := item.PriceFirst
		if openPrice == 0 {
			if item.PDrCotVal > 0 {
				openPrice = item.PDrCotVal
			} else {
				openPrice = closePrice
			}
		}
		highPrice := item.PriceMax
		if highPrice == 0 && closePrice > 0 {
			highPrice = closePrice
		}
		lowPrice := item.PriceMin
		if lowPrice == 0 && closePrice > 0 {
			lowPrice = closePrice
		}

		var yesterdayPtr *float64
		if item.PriceYesterday > 0 {
			yVal := item.PriceYesterday
			yesterdayPtr = &yVal
		}

		dateKey := tradeDate.Format("2006-01-02")
		byDate[dateKey] = &entity.DailyHistory{
			InstrumentType:  normType,
			InstrumentID:    instrumentID,
			TradeDate:       tradeDate,
			OpenPrice:       openPrice,
			HighPrice:       highPrice,
			LowPrice:        lowPrice,
			ClosePrice:      closePrice,
			YesterdayPrice:  yesterdayPtr,
			SettlementPrice: item.SettlementPrice,
			Volume:          int64(math.Round(item.QTotTran5J)),
			Value:           item.QTotCap,
			TradeCount:      int(math.Round(item.ZTotTran)),
			OpenInterest:    item.OpenInterest,
		}
	}

	out := make([]*entity.DailyHistory, 0, len(byDate))
	for _, rec := range byDate {
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].TradeDate.After(out[j].TradeDate)
	})
	return out, nil
}

func (s *DailyHistoryCrawlerService) Crawl(
	ctx context.Context,
	opts crawlerdto.DailyHistoryCrawlOptions,
) (*crawlerdto.DailyHistoryCrawlResult, error) {
	if !opts.IncludeUnderlyings && !opts.IncludeOptions {
		opts.IncludeUnderlyings = true
		opts.IncludeOptions = true
	}
	if opts.RawDir == "" {
		opts.RawDir = defaultDailyRawDir
	}
	if opts.UnderlyingFixturePath == "" {
		opts.UnderlyingFixturePath = defaultUnderlyingDailyFixture
	}
	if opts.OptionFixturePath == "" {
		opts.OptionFixturePath = defaultOptionDailyFixture
	}

	targets, err := s.resolveTargets(opts)
	if err != nil {
		return nil, err
	}

	if opts.SaveRaw {
		if err := os.MkdirAll(opts.RawDir, 0o755); err != nil {
			return nil, fmt.Errorf("create daily raw dir %s: %w", opts.RawDir, err)
		}
	}

	result := &crawlerdto.DailyHistoryCrawlResult{
		InstrumentsTotal: len(targets),
		Items:            make([]crawlerdto.DailyHistoryInstrumentSummary, 0, len(targets)),
	}

	for _, target := range targets {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		itemSummary, circuitBroken := s.crawlOneTarget(ctx, target, opts)
		result.Items = append(result.Items, itemSummary)
		if itemSummary.Error != "" {
			result.InstrumentsFailed++
		} else {
			result.InstrumentsSucceeded++
			result.RowsParsed += itemSummary.RowsParsed
			result.RowsSaved += itemSummary.RowsSaved
		}

		if circuitBroken {
			result.CircuitBroken = true
			if s.logger != nil {
				s.logger.Warn("stopping E2-1 daily crawl: TSETMC circuit breaker open")
			}
			break
		}
	}

	return result, nil
}

func (s *DailyHistoryCrawlerService) resolveTargets(opts crawlerdto.DailyHistoryCrawlOptions) ([]dailyTarget, error) {
	insFilter := strings.TrimSpace(opts.InsCode)

	if s.db == nil {
		var synthetic []dailyTarget
		if opts.IncludeUnderlyings {
			synthetic = append(synthetic, dailyTarget{
				InstrumentType: "UNDERLYING",
				InstrumentID:   1,
				SymbolFA:       "اهرم",
				InsCode:        "17914401175772326",
			})
		}
		if opts.IncludeOptions {
			synthetic = append(synthetic, dailyTarget{
				InstrumentType: "OPTION",
				InstrumentID:   1,
				SymbolFA:       "ضهرم1033",
				InsCode:        "19693832241143005",
			})
		}
		return filterTargets(synthetic, insFilter, opts.Limit), nil
	}

	var targets []dailyTarget
	if opts.IncludeUnderlyings && s.underlyingRepo != nil {
		assets, err := s.underlyingRepo.GetAll(s.db)
		if err != nil {
			return nil, fmt.Errorf("list underlyings for daily crawl: %w", err)
		}
		for _, a := range assets {
			if a.TSETMCID == nil || strings.TrimSpace(*a.TSETMCID) == "" || !a.IsActive {
				continue
			}
			targets = append(targets, dailyTarget{
				InstrumentType: "UNDERLYING",
				InstrumentID:   a.ID,
				SymbolFA:       a.SymbolFA,
				InsCode:        strings.TrimSpace(*a.TSETMCID),
			})
		}
	}

	if opts.IncludeOptions && s.optionRepo != nil {
		active := true
		contracts, err := s.optionRepo.GetAll(s.db, domainrepo.OptionFilter{IsActive: &active})
		if err != nil {
			return nil, fmt.Errorf("list option contracts for daily crawl: %w", err)
		}
		for _, c := range contracts {
			if c.TSETMCID == nil || strings.TrimSpace(*c.TSETMCID) == "" {
				continue
			}
			targets = append(targets, dailyTarget{
				InstrumentType: "OPTION",
				InstrumentID:   c.ID,
				SymbolFA:       c.SymbolFA,
				InsCode:        strings.TrimSpace(*c.TSETMCID),
			})
		}
	}

	return filterTargets(targets, insFilter, opts.Limit), nil
}

func (s *DailyHistoryCrawlerService) crawlOneTarget(
	ctx context.Context,
	target dailyTarget,
	opts crawlerdto.DailyHistoryCrawlOptions,
) (crawlerdto.DailyHistoryInstrumentSummary, bool) {
	summary := crawlerdto.DailyHistoryInstrumentSummary{
		InstrumentType: target.InstrumentType,
		InstrumentID:   target.InstrumentID,
		SymbolFA:       target.SymbolFA,
		InsCode:        target.InsCode,
	}

	raw, circuitBroken, err := s.fetchDailyPayload(ctx, target, opts)
	if err != nil {
		summary.Error = err.Error()
		return summary, circuitBroken
	}

	if opts.SaveRaw {
		rawPath := filepath.Join(opts.RawDir, fmt.Sprintf("%s.json", target.InsCode))
		if writeErr := os.WriteFile(rawPath, raw, 0o644); writeErr != nil {
			summary.Error = fmt.Sprintf("write raw daily payload %s: %v", rawPath, writeErr)
			return summary, false
		}
		summary.RawPath = rawPath
	}

	records, err := s.ParseDailyPayload(raw, target.InstrumentType, target.InstrumentID)
	if err != nil {
		summary.Error = err.Error()
		return summary, false
	}
	summary.RowsParsed = len(records)

	if s.db != nil && s.dailyRepo != nil && len(records) > 0 {
		saved, saveErr := s.dailyRepo.SaveBulk(s.db, records)
		if saveErr != nil {
			summary.Error = fmt.Sprintf("persist daily history: %v", saveErr)
			return summary, false
		}
		summary.RowsSaved = saved
	}

	return summary, false
}

func (s *DailyHistoryCrawlerService) fetchDailyPayload(
	ctx context.Context,
	target dailyTarget,
	opts crawlerdto.DailyHistoryCrawlOptions,
) ([]byte, bool, error) {
	fixturePath := opts.UnderlyingFixturePath
	if target.InstrumentType == "OPTION" {
		fixturePath = opts.OptionFixturePath
	}

	if opts.UseOfflineFixture {
		raw, err := os.ReadFile(fixturePath)
		if err != nil {
			return nil, false, fmt.Errorf("read offline daily fixture %s: %w", fixturePath, err)
		}
		return raw, false, nil
	}

	targetURL := fmt.Sprintf(TSETMCDailyHistoryURL, target.InsCode, opts.TopDays)
	if s.client != nil {
		body, status, err := s.client.Get(ctx, targetURL)
		if err == nil && status == http.StatusOK {
			return body, false, nil
		}
		if errors.Is(err, domaintsetmc.ErrCircuitBreakerOpen) {
			return nil, true, err
		}
		if s.logger != nil {
			s.logger.Warn("live daily history request failed",
				domainlogger.String("ins_code", target.InsCode),
				domainlogger.Int("status_code", status),
				domainlogger.Error("error", err),
			)
		}
	}

	if !opts.DisableFallback {
		if raw, err := os.ReadFile(fixturePath); err == nil {
			return raw, false, nil
		}
	}

	return nil, false, fmt.Errorf("failed fetching daily history for %s", target.InsCode)
}

func filterTargets(items []dailyTarget, insCodeFilter string, limit int) []dailyTarget {
	out := make([]dailyTarget, 0, len(items))
	for _, it := range items {
		if insCodeFilter != "" && it.InsCode != insCodeFilter {
			continue
		}
		out = append(out, it)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}
