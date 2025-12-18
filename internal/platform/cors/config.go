package cors

import (
	"fmt"

	"go-shopping-poc/internal/platform/config"
)

// Config defines shared CORS configuration
type Config struct {
	AllowedOrigins   []string `mapstructure:"CORS_ALLOWED_ORIGINS" validate:"required,min=1"`
	AllowedMethods   []string `mapstructure:"CORS_ALLOWED_METHODS" validate:"required,min=1"`
	AllowedHeaders   []string `mapstructure:"CORS_ALLOWED_HEADERS" validate:"required,min=1"`
	AllowCredentials bool     `mapstructure:"CORS_ALLOW_CREDENTIALS" default:"true"`
	MaxAge           string   `mapstructure:"CORS_MAX_AGE" default:"3600"`
}

// LoadConfig loads shared CORS configuration
func LoadConfig() (*Config, error) {
	return config.LoadConfig[Config]("platform-cors")
}

// Validate performs CORS-specific validation
func (c *Config) Validate() error {
	if len(c.AllowedOrigins) == 0 {
		return fmt.Errorf("at least one allowed origin is required")
	}
	if len(c.AllowedMethods) == 0 {
		return fmt.Errorf("at least one allowed method is required")
	}
	if len(c.AllowedHeaders) == 0 {
		return fmt.Errorf("at least one allowed header is required")
	}
	return nil
}
