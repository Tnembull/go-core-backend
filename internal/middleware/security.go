package middleware

import (
	"net/http"
)

// SecurityHeaders applies OWASP recommended defensive HTTP headers
func SecurityHeaders(isProduction bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent MIME type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Prevent Clickjacking
			w.Header().Set("X-Frame-Options", "DENY")

			// Modern browser XSS hygiene
			w.Header().Set("X-XSS-Protection", "0")

			// Privacy-preserving referrer policy
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Restrict browser capability delegation
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")

			// API-hardened Content Security Policy
			w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none';")

			// HTTP Strict Transport Security (HSTS) in production
			if isProduction {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// BodyLimit protects server memory against large payload denial-of-service (OOM)
func BodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
