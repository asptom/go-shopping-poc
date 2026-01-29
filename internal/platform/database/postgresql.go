package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// PostgreSQLClient implements the Database interface for PostgreSQL
type PostgreSQLClient struct {
	db          *sqlx.DB
	databaseURL string
	connConfig  ConnectionConfig
}

// DB returns the underlying sqlx.DB instance
func (c *PostgreSQLClient) DB() *sqlx.DB {
	return c.db
}

// NewPostgreSQLClient creates a new PostgreSQL database client
func NewPostgreSQLClient(databaseURL string, connConfig ...ConnectionConfig) (Database, error) {
	// Use default connection config if not provided
	cfg := DefaultConnectionConfig()
	if len(connConfig) > 0 {
		cfg = connConfig[0]
	}

	client := &PostgreSQLClient{
		databaseURL: databaseURL,
		connConfig:  cfg,
	}

	return client, nil
}

// DefaultConnectionConfig returns default connection configuration
func DefaultConnectionConfig() ConnectionConfig {
	return ConnectionConfig{
		MaxOpenConns:        25,
		MaxIdleConns:        25,
		ConnMaxLifetime:     5 * time.Minute,
		ConnMaxIdleTime:     5 * time.Minute,
		ConnectTimeout:      30 * time.Second,
		QueryTimeout:        30 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		HealthCheckTimeout:  5 * time.Second,
	}
}

// Connect establishes a connection to the database
func (c *PostgreSQLClient) Connect(ctx context.Context) error {
	log.Printf("[INFO] Database: Connecting to PostgreSQL")

	// Connect to database
	db, err := sqlx.Connect("pgx", c.databaseURL)
	if err != nil {
		log.Printf("[ERROR] Database: Failed to connect to PostgreSQL: %v", err)
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(c.connConfig.MaxOpenConns)
	db.SetMaxIdleConns(c.connConfig.MaxIdleConns)
	db.SetConnMaxLifetime(c.connConfig.ConnMaxLifetime)
	db.SetConnMaxIdleTime(c.connConfig.ConnMaxIdleTime)

	c.db = db

	log.Printf("[INFO] Database: Successfully connected to PostgreSQL")
	return nil
}

// Close closes the database connection
func (c *PostgreSQLClient) Close() error {
	if c.db == nil {
		return nil
	}

	log.Printf("[INFO] Database: Closing PostgreSQL connection")
	err := c.db.Close()
	c.db = nil

	if err != nil {
		log.Printf("[ERROR] Database: Failed to close connection: %v", err)
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	log.Printf("[INFO] Database: PostgreSQL connection closed")
	return nil
}

// Ping checks the database connection
func (c *PostgreSQLClient) Ping(ctx context.Context) error {
	if c.db == nil {
		return fmt.Errorf("database connection not established")
	}

	// Create context with timeout for ping
	pingCtx, cancel := context.WithTimeout(ctx, c.connConfig.HealthCheckTimeout)
	defer cancel()

	start := time.Now()
	err := c.db.PingContext(pingCtx)
	latency := time.Since(start)

	if err != nil {
		log.Printf("[ERROR] Database: Ping failed after %v: %v", latency, err)
		return fmt.Errorf("database ping failed: %w", err)
	}

	log.Printf("[DEBUG] Database: Ping successful (latency: %v)", latency)
	return nil
}

// Query executes a query that returns rows
func (c *PostgreSQLClient) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if c.db == nil {
		return nil, fmt.Errorf("database connection not established")
	}

	// Create context with timeout
	queryCtx, cancel := context.WithTimeout(ctx, c.connConfig.QueryTimeout)
	defer cancel()

	//log.Printf("[DEBUG] Database: Executing query: %s", query)

	start := time.Now()
	rows, err := c.db.QueryContext(queryCtx, query, args...)
	latency := time.Since(start)

	if err != nil {
		log.Printf("[ERROR] Database: Query failed after %v: %v", latency, err)
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	//log.Printf("[DEBUG] Database: Query completed in %v", latency)
	return rows, nil
}

// QueryRow executes a query that returns at most one row
func (c *PostgreSQLClient) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if c.db == nil {
		return nil
	}

	// Create context with timeout
	queryCtx, cancel := context.WithTimeout(ctx, c.connConfig.QueryTimeout)
	defer cancel()

	//log.Printf("[DEBUG] Database: Executing query row: %s", query)

	return c.db.QueryRowContext(queryCtx, query, args...)
}

// Exec executes a query without returning rows
func (c *PostgreSQLClient) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if c.db == nil {
		return nil, fmt.Errorf("database connection not established")
	}

	// Create context with timeout
	queryCtx, cancel := context.WithTimeout(ctx, c.connConfig.QueryTimeout)
	defer cancel()

	//log.Printf("[DEBUG] Database: Executing exec: %s", query)

	start := time.Now()
	result, err := c.db.ExecContext(queryCtx, query, args...)
	latency := time.Since(start)

	if err != nil {
		log.Printf("[ERROR] Database: Exec failed after %v: %v", latency, err)
		return nil, fmt.Errorf("exec execution failed: %w", err)
	}

	//log.Printf("[DEBUG] Database: Exec completed in %v", latency)
	return result, nil
}

