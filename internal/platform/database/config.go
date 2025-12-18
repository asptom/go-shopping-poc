package database

import (
	"fmt"

	"go-shopping-poc/internal/platform/config"
)

// Config defines shared database configuration
type Config struct {
	Host            string `mapstructure:"host" validate:"required"`
	Port            int    `mapstructure:"port" validate:"required,min=1,max=65535"`
	Username        string `mapstructure:"username" validate:"required"`
	Password        string `mapstructure:"password" validate:"required"`
	Database        string `mapstructure:"database" validate:"required"`
	SSLMode         string `mapstructure:"ssl_mode" default:"disable"`
	MaxOpenConns    int    `mapstructure:"max_open_conns" default:"25"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns" default:"25"`
	ConnMaxLifetime string `mapstructure:"conn_max_lifetime" default:"5m"`
}

// LoadConfig loads shared database configuration
func LoadConfig() (*Config, error) {
	return config.LoadConfig[Config]("platform-database")
}

// ConnectionString returns the database connection string
func (c *Config) ConnectionString() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.Username, c.Password, c.Host, c.Port, c.Database, c.SSLMode)
}

// Validate performs database-specific validation
func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid database port: %d", c.Port)
	}
	if c.Username == "" {
		return fmt.Errorf("database username is required")
	}
	if c.Database == "" {
		return fmt.Errorf("database name is required")
	}
	return nil
}
