# Repository Pattern

This document describes the repository pattern used for data access, including interface design, transaction handling, and error patterns.

## Overview

The **Repository Pattern** abstracts database operations behind interfaces, enabling:
- Testability through mock implementations
- Database technology independence
- Transaction management
- Clean separation of data access from business logic

## Repository Interface

### Defining the Interface

Define repository interfaces in the service package alongside the implementation:

```go
// internal/service/customer/repository.go

// Domain-specific errors
var (
    ErrCustomerNotFound   = errors.New("customer not found")
    ErrAddressNotFound    = errors.New("address not found")
    ErrDatabaseOperation  = errors.New("database operation failed")
    ErrTransactionFailed  = errors.New("transaction failed")
    ErrInvalidUUID        = errors.New("invalid UUID format")
)

// CustomerRepository defines the contract for customer data access
type CustomerRepository interface {
    // CRUD operations
    InsertCustomer(ctx context.Context, customer *Customer) error
    GetCustomerByEmail(ctx context.Context, email string) (*Customer, error)
    GetCustomerByID(ctx context.Context, customerID string) (*Customer, error)
    UpdateCustomer(ctx context.Context, customer *Customer) error
    PatchCustomer(ctx context.Context, customerID string, patchData *PatchCustomerRequest) error
    
    // Related entity operations
    AddAddress(ctx context.Context, customerID string, addr *Address) (*Address, error)
    UpdateAddress(ctx context.Context, addressID string, addr *Address) error
    DeleteAddress(ctx context.Context, addressID string) error
}
```

**Key patterns:**
1. Define domain-specific sentinel errors
2. Use `ctx context.Context` as first parameter
3. Return concrete types, not interfaces
4. Use descriptive method names (e.g., `GetCustomerByID` not `Get`)
5. Keep interfaces focused (SRP)

**Reference:** `internal/service/customer/repository.go`

### Implementation Structure

```go
// internal/service/customer/repository.go

// Unexported struct implements the interface
type customerRepository struct {
    db           database.Database
    outboxWriter *outbox.Writer
}

// Exported constructor returns the interface
func NewCustomerRepository(db database.Database, outbox *outbox.Writer) CustomerRepository {
    return &customerRepository{
        db:           db,
        outboxWriter: outbox,
    }
}

// Verify interface compliance at compile time
var _ CustomerRepository = (*customerRepository)(nil)
```

**Key patterns:**
1. Implementation struct is unexported (lowercase)
2. Constructor returns the interface type
3. Dependencies injected via constructor
4. Interface compliance check

## Database Abstraction

### Database Interface

The platform layer provides database abstraction:

```go
// internal/platform/database/interface.go

type Database interface {
    // Connection management
    Connect(ctx context.Context) error
    Close() error
    Ping(ctx context.Context) error
    
    // Basic operations
    Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
    QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
    Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
    
    // Transactions
    BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)
    
    // Statistics
    Stats() sql.DBStats
    
    // SQLX compatibility
    GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
    SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
    
    // Access underlying SQLX (for advanced use)
    DB() *sqlx.DB
}

// Tx represents a database transaction
type Tx interface {
    Context() context.Context
    Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
    QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
    Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
    Commit() error
    Rollback() error
    ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
    NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
}
```

**Key patterns:**
1. Context-first parameter design
2. Transaction interface separate from Database
3. Support for both standard `database/sql` and `jmoiron/sqlx`
4. Context-aware operations

**Reference:** `internal/platform/database/interface.go`

## Transaction Handling

### Transaction Pattern

Use this pattern for operations requiring atomicity:

```go
func (r *customerRepository) insertCustomerWithRelations(
    ctx context.Context,
    customer *Customer,
) error {
    // 1. Begin transaction
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("failed to begin transaction: %w", err)
    }
    
    // 2. Set up deferred rollback
    committed := false
    defer func() {
        if !committed {
            _ = tx.Rollback()
        }
    }()
    
    // 3. Perform all operations within transaction
    if err := r.insertCustomerRecordInTransaction(ctx, tx, customer); err != nil {
        return err
    }
    
    if err := r.insertAddressesInTransaction(ctx, tx, customer); err != nil {
        return err
    }
    
    // 4. Write outbox event (same transaction!)
    evt := events.NewCustomerCreatedEvent(customer.CustomerID, nil)
    if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
        return fmt.Errorf("failed to write event to outbox: %w", err)
    }
    
    // 5. Commit transaction
    if err := tx.Commit(); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }
    committed = true
    
    return nil
}
```

