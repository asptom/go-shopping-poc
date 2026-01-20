# Detailed Testing Plan: Customer Service

## Overview

This plan creates **targeted, business-focused tests** for customer service across two layers:
1. **Bootstrap layer** (`/cmd/customer/`) - Application startup and health monitoring
2. **Business logic layer** (`internal/service/customer/`) - Domain entities, services, and validation

### Testing Philosophy
- ✅ **Test business behavior** - what users/production observe
- ✅ **Test domain rules** - validation and business constraints
- ✅ **Minimal, focused tests** - no 2000+ line monstrosities
- ✅ **Idiomatic Go** - test files alongside source (`*_test.go`)
- ✅ **Programmatic setup** - `os.Setenv()` for configuration, no `.env` files
- ❌ **Don't test implementation details** - wiring, infrastructure, stdlib
- ❌ **Don't test external dependencies** - database, Kafka, platform code

---

## File Structure After Test Implementation

```
cmd/customer/
├── main.go                      (185 lines, refactor: extract healthHandler)
├── main_test.go                  ✅ NEW: Health endpoint tests (~50 lines)
└── README.md                     ✅ NEW: Test documentation (~40 lines)

internal/service/customer/
├── config.go                     (39 lines)
├── config_test.go                ✅ NEW: Configuration tests (~120 lines)
├── entity.go                     (200 lines)
├── entity_test.go                ✅ NEW: Entity validation tests (~180 lines)
├── service.go                    (357 lines)
├── service_test.go               ✅ NEW: Service logic tests (~250 lines)
├── validation.go                 (101 lines)
├── validation_test.go            ✅ NEW: Event validation tests (~80 lines)
├── repository.go                 (1,391 lines) - ❌ NOT tested
├── handler.go                   (373 lines) - ❌ NOT tested
├── service_utils.go              (50 lines) - ❌ NOT tested
└── event_utils.go               (122 lines) - ❌ NOT tested
```

**Total test code**: ~720 lines (vs. 2000+ lines from LLMs)
**Production code changes**: 5 lines (extract healthHandler from main.go)

---

## Detailed Test File Breakdown

### 1. `/cmd/customer/main_test.go` (~50 lines)

**Purpose**: Test health check endpoint for Kubernetes monitoring

**Refactor Required** (5 lines in `main.go`):
```go
// Extract from main.go:111-115
func healthHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte(`{"status":"ok"}`))
}
```

**Tests**:
```go
package main

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestHealthHandler(t *testing.T) {
    t.Parallel()

    req := httptest.NewRequest("GET", "/health", nil)
    rr := httptest.NewRecorder()

    healthHandler(rr, req)

    if status := rr.Code; status != http.StatusOK {
        t.Errorf("returned wrong status: got %v want %v", status, http.StatusOK)
    }

    if rr.Header().Get("Content-Type") != "application/json" {
        t.Error("Content-Type should be application/json")
    }

    expected := `{"status":"ok"}`
    if rr.Body.String() != expected {
        t.Errorf("unexpected body: got %v want %v", rr.Body.String(), expected)
    }
}

func TestHealthHandlerDifferentMethods(t *testing.T) {
    t.Parallel()

    methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

    for _, method := range methods {
        t.Run(method, func(t *testing.T) {
            req := httptest.NewRequest(method, "/health", nil)
            rr := httptest.NewRecorder()

            healthHandler(rr, req)

            if rr.Code != http.StatusOK {
                t.Errorf("%s returned %d, want %d", method, rr.Code, http.StatusOK)
            }
        })
    }
}
```

**Why valid**:
- Health checks are critical for Kubernetes liveness/readiness probes
- Tests user-observable behavior (HTTP response)
- No infrastructure dependencies
- Fast to run

---

### 2. `/internal/service/customer/config_test.go` (~120 lines)

**Purpose**: Test configuration loading and validation logic

