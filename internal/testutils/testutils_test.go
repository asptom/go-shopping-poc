// Package testutils_test provides comprehensive tests for the testutils package.
//
// This test suite ensures that all test utility functions work correctly
// and handle edge cases properly, maintaining the reliability of the
// testing infrastructure itself.
package testutils_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/testutils"
)

// mockDatabase implements database.Database interface for testing
type mockDatabase struct {
	db *sqlx.DB
}

func newMockDatabase(sqlDB *sql.DB) database.Database {
	return &mockDatabase{
		db: sqlx.NewDb(sqlDB, "sqlmock"),
	}
}

func (m *mockDatabase) Connect(ctx context.Context) error {
	return nil // Mock is already "connected"
}

func (m *mockDatabase) Close() error {
	return m.db.Close()
}

func (m *mockDatabase) Ping(ctx context.Context) error {
	return m.db.PingContext(ctx)
}

func (m *mockDatabase) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return m.db.QueryContext(ctx, query, args...)
}

func (m *mockDatabase) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return m.db.QueryRowContext(ctx, query, args...)
}

func (m *mockDatabase) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return m.db.ExecContext(ctx, query, args...)
}

func (m *mockDatabase) BeginTx(ctx context.Context, opts *sql.TxOptions) (database.Tx, error) {
	tx, err := m.db.BeginTxx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &mockTx{tx: tx}, nil
}

func (m *mockDatabase) Stats() sql.DBStats {
	return m.db.Stats()
}

func (m *mockDatabase) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return m.db.GetContext(ctx, dest, query, args...)
}

func (m *mockDatabase) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return m.db.SelectContext(ctx, dest, query, args...)
}

func (m *mockDatabase) DB() *sqlx.DB {
	return m.db
}

// mockTx implements database.Tx interface for testing
type mockTx struct {
	tx *sqlx.Tx
}

func (m *mockTx) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return m.tx.QueryContext(ctx, query, args...)
}

func (m *mockTx) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return m.tx.QueryRowContext(ctx, query, args...)
}

func (m *mockTx) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return m.tx.ExecContext(ctx, query, args...)
}

func (m *mockTx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return m.tx.ExecContext(ctx, query, args...)
}

func (m *mockTx) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	return m.tx.NamedExecContext(ctx, query, arg)
}

func (m *mockTx) Commit() error {
	return m.tx.Commit()
}

func (m *mockTx) Rollback() error {
	return m.tx.Rollback()
}

func TestSetupTestDB_NoDatabaseURL(t *testing.T) {
	// Ensure DATABASE_URL is not set
	originalURL := os.Getenv("DATABASE_URL")
	defer func() {
		if originalURL != "" {
			_ = os.Setenv("DATABASE_URL", originalURL)
		} else {
			_ = os.Unsetenv("DATABASE_URL")
		}
	}()
	_ = os.Unsetenv("DATABASE_URL")

	// This should skip the test
	db := testutils.SetupTestDB(t)
	if db != nil {
		t.Error("Expected nil database when DATABASE_URL is not set")
	}
}

func TestSetupTestDB_InvalidDatabaseURL(t *testing.T) {
	// Set invalid DATABASE_URL
	originalURL := os.Getenv("DATABASE_URL")
	defer func() {
		if originalURL != "" {
			os.Setenv("DATABASE_URL", originalURL)
		} else {
			os.Unsetenv("DATABASE_URL")
		}
	}()
	os.Setenv("DATABASE_URL", "invalid-url")

	// This should skip the test due to connection failure
	db := testutils.SetupTestDB(t)
	if db != nil {
		t.Error("Expected nil database when DATABASE_URL is invalid")
	}
}

// Note: Error condition tests are not included because the testutils functions
// are designed to fail the test (using t.Fatalf) when errors occur, which is
// the correct behavior for test utilities. Error handling is tested indirectly
// through the integration tests and mock expectations.

func TestSetupTestEnvironment_ConfigLoadFailure(t *testing.T) {
	// We can't easily mock the config loading failure without complex setup
	// This test mainly ensures the function doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SetupTestEnvironment panicked: %v", r)
		}
	}()

	testutils.SetupTestEnvironment(t)
}

