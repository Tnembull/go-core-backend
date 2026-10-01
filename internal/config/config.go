package config

import (
	"os"
	"strconv"
)

type Config struct {
	AppEnv         string
	Port           string
	JWTSecret      string
	JWTExpiration  int // in hours
	RateLimitRPS   float64
	RateLimitBurst int
	APIKey         string
}

func Load() *Config {
	return &Config{
		AppEnv:         getEnv("APP_ENV", "production"),
		Port:           getEnv("PORT", "8080"),
		JWTSecret:      getEnv("JWT_SECRET", "super-secure-production-jwt-secret-key-change-me"),
		JWTExpiration:  getEnvInt("JWT_EXPIRATION_HOURS", 24),
		RateLimitRPS:   getEnvFloat("RATE_LIMIT_RPS", 50.0),
		RateLimitBurst: getEnvInt("RATE_LIMIT_BURST", 100),
		APIKey:         getEnv("INTERNAL_API_KEY", "core-secret-api-key-2026"),
	}
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
