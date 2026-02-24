# Slog Migration Plan

## Overview

This document describes the migration from Go's standard `log` package to `log/slog` (hereafter "slog") for consistent, structured logging across the entire project.

## Current State Analysis

### Logging Inventory

| Location | Files | Log Statements |
|----------|-------|----------------|
| `cmd/*` (service entry points) | 8 | ~240 |
| `internal/platform/*` (shared infrastructure) | 45 | ~152 |
| `internal/service/*` (domain services) | ~30 | ~217 |
| **Total** | **~83** | **~609** |

### Current Issues

1. **No structured logging** - Plain text with custom `[DEBUG]`, `[INFO]`, `[WARN]`, `[ERROR]` prefixes
2. **No centralized configuration** - Each service sets `log.SetFlags(log.LstdFlags)` independently
3. **Inconsistent log levels** - Custom prefixes instead of proper log levels
4. **No JSON output** - Cannot easily parse logs in production
5. **No request correlation** - Cannot trace requests across services

### Target State

- JSON output by default (configurable to text)
- Centralized configuration via environment variables
- Structured key-value logging
- Consistent log levels (debug, info, warn, error)
- Service name attribute on all log entries
- Request-scoped logging via `context.Context`

---

## Evaluation Rubric

Each approach is evaluated on the following criteria (1-5 scale, 5 being best):

| Criterion | Description | Weight |
|-----------|-------------|--------|
| **Consistency** | Ensures uniform logging across all code | 20% |
| **Maintainability** | Ease of future changes and additions | 20% |
| **Performance** | Runtime overhead of logging | 15% |
| **Developer Experience** | Ease of use for developers | 15% |
| **Integration** | How well it integrates with existing code | 15% |
| **Observability** | Quality of JSON output for log aggregation | 15% |

---

## Approaches Evaluated

### Approach A: Global Logger with Import Alias

Create a global logger in a new package and import with alias in every file.

```go
// internal/platform/logging/logger.go
var DefaultLogger *slog.Logger

// Usage in each file:
import log "go-shopping-poc/internal/platform/logging"

log.DefaultLogger.Info("message", "key", "value")
```

**Scores:**
- Consistency: 5 (uniform across all files)
- Maintainability: 4
- Performance: 5 (direct slog calls)
- Developer Experience: 3 (verbose, requires import in every file)
- Integration: 4
- Observability: 5

**Weighted Score: 4.3**

---

### Approach B: Context-Aware Logger with Constructor Injection

Pass logger through constructors of services and handlers.

```go
type Service struct {
    logger *slog.Logger
}

func NewService(logger *slog.Logger) *Service {
    return &Service{logger: logger}
}
```

**Scores:**
- Consistency: 4 (requires discipline to pass logger)
- Maintainability: 5 (explicit dependencies)
- Performance: 5
- Developer Experience: 5 (clear dependency)
- Integration: 2 (massive refactoring required)
- Observability: 5

**Weighted Score: 4.3**

---

### Approach C: Package-Level Logger with Helper Functions (SELECTED)

Create a logging package with helper functions that wrap slog, initialized once per service.

```go
// internal/platform/logging/logger.go
package logging

type Logger struct {
    *slog.Logger
    serviceName string
}

func New(serviceName string) *Logger {
    // Configure handler from env vars
    // Add service name as default attribute
    return &Logger{logger: logger, serviceName: serviceName}
}

// Methods: Debug, Info, Warn, Error, Fatal
func (l *Logger) Info(msg string, attrs ...any)
```

**Scores:**
- Consistency: 5 (all code uses same package)
- Maintainability: 5 (centralized, easy changes)
- Performance: 5 (thin wrapper over slog)
- Developer Experience: 5 (simple API, familiar patterns)
- Integration: 4 (requires import, but minimal code change)
- Observability: 5

**Weighted Score: 4.8**

**Rationale:** Approach C provides the best balance of consistency, maintainability, and developer experience. It centralizes configuration while providing a simple API that mirrors existing logging patterns. The thin wrapper over slog ensures minimal performance overhead.

---

## Configuration

### Environment Variables

| Variable | Values | Default | Description |
|----------|--------|---------|-------------|
| `LOG_LEVEL` | debug, info, warn, error | info | Minimum log level to output |
| `LOG_FORMAT` | json, text | json | Output format |

