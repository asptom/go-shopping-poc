package eventreader

import (
	"errors"
	"fmt"

	"go-shopping-poc/internal/platform/config"
)

// Config defines eventreader service configuration
type Config struct {
	// Kafka configuration
	WriteTopic string   `mapstructure:"EVENT_READER_WRITE_TOPIC" validate:"required"`
	ReadTopics []string `mapstructure:"EVENT_READER_READ_TOPICS" validate:"required,min=1"`
	Group      string   `mapstructure:"EVENT_READER_GROUP" validate:"required"`
}

// LoadConfig loads eventreader service configuration
func LoadConfig() (*Config, error) {
	return config.LoadConfig[Config]("eventreader")
}

// Validate performs eventreader service specific validation
func (c *Config) Validate() error {
	// Debug output
	fmt.Printf("DEBUG: Config values - WriteTopic: %s, ReadTopics: %v, Group: %s\n",
		c.WriteTopic, c.ReadTopics, c.Group)

	if c.WriteTopic == "" {
		return errors.New("write topic is required")
	}
	if len(c.ReadTopics) == 0 {
		return errors.New("at least one read topic is required")
	}
	if c.Group == "" {
		return errors.New("consumer group is required")
	}
	return nil
}
