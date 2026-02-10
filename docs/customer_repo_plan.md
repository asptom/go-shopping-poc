# Customer Repository Refactoring Plan

## Overview

The `internal/service/customer/repository.go` file (1392 lines) has become too large and unwieldy. Following the pattern established for `internal/service/product`, this file will be split into multiple focused files based on functionality.

## Target File Structure

```
internal/service/customer/
├── repository.go              (Interface + struct - 88 lines)
├── repository_crud.go         (Customer CRUD operations - 475 lines)
├── repository_query.go        (Query operations - 90 lines)
├── repository_address.go      (Address CRUD + defaults - 380 lines)
├── repository_creditcard.go   (CreditCard CRUD + defaults - 275 lines)
└── repository_util.go         (Utility functions - 47 lines)
```

Total: 1355 lines across 6 files

Note: The actual implementation is larger than originally planned because the refactoring preserved all existing functionality including duplicates that were in the original 1392-line file. The original file contained `InsertCustomerRecord()` which is not part of the interface but was kept for compatibility.

---

## File 1: repository.go

**Purpose**: Interface definitions and repository struct

**Keep**:
- Package documentation
- Custom error types:
  - `ErrCustomerNotFound`
  - `ErrAddressNotFound`
  - `ErrCreditCardNotFound`
  - `ErrInvalidUUID`
  - `ErrDatabaseOperation`
  - `ErrTransactionFailed`
- `CustomerRepository` interface (all method declarations)
- `customerRepository` struct
- `NewCustomerRepository()` constructor

**Remove**: All method implementations

**New file content** (~80 lines):

```go
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

	"github.com/google/uuid"

	events "go-shopping-poc/internal/contracts/events"

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

	UpdateDefaultShippingAddress(ctx context.Context, customerID, addressID string) error
	UpdateDefaultBillingAddress(ctx context.Context, customerID, addressID string) error
	UpdateDefaultCreditCard(ctx context.Context, customerID, cardID string) error
	ClearDefaultShippingAddress(ctx context.Context, customerID string) error
	ClearDefaultBillingAddress(ctx context.Context, customerID string) error
	ClearDefaultCreditCard(ctx context.Context, customerID string) error
}

// customerRepository implements CustomerRepository using PostgreSQL.
type customerRepository struct {
	db           database.Database
	outboxWriter *outbox.Writer
}

// NewCustomerRepository creates a new customer repository instance.
func NewCustomerRepository(db database.Database, outbox *outbox.Writer) *customerRepository {
	return &customerRepository{db: db, outboxWriter: outbox}
}
```

---

## File 2: repository_crud.go

**Purpose**: Customer CRUD operations (Insert, Update, Delete, Get with relations)

**Move from original repository.go**:

### Public Methods
- `InsertCustomer()` - line 98
- `UpdateCustomer()` - line 384
- `PatchCustomer()` - line 621

### Helper Methods (called by CRUD)
- `insertCustomerWithRelations()` - line 113
- `insertCustomerRecordInTransaction()` - line 181
- `PrepareCustomerDefaults()` - line 191 (MAKE PRIVATE: `prepareCustomerDefaults()`)
- `validateCustomerForUpdate()` - line 403
- `executeCustomerUpdate()` - line 411
- `replaceCustomerRelations()` - line 474
- `deleteExistingRelations()` - line 489
- `insertNewRelations()` - line 526
- `insertAddresses()` - line 543
- `insertCreditCards()` - line 564
- `insertStatusHistory()` - line 585 (MOVE TO util)
- `executePatchCustomer()` - line 728
- `LoadCustomerRelations()` - line 242 (MOVE TO query or keep here?)
- `getAddressesByCustomerID()` - line 347 (MOVE TO query)
- `getCreditCardsByCustomerID()` - line 356 (MOVE TO query)
- `getStatusHistoryByCustomerID()` - line 365 (MOVE TO query)

**Note**: Need to decide whether `LoadCustomerRelations()` and related getters stay in CRUD or move to query file. 

**Recommendation**: Keep `LoadCustomerRelations()` in CRUD since it's primarily called after CRUD operations. Move the three getter methods to `repository_query.go`.

**New file content** (~280 lines):