**Helper Functions**:
```go
package customer_test

import (
    "os"
    "testing"

    "go-shopping-poc/internal/service/customer"
)

// setValidTestEnv sets valid environment variables for testing
func setValidTestEnv(t *testing.T) {
    t.Helper()

    os.Setenv("CUSTOMER_DB_URL", "postgres://localhost:5432/test")
    os.Setenv("CUSTOMER_SERVICE_PORT", "8080")
    os.Setenv("CUSTOMER_WRITE_TOPIC", "CustomerEvents")
}

// cleanupTestEnv unsets test environment variables
func cleanupTestEnv() {
    os.Unsetenv("CUSTOMER_DB_URL")
    os.Unsetenv("CUSTOMER_SERVICE_PORT")
    os.Unsetenv("CUSTOMER_WRITE_TOPIC")
    os.Unsetenv("CUSTOMER_GROUP")
}
```

**Tests**:
```go
func TestLoadConfigSuccess(t *testing.T) {
    t.Parallel()

    setValidTestEnv(t)
    defer cleanupTestEnv()

    cfg, err := customer.LoadConfig()
    if err != nil {
        t.Fatalf("LoadConfig() failed: %v", err)
    }

    if cfg.DatabaseURL == "" || cfg.ServicePort == "" || cfg.WriteTopic == "" {
        t.Error("required fields should be set")
    }
}

func TestConfigValidateSuccess(t *testing.T) {
    t.Parallel()

    cfg := &customer.Config{
        DatabaseURL: "postgres://localhost:5432/test",
        ServicePort: "8080",
        WriteTopic:  "CustomerEvents",
    }

    if err := cfg.Validate(); err != nil {
        t.Errorf("valid config should pass validation: %v", err)
    }
}

func TestConfigValidateMissingRequired(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name      string
        cfg       *customer.Config
        wantError bool
    }{
        {
            name:      "missing database URL",
            cfg:       &customer.Config{ServicePort: "8080", WriteTopic: "CustomerEvents"},
            wantError: true,
        },
        {
            name:      "missing service port",
            cfg:       &customer.Config{DatabaseURL: "postgres://localhost:5432/test", WriteTopic: "CustomerEvents"},
            wantError: true,
        },
        {
            name:      "missing write topic",
            cfg:       &customer.Config{DatabaseURL: "postgres://localhost:5432/test", ServicePort: "8080"},
            wantError: true,
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            err := tt.cfg.Validate()
            if (err != nil) != tt.wantError {
                t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
            }
        })
    }
}
```

**Why valid**:
- Tests business rules (what config is required)
- Catches configuration errors before production
- Prevents service startup failures
- No external dependencies

---

### 3. `/internal/service/customer/entity_test.go` (~180 lines)

**Purpose**: Test domain entity validation logic

