package middleware

import (
	"errors"
	"net/http"
	"strings"

	"marketplace/internal/apperror"
	"marketplace/internal/requestctx"
	"marketplace/internal/security"
)

func Auth(tokens *security.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isPublic(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			header := r.Header.Get("Authorization")
			parts := strings.Fields(header)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeAppError(w, apperror.TokenInvalid())
				return
			}

			userID, role, err := tokens.ParseAccess(parts[1])
			if err != nil {
				if errors.Is(err, security.ErrTokenExpired) {
					writeAppError(w, apperror.TokenExpired())
					return
				}
				writeAppError(w, apperror.TokenInvalid())
				return
			}

			ctx := requestctx.WithActor(r.Context(), requestctx.Actor{ID: userID, Role: role})
			if info, ok := requestctx.RequestInfoFrom(ctx); ok {
				info.UserID = &userID
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func isPublic(path string) bool {
	return path == "/health" || strings.HasPrefix(path, "/auth/")
}
