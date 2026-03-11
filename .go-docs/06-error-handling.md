# Error Handling

This document describes error handling patterns used throughout the project, including error types, wrapping, and structured responses.

## Overview

Go uses explicit error handling with the convention: "Always check errors immediately." This project follows idiomatic Go error handling with additional patterns for domain-specific errors and structured error responses.

## Core Principles

### 1. Always Check Errors

```go
// Good - immediate error check
result, err := someOperation()
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
// Continue with success path

// Bad - ignoring error
result, _ := someOperation()  // Never do this
```

### 2. Indent Error Flow

Keep the success path at minimum indentation:

```go
// Good - error handling first, success unindented
if err != nil {
    return fmt.Errorf("failed to load config: %w", err)
}
// Success path continues here without extra indentation

// Bad - success path nested inside if
if err == nil {
    // Success logic nested one level
    processResult(result)
} else {
    return err
}
```

### 3. Wrap Errors with Context

Always add context when returning errors up the call stack:

```go
// Good - provides context about what failed
customer, err := repo.GetCustomerByID(ctx, id)
if err != nil {
    return fmt.Errorf("failed to get customer %s: %w", id, err)
}

// Bad - loses context
if err != nil {
    return err
}
```

## Error Types

### Sentinel Errors

Define package-level errors for domain conditions:

```go
// internal/service/customer/repository.go
var (
    ErrCustomerNotFound   = errors.New("customer not found")
    ErrAddressNotFound    = errors.New("address not found")
    ErrDuplicateEmail     = errors.New("email already exists")
    ErrInvalidUUID        = errors.New("invalid UUID format")
    ErrDatabaseOperation  = errors.New("database operation failed")
    ErrTransactionFailed  = errors.New("transaction failed")
)

// internal/platform/outbox/errors.go
var (
    ErrWriteFailed         = errors.New("outbox: write operation failed")
    ErrPublishFailed       = errors.New("outbox: publish operation failed")
    ErrTransactionRollover = errors.New("outbox: transaction rolled back unexpectedly")
    ErrInvalidEvent        = errors.New("outbox: invalid event")
)
```

**Naming convention:** `Err` + descriptive PascalCase name

### Custom Error Types

For errors needing additional context:

```go
// internal/platform/service/interface.go
type ServiceError struct {
    Service string
    Op      string
    Err     error
}

func (e *ServiceError) Error() string {
    return fmt.Sprintf("service %s: operation %s: %v", e.Service, e.Op, e.Err)
}

func (e *ServiceError) Unwrap() error {
    return e.Err
}
```

**Usage:**
```go
return &ServiceError{
    Service: "customer",
    Op:      "CreateCustomer",
    Err:     err,
}
```

### Error Wrapping Helper

```go
// internal/platform/outbox/errors.go
func WrapWithContext(err error, message string) error {
    if err == nil {
        return nil
    }
    return fmt.Errorf("%s: %w", message, err)
}
```

## Error Checking Patterns

### Using errors.Is

Check for specific error types with `errors.Is`:

```go
import "errors"

customer, err := repo.GetCustomerByID(ctx, id)
if err != nil {
    if errors.Is(err, ErrCustomerNotFound) {
        // Handle not found
        httperr.NotFound(w, "Customer not found")
        return
    }
    // Handle other errors
    httperr.Internal(w, "Internal error")
    return
}
```

### Error Chain Inspection

With wrapped errors, you can check at any level:

```go
// Repository returns sentinel error
return ErrCustomerNotFound

// Service wraps it
return fmt.Errorf("failed to get customer: %w", err)

// Handler can still check original
if errors.Is(err, ErrCustomerNotFound) {
    // Handle not found
}
```

### Error Messages

#### Error Message Format

```go
// Format: "failed to [action]: [context]: %w"
return fmt.Errorf("failed to create customer: %s: %w", customer.Email, err)

// Format: "[component]: [operation] failed: %w"
return fmt.Errorf("repository: insert customer failed: %w", err)
```

#### Good vs Bad Messages

```go
// Good - specific and actionable
fmt.Errorf("failed to connect to database at %s: %w", dsn, err)
fmt.Errorf("invalid customer email format: %s", email)

// Bad - vague
fmt.Errorf("error occurred")
fmt.Errorf("something went wrong")
```

## HTTP Error Responses

### Structured Error Response

```go
// internal/platform/httperr
httperr.InvalidRequest(w, "Invalid request body")
httperr.Validation(w, "Email already exists")
httperr.NotFound(w, "Customer not found")
httperr.Internal(w, "Failed to retrieve customer")
```

**Usage in handlers:**
```go
func (h *CustomerHandler) GetCustomer(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    
    customer, err := h.service.GetCustomer(r.Context(), id)
    if err != nil {
        if errors.Is(err, ErrCustomerNotFound) {
            httperr.NotFound(w, "Customer not found")
            return
        }
        httperr.Internal(w, "Failed to retrieve customer")
        return
    }

    if err := httpx.WriteJSON(w, http.StatusOK, customer); err != nil {
        httperr.Internal(w, "Encode customer response failed")
        return
    }
}
```

**Reference:** `internal/platform/httperr`, `internal/platform/httpx`

## Repository Error Handling

### Converting SQL Errors

```go
func (r *customerRepository) GetCustomerByID(ctx context.Context, id string) (*Customer, error) {
    query := `SELECT ... FROM customers WHERE customer_id = $1`
    
    var customer Customer
    err := r.db.GetContext(ctx, &customer, query, id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrCustomerNotFound  // Convert to domain error
        }
        return nil, fmt.Errorf("failed to query customer: %w", err)
    }
    
    return &customer, nil
}
```

