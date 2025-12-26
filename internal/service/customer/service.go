// Package customer provides business logic for customer management operations.
//
// This package implements the service layer for customer domain operations including
// CRUD operations, validation, and business rule enforcement. It acts as an
// intermediary between HTTP handlers and the data repository layer.
package customer

import (
	"context"
	"fmt"
	"net/http"

	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/outbox"
	"go-shopping-poc/internal/platform/service"

	"github.com/google/uuid"
)

// CustomerInfrastructure defines the infrastructure components required by the customer service.
//
// This struct encapsulates all external dependencies that the customer service needs
// to function, including database connectivity, event publishing, outbox pattern,
// and HTTP middleware. It follows clean architecture principles by clearly defining
// the infrastructure boundaries that the service depends on.
type CustomerInfrastructure struct {
	// Database provides data persistence and transaction management for customer data
	Database database.Database

	// EventBus handles publishing customer domain events to the message broker
	EventBus bus.Bus

	// OutboxWriter writes customer events to the outbox table for reliable publishing
	OutboxWriter *outbox.Writer

	// OutboxPublisher publishes outbox events to Kafka with retry logic and batching
	OutboxPublisher *outbox.Publisher

	// CORSHandler provides HTTP CORS middleware for cross-origin requests
	CORSHandler func(http.Handler) http.Handler
}

// NewCustomerInfrastructure creates a new CustomerInfrastructure instance with the provided components.
//
// Parameters:
//   - db: Database connection for customer data operations
//   - eventBus: Event bus for publishing customer domain events
//   - outboxWriter: Writer for storing events in the outbox table
//   - outboxPublisher: Publisher for sending outbox events to message broker
//   - corsHandler: CORS middleware handler for HTTP requests
//
// Returns a configured CustomerInfrastructure ready for use by the customer service.
func NewCustomerInfrastructure(
	db database.Database,
	eventBus bus.Bus,
	outboxWriter *outbox.Writer,
	outboxPublisher *outbox.Publisher,
	corsHandler func(http.Handler) http.Handler,
) *CustomerInfrastructure {
	return &CustomerInfrastructure{
		Database:        db,
		EventBus:        eventBus,
		OutboxWriter:    outboxWriter,
		OutboxPublisher: outboxPublisher,
		CORSHandler:     corsHandler,
	}
}

// PatchCustomerRequest represents a typed request for patching customer data
type PatchCustomerRequest struct {
	UserName                 *string                  `json:"user_name,omitempty"`
	Email                    *string                  `json:"email,omitempty"`
	FirstName                *string                  `json:"first_name,omitempty"`
	LastName                 *string                  `json:"last_name,omitempty"`
	Phone                    *string                  `json:"phone,omitempty"`
	CustomerStatus           *string                  `json:"customer_status,omitempty"`
	DefaultShippingAddressID *string                  `json:"default_shipping_address_id,omitempty"`
	DefaultBillingAddressID  *string                  `json:"default_billing_address_id,omitempty"`
	DefaultCreditCardID      *string                  `json:"default_credit_card_id,omitempty"`
	Addresses                []PatchAddressRequest    `json:"addresses,omitempty"`
	CreditCards              []PatchCreditCardRequest `json:"credit_cards,omitempty"`
}