// BeginTx begins a transaction
func (c *PostgreSQLClient) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	if c.db == nil {
		return nil, fmt.Errorf("database connection not established")
	}

	log.Printf("[DEBUG] Database: Beginning transaction")

	tx, err := c.db.BeginTxx(ctx, opts)
	if err != nil {
		log.Printf("[ERROR] Database: Failed to begin transaction: %v", err)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &PostgreSQLTx{tx: tx, ctx: ctx}, nil
}

// Stats returns database statistics
func (c *PostgreSQLClient) Stats() sql.DBStats {
	if c.db == nil {
		return sql.DBStats{}
	}
	return c.db.Stats()
}

// GetContext executes a query that returns at most one row and scans it into dest
func (c *PostgreSQLClient) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return c.db.GetContext(ctx, dest, query, args...)
}

// SelectContext executes a query and scans the results into dest
func (c *PostgreSQLClient) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return c.db.SelectContext(ctx, dest, query, args...)
}

// PostgreSQLTx implements the Tx interface for PostgreSQL transactions
type PostgreSQLTx struct {
	tx  *sqlx.Tx
	ctx context.Context
}

// Context returns the transaction's context for cancellation and tracing
func (t *PostgreSQLTx) Context() context.Context {
	return t.ctx
}

// Query executes a query within the transaction
func (t *PostgreSQLTx) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, query, args...)
}

// QueryRow executes a query that returns at most one row within the transaction
func (t *PostgreSQLTx) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return t.tx.QueryRowContext(ctx, query, args...)
}

// Exec executes a command within the transaction
func (t *PostgreSQLTx) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

// ExecContext executes a command within the transaction (alias for Exec)
func (t *PostgreSQLTx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return t.tx.ExecContext(ctx, query, args...)
}

// NamedExecContext executes a named query within the transaction
func (t *PostgreSQLTx) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	return t.tx.NamedExecContext(ctx, query, arg)
}

// Commit commits the transaction
func (t *PostgreSQLTx) Commit() error {
	log.Printf("[DEBUG] Database: Committing transaction")
	err := t.tx.Commit()
	if err != nil {
		log.Printf("[ERROR] Database: Failed to commit transaction: %v", err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	log.Printf("[DEBUG] Database: Transaction committed")
	return nil
}

// Rollback rolls back the transaction
func (t *PostgreSQLTx) Rollback() error {
	log.Printf("[DEBUG] Database: Rolling back transaction")
	err := t.tx.Rollback()
	if err != nil {
		log.Printf("[ERROR] Database: Failed to rollback transaction: %v", err)
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	log.Printf("[DEBUG] Database: Transaction rolled back")
	return nil
}
