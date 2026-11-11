package usecase

import underlyingdto "github.com/QuantDevelopers/Backend/internal/application/dto/underlying"

type UnderlyingService interface {
	GetUnderlyings() (underlyingdto.UnderlyingListResponse, error)
	GetUnderlyingBySymbol(symbolFA string) (underlyingdto.UnderlyingResponse, error)
}
