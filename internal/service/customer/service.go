package customer

import (
	"context"
	entity "go-shopping-poc/internal/entity/customer"
)

type CustomerService struct {
	repo CustomerRepository
}

func NewCustomerService(repo CustomerRepository) *CustomerService {
	return &CustomerService{repo: repo}
}

func (s *CustomerService) CreateCustomer(ctx context.Context, customer *entity.Customer) error {
	return s.repo.InsertCustomer(ctx, customer)
}
func (s *CustomerService) GetCustomerByEmail(ctx context.Context, email string) (*entity.Customer, error) {
	return s.repo.GetCustomerByEmail(ctx, email)
}
