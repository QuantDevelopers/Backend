package controller

import (
	"github.com/QuantDevelopers/Backend/internal/domain/localization"
	"github.com/labstack/echo/v4"
)

func GetTranslator(ctx echo.Context, key string) localization.TranslatorInstance {
	translator := ctx.Get(key)
	if translator == nil {
		panic("translator not registered!")
	}
	return translator.(localization.TranslatorInstance)
}

type SortParams struct {
	SortBy string `query:"sortBy" form:"sortBy"`
	Dir    string `query:"dir" form:"dir"`
}

func GetSort(c echo.Context) SortParams {
	param := Validated[SortParams](c)
	if param.Dir == "" {
		param.Dir = "ASC"
	}
	return param
}