### Kubernetes Integration

**Platform ConfigMap** (`deploy/k8s/platform/config/platform-configmap-for-services.yaml`):

```yaml
data:
  LOG_LEVEL: "info"
  LOG_FORMAT: "json"  # ADD THIS
```

All service deployments already reference `LOG_LEVEL` from the platform ConfigMap. No changes needed to individual deploy files.

---

## Implementation Architecture

### File Structure

```
internal/platform/logging/
├── logger.go      # Main Logger type and methods
├── config.go     # Configuration loading from environment
└── context.go    # Context key types for structured logging
```

### Logger API

```go
package logging

// New creates a new Logger with the given service name
func New(serviceName string) *Logger

// Logger provides structured logging with service context
type Logger struct {
    // Embedded slog.Logger for compatibility
    *slog.Logger
}

// Methods mirror slog with added context
func (l *Logger) Debug(msg string, attrs ...any)
func (l *Logger) Info(msg string, attrs ...any)
func (l *Logger) Warn(msg string, attrs ...any)
func (l *Logger) Error(msg string, attrs ...any)
func (l *Logger) Fatal(msg string, attrs ...any)

// Context-aware methods
func (l *Logger) DebugCtx(ctx context.Context, msg string, attrs ...any)
func (l *Logger) InfoCtx(ctx context.Context, msg string, attrs ...any)
func (l *Logger) WarnCtx(ctx context.Context, msg string, attrs ...any)
func (l *Logger) ErrorCtx(ctx context.Context, msg string, attrs ...any)

// With creates a child logger with additional attributes
func (l *Logger) With(attrs ...any) *Logger
```

### Example Output

**JSON Format** (`LOG_FORMAT=json`):
```json
{"time":"2026-02-24T10:30:00Z","level":"INFO","msg":"Adding item to cart","service":"cart","cart_id":"abc123","product_id":"prod456","quantity":2}
```

**Text Format** (`LOG_FORMAT=text`):
```
2026-02-24T10:30:00Z INFO Adding item to cart service=cart cart_id=abc123 product_id=prod456 quantity=2
```

---

## Migration Phases

### Phase 1: Foundation (Priority)

**Goal:** Create the logging infrastructure and test with cart service

**Steps:**

1. **Create `internal/platform/logging/config.go`**
   - Define `Config` struct with `Level` and `Format` fields
   - Implement `LoadConfig() (*Config, error)` to read from environment
   - Add validation for LOG_LEVEL and LOG_FORMAT values
   - Map LOG_LEVEL strings to `slog.Level`

2. **Create `internal/platform/logging/logger.go`**
   - Define `Logger` struct wrapping `*slog.Logger`
   - Implement `New(serviceName string) *Logger`
   - Read configuration and create appropriate handler (JSON/text)
   - Add service name as default attribute on all logs
   - Implement all log level methods: Debug, Info, Warn, Error, Fatal
   - Implement context-aware methods: DebugCtx, InfoCtx, etc.
   - Implement `With(attrs ...any)` for child loggers

3. **Create `internal/platform/logging/context.go`**
   - Define context key types for request-scoped logging
   - Example: `contextKey string = "logging.ctx.key"`

4. **Update Kubernetes ConfigMap**
   - Add `LOG_FORMAT: "json"` to `platform-configmap-for-services.yaml`

5. **Migrate `cmd/cart/main.go`**
   - Import `go-shopping-poc/internal/platform/logging`
   - Replace `log.Printf("[INFO] Cart: ...")` with `logger.Info("...", "key", value)`
   - Replace `log.Fatalf(...)` with `logger.Fatal(...)`

6. **Migrate cart domain code** (for full POC):
   - `internal/service/cart/*.go` - ~40 statements
   - `internal/service/cart/eventhandlers/*.go` - ~15 statements

**Deliverable:** Cart service running with slog, JSON output verified

**Estimated Statements:** ~70

---

### Phase 2: Migrate HTTP Services

**Goal:** Migrate remaining HTTP services with Kubernetes deployments

**Services:**

| Service | Files | Statements | Priority |
|---------|-------|------------|----------|
| order | main.go, service, repository, handlers | ~80 | High |
| product | main.go, service, repository, handlers | ~90 | High |
| product-admin | main.go, service, handlers | ~50 | High |
| customer | main.go, service, handlers | ~70 | High |
| eventreader | main.go, handlers | ~40 | Medium |

