package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const (
	RequestIDKey    contextKey = "request_id"
	HeaderRequestID string     = "X-Request-ID"
	HeaderTraceID   string     = "X-Trace-ID"
)

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get(HeaderRequestID)
		if reqID == "" {
			reqID = r.Header.Get(HeaderTraceID)
		}
		if reqID == "" {
			reqID = uuid.New().String()
		}

		ctx := context.WithValue(r.Context(), RequestIDKey, reqID)
		w.Header().Set(HeaderRequestID, reqID)
		w.Header().Set(HeaderTraceID, reqID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetRequestID(ctx context.Context) string {
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		return reqID
	}
	return "unknown"
}
