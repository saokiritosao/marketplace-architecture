package middleware

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"marketplace/internal/requestctx"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(status int) {
	if r.status != 0 {
		return
	}
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(body)
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startedAt := time.Now().UTC()
			requestID := newRequestID()
			info := &requestctx.RequestInfo{ID: requestID}
			r = r.WithContext(requestctx.WithRequestInfo(r.Context(), info))
			w.Header().Set("X-Request-Id", requestID)

			requestBody := readBodyForLog(r)
			recorder := &responseRecorder{ResponseWriter: w}
			next.ServeHTTP(recorder, r)
			if recorder.status == 0 {
				recorder.status = http.StatusOK
			}

			endpoint := strings.TrimPrefix(r.Pattern, r.Method+" ")
			if endpoint == "" {
				endpoint = r.URL.Path
			}

			attributes := []any{
				"request_id", requestID,
				"method", r.Method,
				"endpoint", endpoint,
				"status_code", recorder.status,
				"duration_ms", time.Since(startedAt).Milliseconds(),
				"timestamp", startedAt.Format(time.RFC3339Nano),
			}
			if info.UserID == nil {
				attributes = append(attributes, "user_id", nil)
			} else {
				attributes = append(attributes, "user_id", *info.UserID)
			}
			if requestBody != nil {
				attributes = append(attributes, "request_body", requestBody)
			}
			logger.Info("api request", attributes...)
		})
	}
}

func readBodyForLog(r *http.Request) any {
	if r.Method != http.MethodPost && r.Method != http.MethodPut && r.Method != http.MethodDelete {
		return nil
	}
	if r.Body == nil {
		return nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "<failed to read body>"
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}

	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return string(body)
	}
	return maskSensitive(value)
}

func maskSensitive(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			switch strings.ToLower(key) {
			case "password", "access_token", "refresh_token":
				typed[key] = "***"
			default:
				typed[key] = maskSensitive(child)
			}
		}
	case []any:
		for index, child := range typed {
			typed[index] = maskSensitive(child)
		}
	}
	return value
}

func newRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(bytes)
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32]
}
