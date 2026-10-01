package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tnembull/go-core-backend/internal/config"
	"github.com/Tnembull/go-core-backend/internal/handler"
	customMw "github.com/Tnembull/go-core-backend/internal/middleware"
	"github.com/Tnembull/go-core-backend/internal/repository"
	"github.com/Tnembull/go-core-backend/internal/service"
	"github.com/go-chi/chi/v5"
)

func setupTestRouter() (*chi.Mux, service.AuthService) {
	cfg := &config.Config{
		JWTSecret:      "integration-secret-test-key-321",
		JWTExpiration:  1,
		RateLimitRPS:   100,
		RateLimitBurst: 200,
		APIKey:         "test-api-key-xyz",
	}

	repo := repository.NewInMemoryUserRepository()
	authSvc := service.NewAuthService(repo, cfg)
	healthSvc := service.NewHealthService("1.0.0-test", "test")

	healthH := handler.NewHealthHandler(healthSvc)
	authH := handler.NewAuthHandler(authSvc)
	userH := handler.NewUserHandler(authSvc)

	r := chi.NewRouter()
	r.Use(customMw.RequestID)

	r.Get("/healthz", healthH.Liveness)
	r.Get("/readyz", healthH.Readiness)
	r.Get("/api/v1/system/info", healthH.SystemInfo)

	r.Post("/api/v1/auth/register", authH.Register)
	r.Post("/api/v1/auth/login", authH.Login)

	r.Group(func(r chi.Router) {
		r.Use(customMw.JWTAuth(authSvc))
		r.Get("/api/v1/users/me", userH.GetMe)
		r.With(customMw.RequireRole("admin")).Get("/api/v1/admin/dashboard", userH.AdminDashboard)
	})

	r.Group(func(r chi.Router) {
		r.Use(customMw.APIKeyAuth(cfg.APIKey))
		r.Get("/api/v1/internal/ping", func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})
	})

	return r, authSvc
}

func TestHealthEndpoints(t *testing.T) {
	r, _ := setupTestRouter()

	req, _ := http.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 on /healthz, got %d", w.Code)
	}

	reqReady, _ := http.NewRequest("GET", "/readyz", nil)
	wReady := httptest.NewRecorder()
	r.ServeHTTP(wReady, reqReady)

	if wReady.Code != http.StatusOK {
		t.Fatalf("expected status 200 on /readyz, got %d", wReady.Code)
	}
}

func TestAuthAndProtectedFlow(t *testing.T) {
	r, _ := setupTestRouter()

	// 1. Register User
	regPayload := `{"email":"devops@example.com","password":"StrongPassword123","role":"admin"}`
	reqReg, _ := http.NewRequest("POST", "/api/v1/auth/register", bytes.NewBufferString(regPayload))
	reqReg.Header.Set("Content-Type", "application/json")
	wReg := httptest.NewRecorder()
	r.ServeHTTP(wReg, reqReg)

	if wReg.Code != http.StatusCreated {
		t.Fatalf("expected 201 on register, got %d. Body: %s", wReg.Code, wReg.Body.String())
	}

	// 2. Login User
	loginPayload := `{"email":"devops@example.com","password":"StrongPassword123"}`
	reqLogin, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBufferString(loginPayload))
	reqLogin.Header.Set("Content-Type", "application/json")
	wLogin := httptest.NewRecorder()
	r.ServeHTTP(wLogin, reqLogin)

	if wLogin.Code != http.StatusOK {
		t.Fatalf("expected 200 on login, got %d", wLogin.Code)
	}

	var resp struct {
		Success bool                 `json:"success"`
		Data    service.AuthResponse `json:"data"`
	}
	if err := json.Unmarshal(wLogin.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}

	token := resp.Data.AccessToken
	if token == "" {
		t.Fatal("expected JWT access token, got empty string")
	}

	// 3. Access Protected /users/me without token -> 401
	reqMeNoAuth, _ := http.NewRequest("GET", "/api/v1/users/me", nil)
	wMeNoAuth := httptest.NewRecorder()
	r.ServeHTTP(wMeNoAuth, reqMeNoAuth)

	if wMeNoAuth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on unauthenticated access, got %d", wMeNoAuth.Code)
	}

	// 4. Access Protected /users/me with Bearer token -> 200
	reqMe, _ := http.NewRequest("GET", "/api/v1/users/me", nil)
	reqMe.Header.Set("Authorization", "Bearer "+token)
	wMe := httptest.NewRecorder()
	r.ServeHTTP(wMe, reqMe)

	if wMe.Code != http.StatusOK {
		t.Fatalf("expected 200 on authenticated /users/me, got %d. Body: %s", wMe.Code, wMe.Body.String())
	}

	// 5. Access Admin Protected /admin/dashboard with Admin role -> 200
	reqAdmin, _ := http.NewRequest("GET", "/api/v1/admin/dashboard", nil)
	reqAdmin.Header.Set("Authorization", "Bearer "+token)
	wAdmin := httptest.NewRecorder()
	r.ServeHTTP(wAdmin, reqAdmin)

	if wAdmin.Code != http.StatusOK {
		t.Fatalf("expected 200 on admin dashboard, got %d", wAdmin.Code)
	}
}

func TestAPIKeyProtectedEndpoint(t *testing.T) {
	r, _ := setupTestRouter()

	// Without API Key -> 401
	reqNoKey, _ := http.NewRequest("GET", "/api/v1/internal/ping", nil)
	wNoKey := httptest.NewRecorder()
	r.ServeHTTP(wNoKey, reqNoKey)

	if wNoKey.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without API key, got %d", wNoKey.Code)
	}

	// With Valid API Key -> 200
	reqKey, _ := http.NewRequest("GET", "/api/v1/internal/ping", nil)
	reqKey.Header.Set("X-API-Key", "test-api-key-xyz")
	wKey := httptest.NewRecorder()
	r.ServeHTTP(wKey, reqKey)

	if wKey.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid API key, got %d", wKey.Code)
	}
}
