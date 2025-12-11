// Package customer provides data access operations for customer entities.
//
// This package implements the repository pattern for customer domain objects,
// handling database operations including CRUD operations, transactions, and
// event publishing through the outbox pattern.
package customer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	events "go-shopping-poc/internal/contracts/events"
	entity "go-shopping-poc/internal/entity/customer"
	"go-shopping-poc/internal/platform/logging"
	outbox "go-shopping-poc/internal/platform/outbox"
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
	InsertCustomer(ctx context.Context, customer *entity.Customer) error
	GetCustomerByEmail(ctx context.Context, email string) (*entity.Customer, error)
	GetCustomerByID(ctx context.Context, customerID string) (*entity.Customer, error)
	UpdateCustomer(ctx context.Context, customer *entity.Customer) error
	PatchCustomer(ctx context.Context, customerID string, patchData *PatchCustomerRequest) error

	AddAddress(ctx context.Context, customerID string, addr *entity.Address) (*entity.Address, error)
	UpdateAddress(ctx context.Context, addressID string, addr *entity.Address) error
	DeleteAddress(ctx context.Context, addressID string) error

	AddCreditCard(ctx context.Context, customerID string, card *entity.CreditCard) (*entity.CreditCard, error)
	UpdateCreditCard(ctx context.Context, cardID string, card *entity.CreditCard) error
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
// operations using sqlx for database interactions and the outbox pattern
// for reliable event publishing.
type customerRepository struct {
	db           *sqlx.DB
	outboxWriter outbox.Writer
}

// NewCustomerRepository creates a new customer repository instance.
//
// Parameters:
//   - db: Database connection for customer data operations
//   - outbox: Writer for publishing domain events via the outbox pattern
//
// Returns a configured customer repository ready for use.
func NewCustomerRepository(db *sqlx.DB, outbox outbox.Writer) *customerRepository {
	return &customerRepository{db: db, outboxWriter: outbox}
}

// InsertCustomer creates a new customer record in the database.
//
// This method handles the complete customer creation process including:
// - Setting default values for missing fields
// - Inserting the customer record
// - Loading and returning related data (addresses, credit cards, status history)
//
// The customer ID is generated automatically if not provided.
func (r *customerRepository) InsertCustomer(ctx context.Context, customer *entity.Customer) error {
	logging.Debug("Repository: Inserting new customer...")

	// Prepare customer with defaults
	r.PrepareCustomerDefaults(customer)

	// Insert customer record and related data in a transaction
	if err := r.insertCustomerWithRelations(ctx, customer); err != nil {
		return fmt.Errorf("failed to insert customer with relations: %w", err)
	}

	return nil
}

// insertCustomerWithRelations handles the complete customer creation process
func (r *customerRepository) insertCustomerWithRelations(ctx context.Context, customer *entity.Customer) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert basic customer record
	if err := r.insertCustomerRecordInTransaction(ctx, tx, customer); err != nil {
		return err
	}

	customerID, err := uuid.Parse(customer.CustomerID)
	if err != nil {
		return fmt.Errorf("%w: invalid customer ID: %v", ErrInvalidUUID, err)
	}

	// Insert provided addresses
	if len(customer.Addresses) > 0 {
		if err := r.insertAddresses(ctx, tx, customer.Addresses, customerID); err != nil {
			return err
		}
	}

	// Insert provided credit cards
	if len(customer.CreditCards) > 0 {
		if err := r.insertCreditCards(ctx, tx, customer.CreditCards, customerID); err != nil {
			return err
		}
	}

	// Insert initial status history
	initialStatus := []entity.CustomerStatus{{
		CustomerID: customerID,
		OldStatus:  "",
		NewStatus:  customer.CustomerStatus,
		ChangedAt:  customer.StatusDateTime,
	}}
	if err := r.insertStatusHistory(ctx, tx, initialStatus, customerID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Load all data back to ensure consistency
	if err := r.LoadCustomerRelations(ctx, customer); err != nil {
		return fmt.Errorf("failed to load customer relations: %w", err)
	}

	return nil
}

