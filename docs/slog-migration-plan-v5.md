# slog Migration Plan v5 - Domain-First Phased Implementation

## Overview

This document provides a comprehensive plan for migrating from the legacy `log` package to the standard library `log/slog` package. This v5 plan addresses all deficiencies in v4 and implements a domain-first, phase-by-phase approach that:

1. Migrates one domain service at a time (cart → customer → eventreader → order → product)
2. Places platform infrastructure after all domain services are complete
3. Creates a single `logger.go` per platform package for consistent access
4. Makes breaking changes without backward compatibility concerns
5. Provides extremely detailed implementation instructions for LLM execution

## Goals

1. Replace all `log.Printf()` calls with structured `slog` logging
2. Add request correlation IDs for distributed tracing
3. Enable JSON output for Kubernetes environments
4. Implement each platform service in its own dedicated phase
5. Create clean, consistent logging patterns across the entire codebase

---

## Architecture Decisions

### 1. Logger Package Structure

Each platform package will have its own `logger.go` file that exports a package-level logger. This is the Go library convention (see `database/sql`, `net/http`).

```go
// internal/platform/database/logger.go
package database

import (
    "log/slog"
    "os"
)

var (
    logger *slog.Logger
)

func init() {
    logger = slog.New(slog.NewTextHandler(os.Stderr, nil)).With("platform", "database")
}

// Logger returns the package-level logger for database operations
func Logger() *slog.Logger {
    return logger
}

// SetLogger allows overriding the default logger (useful for testing)
func SetLogger(l *slog.Logger) {
    logger = l
}
```

### 2. Domain Service Logger Injection

Domain services receive loggers via constructor injection. The logger is stored on the service struct and made available to handlers.

```go
// internal/service/cart/service.go
type CartService struct {
    *service.EventServiceBase
    logger        *slog.Logger
    repo          CartRepository
    infrastructure *CartInfrastructure
    config        *Config
}

func NewCartService(
    logger *slog.Logger,
    infrastructure *CartInfrastructure,
    config *Config,
) *CartService {
    return &CartService{
        EventServiceBase: service.NewEventServiceBase("cart", infrastructure.EventBus),
        logger:          logger.With("service", "cart"),
        repo:            NewCartRepository(infrastructure.Database, infrastructure.OutboxWriter),
        infrastructure:  infrastructure,
        config:          config,
    }
}
```

### 3. Context-Aware Logging with slog.Middleware

Instead of using `ctx.Value()`, we implement a middleware pattern that enriches the context with a context-aware logger:

```go
// internal/platform/logging/context.go
package logging

import (
    "context"
    "log/slog"
)

type contextKey int

const (
    loggerKey contextKey = iota
)

// WithLogger returns a new context with the given logger
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
    return context.WithValue(ctx, loggerKey, logger)
}

// FromContext returns the logger from the context, or the default logger if not set
func FromContext(ctx context.Context) *slog.Logger {
    if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok && logger != nil {
        return logger
    }
    return slog.Default()
}
```

---

## Implementation Phases

### PHASE 1: Create Logging Infrastructure Package

**Duration**: 1 day  
**Files Created**: 5 files  
**Purpose**: Create the core logging utilities that all services will use

#### Files to Create

```
internal/platform/logging/
├── logger.go       # Core logger setup, provider, and configuration
├── context.go      # Context-aware logging helpers
├── level.go        # Log level utilities
└── attributes.go   # Common attribute builders
```

#### 1.1 internal/platform/logging/logger.go

```go
package logging

import (
    "fmt"
    "log/slog"
    "os"
    "strings"
)

type LoggerConfig struct {
    ServiceName string
    Level       string // "debug", "info", "warn", "error"
    Format      string // "json", "text"
}

func DefaultLoggerConfig(serviceName string) LoggerConfig {
    return LoggerConfig{
        ServiceName: serviceName,
        Level:       getEnv("LOG_LEVEL", "info"),
        Format:      getEnv("LOG_FORMAT", "json"),
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

type LoggerProvider struct {
    logger *slog.Logger
}

func NewLoggerProvider(config LoggerConfig) (*LoggerProvider, error) {
    if config.ServiceName == "" {
        return nil, fmt.Errorf("service name is required")
    }

    level := parseLevel(config.Level)
    format := config.Format

    var handler slog.Handler

    handlerOptions := &slog.HandlerOptions{
        Level: level,
        ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
            // Group correlation IDs under "context" for cleaner JSON
            if a.Key == "request_id" || a.Key == "trace_id" {
                return slog.String("context."+a.Key, a.Value.String())
            }
            return a
        },
    }

    if format == "text" || os.Getenv("ENVIRONMENT") == "development" {
        handler = slog.NewTextHandler(os.Stdout, handlerOptions)
    } else {
        handler = slog.NewJSONHandler(os.Stdout, handlerOptions)
    }

    logger := slog.New(handler).With("service", config.ServiceName)

    return &LoggerProvider{logger: logger}, nil
}

func (p *LoggerProvider) Logger() *slog.Logger {
    return p.logger
}

func (p *LoggerProvider) With(attrs ...slog.Attr) *slog.Logger {
    return p.logger.With(attrs...)
}

func parseLevel(levelStr string) slog.Level {
    switch strings.ToLower(levelStr) {
    case "debug":
        return slog.LevelDebug
    case "warn", "warning":
        return slog.LevelWarn
    case "error":
        return slog.LevelError
    default:
        return slog.LevelInfo
    }
}
```

