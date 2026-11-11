package middleware

import (
	"net/http"
	"strings"

	"github.com/QuantDevelopers/Backend/bootstrap"
	"github.com/QuantDevelopers/Backend/internal/domain/localization"
	"github.com/labstack/echo/v4"
)

type LocalizationMiddleware struct {
	constants  *bootstrap.Constants
	translator localization.Translator
}

func NewLocalization(constants *bootstrap.Constants, translator localization.Translator) *LocalizationMiddleware {
	return &LocalizationMiddleware{
		translator: translator,
		constants:  constants,
	}
}

func (lm *LocalizationMiddleware) Localization(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		locale := normalizeLocale(getLocale(ctx.Request()))
		translatorInstance := lm.translator.GetTranslator(locale)
		ctx.Set(lm.constants.Context.Translator, translatorInstance)
		return next(ctx)
	}
}

func getLocale(request *http.Request) string {
	return request.Header.Get("Accept-Language")
}

func normalizeLocale(header string) string {
	lower := strings.ToLower(strings.TrimSpace(header))
	switch {
	case strings.HasPrefix(lower, "fa"):
		return "fa_IR"
	case strings.HasPrefix(lower, "en"):
		return "en_US"
	default:
		return "en_US"
	}
}
