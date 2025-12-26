package csv

import (
	"os"
	"strconv"
	"strings"
)

// CSVConfig holds CSV processing configuration
type CSVConfig struct {
	Path         string
	CacheDir     string
	Concurrency  int
	LogLevel     string
	ResetOnStart bool
}

// LoadCSVConfig loads CSV configuration from environment variables
func LoadCSVConfig() (*CSVConfig, error) {
	cfg := &CSVConfig{
		Path:         getEnv("CSV_PATH", "/poc-products-short.csv"),
		CacheDir:     getEnv("CACHE_DIR", "/cache"),
		Concurrency:  getEnvInt("CONCURRENCY", 8),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		ResetOnStart: getEnvBool("RESET_ON_START", false),
	}

	return cfg, nil
}

// Helper functions copied from config package
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return os.ExpandEnv(value)
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		l := strings.ToLower(value)
		return l == "1" || l == "true" || l == "yes"
	}
	return fallback
}