#### 1.2 internal/platform/logging/context.go

```go
package logging

import (
    "context"
    "log/slog"
)

type contextKey int

const (
    loggerKey contextKey = iota
)

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
    return context.WithValue(ctx, loggerKey, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
    if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok && logger != nil {
        return logger
    }
    return slog.Default()
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
    logger := FromContext(ctx)
    return WithLogger(ctx, logger.With("request_id", requestID))
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
    logger := FromContext(ctx)
    return WithLogger(ctx, logger.With("trace_id", traceID))
}
```

#### 1.3 internal/platform/logging/level.go

```go
package logging

import "log/slog"

func IsDebugEnabled(logger *slog.Logger, ctx context.Context) bool {
    return logger.Enabled(ctx, slog.LevelDebug)
}

func IsInfoEnabled(logger *slog.Logger, ctx context.Context) bool {
    return logger.Enabled(ctx, slog.LevelInfo)
}

func IsWarnEnabled(logger *slog.Logger, ctx context.Context) bool {
    return logger.Enabled(ctx, slog.LevelWarn)
}

func IsErrorEnabled(logger *slog.Logger, ctx context.Context) bool {
    return logger.Enabled(ctx, slog.LevelError)
}
```

#### 1.4 internal/platform/logging/attributes.go

```go
package logging

import "log/slog"

func ErrorAttr(err error) slog.Attr {
    return slog.String("error", err.Error())
}

func ErrorStackAttr(err error) slog.Attr {
    // For structured error with stack trace
    if st, ok := err.(interface{ StackTrace() string }); ok {
        return slog.String("stack_trace", st.StackTrace())
    }
    return slog.String("error", err.Error())
}

func DurationAttr(d interface{ Nanoseconds() int64 }) slog.Attr {
    return slog.Duration("duration", d.Nanoseconds())
}

func Int64Attr(key string, value int64) slog.Attr {
    return slog.Int64(key, value)
}

func Uint64Attr(key string, value uint64) slog.Attr {
    return slog.Uint64(key, value)
}
```

---

### PHASE 2: Migrate Cart Domain Service

**Duration**: 2 days  
**Files Modified**: ~15 files  
**Files Created**: 0 files (platform logger.go files are created in Platform phases)  
**Purpose**: Migrate cart service completely to slog, serving as the pattern for all subsequent domain services

#### Step 2.1: Add LoggerProvider to providers.go

Modify `internal/platform/providers/providers.go`:

```go
import (
    "log/slog"  // ADD THIS IMPORT
)

// LoggerProvider defines the interface for providing structured logging.
type LoggerProvider interface {
    GetLogger() *slog.Logger
}
```

#### Step 2.2: Modify EventServiceBase to accept logger

Modify `internal/platform/service/base.go`:

```go
package service

import (
    "context"
    "log/slog"  // ADD THIS IMPORT
    
    "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/event/bus"
    kafkabus "go-shopping-poc/internal/platform/event/bus/kafka"
    "reflect"
)

type EventServiceBase struct {
    *BaseService
    eventBus bus.Bus
    handlers []any
    logger   *slog.Logger  // ADD THIS FIELD
}

// MODIFY: NewEventServiceBase to accept logger parameter
func NewEventServiceBase(name string, eventBus bus.Bus, logger *slog.Logger) *EventServiceBase {
    if logger == nil {
        logger = slog.Default().With("service", name)
    }
    return &EventServiceBase{
        BaseService: NewBaseService(name),
        eventBus:    eventBus,
        handlers:    make([]any, 0),
        logger:      logger,
    }
}

// ADD: Logger accessor method
func (s *EventServiceBase) Logger() *slog.Logger {
    return s.logger
}
```

#### Step 2.3: Modify CartService to accept and use logger

Modify `internal/service/cart/service.go`:

