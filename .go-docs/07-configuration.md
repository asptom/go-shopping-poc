# Configuration

This document describes configuration patterns used in the project, including loading from environment variables, validation, and service-specific configuration.

## Overview

The project uses a **configuration-as-code** approach with:
- Environment variables as the primary source
- Viper for loading and unmarshaling
- Struct tags for mapping
- Validation on startup

## Configuration Loading

### Platform-Level Loader

The platform layer provides generic configuration loading:

```go
// internal/platform/config/loader.go

package config

import (
    "fmt"
    "os"
    "strings"
    
    "github.com/spf13/viper"
)

// ConfigLoader provides generic configuration loading
type ConfigLoader interface {
    Load(config interface{}) error
    LoadFromEnv(config interface{}) error
    Validate(config interface{}) error
}

// ViperLoader implements ConfigLoader using Viper
type ViperLoader struct {
    viper *viper.Viper
}

func NewViperLoader() *ViperLoader {
    v := viper.New()
    v.SetConfigType("env")
    v.AutomaticEnv()
    return &ViperLoader{viper: v}
}
```

### Generic Load Function

```go
// LoadConfig loads configuration from environment variables
func LoadConfig[T any](serviceName string) (*T, error) {
    loader := NewViperLoader()
    
    // Load all environment variables into viper
    for _, env := range os.Environ() {
        if idx := strings.Index(env, "="); idx > 0 {
            key := env[:idx]
            value := env[idx+1:]
            loader.viper.Set(key, value)
        }
    }
    
    config := new(T)
    
    // Unmarshal into struct
    if err := loader.viper.Unmarshal(config); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }
    
    // Validate configuration
    if err := loader.Validate(config); err != nil {
        return nil, fmt.Errorf("config validation failed: %w", err)
    }
    
    return config, nil
}
```

**Key features:**
- Generic type parameter (Go 1.18+)
- Automatic environment variable loading
- Struct unmarshaling with mapstructure tags
- Built-in validation

**Reference:** `internal/platform/config/loader.go`

## Service-Specific Configuration

### Configuration Struct

Define configuration structs with `mapstructure` tags:

```go
// internal/service/customer/config.go

package customer

import (
    "errors"
    
    "go-shopping-poc/internal/platform/config"
)

// Config defines customer service configuration
type Config struct {
    // Database
    DatabaseURL  string `mapstructure:"db_url" validate:"required"`
    
    // Service
    ServicePort  string `mapstructure:"customer_service_port" validate:"required"`
    
    // Events
    WriteTopic   string `mapstructure:"customer_write_topic" validate:"required"`
    Group        string `mapstructure:"customer_group"`
}

// LoadConfig loads customer service configuration
func LoadConfig() (*Config, error) {
    return config.LoadConfig[Config]("customer")
}

// Validate performs customer service specific validation
func (c *Config) Validate() error {
    if c.DatabaseURL == "" {
        return errors.New("database URL is required")
    }
    if c.ServicePort == "" {
        return errors.New("service port is required")
    }
    if c.WriteTopic == "" {
        return errors.New("write topic is required")
    }
    return nil
}
```

**Key patterns:**
1. Use `mapstructure` tags to map env vars to struct fields
2. Provide `LoadConfig()` function in service package
3. Implement `Validate()` method for custom validation
4. Group related fields with comments

**Reference:** `internal/service/customer/config.go`

### Configuration Usage in Main

```go
// cmd/customer/main.go

func main() {
    // Load service configuration
    cfg, err := customer.LoadConfig()
    if err != nil {
        log.Fatalf("Customer: Failed to load config: %v", err)
    }
    
    // Use configuration
    dbProvider, err := database.NewDatabaseProvider(cfg.DatabaseURL)
    // ...
}
```

## Environment Variable Naming

### Naming Convention

Use descriptive, service-scoped environment variable names:

```bash
# Global/platform
DB_URL=postgresql://...
KAFKA_BROKERS=localhost:9092

# Service-specific
CUSTOMER_SERVICE_PORT=8080
CUSTOMER_WRITE_TOPIC=customer-events
CUSTOMER_GROUP=customer-service

PRODUCT_SERVICE_PORT=8081
PRODUCT_WRITE_TOPIC=product-events
```

### Mapping Examples

| Environment Variable | Struct Field | Tag |
|---------------------|--------------|-----|
| `DB_URL` | `DatabaseURL` | `mapstructure:"db_url"` |
| `CUSTOMER_SERVICE_PORT` | `ServicePort` | `mapstructure:"customer_service_port"` |
| `KAFKA_BROKERS` | `Brokers` | `mapstructure:"kafka_brokers"` |

## Advanced Configuration

### Nested Configuration

For complex configurations, use nested structs:

```go
type Config struct {
    Database DatabaseConfig `mapstructure:"database"`
    Kafka    KafkaConfig    `mapstructure:"kafka"`
    Cache    CacheConfig    `mapstructure:"cache"`
}

type DatabaseConfig struct {
    URL         string `mapstructure:"url"`
    MaxConns    int    `mapstructure:"max_connections"`
    IdleTimeout int    `mapstructure:"idle_timeout"`
}

// Environment variables:
// DATABASE_URL=postgresql://...
// DATABASE_MAX_CONNECTIONS=20
// DATABASE_IDLE_TIMEOUT=300
```

### Default Values

Set defaults in the loader:

