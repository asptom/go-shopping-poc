# slog Improvement Plan

## Overview

This document outlines the plan to fix two issues identified after the slog migration:

1. **Issue 1**: LOG_LEVEL environment variable is not being used
2. **Issue 2**: Platform services (database, event bus, outbox, etc.) lack service identity in their logs

---

## Issue 1: LOG_LEVEL Not Being Used

### Root Cause

The `logging.DefaultLoggerConfig()` function exists and reads LOG_LEVEL from the environment:

```go
// internal/platform/logging/logger.go
func DefaultLoggerConfig(serviceName string) LoggerConfig {
    return LoggerConfig{
        ServiceName: serviceName,
        Level:       getEnv("LOG_LEVEL", "info"),  // Reads LOG_LEVEL
        Format:      getEnv("LOG_FORMAT", "json"),
    }
}
```

However, in each service's `main.go`, it's not being used:

```go
// cmd/cart/main.go - CURRENT (broken)
loggerProvider, err := logging.NewLoggerProvider(logging.LoggerConfig{
    ServiceName: "cart",  // Level and Format are empty strings!
})
```

### Fix

Change all service main.go files to use `DefaultLoggerConfig()`:

```go
// cmd/cart/main.go - FIXED
config := logging.DefaultLoggerConfig("cart")
loggerProvider, err := logging.NewLoggerProvider(config)
```

### Files to Modify

| Service | File |
|---------|------|
| cart | `cmd/cart/main.go` |
| customer | `cmd/customer/main.go` |
| order | `cmd/order/main.go` |
| product | `cmd/product/main.go` |
| product-admin | `cmd/product-admin/main.go` |
| product-loader | `cmd/product-loader/main.go` |
| eventreader | `cmd/eventreader/main.go` |
| websocket | `cmd/websocket/main.go` |

---

## Issue 2: Platform Services Lack Service Identity

### Root Cause

Each platform package has a package-level logger created in `init()` with no service context:

```go
// internal/platform/database/logger.go
func init() {
    logger = slog.New(slog.NewJSONHandler(os.Stderr, nil)).
        With("platform", "database", "component", "postgresql")
    // No "service" attribute!
}
```

When database logs appear in Loki, there's no way to distinguish "cart service database logs" from "order service database logs".

### Solution: Hybrid Approach

1. **Pass logger to platform providers** via functional options
2. **Keep package-level logger.go files** as fallbacks
3. **Platform packages receive logger at initialization** - the logger already carries the service name via `.With("service", "cart")`

### Pattern: Functional Options

This pattern allows optional logger injection while maintaining backward compatibility:

```go
// Example: database/provider.go

type DatabaseProviderImpl struct {
    database Database
    logger   *slog.Logger
}

// Option is a functional option for configuring DatabaseProviderImpl
type Option func(*DatabaseProviderImpl)

func WithLogger(logger *slog.Logger) Option {
    return func(p *DatabaseProviderImpl) {
        p.logger = logger
    }
}

func NewDatabaseProvider(databaseURL string, opts ...Option) (DatabaseProvider, error) {
    p := &DatabaseProviderImpl{}
    
    // Apply all options
    for _, opt := range opts {
        opt(p)
    }
    
    // Fallback to package logger if none provided
    if p.logger == nil {
        p.logger = Logger()
    }
    
    // Use p.logger for all internal logging
    p.logger.Info("DatabaseProvider: Initializing...")
    // ...
}
```

### Usage in main.go

```go
// cmd/cart/main.go
dbProvider, err := database.NewDatabaseProvider(dbURL, 
    database.WithLogger(logger),  // Logger already has service="cart" baked in
)
```

### Why This Works

When the logger is created in main.go:

```go
loggerProvider, _ := logging.NewLoggerProvider(config)  // config has ServiceName: "cart"
logger := loggerProvider.Logger()  // Returns slog.Logger with .With("service", "cart")
```

So when we pass this logger to the database provider:

```go
database.NewDatabaseProvider(dbURL, database.WithLogger(logger))
```

The database provider's internal logs will automatically include `service="cart"` because the logger already has that attribute baked in. No need to call `InitLogger()` or pass service name separately.

---

## Implementation Order

### Phase 1: Fix LOG_LEVEL (Simple)

1. Modify each service's `main.go` to use `logging.DefaultLoggerConfig()`

### Phase 2: Add Logger Options to Platform Providers

This is a breaking change to provider constructors. We need to:

1. Add logger field to each provider implementation struct
2. Add `WithLogger(*slog.Logger) Option` function
3. Modify constructor to accept variadic `...Option`
4. Apply options and set fallback to package logger

### Phase 3: Update main.go Files

Pass the service logger to each platform provider.

---

## Detailed Changes by Platform Package

### 1. Database Provider (`internal/platform/database/provider.go`)

**Current:**
```go
func NewDatabaseProvider(databaseURL string) (DatabaseProvider, error)
```

