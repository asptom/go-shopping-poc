# slog Migration Plan v4 - Final Production-Ready Implementation with Logger Provider

## Overview

This document provides the final comprehensive plan for migrating from the legacy `log` package to the standard library `log/slog` package across the Go Shopping POC codebase. This v4 plan incorporates all feedback and focuses on **production-ready structured logging** while minimizing complexity and risk through a **Logger Provider pattern**.

## Goals

1. Replace all `log.Printf()` calls with structured `slog` logging
2. Add request correlation IDs for distributed tracing
3. Enable JSON output for Kubernetes environments
4. Maintain backward compatibility during transition
5. Eliminate boilerplate configuration with Logger Provider
6. Minimize complexity while delivering production benefits

## Key Design Decisions

### 1. **Logger Provider Pattern (Eliminates Boilerplate)**

**Problem**: Each service has ~20 lines of logger configuration boilerplate in main.go.

**Solution**: Create a `LoggerProvider` that follows the exact same pattern as other providers (DatabaseProvider, EventBusProvider, etc.).

```go
// internal/platform/providers/providers.go
// Add to existing file
type LoggerProvider interface {
    // GetLogger returns a configured slog.Logger instance
    GetLogger() *slog.Logger
}
```

**Benefits**:
- Eliminates ~20 lines of boilerplate from each main.go
- Follows existing provider pattern (consistent with clean architecture)
- Centralized configuration in one place
- Easy to maintain and test
- Single responsibility principle

### 2. **Simplified Service-Level Injection (NOT Provider Changes)**

**Problem with Provider Approach**: Adding logger injection to platform providers (like DatabaseProvider, EventBusProvider) would require changing interface signatures, breaking all implementations and consumers.

**Solution**: Inject logger through service constructors only. Services already accept infrastructure dependencies - we simply add the logger as another dependency.

```go
// Service constructor pattern (NO INTERFACE CHANGES)
type CartService struct {
    *service.EventServiceBase
    logger *slog.Logger  // Logger stored in service, not provider
    repo   CartRepository
    config *Config
}

func NewCartService(
    logger *slog.Logger,  // NEW: Logger injected here
    infrastructure *CartInfrastructure, 
    config *Config,
) *CartService {
    return &CartService{
        EventServiceBase: service.NewEventServiceBase("cart", infrastructure.EventBus),
        logger:           logger.With("service", "cart"),  // Service-specific logger
        // ... rest of initialization
    }
}
```

**Why This Works**:
- No changes to provider interfaces (no breaking changes)
- Services control their own logger instance
- Loggers can be hierarchical (base logger + service-specific attributes)
- Follows existing dependency injection patterns

### 3. **Package-Level Logger for Platform Code (Library Convention)**

**Problem**: Platform code (database, event bus, outbox) doesn't have services injected into it.

**Solution**: Create package-level loggers with pre-populated attributes. This is the **recommended pattern** from the slog documentation for library code.

```go
// internal/platform/database/postgres.go
var logger *slog.Logger

func init() {
    // Set base logger - can be overridden by main.go
    logger = slog.Default().With("database", "postgresql")
}

// NewPostgreSQLClient now has optional logger parameter
func NewPostgreSQLClient(dsn string, config ConnectionConfig, opts ...LoggerOption) (*PostgreSQLClient, error) {
    client := &PostgreSQLClient{
        // ... existing fields
    }
    
    // Apply logger options
    for _, opt := range opts {
        opt(client)
    }
    
    // Set logger if not already set
    if client.logger == nil {
        client.logger = logger.With("client", fmt.Sprintf("%p", client))
    }
    
    return client, nil
}
```

**Benefits**:
- No constructor signature changes required (backward compatible)
- Services can override platform loggers if needed
- Follows Go library conventions (see `database/sql`, `net/http`)

### 4. **Context-Aware Logging via Helper Functions**

**Problem**: `slog` doesn't automatically extract trace IDs from context.

**Solution**: Create helper functions that combine slog with context extraction.

```go
// internal/platform/logging/context.go
package logging

import (
    "context"
    "log/slog"
    "github.com/google/uuid"
)

type contextKey string

const (
    RequestIDKey contextKey = "request_id"
    TraceIDKey   contextKey = "trace_id"
)

// FromContext extracts logger with context attributes
func FromContext(ctx context.Context, base *slog.Logger) *slog.Logger {
    attrs := []slog.Attr{}
    
    if reqID := ctx.Value(RequestIDKey); reqID != nil {
        attrs = append(attrs, slog.String("request_id", reqID.(string)))
    }
    
    if traceID := ctx.Value(TraceIDKey); traceID != nil {
        attrs = append(attrs, slog.String("trace_id", traceID.(string)))
    }
    
    return base.WithAttrs(attrs...)
}

// GenerateRequestID creates a new correlation ID
func GenerateRequestID() string {
    return uuid.New().String()
}
```

