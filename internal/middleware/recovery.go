package middleware

import (
	"log/slog"
	"net/http"

	"marketplace/internal/apperror"
	"marketplace/internal/requestctx"
)

func Recovery(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					requestID := ""
					if info, ok := requestctx.RequestInfoFrom(r.Context()); ok {
						requestID = info.ID
					}
					logger.Error("panic recovered", "request_id", requestID, "panic", recovered)
					writeAppError(w, apperror.Internal(nil))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
