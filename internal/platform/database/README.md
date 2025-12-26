# Database Abstraction Layer

This package provides a generic database abstraction layer for PostgreSQL that implements Clean Architecture principles. It provides reusable database infrastructure that supports connection pooling, health checks, transactions, and proper error handling.

## Features

- **Generic Interface**: Clean abstraction over database operations
- **PostgreSQL Support**: Optimized implementation for PostgreSQL
- **Connection Pooling**: Configurable connection pool management
- **Health Checks**: Built-in health monitoring and validation
- **Transaction Support**: Full transaction lifecycle management
- **Error Handling**: Structured error handling with proper logging
- **Comprehensive Testing**: Full test coverage with integration tests

## Architecture

```
internal/platform/database/
├── interface.go           # Database and Tx interfaces
├── postgresql.go          # PostgreSQL implementation
├── provider.go            # Database provider implementation
├── health.go              # Health check and monitoring utilities
├── config.go              # Database configuration (existing)
├── database_test.go       # Comprehensive unit tests
└── README.md             # This documentation
```

## Usage

### Basic Setup

```go
import "go-shopping-poc/internal/platform/database"

// Load platform connection configuration
connConfigPtr, err := config.LoadConfig[database.ConnectionConfig]("platform-database")
if err != nil {
    return err
}
connConfig := *connConfigPtr

// Service provides database URL (from environment or service config)
databaseURL := "postgres://user:pass@host:port/db?sslmode=disable"

// Create database client with service URL and platform connection config
db, err := database.NewPostgreSQLClient(databaseURL, connConfig)
if err != nil {
    return err
}

// Connect to database
ctx := context.Background()
err = db.Connect(ctx)
if err != nil {
    return err
}
defer db.Close()
```

### Provider Pattern

For dependency injection and clean architecture, use the `DatabaseProvider`:

```go
import "go-shopping-poc/internal/platform/database"

// Create a database provider (handles config loading and connection)
provider, err := database.NewDatabaseProvider("postgres://user:pass@host:port/db?sslmode=disable")
if err != nil {
    return err
}

// Get the database instance from the provider
db := provider.GetDatabase()

// Use the database for operations
rows, err := db.Query(ctx, "SELECT * FROM users")
if err != nil {
    return err
}
defer rows.Close()
```

The provider encapsulates:
- Platform database configuration loading
- PostgreSQL client creation
- Database connection establishment
- Proper error handling and logging

### Basic Operations

```go
// Query for multiple rows
rows, err := db.Query(ctx, "SELECT id, name FROM users WHERE active = $1", true)
if err != nil {
    return err
}
defer rows.Close()

for rows.Next() {
    var id int
    var name string
    err = rows.Scan(&id, &name)
    if err != nil {
        return err
    }
    fmt.Printf("User: %d - %s\n", id, name)
}

// Query for single row
var userName string
row := db.QueryRow(ctx, "SELECT name FROM users WHERE id = $1", userID)
err = row.Scan(&userName)

// Execute commands
result, err := db.Exec(ctx, "UPDATE users SET last_login = $1 WHERE id = $2", time.Now(), userID)
if err != nil {
    return err
}

rowsAffected, _ := result.RowsAffected()
fmt.Printf("Updated %d users\n", rowsAffected)
```

### Transactions

```go
// Begin transaction
tx, err := db.BeginTx(ctx, nil)
if err != nil {
    return err
}
defer tx.Rollback() // Will be ignored if committed

// Execute operations in transaction
_, err = tx.Exec(ctx, "INSERT INTO users (name, email) VALUES ($1, $2)", "John", "john@example.com")
if err != nil {
    return err
}

_, err = tx.Exec(ctx, "INSERT INTO user_profiles (user_id, bio) VALUES ($1, $2)", 123, "Hello world")
if err != nil {
    return err
}

// Commit transaction
err = tx.Commit()
if err != nil {
    return err
}
```

### Health Checks

```go
// Check health status
status := db.CheckHealth(ctx)
if !status.Available {
    log.Printf("Database unhealthy: %v", status.Error)
    return status.Error
}
log.Printf("Database healthy (latency: %v)", status.Latency)

// Quick health check
if !db.IsHealthy(ctx) {
    return errors.New("database is not healthy")
}

// Validate connection comprehensively
err = db.ValidateConnection(ctx)
if err != nil {
    return fmt.Errorf("connection validation failed: %w", err)
}
```

### Connection Configuration

