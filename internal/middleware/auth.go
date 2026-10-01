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

const UserContextKey userContextKey = "authenticated_user"

func JWTAuth(authService service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := GetRequestID(r.Context())
			authHeader := r.Header.Get("Authorization")

			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				response.Error(w, http.StatusUnauthorized, "missing or malformed authorization token", traceID)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := authService.ValidateToken(tokenStr, "access")
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "invalid, expired, or non-access authorization token", traceID)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := GetRequestID(r.Context())
			claims, ok := r.Context().Value(UserContextKey).(*model.JWTClaims)
			if !ok || claims == nil {
				response.Error(w, http.StatusUnauthorized, "unauthorized identity context", traceID)
				return
			}

			for _, role := range allowedRoles {
				if claims.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			response.Error(w, http.StatusForbidden, "access denied: insufficient role privileges", traceID)
		})
	}
}

func RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			traceID := GetRequestID(r.Context())
			claims, ok := r.Context().Value(UserContextKey).(*model.JWTClaims)
			if !ok || claims == nil {
				response.Error(w, http.StatusUnauthorized, "unauthorized identity context", traceID)
				return
			}

			if !model.HasPermission(claims.Role, permission) {
				response.Error(w, http.StatusForbidden, "access denied: missing required permission '"+permission+"'", traceID)
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
			apiKey := r.Header.Get("X-API-Key")

			if apiKey == "" || apiKey != expectedKey {
				response.Error(w, http.StatusUnauthorized, "invalid or missing x-api-key header", traceID)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
