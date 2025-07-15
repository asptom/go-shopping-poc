package customer

import (
	"context"
	"go-shopping-poc/internal/entity"

	"github.com/google/uuid"
)

type CustomerService struct {
	repo CustomerRepository
}

func NewCustomerService(repo CustomerRepository) *CustomerService {
	return &CustomerService{repo: repo}
}

func (s *CustomerService) CreateCustomer(ctx context.Context, customer *entity.CustomerBase) error {
	return s.repo.InsertCustomer(ctx, customer)
}
func (s *CustomerService) GetCustomerByID(ctx context.Context, id string) (*entity.CustomerBase, error) {
	customerID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	return s.repo.GetCustomerByID(ctx, customerID)
}
