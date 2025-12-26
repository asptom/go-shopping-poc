package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

// setupTestDB creates a test database connection for testing
func setupTestDB(t *testing.T) Database {
	t.Helper()

	// Get current working directory for debugging
	cwd, _ := os.Getwd()
	t.Logf("Test working directory: %s", cwd)

	// Use DATABASE_URL environment variable directly
	// This avoids import cycles with service packages
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping test, DATABASE_URL not set")
	}

	t.Logf("Using DATABASE_URL environment variable: %s", dbURL)

	db, err := NewPostgreSQLClient(dbURL)
	if err != nil {
		t.Skipf("Skipping test, failed to create database client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.Connect(ctx); err != nil {
		t.Skipf("Skipping test, database not available: %v", err)
	}

	return db
}

func TestPostgreSQLClient_Connect(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
		wantErr     bool
	}{
		{
			name:        "invalid host",
			databaseURL: "postgres://test:test@invalid-host:5432/test?sslmode=disable",
			wantErr:     true,
		},
	}

	// Test invalid host case (should always fail)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewPostgreSQLClient(tt.databaseURL)
			if err != nil {
				t.Fatalf("NewPostgreSQLClient() error = %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err = client.Connect(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Connect() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil {
				defer func() { _ = client.Close() }()
			}
		})
	}

	// Test valid config case (requires real database)
	t.Run("valid config", func(t *testing.T) {
		// Skip if no database URL is available
		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			t.Skip("Skipping test, DATABASE_URL not set")
		}

		client, err := NewPostgreSQLClient(dbURL)
		if err != nil {
			t.Fatalf("NewPostgreSQLClient() error = %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err = client.Connect(ctx)
		if err != nil {
			t.Skipf("Skipping test, database not available: %v", err)
		}
		defer func() { _ = client.Close() }()

		// If we get here, connection was successful
		t.Log("Successfully connected to database")
	})
}

func TestPostgreSQLClient_Ping(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	err := db.Ping(ctx)
	if err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}

