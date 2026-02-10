// Package customer provides data access operations for customer entities.
//
// This package implements the repository pattern for customer domain objects,
// handling database operations including CRUD operations, transactions, and
// event publishing through the outbox pattern.
//
// The repository is split into multiple files:
// - repository.go: Interface and struct definitions
// - repository_crud.go: Customer CRUD operations
// - repository_query.go: Query operations
// - repository_address.go: Address operations
// - repository_creditcard.go: Credit card operations
// - repository_util.go: Utility functions
package customer

import (
	"context"
	"errors"

	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/outbox"
)

// Custom error types for different repository failure scenarios
var (
	ErrCustomerNotFound   = errors.New("customer not found")
	ErrAddressNotFound    = errors.New("address not found")
	ErrCreditCardNotFound = errors.New("credit card not found")
	ErrInvalidUUID        = errors.New("invalid UUID format")
	ErrDatabaseOperation  = errors.New("database operation failed")
	ErrTransactionFailed  = errors.New("transaction failed")
)

// CustomerRepository defines the contract for customer data access operations.
//
// This interface abstracts database operations for customer entities,
// providing a clean separation between business logic and data persistence.
// All methods accept a context for proper request tracing and cancellation.
//
// Database Constraints:
// - Deleting a customer cascades to addresses and credit cards (ON DELETE CASCADE)
// - Deleting addresses/credit cards sets default_*_id fields to NULL (ON DELETE SET NULL)
// - These behaviors are enforced at the database level per schema constraints
type CustomerRepository interface {
	InsertCustomer(ctx context.Context, customer *Customer) error
	GetCustomerByEmail(ctx context.Context, email string) (*Customer, error)
	GetCustomerByID(ctx context.Context, customerID string) (*Customer, error)
	UpdateCustomer(ctx context.Context, customer *Customer) error
	PatchCustomer(ctx context.Context, customerID string, patchData *PatchCustomerRequest) error

	AddAddress(ctx context.Context, customerID string, addr *Address) (*Address, error)
	UpdateAddress(ctx context.Context, addressID string, addr *Address) error
	DeleteAddress(ctx context.Context, addressID string) error

	AddCreditCard(ctx context.Context, customerID string, card *CreditCard) (*CreditCard, error)
	UpdateCreditCard(ctx context.Context, cardID string, card *CreditCard) error
	DeleteCreditCard(ctx context.Context, cardID string) error

	// Default address and credit card management
	UpdateDefaultShippingAddress(ctx context.Context, customerID, addressID string) error
	UpdateDefaultBillingAddress(ctx context.Context, customerID, addressID string) error
	UpdateDefaultCreditCard(ctx context.Context, customerID, cardID string) error
	ClearDefaultShippingAddress(ctx context.Context, customerID string) error
	ClearDefaultBillingAddress(ctx context.Context, customerID string) error
	ClearDefaultCreditCard(ctx context.Context, customerID string) error
	//DeleteCustomer(ctx context.Context, id uuid.UUID) error
}

// customerRepository implements CustomerRepository using PostgreSQL.
//
// This struct provides the concrete implementation of customer data access
// operations using the platform database abstraction for database interactions
// and the outbox pattern for reliable event publishing.
type customerRepository struct {
	db           database.Database
	outboxWriter *outbox.Writer
}

// NewCustomerRepository creates a new customer repository instance.
//
// Parameters:
//   - db: Database connection for customer data operations
//   - outbox: Writer for publishing domain events via the outbox pattern
//
// Returns a configured customer repository ready for use.
func NewCustomerRepository(db database.Database, outbox *outbox.Writer) *customerRepository {
	return &customerRepository{db: db, outboxWriter: outbox}
}
