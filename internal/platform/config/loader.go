package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// ViperLoader implements ConfigLoader using Viper
type ViperLoader struct {
	viper *viper.Viper
}

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
	// Read .env file and set viper values directly
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, skip silently
			return nil
		}
		return fmt.Errorf("failed to open config file %s: %w", filename, err)
	}
	defer file.Close()

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

			l.viper.Set(key, value)
			// Debug
			if strings.Contains(filename, "eventreader") && key == "EVENT_READER_WRITE_TOPIC" {
				fmt.Printf("DEBUG: Parsed %s = %s from %s\n", key, value, filename)
			}
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
	// Debug before unmarshal
	fmt.Printf("DEBUG: Before unmarshal, viper has EVENT_READER_WRITE_TOPIC: %s\n", l.viper.Get("EVENT_READER_WRITE_TOPIC"))

	err := l.viper.Unmarshal(config)
	if err != nil {
		return err
	}

	// Debug after unmarshal - try to access the config if it's the right type
	if cfg, ok := config.(*interface{}); ok {
		fmt.Printf("DEBUG: After unmarshal, config is: %+v\n", cfg)
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

// LoadConfig loads configuration with automatic source detection
func LoadConfig[T any](serviceName string) (*T, error) {
	loader := NewViperLoader()

	// Load files in order (each call uses the same viper instance)
	files := []string{
		"config/.env.local",                         // 1. Global local overrides
		filepath.Join("config", serviceName+".env"), // 2. Service-specific (higher precedence)
		"config/.env",                               // 3. Global defaults (lowest precedence)
	}

	for _, file := range files {
		if err := loader.LoadFromFile(file, nil); err != nil {
			return nil, fmt.Errorf("failed to load %s: %w", file, err)
		}
	}

	// Debug: check what viper has
	if serviceName == "eventreader" {
		fmt.Printf("DEBUG: Before unmarshal, viper EVENT_READER_WRITE_TOPIC: %s\n", loader.viper.Get("EVENT_READER_WRITE_TOPIC"))
		fmt.Printf("DEBUG: All viper keys: %v\n", loader.viper.AllKeys())
	}

	// Now unmarshal with all file values loaded
	config := new(T)
	if err := loader.viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Debug: check config after unmarshal
	if serviceName == "eventreader" {
		fmt.Printf("DEBUG: After unmarshal, config type: %T\n", config)
	}

	// Apply environment variable overrides
	if err := loader.applyEnvOverrides(config); err != nil {
		return nil, fmt.Errorf("failed to apply env overrides: %w", err)
	}

	// Validate and return
	if err := loader.Validate(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}
