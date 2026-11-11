package localization

import (
	"fmt"
	"sync"

	domainloc "github.com/QuantDevelopers/Backend/internal/domain/localization"
	"github.com/go-playground/locales/en_US"
	"github.com/go-playground/locales/fa_IR"
	ut "github.com/go-playground/universal-translator"
)

type translationService struct {
	universalTranslator *ut.UniversalTranslator
	mu                  sync.RWMutex
}

func NewTranslationService() domainloc.Translator {
	service := &translationService{
		universalTranslator: createUniversalTranslator(),
	}
	service.loadAndAddTranslations()
	return service
}

func (t *translationService) GetTranslator(locale string) domainloc.TranslatorInstance {
	t.mu.RLock()
	defer t.mu.RUnlock()

	translator, found := t.universalTranslator.GetTranslator(locale)
	if !found {
		translator, _ = t.universalTranslator.GetTranslator("en_US")
	}

	return &translatorInstanceImpl{translator: translator}
}

type translatorInstanceImpl struct {
	translator ut.Translator
}

func (t *translatorInstanceImpl) Translate(key string, params ...string) (string, error) {
	return t.translator.T(key, params...)
}

func (t *translatorInstanceImpl) Locale() string {
	return t.translator.Locale()
}

func createUniversalTranslator() *ut.UniversalTranslator {
	en := en_US.New()
	fa := fa_IR.New()
	return ut.New(en, en, fa)
}

func (t *translationService) loadAndAddTranslations() {
	addTranslations("fa_IR", Persian, t.universalTranslator)
	addTranslations("en_US", English, t.universalTranslator)
}

func addTranslations(locale string, translations map[string]interface{}, universalTranslator *ut.UniversalTranslator) {
	translator, found := universalTranslator.GetTranslator(locale)
	if !found {
		panic(fmt.Errorf("translator for locale %s not found", locale))
	}

	flattened := make(map[string]string)
	flattenMap("", translations, flattened)

	for key, translation := range flattened {
		_ = translator.Add(key, translation, true)
	}
}

func flattenMap(prefix string, input map[string]interface{}, output map[string]string) {
	for k, v := range input {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}
		switch value := v.(type) {
		case map[string]interface{}:
			flattenMap(fullKey, value, output)
		case string:
			output[fullKey] = value
		}
	}
}
