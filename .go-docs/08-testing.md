# Testing

This document describes testing patterns used in the project, including unit tests, mocking, table-driven tests, and test organization.

## Overview

The project follows Go testing conventions with:
- Standard `testing` package
- Table-driven tests
- Mock implementations for unit tests
- Black box testing (external test packages)
- Parallel test execution

## Test Organization

### File Naming

Test files follow Go conventions:

```
service/customer/
├── service.go          # Production code
├── service_test.go     # Unit tests
├── repository.go       # Production code
├── repository_test.go  # Unit tests
└── ...
```

**Pattern:** `*_test.go` suffix

### Package Naming

Use external test packages for black box testing:

```go
// service_test.go
package customer_test  // Note: _test suffix

import (
    "testing"
    
    "go-shopping-poc/internal/service/customer"
)
```

**Benefits:**
- Tests public API only
- Cannot access unexported functions
- Simulates real usage
- Cleaner separation

**Reference:** `internal/service/customer/service_test.go`

## Unit Testing Patterns

### Mock Repository

Create mock implementations for testing:

```go
// internal/service/customer/service_test.go

type mockCustomerRepository struct {
    insertCustomerFunc        func(ctx context.Context, customer *customer.Customer) error
    getCustomerByIDFunc       func(ctx context.Context, customerID string) (*customer.Customer, error)
    getCustomerByEmailFunc    func(ctx context.Context, email string) (*customer.Customer, error)
    patchCustomerFunc         func(ctx context.Context, customerID string, patchData *customer.PatchCustomerRequest) error
}

func (m *mockCustomerRepository) InsertCustomer(ctx context.Context, customer *customer.Customer) error {
    return m.insertCustomerFunc(ctx, customer)
}

func (m *mockCustomerRepository) GetCustomerByID(ctx context.Context, customerID string) (*customer.Customer, error) {
    return m.getCustomerByIDFunc(ctx, customerID)
}

func (m *mockCustomerRepository) GetCustomerByEmail(ctx context.Context, email string) (*customer.Customer, error) {
    return m.getCustomerByEmailFunc(ctx, email)
}

func (m *mockCustomerRepository) PatchCustomer(ctx context.Context, customerID string, patchData *customer.PatchCustomerRequest) error {
    return m.patchCustomerFunc(ctx, customerID, patchData)
}

// Implement remaining interface methods...
```

**Key patterns:**
1. Define mock struct with function fields
2. Implement all interface methods
3. Each method calls the corresponding function field
4. Allow test-specific behavior injection

**Reference:** `internal/service/customer/service_test.go`

### Service Testing with Mocks

Test service logic with injected mocks:

```go
func TestCustomerService_CreateCustomerSuccess(t *testing.T) {
    t.Parallel()

    // Setup mock
    mockRepo := &mockCustomerRepository{
        insertCustomerFunc: func(ctx context.Context, cust *customer.Customer) error {
            return nil
        },
    }

    // Create service with mock
    mockInfra := &customer.CustomerInfrastructure{}
    svc := customer.NewCustomerServiceWithRepo(mockRepo, mockInfra, &customer.Config{})

    // Test
    testCustomer := &customer.Customer{
        Username: "testuser",
        Email:    "test@example.com",
    }

    err := svc.CreateCustomer(context.Background(), testCustomer)
    if err != nil {
        t.Errorf("CreateCustomer() failed: %v", err)
    }
}
```

**Key patterns:**
1. Create mock with test-specific behavior
2. Use `NewCustomerServiceWithRepo()` for injection
3. Test expected behavior
4. Assert no errors (or expected errors)

**Reference:** `internal/service/customer/service_test.go`

## Table-Driven Tests

### Basic Structure

```go
func TestCustomerService_ValidatePatchData(t *testing.T) {
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
        tt := tt  // Capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            
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
```

**Key patterns:**
1. Define test struct with input and expected output
2. Create slice of test cases
3. Use `t.Run()` with test name
4. Capture range variable to avoid closure issues
5. Use `t.Parallel()` for concurrent execution

### Complex Table-Driven Tests

