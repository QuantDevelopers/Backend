package service

import (
	"github.com/QuantDevelopers/Backend/bootstrap"
	optiondto "github.com/QuantDevelopers/Backend/internal/application/dto/option"
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/domain/exception"
	"github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

type OptionService struct {
	constants        *bootstrap.Constants
	optionRepository postgres.OptionContractRepository
	db               database.Database
}

func NewOptionService(
	constants *bootstrap.Constants,
	optionRepository postgres.OptionContractRepository,
	db database.Database,
) *OptionService {
	return &OptionService{
		constants:        constants,
		optionRepository: optionRepository,
		db:               db,
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

func (s *OptionService) GetOptionBySymbol(symbolFA string) (optiondto.OptionResponse, error) {
	contract, err := s.optionRepository.GetBySymbol(s.db, symbolFA)
	if err != nil {
		return optiondto.OptionResponse{}, err
	}
	if contract == nil {
		return optiondto.OptionResponse{}, exception.NotFoundError{Item: s.constants.Field.OptionContract}
	}
	return toOptionDTO(contract), nil
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
