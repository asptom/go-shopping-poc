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

type CustomerRepository interface {
	InsertCustomer(ctx context.Context, customer *entity.Customer) error
	GetCustomerByEmail(ctx context.Context, email string) (*entity.Customer, error)
	UpdateCustomer(ctx context.Context, customer *entity.Customer) error
	AddAddress(ctx context.Context, customerID string, addr *entity.Address) (*entity.Address, error)
	UpdateAddress(ctx context.Context, addressID string, addr *entity.Address) error
	DeleteAddress(ctx context.Context, addressID string) error
	AddCreditCard(ctx context.Context, customerID string, card *entity.CreditCard) (*entity.CreditCard, error)
	UpdateCreditCard(ctx context.Context, cardID string, card *entity.CreditCard) error
	DeleteCreditCard(ctx context.Context, cardID string) error
	//DeleteCustomer(ctx context.Context, id uuid.UUID) error
}
type customerRepository struct {
	db           *sqlx.DB
	outboxWriter outbox.Writer // keep existing field type
}

func NewCustomerRepository(db *sqlx.DB, outbox outbox.Writer) *customerRepository {
	return &customerRepository{db: db, outboxWriter: outbox}
}

func (r *customerRepository) InsertCustomer(ctx context.Context, customer *entity.Customer) error {

	logging.SetLevel("DEBUG")
	logging.Info("Inserting new customer...")

	newID := uuid.New()
	customer.CustomerID = newID.String() // Set the new UUID as string in customer
	query := `INSERT INTO customers.Customer (customer_id, user_name, email, first_name, last_name, phone) VALUES (:customer_id, :user_name, :email, :first_name, :last_name, :phone)`
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.NamedExecContext(ctx, query, customer); err != nil {
		return err
	}
	// Insert addresses, credit cards, and statuses if they exist
	address_query := `INSERT INTO customers.Address (customer_id, address_type, first_name, last_name, address_1, address_2, city, state, zip, is_default) VALUES (:customer_id, :address_type, :first_name, :last_name, :address_1, :address_2, :city, :state, :zip, :is_default)`
	for _, address := range customer.Addresses {
		address.CustomerID = newID
		if _, err := tx.NamedExecContext(ctx, address_query, address); err != nil {
			return err
		}
	}
	credit_card_query := `INSERT INTO customers.CreditCard (customer_id, card_type, card_number, card_holder_name, card_expires, card_cvv, is_default) VALUES (:customer_id, :card_type, :card_number, :card_holder_name, :card_expires, :card_cvv, :is_default)`
	for _, card := range customer.CreditCards {
		card.CustomerID = newID
		if _, err := tx.NamedExecContext(ctx, credit_card_query, card); err != nil {
			return err
		}
	}
	status_query := `INSERT INTO customers.CustomerStatus (customer_id, customer_status, status_date_time) VALUES (:customer_id, :customer_status, :status_date_time)`
	for _, status := range customer.Statuses {
		status.CustomerID = newID
		status.StatusDateTime = time.Now() // Set current time for status
		if _, err := tx.NamedExecContext(ctx, status_query, status); err != nil {
			return err
		}
	}

	customerEvent := events.NewCustomerCreatedEvent(*customer)

	if err := r.outboxWriter.WriteEvent(ctx, *tx, customerEvent); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *customerRepository) GetCustomerByEmail(ctx context.Context, email string) (*entity.Customer, error) {
	logging.Info("Fetching customer by email...")
	query := `select * from customers.customer where customers.customer.email = $1`
	var customer entity.Customer
	if err := r.db.GetContext(ctx, &customer, query, email); err != nil {
		if err == sql.ErrNoRows {
			// no customer found
			return nil, nil
		}
		logging.Error("Error fetching customer by email: %v", err)
		logging.Error("email: %v", email)
		return nil, err
	}

	id, err := uuid.Parse(customer.CustomerID)
	if err != nil {
		return nil, err
	}

	// Fetch related addresses, credit cards, and statuses
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

	statuses, err := r.getStatusesByCustomerID(ctx, id)
	if err != nil {
		return nil, err
	}
	customer.Statuses = statuses

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

func (r *customerRepository) getStatusesByCustomerID(ctx context.Context, customerID uuid.UUID) ([]entity.CustomerStatus, error) {
	query := `SELECT * FROM customers.CustomerStatus WHERE customer_id = $1`
	var statuses []entity.CustomerStatus
	if err := r.db.SelectContext(ctx, &statuses, query, customerID); err != nil {
		return nil, err
	}
	return statuses, nil
}

func (r *customerRepository) UpdateCustomer(ctx context.Context, customer *entity.Customer) error {
	logging.Info("Updating customer...")

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update main customer record
	query := `
        UPDATE customers.Customer 
        SET user_name = :user_name,
            email = :email,
            first_name = :first_name,
            last_name = :last_name,
            phone = :phone
        WHERE customer_id = :customer_id`

	result, err := tx.NamedExecContext(ctx, query, customer)
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

	// Delete existing related records to replace with new ones
	id, err := uuid.Parse(customer.CustomerID)
	if err != nil {
		return err
	}

	deleteQueries := []string{
		`DELETE FROM customers.Address WHERE customer_id = $1`,
		`DELETE FROM customers.CreditCard WHERE customer_id = $1`,
		`DELETE FROM customers.CustomerStatus WHERE customer_id = $1`,
	}

	for _, query := range deleteQueries {
		if _, err := tx.ExecContext(ctx, query, id); err != nil {
			return err
		}
	}

	// Insert new addresses
	if len(customer.Addresses) > 0 {
		addressQuery := `
            INSERT INTO customers.Address (
                customer_id, address_type, first_name, last_name,
                address_1, address_2, city, state, zip, is_default
            ) VALUES (
                :customer_id, :address_type, :first_name, :last_name,
                :address_1, :address_2, :city, :state, :zip, :is_default
            )`
		for _, addr := range customer.Addresses {
			addr.CustomerID = id
			if _, err := tx.NamedExecContext(ctx, addressQuery, addr); err != nil {
				return err
			}
		}
	}

	// Insert new credit cards
	if len(customer.CreditCards) > 0 {
		cardQuery := `
            INSERT INTO customers.CreditCard (
                customer_id, card_type, card_number, card_holder_name,
                card_expires, card_cvv, is_default
            ) VALUES (
                :customer_id, :card_type, :card_number, :card_holder_name,
                :card_expires, :card_cvv, :is_default
            )`
		for _, card := range customer.CreditCards {
			card.CustomerID = id
			if _, err := tx.NamedExecContext(ctx, cardQuery, card); err != nil {
				return err
			}
		}
	}

	// Insert new statuses
	if len(customer.Statuses) > 0 {
		statusQuery := `
            INSERT INTO customers.CustomerStatus (
                customer_id, customer_status, status_date_time
            ) VALUES (
                :customer_id, :customer_status, :status_date_time
            )`
		for _, status := range customer.Statuses {
			status.CustomerID = id
			status.StatusDateTime = time.Now() // Set current time for new status
			if _, err := tx.NamedExecContext(ctx, statusQuery, status); err != nil {
				return err
			}
		}
	}

	// Create and write CustomerUpdated event
	customerEvent := events.NewCustomerUpdatedEvent(*customer)
	if err := r.outboxWriter.WriteEvent(ctx, *tx, customerEvent); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *customerRepository) AddAddress(ctx context.Context, customerID string, addr *entity.Address) (*entity.Address, error) {
	logging.Info("Adding address for customer %s", customerID)
	id, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}

	// ensure IDs are set
	addr.CustomerID = id
	addr.AddressID = uuid.New()

	query := `
        INSERT INTO customers.Address (
            address_id, customer_id, address_type, first_name, last_name,
            address_1, address_2, city, state, zip, is_default
        ) VALUES (
            :address_id, :customer_id, :address_type, :first_name, :last_name,
            :address_1, :address_2, :city, :state, :zip, :is_default
        )`

	// use the struct for named parameter binding; include created_at via a small wrapper map
	params := map[string]interface{}{
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
		"is_default":   addr.IsDefault,
	}

	if _, err := r.db.NamedExecContext(ctx, query, params); err != nil {
		return nil, err
	}
	return addr, nil
}