```go
// Package customer provides data access operations for customer entities.
//
// This file contains CRUD (Create, Read, Update, Delete) operations
// for customer entities, handling database operations with proper transaction
// management and validation.
package customer

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	events "go-shopping-poc/internal/contracts/events"

	"go-shopping-poc/internal/platform/database"
)

// InsertCustomer creates a new customer record in the database.
func (r *customerRepository) InsertCustomer(ctx context.Context, customer *Customer) error {
	log.Printf("[DEBUG] Repository: Inserting new customer...")

	r.prepareCustomerDefaults(customer)

	if err := r.insertCustomerWithRelations(ctx, customer); err != nil {
		return fmt.Errorf("failed to insert customer with relations: %w", err)
	}

	return nil
}

// insertCustomerWithRelations handles the complete customer creation process
func (r *customerRepository) insertCustomerWithRelations(ctx context.Context, customer *Customer) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := r.insertCustomerRecordInTransaction(ctx, tx, customer); err != nil {
		return err
	}

	customerID, err := uuid.Parse(customer.CustomerID)
	if err != nil {
		return fmt.Errorf("%w: invalid customer ID: %v", ErrInvalidUUID, err)
	}

	if len(customer.Addresses) > 0 {
		if err := r.insertAddresses(ctx, tx, customer.Addresses, customerID); err != nil {
			return err
		}
	}

	if len(customer.CreditCards) > 0 {
		if err := r.insertCreditCards(ctx, tx, customer.CreditCards, customerID); err != nil {
			return err
		}
	}

	initialStatus := []CustomerStatus{{
		CustomerID: customerID,
		OldStatus:  "",
		NewStatus:  customer.CustomerStatus,
		ChangedAt:  customer.StatusDateTime,
	}}
	if err := insertStatusHistory(ctx, tx, initialStatus, customerID); err != nil {
		return err
	}

	evt := events.NewCustomerCreatedEvent(customer.CustomerID, nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return fmt.Errorf("failed to publish customer created event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true

	if err := r.LoadCustomerRelations(ctx, customer); err != nil {
		return fmt.Errorf("failed to load customer relations: %w", err)
	}

	return nil
}

// insertCustomerRecordInTransaction inserts customer record within a transaction
func (r *customerRepository) insertCustomerRecordInTransaction(ctx context.Context, tx database.Tx, customer *Customer) error {
	customerQuery := `INSERT INTO customers.Customer (customer_id, user_name, email, first_name, last_name, phone, customer_since, customer_status, status_date_time) VALUES (:customer_id, :user_name, :email, :first_name, :last_name, :phone, :customer_since, :customer_status, :status_date_time)`

	_, err := tx.NamedExecContext(ctx, customerQuery, customer)
	if err != nil {
		return fmt.Errorf("%w: failed to execute insert query: %v", ErrDatabaseOperation, err)
	}
	return nil
}

func (r *customerRepository) prepareCustomerDefaults(customer *Customer) {
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

func (r *customerRepository) UpdateCustomer(ctx context.Context, customer *Customer) error {
	log.Printf("[DEBUG] Repository: Updating customer (PUT - complete replace)...")

	if err := r.validateCustomerForUpdate(customer); err != nil {
		return err
	}

	id, err := uuid.Parse(customer.CustomerID)
	if err != nil {
		return fmt.Errorf("%w: invalid customer ID %s: %v", ErrInvalidUUID, customer.CustomerID, err)
	}

	return r.executeCustomerUpdate(ctx, customer, id)
}

func (r *customerRepository) validateCustomerForUpdate(customer *Customer) error {
	if customer.CustomerID == "" || customer.Username == "" || customer.Email == "" {
		return fmt.Errorf("PUT requires complete customer record with customer_id, username, and email")
	}
	return nil
}

func (r *customerRepository)executeCustomerUpdate(ctx context.Context, customer *Customer, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", ErrTransactionFailed, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := r.updateCustomerRecord(ctx, tx, customer); err != nil {
		return err
	}

	if err := r.replaceCustomerRelations(ctx, tx, customer, id); err != nil {
		return err
	}

	if err := r.publishCustomerUpdateEvent(ctx, tx, customer); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}

func (r *customerRepository) updateCustomerRecord(ctx context.Context, tx database.Tx, customer *Customer) error {
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

func (r *customerRepository) replaceCustomerRelations(ctx context.Context, tx database.Tx, customer *Customer, id uuid.UUID) error {
	if err := r.deleteExistingRelations(ctx, tx, id); err != nil {
		return err
	}

	if err := r.insertNewRelations(ctx, tx, customer, id); err != nil {
		return err
	}

	return nil
}

func (r *customerRepository) deleteExistingRelations(ctx context.Context, tx database.Tx, customerID uuid.UUID) error {
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

func (r *customerRepository) insertNewRelations(ctx context.Context, tx database.Tx, customer *Customer, id uuid.UUID) error {
	if err := r.insertAddresses(ctx, tx, customer.Addresses, id); err != nil {
		return err
	}

	if err := r.insertCreditCards(ctx, tx, customer.CreditCards, id); err != nil {
		return err
	}

	if err := insertStatusHistory(ctx, tx, customer.StatusHistory, id); err != nil {
		return err
	}

	return nil
}

func (r *customerRepository) insertAddresses(ctx context.Context, tx database.Tx, addresses []Address, customerID uuid.UUID) error {
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

func (r *customerRepository) insertCreditCards(ctx context.Context, tx database.Tx, cards []CreditCard, customerID uuid.UUID) error {
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

// PatchCustomer applies partial updates to customer data (PATCH semantics).
func (r *customerRepository) PatchCustomer(ctx context.Context, customerID string, patchData *PatchCustomerRequest) error {
	log.Printf("[DEBUG] Repository: Patching customer %s", customerID)

	existing, err := r.GetCustomerByID(ctx, customerID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("customer not found: %s", customerID)
	}

	updated := *existing

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

	var newAddresses []Address
	if patchData.Addresses != nil {
		for _, patchAddr := range patchData.Addresses {
			addr := Address{
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

	var newCreditCards []CreditCard
	if patchData.CreditCards != nil {
		for _, patchCard := range patchData.CreditCards {
			card := CreditCard{
				CardType:       patchCard.CardType,
				CardNumber:     patchCard.CardNumber,
				CardHolderName: patchCard.CardHolderName,
				CardExpires:    patchCard.CardExpires,
				CardCVV:        patchCard.CardCVV,
			}
			newCreditCards = append(newCreditCards, card)
		}
	}

	id, err := uuid.Parse(customerID)
	if err != nil {
		return fmt.Errorf("%w: invalid customer ID %s: %v", ErrInvalidUUID, customerID, err)
	}

	return r.executePatchCustomer(ctx, &updated, id, newAddresses, newCreditCards)
}

func (r *customerRepository) executePatchCustomer(ctx context.Context, customer *Customer, id uuid.UUID, newAddresses []Address, newCreditCards []CreditCard) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", ErrTransactionFailed, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := r.updateCustomerRecord(ctx, tx, customer); err != nil {
		return err
	}

	if len(newAddresses) > 0 {
		if err := r.deleteAddresses(ctx, tx, id); err != nil {
			return err
		}
		if err := r.insertAddresses(ctx, tx, newAddresses, id); err != nil {
			return err
		}
	}

	if len(newCreditCards) > 0 {
		if err := r.deleteCreditCards(ctx, tx, id); err != nil {
			return err
		}
		if err := r.insertCreditCards(ctx, tx, newCreditCards, id); err != nil {
			return err
		}
	}

	if err := r.publishCustomerUpdateEvent(ctx, tx, customer); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}

func (r *customerRepository) deleteAddresses(ctx context.Context, tx database.Tx, customerID uuid.UUID) error {
	query := `DELETE FROM customers.Address WHERE customer_id = $1`
	_, err := tx.ExecContext(ctx, query, customerID)
	if err != nil {
		return fmt.Errorf("%w: failed to delete addresses: %v", ErrDatabaseOperation, err)
	}
	return nil
}

func (r *customerRepository) deleteCreditCards(ctx context.Context, tx database.Tx, customerID uuid.UUID) error {
	query := `DELETE FROM customers.CreditCard WHERE customer_id = $1`
	_, err := tx.ExecContext(ctx, query, customerID)
	if err != nil {
		return fmt.Errorf("%w: failed to delete credit cards: %v", ErrDatabaseOperation, err)
	}
	return nil
}

func (r *customerRepository) LoadCustomerRelations(ctx context.Context, customer *Customer) error {
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
```

