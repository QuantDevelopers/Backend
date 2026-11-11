package service

import (
	"strings"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	commondto "github.com/QuantDevelopers/Backend/internal/application/dto/common"
	marketdatadto "github.com/QuantDevelopers/Backend/internal/application/dto/marketdata"
	"github.com/QuantDevelopers/Backend/internal/domain/exception"
	domainrepo "github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	domaintext "github.com/QuantDevelopers/Backend/internal/domain/text"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

// MarketDataService serves search, daily OHLCV, and intraday tick queries across REST API v1.
type MarketDataService struct {
	constants      *bootstrap.Constants
	underlyingRepo domainrepo.UnderlyingRepository
	optionRepo     domainrepo.OptionContractRepository
	dailyRepo      domainrepo.DailyHistoryRepository
	tickRawRepo    domainrepo.TickRawRepository
	db             database.Database
}

// NewMarketDataService constructs a MarketDataService instance.
func NewMarketDataService(
	constants *bootstrap.Constants,
	underlyingRepo domainrepo.UnderlyingRepository,
	optionRepo domainrepo.OptionContractRepository,
	dailyRepo domainrepo.DailyHistoryRepository,
	tickRawRepo domainrepo.TickRawRepository,
	db database.Database,
) *MarketDataService {
	return &MarketDataService{
		constants:      constants,
		underlyingRepo: underlyingRepo,
		optionRepo:     optionRepo,
		dailyRepo:      dailyRepo,
		tickRawRepo:    tickRawRepo,
		db:             db,
	}
}

func (s *MarketDataService) SearchInstruments(
	req marketdatadto.SearchInstrumentsRequest,
) (marketdatadto.InstrumentSearchResponse, error) {
	query := domaintext.NormalizePersian(req.Query)
	if query == "" {
		var vErr exception.ValidationErrors
		vErr.Add("query", s.constants.Tag.Required)
		return marketdatadto.InstrumentSearchResponse{}, vErr
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	underlyings, err := s.underlyingRepo.Search(s.db, query, limit)
	if err != nil {
		return marketdatadto.InstrumentSearchResponse{}, err
	}

	items := make([]marketdatadto.InstrumentSearchItem, 0, limit)
	for _, u := range underlyings {
		items = append(items, marketdatadto.InstrumentSearchItem{
			InstrumentType: s.constants.InstrumentTypes.Underlying,
			ID:             u.ID,
			SymbolFA:       u.SymbolFA,
			NameFA:         u.NameFA,
			TSETMCID:       u.TSETMCID,
			ISIN:           u.ISIN,
			MarketType:     u.MarketType,
			IsActive:       u.IsActive,
		})
	}

	remaining := limit - len(items)
	if remaining > 0 {
		options, _, err := s.optionRepo.ListPaginated(s.db, domainrepo.OptionQueryFilter{
			Query: query,
			Limit: remaining,
		})
		if err != nil {
			return marketdatadto.InstrumentSearchResponse{}, err
		}
		for _, o := range options {
			strike := o.StrikePrice
			items = append(items, marketdatadto.InstrumentSearchItem{
				InstrumentType: s.constants.InstrumentTypes.Option,
				ID:             o.ID,
				SymbolFA:       o.SymbolFA,
				NameFA:         o.NameFA,
				TSETMCID:       o.TSETMCID,
				ISIN:           o.ISIN,
				ContractType:   o.ContractType,
				StrikePrice:    &strike,
				IsActive:       o.IsActive,
			})
		}
	}

	return marketdatadto.InstrumentSearchResponse{
		Query: query,
		Total: len(items),
		Items: items,
	}, nil
}

func (s *MarketDataService) GetDailyHistory(
	req marketdatadto.GetDailyHistoryRequest,
) (marketdatadto.DailyHistoryResponse, error) {
	instType, instID, symbolFA, tsetmcID, err := s.resolveInstrument(req.InstrumentType, req.SymbolFA)
	if err != nil {
		return marketdatadto.DailyHistoryResponse{}, err
	}

	var fromPtr, toPtr *time.Time
	if strings.TrimSpace(req.From) != "" {
		parsed, err := parseFlexibleDate(req.From)
		if err != nil {
			var vErr exception.ValidationErrors
			vErr.Add("date", s.constants.Tag.Invalid)
			return marketdatadto.DailyHistoryResponse{}, vErr
		}
		fromPtr = &parsed
	}
	if strings.TrimSpace(req.To) != "" {
		parsed, err := parseFlexibleDate(req.To)
		if err != nil {
			var vErr exception.ValidationErrors
			vErr.Add("date", s.constants.Tag.Invalid)
			return marketdatadto.DailyHistoryResponse{}, vErr
		}
		toPtr = &parsed
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	rows, err := s.dailyRepo.List(s.db, domainrepo.DailyHistoryFilter{
		InstrumentType: &instType,
		InstrumentID:   &instID,
		FromDate:       fromPtr,
		ToDate:         toPtr,
	}, limit)
	if err != nil {
		return marketdatadto.DailyHistoryResponse{}, err
	}

	items := make([]marketdatadto.DailyHistoryItem, len(rows))
	for i, r := range rows {
		items[i] = marketdatadto.DailyHistoryItem{
			ID:              r.ID,
			TradeDate:       r.TradeDate.Format("2006-01-02"),
			OpenPrice:       r.OpenPrice,
			HighPrice:       r.HighPrice,
			LowPrice:        r.LowPrice,
			ClosePrice:      r.ClosePrice,
			YesterdayPrice:  r.YesterdayPrice,
			SettlementPrice: r.SettlementPrice,
			Volume:          r.Volume,
			Value:           r.Value,
			TradeCount:      r.TradeCount,
			OpenInterest:    r.OpenInterest,
		}
	}

	return marketdatadto.DailyHistoryResponse{
		InstrumentType: instType,
		InstrumentID:   instID,
		SymbolFA:       symbolFA,
		TSETMCID:       tsetmcID,
		Total:          len(items),
		Items:          items,
	}, nil
}

func (s *MarketDataService) GetTicks(
	req marketdatadto.GetTicksRequest,
) (marketdatadto.TicksPaginatedResponse, error) {
	instType, instID, symbolFA, _, err := s.resolveInstrument(req.InstrumentType, req.SymbolFA)
	if err != nil {
		return marketdatadto.TicksPaginatedResponse{}, err
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 100
	}
	if pageSize > 500 {
		pageSize = 500
	}
	offset := (page - 1) * pageSize

	var targetDate *time.Time
	if strings.TrimSpace(req.Date) != "" {
		parsed, err := parseFlexibleDate(req.Date)
		if err != nil {
			var vErr exception.ValidationErrors
			vErr.Add("date", s.constants.Tag.Invalid)
			return marketdatadto.TicksPaginatedResponse{}, vErr
		}
		targetDate = &parsed
	} else {
		latest, err := s.tickRawRepo.GetLatestDate(s.db, instType, instID)
		if err != nil {
			return marketdatadto.TicksPaginatedResponse{}, err
		}
		targetDate = latest
	}

	if targetDate == nil {
		return marketdatadto.TicksPaginatedResponse{
			InstrumentType: instType,
			InstrumentID:   instID,
			SymbolFA:       symbolFA,
			Items:          []marketdatadto.TickItem{},
			Pagination:     commondto.BuildPaginationMeta(0, offset, pageSize),
		}, nil
	}

	rows, total, err := s.tickRawRepo.GetByInstrumentAndDate(s.db, instType, instID, *targetDate, offset, pageSize)
	if err != nil {
		return marketdatadto.TicksPaginatedResponse{}, err
	}

	items := make([]marketdatadto.TickItem, len(rows))
	for i, r := range rows {
		items[i] = marketdatadto.TickItem{
			ID:             r.ID,
			TradeDate:      r.TradeDate.Format("2006-01-02"),
			TradeTime:      r.TradeTime,
			SequenceNumber: r.SequenceNumber,
			Price:          r.Price,
			Volume:         r.Volume,
		}
	}

	return marketdatadto.TicksPaginatedResponse{
		InstrumentType: instType,
		InstrumentID:   instID,
		SymbolFA:       symbolFA,
		TradeDate:      targetDate.Format("2006-01-02"),
		Items:          items,
		Pagination:     commondto.BuildPaginationMeta(total, offset, pageSize),
	}, nil
}

func (s *MarketDataService) resolveInstrument(rawType, rawSymbol string) (string, uint, string, *string, error) {
	normType := strings.ToUpper(strings.TrimSpace(rawType))
	normSymbol := domaintext.NormalizePersian(rawSymbol)

	switch normType {
	case s.constants.InstrumentTypes.Underlying, "UNDERLYINGS":
		asset, err := s.underlyingRepo.GetBySymbol(s.db, normSymbol)
		if err != nil {
			return "", 0, "", nil, err
		}
		if asset == nil {
			return "", 0, "", nil, exception.NotFoundError{Item: s.constants.Field.UnderlyingAsset}
		}
		return s.constants.InstrumentTypes.Underlying, asset.ID, asset.SymbolFA, asset.TSETMCID, nil
	case s.constants.InstrumentTypes.Option, "OPTIONS":
		contract, err := s.optionRepo.GetBySymbol(s.db, normSymbol)
		if err != nil {
			return "", 0, "", nil, err
		}
		if contract == nil {
			return "", 0, "", nil, exception.NotFoundError{Item: s.constants.Field.OptionContract}
		}
		return s.constants.InstrumentTypes.Option, contract.ID, contract.SymbolFA, contract.TSETMCID, nil
	default:
		var vErr exception.ValidationErrors
		vErr.Add("instrumentType", s.constants.Tag.Invalid)
		return "", 0, "", nil, vErr
	}
}
