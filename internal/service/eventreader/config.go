package eventreader

import (
	"errors"

	"go-shopping-poc/internal/platform/config"
)

// Config defines eventreader service configuration
type Config struct {
	ReadTopics []string `mapstructure:"EVENT_READER_READ_TOPICS"`
	WriteTopic string   `mapstructure:"EVENT_READER_WRITE_TOPIC" validate:"required"`
	Group      string   `mapstructure:"EVENT_READER_GROUP"`
}

// LoadConfig loads eventreader service configuration
func LoadConfig() (*Config, error) {
	return config.LoadConfig[Config]("eventreader")
}

// Validate performs eventreader service specific validation
func (c *Config) Validate() error {
	if c.WriteTopic == "" {
		return errors.New("write topic is required")
	}
	// ReadTopics and Group are optional - defaults will be provided in main.go
	return nil
}
