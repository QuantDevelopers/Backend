package underlyingdto

type GetUnderlyingBySymbolRequest struct {
	SymbolFA string `param:"symbol_fa" validate:"required"`
}