// insertCustomerRecordInTransaction inserts customer record within a transaction
func (r *customerRepository) insertCustomerRecordInTransaction(ctx context.Context, tx *sqlx.Tx, customer *entity.Customer) error {
	customerQuery := `INSERT INTO customers.Customer (customer_id, user_name, email, first_name, last_name, phone, customer_since, customer_status, status_date_time) VALUES (:customer_id, :user_name, :email, :first_name, :last_name, :phone, :customer_since, :customer_status, :status_date_time)`

	_, err := tx.NamedExecContext(ctx, customerQuery, customer)
	if err != nil {
		return fmt.Errorf("%w: failed to execute insert query: %v", ErrDatabaseOperation, err)
	}
	return nil
}

func (r *customerRepository) PrepareCustomerDefaults(customer *entity.Customer) {
	newID := uuid.New()
	customer.CustomerID = newID.String()

	if customer.CustomerSince.IsZero() {
		customer.CustomerSince = time.Now()
	}
	if customer.CustomerStatus == "" {
		customer.CustomerStatus = "active"
	}
	if customer.StatusDateTime.IsZero() {
		customer.StatusDateTime = time.Now()
	}
}

// InsertCustomerRecord inserts the customer record into the database
func (r *customerRepository) InsertCustomerRecord(ctx context.Context, customer *entity.Customer) error {
	customerQuery := `INSERT INTO customers.Customer (customer_id, user_name, email, first_name, last_name, phone, customer_since, customer_status, status_date_time) VALUES (:customer_id, :user_name, :email, :first_name, :last_name, :phone, :customer_since, :customer_status, :status_date_time)`

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.NamedExecContext(ctx, customerQuery, customer)
	if err != nil {
		return fmt.Errorf("failed to execute insert query: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// LoadCustomerRelations loads addresses, credit cards, and status history for the customer
func (r *customerRepository) LoadCustomerRelations(ctx context.Context, customer *entity.Customer) error {
	id, err := uuid.Parse(customer.CustomerID)
	if err != nil {
		return fmt.Errorf("invalid customer ID: %w", err)
	}

	addresses, err := r.getAddressesByCustomerID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to load addresses: %w", err)
	}
	customer.Addresses = addresses

	creditCards, err := r.getCreditCardsByCustomerID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to load credit cards: %w", err)
	}
	customer.CreditCards = creditCards

	statusHistory, err := r.getStatusHistoryByCustomerID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to load status history: %w", err)
	}
	customer.StatusHistory = statusHistory

	return nil
}

func (r *customerRepository) GetCustomerByID(ctx context.Context, customerID string) (*entity.Customer, error) {
	logging.Debug("Repository: Fetching customer by ID...")

	id, err := uuid.Parse(customerID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrInvalidUUID, customerID, err)
	}

	query := `select * from customers.customer where customers.customer.customer_id = $1`
	var customer entity.Customer
	if err := r.db.GetContext(ctx, &customer, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		logging.Error("Error fetching customer by ID: %v", err)
		return nil, fmt.Errorf("%w: failed to fetch customer %s: %v", ErrDatabaseOperation, customerID, err)
	}

	addresses, err := r.getAddressesByCustomerID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load addresses for customer %s: %w", customerID, err)
	}
	customer.Addresses = addresses

	creditCards, err := r.getCreditCardsByCustomerID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load credit cards for customer %s: %w", customerID, err)
	}
	customer.CreditCards = creditCards

	statusHistory, err := r.getStatusHistoryByCustomerID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load status history for customer %s: %w", customerID, err)
	}
	customer.StatusHistory = statusHistory

	return &customer, nil
}