**Critical requirements:**
1. Always use `committed` flag for deferred rollback
2. All database operations must use the transaction (`tx`)
3. Outbox writes must use same transaction
4. Return detailed error messages with wrapped errors
5. Never ignore transaction errors

**Reference:** `internal/service/customer/repository.go` (lines 112-178)

### Transaction Helper Methods

Create helper methods that accept the transaction:

```go
func (r *customerRepository) insertCustomerRecordInTransaction(
    ctx context.Context,
    tx database.Tx,
    customer *Customer,
) error {
    query := `
        INSERT INTO customers (customer_id, username, email, ...)
        VALUES (:customer_id, :username, :email, ...)
    `
    
    _, err := tx.NamedExecContext(ctx, query, customer)
    if err != nil {
        return fmt.Errorf("failed to insert customer: %w", err)
    }
    
    return nil
}
```

**Key patterns:**
1. Accept `database.Tx` as parameter
2. Use `NamedExecContext` for struct binding (sqlx feature)
3. Return wrapped errors for context

## Error Handling

### Domain-Specific Errors

Define sentinel errors for domain conditions:

```go
var (
    ErrCustomerNotFound   = errors.New("customer not found")
    ErrAddressNotFound    = errors.New("address not found")
    ErrDuplicateEmail     = errors.New("email already exists")
    ErrInvalidUUID        = errors.New("invalid UUID format")
)
```

### Error Wrapping

Always wrap errors with context:

```go
// Good - provides context
if err != nil {
    return fmt.Errorf("failed to get customer by ID %s: %w", customerID, err)
}

// Bad - loses context
if err != nil {
    return err
}
```

### Error Type Checking

Use `errors.Is` for sentinel errors:

```go
// In handler or service layer
customer, err := repo.GetCustomerByID(ctx, id)
if err != nil {
    if errors.Is(err, ErrCustomerNotFound) {
        // Return 404
        http.Error(w, "Customer not found", http.StatusNotFound)
        return
    }
    // Return 500
    http.Error(w, "Internal error", http.StatusInternalServerError)
    return
}
```

## Query Patterns

### SELECT with Struct Scanning

```go
func (r *customerRepository) GetCustomerByID(ctx context.Context, customerID string) (*Customer, error) {
    query := `
        SELECT customer_id, username, email, first_name, last_name
        FROM customers
        WHERE customer_id = $1
    `
    
    var customer Customer
    err := r.db.GetContext(ctx, &customer, query, customerID)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, ErrCustomerNotFound
        }
        return nil, fmt.Errorf("failed to get customer: %w", err)
    }
    
    return &customer, nil
}
```

**Key patterns:**
1. Use `GetContext` for single-row queries (sqlx)
2. Check for `sql.ErrNoRows` explicitly
3. Return domain-specific sentinel errors

### SELECT Multiple Rows

```go
func (r *customerRepository) GetCustomers(ctx context.Context) ([]Customer, error) {
    query := `SELECT customer_id, username, email FROM customers`
    
    var customers []Customer
    err := r.db.SelectContext(ctx, &customers, query)
    if err != nil {
        return nil, fmt.Errorf("failed to get customers: %w", err)
    }
    
    return customers, nil
}
```

**Key patterns:**
1. Use `SelectContext` for multiple rows
2. Slice will be empty (not nil) if no rows

### INSERT with Return

```go
func (r *customerRepository) AddAddress(
    ctx context.Context,
    customerID string,
    addr *Address,
) (*Address, error) {
    query := `
        INSERT INTO addresses (address_id, customer_id, street, city, ...)
        VALUES (:address_id, :customer_id, :street, :city, ...)
        RETURNING address_id, created_at
    `
    
    addr.AddressID = uuid.New().String()
    addr.CustomerID = customerID
    
    rows, err := r.db.NamedQueryContext(ctx, query, addr)
    if err != nil {
        return nil, fmt.Errorf("failed to add address: %w", err)
    }
    defer rows.Close()
    
    if rows.Next() {
        err = rows.Scan(&addr.AddressID, &addr.CreatedAt)
        if err != nil {
            return nil, fmt.Errorf("failed to scan address: %w", err)
        }
    }
    
    return addr, nil
}
```

