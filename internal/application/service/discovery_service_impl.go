package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	crawlerdto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	domainlogger "github.com/QuantDevelopers/Backend/internal/domain/logger"
	domainrepo "github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	domaintext "github.com/QuantDevelopers/Backend/internal/domain/text"
	domaintsetmc "github.com/QuantDevelopers/Backend/internal/domain/tsetmc"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

const (
	// TSETMCOptionWatchURL is the official TSETMC active option watch endpoint.
	TSETMCOptionWatchURL = "https://cdn.tsetmc.com/api/Instrument/GetInstrumentOptionMarketWatch/0"

	defaultDiscoveryFixture  = "tests/fixtures/sample_option_watch.json"
	defaultRegistryDir       = "data/registry"
	discoveryProvenanceLabel = "tsetmc_option_watch_api"
)

// DiscoveryService implements E1-3 active contract discovery in Go.
type DiscoveryService struct {
	client         domaintsetmc.Client
	underlyingRepo domainrepo.UnderlyingRepository
	optionRepo     domainrepo.OptionContractRepository
	db             database.Database
	logger         domainlogger.Logger
}

// NewDiscoveryService constructs a DiscoveryService instance.
func NewDiscoveryService(
	client domaintsetmc.Client,
	underlyingRepo domainrepo.UnderlyingRepository,
	optionRepo domainrepo.OptionContractRepository,
	db database.Database,
	log domainlogger.Logger,
) *DiscoveryService {
	return &DiscoveryService{
		client:         client,
		underlyingRepo: underlyingRepo,
		optionRepo:     optionRepo,
		db:             db,
		logger:         log,
	}
}

func (s *DiscoveryService) ParseMarketWatch(raw []byte) ([]crawlerdto.DiscoveredUnderlying, []crawlerdto.DiscoveredContract, error) {
	var payload crawlerdto.MarketWatchPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, nil, fmt.Errorf("decode market watch JSON: %w", err)
	}

	underlyingsMap := make(map[string]*crawlerdto.DiscoveredUnderlying)
	contracts := make([]crawlerdto.DiscoveredContract, 0, len(payload.InstrumentOptMarketWatch)*2)

	for _, row := range payload.InstrumentOptMarketWatch {
		uaInsCode := strings.TrimSpace(row.UaInsCode)
		if uaInsCode == "" {
			continue
		}

		uaSymbol := domaintext.NormalizePersian(row.LVal30UA)
		contractSize := row.ContractSize
		if contractSize <= 0 {
			contractSize = 1000
		}
		beginDate := strings.TrimSpace(row.BeginDate)
		endDate := strings.TrimSpace(row.EndDate)

		ua, exists := underlyingsMap[uaInsCode]
		if !exists {
			lastPrice := row.PDrCotValUA
			if lastPrice == 0 {
				lastPrice = row.PClosingUA
			}
			var priceChange float64
			if row.PriceYesterdayUA > 0 && lastPrice > 0 {
				priceChange = math.Round(((lastPrice-row.PriceYesterdayUA)/row.PriceYesterdayUA)*10000) / 100
			}
			ua = &crawlerdto.DiscoveredUnderlying{
				InsCode:        uaInsCode,
				Symbol:         uaSymbol,
				Name:           uaSymbol,
				Market:         inferMarketLabel(uaSymbol),
				LastPrice:      lastPrice,
				ClosePrice:     row.PClosingUA,
				YesterdayPrice: row.PriceYesterdayUA,
				DiscoveredBy:   discoveryProvenanceLabel,
				PriceChange:    priceChange,
			}
			underlyingsMap[uaInsCode] = ua
		}

		if callCode := strings.TrimSpace(row.InsCodeC); callCode != "" {
			if callSym := domaintext.NormalizePersian(row.LVal18AFCC); callSym != "" {
				contracts = append(contracts, crawlerdto.DiscoveredContract{
					InsCode:          callCode,
					Symbol:           callSym,
					Name:             domaintext.NormalizePersian(row.LVal30C),
					Side:             "Call",
					UaInsCode:        uaInsCode,
					UnderlyingSymbol: uaSymbol,
					StrikePrice:      row.StrikePrice,
					ContractSize:     contractSize,
					BeginDate:        beginDate,
					ExpiryDate:       endDate,
					RemainedDays:     row.RemainedDay,
					LastPrice:        row.PDrCotValC,
					ClosePrice:       row.PClosingC,
					OpenPositions:    row.OPC,
					TradeCount:       row.ZTotTranC,
					TradeVolume:      row.QTotTran5JC,
					TradeValue:       row.QTotCapC,
					DiscoveredBy:     discoveryProvenanceLabel,
				})
				ua.CallCount++
				ua.ActiveOptionsCount++
			}
		}

		if putCode := strings.TrimSpace(row.InsCodeP); putCode != "" {
			if putSym := domaintext.NormalizePersian(row.LVal18AFCP); putSym != "" {
				contracts = append(contracts, crawlerdto.DiscoveredContract{
					InsCode:          putCode,
					Symbol:           putSym,
					Name:             domaintext.NormalizePersian(row.LVal30P),
					Side:             "Put",
					UaInsCode:        uaInsCode,
					UnderlyingSymbol: uaSymbol,
					StrikePrice:      row.StrikePrice,
					ContractSize:     contractSize,
					BeginDate:        beginDate,
					ExpiryDate:       endDate,
					RemainedDays:     row.RemainedDay,
					LastPrice:        row.PDrCotValP,
					ClosePrice:       row.PClosingP,
					OpenPositions:    row.OPP,
					TradeCount:       row.ZTotTranP,
					TradeVolume:      row.QTotTran5JP,
					TradeValue:       row.QTotCapP,
					DiscoveredBy:     discoveryProvenanceLabel,
				})
				ua.PutCount++
				ua.ActiveOptionsCount++
			}
		}
	}

	underlyings := make([]crawlerdto.DiscoveredUnderlying, 0, len(underlyingsMap))
	for _, ua := range underlyingsMap {
		underlyings = append(underlyings, *ua)
	}
	sort.Slice(underlyings, func(i, j int) bool {
		if underlyings[i].Symbol == underlyings[j].Symbol {
			return underlyings[i].InsCode < underlyings[j].InsCode
		}
		return underlyings[i].Symbol < underlyings[j].Symbol
	})

	return underlyings, contracts, nil
}