func TestPostgreSQLClient_Query(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Create test table
	_, err := db.Exec(ctx, `CREATE TEMP TABLE test_query (id SERIAL PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(ctx, `INSERT INTO test_query (name) VALUES ($1)`, "test")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Test Query
	rows, err := db.Query(ctx, `SELECT id, name FROM test_query WHERE name = $1`, "test")
	if err != nil {
		t.Errorf("Query() error = %v", err)
		return
	}
	defer func() { _ = rows.Close() }()

	// Verify results
	if !rows.Next() {
		t.Errorf("Expected at least one row, got none")
		return
	}

	var id int
	var name string
	err = rows.Scan(&id, &name)
	if err != nil {
		t.Errorf("Scan() error = %v", err)
		return
	}

	if name != "test" {
		t.Errorf("Expected name 'test', got %s", name)
	}

	if rows.Next() {
		t.Errorf("Expected only one row, got more")
	}
}

func TestPostgreSQLClient_QueryRow(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Create test table
	_, err := db.Exec(ctx, `CREATE TEMP TABLE test_queryrow (id SERIAL PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Insert test data
	_, err = db.Exec(ctx, `INSERT INTO test_queryrow (name) VALUES ($1)`, "test")
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Test QueryRow
	row := db.QueryRow(ctx, `SELECT name FROM test_queryrow WHERE id = $1`, 1)

	var name string
	err = row.Scan(&name)
	if err != nil {
		t.Errorf("Scan() error = %v", err)
	}

	if name != "test" {
		t.Errorf("Expected name = 'test', got %s", name)
	}
}

func TestPostgreSQLClient_Exec(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Create test table
	_, err := db.Exec(ctx, `CREATE TEMP TABLE test_exec (id SERIAL PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test Exec
	result, err := db.Exec(ctx, `INSERT INTO test_exec (name) VALUES ($1)`, "test")
	if err != nil {
		t.Errorf("Exec() error = %v", err)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		t.Errorf("RowsAffected() error = %v", err)
	}

	if rowsAffected != 1 {
		t.Errorf("Expected 1 row affected, got %d", rowsAffected)
	}
}

func TestPostgreSQLClient_BeginTx(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Create test table
	_, err := db.Exec(ctx, `CREATE TEMP TABLE test_tx (id SERIAL PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test BeginTx
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Errorf("BeginTx() error = %v", err)
		return
	}

	// Execute in transaction
	_, err = tx.Exec(ctx, `INSERT INTO test_tx (name) VALUES ($1)`, "test")
	if err != nil {
		t.Errorf("Tx Exec() error = %v", err)
		_ = tx.Rollback()
		return
	}

	// Commit transaction
	err = tx.Commit()
	if err != nil {
		t.Errorf("Commit() error = %v", err)
	}

	// Verify data was inserted
	row := db.QueryRow(ctx, `SELECT COUNT(*) FROM test_tx WHERE name = $1`, "test")
	var count int
	err = row.Scan(&count)
	if err != nil {
		t.Errorf("Verification query error = %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 row in table, got %d", count)
	}
}

func TestPostgreSQLClient_BeginTx_Rollback(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Create test table
	_, err := db.Exec(ctx, `CREATE TEMP TABLE test_tx_rollback (id SERIAL PRIMARY KEY, name TEXT)`)
	if err != nil {
		t.Fatalf("Failed to create test table: %v", err)
	}

	// Test BeginTx with rollback
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Errorf("BeginTx() error = %v", err)
		return
	}

	// Execute in transaction
	_, err = tx.Exec(ctx, `INSERT INTO test_tx_rollback (name) VALUES ($1)`, "test")
	if err != nil {
		t.Errorf("Tx Exec() error = %v", err)
		_ = tx.Rollback()
		return
	}

	// Rollback transaction
	err = tx.Rollback()
	if err != nil {
		t.Errorf("Rollback() error = %v", err)
	}

	// Verify data was not inserted
	row := db.QueryRow(ctx, `SELECT COUNT(*) FROM test_tx_rollback WHERE name = $1`, "test")
	var count int
	err = row.Scan(&count)
	if err != nil {
		t.Errorf("Verification query error = %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 rows in table after rollback, got %d", count)
	}
}

func TestPostgreSQLClient_CheckHealth(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	ctx := context.Background()

	// Test ping
	err := db.Ping(ctx)
	if err != nil {
		t.Errorf("Expected database to be available, but ping failed: %v", err)
	}
}

func TestPostgreSQLClient_IsHealthy(t *testing.T) {
	// Skip if no database URL is available
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("Skipping test, DATABASE_URL not set")
	}

	client, err := NewPostgreSQLClient(dbURL)
	if err != nil {
		t.Fatalf("NewPostgreSQLClient() error = %v", err)
	}

	// Connect the client to make it healthy
	ctx := context.Background()
	err = client.Connect(ctx)
	if err != nil {
		t.Skipf("Skipping test, database not available: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Test IsHealthy
	pgClient := client.(*PostgreSQLClient)
	healthy := pgClient.IsHealthy(ctx)

	if !healthy {
		t.Error("Expected database to be healthy")
	}
}

func TestPostgreSQLClient_GetConnectionInfo_WithoutConnection(t *testing.T) {
	client, err := NewPostgreSQLClient("postgres://test:test@localhost:5432/test?sslmode=disable")
	if err != nil {
		t.Fatalf("NewPostgreSQLClient() error = %v", err)
	}

	pgClient := client.(*PostgreSQLClient)

	// Test without connection
	info := pgClient.GetConnectionInfo()

	if info["connected"].(bool) {
		t.Error("Expected connected to be false")
	}
}

func TestPostgreSQLClient_GetConnectionInfo_WithConnection(t *testing.T) {
	db := setupTestDB(t)
	defer func() { _ = db.Close() }()

	pgClient := db.(*PostgreSQLClient)

	// Test with connection
	info := pgClient.GetConnectionInfo()

	if !info["connected"].(bool) {
		t.Error("Expected connected to be true")
	}
}

func TestDefaultConnectionConfig(t *testing.T) {
	config := DefaultConnectionConfig()

	if config.MaxOpenConns != 25 {
		t.Errorf("Expected MaxOpenConns = 25, got %d", config.MaxOpenConns)
	}

	if config.MaxIdleConns != 25 {
		t.Errorf("Expected MaxIdleConns = 25, got %d", config.MaxIdleConns)
	}

	if config.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("Expected ConnMaxLifetime = 5m, got %v", config.ConnMaxLifetime)
	}

	if config.ConnectTimeout != 30*time.Second {
		t.Errorf("Expected ConnectTimeout = 30s, got %v", config.ConnectTimeout)
	}
}

func TestPostgreSQLClient_ValidateConnection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	// Test validation
	pgClient := db.(*PostgreSQLClient)
	err := pgClient.ValidateConnection(ctx)
	if err != nil {
		t.Errorf("ValidateConnection() error = %v", err)
	}
}

func TestPostgreSQLClient_OperationsWithoutConnection(t *testing.T) {
	client, err := NewPostgreSQLClient("postgres://test:test@localhost:5432/test?sslmode=disable")
	if err != nil {
		t.Fatalf("NewPostgreSQLClient() error = %v", err)
	}

	ctx := context.Background()

	// Test operations without connection
	err = client.Ping(ctx)
	if err == nil {
		t.Error("Expected Ping() to fail without connection")
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}

// TestJSON_Scan_Value tests JSON marshaling/unmarshaling for database operations
func TestJSON_Scan_Value(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "nil value",
			input:    nil,
			expected: nil,
		},
		{
			name:     "string value",
			input:    "test string",
			expected: "test string",
		},
		{
			name:     "map value",
			input:    map[string]interface{}{"key": "value", "number": 42},
			expected: map[string]interface{}{"key": "value", "number": float64(42)}, // JSON unmarshaling converts numbers to float64
		},
		{
			name:     "slice value",
			input:    []string{"a", "b", "c"},
			expected: []interface{}{"a", "b", "c"},
		},
		{
			name:     "number value",
			input:    123,
			expected: float64(123),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Value() method
			jsonValue := JSON{Data: tt.input}
			driverValue, err := jsonValue.Value()
			if err != nil {
				t.Fatalf("Value() error = %v", err)
			}

			if tt.input == nil {
				if driverValue != nil {
					t.Errorf("Value() = %v, want nil", driverValue)
				}
				return
			}

			// Test Scan() method with the driver value
			var scanned JSON
			err = scanned.Scan(driverValue)
			if err != nil {
				t.Fatalf("Scan() error = %v", err)
			}

			// Compare the scanned data with expected
			if scanned.Data == nil && tt.expected != nil {
				t.Errorf("Scan() result = nil, want %v", tt.expected)
				return
			}

			if scanned.Data != nil && tt.expected == nil {
				t.Errorf("Scan() result = %v, want nil", scanned.Data)
				return
			}

			// For complex types, compare JSON representations
			if tt.expected != nil {
				expectedJSON, _ := json.Marshal(tt.expected)
				actualJSON, _ := json.Marshal(scanned.Data)
				if string(expectedJSON) != string(actualJSON) {
					t.Errorf("Scan() result = %v, want %v", scanned.Data, tt.expected)
				}
			}
		})
	}
}