### UPDATE

```go
func (r *customerRepository) UpdateCustomer(
    ctx context.Context,
    customer *Customer,
) error {
    query := `
        UPDATE customers
        SET username = :username,
            email = :email,
            updated_at = :updated_at
        WHERE customer_id = :customer_id
    `
    
    result, err := r.db.NamedExecContext(ctx, query, customer)
    if err != nil {
        return fmt.Errorf("failed to update customer: %w", err)
    }
    
    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    
    if rows == 0 {
        return ErrCustomerNotFound
    }
    
    return nil
}
```

**Key patterns:**
1. Check `RowsAffected()` for updates
2. Return `ErrNotFound` if no rows updated

### DELETE

```go
func (r *customerRepository) DeleteAddress(ctx context.Context, addressID string) error {
    query := `DELETE FROM addresses WHERE address_id = $1`
    
    result, err := r.db.ExecContext(ctx, query, addressID)
    if err != nil {
        return fmt.Errorf("failed to delete address: %w", err)
    }
    
    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    
    if rows == 0 {
        return ErrAddressNotFound
    }
    
    return nil
}
```

## Complex Queries

### JOIN Queries

```go
func (r *customerRepository) GetCustomerWithAddresses(
    ctx context.Context,
    customerID string,
) (*Customer, []Address, error) {
    // Get customer
    customer, err := r.GetCustomerByID(ctx, customerID)
    if err != nil {
        return nil, nil, err
    }
    
    // Get addresses separately (cleaner than complex join)
    addresses, err := r.GetAddressesByCustomerID(ctx, customerID)
    if err != nil {
        return nil, nil, err
    }
    
    return customer, addresses, nil
}
```

**Pattern:** Prefer multiple simple queries over complex JOINs for clarity.

### Pagination

```go
func (r *customerRepository) GetCustomersPaginated(
    ctx context.Context,
    offset, limit int,
) ([]Customer, error) {
    query := `
        SELECT customer_id, username, email
        FROM customers
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2
    `
    
    var customers []Customer
    err := r.db.SelectContext(ctx, &customers, query, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to get customers: %w", err)
    }
    
    return customers, nil
}
```

## Testing Repositories

See 08-testing.md for detailed testing patterns. Key approaches:

1. **Unit tests with mock database:**
   ```go
   type mockDatabase struct {
       getFunc func(ctx context.Context, dest interface{}, query string, args ...interface{}) error
   }
   ```

2. **Integration tests with test database:**
   - Use testcontainers or Docker
   - Test against real PostgreSQL
   - Clean up after each test

**Reference:** `internal/service/customer/service_test.go` for mock patterns

## Best Practices

### DO:
- ✅ Define interfaces in the same package as implementation
- ✅ Use context for all operations
- ✅ Wrap errors with context
- ✅ Use transactions for multi-step operations
- ✅ Write outbox events within transactions
- ✅ Return domain-specific errors
- ✅ Keep interfaces focused (single responsibility)
- ✅ Verify interface compliance: `var _ Interface = (*Type)(nil)`

### DON'T:
- ❌ Return `sql.ErrNoRows` directly (convert to domain error)
- ❌ Mix SQL logic with business logic
- ❌ Ignore transaction errors
- ❌ Use global database connections
- ❌ Write outbox events outside transactions
- ❌ Create repository interfaces with too many methods

## Migration Guide

### Adding a New Repository

1. Define domain errors
2. Create interface with CRUD methods
3. Create unexported implementation struct
4. Implement methods with context
5. Add constructor returning interface
6. Add interface compliance check
7. Write tests with mock database

### Adding a New Query Method

1. Add method signature to interface
2. Implement method on repository struct
3. Use parameterized queries (never string concatenation)
4. Handle `sql.ErrNoRows` explicitly
5. Wrap errors with context
6. Add unit tests