// PatchAddressRequest represents a typed request for patching address data
type PatchAddressRequest struct {
	AddressType string `json:"address_type"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Address1    string `json:"address_1"`
	Address2    string `json:"address_2"`
	City        string `json:"city"`
	State       string `json:"state"`
	Zip         string `json:"zip"`
}

// PatchCreditCardRequest represents a typed request for patching credit card data
type PatchCreditCardRequest struct {
	CardType       string `json:"card_type"`
	CardNumber     string `json:"card_number"`
	CardHolderName string `json:"card_holder_name"`
	CardExpires    string `json:"card_expires"`
	CardCVV        string `json:"card_cvv"`
}

// CustomerService orchestrates customer business operations.
//
// CustomerService acts as the service layer, coordinating between
// the HTTP handlers and the repository layer. It contains business
// logic, validation, and data transformation.
type CustomerService struct {
	*service.BaseService
	repo           CustomerRepository
	infrastructure *CustomerInfrastructure
	config         *Config // Store config for potential future use
}

// NewCustomerService creates a new customer service instance.
func NewCustomerService(infrastructure *CustomerInfrastructure, config *Config) *CustomerService {
	// Create repository from infrastructure components
	repo := NewCustomerRepository(infrastructure.Database, infrastructure.OutboxWriter)

	return &CustomerService{
		BaseService:    service.NewBaseService("customer"),
		repo:           repo,
		infrastructure: infrastructure,
		config:         config,
	}
}

// NewCustomerServiceWithRepo creates a new customer service instance with a custom repository.
// This is primarily used for testing to inject mock repositories.
func NewCustomerServiceWithRepo(repo CustomerRepository, infrastructure *CustomerInfrastructure, config *Config) *CustomerService {
	return &CustomerService{
		BaseService:    service.NewBaseService("customer"),
		repo:           repo,
		infrastructure: infrastructure,
		config:         config,
	}
}

func (s *CustomerService) GetCustomerByEmail(ctx context.Context, email string) (*Customer, error) {
	return s.repo.GetCustomerByEmail(ctx, email)
}

func (s *CustomerService) GetCustomerByID(ctx context.Context, customerID string) (*Customer, error) {
	return s.repo.GetCustomerByID(ctx, customerID)
}

func (s *CustomerService) CreateCustomer(ctx context.Context, customer *Customer) error {
	// Validate the customer entity
	if err := customer.Validate(); err != nil {
		return fmt.Errorf("customer validation failed: %w", err)
	}

	// Validate addresses if provided
	for i, addr := range customer.Addresses {
		if err := addr.Validate(); err != nil {
			return fmt.Errorf("address %d validation failed: %w", i, err)
		}
	}

	// Validate credit cards if provided
	for i, card := range customer.CreditCards {
		if err := card.Validate(); err != nil {
			return fmt.Errorf("credit card %d validation failed: %w", i, err)
		}
	}

	return s.repo.InsertCustomer(ctx, customer)
}

func (s *CustomerService) UpdateCustomer(ctx context.Context, customer *Customer) error {
	return s.repo.UpdateCustomer(ctx, customer)
}

// PatchCustomer applies partial updates to an existing customer.
//
// This method supports PATCH operations, allowing clients to update
// specific fields of a customer without replacing the entire record.
// It validates the patch data and delegates to the repository for persistence.
func (s *CustomerService) PatchCustomer(ctx context.Context, customerID string, patchData *PatchCustomerRequest) error {
	// Validate patch data
	if err := s.ValidatePatchData(patchData); err != nil {
		return fmt.Errorf("invalid patch data: %w", err)
	}

	// Delegate to repository for selective updates
	return s.repo.PatchCustomer(ctx, customerID, patchData)
}

// ValidatePatchData validates the patch request data
func (s *CustomerService) ValidatePatchData(patchData *PatchCustomerRequest) error {
	if patchData == nil {
		return fmt.Errorf("patch data cannot be nil")
	}

	// Validate UUID fields if provided
	if patchData.DefaultShippingAddressID != nil && *patchData.DefaultShippingAddressID != "" {
		if _, err := uuid.Parse(*patchData.DefaultShippingAddressID); err != nil {
			return fmt.Errorf("invalid default_shipping_address_id: %w", err)
		}
	}
	if patchData.DefaultBillingAddressID != nil && *patchData.DefaultBillingAddressID != "" {
		if _, err := uuid.Parse(*patchData.DefaultBillingAddressID); err != nil {
			return fmt.Errorf("invalid default_billing_address_id: %w", err)
		}
	}
	if patchData.DefaultCreditCardID != nil && *patchData.DefaultCreditCardID != "" {
		if _, err := uuid.Parse(*patchData.DefaultCreditCardID); err != nil {
			return fmt.Errorf("invalid default_credit_card_id: %w", err)
		}
	}

	return nil
}

// ApplyFieldUpdates applies basic field updates to the customer
func (s *CustomerService) ApplyFieldUpdates(customer *Customer, patchData *PatchCustomerRequest) error {
	if patchData.UserName != nil {
		customer.Username = *patchData.UserName
	}
	if patchData.Email != nil {
		customer.Email = *patchData.Email
	}
	if patchData.FirstName != nil {
		customer.FirstName = *patchData.FirstName
	}
	if patchData.LastName != nil {
		customer.LastName = *patchData.LastName
	}
	if patchData.Phone != nil {
		customer.Phone = *patchData.Phone
	}
	if patchData.CustomerStatus != nil {
		customer.CustomerStatus = *patchData.CustomerStatus
	}

	// Handle UUID pointer fields
	if patchData.DefaultShippingAddressID != nil {
		if *patchData.DefaultShippingAddressID == "" {
			customer.DefaultShippingAddressID = nil
		} else {
			uuid, err := uuid.Parse(*patchData.DefaultShippingAddressID)
			if err != nil {
				return fmt.Errorf("invalid default_shipping_address_id: %w", err)
			}
			customer.DefaultShippingAddressID = &uuid
		}
	}
	if patchData.DefaultBillingAddressID != nil {
		if *patchData.DefaultBillingAddressID == "" {
			customer.DefaultBillingAddressID = nil
		} else {
			uuid, err := uuid.Parse(*patchData.DefaultBillingAddressID)
			if err != nil {
				return fmt.Errorf("invalid default_billing_address_id: %w", err)
			}
			customer.DefaultBillingAddressID = &uuid
		}
	}
	if patchData.DefaultCreditCardID != nil {
		if *patchData.DefaultCreditCardID == "" {
			customer.DefaultCreditCardID = nil
		} else {
			uuid, err := uuid.Parse(*patchData.DefaultCreditCardID)
			if err != nil {
				return fmt.Errorf("invalid default_credit_card_id: %w", err)
			}
			customer.DefaultCreditCardID = &uuid
		}
	}
	return nil
}

// TransformAddressesFromPatch converts patch address requests to entity addresses
func (s *CustomerService) TransformAddressesFromPatch(patchAddresses []PatchAddressRequest) []Address {
	var addresses []Address
	for _, patchAddr := range patchAddresses {
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
		addresses = append(addresses, addr)
	}
	return addresses
}

// TransformCreditCardsFromPatch converts patch credit card requests to entity credit cards
func (s *CustomerService) TransformCreditCardsFromPatch(patchCards []PatchCreditCardRequest) []CreditCard {
	var cards []CreditCard
	for _, patchCard := range patchCards {
		card := CreditCard{
			CardType:       patchCard.CardType,
			CardNumber:     patchCard.CardNumber,
			CardHolderName: patchCard.CardHolderName,
			CardExpires:    patchCard.CardExpires,
			CardCVV:        patchCard.CardCVV,
		}
		cards = append(cards, card)
	}
	return cards
}

func (s *CustomerService) AddAddress(ctx context.Context, customerID string, addr *Address) (*Address, error) {
	return s.repo.AddAddress(ctx, customerID, addr)
}

func (s *CustomerService) UpdateAddress(ctx context.Context, addressID string, addr *Address) error {
	return s.repo.UpdateAddress(ctx, addressID, addr)
}

func (s *CustomerService) DeleteAddress(ctx context.Context, addressID string) error {
	return s.repo.DeleteAddress(ctx, addressID)
}

func (s *CustomerService) AddCreditCard(ctx context.Context, customerID string, card *CreditCard) (*CreditCard, error) {
	return s.repo.AddCreditCard(ctx, customerID, card)
}

func (s *CustomerService) UpdateCreditCard(ctx context.Context, customerID string, card *CreditCard) error {
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
