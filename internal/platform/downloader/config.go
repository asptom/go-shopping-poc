package downloader

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// DownloaderConfig holds image downloader configuration
type DownloaderConfig struct {
	CacheDir     string
	ResetOnStart bool
	HTTPTimeout  time.Duration
}

// LoadDownloaderConfig loads downloader configuration from environment variables
func LoadDownloaderConfig() (*DownloaderConfig, error) {
	config := &DownloaderConfig{
		CacheDir:     getEnv("CACHE_DIR", "/cache"),
		ResetOnStart: getEnvBool("RESET_ON_START", false),
		HTTPTimeout:  getEnvTimeDuration("DOWNLOAD_TIMEOUT", 30*time.Second),
	}

	return config, nil
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