---

## File 3: repository_query.go

**Purpose**: Query operations (Get methods that fetch data)

**Move from original repository.go**:

### Public Methods
- `GetCustomerByID()` - line 269 (with relations loading - this loads addresses/cards/history)
- `GetCustomerByEmail()` - line 308 (with relations loading)

### Private Helper Methods (for queries)
- `getAddressesByCustomerID()` - line 347
- `getCreditCardsByCustomerID()` - line 356
- `getStatusHistoryByCustomerID()` - line 365

**New file content** (~100 lines):

```go
// Package customer provides data access operations for customer entities.
//
// This file contains query operations for fetching customers and their related data.
package customer

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/google/uuid"
)

// GetCustomerByID retrieves a customer by ID with all related data.
func (r *customerRepository) GetCustomerByID(ctx context.Context, customerID string) (*Customer, error) {
	log.Printf("[DEBUG] Repository: Fetching customer by ID...")

	id, err := uuid.Parse(customerID)
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %v", ErrInvalidUUID, customerID, err)
	}

	query := `select * from customers.customer where customers.customer.customer_id = $1`
	var customer Customer
	if err := r.db.GetContext(ctx, &customer, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Printf("[ERROR] Error fetching customer by ID: %v", err)
		return nil, fmt.Errorf("%w: failed to fetch customer %s: %v", ErrDatabaseOperation, customerID, err)
	}

	if err := r.LoadCustomerRelations(ctx, &customer); err != nil {
		return nil, fmt.Errorf("failed to load customer relations for %s: %w", customerID, err)
	}

	return &customer, nil
}

// GetCustomerByEmail retrieves a customer by email with all related data.
func (r *customerRepository) GetCustomerByEmail(ctx context.Context, email string) (*Customer, error) {
	log.Printf("[DEBUG] Repository: Fetching customer by email...")

	query := `SELECT * FROM customers.Customer WHERE email = $1`
	var customer Customer
	if err := r.db.GetContext(ctx, &customer, query, email); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		log.Printf("[ERROR] Error fetching customer by email: %v", err)
		return nil, fmt.Errorf("%w: failed to fetch customer by email %s: %v", ErrDatabaseOperation, email, err)
	}

	id, err := uuid.Parse(customer.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid customer ID %s: %v", ErrInvalidUUID, customer.CustomerID, err)
	}

	if err := r.LoadCustomerRelations(ctx, &customer); err != nil {
		return nil, fmt.Errorf("failed to load customer relations for %s: %w", email, err)
	}

	return &customer, nil
}

// getAddressesByCustomerID retrieves all addresses for a customer.
func (r *customerRepository) getAddressesByCustomerID(ctx context.Context, customerID uuid.UUID) ([]Address, error) {
	query := `SELECT * FROM customers.Address WHERE customer_id = $1`
	var addresses []Address
	if err := r.db.SelectContext(ctx, &addresses, query, customerID); err != nil {
		return nil, fmt.Errorf("%w: failed to fetch addresses for customer %s: %v", ErrDatabaseOperation, customerID, err)
	}
	return addresses, nil
}

// getCreditCardsByCustomerID retrieves all credit cards for a customer.
func (r *customerRepository) getCreditCardsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]CreditCard, error) {
	query := `SELECT * FROM customers.CreditCard WHERE customer_id = $1`
	var creditCards []CreditCard
	if err := r.db.SelectContext(ctx, &creditCards, query, customerID); err != nil {
		return nil, fmt.Errorf("%w: failed to fetch credit cards for customer %s: %v", ErrDatabaseOperation, customerID, err)
	}
	return creditCards, nil
}

// getStatusHistoryByCustomerID retrieves status history for a customer.
func (r *customerRepository) getStatusHistoryByCustomerID(ctx context.Context, customerID uuid.UUID) ([]CustomerStatus, error) {
	query := `SELECT * FROM customers.CustomerStatusHistory WHERE customer_id = $1`
	var statusHistory []CustomerStatus
	if err := r.db.SelectContext(ctx, &statusHistory, query, customerID); err != nil {
		return nil, fmt.Errorf("%w: failed to fetch status history for customer %s: %v", ErrDatabaseOperation, customerID, err)
	}
	return statusHistory, nil
}
```

