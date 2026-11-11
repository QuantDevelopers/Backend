package optiondto

import "time"

type ListOptionsRequest struct {
	UnderlyingAssetID *uint
	ContractType      *string
	IsActive          *bool
}

type SaveOptionRequest struct {
	UnderlyingAssetID uint
	SymbolFA          string
	NameFA            string
	ContractType      string
	StrikePrice       float64
	ContractSize      int
	ExpirationDate    time.Time
	BeginDate         *time.Time
	TSETMCID          *string
	ISIN              *string
	IsActive          bool
	Status            string
}
