package controller

import (
	"errors"
	"reflect"
	"strings"

	"github.com/QuantDevelopers/Backend/internal/domain/exception"
	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

var validate = validator.New()

func Validated[T any](ctx echo.Context) T {
	var params T

	if err := ctx.Bind(&params); err != nil {
		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) && httpErr.Internal != nil {
			panic(exception.BindingError{Err: httpErr.Internal})
		}
		panic(exception.BindingError{Err: err})
	}

	if err := validate.Struct(params); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			panic(formatValidationErrors[T](validationErrors))
		}
		panic(err)
	}

	return params
}

func formatValidationErrors[T any](errs validator.ValidationErrors) exception.ValidationErrors {
	var params T
	var validationErrors exception.ValidationErrors
	tagTypes := []string{"json", "param", "query", "form"}

	t := reflect.TypeOf(params)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}

	for _, e := range errs {
		field, found := t.FieldByName(e.StructField())
		fieldName := e.StructField()
		if found {
			fieldName = getAnyTag(field, tagTypes...)
		}
		validationErrors.Add(fieldName, e.Tag())
	}

	return validationErrors
}

func getAnyTag(field reflect.StructField, tagNames ...string) string {
	for _, tagName := range tagNames {
		if tag := field.Tag.Get(tagName); tag != "" {
			name := strings.Split(tag, ",")[0]
			if name != "" && name != "-" {
				return name
			}
		}
	}
	return field.Name
}