**Note**: The original `GetCustomerByID()` and `GetCustomerByEmail()` are kept in query file because they're pure query operations. They call `LoadCustomerRelations()` which is defined in `repository_crud.go`. This creates a cross-file dependency which is fine since they're in the same package.

---

## File 4: repository_address.go

**Purpose**: Address CRUD operations and default address management

**Move from original repository.go**:

### Public Methods
- `AddAddress()` - line 778
- `UpdateAddress()` - line 839
- `DeleteAddress()` - line 896
- `UpdateDefaultShippingAddress()` - line 1096
- `UpdateDefaultBillingAddress()` - line 1148
- `ClearDefaultShippingAddress()` - line 1252
- `ClearDefaultBillingAddress()` - line 1299

**Remove**: Duplicate helpers that are now in `repository_crud.go` (`insertAddresses()`, `deleteAddresses()`)

**New file content** (~220 lines):

```go
// Package customer provides data access operations for customer entities.
//
// This file contains address-related operations including CRUD and default
// address management.
package customer

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"

	events "go-shopping-poc/internal/contracts/events"

	"go-shopping-poc/internal/platform/database"
)

// AddAddress adds a new address to a customer.
func (r *customerRepository) AddAddress(ctx context.Context, customerID string, addr *Address) (*Address, error) {
	log.Printf("[DEBUG] Repository: Adding address for customer %s", customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}

	addr.CustomerID = custUUID
	addr.AddressID = uuid.New()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

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

	evt := events.NewAddressAddedEvent(customerID, addr.AddressID.String(), map[string]string{"address_type": addr.AddressType})
	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	committed = true
	return addr, nil
}

// UpdateAddress updates an existing address.
func (r *customerRepository) UpdateAddress(ctx context.Context, addressID string, addr *Address) error {
	log.Printf("[DEBUG] Repository: Updating address %s", addressID)

	id, err := uuid.Parse(addressID)
	if err != nil {
		return err
	}
	addr.AddressID = id

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

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

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}

// DeleteAddress removes an address.
func (r *customerRepository) DeleteAddress(ctx context.Context, addressID string) error {
	log.Printf("[DEBUG] Repository: Deleting address with ID %s", addressID)

	id, err := uuid.Parse(addressID)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

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

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}

// UpdateDefaultShippingAddress sets the default shipping address for a customer.
func (r *customerRepository) UpdateDefaultShippingAddress(ctx context.Context, customerID, addressID string) error {
	log.Printf("[DEBUG] Repository: Setting default shipping address %s for customer %s", addressID, customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}

	addrUUID, err := uuid.Parse(addressID)
	if err != nil {
		return err
	}

	query := `UPDATE customers.Customer SET default_shipping_address_id = $1 WHERE customer_id = $2`

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

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

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}

// UpdateDefaultBillingAddress sets the default billing address for a customer.
func (r *customerRepository) UpdateDefaultBillingAddress(ctx context.Context, customerID, addressID string) error {
	log.Printf("[DEBUG] Repository: Setting default billing address %s for customer %s", addressID, customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}

	addrUUID, err := uuid.Parse(addressID)
	if err != nil {
		return err
	}

	query := `UPDATE customers.Customer SET default_billing_address_id = $1 WHERE customer_id = $2`

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

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

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}

// ClearDefaultShippingAddress clears the default shipping address for a customer.
func (r *customerRepository) ClearDefaultShippingAddress(ctx context.Context, customerID string) error {
	log.Printf("[DEBUG] Repository: Clearing default shipping address for customer %s", customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}

	query := `UPDATE customers.Customer SET default_shipping_address_id = NULL WHERE customer_id = $1`

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

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

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}

// ClearDefaultBillingAddress clears the default billing address for a customer.
func (r *customerRepository) ClearDefaultBillingAddress(ctx context.Context, customerID string) error {
	log.Printf("[DEBUG] Repository: Clearing default billing address for customer %s", customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}

	query := `UPDATE customers.Customer SET default_billing_address_id = NULL WHERE customer_id = $1`

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

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

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}
```