```go
func TestCustomerService_GetCustomer(t *testing.T) {
    t.Parallel()

    ctx := context.Background()
    
    tests := []struct {
        name           string
        customerID     string
        mockSetup      func(*mockCustomerRepository)
        wantCustomer   *customer.Customer
        wantErr        bool
        wantErrType    error
    }{
        {
            name:       "customer found",
            customerID: "cust-123",
            mockSetup: func(m *mockCustomerRepository) {
                m.getCustomerByIDFunc = func(ctx context.Context, id string) (*customer.Customer, error) {
                    return &customer.Customer{
                        CustomerID: "cust-123",
                        Email:      "test@example.com",
                    }, nil
                }
            },
            wantCustomer: &customer.Customer{
                CustomerID: "cust-123",
                Email:      "test@example.com",
            },
            wantErr: false,
        },
        {
            name:       "customer not found",
            customerID: "cust-999",
            mockSetup: func(m *mockCustomerRepository) {
                m.getCustomerByIDFunc = func(ctx context.Context, id string) (*customer.Customer, error) {
                    return nil, customer.ErrCustomerNotFound
                }
            },
            wantCustomer: nil,
            wantErr:      true,
            wantErrType:  customer.ErrCustomerNotFound,
        },
        {
            name:       "database error",
            customerID: "cust-123",
            mockSetup: func(m *mockCustomerRepository) {
                m.getCustomerByIDFunc = func(ctx context.Context, id string) (*customer.Customer, error) {
                    return nil, errors.New("connection failed")
                }
            },
            wantCustomer: nil,
            wantErr:      true,
        },
    }

    for _, tt := range tests {
        tt := tt
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            
            mockRepo := &mockCustomerRepository{}
            tt.mockSetup(mockRepo)
            
            mockInfra := &customer.CustomerInfrastructure{}
            svc := customer.NewCustomerServiceWithRepo(mockRepo, mockInfra, &customer.Config{})

            got, err := svc.GetCustomer(ctx, tt.customerID)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("GetCustomer() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if tt.wantErrType != nil && !errors.Is(err, tt.wantErrType) {
                t.Errorf("GetCustomer() error type = %v, want %v", err, tt.wantErrType)
            }
            
            if !reflect.DeepEqual(got, tt.wantCustomer) {
                t.Errorf("GetCustomer() = %v, want %v", got, tt.wantCustomer)
            }
        })
    }
}
```

## Parallel Testing

### Using t.Parallel()

```go
func TestSuite(t *testing.T) {
    t.Parallel()  // Run this suite in parallel with other suites
    
    tests := []struct{...}{...}
    
    for _, tt := range tests {
        tt := tt  // Essential: capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()  // Run individual tests in parallel
            
            // Test code here
        })
    }
}
```

**Important:** Always capture the range variable: `tt := tt`

## Test Helpers

### String Pointer Helper

```go
func strPtr(s string) *string {
    return &s
}
```

### Context Helper

```go
func testContext() context.Context {
    return context.Background()
}
```

## Entity Testing

### Testing Validation Logic

```go
func TestCustomer_Validate(t *testing.T) {
    tests := []struct {
        name      string
        customer  customer.Customer
        wantError bool
    }{
        {
            name: "valid customer",
            customer: customer.Customer{
                Email:    "test@example.com",
                Username: "testuser",
            },
            wantError: false,
        },
        {
            name: "missing email",
            customer: customer.Customer{
                Username: "testuser",
            },
            wantError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.customer.Validate()
            if (err != nil) != tt.wantError {
                t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
            }
        })
    }
}
```

**Reference:** `internal/service/customer/entity_test.go`

## Configuration Testing

### Testing Config Validation

```go
func TestConfig_Validate(t *testing.T) {
    tests := []struct {
        name    string
        config  customer.Config
        wantErr bool
    }{
        {
            name: "valid config",
            config: customer.Config{
                DatabaseURL: "postgresql://localhost/test",
                ServicePort: "8080",
                WriteTopic:  "customer-events",
            },
            wantErr: false,
        },
        {
            name: "missing database URL",
            config: customer.Config{
                ServicePort: "8080",
                WriteTopic:  "customer-events",
            },
            wantErr: true,
        },
        {
            name: "missing service port",
            config: customer.Config{
                DatabaseURL: "postgresql://localhost/test",
                WriteTopic:  "customer-events",
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

## Integration Testing

### Database Integration Tests

For tests requiring real database:

```go
// +build integration

