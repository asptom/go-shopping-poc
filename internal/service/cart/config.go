package cart

import (
	"errors"

	"go-shopping-poc/internal/platform/config"
)

type Config struct {
	DatabaseURL string   `mapstructure:"db_url" validate:"required"`
	ServicePort string   `mapstructure:"cart_service_port" validate:"required"`
	WriteTopic  string   `mapstructure:"cart_write_topic" validate:"required"`
	ReadTopics  []string `mapstructure:"cart_read_topics"`
	Group       string   `mapstructure:"cart_group"`
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

	return nil
}