func (r *customerRepository) UpdateAddress(ctx context.Context, addressID string, addr *entity.Address) error {
	logging.Info("Updating address for customer %s", addr.CustomerID)
	id, err := uuid.Parse(addressID)
	if err != nil {
		return err
	}

	query := `
        UPDATE customers.Address
        SET first_name = :first_name,
            last_name = :last_name,
            address_1 = :address_1,
            address_2 = :address_2,
            city = :city,
            state = :state,
            zip = :zip,
            is_default = :is_default
        WHERE address_id = :address_id`

	addr.AddressID = id
	result, err := r.db.NamedExecContext(ctx, query, addr)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("address not found for customer %s", addressID)
	}
	return nil
}

func (r *customerRepository) DeleteAddress(ctx context.Context, addressID string) error {
	logging.Info("Deleting address with ID %s", addressID)
	id, err := uuid.Parse(addressID)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `DELETE FROM customers.Address WHERE address_id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("address not found for customer %s", addressID)
	}
	return nil
}

func (r *customerRepository) AddCreditCard(ctx context.Context, customerID string, card *entity.CreditCard) (*entity.CreditCard, error) {
	logging.Info("Adding credit card for customer %s", customerID)
	id, err := uuid.Parse(customerID)
	if err != nil {
		return nil, err
	}

	card.CustomerID = id
	card.CardID = uuid.New()

	query := `
        INSERT INTO customers.CreditCard (
            card_id, customer_id, card_number, card_type, card_holder_name,
            card_expires, card_cvv, is_default
        ) VALUES (
            :card_id, :customer_id, :card_number, :card_type, :card_holder_name,
            :card_expires, :card_cvv, :is_default
        )`

	params := map[string]interface{}{
		"card_id":          card.CardID,
		"customer_id":      card.CustomerID,
		"card_number":      card.CardNumber,
		"card_type":        card.CardType,
		"card_holder_name": card.CardHolderName,
		"card_expires":     card.CardExpires,
		"card_cvv":         card.CardCVV,
		"is_default":       card.IsDefault,
	}

	if _, err := r.db.NamedExecContext(ctx, query, params); err != nil {
		return nil, err
	}
	return card, nil
}

func (r *customerRepository) UpdateCreditCard(ctx context.Context, cardID string, card *entity.CreditCard) error {
	logging.Info("Updating credit card for customer %s", card.CustomerID)
	id, err := uuid.Parse(cardID)
	if err != nil {
		return err
	}

	query := `
        UPDATE customers.CreditCard
        SET card_type = :card_type,
            card_holder_name = :card_holder_name,
            card_expires = :card_expires,
            card_cvv = :card_cvv,
            is_default = :is_default
        WHERE card_id = :card_id`

	card.CardID = id
	result, err := r.db.NamedExecContext(ctx, query, card)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("credit card not found for customer %s", card.CustomerID)
	}
	return nil
}

func (r *customerRepository) DeleteCreditCard(ctx context.Context, cardID string) error {
	logging.Info("Deleting credit card with ID %s", cardID)
	id, err := uuid.Parse(cardID)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `DELETE FROM customers.CreditCard WHERE card_id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("credit card not found")
	}
	return nil
}