```go
package cart

import (
    "context"
    "errors"
    "fmt"
    "log/slog"  // ADD THIS IMPORT
    
    "net/http"

    "github.com/google/uuid"

    events "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/database"
    "go-shopping-poc/internal/platform/event/bus"
    "go-shopping-poc/internal/platform/logging"
    "go-shopping-poc/internal/platform/outbox"
    "go-shopping-poc/internal/platform/service"
    "go-shopping-poc/internal/platform/sse"
)

// MODIFY: NewCartService to accept logger parameter
func NewCartService(
    logger *slog.Logger,
    infrastructure *CartInfrastructure,
    config *Config,
) *CartService {
    if logger == nil {
        logger = logging.FromContext(context.Background())
    }
    
    return &CartService{
        EventServiceBase: service.NewEventServiceBase("cart", infrastructure.EventBus, logger),
        logger:          logger.With("component", "cart_service"),
        repo:            NewCartRepository(infrastructure.Database, infrastructure.OutboxWriter),
        infrastructure:  infrastructure,
        config:          config,
    }
}

// MODIFY: NewCartServiceWithRepo to accept logger parameter
func NewCartServiceWithRepo(
    logger *slog.Logger,
    repo CartRepository,
    infrastructure *CartInfrastructure,
    config *Config,
) *CartService {
    if logger == nil {
        logger = logging.FromContext(context.Background())
    }
    
    return &CartService{
        EventServiceBase: service.NewEventServiceBase("cart", infrastructure.EventBus, logger),
        logger:          logger.With("component", "cart_service"),
        repo:            repo,
        infrastructure:  infrastructure,
        config:          config,
    }
}

// REPLACE ALL log.Printf with slog calls
// Example transformations:

// BEFORE:
/*
log.Printf("[DEBUG] CartService: Adding item to cart %s: product_id=%s, quantity=%d", cartID, productID, quantity)
*/

// AFTER:
/*
s.logger.Debug("Adding item to cart",
    "cart_id", cartID,
    "product_id", productID,
    "quantity", quantity,
)
*/

// Continue replacing all log.Printf calls in the file:
```

#### Step 2.4: Migrate all CartService log.Printf calls

Replace ALL `log.Printf` calls in `internal/service/cart/service.go` with equivalent slog calls:

| Original | Replacement |
|----------|-------------|
| `log.Printf("[DEBUG] CartService: Adding item...")` | `s.logger.Debug("Adding item to cart", ...)` |
| `log.Printf("[WARN] Cart: Failed to trigger...")` | `s.logger.Warn("Failed to trigger outbox processing", ...)` |
| `log.Printf("[INFO] CartService: Added pending item...")` | `s.logger.Info("Added pending item to cart", ...)` |

#### Step 2.5: Migrate CartRepository log.Printf calls

Find and replace in `internal/service/cart/repository*.go` files:

```go
// Add to repository struct:
type CartRepository struct {
    db          database.Database
    outbox      *outbox.Writer
    logger      *slog.Logger  // ADD
}

// Modify repository constructor:
func NewCartRepository(db database.Database, outbox *outbox.Writer) *CartRepository {
    return &CartRepository{
        db:     db,
        outbox: outbox,
        logger: database.Logger().With("component", "cart_repository"),  // ADD
    }
}
```

Replace all `log.Printf` calls in repository files with `r.logger.Debug/Info/Warn/Error` calls.

#### Step 2.6: Migrate Cart event handlers

Files in `internal/service/cart/eventhandlers/*.go`:

```go
// Each handler factory receives the service logger
func NewOnOrderCreatedHandler(sseHub *sse.Hub) *OnOrderCreatedHandler {
    return &OnOrderCreatedHandler{
        sseHub: sseHub,
        logger: slog.Default().With("handler", "on_order_created"),  // ADD - will be injected later
    }
}

// Modify to accept logger:
func NewOnOrderCreatedHandler(sseHub *sse.Hub, logger *slog.Logger) *OnOrderCreatedHandler {
    return &OnOrderCreatedHandler{
        sseHub: sseHub,
        logger: logger.With("handler", "on_order_created"),
    }
}
```

#### Step 2.7: Migrate cmd/cart/main.go

