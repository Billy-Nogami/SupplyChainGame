package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type App struct {
	HTTPAddr       string
	AllowedOrigins []string
	Redis          Redis
}

type Redis struct {
	Enabled  bool
	Addr     string
	Password string
	DB       int
	KeyTTL   time.Duration
}

func Load() App {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return App{
		HTTPAddr:       ":" + port,
		AllowedOrigins: splitCSV(os.Getenv("ALLOWED_ORIGINS")),
		Redis: Redis{
			Enabled:  os.Getenv("REDIS_ADDR") != "",
			Addr:     os.Getenv("REDIS_ADDR"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       envInt("REDIS_DB", 0),
			KeyTTL:   envDuration("REDIS_KEY_TTL", 72*time.Hour),
		},
	}
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}

	return result
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}
