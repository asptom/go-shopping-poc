package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// PostgreSQLClient implements the Database interface for PostgreSQL
type PostgreSQLClient struct {
	db          *sqlx.DB
	databaseURL string
	connConfig  ConnectionConfig
	logger      *slog.Logger
}

// DB returns the underlying sqlx.DB instance
func (c *PostgreSQLClient) DB() *sqlx.DB {
	return c.db
}

// NewPostgreSQLClient creates a new PostgreSQL database client
func NewPostgreSQLClient(databaseURL string, connConfig ...ConnectionConfig) (Database, error) {
	return NewPostgreSQLClientWithLogger(databaseURL, nil, connConfig...)
}

// NewPostgreSQLClientWithLogger creates a new PostgreSQL database client with a custom logger
func NewPostgreSQLClientWithLogger(databaseURL string, logger *slog.Logger, connConfig ...ConnectionConfig) (Database, error) {
	// Use default connection config if not provided
	cfg := DefaultConnectionConfig()
	if len(connConfig) > 0 {
		cfg = connConfig[0]
	}

	if logger == nil {
		logger = Logger()
	}

	client := &PostgreSQLClient{
		databaseURL: databaseURL,
		connConfig:  cfg,
		logger:      logger,
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
	c.logger.Debug("Connecting to PostgreSQL",
		"database", c.databaseURL,
	)

	// Connect to database
	db, err := sqlx.Connect("pgx", c.databaseURL)
	if err != nil {
		c.logger.Error("Failed to connect to PostgreSQL",
			"error", err.Error(),
		)
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(c.connConfig.MaxOpenConns)
	db.SetMaxIdleConns(c.connConfig.MaxIdleConns)
	db.SetConnMaxLifetime(c.connConfig.ConnMaxLifetime)
	db.SetConnMaxIdleTime(c.connConfig.ConnMaxIdleTime)

	c.db = db

	c.logger.Debug("Successfully connected to PostgreSQL")
	return nil
}

// Close closes the database connection
func (c *PostgreSQLClient) Close() error {
	if c.db == nil {
		return nil
	}

	c.logger.Debug("Closing PostgreSQL connection")
	err := c.db.Close()
	c.db = nil

	if err != nil {
		c.logger.Error("Failed to close connection",
			"error", err.Error(),
		)
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	c.logger.Debug("PostgreSQL connection closed")
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
		c.logger.Error("Ping failed",
			"latency", latency.String(),
			"error", err.Error(),
		)
		return fmt.Errorf("database ping failed: %w", err)
	}

	c.logger.Debug("Ping successful", "latency", latency.String())
	return nil
}

// Query executes a query that returns rows
// NOTE: Uses parent context directly (no timeout wrapper) because returned rows
// need the context to remain valid during iteration. The underlying pgx driver
// has its own connection timeouts for the initial query execution.
func (c *PostgreSQLClient) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if c.db == nil {
		return nil, fmt.Errorf("database connection not established")
	}

	start := time.Now()
	rows, err := c.db.QueryContext(ctx, query, args...)
	latency := time.Since(start)

	if err != nil {
		c.logger.Error("Query failed", "latency", latency.String(), "error", err.Error())
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	//logger.Debug("Query completed", "latency", latency.String())
	return rows, nil
}

// QueryRow executes a query that returns at most one row
// NOTE: Uses parent context directly (no timeout wrapper) because the returned
// *sql.Row needs the context to remain valid when .Scan() is called later.
// The underlying pgx driver has its own connection timeouts.
func (c *PostgreSQLClient) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if c.db == nil {
		return nil
	}

	//c.logger.Debug("Executing query row", "query", query)

	return c.db.QueryRowContext(ctx, query, args...)
}

// Exec executes a query without returning rows
func (c *PostgreSQLClient) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if c.db == nil {
		return nil, fmt.Errorf("database connection not established")
	}

	// Create context with timeout
	queryCtx, cancel := context.WithTimeout(ctx, c.connConfig.QueryTimeout)
	defer cancel()

	start := time.Now()
	result, err := c.db.ExecContext(queryCtx, query, args...)
	latency := time.Since(start)

	if err != nil {
		c.logger.Error("Exec failed", "latency", latency.String(), "error", err.Error())
		return nil, fmt.Errorf("exec execution failed: %w", err)
	}

	//c.logger.Debug("Exec completed", "latency", latency.String())
	return result, nil
}

// BeginTx begins a transaction
func (c *PostgreSQLClient) BeginTx(ctx context.Context, opts *sql.TxOptions) (Tx, error) {
	if c.db == nil {
		return nil, fmt.Errorf("database connection not established")
	}

	c.logger.Debug("Beginning transaction")

	tx, err := c.db.BeginTxx(ctx, opts)
	if err != nil {
		c.logger.Error("Failed to begin transaction",
			"error", err.Error(),
		)
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &PostgreSQLTx{tx: tx, ctx: ctx, logger: c.logger}, nil
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
	tx     *sqlx.Tx
	ctx    context.Context
	logger *slog.Logger
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

// GetContext executes a query that returns at most one row and scans it into dest
func (t *PostgreSQLTx) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return t.tx.GetContext(ctx, dest, query, args...)
}

// SelectContext executes a query and scans the results into dest
func (t *PostgreSQLTx) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return t.tx.SelectContext(ctx, dest, query, args...)
}

// Commit commits the transaction
func (t *PostgreSQLTx) Commit() error {
	t.logger.Debug("Committing transaction")
	err := t.tx.Commit()
	if err != nil {
		t.logger.Error("Failed to commit transaction",
			"error", err.Error(),
		)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	t.logger.Debug("Transaction committed")
	return nil
}

// Rollback rolls back the transaction
func (t *PostgreSQLTx) Rollback() error {
	t.logger.Debug("Rolling back transaction")
	err := t.tx.Rollback()
	if err != nil {
		t.logger.Error("Failed to rollback transaction",
			"error", err.Error(),
		)
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	t.logger.Debug("Transaction rolled back")
	return nil
}
