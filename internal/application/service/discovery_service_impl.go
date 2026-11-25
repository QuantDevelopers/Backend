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
	if err := requireArray(raw, "instrumentOptMarketWatch"); err != nil {
		return nil, nil, err
	}
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
	if opts.PersistDB && (s.db == nil || s.underlyingRepo == nil || s.optionRepo == nil) {
		return nil, fmt.Errorf("database repositories required for registry persistence")
	}
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

// DiscoverFullRegistry implements E1-5 full instrument registry discovery.
// It discovers all underlying stocks/ETFs, active options, detects expired contracts,
// builds unified instrument records, writes complete artifacts to data/registry,
// and optionally persists both active and expired entities into PostgreSQL.
func (s *DiscoveryService) DiscoverFullRegistry(
	ctx context.Context,
	opts crawlerdto.FullRegistryOptions,
) (*crawlerdto.FullRegistryResult, error) {
	if opts.PersistDB && (s.db == nil || s.underlyingRepo == nil || s.optionRepo == nil) {
		return nil, fmt.Errorf("database repositories required for registry persistence")
	}
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

	underlyings, allContracts, err := s.ParseMarketWatch(rawPayload)
	if err != nil {
		return nil, err
	}

	underlyings, allContracts, history, err := s.discoverHistory(ctx, opts, underlyings, allContracts)
	if err != nil {
		return nil, err
	}
	todayStr := marketToday()
	activeContracts := make([]crawlerdto.DiscoveredContract, 0, len(allContracts))
	expiredContracts := make([]crawlerdto.DiscoveredContract, 0, len(allContracts)/4)
	seenContracts := make(map[string]bool, len(allContracts))

	for _, c := range allContracts {
		if _, err := parseTSETMCDate(c.ExpiryDate); err != nil {
			return nil, err
		}
		if seenContracts[c.InsCode] {
			continue
		}
		seenContracts[c.InsCode] = true
		isExpired := c.ExpiryDate != "" && c.ExpiryDate < todayStr
		if isExpired {
			if opts.IncludeExpired {
				expiredContracts = append(expiredContracts, c)
			}
		} else {
			activeContracts = append(activeContracts, c)
		}
	}

	// Also inspect database for any known contracts not in active watch or already marked expired
	if opts.IncludeExpired && s.db != nil && s.optionRepo != nil {
		dbContracts, err := s.optionRepo.GetAll(s.db, domainrepo.OptionFilter{})
		if err != nil {
			return nil, fmt.Errorf("read historical registry: %w", err)
		}
		{
			for _, dbc := range dbContracts {
				if dbc.TSETMCID == nil {
					continue
				}
				ins := *dbc.TSETMCID
				if seenContracts[ins] {
					continue
				}
				seenContracts[ins] = true
				expStr := dbc.ExpirationDate.Format("20060102")
				uaSym := ""
				uaIns := ""
				if dbc.UnderlyingAsset != nil {
					uaSym = dbc.UnderlyingAsset.SymbolFA
					if dbc.UnderlyingAsset.TSETMCID != nil {
						found := false
						for _, u := range underlyings {
							if u.InsCode == *dbc.UnderlyingAsset.TSETMCID {
								found = true
								break
							}
						}
						if !found {
							underlyings = append(underlyings, crawlerdto.DiscoveredUnderlying{InsCode: *dbc.UnderlyingAsset.TSETMCID, Symbol: uaSym, Name: dbc.UnderlyingAsset.NameFA, Market: dbc.UnderlyingAsset.MarketType, DiscoveredBy: "database_historical_registry"})
						}
					}
					if dbc.UnderlyingAsset.TSETMCID != nil {
						uaIns = *dbc.UnderlyingAsset.TSETMCID
					}
				}
				begStr := ""
				if dbc.BeginDate != nil {
					begStr = dbc.BeginDate.Format("20060102")
				}
				side := "Call"
				if strings.EqualFold(dbc.ContractType, "put") {
					side = "Put"
				}
				historical := crawlerdto.DiscoveredContract{
					InsCode:          ins,
					Symbol:           dbc.SymbolFA,
					Name:             dbc.NameFA,
					Side:             side,
					UaInsCode:        uaIns,
					UnderlyingSymbol: uaSym,
					StrikePrice:      dbc.StrikePrice,
					ContractSize:     dbc.ContractSize,
					BeginDate:        begStr,
					ExpiryDate:       expStr,
					RemainedDays:     0,
					DiscoveredBy:     "database_historical_registry",
				}
				if expStr < todayStr {
					expiredContracts = append(expiredContracts, historical)
				} else {
					activeContracts = append(activeContracts, historical)
				}
			}
		}
	}

	// Build unified RegistryInstrument array
	discoveredAtStr := time.Now().UTC().Format(time.RFC3339)
	instruments := make([]crawlerdto.RegistryInstrument, 0, len(underlyings)+len(activeContracts)+len(expiredContracts))
	underlyingsSummary := make(map[string]int, len(underlyings))
	expiriesSet := make(map[string]struct{})
	callCount := 0
	putCount := 0

	for _, u := range underlyings {
		assetClass := "STOCK"
		if u.Market == "صندوق" || strings.HasPrefix(u.ISIN, "IRT") || inferMarketLabel(u.Symbol) == "صندوق" {
			assetClass = "ETF"
		}
		instruments = append(instruments, crawlerdto.RegistryInstrument{
			InsCode: u.InsCode,
			ISIN:    u.ISIN, InstrumentID: u.InstrumentID,
			Symbol:       u.Symbol,
			Name:         u.Name,
			AssetClass:   assetClass,
			Market:       mapMarketTypeForDB(u.Market),
			Status:       "ACTIVE",
			IsActive:     true,
			LastPrice:    u.LastPrice,
			ClosePrice:   u.ClosePrice,
			DiscoveredBy: u.DiscoveredBy,
			DiscoveredAt: discoveredAtStr,
		})
	}

	appendContractToRegistry := func(c crawlerdto.DiscoveredContract, status string, isActive bool) {
		assetClass := "OPTION_CALL"
		if strings.EqualFold(c.Side, "put") {
			assetClass = "OPTION_PUT"
			putCount++
		} else {
			callCount++
		}
		if c.UnderlyingSymbol != "" {
			underlyingsSummary[c.UnderlyingSymbol]++
		}
		if c.ExpiryDate != "" {
			expiriesSet[c.ExpiryDate] = struct{}{}
		}
		instruments = append(instruments, crawlerdto.RegistryInstrument{
			InsCode: c.InsCode,
			ISIN:    c.ISIN, InstrumentID: c.InstrumentID, ContractSizeEstimated: c.ContractSizeEstimated,
			Symbol:           c.Symbol,
			Name:             c.Name,
			AssetClass:       assetClass,
			Market:           "Bourse",
			UnderlyingSymbol: c.UnderlyingSymbol,
			UaInsCode:        c.UaInsCode,
			StrikePrice:      c.StrikePrice,
			ContractSize:     c.ContractSize,
			BeginDate:        c.BeginDate,
			ExpiryDate:       c.ExpiryDate,
			RemainedDays:     c.RemainedDays,
			Status:           status,
			IsActive:         isActive,
			LastPrice:        c.LastPrice,
			ClosePrice:       c.ClosePrice,
			OpenPositions:    c.OpenPositions,
			TradeCount:       c.TradeCount,
			TradeVolume:      c.TradeVolume,
			TradeValue:       c.TradeValue,
			DiscoveredBy:     c.DiscoveredBy,
			DiscoveredAt:     discoveredAtStr,
		})
	}

	for _, c := range activeContracts {
		appendContractToRegistry(c, "ACTIVE", true)
	}
	for _, c := range expiredContracts {
		appendContractToRegistry(c, "EXPIRED", false)
	}

	sort.Slice(instruments, func(i, j int) bool {
		if instruments[i].Symbol == instruments[j].Symbol {
			return instruments[i].InsCode < instruments[j].InsCode
		}
		return instruments[i].Symbol < instruments[j].Symbol
	})

	summary := crawlerdto.FullRegistrySummary{
		HistoryDatesCompleted: len(history.Dates), HistoricalOptionsAdded: history.Added, EstimatedContractSizes: history.EstimatedSizes,
		GeneratedAt:         time.Now().UTC(),
		TotalInstruments:    len(instruments),
		UnderlyingsCount:    len(underlyings),
		ActiveOptionsCount:  len(activeContracts),
		ExpiredOptionsCount: len(expiredContracts),
		CallCount:           callCount,
		PutCount:            putCount,
		ExpiriesCount:       len(expiriesSet),
		UnderlyingsSummary:  underlyingsSummary,
	}

	if len(history.Dates) > 0 {
		summary.HistoryStartDate = history.Dates[len(history.Dates)-1]
		summary.HistoryEndDate = history.Dates[0]
	}
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		return nil, fmt.Errorf("create registry dir %s: %w", registryDir, err)
	}

	underlyingsPath := filepath.Join(registryDir, "underlyings.json")
	if err := writeJSONPretty(underlyingsPath, underlyings); err != nil {
		return nil, err
	}

	activePath := filepath.Join(registryDir, "active_contracts.json")
	if err := writeJSONPretty(activePath, activeContracts); err != nil {
		return nil, err
	}

	expiredPath := filepath.Join(registryDir, "expired_contracts.json")
	if err := writeJSONPretty(expiredPath, expiredContracts); err != nil {
		return nil, err
	}

	instrumentsPath := filepath.Join(registryDir, "instruments.json")
	if err := writeJSONPretty(instrumentsPath, instruments); err != nil {
		return nil, err
	}

	summaryPath := filepath.Join(registryDir, "registry_summary.json")
	if err := writeJSONPretty(summaryPath, summary); err != nil {
		return nil, err
	}

	persisted := 0
	if opts.PersistDB && s.db != nil && s.underlyingRepo != nil && s.optionRepo != nil {
		pActive, err := s.persistToPostgres(underlyings, activeContracts)
		if err != nil {
			return nil, fmt.Errorf("persist active instruments: %w", err)
		}
		persisted += pActive

		if len(expiredContracts) > 0 {
			pExpired, err := s.persistExpiredContractsToPostgres(underlyings, expiredContracts)
			if err != nil {
				return nil, fmt.Errorf("persist expired instruments: %w", err)
			}
			persisted += pExpired
		}
	}

	return &crawlerdto.FullRegistryResult{
		Summary:          summary,
		Instruments:      instruments,
		Underlyings:      underlyings,
		ActiveContracts:  activeContracts,
		ExpiredContracts: expiredContracts,
		InstrumentsPath:  instrumentsPath,
		UnderlyingsPath:  underlyingsPath,
		ActivePath:       activePath,
		ExpiredPath:      expiredPath,
		SummaryPath:      summaryPath,
		PersistedDBRows:  persisted,
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

	return nil, fmt.Errorf("live discovery failed; offline fixtures require explicit offline mode (%s)", fixturePath)
}