func (r *customerRepository) GetCustomerByEmail(ctx context.Context, email string) (*entity.Customer, error) {
	logging.Debug("Repository: Fetching customer by email...")

	query := `SELECT * FROM customers.Customer WHERE email = $1`
	var customer entity.Customer
	if err := r.db.GetContext(ctx, &customer, query, email); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		logging.Error("Error fetching customer by email: %v", err)
		return nil, fmt.Errorf("%w: failed to fetch customer by email %s: %v", ErrDatabaseOperation, email, err)
	}

	id, err := uuid.Parse(customer.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid customer ID %s: %v", ErrInvalidUUID, customer.CustomerID, err)
	}

	addresses, err := r.getAddressesByCustomerID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load addresses for customer %s: %w", email, err)
	}
	customer.Addresses = addresses

	creditCards, err := r.getCreditCardsByCustomerID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load credit cards for customer %s: %w", email, err)
	}
	customer.CreditCards = creditCards

	statusHistory, err := r.getStatusHistoryByCustomerID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to load status history for customer %s: %w", email, err)
	}
	customer.StatusHistory = statusHistory

	return &customer, nil
}

func (r *customerRepository) getAddressesByCustomerID(ctx context.Context, customerID uuid.UUID) ([]entity.Address, error) {
	query := `SELECT * FROM customers.Address WHERE customer_id = $1`
	var addresses []entity.Address
	if err := r.db.SelectContext(ctx, &addresses, query, customerID); err != nil {
		return nil, fmt.Errorf("%w: failed to fetch addresses for customer %s: %v", ErrDatabaseOperation, customerID, err)
	}
	return addresses, nil
}

func (r *customerRepository) getCreditCardsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]entity.CreditCard, error) {
	query := `SELECT * FROM customers.CreditCard WHERE customer_id = $1`
	var creditCards []entity.CreditCard
	if err := r.db.SelectContext(ctx, &creditCards, query, customerID); err != nil {
		return nil, fmt.Errorf("%w: failed to fetch credit cards for customer %s: %v", ErrDatabaseOperation, customerID, err)
	}
	return creditCards, nil
}

func (r *customerRepository) getStatusHistoryByCustomerID(ctx context.Context, customerID uuid.UUID) ([]entity.CustomerStatus, error) {
	query := `SELECT * FROM customers.CustomerStatusHistory WHERE customer_id = $1`
	var statusHistory []entity.CustomerStatus
	if err := r.db.SelectContext(ctx, &statusHistory, query, customerID); err != nil {
		return nil, fmt.Errorf("%w: failed to fetch status history for customer %s: %v", ErrDatabaseOperation, customerID, err)
	}
	return statusHistory, nil
}

// UpdateCustomer performs a complete replacement of customer data (PUT semantics).
//
// This method replaces the entire customer record and all related data:
// - Updates the main customer record
// - Deletes and recreates all addresses
// - Deletes and recreates all credit cards
// - Deletes and recreates status history
// - Publishes a customer updated event
//
// Use PatchCustomer for partial updates instead of complete replacement.
func (r *customerRepository) UpdateCustomer(ctx context.Context, customer *entity.Customer) error {
	logging.Debug("Repository: Updating customer (PUT - complete replace)...")

	// Validate required fields
	if err := r.validateCustomerForUpdate(customer); err != nil {
		return err
	}

	// Parse customer ID
	id, err := uuid.Parse(customer.CustomerID)
	if err != nil {
		return fmt.Errorf("%w: invalid customer ID %s: %v", ErrInvalidUUID, customer.CustomerID, err)
	}

	// Execute update in transaction
	return r.executeCustomerUpdate(ctx, customer, id)
}

// validateCustomerForUpdate validates that required fields are present for a PUT update
func (r *customerRepository) validateCustomerForUpdate(customer *entity.Customer) error {
	if customer.CustomerID == "" || customer.Username == "" || customer.Email == "" {
		return fmt.Errorf("PUT requires complete customer record with customer_id, username, and email")
	}
	return nil
}