**Steps for each service:**

1. Update `cmd/{service}/main.go`:
   - Import logging package
   - Initialize logger with service name
   - Replace all `log.Printf`/`log.Fatalf` with appropriate level

2. Update `internal/service/{service}/*.go`:
   - Pass logger through service constructor
   - Replace all logging statements

3. Update `internal/service/{service}/eventhandlers/*.go`:
   - Pass logger through handler constructor
   - Replace all logging statements

**Estimated Statements:** ~330

---

### Phase 3: Migrate Platform Code

**Goal:** Migrate shared infrastructure code used by all services

**Files by Component:**

| Component | Files | Statements |
|-----------|-------|------------|
| database | postgresql.go, provider.go, health.go | ~35 |
| outbox | publisher.go, writer.go, providers/*.go | ~35 |
| event/bus | handler.go, eventbus.go | ~15 |
| sse | hub.go, handler.go, provider.go | ~30 |
| storage | provider.go, minio/client.go | ~15 |
| websocket | websocket.go | ~5 |
| config | loader.go | ~5 |
| auth | jwt_validator.go | ~2 |
| cors | provider.go, cors.go | ~3 |
| csv | parser.go | ~2 |
| downloader | downloader.go, provider.go | ~3 |
| service | base.go, interface.go | ~2 |

**Total:** ~152 statements

**Strategy:** Since platform code is used by multiple services, the logger should be injected via constructor. Each platform component that needs logging should accept a `*logging.Logger` in its constructor or provider.

**Steps:**

1. **Update provider functions** to accept logger parameter:
   ```go
   func NewDatabaseProvider(cfg *Config, logger *logging.Logger) (*DatabaseProvider, error)
   ```

2. **Update main.go** of each service to pass its logger to platform providers

3. **Replace logging statements** in platform code with logger calls

**Estimated Statements:** ~152

---

### Phase 4: Migrate Remaining Services

**Goal:** Migrate services without Kubernetes deployment files

| Service | Files | Notes |
|---------|-------|-------|
| websocket | cmd/websocket/main.go | No K8s deploy |
| product-loader | cmd/product-loader/main.go | Batch job |

**Steps:**

1. Follow same pattern as Phase 2
2. Document environment variables needed if running outside K8s

**Estimated Statements:** ~30

---

### Phase 5: Cleanup

**Goal:** Remove all old logging code

**Steps:**

1. **Audit for remaining `log.` calls:**
   ```bash
   grep -r "log\." --include="*.go" cmd/ internal/
   ```

2. **Remove `log` import** from all files that no longer use it

3. **Remove `log.SetFlags` calls** from all main.go files

4. **Verify build:**
   ```bash
   go build ./...
   ```

5. **Run tests** to ensure nothing broke:
   ```bash
   go test ./...
   ```

**Deliverable:** Zero references to `log` package (except perhaps in rare edge cases)

---

## Migration Checklist

### Per-File Pattern

**Before:**
```go
import "log"

func someFunction() {
    log.Printf("[DEBUG] Cart: Adding item to cart %s: product_id=%s", cartID, productID)
    log.Printf("[ERROR] Cart: failed to get cart %s: %v", cartID, err)
}
```

**After:**
```go
import "go-shopping-poc/internal/platform/logging"

func someFunction(logger *logging.Logger) {
    logger.Debug("Adding item to cart",
        "cart_id", cartID,
        "product_id", productID,
    )
    logger.Error("failed to get cart",
        "cart_id", cartID,
        "error", err,
    )
}
```

### Main.go Pattern

**Before:**
```go
import "log"

func main() {
    log.SetFlags(log.LstdFlags)
    log.Printf("[INFO] Cart: Cart service started...")
    
    cfg, err := cart.LoadConfig()
    if err != nil {
        log.Fatalf("Cart: Failed to load config: %v", err)
    }
    // ...
}
```

**After:**
```go
import (
    "go-shopping-poc/internal/platform/logging"
)

func main() {
    logger := logging.New("cart")
    logger.Info("Cart service started")
    
    cfg, err := cart.LoadConfig()
    if err != nil {
        logger.Fatal("Failed to load config", "error", err)
    }
    // ...
}
```

---

## Log Level Mapping

| Old Prefix | New Level | Usage |
|------------|-----------|-------|
| `[DEBUG]` | `Debug` | Detailed diagnostic information |
| `[INFO]` | `Info` | General operational information |
| `[WARN]` | `Warn` | Warning conditions |
| `[ERROR]` | `Error` | Error conditions |
| `[FATAL]` | `Fatal` | Critical conditions causing shutdown |

---

## Attribute Naming Convention

Use lowercase with underscores (snake_case) for consistency with JSON:

| Old Format | New Format |
|------------|------------|
| `cartID` | `cart_id` |
| `productID` | `product_id` |
| `OrderNumber` | `order_number` |
| `totalAmount` | `total_amount` |

---

## Context-Scoped Logging

For HTTP handlers, use context to enable request correlation:

```go
func (h *Handler) GetCart(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    // All logs in this request will have trace_id
    h.logger.InfoCtx(ctx, "GetCart request received", "path", r.URL.Path)
    
    cart, err := h.service.GetCart(ctx, cartID)
    if err != nil {
        h.logger.ErrorCtx(ctx, "Failed to get cart", "cart_id", cartID, "error", err)
        // ...
    }
}
```

---

## Rollback Plan

If issues arise:

1. **Revert to previous log package**: Keep `log` imports alongside slog during transition
2. **Environment variable toggle**: Set `LOG_FORMAT=text` and `LOG_LEVEL=debug` for debugging
3. **Gradual rollout**: Deploy to one service at a time with careful monitoring

---

## Success Criteria

1. All ~609 log statements migrated to slog
2. All services output JSON logs by default
3. Service name appears on every log entry
4. Zero remaining imports of `log` package (except where absolutely necessary)
5. Build passes with `go build ./...`
6. Tests pass with `go test ./...`
7. Kubernetes deployments use centralized LOG_FORMAT config

---

## Appendix: Files Requiring Changes

### Phase 1 - Foundation
- `internal/platform/logging/config.go` (NEW)
- `internal/platform/logging/logger.go` (NEW)
- `internal/platform/logging/context.go` (NEW)
- `deploy/k8s/platform/config/platform-configmap-for-services.yaml` (MODIFY)
- `cmd/cart/main.go` (MODIFY)
- `internal/service/cart/*.go` (MODIFY)
- `internal/service/cart/eventhandlers/*.go` (MODIFY)

### Phase 2 - HTTP Services
- `cmd/order/main.go` (MODIFY)
- `cmd/product/main.go` (MODIFY)
- `cmd/product-admin/main.go` (MODIFY)
- `cmd/customer/main.go` (MODIFY)
- `cmd/eventreader/main.go` (MODIFY)
- `internal/service/order/*.go` (MODIFY)
- `internal/service/product/*.go` (MODIFY)
- `internal/service/customer/*.go` (MODIFY)
- `internal/service/eventreader/*.go` (MODIFY)

### Phase 3 - Platform
- `internal/platform/database/postgresql.go` (MODIFY)
- `internal/platform/database/provider.go` (MODIFY)
- `internal/platform/outbox/publisher.go` (MODIFY)
- `internal/platform/outbox/writer.go` (MODIFY)
- `internal/platform/outbox/providers/*.go` (MODIFY)
- `internal/platform/event/bus/kafka/handler.go` (MODIFY)
- `internal/platform/event/bus/kafka/eventbus.go` (MODIFY)
- `internal/platform/sse/hub.go` (MODIFY)
- `internal/platform/sse/handler.go` (MODIFY)
- `internal/platform/sse/provider.go` (MODIFY)
- `internal/platform/storage/provider.go` (MODIFY)
- `internal/platform/storage/minio/client.go` (MODIFY)
- `internal/platform/websocket/websocket.go` (MODIFY)
- `internal/platform/config/loader.go` (MODIFY)
- `internal/platform/auth/jwt_validator.go` (MODIFY)
- `internal/platform/cors/*.go` (MODIFY)
- `internal/platform/csv/parser.go` (MODIFY)
- `internal/platform/downloader/*.go` (MODIFY)
- `internal/platform/service/*.go` (MODIFY)

### Phase 4 - Remaining Services
- `cmd/websocket/main.go` (MODIFY)
- `cmd/product-loader/main.go` (MODIFY)

### Phase 5 - Cleanup
- All files (AUDIT)