```go
func LoadConfigWithDefaults[T any](serviceName string) (*T, error) {
    loader := NewViperLoader()
    
    // Set defaults
    loader.viper.SetDefault("max_connections", 10)
    loader.viper.SetDefault("idle_timeout", 300)
    loader.viper.SetDefault("log_level", "info")
    
    // ... rest of loading logic
}
```

### Slices and Maps

Configure complex types using environment variables:

```go
type Config struct {
    // Comma-separated list
    KafkaBrokers []string `mapstructure:"kafka_brokers"`
    
    // JSON-encoded map
    Labels map[string]string `mapstructure:"labels"`
}
```

Environment variables:
```bash
KAFKA_BROKERS=broker1:9092,broker2:9092,broker3:9092
LABELS='{"env":"production","region":"us-east"}'
```

## Validation Patterns

### Built-in Validation

Use struct tags with a validation library:

```go
import "github.com/go-playground/validator/v10"

type Config struct {
    DatabaseURL string `mapstructure:"db_url" validate:"required,url"`
    ServicePort string `mapstructure:"port" validate:"required,numeric"`
    MaxRetries  int    `mapstructure:"max_retries" validate:"min=0,max=10"`
}

func (l *ViperLoader) Validate(config interface{}) error {
    validate := validator.New()
    return validate.Struct(config)
}
```

### Custom Validation

Implement service-specific validation:

```go
func (c *ProductConfig) Validate() error {
    // Validate database URL format
    if !strings.HasPrefix(c.DatabaseURL, "postgresql://") {
        return fmt.Errorf("invalid database URL scheme: %s", c.DatabaseURL)
    }
    
    // Validate port range
    port, err := strconv.Atoi(c.ServicePort)
    if err != nil || port < 1024 || port > 65535 {
        return fmt.Errorf("invalid port: %s", c.ServicePort)
    }
    
    // Validate topic naming
    if !strings.HasSuffix(c.WriteTopic, "-events") {
        return fmt.Errorf("topic must end with '-events': %s", c.WriteTopic)
    }
    
    return nil
}
```

### Validation on Startup

Fail fast on invalid configuration:

```go
func main() {
    cfg, err := customer.LoadConfig()
    if err != nil {
        log.Fatalf("[FATAL] Invalid configuration: %v", err)
    }
    
    log.Printf("[INFO] Configuration loaded successfully")
    // Continue with validated config
}
```

## Configuration Reloading (Advanced)

### Watching for Changes

For configurations that can change at runtime:

```go
type ReloadableConfig struct {
    viper *viper.Viper
    config atomic.Value  // atomic for thread safety
}

func (rc *ReloadableConfig) Watch() {
    rc.viper.WatchConfig()
    rc.viper.OnConfigChange(func(e fsnotify.Event) {
        var cfg Config
        if err := rc.viper.Unmarshal(&cfg); err == nil {
            rc.config.Store(&cfg)
            log.Printf("[INFO] Configuration reloaded")
        }
    })
}

func (rc *ReloadableConfig) Get() *Config {
    return rc.config.Load().(*Config)
}
```

**Note:** Use sparingly. Most configurations should be immutable after startup.

## Best Practices

### DO:
- ✅ Use environment variables for all configuration
- ✅ Group related configuration in structs
- ✅ Use `mapstructure` tags for mapping
- ✅ Validate configuration on startup
- ✅ Provide defaults for optional values
- ✅ Use descriptive, service-scoped names
- ✅ Fail fast on invalid configuration
- ✅ Document required vs optional variables

### DON'T:
- ❌ Use configuration files in production (use env vars)
- ❌ Hardcode configuration values
- ❌ Allow invalid configurations to proceed
- ❌ Use global configuration variables
- ❌ Mix configuration with business logic
- ❌ Store secrets in plain text configs

## Migration Guide

### Adding a New Configuration Value

1. Add field to service Config struct with `mapstructure` tag
2. Update validation if required
3. Add default value if optional
4. Update deployment configuration (K8s ConfigMap)
5. Document in README

### Creating a New Service Configuration

1. Create `internal/service/{domain}/config.go`
2. Define Config struct with mapstructure tags
3. Implement `LoadConfig()` function
4. Add `Validate()` method
5. Call in `cmd/{domain}/main.go`
6. Add tests in `config_test.go`

## Testing Configuration

### Unit Tests

```go
func TestConfig_Validate(t *testing.T) {
    tests := []struct {
        name    string
        config  Config
        wantErr bool
    }{
        {
            name: "valid config",
            config: Config{
                DatabaseURL: "postgresql://localhost/test",
                ServicePort: "8080",
                WriteTopic:  "test-events",
            },
            wantErr: false,
        },
        {
            name: "missing database URL",
            config: Config{
                ServicePort: "8080",
                WriteTopic:  "test-events",
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.config.Validate()
            if (err != nil) != tt.wantErr {
                t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

**Reference:** `internal/service/customer/config_test.go`

## Common Configuration Errors

### Issue: Environment variable not mapped
**Cause:** Missing or incorrect `mapstructure` tag
**Fix:** Ensure tag matches environment variable name (snake_case)

### Issue: Validation fails silently
**Cause:** Not calling `Validate()` after loading
**Fix:** Ensure loader calls validation, or call explicitly

### Issue: Configuration changes not reflected
**Cause:** Caching or not reloading
**Fix:** Restart service (configs should be immutable) or implement watching
