package database

import (
	"context"
	"fmt"
	"log"
	"time"
)

// CheckHealth performs a comprehensive health check on the database
func (c *PostgreSQLClient) CheckHealth(ctx context.Context) *HealthStatus {
	status := &HealthStatus{}

	if c.db == nil {
		status.Error = fmt.Errorf("database connection not established")
		log.Printf("[ERROR] Database: Health check failed - no connection")
		return status
	}

	start := time.Now()

	// Create context with timeout for health check
	healthCtx, cancel := context.WithTimeout(ctx, c.connConfig.HealthCheckTimeout)
	defer cancel()

	// Perform ping
	err := c.db.PingContext(healthCtx)
	latency := time.Since(start)

	status.Latency = latency

	if err != nil {
		status.Error = fmt.Errorf("database ping failed: %w", err)
		log.Printf("[ERROR] Database: Health check failed after %v: %v", latency, err)
		return status
	}

	status.Available = true
	log.Printf("[DEBUG] Database: Health check passed (latency: %v)", latency)
	return status
}

// IsHealthy returns true if the database is healthy
func (c *PostgreSQLClient) IsHealthy(ctx context.Context) bool {
	status := c.CheckHealth(ctx)
	return status.Available
}

// GetConnectionInfo returns information about the current connection
func (c *PostgreSQLClient) GetConnectionInfo() map[string]interface{} {
	info := map[string]interface{}{
		"connected": c.db != nil,
		"config": map[string]interface{}{
			"max_open_conns":     c.connConfig.MaxOpenConns,
			"max_idle_conns":     c.connConfig.MaxIdleConns,
			"conn_max_lifetime":  c.connConfig.ConnMaxLifetime.String(),
			"conn_max_idle_time": c.connConfig.ConnMaxIdleTime.String(),
		},
	}

	if c.db != nil {
		stats := c.db.Stats()
		info["stats"] = map[string]interface{}{
			"open_connections":    stats.OpenConnections,
			"in_use":              stats.InUse,
			"idle":                stats.Idle,
			"wait_count":          stats.WaitCount,
			"wait_duration":       stats.WaitDuration.String(),
			"max_idle_closed":     stats.MaxIdleClosed,
			"max_lifetime_closed": stats.MaxLifetimeClosed,
		}
	}

	return info
}

// ValidateConnection validates that the database connection is working properly
func (c *PostgreSQLClient) ValidateConnection(ctx context.Context) error {
	// Check if connected
	if c.db == nil {
		return fmt.Errorf("database not connected")
	}

	// Perform health check
	status := c.CheckHealth(ctx)
	if !status.Available {
		return fmt.Errorf("database health check failed: %w", status.Error)
	}

	// Try a simple query to validate functionality
	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var result int
	err := c.db.QueryRowContext(queryCtx, "SELECT 1").Scan(&result)
	if err != nil {
		return fmt.Errorf("database validation query failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("database validation query returned unexpected result: %d", result)
	}

	log.Printf("[INFO] Database: Connection validation successful")
	return nil
}

// Reconnect attempts to reconnect to the database
func (c *PostgreSQLClient) Reconnect(ctx context.Context) error {
	log.Printf("[INFO] Database: Attempting to reconnect")

	// Close existing connection if any
	if c.db != nil {
		if err := c.Close(); err != nil {
			log.Printf("[WARNING] Database: Error closing existing connection during reconnect: %v", err)
		}
	}

	// Attempt to connect
	return c.Connect(ctx)
}

// WithTimeout creates a context with the configured query timeout
func (c *PostgreSQLClient) WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, c.connConfig.QueryTimeout)
}

// WithConnectTimeout creates a context with the configured connect timeout
func (c *PostgreSQLClient) WithConnectTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, c.connConfig.ConnectTimeout)
}
