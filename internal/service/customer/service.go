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

func (s *CustomerService) GetCustomerByEmail(ctx context.Context, email string) (*entity.Customer, error) {
	return s.repo.GetCustomerByEmail(ctx, email)
}

func (s *CustomerService) CreateCustomer(ctx context.Context, customer *entity.Customer) error {
	return s.repo.InsertCustomer(ctx, customer)
}

func (s *CustomerService) UpdateCustomer(ctx context.Context, customer *entity.Customer) error {
	return s.repo.UpdateCustomer(ctx, customer)
}

func (s *CustomerService) AddAddress(ctx context.Context, customerID string, addr *entity.Address) (*entity.Address, error) {
	return s.repo.AddAddress(ctx, customerID, addr)
}

func (s *CustomerService) UpdateAddress(ctx context.Context, addressID string, addr *entity.Address) error {
	return s.repo.UpdateAddress(ctx, addressID, addr)
}

func (s *CustomerService) DeleteAddress(ctx context.Context, addressID string) error {
	return s.repo.DeleteAddress(ctx, addressID)
}

func (s *CustomerService) AddCreditCard(ctx context.Context, customerID string, card *entity.CreditCard) (*entity.CreditCard, error) {
	return s.repo.AddCreditCard(ctx, customerID, card)
}

func (s *CustomerService) UpdateCreditCard(ctx context.Context, customerID string, card *entity.CreditCard) error {
	return s.repo.UpdateCreditCard(ctx, customerID, card)
}

func (s *CustomerService) DeleteCreditCard(ctx context.Context, cardID string) error {
	return s.repo.DeleteCreditCard(ctx, cardID)
}
