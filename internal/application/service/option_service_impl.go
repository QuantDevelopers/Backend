package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	commondto "github.com/QuantDevelopers/Backend/internal/application/dto/common"
	optiondto "github.com/QuantDevelopers/Backend/internal/application/dto/option"
	underlyingdto "github.com/QuantDevelopers/Backend/internal/application/dto/underlying"
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/domain/exception"
	domainoptions "github.com/QuantDevelopers/Backend/internal/domain/options"
	"github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	domaintext "github.com/QuantDevelopers/Backend/internal/domain/text"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

type OptionService struct {
	constants            *bootstrap.Constants
	optionRepository     postgres.OptionContractRepository
	underlyingRepository postgres.UnderlyingRepository
	dailyRepository      postgres.DailyHistoryRepository
	db                   database.Database
}

func NewOptionService(
	constants *bootstrap.Constants,
	optionRepository postgres.OptionContractRepository,
	underlyingRepository postgres.UnderlyingRepository,
	dailyRepository postgres.DailyHistoryRepository,
	db database.Database,
) *OptionService {
	return &OptionService{
		constants:            constants,
		optionRepository:     optionRepository,
		underlyingRepository: underlyingRepository,
		dailyRepository:      dailyRepository,
		db:                   db,
	}
}

func toOptionDTO(contract *entity.OptionContract) optiondto.OptionResponse {
	return optiondto.OptionResponse{
		ID:                contract.ID,
		UnderlyingAssetID: contract.UnderlyingAssetID,
		SymbolFA:          contract.SymbolFA,
		NameFA:            contract.NameFA,
		ContractType:      contract.ContractType,
		StrikePrice:       contract.StrikePrice,
		ContractSize:      contract.ContractSize,
		ExpirationDate:    contract.ExpirationDate,
		BeginDate:         contract.BeginDate,
		TSETMCID:          contract.TSETMCID,
		ISIN:              contract.ISIN,
		IsActive:          contract.IsActive,
		Status:            contract.Status,
		CreatedAt:         contract.CreatedAt,
	}
}

func (s *OptionService) ListOptions(req optiondto.ListOptionsRequest) ([]optiondto.OptionResponse, error) {
	contracts, err := s.optionRepository.GetAll(s.db, postgres.OptionFilter{
		UnderlyingAssetID: req.UnderlyingAssetID,
		ContractType:      req.ContractType,
		IsActive:          req.IsActive,
	})
	if err != nil {
		return nil, err
	}

	items := make([]optiondto.OptionResponse, len(contracts))
	for i, c := range contracts {
		items[i] = toOptionDTO(c)
	}
	return items, nil
}

func (s *OptionService) ListOptionsPaginated(req optiondto.ListOptionsQueryRequest) (optiondto.OptionListPaginatedResponse, error) {
	uaID := req.UnderlyingID
	if uaID == nil && strings.TrimSpace(req.UnderlyingSymbol) != "" && s.underlyingRepository != nil {
		normUA := domaintext.NormalizePersian(req.UnderlyingSymbol)
		ua, err := s.underlyingRepository.GetBySymbol(s.db, normUA)
		if err != nil {
			return optiondto.OptionListPaginatedResponse{}, err
		}
		if ua == nil {
			return optiondto.OptionListPaginatedResponse{}, exception.NotFoundError{Item: s.constants.Field.UnderlyingAsset}
		}
		uaID = &ua.ID
	}

	var contractTypePtr *string
	if trimmedCT := strings.ToUpper(strings.TrimSpace(req.ContractType)); trimmedCT != "" {
		if trimmedCT != "CALL" && trimmedCT != "PUT" {
			var vErr exception.ValidationErrors
			vErr.Add(s.constants.Field.ContractType, s.constants.Tag.Invalid)
			return optiondto.OptionListPaginatedResponse{}, vErr
		}
		contractTypePtr = &trimmedCT
	}

	var expiryPtr *time.Time
	if strings.TrimSpace(req.ExpirationDate) != "" {
		parsed, err := parseFlexibleDate(req.ExpirationDate)
		if err != nil {
			var vErr exception.ValidationErrors
			vErr.Add("expirationDate", s.constants.Tag.Invalid)
			return optiondto.OptionListPaginatedResponse{}, vErr
		}
		expiryPtr = &parsed
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 25
	}
	if pageSize > 200 {
		pageSize = 200
	}
	offset := (page - 1) * pageSize

	contracts, total, err := s.optionRepository.ListPaginated(s.db, postgres.OptionQueryFilter{
		UnderlyingAssetID: uaID,
		ContractType:      contractTypePtr,
		IsActive:          req.IsActive,
		Query:             domaintext.NormalizePersian(req.Query),
		ExpirationDate:    expiryPtr,
		Offset:            offset,
		Limit:             pageSize,
	})
	if err != nil {
		return optiondto.OptionListPaginatedResponse{}, err
	}

	items := make([]optiondto.OptionResponse, len(contracts))
	for i, c := range contracts {
		items[i] = toOptionDTO(c)
	}

	return optiondto.OptionListPaginatedResponse{
		Items:      items,
		Pagination: commondto.BuildPaginationMeta(total, offset, pageSize),
	}, nil
}

func (s *OptionService) GetOptionBySymbol(symbolFA string) (optiondto.OptionResponse, error) {
	norm := domaintext.NormalizePersian(symbolFA)
	contract, err := s.optionRepository.GetBySymbol(s.db, norm)
	if err != nil {
		return optiondto.OptionResponse{}, err
	}
	if contract == nil {
		return optiondto.OptionResponse{}, exception.NotFoundError{Item: s.constants.Field.OptionContract}
	}
	return toOptionDTO(contract), nil
}

