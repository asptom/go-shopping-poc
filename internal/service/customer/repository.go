package customer

import (
	"context"
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
	GetCustomerByID(ctx context.Context, id uuid.UUID) (*entity.Customer, error)
	//UpdateCustomer(ctx context.Context, customer *entity.Customer) error
	//DeleteCustomer(ctx context.Context, id uuid.UUID) error
}
type customerRepository struct {
	db           *sqlx.DB
	outboxWriter *outbox.Writer
}

func NewCustomerRepository(db *sqlx.DB, outboxWriter *outbox.Writer) CustomerRepository {
	return &customerRepository{db: db, outboxWriter: outboxWriter}
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

func (r *customerRepository) GetCustomerByID(ctx context.Context, id uuid.UUID) (*entity.Customer, error) {
	query := `SELECT * FROM customers.Customer WHERE customer_id = $1`
	var customer entity.Customer
	if err := r.db.GetContext(ctx, &customer, query, id); err != nil {
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
