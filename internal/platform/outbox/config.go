package outbox

import (
	"fmt"
	"time"

	"go-shopping-poc/internal/platform/config"
)

// Config defines shared outbox configuration
type Config struct {
	BatchSize       int           `mapstructure:"OUTBOX_BATCH_SIZE" default:"10"`
	DeleteBatchSize int           `mapstructure:"OUTBOX_DELETE_BATCH_SIZE" default:"10"`
	ProcessInterval time.Duration `mapstructure:"OUTBOX_PROCESS_INTERVAL" default:"5s"`
	MaxRetries      int           `mapstructure:"OUTBOX_MAX_RETRIES" default:"3"`
}

// LoadConfig loads shared outbox configuration
func LoadConfig() (*Config, error) {
	return config.LoadConfig[Config]("platform-outbox")
}

// Validate performs outbox-specific validation
func (c *Config) Validate() error {
	if c.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}
	if c.DeleteBatchSize <= 0 {
		return fmt.Errorf("delete batch size must be positive")
	}
	if c.ProcessInterval <= 0 {
		return fmt.Errorf("process interval must be positive")
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}
	return nil
}
