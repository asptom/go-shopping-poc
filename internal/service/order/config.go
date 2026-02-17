package order

import (
	"errors"

	"go-shopping-poc/internal/platform/config"
)

type Config struct {
	DatabaseURL string   `mapstructure:"db_url" validate:"required"`
	ServicePort string   `mapstructure:"order_service_port" validate:"required"`
	WriteTopic  string   `mapstructure:"order_write_topic" validate:"required"`
	ReadTopics  []string `mapstructure:"order_read_topics"`
	Group       string   `mapstructure:"order_group"`
}

func LoadConfig() (*Config, error) {
	return config.LoadConfig[Config]("order")
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
	return nil
}