package customer_test

import (
    "testing"
    
    "go-shopping-poc/internal/platform/database"
    "go-shopping-poc/internal/service/customer"
)

func TestCustomerRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Setup test database
    db, err := database.NewDatabaseProvider("postgresql://test@localhost/testdb")
    if err != nil {
        t.Fatalf("Failed to connect to test database: %v", err)
    }
    
    repo := customer.NewCustomerRepository(db.GetDatabase(), nil)
    
    // Run tests
    ctx := context.Background()
    
    cust := &customer.Customer{
        CustomerID: "test-123",
        Email:      "test@example.com",
    }
    
    err = repo.InsertCustomer(ctx, cust)
    if err != nil {
        t.Errorf("InsertCustomer() error = %v", err)
    }
    
    // Cleanup
    // ...
}
```

**Running integration tests:**
```bash
go test -tags=integration ./...
go test -short ./...  # Skip integration tests
```

## Test Coverage

### Running Tests with Coverage

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/service/customer/...
```

### Makefile Targets

```makefile
customer-test:
	go test -v ./cmd/customer/... ./internal/service/customer/...

test:
	go test -v ./...

test-coverage:
	go test -cover ./...
```

**Reference:** `Makefile` (lines 130-140)

## Best Practices

### DO:
- ✅ Use table-driven tests for multiple scenarios
- ✅ Run tests in parallel with `t.Parallel()`
- ✅ Use external test packages (`_test` suffix)
- ✅ Create mock implementations for unit tests
- ✅ Test both success and failure cases
- ✅ Use `t.Run()` for subtest naming
- ✅ Keep tests independent (no shared state)
- ✅ Clean up resources in tests
- ✅ Skip long tests with `testing.Short()`

### DON'T:
- ❌ Write tests in the same package (use `_test`)
- ❌ Forget to capture range variable in parallel tests
- ❌ Share state between tests
- ❌ Test unexported functions directly
- ❌ Ignore errors in tests
- ❌ Write tests without assertions
- ❌ Use sleep/wait in tests (use channels or sync)

## Common Testing Patterns

### Testing Error Conditions

```go
func TestOperation_ErrorCases(t *testing.T) {
    tests := []struct {
        name      string
        setup     func() *mockRepository
        wantErr   error
    }{
        {
            name: "not found error",
            setup: func() *mockRepository {
                return &mockRepository{
                    getFunc: func() (*Entity, error) {
                        return nil, ErrNotFound
                    },
                }
            },
            wantErr: ErrNotFound,
        },
        {
            name: "database error",
            setup: func() *mockRepository {
                return &mockRepository{
                    getFunc: func() (*Entity, error) {
                        return nil, errors.New("connection refused")
                    },
                }
            },
            wantErr: nil, // Any error is acceptable
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            repo := tt.setup()
            svc := NewService(repo)
            
            _, err := svc.Operation()
            
            if tt.wantErr != nil {
                if !errors.Is(err, tt.wantErr) {
                    t.Errorf("Operation() error = %v, want %v", err, tt.wantErr)
                }
            } else if err == nil {
                t.Error("Operation() expected error, got nil")
            }
        })
    }
}
```

### Testing with Timeouts

```go
func TestOperation_WithTimeout(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()
    
    err := svc.Operation(ctx)
    if !errors.Is(err, context.DeadlineExceeded) {
        t.Errorf("Expected timeout error, got: %v", err)
    }
}
```

### Testing Goroutines

```go
func TestAsyncOperation(t *testing.T) {
    done := make(chan error, 1)
    
    go func() {
        done <- svc.AsyncOperation()
    }()
    
    select {
    case err := <-done:
        if err != nil {
            t.Errorf("AsyncOperation() error = %v", err)
        }
    case <-time.After(time.Second):
        t.Error("AsyncOperation() timed out")
    }
}
```
