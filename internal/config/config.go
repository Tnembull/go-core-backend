package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv         string
	Port           string
	JWTSecret      string
	JWTExpiration  int // in hours
	RateLimitRPS   float64
	RateLimitBurst int
	APIKey         string
	AllowedOrigins []string
	EnablePProf    bool
}

func Load() *Config {
	// Attempt loading environment files in order of precedence:
	// .env.local -> .env.<APP_ENV> -> .env
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development" // default to development when run locally
	}

	_ = godotenv.Load(".env.local")
	if env != "" {
		_ = godotenv.Load(".env." + env)
	}
	_ = godotenv.Load(".env")

	// Re-evaluate APP_ENV after loading .env files
	appEnv := getEnv("APP_ENV", "development")
	isDev := appEnv == "development"

	defaultRPS := 50.0
	defaultBurst := 100
	if isDev {
		defaultRPS = 200.0 // lenient for local developer iteration
		defaultBurst = 500
	}

	defaultOrigins := []string{"http://localhost:3000", "http://localhost:5173", "http://127.0.0.1:8080"}
	if rawOrigins := os.Getenv("CORS_ALLOWED_ORIGINS"); rawOrigins != "" {
		parts := strings.Split(rawOrigins, ",")
		defaultOrigins = make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				defaultOrigins = append(defaultOrigins, trimmed)
			}
		}
	} else if !isDev {
		defaultOrigins = []string{"https://example.com", "https://api.example.com"}
	}

	return &Config{
		AppEnv:         appEnv,
		Port:           getEnv("PORT", "8080"),
		JWTSecret:      getEnv("JWT_SECRET", "super-secure-production-jwt-secret-key-change-me"),
		JWTExpiration:  getEnvInt("JWT_EXPIRATION_HOURS", 24),
		RateLimitRPS:   getEnvFloat("RATE_LIMIT_RPS", defaultRPS),
		RateLimitBurst: getEnvInt("RATE_LIMIT_BURST", defaultBurst),
		APIKey:         getEnv("INTERNAL_API_KEY", "core-secret-api-key-2026"),
		AllowedOrigins: defaultOrigins,
		EnablePProf:    getEnvBool("ENABLE_PPROF", isDev),
	}
}

func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func (c *Config) IsTest() bool {
	return c.AppEnv == "test"
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
	if val := os.Getenv(key); val != "" {
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}
