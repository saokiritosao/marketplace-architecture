package middleware

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"marketplace/internal/apperror"
)

var (
	propertyPattern  = regexp.MustCompile(`(?i)property[ =\"]+([A-Za-z0-9_]+)`)
	parameterPattern = regexp.MustCompile(`(?i)(?:parameter|argument)[ =\"]+([A-Za-z0-9_]+)`)
)

func OpenAPI(spec *openapi3.T) func(http.Handler) http.Handler {
	spec.Servers = nil

	return nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		Options: openapi3filter.Options{
			AuthenticationFunc: func(context.Context, *openapi3filter.AuthenticationInput) error {
				// JWT проверяется отдельным middleware. Валидатор здесь отвечает
				// только за форму запроса и ограничения из OpenAPI.
				return nil
			},
			MultiError: true,
		},
		DoNotValidateServers: true,
		ErrorHandlerWithOpts: func(
			_ context.Context,
			err error,
			w http.ResponseWriter,
			_ *http.Request,
			_ nethttpmiddleware.ErrorHandlerOpts,
		) {
			writeAppError(w, apperror.Validation(validationDetails(err)))
		},
	})
}

func validationDetails(err error) map[string]any {
	details := make(map[string]any)
	appendValidationError(details, err)
	if len(details) == 0 {
		details["request"] = "request does not match the OpenAPI contract"
	}
	return details
}

func appendValidationError(details map[string]any, err error) {
	if err == nil {
		return
	}

	var multi openapi3.MultiError
	if errors.As(err, &multi) {
		for _, item := range multi {
			appendValidationError(details, item)
		}
		return
	}

	message := err.Error()
	field := validationField(message)
	if old, exists := details[field]; exists {
		details[field] = old.(string) + "; " + message
		return
	}
	details[field] = message
}

func validationField(message string) string {
	if match := propertyPattern.FindStringSubmatch(message); len(match) == 2 {
		return match[1]
	}
	if match := parameterPattern.FindStringSubmatch(message); len(match) == 2 {
		return match[1]
	}

	lower := strings.ToLower(message)
	for _, field := range []string{
		"name", "description", "price", "stock", "category", "items", "quantity",
		"promo_code", "email", "password", "page", "size", "status", "id",
	} {
		if strings.Contains(lower, field) {
			return field
		}
	}
	return "request"
}
