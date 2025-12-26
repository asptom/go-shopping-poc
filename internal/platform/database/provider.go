package database

import (
	"context"
	"fmt"
	"log"

	"go-shopping-poc/internal/platform/config"
)

// DatabaseProviderImpl implements the DatabaseProvider interface.
// It encapsulates database connection logic and provides a configured
// PostgreSQL database instance to services.
type DatabaseProviderImpl struct {
	database Database
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
func NewDatabaseProvider(databaseURL string) (DatabaseProvider, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	log.Printf("[INFO] DatabaseProvider: Initializing database provider")

	// Load platform database connection configuration
	connConfigPtr, err := config.LoadConfig[ConnectionConfig]("platform-database")
	if err != nil {
		log.Printf("[ERROR] DatabaseProvider: Failed to load connection config: %v", err)
		return nil, fmt.Errorf("failed to load database connection config: %w", err)
	}
	connConfig := *connConfigPtr

	log.Printf("[DEBUG] DatabaseProvider: Connection config loaded successfully")

	// Create PostgreSQL database client
	db, err := NewPostgreSQLClient(databaseURL, connConfig)
	if err != nil {
		log.Printf("[ERROR] DatabaseProvider: Failed to create database client: %v", err)
		return nil, fmt.Errorf("failed to create database client: %w", err)
	}

	// Establish database connection
	ctx, cancel := context.WithTimeout(context.Background(), connConfig.ConnectTimeout)
	defer cancel()

	if err := db.Connect(ctx); err != nil {
		log.Printf("[ERROR] DatabaseProvider: Failed to connect to database: %v", err)
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	log.Printf("[INFO] DatabaseProvider: Database provider initialized successfully")

	return &DatabaseProviderImpl{
		database: db,
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