**After:**
```go
func NewDatabaseProvider(databaseURL string, opts ...Option) (DatabaseProvider, error)

// Where Option is defined in a new file or at the top of provider.go
type Option func(*DatabaseProviderImpl)
```

**Changes needed:**
- Add `logger *slog.Logger` field to `DatabaseProviderImpl`
- Add `WithLogger()` function
- Modify constructor to apply options
- Replace all `logger.Info/Debug/Error` calls with `p.logger.Info/Debug/Error`
- Ensure package logger is used as fallback

### 2. Event Bus Provider (`internal/platform/event/provider.go`)

**Current:**
```go
func NewEventBusProvider(config EventBusConfig) (EventBusProvider, error)
```

**After:**
```go
func NewEventBusProvider(config EventBusConfig, opts ...Option) (EventBusProvider, error)
```

**Changes needed:**
- Same pattern as database provider
- Add logger field to `EventBusProviderImpl`
- Add `WithLogger()` function

### 3. Outbox Providers (`internal/platform/outbox/providers/writer.go` and `publisher.go`)

**Current:**
```go
func NewWriterProvider(db database.Database) WriterProvider
func NewPublisherProvider(db database.Database, eventBus bus.Bus) PublisherProvider
```

**After:**
```go
func NewWriterProvider(db database.Database, opts ...Option) WriterProvider
func NewPublisherProvider(db database.Database, eventBus bus.Bus, opts ...Option) PublisherProvider
```

### 4. SSE Provider (`internal/platform/sse/provider.go`)

**Current:**
```go
func NewProvider() *Provider
```

**After:**
```go
func NewProvider(opts ...Option) *Provider
```

Note: The SSE provider doesn't currently log much, but adding the option ensures consistency.

### 5. CORS Provider (`internal/platform/cors/provider.go`)

Check if it has logging and apply same pattern if needed.

### 6. Storage Provider (`internal/platform/storage/provider.go`)

Check if it has logging and apply same pattern if needed.

### 7. Downloader Provider (`internal/platform/downloader/provider.go`)

Check if it has logging and apply same pattern if needed.

---

## Main.go Updates (Per Service)

For each service, update the main.go to pass the logger to platform providers:

```go
// BEFORE (cart/main.go)
dbProvider, err := database.NewDatabaseProvider(dbURL)
eventBusProvider, err := event.NewEventBusProvider(eventBusConfig)
writerProvider := providers.NewWriterProvider(db)
publisherProvider := providers.NewPublisherProvider(db, eventBus)
sseProvider := sse.NewProvider()

// AFTER
dbProvider, err := database.NewDatabaseProvider(dbURL, database.WithLogger(logger))
eventBusProvider, err := event.NewEventBusProvider(eventBusConfig, event.WithLogger(logger))
writerProvider := providers.NewWriterProvider(db, providers.WithLogger(logger))
publisherProvider := providers.NewPublisherProvider(db, eventBus, providers.WithLogger(logger))
sseProvider := sse.NewProvider(sse.WithLogger(logger))
```

---

## Backward Compatibility

The functional options pattern maintains backward compatibility:

- Calling `NewDatabaseProvider(url)` (no options) still works
- Package-level logger is used as fallback
- No changes needed to code that doesn't pass a logger

---

## Testing Verification

After implementation, verify:

1. **LOG_LEVEL works**: Set `LOG_LEVEL=debug` and verify debug logs appear
2. **Service identity in logs**: Check that database/event/outbox logs now include the service name attribute
3. **Backward compatibility**: Services still start without passing loggers to providers (using fallback)

Example Loki query to verify:
```
{job="go-shopping-poc"} | json | service="cart"
```

---

## Summary of Changes

| Category | Files | Change Type |
|----------|-------|-------------|
| Service main.go | 8 files | Use DefaultLoggerConfig, pass logger to providers |
| Platform providers | 8-10 files | Add functional options for logger |
| Package loggers | 0 (keep existing) | No changes needed |

All platform providers should receive logger options:
- `internal/platform/database/provider.go`
- `internal/platform/event/provider.go`
- `internal/platform/outbox/providers/writer.go`
- `internal/platform/outbox/providers/publisher.go`
- `internal/platform/sse/provider.go`
- `internal/platform/cors/provider.go`
- `internal/platform/storage/provider.go`
- `internal/platform/downloader/provider.go`
- `internal/platform/outbox/writer.go` (struct, not a provider, but gets option)

---

## Open Questions / Clarifications Needed

1. **CORS, Storage, Downloader**: Do these packages have logging that would benefit from service identity? Should they be updated too?
   > **Answer**: Yes - All platform services that have a provider should get the logger option. This is now the project pattern.

2. **Testing**: Are there existing tests for providers that would need updating after changing constructor signatures?
   > **Answer**: There are no tests yet - none needed.

3. **Outbox Writer**: Currently the `outbox.Writer` struct doesn't have a logger. Should it receive one too, or is it sufficient for the provider to have the logger?
   > **Answer**: Give it the option of receiving one. We may want to create logs from there at some point.
