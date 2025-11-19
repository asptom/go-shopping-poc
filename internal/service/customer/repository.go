package customer

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	entity "go-shopping-poc/internal/entity/customer"
	events "go-shopping-poc/internal/event/customer"
	"go-shopping-poc/pkg/logging"
	outbox "go-shopping-poc/pkg/outbox"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// CustomerRepository defines data access operations for customer entities.
//
// NOTE: Foreign Key Cascade Behavior:
// - Deleting a customer will automatically delete all associated addresses and credit cards (ON DELETE CASCADE)
// - Deleting an address or credit card will set the customer's default_*_id fields to NULL (ON DELETE SET NULL)
// - These behaviors are enforced at the database level per the updated schema constraints
type CustomerRepository interface {
	InsertCustomer(ctx context.Context, customer *entity.Customer) error
	GetCustomerByEmail(ctx context.Context, email string) (*entity.Customer, error)
	GetCustomerByID(ctx context.Context, customerID string) (*entity.Customer, error)
	UpdateCustomer(ctx context.Context, customer *entity.Customer) error
	PatchCustomer(ctx context.Context, customerID string, patchData map[string]interface{}) error

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

type customerRepository struct {
	db           *sqlx.DB
	outboxWriter outbox.Writer // existing outbox writer
}

func NewCustomerRepository(db *sqlx.DB, outbox outbox.Writer) *customerRepository {
	return &customerRepository{db: db, outboxWriter: outbox}
}

func (r *customerRepository) InsertCustomer(ctx context.Context, customer *entity.Customer) error {
	logging.Debug("Repository: Inserting new customer...")

	newID := uuid.New()
	customer.CustomerID = newID.String()

	// Set default values if not provided
	if customer.CustomerSince.IsZero() {
		customer.CustomerSince = time.Now()
	}
	if customer.CustomerStatus == "" {
		customer.CustomerStatus = "active"
	}
	if customer.StatusDateTime.IsZero() {
		customer.StatusDateTime = time.Now()
	}

	// Insert customer first without default IDs to avoid foreign key constraint
	customerQuery := `INSERT INTO customers.Customer (customer_id, user_name, email, first_name, last_name, phone, customer_since, customer_status, status_date_time) VALUES (:customer_id, :user_name, :email, :first_name, :last_name, :phone, :customer_since, :customer_status, :status_date_time)`

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.NamedExecContext(ctx, customerQuery, customer)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(customer.CustomerID)
	if err != nil {
		return err
	}

	// Commit the insert before reading related data to ensure visibility
	if err := tx.Commit(); err != nil {
		return err
	}

	addresses, err := r.getAddressesByCustomerID(ctx, id)
	if err != nil {
		return err
	}
	customer.Addresses = addresses

	creditCards, err := r.getCreditCardsByCustomerID(ctx, id)
	if err != nil {
		return err
	}
	customer.CreditCards = creditCards

	statusHistory, err := r.getStatusHistoryByCustomerID(ctx, id)
	if err != nil {
		return err
	}
	customer.StatusHistory = statusHistory

	return nil
}

func (r *customerRepository) GetCustomerByID(ctx context.Context, customerID string) (*entity.Customer, error) {
	logging.Debug("Repository: Fetching customer by ID...")

	id, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}

	query := `select * from customers.customer where customers.customer.customer_id = $1`
	var customer entity.Customer
	if err := r.db.GetContext(ctx, &customer, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		logging.Error("Error fetching customer by ID: %v", err)
		return nil, err
	}

	addresses, err := r.getAddressesByCustomerID(ctx, id)
	if err != nil {
		return nil, err
	}
	customer.Addresses = addresses

	creditCards, err := r.getCreditCardsByCustomerID(ctx, id)
	if err != nil {
		return nil, err
	}
	customer.CreditCards = creditCards

	statusHistory, err := r.getStatusHistoryByCustomerID(ctx, id)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	id, err := uuid.Parse(customer.CustomerID)
	if err != nil {
		return nil, err
	}

	addresses, err := r.getAddressesByCustomerID(ctx, id)
	if err != nil {
		return nil, err
	}
	customer.Addresses = addresses

	creditCards, err := r.getCreditCardsByCustomerID(ctx, id)
	if err != nil {
		return nil, err
	}
	customer.CreditCards = creditCards

	statusHistory, err := r.getStatusHistoryByCustomerID(ctx, id)
	if err != nil {
		return nil, err
	}
	customer.StatusHistory = statusHistory

	return &customer, nil
}

func (r *customerRepository) getAddressesByCustomerID(ctx context.Context, customerID uuid.UUID) ([]entity.Address, error) {
	query := `SELECT * FROM customers.Address WHERE customer_id = $1`
	var addresses []entity.Address
	if err := r.db.SelectContext(ctx, &addresses, query, customerID); err != nil {
		return nil, err
	}
	return addresses, nil
}

func (r *customerRepository) getCreditCardsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]entity.CreditCard, error) {
	query := `SELECT * FROM customers.CreditCard WHERE customer_id = $1`
	var creditCards []entity.CreditCard
	if err := r.db.SelectContext(ctx, &creditCards, query, customerID); err != nil {
		return nil, err
	}
	return creditCards, nil
}

func (r *customerRepository) getStatusHistoryByCustomerID(ctx context.Context, customerID uuid.UUID) ([]entity.CustomerStatus, error) {
	query := `SELECT * FROM customers.CustomerStatusHistory WHERE customer_id = $1`
	var statusHistory []entity.CustomerStatus
	if err := r.db.SelectContext(ctx, &statusHistory, query, customerID); err != nil {
		return nil, err
	}
	return statusHistory, nil
}

