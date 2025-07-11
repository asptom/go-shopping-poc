package customer

import (
	"context"
	api "go-shopping-poc/internal/customermodel"
	"go-shopping-poc/resources/domain/customer"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store defines the repository interface using API types.
type Store interface {
	GetCustomer(ctx context.Context, id string) (*api.Customer, error)
	AddCustomer(ctx context.Context, c *api.Customer) (string, error)
	UpdateCustomer(ctx context.Context, c *api.Customer) error
}

type store struct {
	queries *customer.Queries
}

func NewStore(db *pgxpool.Pool) Store {
	return &store{queries: customer.New(db)}
}

func (s *store) GetCustomer(ctx context.Context, id string) (*api.Customer, error) {
	dbCustomer, err := s.queries.GetCustomer(ctx, id)
	if err != nil {
		return nil, err
	}
	dbAddresses, _ := s.queries.GetCustomerAddresses(ctx, id)
	dbCreditCards, _ := s.queries.GetCustomerCreditCards(ctx, id)
	dbStatuses, _ := s.queries.GetCustomerStatuses(ctx, id)

	apiCustomer := api.ConvertCustomerDBToAPI(dbCustomer, dbAddresses, dbCreditCards, dbStatuses)
	return &apiCustomer, nil
}

func (s *store) AddCustomer(ctx context.Context, c *api.Customer) (string, error) {
	dbCustomer := api.ConvertCustomerAPIToDB(*c)
	// Insert customer (adjust query name as needed)
	id, err := s.queries.AddCustomer(ctx, dbCustomer)
	if err != nil {
		return "", err
	}
	// Optionally insert addresses, credit cards, statuses here using ConvertAddressesAPIToDB, etc.
	return id.String(), nil
}

func (s *store) UpdateCustomer(ctx context.Context, c *api.Customer) error {
	dbCustomer := api.ConvertCustomerAPIToDB(*c)
	return s.queries.UpdateCustomer(ctx, dbCustomer)
}
