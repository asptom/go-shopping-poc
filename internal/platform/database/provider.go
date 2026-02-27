package database

import (
	"context"
	"fmt"
	"log/slog"

	"go-shopping-poc/internal/platform/config"
)

// Option is a functional option for configuring DatabaseProviderImpl.
type Option func(*DatabaseProviderImpl)

// WithLogger sets the logger for the DatabaseProviderImpl.
func WithLogger(logger *slog.Logger) Option {
	return func(p *DatabaseProviderImpl) {
		p.logger = logger
	}
}

// DatabaseProviderImpl implements the DatabaseProvider interface.
// It encapsulates database connection logic and provides a configured
// PostgreSQL database instance to services.
type DatabaseProviderImpl struct {
	database Database
	logger   *slog.Logger
}

// DatabaseProvider defines the interface for providing database connectivity.
// This interface is implemented by DatabaseProviderImpl.
type DatabaseProvider interface {
	GetDatabase() Database
}

// NewDatabaseProvider creates a new database provider with the given database URL.
// It loads platform database configuration, creates a PostgreSQL client,
// and establishes a connection to the database.
//
// Parameters:
//   - databaseURL: The PostgreSQL connection string (e.g., "postgres://user:pass@host:port/db?sslmode=disable")
//   - opts: Optional functional options for configuring the provider
//
// Returns:
//   - A configured DatabaseProvider that provides database connectivity
//   - An error if configuration loading, client creation, or connection fails
//
// Usage:
//
//	provider, err := database.NewDatabaseProvider("postgres://user:pass@localhost:5432/mydb?sslmode=disable")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	db := provider.GetDatabase()
//
// Or with custom logger:
//
//	provider, err := database.NewDatabaseProvider(url, database.WithLogger(logger))
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewDatabaseProvider(databaseURL string, opts ...Option) (DatabaseProvider, error) {
	p := &DatabaseProviderImpl{}

	for _, opt := range opts {
		opt(p)
	}

	if p.logger == nil {
		p.logger = Logger()
	}

	if databaseURL == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	// Load platform database connection configuration
	connConfigPtr, err := config.LoadConfig[ConnectionConfig]("platform-database")
	if err != nil {
		p.logger.Error("DatabaseProvider: Failed to load connection config", "error", err)
		return nil, fmt.Errorf("failed to load database connection config: %w", err)
	}
	connConfig := *connConfigPtr

	p.logger.Debug("DatabaseProvider: Connection config loaded successfully")

	// Create PostgreSQL database client with platform attributes added
	dbLogger := p.logger.With("platform", "database", "component", "postgresql")
	db, err := NewPostgreSQLClientWithLogger(databaseURL, dbLogger, connConfig)
	if err != nil {
		p.logger.Error("DatabaseProvider: Failed to create database client", "error", err)
		return nil, fmt.Errorf("failed to create database client: %w", err)
	}

	// Establish database connection
	ctx, cancel := context.WithTimeout(context.Background(), connConfig.ConnectTimeout)
	defer cancel()

	if err := db.Connect(ctx); err != nil {
		p.logger.Error("DatabaseProvider: Failed to connect to database", "error", err)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	p.logger.Info("DatabaseProvider: Database provider initialized successfully")

	return &DatabaseProviderImpl{
		database: db,
		logger:   p.logger,
	}, nil
}

// GetDatabase returns the configured database instance.
// The database connection is already established and ready for use.
//
// Returns:
//   - A Database interface implementation that can be used for queries, transactions, etc.
//
// Usage:
//
//	db := provider.GetDatabase()
//	rows, err := db.Query(ctx, "SELECT * FROM users")
//	if err != nil {
//	    return err
//	}
//	defer rows.Close()
func (p *DatabaseProviderImpl) GetDatabase() Database {
	return p.database
}