Modify `cmd/cart/main.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"  // KEEP for fatal errors only
    "log/slog"  // ADD
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "go-shopping-poc/internal/contracts/events"
    "go-shopping-poc/internal/platform/cors"
    "go-shopping-poc/internal/platform/database"
    "go-shopping-poc/internal/platform/event"
    "go-shopping-poc/internal/platform/logging"
    "go-shopping-poc/internal/platform/outbox"
    "go-shopping-poc/internal/platform/outbox/providers"
    "go-shopping-poc/internal/platform/sse"
    "go-shopping-poc/internal/service/cart"
    "go-shopping-poc/internal/service/cart/eventhandlers"

    "github.com/go-chi/chi/v5"
)

func main() {
    // Create logger provider
    loggerProvider, err := logging.NewLoggerProvider(logging.LoggerConfig{
        ServiceName: "cart",
    })
    if err != nil {
        log.Fatalf("Cart: Failed to create logger provider: %v", err)
    }
    logger := loggerProvider.Logger()
    
    logger.Info("Cart service starting", "version", "1.0.0")

    cfg, err := cart.LoadConfig()
    if err != nil {
        logger.Error("Failed to load config", logging.ErrorAttr(err))
        os.Exit(1)
    }

    logger.Debug("Configuration loaded", "read_topics", cfg.ReadTopics, "write_topic", cfg.WriteTopic, "group", cfg.Group)

    // Database setup
    dbURL := cfg.DatabaseURL
    if dbURL == "" {
        logger.Error("Database URL is required")
        os.Exit(1)
    }

    logger.Debug("Creating database provider")
    dbProvider, err := database.NewDatabaseProvider(dbURL)
    if err != nil {
        logger.Error("Failed to create database provider", logging.ErrorAttr(err))
        os.Exit(1)
    }
    db := dbProvider.GetDatabase()
    defer func() {
        if err := db.Close(); err != nil {
            logger.Error("Error closing database", logging.ErrorAttr(err))
        }
    }()

    // Event bus setup
    logger.Debug("Creating event bus provider")
    eventBusConfig := event.EventBusConfig{
        WriteTopic: cfg.WriteTopic,
        GroupID:    cfg.Group,
    }
    eventBusProvider, err := event.NewEventBusProvider(eventBusConfig)
    if err != nil {
        logger.Error("Failed to create event bus provider", logging.ErrorAttr(err))
        os.Exit(1)
    }
    eventBus := eventBusProvider.GetEventBus()

    // Outbox setup
    logger.Debug("Creating outbox components")
    writerProvider := providers.NewWriterProvider(db)
    outboxWriter := writerProvider.GetWriter()

    outboxConfig := outbox.Config{
        BatchSize:       10,
        ProcessInterval: 5 * time.Second,
    }
    logger.Info("Outbox publisher configured", "interval", outboxConfig.ProcessInterval, "batch_size", outboxConfig.BatchSize)
    outboxPublisher := outbox.NewPublisher(db, eventBus, outboxConfig)
    outboxPublisher.Start()
    defer outboxPublisher.Stop()

    // CORS setup
    logger.Debug("Creating CORS provider")
    corsProvider, err := cors.NewCORSProvider()
    if err != nil {
        logger.Error("Failed to create CORS provider", logging.ErrorAttr(err))
        os.Exit(1)
    }
    corsHandler := corsProvider.GetCORSHandler()

    // SSE provider setup
    logger.Debug("Creating SSE provider")
    sseProvider := sse.NewProvider()

    // Infrastructure and service setup
    logger.Debug("Creating cart infrastructure")
    infrastructure := cart.NewCartInfrastructure(
        db, eventBus, outboxWriter, outboxPublisher, corsHandler, sseProvider,
    )

    // MODIFIED: Pass logger to service
    logger.Debug("Creating cart service")
    service := cart.NewCartService(logger, infrastructure, cfg)

    // Register event handlers
    logger.Debug("Registering event handlers")
    if err := registerEventHandlers(service, sseProvider.GetHub()); err != nil {
        logger.Error("Failed to register event handlers", logging.ErrorAttr(err))
        os.Exit(1)
    }

    // Start consuming events
    logger.Info("Starting event consumer", "topics", service.EventBus().ReadTopics())
    go func() {
        ctx := context.Background()
        if err := service.Start(ctx); err != nil {
            logger.Error("Event consumer error", logging.ErrorAttr(err))
        }
    }()

    logger.Debug("Creating cart handler")
    handler := cart.NewCartHandler(service)

    // HTTP Server setup
    logger.Debug("Setting up HTTP router")
    router := chi.NewRouter()
    router.Use(corsHandler)

    router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"status":"ok"}`))
    })

    cartRouter := chi.NewRouter()
    cartRouter.Post("/carts", handler.CreateCart)
    cartRouter.Get("/carts/{id}", handler.GetCart)
    cartRouter.Delete("/carts/{id}", handler.DeleteCart)

    cartRouter.Post("/carts/{id}/items", handler.AddItem)
    cartRouter.Put("/carts/{id}/items/{line}", handler.UpdateItem)
    cartRouter.Delete("/carts/{id}/items/{line}", handler.RemoveItem)

    cartRouter.Put("/carts/{id}/contact", handler.SetContact)
    cartRouter.Post("/carts/{id}/addresses", handler.AddAddress)
    cartRouter.Put("/carts/{id}/payment", handler.SetPayment)
    cartRouter.Post("/carts/{id}/checkout", handler.Checkout)
    cartRouter.Get("/carts/{id}/stream", sseProvider.GetHandler().ServeHTTP)

    router.Mount("/api/v1", cartRouter)

    serverAddr := "0.0.0.0" + cfg.ServicePort
    server := &http.Server{
        Addr:         serverAddr,
        Handler:      router,
        ReadTimeout:  30 * time.Second,
        IdleTimeout:  120 * time.Second,
    }

    done := make(chan bool, 1)
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

    go func() {
        logger.Info("Starting HTTP server", "address", serverAddr)
        if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Error("Failed to start HTTP server", logging.ErrorAttr(err))
        }
    }()

    <-quit
    logger.Info("Shutting down server")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := server.Shutdown(ctx); err != nil {
        logger.Error("Server forced to shutdown", logging.ErrorAttr(err))
    }

    close(done)
    logger.Info("Server exited")
}