func (s *DiscoveryService) Discover(ctx context.Context, opts crawlerdto.DiscoveryOptions) (*crawlerdto.DiscoveryResult, error) {
	fixturePath := opts.FixturePath
	if fixturePath == "" {
		fixturePath = defaultDiscoveryFixture
	}
	registryDir := opts.RegistryDir
	if registryDir == "" {
		registryDir = defaultRegistryDir
	}

	rawPayload, err := s.fetchPayload(ctx, opts.UseOfflineFixture, fixturePath, opts.DisableFallback)
	if err != nil {
		return nil, err
	}

	underlyings, contracts, err := s.ParseMarketWatch(rawPayload)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		return nil, fmt.Errorf("create registry dir: %w", err)
	}

	underlyingsPath := filepath.Join(registryDir, "underlyings.json")
	if err := writeJSONPretty(underlyingsPath, underlyings); err != nil {
		return nil, err
	}

	contractsPath := filepath.Join(registryDir, "active_contracts.json")
	if err := writeJSONPretty(contractsPath, contracts); err != nil {
		return nil, err
	}

	persisted := 0
	if opts.PersistDB && s.db != nil && s.underlyingRepo != nil && s.optionRepo != nil {
		persisted, err = s.persistToPostgres(underlyings, contracts)
		if err != nil {
			return nil, err
		}
	}

	return &crawlerdto.DiscoveryResult{
		Underlyings:     underlyings,
		Contracts:       contracts,
		UnderlyingsPath: underlyingsPath,
		ContractsPath:   contractsPath,
		PersistedDBRows: persisted,
	}, nil
}

func (s *DiscoveryService) fetchPayload(ctx context.Context, useOffline bool, fixturePath string, disableFallback bool) ([]byte, error) {
	if useOffline {
		raw, err := os.ReadFile(fixturePath)
		if err != nil {
			return nil, fmt.Errorf("read offline market watch fixture %s: %w", fixturePath, err)
		}
		return raw, nil
	}

	if s.client != nil {
		body, status, err := s.client.Get(ctx, TSETMCOptionWatchURL)
		if err == nil && status == http.StatusOK {
			return body, nil
		}
		if s.logger != nil {
			s.logger.Warn("live option watch request failed",
				domainlogger.Int("status_code", status),
				domainlogger.Error("error", err),
			)
		}
	}

	if !disableFallback {
		if raw, err := os.ReadFile(fixturePath); err == nil {
			if s.logger != nil {
				s.logger.Info("falling back to cached option watch fixture",
					domainlogger.String("fixture_path", fixturePath),
				)
			}
			return raw, nil
		}
	}

	return nil, fmt.Errorf("failed to fetch TSETMC option watch and no offline fixture found at %s", fixturePath)
}