**Tests**:
```go
package customer_test

import (
    "testing"

    "go-shopping-poc/internal/service/customer"
)

func TestCustomerValidateSuccess(t *testing.T) {
    t.Parallel()

    cust := &customer.Customer{
        Username:  "testuser",
        Email:     "test@example.com",
        CustomerStatus: "active",
    }

    if err := cust.Validate(); err != nil {
        t.Errorf("valid customer should pass validation: %v", err)
    }
}

func TestCustomerValidateUsername(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name      string
        username  string
        wantError bool
    }{
        {"empty username", "", true},
        {"whitespace only", "   ", true},
        {"too short", "ab", true},
        {"valid username", "testuser", false},
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            cust := &customer.Customer{Username: tt.username}
            err := cust.Validate()

            if (err != nil) != tt.wantError {
                t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
            }
        })
    }
}

func TestCustomerValidateEmail(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name      string
        email     string
        wantError bool
    }{
        {"missing @", "testexample.com", true},
        {"valid email", "test@example.com", false},
        {"empty email", "", false}, // Email is optional
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            cust := &customer.Customer{
                Username: "testuser",
                Email:    tt.email,
            }
            err := cust.Validate()

            if (err != nil) != tt.wantError {
                t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
            }
        })
    }
}

func TestCustomerValidateStatus(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name      string
        status    string
        wantError bool
    }{
        {"active status", "active", false},
        {"inactive status", "inactive", false},
        {"suspended status", "suspended", false},
        {"invalid status", "pending", true},
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            cust := &customer.Customer{
                Username:       "testuser",
                CustomerStatus: tt.status,
            }
            err := cust.Validate()

            if (err != nil) != tt.wantError {
                t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
            }
        })
    }
}

func TestCustomerIsActive(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name     string
        status   string
        expected bool
    }{
        {"active customer", "active", true},
        {"inactive customer", "inactive", false},
        {"suspended customer", "suspended", false},
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            cust := &customer.Customer{CustomerStatus: tt.status}

            if got := cust.IsActive(); got != tt.expected {
                t.Errorf("IsActive() = %v, want %v", got, tt.expected)
            }
        })
    }
}

func TestCustomerFullName(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name     string
        first    string
        last     string
        expected string
    }{
        {"both names", "John", "Doe", "John Doe"},
        {"first only", "John", "", "John"},
        {"last only", "", "Doe", "Doe"},
        {"no names", "", "", ""},
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            cust := &customer.Customer{
                FirstName: tt.first,
                LastName:  tt.last,
            }

            if got := cust.FullName(); got != tt.expected {
                t.Errorf("FullName() = %v, want %v", got, tt.expected)
            }
        })
    }
}

func TestAddressValidate(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name      string
        address   *customer.Address
        wantError bool
    }{
        {
            name: "valid shipping address",
            address: &customer.Address{
                AddressType: "shipping",
                Address1:   "123 Main St",
                City:       "Springfield",
                State:      "IL",
                Zip:        "62701",
            },
            wantError: false,
        },
        {
            name: "valid billing address",
            address: &customer.Address{
                AddressType: "billing",
                Address1:   "456 Oak Ave",
                City:       "Springfield",
                State:      "IL",
                Zip:        "62702",
            },
            wantError: false,
        },
        {
            name: "invalid address type",
            address: &customer.Address{
                AddressType: "invalid",
                Address1:   "123 Main St",
                City:       "Springfield",
                State:      "IL",
                Zip:        "62701",
            },
            wantError: true,
        },
        {
            name: "missing address line 1",
            address: &customer.Address{
                AddressType: "shipping",
                City:       "Springfield",
                State:      "IL",
                Zip:        "62701",
            },
            wantError: true,
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            err := tt.address.Validate()

            if (err != nil) != tt.wantError {
                t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
            }
        })
    }
}

func TestCreditCardValidate(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name      string
        card      *customer.CreditCard
        wantError bool
    }{
        {
            name: "valid visa card",
            card: &customer.CreditCard{
                CardType:       "visa",
                CardNumber:     "4111111111111111",
                CardHolderName: "John Doe",
                CardExpires:    "12/25",
                CardCVV:        "123",
            },
            wantError: false,
        },
        {
            name: "invalid card type",
            card: &customer.CreditCard{
                CardType:       "discover", // Invalid type in code
                CardNumber:     "6011111111111117",
                CardHolderName: "John Doe",
                CardExpires:    "12/25",
                CardCVV:        "123",
            },
            wantError: true,
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            err := tt.card.Validate()

            if (err != nil) != tt.wantError {
                t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
            }
        })
    }
}
```

