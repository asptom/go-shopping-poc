package product

import (
	"errors"
	"time"

	"go-shopping-poc/internal/platform/config"
)

// AdminConfig defines product admin service configuration
type AdminConfig struct {
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

	// Kafka configuration
	WriteTopic string `mapstructure:"product_write_topic" validate:"required"`
	Group      string `mapstructure:"product_group"`

	// Keycloak configuration
	KeycloakIssuer       string `mapstructure:"keycloak_issuer" validate:"required"`
	KeycloakJWKSURL      string `mapstructure:"keycloak_jwks_url" validate:"required"`
	KeycloakClientSecret string `mapstructure:"keycloak_client_secret"`
}

// LoadAdminConfig loads product admin service configuration
func LoadAdminConfig() (*AdminConfig, error) {
	return config.LoadConfig[AdminConfig]("product-admin")
}

// Validate performs admin service specific validation
func (c *AdminConfig) Validate() error {
	if c.DatabaseURL == "" {
		return errors.New("database URL is required")
	}
	if c.ServicePort == "" {
		return errors.New("service port is required")
	}
	if c.MinIOBucket == "" {
		return errors.New("MinIO bucket is required")
	}
	if c.WriteTopic == "" {
		return errors.New("write topic is required")
	}
	if c.KeycloakIssuer == "" {
		return errors.New("keycloak issuer is required")
	}
	if c.KeycloakJWKSURL == "" {
		return errors.New("keycloak JWKS URL is required")
	}
	return nil
}