func (s *DiscoveryService) persistToPostgres(underlyings []crawlerdto.DiscoveredUnderlying, contracts []crawlerdto.DiscoveredContract) (int, error) {
	uaIDByInsCode := make(map[string]uint, len(underlyings))

	for _, u := range underlyings {
		insCode := u.InsCode
		existing, err := s.underlyingRepo.GetBySymbol(s.db, u.Symbol)
		if err != nil {
			return 0, fmt.Errorf("lookup underlying %s: %w", u.Symbol, err)
		}
		if existing != nil {
			if existing.TSETMCID == nil {
				existing.TSETMCID = &insCode
				if err := s.underlyingRepo.Upsert(s.db, existing); err != nil {
					return 0, fmt.Errorf("update underlying %s: %w", u.Symbol, err)
				}
			}
			uaIDByInsCode[insCode] = existing.ID
			continue
		}

		asset := entity.UnderlyingAsset{
			SymbolFA:   u.Symbol,
			NameFA:     u.Name,
			TSETMCID:   &insCode,
			MarketType: mapMarketTypeForDB(u.Market),
			IsActive:   true,
		}
		if err := s.underlyingRepo.Upsert(s.db, &asset); err != nil {
			return 0, fmt.Errorf("upsert underlying %s: %w", u.Symbol, err)
		}
		if asset.ID == 0 {
			fetched, err := s.underlyingRepo.GetBySymbol(s.db, u.Symbol)
			if err != nil || fetched == nil {
				return 0, fmt.Errorf("resolve ID after upsert for %s: %w", u.Symbol, err)
			}
			asset.ID = fetched.ID
		}
		uaIDByInsCode[insCode] = asset.ID
	}

	entities := make([]*entity.OptionContract, 0, len(contracts))
	for _, c := range contracts {
		uaID, ok := uaIDByInsCode[c.UaInsCode]
		if !ok || uaID == 0 {
			continue
		}
		expiry, err := parseTSETMCDate(c.ExpiryDate)
		if err != nil {
			continue
		}
		var beginPtr *time.Time
		if b, err := parseTSETMCDate(c.BeginDate); err == nil {
			beginPtr = &b
		}
		insCode := c.InsCode
		entities = append(entities, &entity.OptionContract{
			UnderlyingAssetID: uaID,
			SymbolFA:          c.Symbol,
			NameFA:            c.Name,
			ContractType:      strings.ToUpper(c.Side),
			StrikePrice:       c.StrikePrice,
			ContractSize:      c.ContractSize,
			ExpirationDate:    expiry,
			BeginDate:         beginPtr,
			TSETMCID:          &insCode,
			IsActive:          true,
			Status:            "ACTIVE",
		})
	}

	if len(entities) == 0 {
		return 0, nil
	}
	return s.optionRepo.SaveBulk(s.db, entities)
}

func inferMarketLabel(symbol string) string {
	if strings.Contains(symbol, "اهرم") || strings.Contains(symbol, "توان") || strings.Contains(symbol, "جهش") {
		return "صندوق"
	}
	return "بورس"
}

func mapMarketTypeForDB(market string) string {
	switch market {
	case "صندوق":
		return "ETF"
	case "فرابورس":
		return "Farabourse"
	default:
		return "Bourse"
	}
}

func parseTSETMCDate(yyyymmdd string) (time.Time, error) {
	trimmed := strings.TrimSpace(yyyymmdd)
	if len(trimmed) != 8 {
		return time.Time{}, fmt.Errorf("invalid YYYYMMDD date %q", yyyymmdd)
	}
	return time.Parse("20060102", trimmed)
}

func writeJSONPretty(path string, v interface{}) error {
	encoded, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	if err := os.WriteFile(path, encoded, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
