package usecase

import optiondto "github.com/QuantDevelopers/Backend/internal/application/dto/option"

type OptionService interface {
	ListOptions(req optiondto.ListOptionsRequest) ([]optiondto.OptionResponse, error)
	ListOptionsPaginated(req optiondto.ListOptionsQueryRequest) (optiondto.OptionListPaginatedResponse, error)
	GetOptionBySymbol(symbolFA string) (optiondto.OptionResponse, error)
	GetOptionChain(req optiondto.GetOptionChainRequest) (optiondto.OptionChainResponse, error)
	CreateOrUpdateOption(req optiondto.SaveOptionRequest) (optiondto.OptionResponse, error)
}
