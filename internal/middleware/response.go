package middleware

import (
	"encoding/json"
	"net/http"

	"marketplace/internal/api/generated"
	"marketplace/internal/apperror"
)

func writeAppError(w http.ResponseWriter, appErr *apperror.Error) {
	response := generated.ErrorResponse{
		ErrorCode: generated.ErrorCode(appErr.Code),
		Message:   appErr.Message,
	}
	if appErr.Details != nil {
		details := map[string]interface{}(appErr.Details)
		response.Details = &details
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(appErr.Status)
	_ = json.NewEncoder(w).Encode(response)
}