// executeCustomerUpdate performs the actual database update within a transaction
func (r *customerRepository) executeCustomerUpdate(ctx context.Context, customer *entity.Customer, id uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", ErrTransactionFailed, err)
	}
	defer tx.Rollback()

	// Update customer record
	if err := r.updateCustomerRecord(ctx, tx, customer); err != nil {
		return err
	}

	// Replace related records
	if err := r.replaceCustomerRelations(ctx, tx, customer, id); err != nil {
		return err
	}

	// Publish event and commit
	if err := r.publishCustomerUpdateEvent(ctx, tx, customer); err != nil {
		return err
	}

	return tx.Commit()
}

// updateCustomerRecord updates the main customer record
func (r *customerRepository) updateCustomerRecord(ctx context.Context, tx *sqlx.Tx, customer *entity.Customer) error {
	customerQuery := `UPDATE customers.Customer
		SET user_name = :user_name, email = :email, first_name = :first_name,
		last_name = :last_name, phone = :phone, customer_since = :customer_since,
		customer_status = :customer_status, status_date_time = :status_date_time,
		default_shipping_address_id = :default_shipping_address_id,
		default_billing_address_id = :default_billing_address_id,
		default_credit_card_id = :default_credit_card_id
		WHERE customer_id = :customer_id`

	result, err := tx.NamedExecContext(ctx, customerQuery, customer)
	if err != nil {
		return fmt.Errorf("%w: failed to update customer record: %v", ErrDatabaseOperation, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: failed to get rows affected: %v", ErrDatabaseOperation, err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: customer not found: %s", ErrCustomerNotFound, customer.CustomerID)
	}

	return nil
}

// replaceCustomerRelations replaces all addresses, credit cards, and status history
func (r *customerRepository) replaceCustomerRelations(ctx context.Context, tx *sqlx.Tx, customer *entity.Customer, id uuid.UUID) error {
	// Delete existing relations
	if err := r.deleteExistingRelations(ctx, tx, id); err != nil {
		return err
	}

	// Insert new relations
	if err := r.insertNewRelations(ctx, tx, customer, id); err != nil {
		return err
	}

	return nil
}

// deleteExistingRelations removes all existing related records
func (r *customerRepository) deleteExistingRelations(ctx context.Context, tx *sqlx.Tx, customerID uuid.UUID) error {
	queries := []string{
		`DELETE FROM customers.Address WHERE customer_id = $1`,
		`DELETE FROM customers.CreditCard WHERE customer_id = $1`,
		`DELETE FROM customers.CustomerStatusHistory WHERE customer_id = $1`,
	}

	for _, query := range queries {
		if _, err := tx.ExecContext(ctx, query, customerID); err != nil {
			return fmt.Errorf("%w: failed to delete existing relations: %v", ErrDatabaseOperation, err)
		}
	}

	return nil
}

// deleteAddresses removes all existing addresses for a customer
func (r *customerRepository) deleteAddresses(ctx context.Context, tx *sqlx.Tx, customerID uuid.UUID) error {
	query := `DELETE FROM customers.Address WHERE customer_id = $1`
	_, err := tx.ExecContext(ctx, query, customerID)
	if err != nil {
		return fmt.Errorf("%w: failed to delete addresses: %v", ErrDatabaseOperation, err)
	}
	return nil
}

// deleteCreditCards removes all existing credit cards for a customer
func (r *customerRepository) deleteCreditCards(ctx context.Context, tx *sqlx.Tx, customerID uuid.UUID) error {
	query := `DELETE FROM customers.CreditCard WHERE customer_id = $1`
	_, err := tx.ExecContext(ctx, query, customerID)
	if err != nil {
		return fmt.Errorf("%w: failed to delete credit cards: %v", ErrDatabaseOperation, err)
	}
	return nil
}