**Key pattern:** Convert `sql.ErrNoRows` to domain-specific `ErrNotFound`

### Transaction Errors

```go
func (r *customerRepository) CreateWithEvent(ctx context.Context, customer *Customer) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    
    committed := false
    defer func() {
        if !committed {
            if rbErr := tx.Rollback(); rbErr != nil {
                // Log rollback error but don't override original error
                logger.Error("Transaction rollback failed", "operation", "create_with_event", "error", rbErr.Error())
            }
        }
    }()
    
    // ... perform operations ...
    
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }
    committed = true
    
    return nil
}
```

## Service Layer Error Handling

### Input Validation

```go
func (s *CustomerService) CreateCustomer(ctx context.Context, customer *Customer) error {
    // Validate input
    if err := customer.Validate(); err != nil {
        return fmt.Errorf("invalid customer data: %w", err)
    }
    
    // Check business rules
    existing, err := s.repo.GetCustomerByEmail(ctx, customer.Email)
    if err != nil && !errors.Is(err, ErrCustomerNotFound) {
        return fmt.Errorf("failed to check existing customer: %w", err)
    }
    if existing != nil {
        return ErrDuplicateEmail  // Return domain error directly
    }
    
    // ... continue with creation
}
```

### Error Translation

```go
func (s *CustomerService) GetCustomer(ctx context.Context, id string) (*Customer, error) {
    if id == "" {
        return nil, ErrInvalidInput
    }
    
    customer, err := s.repo.GetCustomerByID(ctx, id)
    if err != nil {
        // Pass through domain errors
        if errors.Is(err, ErrCustomerNotFound) {
            return nil, err
        }
        // Wrap infrastructure errors
        return nil, fmt.Errorf("failed to retrieve customer: %w", err)
    }
    
    return customer, nil
}
```

## Event Handler Error Handling

### Retryable vs Non-Retryable

```go
func (h *OnCustomerCreated) Handle(ctx context.Context, event events.Event) error {
    log := h.logger.With("operation", "handle_customer_created")
    // Type check - non-retryable (don't retry bad events)
    customerEvent, ok := event.(events.CustomerEvent)
    if !ok {
        log.Warn("Unexpected event type", "actual_type", fmt.Sprintf("%T", event))
        return nil  // Acknowledge but don't retry
    }
    
    // Event type filtering
    if customerEvent.EventType != events.CustomerCreated {
        log.Debug("Ignore event type", "event_type", customerEvent.EventType)
        return nil
    }
    
    // Process - retryable errors
    if err := h.process(ctx, customerEvent); err != nil {
        return fmt.Errorf("failed to process customer created: %w", err)
    }
    
    return nil
}
```

**Guidelines:**
- Return `nil` for bad input (don't retry)
- Return error for transient failures (will be retried)
- Log appropriately for each case

## Logging Errors

### Log Levels

```go
logger.Error("Start service failed", "operation", "startup", "error", err.Error())
logger.Error("Process order failed", "operation", "process_order", "order_id", orderID, "error", err.Error())
logger.Warn("Skip invalid event", "operation", "handle_event", "event_type", eventType, "error", err.Error())
logger.Debug("Retry publish event", "operation", "publish_event", "attempt", attempt, "error", err.Error())
```

### Structured Logging (Recommended)

```go
logger.With("component", "customer_service").
    Error("Create customer failed", "operation", "create_customer", "customer_id", customerID, "error", err.Error())
```

## Panic Recovery

### Service Entry Points

```go
// cmd/customer/main.go
func main() {
    defer func() {
        if r := recover(); r != nil {
            logger.Error("Panic recovered", "operation", "main", "panic", r, "stack", string(debug.Stack()))
            os.Exit(1)
        }
    }()
    
    // ... service initialization ...
}
```

### HTTP Handlers

```go
func RecoveryMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if r := recover(); r != nil {
                logger.Error("Panic in handler", "operation", "http_recovery", "panic", r, "stack", string(debug.Stack()))
                httperr.Internal(w, "Internal Server Error")
            }
        }()
        
        next.ServeHTTP(w, r)
    })
}
```

## Best Practices

### DO:
- ✅ Always check errors immediately
- ✅ Wrap errors with context using `%w`
- ✅ Define domain-specific sentinel errors
- ✅ Convert infrastructure errors to domain errors
- ✅ Use `errors.Is()` for error checking
- ✅ Provide structured error responses for APIs
- ✅ Log errors with appropriate levels
- ✅ Handle panics at entry points
- ✅ Distinguish retryable from non-retryable errors

### DON'T:
- ❌ Ignore errors with `_`
- ❌ Return raw infrastructure errors to clients
- ❌ Use `%v` instead of `%w` when wrapping
- ❌ Panic for expected error conditions
- ❌ Create error messages that are too vague
- ❌ Check errors with `==` (use `errors.Is`)
- ❌ Let panics crash the service

## Common Patterns Summary

```go
// Basic error check
if err != nil {
    return fmt.Errorf("context: %w", err)
}

// Sentinel error check
if err != nil {
    if errors.Is(err, ErrNotFound) {
        return err  // Pass through
    }
    return fmt.Errorf("context: %w", err)
}

// Custom error type
return &ServiceError{Service: "name", Op: "operation", Err: err}

// Error with multiple context
return fmt.Errorf("failed to %s %s: %w", action, target, err)

// Structured HTTP error
httperr.NotFound(w, "Not found")
```
