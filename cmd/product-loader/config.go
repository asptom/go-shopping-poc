package main

import (
	"errors"
	"time"

	"go-shopping-poc/internal/platform/config"
)

// Config defines product loader configuration
type Config struct {
	// Database configuration
	DatabaseURL string `mapstructure:"psql_product_db_url" validate:"required"`

	// HTTP server configuration (not used by loader)
	ServicePort string `mapstructure:"product_service_port"`

	// Image processing configuration
	CacheDir     string        `mapstructure:"image_cache_dir" validate:"required"`
	CacheMaxAge  time.Duration `mapstructure:"image_cache_max_age"`
	CacheMaxSize int64         `mapstructure:"image_cache_max_size"`

	// CSV processing configuration
	CSVBatchSize int    `mapstructure:"csv_batch_size"`
	CSVPath      string `mapstructure:"csv_path" validate:"required"`

	// MinIO configuration
	MinIOBucket string `mapstructure:"minio_bucket" validate:"required"`

	// Processing configuration
	Concurrency  int    `mapstructure:"concurrency"`
	LogLevel     string `mapstructure:"log_level"`
	ResetOnStart bool   `mapstructure:"reset_on_start"`
}

// LoadConfig loads product loader configuration
func LoadConfig() (*Config, error) {
	return config.LoadConfig[Config]("product-loader")
}

// Validate performs product loader specific validation
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return errors.New("database URL is required")
	}
	if c.CacheDir == "" {
		return errors.New("cache directory is required")
	}
	if c.CacheMaxAge < 0 {
		return errors.New("cache max age cannot be negative")
	}
	if c.CacheMaxSize < 0 {
		return errors.New("cache max size cannot be negative")
	}
	if c.CSVBatchSize < 0 {
		return errors.New("CSV batch size cannot be negative")
	}
	if c.CSVPath == "" {
		return errors.New("CSV path is required")
	}
	if c.Concurrency < 0 {
		return errors.New("concurrency cannot be negative")
	}

	return nil
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		CacheMaxAge:  24 * time.Hour,                                      // 24 hours
		CacheMaxSize: 0,                                                   // unlimited
		CSVBatchSize: 100,                                                 // process 100 products at a time
		CSVPath:      "./resources/product-loader/poc-products-short.csv", // default CSV file path
		Concurrency:  8,                                                   // 8 concurrent downloads
		LogLevel:     "info",
		ResetOnStart: false,
		MinIOBucket:  "productimages", // default bucket for product images
	}
}