---

## File 5: repository_creditcard.go

**Purpose**: Credit card CRUD operations and default credit card management

**Move from original repository.go**:

### Public Methods
- `AddCreditCard()` - line 940
- `UpdateCreditCard()` - line 999
- `DeleteCreditCard()` - line 1052
- `UpdateDefaultCreditCard()` - line 1200
- `ClearDefaultCreditCard()` - line 1346

**Remove**: Duplicate helpers (`insertCreditCards()`, `deleteCreditCards()`)

**New file content** (~220 lines):

```go
// Package customer provides data access operations for customer entities.
//
// This file contains credit card-related operations including CRUD and default
// credit card management.
package customer

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"

	events "go-shopping-poc/internal/contracts/events"

	"go-shopping-poc/internal/platform/database"
)

// AddCreditCard adds a new credit card to a customer.
func (r *customerRepository) AddCreditCard(ctx context.Context, customerID string, card *CreditCard) (*CreditCard, error) {
	log.Printf("[DEBUG] Repository: Adding credit card for customer %s", customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}

	card.CustomerID = custUUID
	card.CardID = uuid.New()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

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
	committed = true
	return card, nil
}

// UpdateCreditCard updates an existing credit card.
func (r *customerRepository) UpdateCreditCard(ctx context.Context, cardID string, card *CreditCard) error {
	log.Printf("[DEBUG] Repository: Updating credit card %s", cardID)

	id, err := uuid.Parse(cardID)
	if err != nil {
		return err
	}
	card.CardID = id

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

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

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}

// DeleteCreditCard removes a credit card.
func (r *customerRepository) DeleteCreditCard(ctx context.Context, cardID string) error {
	log.Printf("[DEBUG] Repository: Deleting credit card with ID %s", cardID)

	id, err := uuid.Parse(cardID)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

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

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}

// UpdateDefaultCreditCard sets the default credit card for a customer.
func (r *customerRepository) UpdateDefaultCreditCard(ctx context.Context, customerID, cardID string) error {
	log.Printf("[DEBUG] Repository: Setting default credit card %s for customer %s", cardID, customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}

	cardUUID, err := uuid.Parse(cardID)
	if err != nil {
		return err
	}

	query := `UPDATE customers.Customer SET default_credit_card_id = $1 WHERE customer_id = $2`

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

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

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}

// ClearDefaultCreditCard clears the default credit card for a customer.
func (r *customerRepository) ClearDefaultCreditCard(ctx context.Context, customerID string) error {
	log.Printf("[DEBUG] Repository: Clearing default credit card for customer %s", customerID)

	custUUID, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}

	query := `UPDATE customers.Customer SET default_credit_card_id = NULL WHERE customer_id = $1`

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

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

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	return nil
}
```

