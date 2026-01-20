package customer

import (
	"errors"

	"go-shopping-poc/internal/platform/config"
)

// Config defines customer service configuration
type Config struct {
	// Database configuration
	DatabaseURL string `mapstructure:"db_url" validate:"required"`

	// HTTP server configuration
	ServicePort string `mapstructure:"customer_service_port" validate:"required"`

	// Kafka configuration
	WriteTopic string `mapstructure:"customer_write_topic" validate:"required"`
	Group      string `mapstructure:"customer_group"`
}

// LoadConfig loads customer service configuration
func LoadConfig() (*Config, error) {
	return config.LoadConfig[Config]("customer")
}

// Validate performs customer service specific validation
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return errors.New("database URL is required")
	}
	if c.ServicePort == "" {
		return errors.New("service port is required")
	}
	if c.WriteTopic == "" {
		return errors.New("write topic is required")
	}
	return nil
}
