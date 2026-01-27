package product

import (
	"errors"
	"time"

	"go-shopping-poc/internal/platform/config"
)

// Config defines product service configuration
type Config struct {
	// Database configuration
	DatabaseURL string `mapstructure:"db_url" validate:"required"`

	// HTTP server configuration
	ServicePort string `mapstructure:"product_service_port" validate:"required"`

	// Image processing configuration
	CacheDir     string        `mapstructure:"image_cache_dir"`
	CacheMaxAge  time.Duration `mapstructure:"image_cache_max_age"`
	CacheMaxSize int64         `mapstructure:"image_cache_max_size"`

	// CSV processing configuration
	CSVBatchSize int `mapstructure:"csv_batch_size"`

	// MinIO bucket configuration
	MinIOBucket string `mapstructure:"minio_bucket" validate:"required"`
}

// LoadConfig loads product service configuration
func LoadConfig() (*Config, error) {
	return config.LoadConfig[Config]("product")
}

// Validate performs product service specific validation
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return errors.New("database URL is required")
	}
	if c.ServicePort == "" {
		return errors.New("service port is required")
	}
	// CacheDir is optional for catalog service
	if c.CacheMaxAge < 0 {
		return errors.New("cache max age cannot be negative")
	}
	if c.CacheMaxSize < 0 {
		return errors.New("cache max size cannot be negative")
	}
	if c.CSVBatchSize < 0 {
		return errors.New("CSV batch size cannot be negative")
	}
	if c.MinIOBucket == "" {
		return errors.New("MinIO bucket is required")
	}
	return nil
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		CacheMaxAge:  24 * time.Hour,
		CacheMaxSize: 0,
		CSVBatchSize: 100,
		MinIOBucket:  "productimages",
	}
}