// insertNewRelations inserts new addresses, credit cards, and status history
func (r *customerRepository) insertNewRelations(ctx context.Context, tx *sqlx.Tx, customer *entity.Customer, id uuid.UUID) error {
	if err := r.insertAddresses(ctx, tx, customer.Addresses, id); err != nil {
		return err
	}

	if err := r.insertCreditCards(ctx, tx, customer.CreditCards, id); err != nil {
		return err
	}

	if err := r.insertStatusHistory(ctx, tx, customer.StatusHistory, id); err != nil {
		return err
	}

	return nil
}

// insertAddresses inserts new address records
func (r *customerRepository) insertAddresses(ctx context.Context, tx *sqlx.Tx, addresses []entity.Address, customerID uuid.UUID) error {
	addressQuery := `INSERT INTO customers.Address (
		address_id, customer_id, address_type, first_name, last_name,
		address_1, address_2, city, state, zip
	) VALUES (
		:address_id, :customer_id, :address_type, :first_name, :last_name,
		:address_1, :address_2, :city, :state, :zip
	)`

	for i := range addresses {
		addresses[i].CustomerID = customerID
		addresses[i].AddressID = uuid.New()
		if _, err := tx.NamedExecContext(ctx, addressQuery, &addresses[i]); err != nil {
			return fmt.Errorf("%w: failed to insert address: %v", ErrDatabaseOperation, err)
		}
	}

	return nil
}

// insertCreditCards inserts new credit card records
func (r *customerRepository) insertCreditCards(ctx context.Context, tx *sqlx.Tx, cards []entity.CreditCard, customerID uuid.UUID) error {
	cardQuery := `INSERT INTO customers.CreditCard (
		card_id, customer_id, card_type, card_number, card_holder_name,
		card_expires, card_cvv
	) VALUES (
		:card_id, :customer_id, :card_type, :card_number, :card_holder_name,
		:card_expires, :card_cvv
	)`

	for i := range cards {
		cards[i].CustomerID = customerID
		cards[i].CardID = uuid.New()
		if _, err := tx.NamedExecContext(ctx, cardQuery, &cards[i]); err != nil {
			return fmt.Errorf("%w: failed to insert credit card: %v", ErrDatabaseOperation, err)
		}
	}

	return nil
}

// insertStatusHistory inserts new status history records
func (r *customerRepository) insertStatusHistory(ctx context.Context, tx *sqlx.Tx, history []entity.CustomerStatus, customerID uuid.UUID) error {
	statusQuery := `INSERT INTO customers.CustomerStatusHistory (
		customer_id, old_status, new_status, changed_at
	) VALUES (
		:customer_id, :old_status, :new_status, :changed_at
	)`

	for _, status := range history {
		status.CustomerID = customerID
		if status.ChangedAt.IsZero() {
			status.ChangedAt = time.Now()
		}
		if _, err := tx.NamedExecContext(ctx, statusQuery, status); err != nil {
			return fmt.Errorf("%w: failed to insert status history: %v", ErrDatabaseOperation, err)
		}
	}

	return nil
}

// publishCustomerUpdateEvent publishes the customer update event
func (r *customerRepository) publishCustomerUpdateEvent(ctx context.Context, tx *sqlx.Tx, customer *entity.Customer) error {
	customerEvent := events.NewCustomerUpdatedEvent(customer.CustomerID, nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, customerEvent); err != nil {
		return fmt.Errorf("failed to publish customer update event: %w", err)
	}
	return nil
}

