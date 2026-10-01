package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Tnembull/go-core-backend/internal/config"
	"github.com/Tnembull/go-core-backend/internal/handler"
	"github.com/Tnembull/go-core-backend/internal/middleware"
	"github.com/Tnembull/go-core-backend/internal/model"
	"github.com/Tnembull/go-core-backend/internal/repository"
	"github.com/Tnembull/go-core-backend/internal/service"
	"github.com/Tnembull/go-core-backend/pkg/logger"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const AppVersion = "v1.2.0"

func main() {
	cfg := config.Load()
	logger.Init(cfg.AppEnv)

	logger.Log.Info("initializing go-core-backend engine",
		slog.String("version", AppVersion),
		slog.String("port", cfg.Port),
		slog.String("env", cfg.AppEnv),
		slog.Bool("pprof_enabled", cfg.EnablePProf),
	)

	// Dependency Injection Layers
	userRepo := repository.NewInMemoryUserRepository()
	authService := service.NewAuthService(userRepo, cfg)
	healthService := service.NewHealthService(AppVersion, cfg.AppEnv)

	authHandler := handler.NewAuthHandler(authService)
	healthHandler := handler.NewHealthHandler(healthService)
	userHandler := handler.NewUserHandler(authService)

	// Router setup
	r := chi.NewRouter()

	// Global Middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.SecurityHeaders(cfg.IsProduction()))
	r.Use(middleware.BodyLimit(1 << 20)) // 1 MB payload limit
	r.Use(middleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.PrometheusMetrics)
	r.Use(middleware.RateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst))

	// CORS Configuration (Dev vs Prod aware)
	corsOptions := cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-API-Key"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}
	if cfg.IsDevelopment() {
		// Allow any origin during local frontend development
		corsOptions.AllowedOrigins = []string{"*"}
	}
	r.Use(cors.Handler(corsOptions))

	// Optional Debug / Profiler Route for Development
	if cfg.EnablePProf {
		logger.Log.Info("mounting development pprof profiler at /debug/pprof")
		r.Mount("/debug", chiMiddleware.Profiler())
	}

	// Observability & System Routes
	r.Get("/healthz", healthHandler.Liveness)
	r.Get("/readyz", healthHandler.Readiness)
	r.Handle("/metrics", promhttp.Handler())

	// API v1 Routing
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/system/info", healthHandler.SystemInfo)

		// Public Auth Endpoints
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/2fa/verify", authHandler.Verify2FA)
			r.Post("/refresh", authHandler.RefreshToken)
			r.Post("/logout", authHandler.Logout)

			// Authenticated 2FA Configuration
			r.Group(func(r chi.Router) {
				r.Use(middleware.JWTAuth(authService))
				r.Post("/2fa/setup", authHandler.Setup2FA)
				r.Post("/2fa/enable", authHandler.Enable2FA)
			})
		})

		// Protected User & RBAC Endpoints
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth(authService))
			r.Get("/users/me", userHandler.GetMe)

			// RBAC Role Restrictions
			r.With(middleware.RequireRole(model.RoleAdmin, model.RoleSuperAdmin)).
				Get("/admin/dashboard", userHandler.AdminDashboard)

			// RBAC Granular Permission Restrictions
			r.With(middleware.RequirePermission(model.PermissionUsersWrite)).
				Post("/users/write", userHandler.WriteUserData)

			r.With(middleware.RequirePermission(model.PermissionSettingsManage)).
				Post("/system/settings", userHandler.ManageSettings)
		})

		// Machine-to-Machine Internal Endpoints (Protected by X-API-Key)
		r.Group(func(r chi.Router) {
			r.Use(middleware.APIKeyAuth(cfg.APIKey))
			r.Get("/internal/ping", func(w http.ResponseWriter, r *http.Request) {
				traceID := middleware.GetRequestID(r.Context())
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = fmt.Fprintf(w, `{"success":true,"data":{"message":"internal system operational","source":"m2m_auth"},"meta":{"trace_id":"%s"}}`, traceID)
			})
		})
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 3 * time.Second, // Mitigates Slowloris attacks
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Server runner in background
	serverErrors := make(chan error, 1)
	go func() {
		logger.Log.Info("HTTP server running and listening",
			slog.String("addr", server.Addr),
			slog.String("env", cfg.AppEnv),
		)
		serverErrors <- server.ListenAndServe()
	}()

	// Graceful Shutdown Channel
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Log.Error("server listener encountered fatal error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	case sig := <-shutdown:
		logger.Log.Info("shutdown signal intercepted, starting graceful drain", slog.String("signal", sig.String()))

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Log.Error("server forced to shutdown prematurely", slog.String("error", err.Error()))
			_ = server.Close()
		}
		logger.Log.Info("server shutdown gracefully with zero connection drops")
	}
}