---

## File 6: repository_util.go

**Purpose**: Utility functions and helper methods

**Move from original repository.go**:

### Public Methods
- (none - all are private)

### Private Functions
- `insertCustomerRecordInTransaction()` - line 181 (already in CRUD, remove)
- `PrepareCustomerDefaults()` - line 191 (already in CRUD as `prepareCustomerDefaults()`)
- `insertStatusHistory()` - line 585 (keep as public `insertStatusHistory()`)
- `publishCustomerUpdateEvent()` - line 606 (keep as public `publishCustomerUpdateEvent()`)

**New file content** (~50 lines):

```go
// Package customer provides data access operations for customer entities.
//
// This file contains utility functions for customer repository operations,
// including helper methods for status history and event publishing.
package customer

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"go-shopping-poc/internal/platform/database"
)

// insertStatusHistory inserts new status history records.
func insertStatusHistory(ctx context.Context, tx database.Tx, history []CustomerStatus, customerID uuid.UUID) error {
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

// publishCustomerUpdateEvent publishes the customer update event.
func publishCustomerUpdateEvent(ctx context.Context, tx database.Tx, customer *Customer) error {
	customerEvent := events.NewCustomerUpdatedEvent(customer.CustomerID, nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, customerEvent); err != nil {
		return fmt.Errorf("failed to publish customer update event: %w", err)
	}
	return nil
}
```

