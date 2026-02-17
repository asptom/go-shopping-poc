package cart

import (
	"errors"
	"time"

	"go-shopping-poc/internal/platform/config"
)

type Config struct {
	DatabaseURL string   `mapstructure:"db_url" validate:"required"`
	ServicePort string   `mapstructure:"cart_service_port" validate:"required"`
	WriteTopic  string   `mapstructure:"cart_write_topic" validate:"required"`
	ReadTopics  []string `mapstructure:"cart_read_topics"`
	Group       string   `mapstructure:"cart_group"`

	// Outbox configuration for fast validation events (target: 200ms interval)
	// Using cart-specific env vars to avoid conflict with platform-outbox defaults
	OutboxBatchSize       int           `mapstructure:"cart_outbox_batch_size"`
	OutboxProcessInterval time.Duration `mapstructure:"cart_outbox_process_interval"`
}

func LoadConfig() (*Config, error) {
	return config.LoadConfig[Config]("cart")
}

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
	// Set defaults for outbox if not configured
	if c.OutboxBatchSize <= 0 {
		c.OutboxBatchSize = 10
	}
	if c.OutboxProcessInterval <= 0 {
		c.OutboxProcessInterval = 200 * time.Millisecond
	}
	return nil
}