// TestJSON_JSONMarshaling tests JSON marshaling/unmarshaling interfaces
func TestJSON_JSONMarshaling(t *testing.T) {
	original := JSON{Data: map[string]interface{}{"theme": "dark", "count": 5}}

	// Test MarshalJSON
	jsonBytes, err := original.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	// Test UnmarshalJSON
	var unmarshaled JSON
	err = unmarshaled.UnmarshalJSON(jsonBytes)
	if err != nil {
		t.Fatalf("UnmarshalJSON() error = %v", err)
	}

	// Compare
	originalJSON, _ := json.Marshal(original.Data)
	unmarshaledJSON, _ := json.Marshal(unmarshaled.Data)
	if string(originalJSON) != string(unmarshaledJSON) {
		t.Errorf("JSON marshaling round-trip failed: got %v, want %v", unmarshaled.Data, original.Data)
	}
}

// TestPostgreSQLClient_SimplifiedApproach_DirectOperations tests that the simplified approach
// with direct sqlx operations works correctly (unit test without database)
func TestPostgreSQLClient_SimplifiedApproach_DirectOperations(t *testing.T) {
	// Test that the client can be created with valid config
	client, err := NewPostgreSQLClient("postgres://test:test@localhost:5432/test?sslmode=disable")
	if err != nil {
		t.Fatalf("NewPostgreSQLClient() error = %v", err)
	}

	// Test that operations fail gracefully without connection
	ctx := context.Background()

	// Test Query without connection
	_, err = client.Query(ctx, "SELECT 1")
	if err == nil {
		t.Errorf("Expected error for Query without connection, got nil")
	}
	if !containsString(err.Error(), "database connection not established") {
		t.Errorf("Expected connection error, got: %v", err)
	}

	// Test Exec without connection
	_, err = client.Exec(ctx, "SELECT 1")
	if err == nil {
		t.Errorf("Expected error for Exec without connection, got nil")
	}
	if !containsString(err.Error(), "database connection not established") {
		t.Errorf("Expected connection error, got: %v", err)
	}

	// Test QueryRow without connection
	row := client.QueryRow(ctx, "SELECT 1")
	if row != nil {
		t.Errorf("Expected nil row when no connection, got %v", row)
	}

	// Test BeginTx without connection
	_, err = client.BeginTx(ctx, nil)
	if err == nil {
		t.Errorf("Expected error for BeginTx without connection, got nil")
	}
	if !containsString(err.Error(), "database connection not established") {
		t.Errorf("Expected connection error, got: %v", err)
	}
}

