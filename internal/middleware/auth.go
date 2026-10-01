package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Tnembull/go-core-backend/internal/model"
	"github.com/Tnembull/go-core-backend/internal/service"
	"github.com/Tnembull/go-core-backend/pkg/response"
)

type userContextKey string

const (
	UserClaimsKey userContextKey = "user_claims"
	HeaderAPIKey  string         = "X-API-Key"
)

func JWTAuth(authSvc service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := GetRequestID(r.Context())

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Error(w, http.StatusUnauthorized, "authorization header missing", traceID)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				response.Error(w, http.StatusUnauthorized, "invalid authorization format: expected Bearer <token>", traceID)
				return
			}

			claims, err := authSvc.ValidateToken(parts[1])
			if err != nil {
				response.Error(w, http.StatusUnauthorized, err.Error(), traceID)
				return
			}

			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := GetRequestID(r.Context())
			claims, ok := r.Context().Value(UserClaimsKey).(*model.JWTClaims)
			if !ok || claims.Role != role {
				response.Error(w, http.StatusForbidden, "forbidden: insufficient permissions", traceID)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func APIKeyAuth(expectedKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := GetRequestID(r.Context())
			key := r.Header.Get(HeaderAPIKey)
			if key == "" || key != expectedKey {
				response.Error(w, http.StatusUnauthorized, "invalid or missing X-API-Key header", traceID)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func GetUserClaims(ctx context.Context) *model.JWTClaims {
	if claims, ok := ctx.Value(UserClaimsKey).(*model.JWTClaims); ok {
		return claims
	}
	return nil
}