// MODIFIED: registerEventHandlers passes logger to handlers
func registerEventHandlers(service *cart.CartService, sseHub *sse.Hub) error {
    logger := service.Logger()
    
    logger.Info("Registering event handlers")

    // Get service logger for handlers
    handlerLogger := logger.With("component", "event_handler")

    orderCreatedHandler := eventhandlers.NewOnOrderCreatedHandler(sseHub, handlerLogger)
    logger.Info("Registering handler", "event_type", orderCreatedHandler.EventType(), "topic", events.OrderEvent{}.Topic())

    if err := cart.RegisterHandler(
        service,
        orderCreatedHandler.CreateFactory(),
        orderCreatedHandler.CreateHandler(),
    ); err != nil {
        return fmt.Errorf("failed to register OrderCreated handler: %w", err)
    }

    productValidatedHandler := eventhandlers.NewOnProductValidatedHandler(service.GetRepository(), sseHub, handlerLogger)
    logger.Info("Registering handler", "event_type", productValidatedHandler.EventType())

    if err := cart.RegisterHandler(
        service,
        productValidatedHandler.CreateFactory(),
        productValidatedHandler.CreateHandler(),
    ); err != nil {
        return fmt.Errorf("failed to register ProductValidated handler: %w", err)
    }

    logger.Info("Event handler registration completed")
    return nil
}
```

#### Step 2.8: Verify Cart builds and runs

```bash
cd cmd/cart && go build ./...
```

---

### PHASE 3: Migrate Customer Domain Service

**Duration**: 2 days  
**Files Modified**: ~12 files  
**Purpose**: Follow the same pattern as cart service

#### Step 3.1: Files to Modify

1. `internal/service/customer/service.go` - Add logger parameter to constructors, replace log.Printf
2. `internal/service/customer/repository*.go` - Add logger to repository, replace log.Printf
3. `internal/service/customer/eventhandlers/*.go` - Add logger to handlers
4. `cmd/customer/main.go` - Use logger provider, pass logger to service

#### Step 3.2: Quick Reference (Same Pattern as Cart)

- Add `logger *slog.Logger` to service struct
- Modify constructor: `func NewCustomerService(logger *slog.Logger, infra *CustomerInfrastructure, cfg *Config) *CustomerService`
- Replace all `log.Printf` with `s.logger.Debug/Info/Warn/Error`
- In main.go, create logger provider and pass to service constructor

---

### PHASE 4: Migrate EventReader Domain Service

**Duration**: 2 days  
**Files Modified**: ~8 files  
**Purpose**: Migrate event reader service

#### Step 4.1: Files to Modify

1. `internal/service/eventreader/service.go`
2. `cmd/eventreader/main.go`

---

### PHASE 5: Migrate Order Domain Service

**Duration**: 2 days  
**Files Modified**: ~10 files  
**Purpose**: Migrate order service

#### Step 5.1: Files to Modify

1. `internal/service/order/service.go`
2. `internal/service/order/repository*.go`
3. `internal/service/order/eventhandlers/*.go`
4. `cmd/order/main.go`

---

### PHASE 6: Migrate Product Domain Service

**Duration**: 3 days  
**Files Modified**: ~15 files  
**Purpose**: Migrate product service (largest service)

#### Step 6.1: Files to Modify

1. `internal/service/product/service.go`
2. `internal/service/product/service_admin.go`
3. `internal/service/product/repository*.go`
4. `internal/service/product/eventhandlers/*.go`
5. `cmd/product/main.go`
6. `cmd/product-loader/main.go` (if exists)
7. `cmd/product-admin/main.go`

---

### PHASE 7: Migrate Remaining Domain Services

**Duration**: 2 days  
**Files Modified**: ~5 files  
**Purpose**: Migrate any remaining domain services

#### Step 7.1: Services to check

- websocket (if it has domain logic)
- Any other services found in `cmd/*/main.go` not yet migrated

---

### PHASE 8: Migrate Platform - Database Package

**Duration**: 1 day  
**Files Modified**: ~5 files  
**Purpose**: Migrate database platform to slog

#### Step 8.1: Create/update logger.go

File: `internal/platform/database/logger.go` (created in Phase 2, enhance it):

```go
package database

