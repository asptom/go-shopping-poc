package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// ConfigLoader provides generic configuration loading capabilities
type ConfigLoader interface {
	Load(config interface{}) error
	LoadFromEnv(config interface{}) error
	Validate(config interface{}) error
}

// Global cache removed - no longer loading shared configuration files

// ViperLoader implements ConfigLoader using Viper
type ViperLoader struct {
	viper *viper.Viper
}

// Global config cache removed - no longer loading shared config files

// Removed loadConfigFileToMap - no longer needed without global caching

// NewViperLoader creates a new Viper-based loader
func NewViperLoader() *ViperLoader {
	v := viper.New()
	v.SetConfigType("env")
	v.AutomaticEnv()
	// Remove replacer for now to test
	// v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	return &ViperLoader{viper: v}
}

// LoadFromEnv loads configuration from environment variables
func (l *ViperLoader) LoadFromEnv(config interface{}) error {

	err := l.viper.Unmarshal(config)
	if err != nil {
		return err
	}

	return nil
}

// Validate performs comprehensive validation
func (l *ViperLoader) Validate(config interface{}) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// Use go-playground/validator for struct tag validation
	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return fmt.Errorf("struct validation failed: %w", err)
	}

	// Domain-specific validation
	if validator, ok := config.(interface{ Validate() error }); ok {
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("domain validation failed: %w", err)
		}
	}

	return nil
}

// Global config cache removed - no longer needed

// LoadConfig loads configuration from environment variables only
func LoadConfig[T any](serviceName string) (*T, error) {
	fmt.Printf("[DEBUG] Loading config for service: %s\n", serviceName)

	loader := NewViperLoader()

	// Load environment variables into viper to support lowercase mapstructure tags
	for _, env := range os.Environ() {
		if idx := strings.Index(env, "="); idx > 0 {
			key := env[:idx]
			value := env[idx+1:]
			loader.viper.Set(key, value)
		}
	}

	// Unmarshal configuration from environment variables
	config := new(T)
	if err := loader.viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate and return
	if err := loader.Validate(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}
