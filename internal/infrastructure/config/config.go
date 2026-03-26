package config

import (
	"os"
	"strings"
)

type App struct {
	HTTPAddr       string
	AllowedOrigins []string
}

func Load() App {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return App{
		HTTPAddr:       ":" + port,
		AllowedOrigins: splitCSV(os.Getenv("ALLOWED_ORIGINS")),
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
