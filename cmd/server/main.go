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
	customMw "github.com/Tnembull/go-core-backend/internal/middleware"
	"github.com/Tnembull/go-core-backend/internal/repository"
	"github.com/Tnembull/go-core-backend/internal/service"
	"github.com/Tnembull/go-core-backend/pkg/logger"
	"github.com/Tnembull/go-core-backend/pkg/response"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const AppVersion = "1.0.0"

func main() {
	cfg := config.Load()
	logger.Init(cfg.AppEnv)

	slog.Info("initializing go-core-backend", "version", AppVersion, "env", cfg.AppEnv)

	// Layer Inversion & Dependency Injection
	userRepo := repository.NewInMemoryUserRepository()
	authSvc := service.NewAuthService(userRepo, cfg)
	healthSvc := service.NewHealthService(AppVersion)

	healthHandler := handler.NewHealthHandler(healthSvc)
	authHandler := handler.NewAuthHandler(authSvc)
	userHandler := handler.NewUserHandler(authSvc)

	// Router setup
	r := chi.NewRouter()

	// Global Core Middlewares
	r.Use(customMw.RequestID)
	r.Use(customMw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(customMw.PrometheusMetrics)

	// Rate Limiting Middleware
	rateLimiter := customMw.NewIPRateLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)
	r.Use(rateLimiter.Middleware)

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Request-ID", "X-Trace-ID", "X-API-Key"},
		ExposedHeaders:   []string{"Link", "X-Request-ID", "X-Trace-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Kubernetes Probes & Observability Endpoints
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
		})

		// JWT Protected Routes
		r.Group(func(r chi.Router) {
			r.Use(customMw.JWTAuth(authSvc))

			r.Get("/users/me", userHandler.GetMe)

			// Admin RBAC Protected Route
			r.With(customMw.RequireRole("admin")).Get("/admin/dashboard", userHandler.AdminDashboard)
		})

		// Internal API Key Protected Route
		r.Group(func(r chi.Router) {
			r.Use(customMw.APIKeyAuth(cfg.APIKey))

			r.Get("/internal/ping", func(w http.ResponseWriter, req *http.Request) {
				traceID := customMw.GetRequestID(req.Context())
				response.JSON(w, http.StatusOK, map[string]string{"message": "internal system operational"}, traceID)
			})
		})
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Server execution in background goroutine
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	// Graceful Shutdown Listener
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server fatal error", "error", err)
			os.Exit(1)
		}
	case sig := <-shutdown:
		slog.Info("received shutdown signal", "signal", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			slog.Error("graceful shutdown failed, forcing close", "error", err)
			_ = server.Close()
			os.Exit(1)
		}
		slog.Info("server shutdown gracefully completed")
	}
}