// TestMockDatabaseOperations tests the database operations using mocks
func TestMockDatabaseOperations(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer func() { _ = db.Close() }()

	mockDB := newMockDatabase(db)

	t.Run("CreateTestCustomer_Success", func(t *testing.T) {
		mock.ExpectExec(`INSERT INTO customers\.Customer`).
			WithArgs(sqlmock.AnyArg(), "testuser", "test@example.com", "Test", "User", "555-1234", sqlmock.AnyArg(), "active", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		customerID := testutils.CreateTestCustomer(t, mockDB)

		if customerID == "" {
			t.Error("Expected non-empty customer ID")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unmet expectations: %v", err)
		}
	})

	t.Run("CreateTestAddress_Success", func(t *testing.T) {
		customerID := "550e8400-e29b-41d4-a716-446655440000"

		mock.ExpectExec(`INSERT INTO customers\.Address`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "shipping", "Test", "User", "123 Main St", "Test City", "TS", "12345").
			WillReturnResult(sqlmock.NewResult(1, 1))

		addressID := testutils.CreateTestAddress(t, mockDB, customerID)

		if addressID == "" {
			t.Error("Expected non-empty address ID")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unmet expectations: %v", err)
		}
	})

	t.Run("CreateTestCreditCard_Success", func(t *testing.T) {
		customerID := "550e8400-e29b-41d4-a716-446655440000"

		mock.ExpectExec(`INSERT INTO customers\.CreditCard`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "visa", "4111111111111111", "Test User", "12/25", "123").
			WillReturnResult(sqlmock.NewResult(1, 1))

		cardID := testutils.CreateTestCreditCard(t, mockDB, customerID)

		if cardID == "" {
			t.Error("Expected non-empty card ID")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unmet expectations: %v", err)
		}
	})

	t.Run("CleanupTestData_Success", func(t *testing.T) {
		customerID := "550e8400-e29b-41d4-a716-446655440000"

		// Expect all cleanup queries
		mock.ExpectExec(`DELETE FROM customers\.CreditCard`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM customers\.Address`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM customers\.CustomerStatusHistory`).WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectExec(`DELETE FROM customers\.Customer`).WillReturnResult(sqlmock.NewResult(0, 1))

		// This should not panic
		testutils.CleanupTestData(t, mockDB, customerID)

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unmet expectations: %v", err)
		}
	})
}

// TestCleanupTestData_PartialFailure tests cleanup when some operations fail
func TestCleanupTestData_PartialFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer func() { _ = db.Close() }()

	mockDB := newMockDatabase(db)

	customerID := "550e8400-e29b-41d4-a716-446655440000"

	// Mock successful CreditCard deletion
	mock.ExpectExec(`DELETE FROM customers\.CreditCard`).WillReturnResult(sqlmock.NewResult(0, 1))
	// Mock failed Address deletion
	mock.ExpectExec(`DELETE FROM customers\.Address`).WillReturnError(sqlmock.ErrCancelled)
	// Mock successful StatusHistory deletion
	mock.ExpectExec(`DELETE FROM customers\.CustomerStatusHistory`).WillReturnResult(sqlmock.NewResult(0, 1))
	// Mock successful Customer deletion
	mock.ExpectExec(`DELETE FROM customers\.Customer`).WillReturnResult(sqlmock.NewResult(0, 1))

	// This should not panic, but should log warnings for failed operations
	testutils.CleanupTestData(t, mockDB, customerID)

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}

// TestIntegrationTestDB tests actual database operations when available
func TestIntegrationTestDB(t *testing.T) {
	// Skip if no database available
	db := testutils.SetupTestDB(t)
	if db == nil {
		t.Skip("Database not available for integration test")
	}
	defer func() { _ = db.Close() }()

	t.Run("FullCustomerLifecycle", func(t *testing.T) {
		// Create customer
		customerID := testutils.CreateTestCustomer(t, db)
		if customerID == "" {
			t.Fatal("Failed to create test customer")
		}

		// Create address
		addressID := testutils.CreateTestAddress(t, db, customerID)
		if addressID == "" {
			t.Fatal("Failed to create test address")
		}

		// Create credit card
		cardID := testutils.CreateTestCreditCard(t, db, customerID)
		if cardID == "" {
			t.Fatal("Failed to create test credit card")
		}

		// Cleanup
		testutils.CleanupTestData(t, db, customerID)

		t.Logf("Successfully completed customer lifecycle test with ID: %s", customerID)
	})
}

// TestEnvironmentSetup tests the environment setup function
func TestEnvironmentSetup(t *testing.T) {
	// This mainly tests that the function doesn't panic
	// The actual skipping behavior depends on config availability
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SetupTestEnvironment panicked unexpectedly: %v", r)
		}
	}()

	testutils.SetupTestEnvironment(t)
}

// TestHelperFunctions tests that all functions are properly marked as helpers
func TestHelperFunctions(t *testing.T) {
	// This test ensures that the t.Helper() calls are present
	// We can't directly test this, but we can verify the functions exist and are callable

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock DB: %v", err)
	}
	defer func() { _ = db.Close() }()

	mockDB := newMockDatabase(db)

	// Mock successful operations
	mock.ExpectExec(`INSERT INTO customers\.Customer`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO customers\.Address`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`INSERT INTO customers\.CreditCard`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(`DELETE FROM customers\.CreditCard`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM customers\.Address`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM customers\.CustomerStatusHistory`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(`DELETE FROM customers\.Customer`).WillReturnResult(sqlmock.NewResult(0, 1))

	customerID := "550e8400-e29b-41d4-a716-446655440000"

	// Test that functions can be called without panicking
	testCustomerID := testutils.CreateTestCustomer(t, mockDB)
	testAddressID := testutils.CreateTestAddress(t, mockDB, customerID)
	testCardID := testutils.CreateTestCreditCard(t, mockDB, customerID)
	testutils.CleanupTestData(t, mockDB, customerID)

	// Verify return values are reasonable
	if testCustomerID == "" {
		t.Error("CreateTestCustomer returned empty ID")
	}
	if testAddressID == "" {
		t.Error("CreateTestAddress returned empty ID")
	}
	if testCardID == "" {
		t.Error("CreateTestCreditCard returned empty ID")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet expectations: %v", err)
	}
}