// PatchCustomer applies partial updates to customer data (PATCH semantics).
//
// This method allows selective updates to customer fields without replacing
// the entire record. It supports updating basic fields, default addresses/cards,
// and replacing addresses/credit cards arrays.
//
// Only non-nil fields in patchData will be updated.
func (r *customerRepository) PatchCustomer(ctx context.Context, customerID string, patchData *PatchCustomerRequest) error {
	logging.Debug("Repository: Patching customer %s", customerID)

	// Get existing customer first
	existing, err := r.GetCustomerByID(ctx, customerID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("customer not found: %s", customerID)
	}

	// Apply patch data to existing customer
	updated := *existing

	// Apply basic field updates
	if patchData.UserName != nil {
		updated.Username = *patchData.UserName
	}
	if patchData.Email != nil {
		updated.Email = *patchData.Email
	}
	if patchData.FirstName != nil {
		updated.FirstName = *patchData.FirstName
	}
	if patchData.LastName != nil {
		updated.LastName = *patchData.LastName
	}
	if patchData.Phone != nil {
		updated.Phone = *patchData.Phone
	}
	if patchData.CustomerStatus != nil {
		updated.CustomerStatus = *patchData.CustomerStatus
	}

	// Handle UUID pointer fields
	if patchData.DefaultShippingAddressID != nil {
		if *patchData.DefaultShippingAddressID == "" {
			updated.DefaultShippingAddressID = nil
		} else {
			if uuid, err := uuid.Parse(*patchData.DefaultShippingAddressID); err == nil {
				updated.DefaultShippingAddressID = &uuid
			}
		}
	}
	if patchData.DefaultBillingAddressID != nil {
		if *patchData.DefaultBillingAddressID == "" {
			updated.DefaultBillingAddressID = nil
		} else {
			if uuid, err := uuid.Parse(*patchData.DefaultBillingAddressID); err == nil {
				updated.DefaultBillingAddressID = &uuid
			}
		}
	}
	if patchData.DefaultCreditCardID != nil {
		if *patchData.DefaultCreditCardID == "" {
			updated.DefaultCreditCardID = nil
		} else {
			if uuid, err := uuid.Parse(*patchData.DefaultCreditCardID); err == nil {
				updated.DefaultCreditCardID = &uuid
			}
		}
	}

	// Prepare new addresses and credit cards for insertion if provided
	var newAddresses []entity.Address
	if patchData.Addresses != nil {
		for _, patchAddr := range patchData.Addresses {
			addr := entity.Address{
				AddressType: patchAddr.AddressType,
				FirstName:   patchAddr.FirstName,
				LastName:    patchAddr.LastName,
				Address1:    patchAddr.Address1,
				Address2:    patchAddr.Address2,
				City:        patchAddr.City,
				State:       patchAddr.State,
				Zip:         patchAddr.Zip,
			}
			newAddresses = append(newAddresses, addr)
		}
	}

	var newCreditCards []entity.CreditCard
	if patchData.CreditCards != nil {
		for _, patchCard := range patchData.CreditCards {
			card := entity.CreditCard{
				CardType:       patchCard.CardType,
				CardNumber:     patchCard.CardNumber,
				CardHolderName: patchCard.CardHolderName,
				CardExpires:    patchCard.CardExpires,
				CardCVV:        patchCard.CardCVV,
			}
			newCreditCards = append(newCreditCards, card)
		}
	}

	// Parse customer ID
	id, err := uuid.Parse(customerID)
	if err != nil {
		return fmt.Errorf("%w: invalid customer ID %s: %v", ErrInvalidUUID, customerID, err)
	}

	// Execute selective patch update
	return r.executePatchCustomer(ctx, &updated, id, newAddresses, newCreditCards)
}