import (
    "log/slog"
    "os"
)

var (
    logger *slog.Logger
)

func init() {
    // Default logger - will be replaced when database provider is initialized
    logger = slog.New(slog.NewJSONHandler(os.Stderr, nil)).
        With("platform", "database", "component", "postgresql")
}

func Logger() *slog.Logger {
    return logger
}

func SetLogger(l *slog.Logger) {
    logger = l
}

// InitLogger initializes the database logger with configuration
func InitLogger(serviceName string, level string) {
    var lvl slog.Level
    switch level {
    case "debug":
        lvl = slog.LevelDebug
    case "warn", "warning":
        lvl = slog.LevelWarn
    case "error":
        lvl = slog.LevelError
    default:
        lvl = slog.LevelInfo
    }
    
    logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
        Level: lvl,
    })).With("platform", "database", "component", "postgresql", "service", serviceName)
}
```

#### Step 8.2: Modify postgresql.go

Modify `internal/platform/database/postgresql.go`:

```go
package database

import (
    "context"
    "database/sql"
    "fmt"
    "log/slog"  // ADD
    "time"

    _ "github.com/jackc/pgx/v5/stdlib"
    "github.com/jmoiron/sqlx"
)

type PostgreSQLClient struct {
    db          *sqlx.DB
    databaseURL string
    connConfig  ConnectionConfig
    logger      *slog.Logger  // ADD
}

// MODIFY: NewPostgreSQLClient to accept optional logger
func NewPostgreSQLClient(databaseURL string, connConfig ...ConnectionConfig) (Database, error) {
    cfg := DefaultConnectionConfig()
    if len(connConfig) > 0 {
        cfg = connConfig[0]
    }

    client := &PostgreSQLClient{
        databaseURL: databaseURL,
        connConfig:  cfg,
        logger:      Logger(),  // ADD - use package logger
    }

    return client, nil
}

// MODIFY: Add logger parameter to Connect
func (c *PostgreSQLClient) Connect(ctx context.Context) error {
    c.logger.Info("Connecting to PostgreSQL",
        "host", c.connConfig.Host,
        "port", c.connConfig.Port,
        "database", c.connConfig.Database,
    )

    db, err := sqlx.Connect("pgx", c.databaseURL)
    if err != nil {
        c.logger.Error("Failed to connect to PostgreSQL",
            "error", err.Error(),
        )
        return fmt.Errorf("failed to connect to database: %w", err)
    }

    db.SetMaxOpenConns(c.connConfig.MaxOpenConns)
    db.SetMaxIdleConns(c.connConfig.MaxIdleConns)
    db.SetConnMaxLifetime(c.connConfig.ConnMaxLifetime)
    db.SetConnMaxIdleTime(c.connConfig.ConnMaxIdleTime)

    c.db = db

    c.logger.Info("Successfully connected to PostgreSQL")
    return nil
}

// MODIFY: Add logger to Close
func (c *PostgreSQLClient) Close() error {
    if c.db == nil {
        return nil
    }

    c.logger.Info("Closing PostgreSQL connection")
    err := c.db.Close()
    c.db = nil

    if err != nil {
        c.logger.Error("Failed to close connection",
            "error", err.Error(),
        )
        return fmt.Errorf("failed to close database connection: %w", err)
    }

    c.logger.Info("PostgreSQL connection closed")
    return nil
}

// MODIFY: Add logger to Ping
func (c *PostgreSQLClient) Ping(ctx context.Context) error {
    if c.db == nil {
        return fmt.Errorf("database connection not established")
    }

    pingCtx, cancel := context.WithTimeout(ctx, c.connConfig.HealthCheckTimeout)
    defer cancel()

    start := time.Now()
    err := c.db.PingContext(pingCtx)
    latency := time.Since(start)

    if err != nil {
        c.logger.Error("Ping failed",
            "latency", latency.String(),
            "error", err.Error(),
        )
        return fmt.Errorf("database ping failed: %w", err)
    }

    c.logger.Debug("Ping successful", "latency", latency.String())
    return nil
}

// MODIFY: Add optional query logging to Query
func (c *PostgreSQLClient) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
    if c.db == nil {
        return nil, fmt.Errorf("database connection not established")
    }

    start := time.Now()
    rows, err := c.db.QueryContext(ctx, query, args...)
    latency := time.Since(start)

    if err != nil {
        c.logger.Error("Query failed",
            "latency", latency.String(),
            "error", err.Error(),
        )
        return nil, fmt.Errorf("query execution failed: %w", err)
    }

    c.logger.Debug("Query completed", "latency", latency.String())
    return rows, nil
}