**Why valid**:
- Tests domain rules (what's valid for customers, addresses, credit cards)
- Tests business constraints (status values, card types, address types)
- No infrastructure dependencies
- Pure business logic

---

### 4. `/internal/service/customer/service_test.go` (~250 lines)

**Purpose**: Test business logic in service layer (orchestration, transformation, validation)

**Mock Interface Required**:
```go
// Create mock repository for testing
type mockCustomerRepository struct {
    createCustomerFunc func(ctx context.Context, customer *customer.Customer) error
    getCustomerByIDFunc func(ctx context.Context, customerID string) (*customer.Customer, error)
    // Add other methods as needed...
}

func (m *mockCustomerRepository) InsertCustomer(ctx context.Context, cust *customer.Customer) error {
    return m.createCustomerFunc(ctx, cust)
}

func (m *mockCustomerRepository) GetCustomerByID(ctx context.Context, customerID string) (*customer.Customer, error) {
    return m.getCustomerByIDFunc(ctx, customerID)
}

// Implement all other CustomerRepository interface methods...
```

**Tests**:
```go
package customer_test

import (
    "context"
    "testing"

    "go-shopping-poc/internal/service/customer"
)

func TestCustomerServiceCreateCustomerSuccess(t *testing.T) {
    t.Parallel()

    mockRepo := &mockCustomerRepository{
        createCustomerFunc: func(ctx context.Context, cust *customer.Customer) error {
            return nil
        },
    }

    mockInfra := &customer.CustomerInfrastructure{
        // Mock infrastructure components
    }

    svc := customer.NewCustomerServiceWithRepo(mockRepo, mockInfra, &customer.Config{})

    testCustomer := &customer.Customer{
        Username: "testuser",
        Email:    "test@example.com",
    }

    err := svc.CreateCustomer(context.Background(), testCustomer)
    if err != nil {
        t.Errorf("CreateCustomer() failed: %v", err)
    }
}

func TestCustomerServiceCreateCustomerValidation(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name      string
        customer  *customer.Customer
        wantError bool
    }{
        {
            name:      "missing username",
            customer:  &customer.Customer{Email: "test@example.com"},
            wantError: true,
        },
        {
            name:      "missing email",
            customer:  &customer.Customer{Username: "testuser"},
            wantError: true,
        },
        {
            name: "invalid address in customer",
            customer: &customer.Customer{
                Username: "testuser",
                Email:    "test@example.com",
                Addresses: []customer.Address{
                    {AddressType: "invalid", Address1: "123 Main St"},
                },
            },
            wantError: true,
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := &mockCustomerRepository{}
            mockInfra := &customer.CustomerInfrastructure{}
            svc := customer.NewCustomerServiceWithRepo(mockRepo, mockInfra, &customer.Config{})

            err := svc.CreateCustomer(context.Background(), tt.customer)

            if (err != nil) != tt.wantError {
                t.Errorf("CreateCustomer() error = %v, wantError %v", err, tt.wantError)
            }
        })
    }
}

func TestCustomerServicePatchCustomerSuccess(t *testing.T) {
    t.Parallel()

    existingCustomer := &customer.Customer{
        CustomerID:    "123e4567-e89b-12d3-a456-426614174000",
        Username:       "oldusername",
        Email:          "old@example.com",
    }

    mockRepo := &mockCustomerRepository{
        getCustomerByIDFunc: func(ctx context.Context, customerID string) (*customer.Customer, error) {
            return existingCustomer, nil
        },
        patchCustomerFunc: func(ctx context.Context, customerID string, patchData *customer.PatchCustomerRequest) error {
            return nil
        },
    }

    mockInfra := &customer.CustomerInfrastructure{}
    svc := customer.NewCustomerServiceWithRepo(mockRepo, mockInfra, &customer.Config{})

    patchData := &customer.PatchCustomerRequest{
        UserName: strPtr("newusername"),
    }

    err := svc.PatchCustomer(context.Background(), existingCustomer.CustomerID, patchData)
    if err != nil {
        t.Errorf("PatchCustomer() failed: %v", err)
    }
}

func TestCustomerServiceValidatePatchData(t *testing.T) {
    t.Parallel()

    tests := []struct {
        name      string
        patchData *customer.PatchCustomerRequest
        wantError bool
    }{
        {
            name:      "nil patch data",
            patchData: nil,
            wantError: true,
        },
        {
            name: "valid patch",
            patchData: &customer.PatchCustomerRequest{
                UserName: strPtr("newusername"),
            },
            wantError: false,
        },
        {
            name: "invalid UUID in address ID",
            patchData: &customer.PatchCustomerRequest{
                DefaultShippingAddressID: strPtr("invalid-uuid"),
            },
            wantError: true,
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := &mockCustomerRepository{}
            mockInfra := &customer.CustomerInfrastructure{}
            svc := customer.NewCustomerServiceWithRepo(mockRepo, mockInfra, &customer.Config{})

            err := svc.ValidatePatchData(tt.patchData)

            if (err != nil) != tt.wantError {
                t.Errorf("ValidatePatchData() error = %v, wantError %v", err, tt.wantError)
            }
        })
    }
}

func TestCustomerServiceTransformAddressesFromPatch(t *testing.T) {
    t.Parallel()

    mockRepo := &mockCustomerRepository{}
    mockInfra := &customer.CustomerInfrastructure{}
    svc := customer.NewCustomerServiceWithRepo(mockRepo, mockInfra, &customer.Config{})

    patchAddresses := []customer.PatchAddressRequest{
        {
            AddressType: "shipping",
            FirstName:   "John",
            LastName:    "Doe",
            Address1:    "123 Main St",
            City:        "Springfield",
            State:       "IL",
            Zip:         "62701",
        },
    }

    addresses := svc.TransformAddressesFromPatch(patchAddresses)

    if len(addresses) != 1 {
        t.Fatalf("expected 1 address, got %d", len(addresses))
    }

    if addresses[0].AddressType != "shipping" {
        t.Errorf("expected address_type shipping, got %s", addresses[0].AddressType)
    }
}

func TestCustomerServiceTransformCreditCardsFromPatch(t *testing.T) {
    t.Parallel()

    mockRepo := &mockCustomerRepository{}
    mockInfra := &customer.CustomerInfrastructure{}
    svc := customer.NewCustomerServiceWithRepo(mockRepo, mockInfra, &customer.Config{})

    patchCards := []customer.PatchCreditCardRequest{
        {
            CardType:       "visa",
            CardNumber:     "4111111111111111",
            CardHolderName: "John Doe",
            CardExpires:    "12/25",
            CardCVV:        "123",
        },
    }

    cards := svc.TransformCreditCardsFromPatch(patchCards)

    if len(cards) != 1 {
        t.Fatalf("expected 1 credit card, got %d", len(cards))
    }

    if cards[0].CardType != "visa" {
        t.Errorf("expected card_type visa, got %s", cards[0].CardType)
    }
}

// Helper function to create string pointers
func strPtr(s string) *string {
    return &s
}
```

**Why valid**:
- Tests business orchestration (service coordinates repository calls)
- Tests validation and transformation logic
- Tests business rules (what's valid for patch operations)
- Mocks repository, not database
- Tests behavior, not implementation details

**Note**: This file uses a hand-written mock. No external mock library needed.

---

### 5. `/internal/service/customer/validation_test.go` (~80 lines)

**Purpose**: Test event validation logic

**Tests**:
```go
package customer_test

import (
    "context"
    "testing"

    "go-shopping-poc/internal/service/customer"
    "go-shopping-poc/internal/contracts/events"
)

func TestValidateCustomerEventSuccess(t *testing.T) {
    t.Parallel()

    validator := customer.NewCustomerEventValidator()

    evt := events.NewCustomerCreatedEvent("customer-123", map[string]string{
        "username": "testuser",
    })

    err := validator.ValidateCustomerEvent(context.Background(), *evt)
    if err != nil {
        t.Errorf("valid event should pass validation: %v", err)
    }
}

func TestValidateCustomerEventMissingCustomerID(t *testing.T) {
    t.Parallel()

    validator := customer.NewCustomerEventValidator()

    evt := events.NewCustomerCreatedEvent("", nil)

    err := validator.ValidateCustomerEvent(context.Background(), *evt)
    if err == nil {
        t.Error("expected error for missing customer ID")
    }
}

func TestValidateCustomerEventInvalidEventType(t *testing.T) {
    t.Parallel()

    validator := customer.NewCustomerEventValidator()

    payload := events.CustomerEventPayload{
        CustomerID: "customer-123",
        EventType:  "invalid.event.type",
    }

    evt := &events.CustomerEvent{
        EventType:    "invalid.event.type",
        EventPayload: payload,
    }

    err := validator.ValidateCustomerEvent(context.Background(), *evt)
    if err == nil {
        t.Error("expected error for invalid event type")
    }
}

func TestValidateCustomerEventResourceIDRequired(t *testing.T) {
    t.Parallel()

    validator := customer.NewCustomerEventValidator()

    evt := events.NewAddressAddedEvent("customer-123", "", nil)

    err := validator.ValidateCustomerEvent(context.Background(), *evt)
    if err == nil {
        t.Error("expected error for missing resource ID on address event")
    }
}

func TestValidateCustomerEventPayloadSuccess(t *testing.T) {
    t.Parallel()

    validator := customer.NewCustomerEventValidator()

    payload := events.CustomerEventPayload{
        CustomerID: "customer-123",
        EventType:  events.CustomerCreated,
    }

    err := validator.ValidateCustomerEventPayload(context.Background(), payload)
    if err != nil {
        t.Errorf("valid payload should pass validation: %v", err)
    }
}

func TestValidateCustomerEventPayloadMissingCustomerID(t *testing.T) {
    t.Parallel()

    validator := customer.NewCustomerEventValidator()

    payload := events.CustomerEventPayload{
        CustomerID: "",
        EventType:  events.CustomerCreated,
    }

    err := validator.ValidateCustomerEventPayload(context.Background(), payload)
    if err == nil {
        t.Error("expected error for missing customer ID")
    }
}
```

**Why valid**:
- Tests domain rules (what events are valid)
- Tests event-specific validation (resource ID requirements)
- Tests business constraints (allowed event types)
- No infrastructure dependencies

---

## What We WILL NOT Test (and Why)

### ❌ `/internal/service/customer/repository.go` (1,391 lines)

**Why**:
- Contains data access logic (SQL queries, transactions)
- Would require database integration or heavy mocking
- Integration test responsibility, not unit test
- Database layer should be tested in integration tests

**Testing Strategy**: If needed, create integration tests with test database container.

---

### ❌ `/internal/service/customer/handler.go` (373 lines)

**Why**:
- HTTP handlers are thin wrappers around service layer
- Would require mocking service (testing implementation details)
- Handler logic is simple: decode JSON → call service → encode response
- Behavior tested by integration/end-to-end tests

**Testing Strategy**: If you want handler tests, add later. They would test HTTP status codes, error responses, request parsing.

---

### ❌ `/internal/service/customer/service_utils.go` (50 lines)

**Why**:
- Contains simple utility functions (logging, error wrapping)
- Low business value
- Testing logging/wrapping doesn't provide meaningful coverage
- No complex logic to test

---

### ❌ `/internal/service/customer/event_utils.go` (122 lines)

**Why**:
- Contains wrapper functions for event publishing
- No business logic, just calls platform code
- Platform code should be tested separately
- Testing wrappers provides no additional value

---

## Implementation Dependencies

### Required (Already in go.mod)
- `testing` (Go stdlib)
- `context` (Go stdlib)
- `os` (Go stdlib)

### Not Required
- No mock libraries needed (hand-written mocks for service layer)
- No `.env` file libraries (use `os.Setenv()` directly)
- No test assertion libraries (use `t.Errorf`, `t.Fatal()`)

---

## Documentation Files

### `/cmd/customer/README.md` (~40 lines)

```markdown
# Customer Service Bootstrap Tests

## What We Test

### Health Endpoint (`main_test.go`)
- ✅ Returns HTTP 200 OK status
- ✅ Returns correct JSON response body
- ✅ Sets correct Content-Type header
- ✅ Works with different HTTP methods

**Why**: Critical for Kubernetes liveness/readiness probes and production monitoring.

## What We DON'T Test

- ❌ main() function (wiring/glue code, exits on failure)
- ❌ HTTP server startup (stdlib, integration concern)
- ❌ Graceful shutdown (platform-specific, OS behavior)
- ❌ Database/Event bus creation (external dependencies)
- ❌ Route registration (implementation detail)
- ❌ Signal handling (OS-level behavior)

## Philosophy

Tests follow principle: **Test business behavior, not implementation details.**

## Running Tests

```bash
# Run all customer bootstrap tests
go test -v ./cmd/customer/...

# Run specific test
go test -v ./cmd/customer/... -run TestHealthHandler
```
```

---

## Execution Phases

### Phase 1: Bootstrap Layer Tests
1. Refactor `main.go`: Extract `healthHandler` function (5 lines)
2. Create `/cmd/customer/main_test.go` (~50 lines)
3. Create `/cmd/customer/README.md` (~40 lines)
4. Run tests: `go test -v ./cmd/customer/...`

### Phase 2: Configuration Tests
1. Create `/internal/service/customer/config_test.go` (~120 lines)
2. Run tests: `go test -v ./internal/service/customer/... -run TestLoadConfig`

### Phase 3: Entity Tests
1. Create `/internal/service/customer/entity_test.go` (~180 lines)
2. Run tests: `go test -v ./internal/service/customer/... -run TestCustomer`

### Phase 4: Service Tests
1. Create `/internal/service/customer/service_test.go` (~250 lines)
2. Add mock repository struct to test file
3. Run tests: `go test -v ./internal/service/customer/... -run TestCustomerService`

### Phase 5: Validation Tests
1. Create `/internal/service/customer/validation_test.go` (~80 lines)
2. Run tests: `go test -v ./internal/service/customer/... -run TestValidateCustomerEvent`

### Phase 6: Full Test Suite
1. Run all tests: `go test -v ./cmd/customer/... ./internal/service/customer/...`
2. Check coverage: `go test -cover ./...`
3. Fix any failures
4. Commit to repository

---

## Summary

### Files to Create
1. `/cmd/customer/main_test.go` (~50 lines)
2. `/cmd/customer/README.md` (~40 lines)
3. `/internal/service/customer/config_test.go` (~120 lines)
4. `/internal/service/customer/entity_test.go` (~180 lines)
5. `/internal/service/customer/service_test.go` (~250 lines)
6. `/internal/service/customer/validation_test.go` (~80 lines)

### Files to Modify
1. `/cmd/customer/main.go` - Extract `healthHandler` function (5 lines)

### Total Impact
- **Test code**: ~720 lines (business-focused, minimal)
- **Production code changes**: 5 lines (no behavior change)
- **Test coverage**: Business logic, validation, configuration
- **Execution time**: <5 seconds (no infrastructure)
- **Dependencies**: Zero new packages

### Key Characteristics
- ✅ **Business-focused**: Tests what users/production care about
- ✅ **Idiomatic Go**: `*_test.go` alongside source, `os.Setenv()` for config
- ✅ **Minimal**: ~720 lines vs. 2000+ lines from LLMs
- ✅ **Fast**: <5 seconds execution, no infrastructure required
- ✅ **Maintainable**: Clear purpose, no complex mocking

---

## Decisions Made During Planning

### 1. Test Scope
**Approved**: Core business logic only
**Excluded**: Repository (data access), handlers (HTTP wrappers), utilities

### 2. Mock Strategy
**Approved**: Hand-written mocks in test files
**Not using**: `go:generate`, external mock libraries
**Reason**: Explicit, minimal dependencies, idiomatic Go

### 3. Environment Variables
**Approved**: Programmatic `os.Setenv()` in tests
**Not using**: `.env.test` file, `godotenv` library
**Reason**: Self-contained tests, explicit dependencies, idiomatic Go

### 4. Test File Structure
**Approved**: `*_test.go` alongside source files (idiomatic Go)
**Not using**: `/tests/` subdirectory
**Reason**: Works with Go tooling, IDE integration, standard practice

---

## Running Tests During Implementation

```bash
# Run all customer service tests
go test -v ./cmd/customer/... ./internal/service/customer/...

# Run with coverage
go test -cover ./cmd/customer/... ./internal/service/customer/...

# Run specific test file
go test -v ./internal/service/customer/... -run TestCustomerValidate

# Run specific test case
go test -v ./internal/service/customer/... -run TestCustomerValidateUsername/empty_username

# Run tests in parallel (faster)
go test -parallel=4 ./...
```

---

## Notes for Implementation

1. **Import package naming**: Use `package main_test` for bootstrap tests and `package customer_test` for business logic tests (black-box testing)

2. **Helper functions**: Mark with `t.Helper()` for proper error reporting line numbers

3. **Parallel tests**: Use `t.Parallel()` at the start of each test to enable concurrent execution

4. **Table-driven tests**: Use struct slices for multiple test cases with clear names

5. **Mock completeness**: Only implement methods actually called by tests, not full interface

6. **UUID handling**: For tests, use valid UUID strings or generate with `uuid.New()` in test setup

7. **Context usage**: Always pass `context.Background()` or test-specific context to service methods

8. **Test cleanup**: Use `defer` statements for cleanup (environment variables, resources)