**Usage**:
```go
func (s *CartService) GetCart(ctx context.Context, cartID string) (*Cart, error) {
    // Automatically includes request_id and trace_id from context
    logger := logging.FromContext(ctx, s.logger)
    logger.Debug("Fetching cart", "cart_id", cartID)
    
    // ... rest of implementation
}
```

## Implementation Phases

### Phase 1: Platform Logger Infrastructure (Days 1-2)

**Files Modified**: ~21 files

#### 1.1 Create Logging Package

```
internal/platform/logging/
├── config.go         # Configuration loading and log level parsing
├── logger.go         # Core logger functions and types
├── context.go        # Context extraction helpers
├── options.go        # Logger option patterns
├── helpers.go        # Utility functions
└── provider.go       # Logger provider implementation (NEW)
```

**provider.go**:

```go
package logging

import (
    "fmt"
    "log/slog"
    "os"
    "time"
)

type LoggerProviderImpl struct {
    logger *slog.Logger
}

// NewLoggerProvider creates a new logger provider with the given service name.
// It loads logging configuration, creates a slog.Logger,
// and configures it based on environment settings.
//
// Parameters:
//   - serviceName: The name of the service (e.g., "cart", "order", "product")
//
// Returns:
//   - A configured LoggerProvider that provides structured logging
//   - An error if configuration loading or logger creation fails
//
// Usage:
//
//	provider, err := logging.NewLoggerProvider("cart")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	logger := provider.GetLogger()
func NewLoggerProvider(serviceName string) (*LoggerProviderImpl, error) {
    if serviceName == "" {
        return nil, fmt.Errorf("service name is required")
    }

    // Read log level from environment
    logLevel := os.Getenv("LOG_LEVEL")
    if logLevel == "" {
        logLevel = "info"
    }

    level := new(slog.LevelVar)
    level.Set(slog.LevelInfo)
    level.Set(logLevel) // Parse from env

    // Configure handler based on environment
    var handler slog.Handler

    if os.Getenv("ENVIRONMENT") == "development" {
        // Text handler for local development
        handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
            Level: level,
            ReplaceAttr: groupAttrs,
        })
    } else {
        // JSON handler for production
        handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
            Level: level,
            ReplaceAttr: groupAttrs,
        })
    }

    logger := slog.New(handler)

    return &LoggerProviderImpl{
        logger: logger,
    }, nil
}

// GetLogger returns the configured slog.Logger instance.
// The logger is already configured with service identification
// and context support.
//
// Returns:
//   - A slog.Logger interface implementation that can be used for structured logging
//
// Usage:
//
//	logger := provider.GetLogger()
//	logger.Info("Service started", "port", 8080)
func (p *LoggerProviderImpl) GetLogger() *slog.Logger {
    return p.logger
}

// Group common attributes
func groupAttrs(groups []string, a slog.Attr) slog.Attr {
    // Group known attributes under categories
    switch a.Key {
    case "request_id", "trace_id":
        return slog.String("context."+a.Key, a.Value.String())
    }

    return a
}
```

#### 1.2 Update providers.go

Add LoggerProvider interface to `internal/platform/providers/providers.go`:

```go
// LoggerProvider defines the interface for providing structured logging infrastructure.
// Implementations should return a configured slog.Logger that services can use
// for structured logging with service identification and context support.
type LoggerProvider interface {
    // GetLogger returns a configured slog.Logger instance
    GetLogger() *slog.Logger
}
```

#### 1.3 Update Platform Packages

**Pattern for each package** (`database/*.go`, `event/bus/kafka/*.go`, `outbox/*.go`, `sse/*.go`, `websocket/*.go`):

```go
// Add to existing struct
type PostgreSQLClient struct {
    db     *sqlx.DB
    logger *slog.Logger  // NEW
    config ConnectionConfig
}

// NewPostgreSQLClient with optional logger
func NewPostgreSQLClient(dsn string, config ConnectionConfig, opts ...LoggerOption) (*PostgreSQLClient, error) {
    client := &PostgreSQLClient{
        config: config,
    }
    
    // Apply logger options
    for _, opt := range opts {
        client.logger = opt(client.logger)
    }
    
    // Set default if not already set
    if client.logger == nil {
        client.logger = logging.NewPlatform("database").With("client", "postgresql")
    }
    
    // ... rest of initialization
    return client, nil
}
```