func (r *customerRepository) UpdateCustomer(ctx context.Context, customer *entity.Customer) error {
	logging.Debug("Repository: Updating customer (PUT - complete replace)...")

	// PUT requires complete customer record - validate required fields
	if customer.CustomerID == "" || customer.Username == "" || customer.Email == "" {
		return fmt.Errorf("PUT requires complete customer record with customer_id, username, and email")
	}

	id, err := uuid.Parse(customer.CustomerID)
	if err != nil {
		return err
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update basic customer info
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
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("customer not found: %s", customer.CustomerID)
	}

	// Delete existing addresses and credit cards
	if _, err := tx.ExecContext(ctx, `DELETE FROM customers.Address WHERE customer_id = $1`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM customers.CreditCard WHERE customer_id = $1`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM customers.CustomerStatusHistory WHERE customer_id = $1`, id); err != nil {
		return err
	}

	// Insert new addresses
	addressQuery := `INSERT INTO customers.Address (
		address_id, customer_id, address_type, first_name, last_name,
		address_1, address_2, city, state, zip
	) VALUES (
		:address_id, :customer_id, :address_type, :first_name, :last_name,
		:address_1, :address_2, :city, :state, :zip
	)`
	for i := range customer.Addresses {
		customer.Addresses[i].CustomerID = id
		customer.Addresses[i].AddressID = uuid.New()
		if _, err := tx.NamedExecContext(ctx, addressQuery, &customer.Addresses[i]); err != nil {
			return err
		}
	}

	// Insert new credit cards
	cardQuery := `INSERT INTO customers.CreditCard (
		card_id, customer_id, card_type, card_number, card_holder_name,
		card_expires, card_cvv
	) VALUES (
		:card_id, :customer_id, :card_type, :card_number, :card_holder_name,
		:card_expires, :card_cvv
	)`
	for i := range customer.CreditCards {
		customer.CreditCards[i].CustomerID = id
		customer.CreditCards[i].CardID = uuid.New()
		if _, err := tx.NamedExecContext(ctx, cardQuery, &customer.CreditCards[i]); err != nil {
			return err
		}
	}

	// Insert status history
	statusQuery := `INSERT INTO customers.CustomerStatusHistory (
		customer_id, old_status, new_status, changed_at
	) VALUES (
		:customer_id, :old_status, :new_status, :changed_at
	)`
	for _, status := range customer.StatusHistory {
		status.CustomerID = id
		if status.ChangedAt.IsZero() {
			status.ChangedAt = time.Now()
		}
		if _, err := tx.NamedExecContext(ctx, statusQuery, status); err != nil {
			return err
		}
	}

	// Publish event
	customerEvent := events.NewCustomerUpdatedEvent(customer.CustomerID, nil)
	if err := r.outboxWriter.WriteEvent(ctx, tx, customerEvent); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *customerRepository) PatchCustomer(ctx context.Context, customerID string, patchData map[string]interface{}) error {
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

	for field, value := range patchData {
		switch field {
		case "user_name":
			if str, ok := value.(string); ok {
				updated.Username = str
			}
		case "email":
			if str, ok := value.(string); ok {
				updated.Email = str
			}
		case "first_name":
			if str, ok := value.(string); ok {
				updated.FirstName = str
			}
		case "last_name":
			if str, ok := value.(string); ok {
				updated.LastName = str
			}
		case "phone":
			if str, ok := value.(string); ok {
				updated.Phone = str
			}
		case "customer_status":
			if str, ok := value.(string); ok {
				updated.CustomerStatus = str
			}
		case "default_shipping_address_id":
			if str, ok := value.(string); ok && str != "" {
				if uuid, err := uuid.Parse(str); err == nil {
					updated.DefaultShippingAddressID = &uuid
				}
			} else if str == "" {
				updated.DefaultShippingAddressID = nil
			}
		case "default_billing_address_id":
			if str, ok := value.(string); ok && str != "" {
				if uuid, err := uuid.Parse(str); err == nil {
					updated.DefaultBillingAddressID = &uuid
				}
			} else if str == "" {
				updated.DefaultBillingAddressID = nil
			}
		case "default_credit_card_id":
			if str, ok := value.(string); ok && str != "" {
				if uuid, err := uuid.Parse(str); err == nil {
					updated.DefaultCreditCardID = &uuid
				}
			} else if str == "" {
				updated.DefaultCreditCardID = nil
			}
		case "addresses":
			if addresses, ok := value.([]interface{}); ok {
				var addrList []entity.Address
				for _, addrInterface := range addresses {
					if addrMap, ok := addrInterface.(map[string]interface{}); ok {
						addr := entity.Address{
							AddressType: getString(addrMap, "address_type"),
							FirstName:   getString(addrMap, "first_name"),
							LastName:    getString(addrMap, "last_name"),
							Address1:    getString(addrMap, "address_1"),
							Address2:    getString(addrMap, "address_2"),
							City:        getString(addrMap, "city"),
							State:       getString(addrMap, "state"),
							Zip:         getString(addrMap, "zip"),
						}
						addrList = append(addrList, addr)
					}
				}
				updated.Addresses = addrList
			}
		case "credit_cards":
			if cards, ok := value.([]interface{}); ok {
				var cardList []entity.CreditCard
				for _, cardInterface := range cards {
					if cardMap, ok := cardInterface.(map[string]interface{}); ok {
						card := entity.CreditCard{
							CardType:       getString(cardMap, "card_type"),
							CardNumber:     getString(cardMap, "card_number"),
							CardHolderName: getString(cardMap, "card_holder_name"),
							CardExpires:    getString(cardMap, "card_expires"),
							CardCVV:        getString(cardMap, "card_cvv"),
						}
						cardList = append(cardList, card)
					}
				}
				updated.CreditCards = cardList
			}
		}
	}

	return r.UpdateCustomer(ctx, &updated)
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
