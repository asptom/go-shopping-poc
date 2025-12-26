package config

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// ConfigLoader provides generic configuration loading capabilities
type ConfigLoader interface {
	Load(config interface{}) error
	LoadFromFile(filename string, config interface{}) error
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

// LoadFromFile loads configuration from file into viper
func (l *ViperLoader) LoadFromFile(filename string, config interface{}) error {
	fmt.Printf("[DEBUG] Checking config file: %s\n", filename)

	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("[DEBUG] Config file does not exist: %s\n", filename)
			// File doesn't exist, skip silently
			return nil
		}
		fmt.Printf("[DEBUG] Failed to open config file %s: %v\n", filename, err)
		return fmt.Errorf("failed to open config file %s: %w", filename, err)
	}
	defer func() { _ = file.Close() }()

	fmt.Printf("[DEBUG] Successfully opened config file: %s\n", filename)

	// Parse simple KEY=VALUE format
	content, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read config file %s: %w", filename, err)
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			value := strings.TrimSpace(line[idx+1:])

			// Strip surrounding quotes
			value = strings.Trim(value, "\"")
			value = strings.Trim(value, "'")

			// Expand environment variables in the value
			value = os.ExpandEnv(value)

			fmt.Printf("[DEBUG] Loaded from %s: %s = %s\n", filename, key, value)
			l.viper.Set(key, value)
		}
	}

	return nil
}

// applyEnvOverrides applies environment variable overrides
func (l *ViperLoader) applyEnvOverrides(config interface{}) error {
	// For now, just re-unmarshal to pick up any env vars
	return l.viper.Unmarshal(config)
}

// LoadFromEnv loads configuration from environment variables
func (l *ViperLoader) LoadFromEnv(config interface{}) error {

	err := l.viper.Unmarshal(config)
	if err != nil {
		return err
	}

	return nil
}

// Load is a convenience method that combines file and env loading
func (l *ViperLoader) Load(config interface{}) error {
	// This would be implemented if we need a single method
	// For now, we use LoadFromFile + LoadFromEnv separately
	return fmt.Errorf("use LoadFromFile and LoadFromEnv separately")
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