**Note**: Wait - `publishCustomerUpdateEvent()` references `r.outboxWriter` which requires receiver `*customerRepository`. This should stay as a method on the repository, not a standalone function. Let me fix this.

**Corrected utility functions**:

```go
// insertStatusHistory inserts new status history records.
func (r *customerRepository) insertStatusHistory(ctx context.Context, tx database.Tx, history []CustomerStatus, customerID uuid.UUID) error {
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
```

And update `repository_crud.go` to call `r.insertStatusHistory()` instead of `insertStatusHistory()`.

---

## Implementation Checklist

### Phase 1: Setup
- [ ] Create empty `repository_crud.go` with package header and imports
- [ ] Create empty `repository_query.go` with package header and imports
- [ ] Create empty `repository_address.go` with package header and imports
- [ ] Create empty `repository_creditcard.go` with package header and imports
- [ ] Create empty `repository_util.go` with package header and imports
- [ ] Update `repository.go` to contain only interface/struct/constructor

### Phase 2: Move CRUD operations
- [ ] Move method implementations from `repository.go` to `repository_crud.go`
- [ ] Rename `PrepareCustomerDefaults()` to `prepareCustomerDefaults()` (private)
- [ ] Ensure all helpers are in correct files

### Phase 3: Move query operations
- [ ] Move `GetCustomerByID()` to `repository_query.go`
- [ ] Move `GetCustomerByEmail()` to `repository_query.go`
- [ ] Move helper methods: `getAddressesByCustomerID()`, `getCreditCardsByCustomerID()`, `getStatusHistoryByCustomerID()`

### Phase 4: Move address operations
- [ ] Move `AddAddress()`, `UpdateAddress()`, `DeleteAddress()`
- [ ] Move default address management methods
- [ ] Ensure no duplicate helpers from CRUD file

### Phase 5: Move credit card operations
- [ ] Move `AddCreditCard()`, `UpdateCreditCard()`, `DeleteCreditCard()`
- [ ] Move default credit card management methods
- [ ] Ensure no duplicate helpers from CRUD file

### Phase 6: Move utilities
- [ ] Move `insertStatusHistory()` to `repository_util.go`
- [ ] Make it a receiver method: `func (r *customerRepository) insertStatusHistory()`
- [ ] Update all calls to use `r.insertStatusHistory()`

### Phase 7: Final cleanup
- [ ] Delete all method implementations from `repository.go`
- [ ] Verify all imports are correct
- [ ] Run `go build` to verify no compilation errors
- [ ] Run `go test` to verify no functionality broken
- [ ] Format code with `gofmt`
- [ ] Run linting if configured

### Phase 8: Verification
- [ ] TestInsertCustomer
- [ ] TestGetCustomerByID
- [ ] TestGetCustomerByEmail
- [ ] TestUpdateCustomer
- [ ] TestPatchCustomer
- [ ] TestAddAddress
- [ ] TestUpdateAddress
- [ ] TestDeleteAddress
- [ ] TestAddCreditCard
- [ ] TestUpdateCreditCard
- [ ] TestDeleteCreditCard
- [ ] TestDefaultAddress/Card methods

---

## Notes

1. **Cross-file dependencies are OK**: Methods in `repository_query.go` call methods in `repository_crud.go` (e.g., `GetCustomerByID()` calls `LoadCustomerRelations()`). This is fine since they're in the same package.

2. **Error handling**: All error wrappers use the pattern `fmt.Errorf("%w: message: %v", ErrType, err)` for proper error wrapping.

3. **Transaction patterns**: All database operations use the same transaction pattern:
   - BeginTx
   - Deferred rollback if not committed
   - NamedExecContext for inserts/updates
   - ExecContext for deletes
   - Commit with committed flag

4. **Event publishing**: All operations publish outbox events within the same transaction using `r.outboxWriter.WriteEvent(ctx, tx, evt)`.

5. **UUID handling**: Use `github.com/google/uuid` package for UUID operations.

6. **No public helper methods**: Helper methods that are implementation details should be private (lowercase).

7. **consistency with product repo**: This follows the same pattern as `internal/service/product/repository_*.go` files.
