package kafka

import (
	"fmt"
)

// Config defines shared Kafka configuration
type Config struct {
	Brokers []string `mapstructure:"KAFKA_BROKERS" validate:"required,min=1"`
	Topic   string   `mapstructure:"kafka.topic" default:"events"`
	GroupID string   `mapstructure:"kafka.group_id" default:"default-group"`
}

// Validate performs Kafka-specific validation
func (c *Config) Validate() error {
	if len(c.Brokers) == 0 {
		return fmt.Errorf("at least one kafka broker is required")
	}
	return nil
}
