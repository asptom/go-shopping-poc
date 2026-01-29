// Package database provides a generic database abstraction layer for PostgreSQL.
// It implements Clean Architecture by providing reusable database infrastructure
// that supports connection pooling, health checks, transactions, and proper error handling.
//
// Key interfaces:
//   - Database: Common database operations interface
//   - Tx: Transaction interface for atomic operations
//   - JSON: JSON value type for PostgreSQL JSONB columns
//
// Usage patterns:
//   - Basic operations: Use Database interface for queries and commands
//   - Transactions: Use BeginTx() for atomic operations
//   - Health checks: Use Ping() for connection validation
//   - JSON columns: Use JSON type for PostgreSQL JSONB columns
//
// Example usage:
//
//	// Load platform connection configuration
//	connConfigPtr, err := config.LoadConfig[database.ConnectionConfig]("platform-database")
//	if err != nil {
//	    return err
//	}
//	connConfig := *connConfigPtr
//
//	// Service provides database URL
//	databaseURL := "postgres://user:pass@host:port/db?sslmode=disable"
//
//	// Create database client with service URL and platform config
//	db, err := database.NewPostgreSQLClient(databaseURL, connConfig)
//	if err != nil {
//	    return err
//	}
//
//	// Basic query
//	rows, err := db.Query(ctx, "SELECT * FROM users WHERE id = $1", userID)
//
//	// Transaction
//	tx, err := db.BeginTx(ctx, nil)
//	if err != nil {
//	    return err
//	}
//	defer tx.Rollback()
//
//	// Execute in transaction
//	_, err = tx.Exec(ctx, "INSERT INTO users (name) VALUES ($1)", "John")
//	if err != nil {
//	    return err
//	}
//
//	return tx.Commit()
//
// JSON usage:
//
//	// Define a struct with JSON field
//	type UserProfile struct {
//	    ID       int    `db:"id"`
//	    Name     string `db:"name"`
//	    Settings JSON   `db:"settings"`  // PostgreSQL JSONB column
//	}
//
//	// Insert with JSON data
//	settings := database.JSON{Data: map[string]interface{}{"theme": "dark", "notifications": true}}
//	_, err = db.Exec(ctx, "INSERT INTO user_profiles (name, settings) VALUES ($1, $2)",
//	    "John", settings)
//
//	// Query and scan JSON data
//	var profile UserProfile
//	err = db.QueryRow(ctx, "SELECT id, name, settings FROM user_profiles WHERE id = $1", 1).
//	    Scan(&profile.ID, &profile.Name, &profile.Settings)
//
//	// Access JSON data
//	if settings, ok := profile.Settings.Data.(map[string]interface{}); ok {
//	    theme := settings["theme"].(string)  // "dark"
//	}
package database

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
)

// Database defines the interface for database operations
type Database interface {
	// Connection management
	Connect(ctx context.Context) error
	Close() error
	Ping(ctx context.Context) error

	// Query operations
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)

	// Transaction operations
	BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error)

	// Health and monitoring
	Stats() sql.DBStats

	// SQLX compatibility methods for existing code
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error

	// Direct access to underlying sqlx.DB for repositories that need it
	DB() *sqlx.DB
}

// Tx defines the interface for database transactions
type Tx interface {
	// Context returns the transaction's context for cancellation and tracing
	Context() context.Context

	// Query operations within transaction
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)

	// Transaction control
	Commit() error
	Rollback() error

	// SQLX compatibility methods for existing code
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
}

// HealthStatus represents the health status of the database
type HealthStatus struct {
	Available bool
	Latency   time.Duration
	Error     error
}

// ConnectionConfig defines database connection configuration
type ConnectionConfig struct {
	// Connection parameters
	MaxOpenConns    int           `mapstructure:"database_max_open_conns"`
	MaxIdleConns    int           `mapstructure:"database_max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"database_conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"database_conn_max_idle_time"`

	// Timeout settings
	ConnectTimeout time.Duration `mapstructure:"database_connect_timeout"`
	QueryTimeout   time.Duration `mapstructure:"database_query_timeout"`

	// Health check settings
	HealthCheckInterval time.Duration `mapstructure:"database_health_check_interval"`
	HealthCheckTimeout  time.Duration `mapstructure:"database_health_check_timeout"`
}

// JSON represents a JSON value that can be stored in PostgreSQL JSONB columns.
// It implements sql.Scanner and driver.Valuer interfaces for proper JSON marshaling/unmarshaling.
type JSON struct {
	Data interface{}
}

// Scan implements sql.Scanner interface for reading JSON from database
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		j.Data = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return json.Unmarshal([]byte(value.(string)), &j.Data)
	}

	return json.Unmarshal(bytes, &j.Data)
}

// Value implements driver.Valuer interface for writing JSON to database
func (j JSON) Value() (driver.Value, error) {
	if j.Data == nil {
		return nil, nil
	}
	return json.Marshal(j.Data)
}

// MarshalJSON implements json.Marshaler interface
func (j JSON) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.Data)
}

// UnmarshalJSON implements json.Unmarshaler interface
func (j *JSON) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.Data)
}
