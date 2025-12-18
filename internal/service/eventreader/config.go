package eventreader

import (
	"errors"
	"fmt"
)

// Config defines eventreader service configuration
type Config struct {
	// Kafka configuration
	WriteTopic string   `mapstructure:"event_reader_write_topic" validate:"required"`
	ReadTopics []string `mapstructure:"event_reader_read_topics" validate:"required,min=1"`
	Group      string   `mapstructure:"event_reader_group" validate:"required"`
}

// LoadConfig loads eventreader service configuration
func LoadConfig() (*Config, error) {
	// TEMP: Manual config loading for testing
	cfg := &Config{
		WriteTopic: "processed-events",
		ReadTopics: []string{"customer-events"},
		Group:      "event-reader-group",
	}
	return cfg, nil
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