// TestPostgreSQLClient_SimplifiedApproach_ErrorHandling tests the simple error handling approach
func TestPostgreSQLClient_SimplifiedApproach_ErrorHandling(t *testing.T) {
	client, err := NewPostgreSQLClient("postgres://test:test@localhost:5432/test?sslmode=disable")
	if err != nil {
		t.Fatalf("NewPostgreSQLClient() error = %v", err)
	}

	ctx := context.Background()

	// Test that all operations return consistent error messages
	operations := []struct {
		name string
		fn   func() error
	}{
		{"Ping", func() error { return client.Ping(ctx) }},
		{"Query", func() error { _, err := client.Query(ctx, "SELECT 1"); return err }},
		{"Exec", func() error { _, err := client.Exec(ctx, "SELECT 1"); return err }},
		{"BeginTx", func() error { _, err := client.BeginTx(ctx, nil); return err }},
	}

	for _, op := range operations {
		t.Run(op.name+"ErrorHandling", func(t *testing.T) {
			err := op.fn()
			if err == nil {
				t.Errorf("Expected error for %s without connection, got nil", op.name)
				return
			}

			// Check that error is wrapped with context
			if !containsString(err.Error(), "failed") && !containsString(err.Error(), "not established") {
				t.Errorf("Expected wrapped error for %s, got: %v", op.name, err)
			}
		})
	}
}

// TestPostgreSQLClient_SimplifiedApproach_Configuration tests that configuration works correctly
func TestPostgreSQLClient_SimplifiedApproach_Configuration(t *testing.T) {
	// Test default configuration
	client, err := NewPostgreSQLClient("postgres://test:test@localhost:5432/test?sslmode=disable")
	if err != nil {
		t.Fatalf("NewPostgreSQLClient() error = %v", err)
	}

	pgClient := client.(*PostgreSQLClient)

	// Check default config values
	expectedConfig := DefaultConnectionConfig()
	if pgClient.connConfig.MaxOpenConns != expectedConfig.MaxOpenConns {
		t.Errorf("Expected MaxOpenConns %d, got %d", expectedConfig.MaxOpenConns, pgClient.connConfig.MaxOpenConns)
	}
	if pgClient.connConfig.ConnectTimeout != expectedConfig.ConnectTimeout {
		t.Errorf("Expected ConnectTimeout %v, got %v", expectedConfig.ConnectTimeout, pgClient.connConfig.ConnectTimeout)
	}

	// Test custom configuration
	customConfig := ConnectionConfig{
		MaxOpenConns:       50,
		ConnectTimeout:     60 * time.Second,
		QueryTimeout:       45 * time.Second,
		HealthCheckTimeout: 15 * time.Second,
	}

	client2, err := NewPostgreSQLClient("postgres://test:test@localhost:5432/test?sslmode=disable", customConfig)
	if err != nil {
		t.Fatalf("NewPostgreSQLClient() with custom config error = %v", err)
	}

	pgClient2 := client2.(*PostgreSQLClient)
	if pgClient2.connConfig.MaxOpenConns != 50 {
		t.Errorf("Expected custom MaxOpenConns 50, got %d", pgClient2.connConfig.MaxOpenConns)
	}
	if pgClient2.connConfig.ConnectTimeout != 60*time.Second {
		t.Errorf("Expected custom ConnectTimeout 60s, got %v", pgClient2.connConfig.ConnectTimeout)
	}
}

// TestPostgreSQLClient_SimplifiedApproach_SequentialProcessingLogic tests the logic
// for sequential processing without requiring database connectivity
func TestPostgreSQLClient_SimplifiedApproach_SequentialProcessingLogic(t *testing.T) {
	client, err := NewPostgreSQLClient("postgres://test:test@localhost:5432/test?sslmode=disable")
	if err != nil {
		t.Fatalf("NewPostgreSQLClient() error = %v", err)
	}

	ctx := context.Background()

	// Test that operations are designed for sequential execution
	// This is more of a documentation test - the actual sequential behavior
	// would be tested in integration tests with a real database

	// Test that context timeouts are properly applied
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
	defer cancel()

	// Give a tiny bit of time for timeout
	time.Sleep(1 * time.Microsecond)

	// Operations should respect context timeout (though they fail for other reasons first)
	_, err = client.Query(timeoutCtx, "SELECT 1")
	if err == nil {
		t.Errorf("Expected error due to context or connection, got nil")
	}

	// The error should still be wrapped properly
	if !containsString(err.Error(), "failed") && !containsString(err.Error(), "not established") {
		t.Errorf("Expected wrapped error, got: %v", err)
	}
}