// executePatchCustomer performs selective PATCH updates in a transaction
func (r *customerRepository) executePatchCustomer(ctx context.Context, customer *entity.Customer, id uuid.UUID, newAddresses []entity.Address, newCreditCards []entity.CreditCard) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", ErrTransactionFailed, err)
	}
	defer tx.Rollback()

	// Update customer record
	if err := r.updateCustomerRecord(ctx, tx, customer); err != nil {
		return err
	}

	// Replace addresses only if provided
	if len(newAddresses) > 0 {
		if err := r.deleteAddresses(ctx, tx, id); err != nil {
			return err
		}
		if err := r.insertAddresses(ctx, tx, newAddresses, id); err != nil {
			return err
		}
	}

	// Replace credit cards only if provided
	if len(newCreditCards) > 0 {
		if err := r.deleteCreditCards(ctx, tx, id); err != nil {
			return err
		}
		if err := r.insertCreditCards(ctx, tx, newCreditCards, id); err != nil {
			return err
		}
	}

	// Publish event
	if err := r.publishCustomerUpdateEvent(ctx, tx, customer); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *customerRepository) AddAddress(ctx context.Context, customerID string, addr *entity.Address) (*entity.Address, error) {
	logging.Debug("Repository: Adding address for customer %s", customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}

	addr.CustomerID = custUUID
	addr.AddressID = uuid.New()

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
        INSERT INTO customers.Address (
            address_id, customer_id, address_type, first_name, last_name,
            address_1, address_2, city, state, zip
        ) VALUES (
            :address_id, :customer_id, :address_type, :first_name, :last_name,
            :address_1, :address_2, :city, :state, :zip
        )`

	params := map[string]any{
		"address_id":   addr.AddressID,
		"customer_id":  addr.CustomerID,
		"address_type": addr.AddressType,
		"first_name":   addr.FirstName,
		"last_name":    addr.LastName,
		"address_1":    addr.Address1,
		"address_2":    addr.Address2,
		"city":         addr.City,
		"state":        addr.State,
		"zip":          addr.Zip,
	}

	if _, err := tx.NamedExecContext(ctx, query, params); err != nil {
		return nil, err
	}

	// emit outbox event inside same transaction

	evt := events.NewAddressAddedEvent(customerID, addr.AddressID.String(), map[string]string{"address_type": addr.AddressType})
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return addr, nil
}

func (r *customerRepository) UpdateAddress(ctx context.Context, addressID string, addr *entity.Address) error {
	logging.Debug("Repository: Updating address %s", addressID)

	id, err := uuid.Parse(addressID)
	if err != nil {
		return err
	}
	addr.AddressID = id

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
        UPDATE customers.Address
        SET first_name = :first_name,
            last_name = :last_name,
            address_1 = :address_1,
            address_2 = :address_2,
            city = :city,
            state = :state,
            zip = :zip,
			address_type = :address_type
        WHERE address_id = :address_id`

	res, err := tx.NamedExecContext(ctx, query, addr)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("address not found: %s", addressID)
	}

	evt := events.NewAddressUpdatedEvent(addr.CustomerID.String(), addr.AddressID.String(), nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *customerRepository) DeleteAddress(ctx context.Context, addressID string) error {
	logging.Debug("Repository: Deleting address with ID %s", addressID)

	id, err := uuid.Parse(addressID)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `DELETE FROM customers.Address WHERE address_id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("address not found: %s", addressID)
	}

	evt := events.NewAddressDeletedEvent("", addressID, nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *customerRepository) AddCreditCard(ctx context.Context, customerID string, card *entity.CreditCard) (*entity.CreditCard, error) {
	logging.Debug("Repository: Adding credit card for customer %s", customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}

	card.CustomerID = custUUID
	card.CardID = uuid.New()

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
        INSERT INTO customers.CreditCard (
            card_id, customer_id, card_number, card_type, card_holder_name,
            card_expires, card_cvv
        ) VALUES (
            :card_id, :customer_id, :card_number, :card_type, :card_holder_name,
            :card_expires, :card_cvv
        )`

	params := map[string]any{
		"card_id":          card.CardID,
		"customer_id":      card.CustomerID,
		"card_number":      card.CardNumber,
		"card_type":        card.CardType,
		"card_holder_name": card.CardHolderName,
		"card_expires":     card.CardExpires,
		"card_cvv":         card.CardCVV,
	}

	if _, err := tx.NamedExecContext(ctx, query, params); err != nil {
		return nil, err
	}

	evt := events.NewCardAddedEvent(card.CustomerID.String(), card.CardID.String(), map[string]string{
		"card_number": card.CardNumber,
	})
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return card, nil
}

