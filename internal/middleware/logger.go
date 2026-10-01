package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		traceID := GetRequestID(r.Context())

		wrapper := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(wrapper, r)

		duration := time.Since(start)

		slog.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapper.statusCode,
			"duration_ms", duration.Milliseconds(),
			"client_ip", r.RemoteAddr,
			"trace_id", traceID,
		)
	})
}