func (s *OptionService) GetOptionChain(req optiondto.GetOptionChainRequest) (optiondto.OptionChainResponse, error) {
	if s.underlyingRepository == nil {
		return optiondto.OptionChainResponse{}, fmt.Errorf("underlying repository is not configured")
	}

	normUA := domaintext.NormalizePersian(req.SymbolFA)
	ua, err := s.underlyingRepository.GetBySymbol(s.db, normUA)
	if err != nil {
		return optiondto.OptionChainResponse{}, err
	}
	if ua == nil {
		return optiondto.OptionChainResponse{}, exception.NotFoundError{Item: s.constants.Field.UnderlyingAsset}
	}

	var expiryFilter *time.Time
	if strings.TrimSpace(req.ExpirationDate) != "" {
		parsed, err := parseFlexibleDate(req.ExpirationDate)
		if err != nil {
			var vErr exception.ValidationErrors
			vErr.Add("expirationDate", s.constants.Tag.Invalid)
			return optiondto.OptionChainResponse{}, vErr
		}
		expiryFilter = &parsed
	}

	spotPrice := req.SpotPrice
	if spotPrice == nil && s.dailyRepository != nil {
		latest, err := s.dailyRepository.GetByInstrument(s.db, "UNDERLYING", ua.ID, 1)
		if err == nil && len(latest) > 0 && latest[0].ClosePrice > 0 {
			val := latest[0].ClosePrice
			spotPrice = &val
		}
	}

	active := true
	contracts, err := s.optionRepository.GetAll(s.db, postgres.OptionFilter{
		UnderlyingAssetID: &ua.ID,
		IsActive:          &active,
	})
	if err != nil {
		return optiondto.OptionChainResponse{}, err
	}

	expSet := make(map[string]struct{})
	rowMap := make(map[string]*optiondto.OptionChainRow)

	for _, c := range contracts {
		expStr := c.ExpirationDate.Format("2006-01-02")
		expSet[expStr] = struct{}{}
		if expiryFilter != nil && expStr != expiryFilter.Format("2006-01-02") {
			continue
		}

		key := fmt.Sprintf("%s|%.2f", expStr, c.StrikePrice)
		row, exists := rowMap[key]
		if !exists {
			row = &optiondto.OptionChainRow{
				ExpirationDate: expStr,
				StrikePrice:    c.StrikePrice,
				ContractSize:   c.ContractSize,
			}
			rowMap[key] = row
		}

		dto := toOptionDTO(c)
		switch strings.ToUpper(c.ContractType) {
		case "CALL":
			row.Call = &dto
			if spotPrice != nil {
				if m, mErr := domainoptions.ClassifyMoneyness(domainoptions.ContractTypeCall, c.StrikePrice, *spotPrice, domainoptions.DefaultATMThresholdPct); mErr == nil {
					row.CallMoneyness = string(m)
				}
			}
		case "PUT":
			row.Put = &dto
			if spotPrice != nil {
				if m, mErr := domainoptions.ClassifyMoneyness(domainoptions.ContractTypePut, c.StrikePrice, *spotPrice, domainoptions.DefaultATMThresholdPct); mErr == nil {
					row.PutMoneyness = string(m)
				}
			}
		}
	}

	expirations := make([]string, 0, len(expSet))
	for e := range expSet {
		expirations = append(expirations, e)
	}
	sort.Strings(expirations)

	rows := make([]optiondto.OptionChainRow, 0, len(rowMap))
	for _, r := range rowMap {
		rows = append(rows, *r)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].ExpirationDate == rows[j].ExpirationDate {
			return rows[i].StrikePrice < rows[j].StrikePrice
		}
		return rows[i].ExpirationDate < rows[j].ExpirationDate
	})

	return optiondto.OptionChainResponse{
		Underlying: underlyingdto.UnderlyingResponse{
			ID:         ua.ID,
			SymbolFA:   ua.SymbolFA,
			NameFA:     ua.NameFA,
			TSETMCID:   ua.TSETMCID,
			MarketType: ua.MarketType,
			IsActive:   ua.IsActive,
			CreatedAt:  ua.CreatedAt,
		},
		SpotPrice:   spotPrice,
		Expirations: expirations,
		TotalRows:   len(rows),
		Rows:        rows,
	}, nil
}

func parseFlexibleDate(raw string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) == 8 {
		return time.Parse("20060102", trimmed)
	}
	return time.Parse("2006-01-02", trimmed)
}

func (s *OptionService) CreateOrUpdateOption(req optiondto.SaveOptionRequest) (optiondto.OptionResponse, error) {
	contractSize := req.ContractSize
	if contractSize == 0 {
		contractSize = 1000
	}
	status := req.Status
	if status == "" {
		status = s.constants.ContractStatuses.Active
	}

	contract := &entity.OptionContract{
		UnderlyingAssetID: req.UnderlyingAssetID,
		SymbolFA:          req.SymbolFA,
		NameFA:            req.NameFA,
		ContractType:      req.ContractType,
		StrikePrice:       req.StrikePrice,
		ContractSize:      contractSize,
		ExpirationDate:    req.ExpirationDate,
		BeginDate:         req.BeginDate,
		TSETMCID:          req.TSETMCID,
		ISIN:              req.ISIN,
		IsActive:          req.IsActive,
		Status:            status,
	}
	if err := s.optionRepository.Save(s.db, contract); err != nil {
		return optiondto.OptionResponse{}, err
	}
	return toOptionDTO(contract), nil
}