**Backward Compatibility**: All existing calls continue to work with default logger.

### Phase 2: Service Base Extension (Days 3-4)

**Files Modified**: ~5 files

#### 2.1 EventServiceBase Enhancement

Add logger field without breaking existing code:

```go
// internal/platform/service/base.go
type EventServiceBase struct {
    *BaseService
    eventBus bus.Bus
    handlers []any
    logger   *slog.Logger  // NEW: Add to existing struct
}

// NewEventServiceBase with optional logger
func NewEventServiceBase(name string, eventBus bus.Bus, opts ...LoggerOption) *EventServiceBase {
    esb := &EventServiceBase{
        BaseService: NewBaseService(name),
        eventBus:    eventBus,
        handlers:    make([]any, 0),
    }
    
    // Apply logger options
    for _, opt := range opts {
        esb.logger = opt(esb.logger)
    }
    
    // Set default
    if esb.logger == nil {
        esb.logger = slog.Default().With("service", name)
    }
    
    return esb
}
```

**Services can still use constructor without logger (backward compatible)**:

```go
// OLD (still works):
service := service.NewEventServiceBase("cart", eventBus)

// NEW (with logger):
service := service.NewEventServiceBase(
    "cart", 
    eventBus,
    logging.WithService("cart"),
)
```

### Phase 3: Services Migration (Days 5-9)

**Files Modified**: ~58 files (all service packages)

#### 3.1 Service Constructor Pattern

Update each service constructor to accept logger:

```go
// internal/service/cart/service.go

func NewCartService(
    logger *slog.Logger,  // NEW: Add logger parameter
    infrastructure *CartInfrastructure, 
    config *Config,
) *CartService {
    repo := NewCartRepository(infrastructure.Database, infrastructure.OutboxWriter)
    
    return &CartService{
        EventServiceBase: service.NewEventServiceBase("cart", infrastructure.EventBus),
        logger:           logger.With("service", "cart", "component", "cart"),  // NEW
        repo:             repo,
        infrastructure:   infrastructure,
        config:           config,
    }
}

// Helper for existing callers
func NewCartServiceDefault(infrastructure *CartInfrastructure, config *Config) *CartService {
    return NewCartService(slog.Default(), infrastructure, config)
}
```

#### 3.2 Replace log.Printf with Logger

Use find and replace pattern (with careful review):

**Before**:
```go
log.Printf("[DEBUG] CartService: Creating cart for customer: %s", customerID)
```

**After**:
```go
s.logger.Debug("Creating cart",
    slog.String("customer_id", customerID),
    slog.String("currency", cart.Currency),
)
```

**Replacement Strategy**:
1. Identify all `log.Printf` calls in service code
2. Replace with appropriate logger method
3. Add related attributes as structured fields
4. Remove log level prefixes (DEBUG, INFO) - let slog handle this

### Phase 4: Event Handler Logging (Days 8-9)

**Files Modified**: ~10 files (event handlers)

Event handlers already have access to service logger:

```go
// internal/service/cart/eventhandlers/*.go

func NewCustomerCreatedHandler(s Service) bus.HandlerFunc[events.CustomerCreated] {
    return func(ctx context.Context, evt *events.CustomerCreated) error {
        // Access service logger through the service
        svcLogger := logging.FromContext(ctx, s.Logger())  // s.Logger() from service
        
        svcLogger.Info("Processing customer created event",
            "customer_id", evt.CustomerID,
            "event_id", evt.EventID,
        )
        
        // ... handler logic
    }
}
```

**Add Logger method to EventServiceBase**:

```go
// internal/platform/service/base.go
func (s *EventServiceBase) Logger() *slog.Logger {
    return s.logger
}
```

### Phase 5: Main.go Configuration (Days 10-11)

**Files Modified**: ~8 files (main.go in each service)

#### 5.1 Create Logger Provider (No Boilerplate)

```go
// cmd/cart/main.go

func main() {
    // Create logger provider
    loggerProvider, err := logging.NewLoggerProvider("cart")
    if err != nil {
        log.Fatalf("Cart: Failed to create logger provider: %v", err)
    }
    logger := loggerProvider.GetLogger()

    // ... database setup, etc.

    cartService := cart.NewCartService(
        logger,  // Pass logger
        cartInfra,
        cartConfig,
    )

    // ... start services
}
```