func (s *DiscoveryService) persistToPostgres(underlyings []crawlerdto.DiscoveredUnderlying, contracts []crawlerdto.DiscoveredContract) (int, error) {
	uaIDByInsCode := make(map[string]uint, len(underlyings))

	for _, u := range underlyings {
		insCode := u.InsCode
		existing, err := s.underlyingRepo.GetBySymbol(s.db, u.Symbol)
		if err != nil {
			return 0, fmt.Errorf("lookup underlying %s: %w", u.Symbol, err)
		}
		if existing != nil && existing.TSETMCID != nil && *existing.TSETMCID != insCode {
			existing = nil
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
			ISIN:       optionalString(u.ISIN),
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
			return 0, fmt.Errorf("missing underlying for contract %s", c.Symbol)
		}
		expiry, err := parseTSETMCDate(c.ExpiryDate)
		if err != nil {
			return 0, err
		}
		var beginPtr *time.Time
		if b, err := parseTSETMCDate(c.BeginDate); err == nil {
			beginPtr = &b
		}
		insCode := c.InsCode
		entities = append(entities, &entity.OptionContract{
			UnderlyingAssetID: uaID,
			ISIN:              optionalString(c.ISIN),
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

func (s *DiscoveryService) persistExpiredContractsToPostgres(underlyings []crawlerdto.DiscoveredUnderlying, contracts []crawlerdto.DiscoveredContract) (int, error) {
	uaIDByInsCode := make(map[string]uint, len(underlyings))
	for _, u := range underlyings {
		insCode := u.InsCode
		existing, err := s.underlyingRepo.GetByTSETMCID(s.db, u.InsCode)
		if err == nil && existing != nil {
			uaIDByInsCode[insCode] = existing.ID
		}
	}

	entities := make([]*entity.OptionContract, 0, len(contracts))
	for _, c := range contracts {
		uaID, ok := uaIDByInsCode[c.UaInsCode]
		if !ok || uaID == 0 {
			if c.UnderlyingSymbol != "" {
				if existing, err := s.underlyingRepo.GetBySymbol(s.db, c.UnderlyingSymbol); err == nil && existing != nil {
					uaID = existing.ID
				}
			}
		}
		if uaID == 0 {
			return 0, fmt.Errorf("missing underlying for expired contract %s", c.Symbol)
		}
		expiry, err := parseTSETMCDate(c.ExpiryDate)
		if err != nil {
			return 0, err
		}
		var beginPtr *time.Time
		if b, err := parseTSETMCDate(c.BeginDate); err == nil {
			beginPtr = &b
		}
		insCode := c.InsCode
		entities = append(entities, &entity.OptionContract{
			UnderlyingAssetID: uaID,
			ISIN:              optionalString(c.ISIN),
			SymbolFA:          c.Symbol,
			NameFA:            c.Name,
			ContractType:      strings.ToUpper(c.Side),
			StrikePrice:       c.StrikePrice,
			ContractSize:      c.ContractSize,
			ExpirationDate:    expiry,
			BeginDate:         beginPtr,
			TSETMCID:          &insCode,
			IsActive:          false,
			Status:            "EXPIRED",
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
	if err := writeAtomic(path, encoded); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
