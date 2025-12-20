package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return os.ExpandEnv(value)
	}
	return fallback
}

func getEnvArray(key string, fallback []string) []string {
	if value := os.Getenv(key); value != "" {
		// Expand environment variables first
		expandedValue := os.ExpandEnv(value)
		// Split the string by comma and trim spaces
		var parts []string
		for _, v := range strings.Split(expandedValue, ",") {
			// Only append non-empty values after trimming
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				parts = append(parts, trimmed)
			}
		}
		return parts
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

func getEnvTimeDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
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