```go
// Service provides database URL
databaseURL := "postgres://user:pass@host:port/db?sslmode=disable"

// Platform provides connection configuration
connConfigPtr, err := config.LoadConfig[database.ConnectionConfig]("platform-database")
if err != nil {
    return err
}
connConfig := *connConfigPtr

// Create client with service URL and platform config
db, err := database.NewPostgreSQLClient(databaseURL, connConfig)

// Alternative: Use custom configuration for specific service needs
customConfig := database.ConnectionConfig{
    MaxOpenConns:        50,
    MaxIdleConns:        10,
    ConnMaxLifetime:     10 * time.Minute,
    ConnMaxIdleTime:     5 * time.Minute,
    ConnectTimeout:      60 * time.Second,
    QueryTimeout:        30 * time.Second,
    HealthCheckInterval: 60 * time.Second,
    HealthCheckTimeout:  10 * time.Second,
}

db, err := database.NewPostgreSQLClient(databaseURL, customConfig)
```

## Configuration

The database package follows Clean Architecture principles with clear separation of concerns:

- **Services** provide database URLs (connection endpoints, credentials, database names)
- **Platform** provides connection configuration (pooling, timeouts, health checks)

### Platform Connection Configuration

Platform-level connection configuration is loaded from environment variables using the `platform-database` config prefix:

```env
PLATFORM_DATABASE_MAX_OPEN_CONNS=25
PLATFORM_DATABASE_MAX_IDLE_CONNS=25
PLATFORM_DATABASE_CONN_MAX_LIFETIME=5m
PLATFORM_DATABASE_CONN_MAX_IDLE_TIME=5m
PLATFORM_DATABASE_CONNECT_TIMEOUT=30s
PLATFORM_DATABASE_QUERY_TIMEOUT=30s
PLATFORM_DATABASE_HEALTH_CHECK_INTERVAL=30s
PLATFORM_DATABASE_HEALTH_CHECK_TIMEOUT=5s
```

### Service Database URLs

Services provide their own database URLs, typically loaded from service-specific environment variables:

```env
# Customer service
CUSTOMER_DATABASE_URL=postgres://customersuser:customerspass@localhost:5432/customersdb?sslmode=disable

# Product service
PRODUCT_DATABASE_URL=postgres://productsuser:productspass@localhost:5432/productsdb?sslmode=disable
```

This separation ensures:
- **Clean Architecture**: Platform provides infrastructure, services provide domain-specific configuration
- **Flexibility**: Different services can connect to different databases
- **Security**: Database credentials are managed at the service level
- **Reusability**: Platform connection config can be shared across services

## Error Handling

The package provides structured error handling:

- **Connection errors**: Issues with establishing or maintaining database connections
- **Query errors**: Problems executing SQL queries or commands
- **Transaction errors**: Issues with transaction lifecycle management
- **Health check errors**: Database availability and performance issues

All errors are properly wrapped with context and logged with appropriate severity levels.

## Testing

### Unit Tests

Run the database unit tests:

```bash
go test ./internal/platform/database/
```

### Integration Tests

The tests use the existing test infrastructure from `internal/testutils`. Make sure you have a test database available:

```bash
# Set DATABASE_URL environment variable
export DATABASE_URL="postgres://user:pass@localhost:5432/testdb?sslmode=disable"

# Run tests
go test ./internal/platform/database/ -v
```

## Clean Architecture Compliance

This package follows Clean Architecture principles with clear separation between platform and service concerns:

### Platform Layer (Infrastructure)
- **Connection Configuration**: Platform provides reusable connection pooling, timeouts, and health check settings
- **Generic Interface**: Allows different database implementations (PostgreSQL, MySQL, etc.)
- **Infrastructure Abstraction**: Provides the "HOW" - database connectivity and management

### Service Layer (Domain)
- **Database URLs**: Services provide domain-specific connection endpoints and credentials
- **Business Logic**: Services implement domain-specific database operations
- **Configuration**: Services manage their own database schema and connection details

### Key Benefits
- **Dependency Inversion**: Services depend on platform abstractions, not concretions
- **Separation of Concerns**: Database infrastructure separated from business logic
- **Flexibility**: Services can connect to different databases while sharing platform infrastructure
- **Testability**: Comprehensive testing with mocked dependencies where appropriate
- **Reusability**: Platform configuration can be shared across multiple services

## Future Extensions

The design supports easy extension for:

- **Additional Databases**: MySQL, SQLite implementations
- **Advanced Features**: Connection retry logic, read/write splitting
- **Metrics**: Prometheus metrics integration
- **Tracing**: Distributed tracing support
- **Migrations**: Database migration management

## Dependencies

- `github.com/jmoiron/sqlx`: Extended SQL operations
- `github.com/jackc/pgx/v5/stdlib`: PostgreSQL driver
- Standard library: `database/sql`, `context`, `time`, `log`</content>
<parameter name="filePath">internal/platform/database/README.md