// TestPostgreSQLClient_SimplifiedApproach_ConcurrentSafety tests that the simplified approach
// is safe for concurrent use (unit test without database)
func TestPostgreSQLClient_SimplifiedApproach_ConcurrentSafety(t *testing.T) {
	// Test that multiple clients can be created concurrently without issues
	const numClients = 10

	clients := make([]Database, numClients)
	errors := make(chan error, numClients)

	// Create multiple clients concurrently
	for i := 0; i < numClients; i++ {
		go func(id int) {
			client, err := NewPostgreSQLClient("postgres://test:test@localhost:5432/test?sslmode=disable")
			if err != nil {
				errors <- err
				return
			}
			clients[id] = client
			errors <- nil
		}(i)
	}

	// Wait for all creations to complete
	for i := 0; i < numClients; i++ {
		err := <-errors
		if err != nil {
			t.Errorf("Client creation %d failed: %v", i, err)
		}
	}

	// Test that concurrent operations on different clients work
	ctx := context.Background()
	done := make(chan bool, numClients)

	for i := 0; i < numClients; i++ {
		go func(clientID int) {
			defer func() { done <- true }()

			client := clients[clientID]
			if client == nil {
				t.Errorf("Client %d is nil", clientID)
				return
			}

			// Test concurrent calls to methods that don't require connection
			// These should not panic or cause race conditions
			_ = client.Ping(ctx)                 // Should fail gracefully
			_, _ = client.Query(ctx, "SELECT 1") // Should fail gracefully
			_, _ = client.Exec(ctx, "SELECT 1")  // Should fail gracefully
			_ = client.QueryRow(ctx, "SELECT 1") // Should return row that errors on scan
		}(i)
	}

	// Wait for all concurrent operations to complete
	for i := 0; i < numClients; i++ {
		<-done
	}

	// Test that the same client can be accessed concurrently
	client := clients[0]
	concurrentOps := 20
	done2 := make(chan bool, concurrentOps)

	for i := 0; i < concurrentOps; i++ {
		go func(opID int) {
			defer func() { done2 <- true }()

			// These operations should be safe to call concurrently
			// even though they will fail due to no connection
			_ = client.Ping(ctx)
			_, _ = client.Query(ctx, "SELECT 1")
		}(i)
	}

	// Wait for all operations to complete
	for i := 0; i < concurrentOps; i++ {
		<-done2
	}
}

// TestPostgreSQLClient_SimplifiedApproach_ErrorConsistency tests that the simplified error
// handling approach provides consistent error messages
func TestPostgreSQLClient_SimplifiedApproach_ErrorConsistency(t *testing.T) {
	client, err := NewPostgreSQLClient("postgres://test:test@localhost:5432/test?sslmode=disable")
	if err != nil {
		t.Fatalf("NewPostgreSQLClient() error = %v", err)
	}

	ctx := context.Background()

	// Test that all database operations return consistent error patterns
	testCases := []struct {
		name      string
		operation func() error
	}{
		{
			name:      "Ping",
			operation: func() error { return client.Ping(ctx) },
		},
		{
			name:      "Query",
			operation: func() error { _, err := client.Query(ctx, "SELECT 1"); return err },
		},
		{
			name: "QueryRow",
			operation: func() error {
				row := client.QueryRow(ctx, "SELECT 1")
				if row == nil {
					return fmt.Errorf("database connection not established")
				}
				var dummy int
				return row.Scan(&dummy)
			},
		},
		{
			name:      "Exec",
			operation: func() error { _, err := client.Exec(ctx, "SELECT 1"); return err },
		},
		{
			name:      "BeginTx",
			operation: func() error { _, err := client.BeginTx(ctx, nil); return err },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"ErrorConsistency", func(t *testing.T) {
			err := tc.operation()
			if err == nil {
				t.Errorf("Expected error for %s without connection, got nil", tc.name)
				return
			}

			// Check that error messages are consistent and informative
			errMsg := err.Error()
			if !containsString(errMsg, "database connection not established") &&
				!containsString(errMsg, "failed to") {
				t.Errorf("Error message for %s not consistent with simplified approach: %s", tc.name, errMsg)
			}

			// Check that errors are properly wrapped (contain context)
			if len(errMsg) < 10 { // Very basic check for meaningful error messages
				t.Errorf("Error message for %s is too short: %s", tc.name, errMsg)
			}
		})
	}
}

// containsString is a helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
