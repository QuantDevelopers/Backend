package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/QuantDevelopers/Backend/bootstrap"
	"github.com/QuantDevelopers/Backend/internal/domain/exception"
	"github.com/QuantDevelopers/Backend/internal/domain/logger"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller"
	"github.com/labstack/echo/v4"
)

const genericErrorKey = "errors.generic"

type RecoveryMiddleware struct {
	constants *bootstrap.Constants
	logger    logger.Logger
}

func NewRecovery(constants *bootstrap.Constants, log logger.Logger) *RecoveryMiddleware {
	return &RecoveryMiddleware{
		constants: constants,
		logger:    log,
	}
}

func (recovery *RecoveryMiddleware) Recovery(next echo.HandlerFunc) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		defer func() {
			if rec := recover(); rec != nil {
				if err, ok := rec.(error); ok {
					recovery.handleRecoveredError(ctx, err)
				} else {
					recovery.handleRecoveredError(ctx, fmt.Errorf("%v", rec))
				}
			}
		}()

		return next(ctx)
	}
}

func (recovery *RecoveryMiddleware) handleRecoveredError(ctx echo.Context, err error) {
	transKey := recovery.constants.Context.Translator

	switch typedErr := err.(type) {
	case exception.ValidationErrors:
		handleValidationError(ctx, typedErr, transKey)
	case exception.BindingError:
		handleBindingError(ctx, typedErr, transKey)
	case *exception.RateLimitError:
		handleRateLimitError(ctx, *typedErr, transKey)
	case exception.ConflictErrors:
		handleConflictError(ctx, typedErr, transKey)
	case *exception.AuthError:
		handleAuthError(ctx, *typedErr, transKey)
	case exception.NotFoundError:
		handleNotFoundError(ctx, typedErr, transKey)
	case exception.ForbiddenError:
		handleForbiddenError(ctx, typedErr, transKey)
	default:
		recovery.unhandledErrors(ctx, err, transKey)
	}
}

func handleValidationError(ctx echo.Context, validationErrors exception.ValidationErrors, transKey string) {
	trans := controller.GetTranslator(ctx, transKey)
	errorMessages := make(map[string]map[string]string)

	for _, validationError := range validationErrors.Errors {
		if _, ok := errorMessages[validationError.Field]; !ok {
			errorMessages[validationError.Field] = make(map[string]string)
		}
		fieldName, _ := trans.Translate(validationError.Field)
		if fieldName == "" {
			fieldName = validationError.Field
		}
		message, _ := trans.Translate(fmt.Sprintf("errors.%s", validationError.Tag), fieldName)
		if message == "" {
			message = validationError.Error()
		}
		errorMessages[validationError.Field][validationError.Tag] = message
	}

	_ = controller.Response(ctx, http.StatusUnprocessableEntity, errorMessages, nil)
}

func handleBindingError(ctx echo.Context, bindingError exception.BindingError, transKey string) {
	trans := controller.GetTranslator(ctx, transKey)
	message, _ := trans.Translate(genericErrorKey)

	var numError *strconv.NumError
	if errors.As(bindingError.Err, &numError) {
		message, _ = trans.Translate("errors.numeric", numError.Num)
	} else if errors.Is(bindingError.Err, http.ErrMissingFile) {
		message, _ = trans.Translate("errors.fileRequired")
	}

	_ = controller.Response(ctx, http.StatusBadRequest, message, nil)
}

func handleRateLimitError(ctx echo.Context, _ exception.RateLimitError, transKey string) {
	trans := controller.GetTranslator(ctx, transKey)
	message, _ := trans.Translate("errors.rateLimit")
	_ = controller.Response(ctx, http.StatusTooManyRequests, message, nil)
}

func handleConflictError(ctx echo.Context, conflictErrors exception.ConflictErrors, transKey string) {
	trans := controller.GetTranslator(ctx, transKey)
	errorMessages := make(map[string]map[string]string)

	for _, conflictError := range conflictErrors.Errors {
		if _, ok := errorMessages[conflictError.Field]; !ok {
			errorMessages[conflictError.Field] = make(map[string]string)
		}
		fieldName, _ := trans.Translate(conflictError.Field)
		message, _ := trans.Translate(fmt.Sprintf("errors.%s", conflictError.Tag), fieldName)
		errorMessages[conflictError.Field][conflictError.Tag] = message
	}

	_ = controller.Response(ctx, http.StatusConflict, errorMessages, nil)
}

func handleAuthError(ctx echo.Context, _ exception.AuthError, transKey string) {
	trans := controller.GetTranslator(ctx, transKey)
	message, _ := trans.Translate("errors.unauthorized")
	_ = controller.Response(ctx, http.StatusUnauthorized, message, nil)
}

func handleNotFoundError(ctx echo.Context, notFoundError exception.NotFoundError, transKey string) {
	trans := controller.GetTranslator(ctx, transKey)
	itemName, _ := trans.Translate(notFoundError.Item)
	if itemName == "" {
		itemName = notFoundError.Item
	}
	message, _ := trans.Translate("errors.notFound", itemName)
	_ = controller.Response(ctx, http.StatusNotFound, message, nil)
}

func handleForbiddenError(ctx echo.Context, forbiddenError exception.ForbiddenError, transKey string) {
	trans := controller.GetTranslator(ctx, transKey)
	resourceName, _ := trans.Translate(forbiddenError.Resource)
	message, _ := trans.Translate("errors.forbiddenError", resourceName)
	_ = controller.Response(ctx, http.StatusForbidden, message, nil)
}

func (recovery *RecoveryMiddleware) unhandledErrors(ctx echo.Context, err error, transKey string) {
	if recovery.logger != nil {
		recovery.logger.Error("unhandled error recovery middleware", logger.Error("error", err))
	}
	trans := controller.GetTranslator(ctx, transKey)
	errorMessage, _ := trans.Translate(genericErrorKey)
	_ = controller.Response(ctx, http.StatusInternalServerError, errorMessage, nil)
}