#### 5.2 No Boilerplate Setup

**Before (with boilerplate)**:
```go
func setupLogger(serviceName string) *slog.Logger {
    // ... 20+ lines of logger setup
}
```

**After (using provider)**:
```go
func main() {
    // Single line logger setup
    loggerProvider, err := logging.NewLoggerProvider("cart")
    if err != nil {
        log.Fatalf("Cart: Failed to create logger provider: %v", err)
    }
    logger := loggerProvider.GetLogger()
    
    // ... rest of code
}
```

### Phase 6: Configuration (Day 12)

**Files Modified**: 1 file

**No code changes needed** - existing config system works with slog. Environment variables control log level:

```yaml
# deploy/k8s/platform/config/platform-configmap-for-services.yaml
data:
  LOG_LEVEL: "info"  # Already exists, works with slog
```

Implement env var parsing in each main.go (see 5.1 above).

## Testing Strategy

### Unit Tests

```go
func TestCartService_CreateCart(t *testing.T) {
    // Create test logger
    logger := slog.New(slog.NewTextHandler(io.Discard, nil))
    
    svc := NewCartService(logger, infra, config)
    
    // ... test logic
}
```

### Integration Tests
- Verify log output format matches expectations
- Verify context attributes are included
- Verify JSON format in production environment

## Risk Mitigation

### Risk 1: Breaking Changes to Constructors
**Mitigation**: Provide `*Default` constructors for backward compatibility during transition.

### Risk 2: Context Attributes Not Extracted
**Mitigation**: Use helper functions (`FromContext`) and implement gradually.

### Risk 3: Missing Error Context
**Mitigation**: Always include error message AND stack trace pattern:

```go
logger.Error("Database query failed",
    slog.String("query", query),
    slog.String("error", err.Error()),
)
```

### Risk 4: Performance Overhead
**Mitigation**: Slog is optimized; use `Debug` level for expensive operations:

```go
if logger.Enabled(ctx, slog.LevelDebug) {
    logger.Debug("Expensive operation",
        slog.String("data", expensiveToGenerate()),
    )
}
```

## Success Metrics

1. All `log.Printf` calls replaced with logger instance calls
2. No breaking changes to public interfaces
3. All logs include service name and request ID
4. JSON format logs in production environments
5. All tests pass after migration
6. No logger configuration boilerplate in main.go files

## Migration Timeline

| Phase | Duration | Files | Risk Level |
|-------|----------|-------|------------|
| Platform Infrastructure | 2 days | 21 | Low |
| Service Base Extension | 1 day | 5 | Low |
| Services Migration | 5 days | 58 | Medium |
| Event Handler Updates | 2 days | 10 | Medium |
| Main.go Configuration | 1 day | 8 | Low |
| Configuration | 1 day | 1 | Low |
| **Total** | **12 days** | **~103** | **Low-Medium** |

## Conclusion

This v4 plan delivers **enhanced logging** while:
- Avoiding breaking changes to provider interfaces
- Maintaining existing service patterns
- Following Go conventions
- Minimizing architectural modifications
- Eliminating logger configuration boilerplate

The key insight: **inject logger through service constructors, not platform providers**. This preserves clean architecture while enabling comprehensive logging.

**Why This Plan Works**:
- Solves the event handler lifecycle problem by using package-level loggers
- Maintains backward compatibility with `*Default` constructors
- Provides structured logging benefits without massive refactoring
- Enables request correlation for distributed tracing
- Minimal risk with gradual migration approach
- Eliminates ~20 lines of boilerplate from each main.go

**Expected Outcome**:
- All services output structured JSON logs by default
- Service name appears on every log entry
- Request IDs enable end-to-end tracing
- Zero remaining `log` package imports
- Production-ready logging with minimal complexity
- No logger configuration boilerplate in main.go files

This is the final, production-ready implementation that balances all requirements while minimizing risk and complexity. The Logger Provider pattern eliminates the boilerplate concern while maintaining clean architecture principles.

**Final Benefits**:
- **No Boilerplate**: Eliminates ~20 lines of logger setup from each main.go
- **Consistent Pattern**: Follows existing provider pattern throughout codebase
- **Single Responsibility**: Logger provider handles all logger configuration
- **Easy Testing**: Can easily mock logger provider in tests
- **Maintainable**: Changes to logging configuration only need to be made in one place
- **Production Ready**: Structured logging with request correlation and JSON output
- **Low Risk**: Gradual migration with backward compatibility

This plan represents the optimal balance of production benefits, minimal complexity, and clean architecture principles.
