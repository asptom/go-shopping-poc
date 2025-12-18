package customer

import (
	"errors"
	"time"

	"go-shopping-poc/internal/platform/config"
)

// Config defines customer service configuration
type Config struct {
	// Database configuration
	DatabaseURL string `mapstructure:"PSQL_CUSTOMER_DB_URL" validate:"required"`

	// HTTP server configuration
	ServicePort string `mapstructure:"CUSTOMER_SERVICE_PORT" validate:"required"`

	// Kafka configuration
	WriteTopic     string        `mapstructure:"CUSTOMER_WRITE_TOPIC" validate:"required"`
	ReadTopics     []string      `mapstructure:"CUSTOMER_READ_TOPICS"`
	Group          string        `mapstructure:"CUSTOMER_GROUP"`
	OutboxInterval time.Duration `mapstructure:"CUSTOMER_OUTBOX_PROCESSING_INTERVAL" validate:"required"`
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
	if c.OutboxInterval <= 0 {
		return errors.New("outbox interval must be positive")
	}
	return nil
}