// ... Continue modifying all methods that have log.Printf
```

#### Step 8.3: Modify all database package files

- `internal/platform/database/postgresql.go`
- `internal/platform/database/provider.go`
- `internal/platform/database/health.go`
- `internal/platform/database/interface.go`

Replace all `log.Printf` with `c.logger.Debug/Info/Warn/Error`.

---

### PHASE 9: Migrate Platform - Event Bus (Kafka)

**Duration**: 1 day  
**Files Modified**: ~5 files  
**Purpose**: Migrate Kafka event bus to slog

#### Step 9.1: Create logger.go

Create `internal/platform/event/bus/kafka/logger.go`:

```go
package kafka

import (
    "log/slog"
    "os"
)

var (
    logger *slog.Logger
)

func init() {
    logger = slog.New(slog.NewJSONHandler(os.Stderr, nil)).
        With("platform", "event", "component", "kafka")
}

func Logger() *slog.Logger {
    return logger
}

func SetLogger(l *slog.Logger) {
    logger = l
}
```

#### Step 9.2: Modify Kafka event bus files

Files to modify:
- `internal/platform/event/bus/kafka/eventbus.go`
- `internal/platform/event/bus/kafka/handler.go`

Replace all `log.Printf` with logger calls.

---

### PHASE 10: Migrate Platform - Outbox

**Duration**: 1 day  
**Files Modified**: ~5 files  
**Purpose**: Migrate outbox pattern to slog

#### Step 10.1: Create logger.go

Create `internal/platform/outbox/logger.go`:

```go
package outbox

import (
    "log/slog"
    "os"
)

var (
    logger *slog.Logger
)

func init() {
    logger = slog.New(slog.NewJSONHandler(os.Stderr, nil)).
        With("platform", "outbox")
}

func Logger() *slog.Logger {
    return logger
}

func SetLogger(l *slog.Logger) {
    logger = l
}
```

#### Step 10.2: Modify outbox files

- `internal/platform/outbox/writer.go`
- `internal/platform/outbox/publisher.go`
- `internal/platform/outbox/outbox.go`

---

### PHASE 11: Migrate Platform - SSE

**Duration**: 1 day  
**Files Modified**: ~4 files  
**Purpose**: Migrate Server-Sent Events to slog

#### Step 11.1: Create logger.go

Create `internal/platform/sse/logger.go`:

```go
package sse

import (
    "log/slog"
    "os"
)

var (
    logger *slog.Logger
)

func init() {
    logger = slog.New(slog.NewJSONHandler(os.Stderr, nil)).
        With("platform", "sse")
}

func Logger() *slog.Logger {
    return logger
}

func SetLogger(l *slog.Logger) {
    logger = l
}
```

#### Step 11.2: Modify SSE files

- `internal/platform/sse/hub.go`
- `internal/platform/sse/handler.go`
- `internal/platform/sse/client.go`

---

### PHASE 12: Migrate Platform - WebSocket

**Duration**: 1 day  
**Files Modified**: ~3 files  
**Purpose**: Migrate WebSocket to slog

#### Step 12.1: Create logger.go

Create `internal/platform/websocket/logger.go`:

```go
package websocket

import (
    "log/slog"
    "os"
)

var (
    logger *slog.Logger
)

func init() {
    logger = slog.New(slog.NewJSONHandler(os.Stderr, nil)).
        With("platform", "websocket")
}

func Logger() *slog.Logger {
    return logger
}

func SetLogger(l *slog.Logger) {
    logger = l
}
```

#### Step 12.2: Modify websocket files

- `internal/platform/websocket/websocket.go`

---

### PHASE 13: Migrate Platform - Remaining Packages

**Duration**: 2 days  
**Files Modified**: ~15 files  
**Purpose**: Migrate all remaining platform packages

#### Step 13.1: Create logger.go for each package

Create `logger.go` files for:
- `internal/platform/storage/minio/logger.go`
- `internal/platform/downloader/logger.go`
- `internal/platform/cors/logger.go`
- `internal/platform/csv/logger.go`
- `internal/platform/auth/logger.go`
- `internal/platform/config/logger.go` (if it logs)

#### Step 13.2: Modify files in each package

Replace all `log.Printf` with appropriate logger calls.

---

### PHASE 14: Final Verification

**Duration**: 1 day  
**Purpose**: Verify all code compiles and remove any remaining log imports

#### Step 14.1: Verify no log.Printf remains

```bash
grep -r "log\.Print" --include="*.go" internal/ cmd/
```

#### Step 14.2: Build all services

```bash
for service in cart customer eventreader order product product-loader product-admin websocket; do
    echo "Building $service..."
    go build -o /dev/null ./cmd/$service/...
