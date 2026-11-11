package usecase

import optiondto "github.com/QuantDevelopers/Backend/internal/application/dto/option"

type OptionService interface {
	ListOptions(req optiondto.ListOptionsRequest) ([]optiondto.OptionResponse, error)
	GetOptionBySymbol(symbolFA string) (optiondto.OptionResponse, error)
	CreateOrUpdateOption(req optiondto.SaveOptionRequest) (optiondto.OptionResponse, error)
}
