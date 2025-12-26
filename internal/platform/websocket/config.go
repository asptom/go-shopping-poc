package websocket

import (
	"fmt"
	"time"

	"go-shopping-poc/internal/platform/config"
)

// Config defines shared WebSocket configuration
type Config struct {
	URL            string        `mapstructure:"WEBSOCKET_URL" validate:"required"`
	Timeout        time.Duration `mapstructure:"WEBSOCKET_TIMEOUT" default:"30s"`
	ReadBuffer     int           `mapstructure:"WEBSOCKET_READ_BUFFER" default:"1024"`
	WriteBuffer    int           `mapstructure:"WEBSOCKET_WRITE_BUFFER" default:"1024"`
	Port           string        `mapstructure:"WEBSOCKET_PORT" default:":8080"`
	Path           string        `mapstructure:"WEBSOCKET_PATH" default:"/ws"`
	AllowedOrigins []string      `mapstructure:"WEBSOCKET_ALLOWED_ORIGINS" validate:"required,min=1"`
}

// LoadConfig loads shared WebSocket configuration
func LoadConfig() (*Config, error) {
	return config.LoadConfig[Config]("platform-websocket")
}

// Validate performs WebSocket-specific validation and sets defaults
func (c *Config) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("WebSocket URL is required")
	}
	if c.Timeout < 0 {
		return fmt.Errorf("timeout must not be negative")
	}
	if c.Timeout == 0 {
		c.Timeout = 30 * time.Second // Set default
	}
	if c.ReadBuffer < 0 {
		return fmt.Errorf("read buffer size must not be negative")
	}
	if c.ReadBuffer == 0 {
		c.ReadBuffer = 1024 // Set default
	}
	if c.WriteBuffer < 0 {
		return fmt.Errorf("write buffer size must not be negative")
	}
	if c.WriteBuffer == 0 {
		c.WriteBuffer = 1024 // Set default
	}
	if c.Port == "" {
		c.Port = ":8080" // Set default
	}
	if c.Path == "" {
		c.Path = "/ws" // Set default
	}
	return nil
}