done
```

#### Step 14.3: Run tests

```bash
go test ./...
```

---

## Summary of Changes

### New Files Created

| File | Purpose |
|------|---------|
| `internal/platform/logging/logger.go` | Core logging infrastructure |
| `internal/platform/logging/context.go` | Context-aware logging helpers |
| `internal/platform/logging/level.go` | Log level utilities |
| `internal/platform/logging/attributes.go` | Common attribute builders |
| `internal/platform/*/logger.go` | Package-level loggers for each platform package |

### Files Modified (by phase)

| Phase | Domain/Platform | Files |
|-------|-----------------|-------|
| 1 | Infrastructure | 4 |
| 2 | Cart | 15 |
| 3 | Customer | 12 |
| 4 | EventReader | 8 |
| 5 | Order | 10 |
| 6 | Product | 15 |
| 7 | Remaining domains | 5 |
| 8 | Platform: Database | 5 |
| 9 | Platform: Event/Kafka | 5 |
| 10 | Platform: Outbox | 5 |
| 11 | Platform: SSE | 4 |
| 12 | Platform: WebSocket | 3 |
| 13 | Platform: Storage, Downloader, etc. | 15 |
| 14 | Verification | All |

### Total: ~101 files modified/created

---

## Key Patterns to Follow

### 1. Constructor Pattern

```go
func NewServiceName(
    logger *slog.Logger,
    infrastructure *ServiceInfrastructure,
    config *Config,
) *ServiceName {
    if logger == nil {
        logger = slog.Default()
    }
    return &ServiceName{
        EventServiceBase: service.NewEventServiceBase("serviceName", infrastructure.EventBus, logger),
        logger:          logger.With("component", "service_name"),
        // ... other fields
    }
}
```

### 2. Package Logger Pattern

```go
// internal/platform/somepackage/logger.go
package somepackage

import (
    "log/slog"
    "os"
)

var logger *slog.Logger

func init() {
    logger = slog.New(slog.NewJSONHandler(os.Stderr, nil)).
        With("platform", "somepackage")
}

func Logger() *slog.Logger {
    return logger
}
```

### 3. Context-Aware Logging

```go
// In handlers that receive context
func (h *Handler) Handle(ctx context.Context, event *Event) error {
    logger := logging.FromContext(ctx)
    logger.Info("Processing event", "event_type", event.Type())
    // ...
}
```

### 4. Log Level Mapping

| Original | New |
|----------|-----|
| `log.Printf("[DEBUG] ...")` | `logger.Debug("...")` |
| `log.Printf("[INFO] ...")` | `logger.Info("...")` |
| `log.Printf("[WARN] ...")` | `logger.Warn("...")` |
| `log.Printf("[ERROR] ...")` | `logger.Error("...", slog.String("error", err.Error()))` |

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LOG_LEVEL` | `info` | Log level: debug, info, warn, error |
| `LOG_FORMAT` | `json` | Output format: json, text |
| `ENVIRONMENT` | (none) | If set to "development", uses text format |

---

## Migration Order Justification

1. **Domain services first**: These contain the most business logic and are the primary focus of the application. Migrating them one at a time ensures the core functionality is working correctly.

2. **Cart first**: It's the simplest domain service with the least number of files, making it ideal as the "pilot" migration.

3. **Platform after domains**: Platform infrastructure is used by all services. Migrating it after the domains ensures the patterns are established and we don't break multiple services at once.

4. **Each platform service in its own phase**: This isolates changes and makes debugging easier. If something goes wrong, we know exactly which platform package caused it.

---

## Breaking Changes

This plan intentionally makes the following breaking changes:

1. **Constructor signatures change**: All service constructors now require a `*slog.Logger` parameter
2. **Handler constructors change**: Event handlers now receive a logger parameter
3. **No backward compatibility**: No `New*ServiceDefault` functions are created
4. **Package-level loggers**: Platform packages now have their own logger accessors
5. **Removed log imports**: All `log` package imports are removed from service code

These breaking changes are acceptable because:
- The migration is done in controlled phases
- Each phase is tested before moving to the next
- The LLM implementing this has clear, detailed instructions for each file

---

## Success Criteria

1. ✅ All `log.Printf` calls replaced with structured slog calls
2. ✅ Each platform package has a `logger.go` file
3. ✅ All domain services use constructor-injected loggers
4. ✅ All event handlers receive loggers
5. ✅ Context-aware logging implemented via `logging.FromContext(ctx)`
6. ✅ JSON format in production, text format in development
7. ✅ All services compile successfully
8. ✅ All tests pass
