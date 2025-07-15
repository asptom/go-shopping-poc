package customer

import (
	"context"
	"go-shopping-poc/internal/entity"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type CustomerRepository interface {
	InsertCustomer(ctx context.Context, customer *entity.CustomerBase) error
	GetCustomerByID(ctx context.Context, id uuid.UUID) (*entity.CustomerBase, error)
	//UpdateCustomer(ctx context.Context, customer *entity.Customer) error
	//DeleteCustomer(ctx context.Context, id uuid.UUID) error
}
type customerRepository struct {
	db *sqlx.DB // Assume Database is an interface for database operations
}

func NewCustomerRepository(db *sqlx.DB) CustomerRepository {
	return &customerRepository{db: db}
}

func (r *customerRepository) InsertCustomer(ctx context.Context, customer *entity.CustomerBase) error {
	newID := uuid.New()
	customer.CustomerID = newID
	query := `INSERT INTO customers.Customer (customerId, username, email, firstName, lastName, phone) VALUES (:customerId, :username, :email, :firstName, :lastName, :phone)`
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.NamedExecContext(ctx, query, customer); err != nil {
		return err
	}
	// Insert addresses, credit cards, and statuses if they exist
	return tx.Commit()
}

func (r *customerRepository) GetCustomerByID(ctx context.Context, id uuid.UUID) (*entity.CustomerBase, error) {
	query := `SELECT * FROM customers.Customer WHERE customerId = $1`
	var customer entity.CustomerBase
	if err := r.db.GetContext(ctx, &customer, query, id); err != nil {
		return nil, err
	}

	// Fetch related addresses, credit cards, and statuses
	//addresses, err := r.getAddressesByCustomerID(ctx, id)
	//if err != nil {
	//	return nil, err
	//}
	//customer.Addresses = addresses

	//creditCards, err := r.getCreditCardsByCustomerID(ctx, id)
	//if err != nil {
	//	return nil, err
	//}
	//customer.CreditCards = creditCards

	//statuses, err := r.getStatusesByCustomerID(ctx, id)
	//if err != nil {
	//	return nil, err
	//}
	//customer.Statuses = statuses

	return &customer, nil
}