func (r *customerRepository) UpdateCreditCard(ctx context.Context, cardID string, card *entity.CreditCard) error {
	logging.Debug("Repository: Updating credit card %s", cardID)

	id, err := uuid.Parse(cardID)
	if err != nil {
		return err
	}
	card.CardID = id

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
        UPDATE customers.CreditCard
        SET card_type = :card_type,
            card_holder_name = :card_holder_name,
            card_expires = :card_expires,
            card_cvv = :card_cvv
        WHERE card_id = :card_id`

	res, err := tx.NamedExecContext(ctx, query, card)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("credit card not found: %s", cardID)
	}

	evt := events.NewCardUpdatedEvent(card.CustomerID.String(), card.CardID.String(), nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *customerRepository) DeleteCreditCard(ctx context.Context, cardID string) error {
	logging.Debug("Repository: Deleting credit card with ID %s", cardID)

	id, err := uuid.Parse(cardID)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `DELETE FROM customers.CreditCard WHERE card_id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("credit card not found: %s", cardID)
	}

	evt := events.NewCardDeletedEvent("", cardID, nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *customerRepository) UpdateDefaultShippingAddress(ctx context.Context, customerID, addressID string) error {
	logging.Debug("Repository: Setting default shipping address %s for customer %s", addressID, customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}

	addrUUID, err := uuid.Parse(addressID)
	if err != nil {
		return err
	}

	query := `UPDATE customers.Customer SET default_shipping_address_id = $1 WHERE customer_id = $2`

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, query, addrUUID, custUUID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("customer not found: %s", customerID)
	}

	evt := events.NewDefaultShippingAddressChangedEvent(customerID, addressID, nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *customerRepository) UpdateDefaultBillingAddress(ctx context.Context, customerID, addressID string) error {
	logging.Debug("Repository: Setting default billing address %s for customer %s", addressID, customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}

	addrUUID, err := uuid.Parse(addressID)
	if err != nil {
		return err
	}

	query := `UPDATE customers.Customer SET default_billing_address_id = $1 WHERE customer_id = $2`

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, query, addrUUID, custUUID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("customer not found: %s", customerID)
	}

	evt := events.NewDefaultBillingAddressChangedEvent(customerID, addressID, nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *customerRepository) UpdateDefaultCreditCard(ctx context.Context, customerID, cardID string) error {
	logging.Debug("Repository: Setting default credit card %s for customer %s", cardID, customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}

	cardUUID, err := uuid.Parse(cardID)
	if err != nil {
		return err
	}

	query := `UPDATE customers.Customer SET default_credit_card_id = $1 WHERE customer_id = $2`

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, query, cardUUID, custUUID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("customer not found: %s", customerID)
	}

	evt := events.NewDefaultCreditCardChangedEvent(customerID, cardID, nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *customerRepository) ClearDefaultShippingAddress(ctx context.Context, customerID string) error {
	logging.Debug("Repository: Clearing default shipping address for customer %s", customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}

	query := `UPDATE customers.Customer SET default_shipping_address_id = NULL WHERE customer_id = $1`

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, query, custUUID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("customer not found: %s", customerID)
	}

	evt := events.NewDefaultShippingAddressChangedEvent(customerID, "", nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *customerRepository) ClearDefaultBillingAddress(ctx context.Context, customerID string) error {
	logging.Debug("Repository: Clearing default billing address for customer %s", customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}

	query := `UPDATE customers.Customer SET default_billing_address_id = NULL WHERE customer_id = $1`

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, query, custUUID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("customer not found: %s", customerID)
	}

	evt := events.NewDefaultBillingAddressChangedEvent(customerID, "", nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *customerRepository) ClearDefaultCreditCard(ctx context.Context, customerID string) error {
	logging.Debug("Repository: Clearing default credit card for customer %s", customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}

	query := `UPDATE customers.Customer SET default_credit_card_id = NULL WHERE customer_id = $1`

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, query, custUUID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("customer not found: %s", customerID)
	}

	evt := events.NewDefaultCreditCardChangedEvent(customerID, "", nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return err
	}

	return tx.Commit()
}
