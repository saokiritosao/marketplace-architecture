package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"marketplace/internal/api/generated"
	"marketplace/internal/apperror"
	"marketplace/internal/requestctx"
)

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return apperror.Validation(map[string]any{"body": err.Error()})
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return apperror.Validation(map[string]any{"body": "request must contain one JSON object"})
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if value != nil {
		_ = json.NewEncoder(w).Encode(value)
	}
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	appErr, ok := apperror.As(err)
	if !ok {
		appErr = apperror.Internal(err)
	}

	if appErr.Status >= http.StatusInternalServerError {
		requestID := ""
		if info, ok := requestctx.RequestInfoFrom(r.Context()); ok {
			requestID = info.ID
		}
		slog.Error("request failed", "request_id", requestID, "error", appErr.Cause)
	}

	response := generated.ErrorResponse{
		ErrorCode: generated.ErrorCode(appErr.Code),
		Message:   appErr.Message,
	}
	if appErr.Details != nil {
		details := map[string]interface{}(appErr.Details)
		response.Details = &details
	}
	writeJSON(w, appErr.Status, response)
}

func serviceUnavailable(err error) *apperror.Error {
	return apperror.Wrap(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Database is unavailable", err)
}

// RouteErrorHandler используется сгенерированным роутером, если path/query
// параметр нельзя преобразовать в нужный тип.
func RouteErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	writeError(w, r, apperror.Validation(map[string]any{"request": err.Error()}))
}
