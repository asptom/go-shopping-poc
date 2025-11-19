package customer

import (
	"context"
	"fmt"
	entity "go-shopping-poc/internal/entity/customer"

	"github.com/google/uuid"
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

func (s *CustomerService) GetCustomerByID(ctx context.Context, customerID string) (*entity.Customer, error) {
	return s.repo.GetCustomerByID(ctx, customerID)
}

func (s *CustomerService) CreateCustomer(ctx context.Context, customer *entity.Customer) error {
	return s.repo.InsertCustomer(ctx, customer)
}

func (s *CustomerService) UpdateCustomer(ctx context.Context, customer *entity.Customer) error {
	return s.repo.UpdateCustomer(ctx, customer)
}

func (s *CustomerService) PatchCustomer(ctx context.Context, customerID string, patchData map[string]interface{}) error {
	// Get existing customer first
	existing, err := s.repo.GetCustomerByID(ctx, customerID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("customer not found: %s", customerID)
	}

	// Apply patch data to existing customer
	updated := *existing // Copy existing customer

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

	return s.repo.UpdateCustomer(ctx, &updated)
}

// Helper function to safely extract string from interface
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
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

func (s *CustomerService) SetDefaultShippingAddress(ctx context.Context, customerID, addressID string) error {
	return s.repo.UpdateDefaultShippingAddress(ctx, customerID, addressID)
}

func (s *CustomerService) SetDefaultBillingAddress(ctx context.Context, customerID, addressID string) error {
	return s.repo.UpdateDefaultBillingAddress(ctx, customerID, addressID)
}

func (s *CustomerService) SetDefaultCreditCard(ctx context.Context, customerID, cardID string) error {
	return s.repo.UpdateDefaultCreditCard(ctx, customerID, cardID)
}

func (s *CustomerService) ClearDefaultShippingAddress(ctx context.Context, customerID string) error {
	return s.repo.ClearDefaultShippingAddress(ctx, customerID)
}

func (s *CustomerService) ClearDefaultBillingAddress(ctx context.Context, customerID string) error {
	return s.repo.ClearDefaultBillingAddress(ctx, customerID)
}

func (s *CustomerService) ClearDefaultCreditCard(ctx context.Context, customerID string) error {
	return s.repo.ClearDefaultCreditCard(ctx, customerID)
}
