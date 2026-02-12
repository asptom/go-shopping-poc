package cart

import (
	"context"
	"errors"
	"fmt"
	"log"

	"net/http"

	"github.com/google/uuid"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/outbox"
	"go-shopping-poc/internal/platform/service"
)

type CartInfrastructure struct {
	Database        database.Database
	EventBus        bus.Bus
	OutboxWriter    *outbox.Writer
	OutboxPublisher *outbox.Publisher
	ProductClient   ProductClient
	CORSHandler     func(http.Handler) http.Handler
}

func NewCartInfrastructure(
	db database.Database,
	eventBus bus.Bus,
	outboxWriter *outbox.Writer,
	outboxPublisher *outbox.Publisher,
	productClient ProductClient,
	corsHandler func(http.Handler) http.Handler,
) *CartInfrastructure {
	return &CartInfrastructure{
		Database:        db,
		EventBus:        eventBus,
		OutboxWriter:    outboxWriter,
		OutboxPublisher: outboxPublisher,
		ProductClient:   productClient,
		CORSHandler:     corsHandler,
	}
}

// Service defines the interface for event reader business operations
// This extends the platform service interface with domain-specific methods
type Service interface {
	service.Service
}

// RegisterHandler adds a new event handler for any event type to the service
// This is a convenience wrapper around the platform service RegisterHandler
func RegisterHandler[T events.Event](s Service, factory events.EventFactory[T], handler bus.HandlerFunc[T]) error {
	return service.RegisterHandler(s, factory, handler)
}

type CartService struct {
	*service.EventServiceBase
	repo           CartRepository
	infrastructure *CartInfrastructure
	config         *Config
}

func NewCartService(infrastructure *CartInfrastructure, config *Config) *CartService {
	repo := NewCartRepository(infrastructure.Database, infrastructure.OutboxWriter)

	return &CartService{
		EventServiceBase: service.NewEventServiceBase("cart", infrastructure.EventBus),
		repo:             repo,
		infrastructure:   infrastructure,
		config:           config,
	}
}

func NewCartServiceWithRepo(repo CartRepository, infrastructure *CartInfrastructure, config *Config) *CartService {
	return &CartService{
		EventServiceBase: service.NewEventServiceBase("cart", infrastructure.EventBus),
		repo:             repo,
		infrastructure:   infrastructure,
		config:           config,
	}
}

func (s *CartService) CreateCart(ctx context.Context, customerID *string) (*Cart, error) {
	cart := &Cart{
		Currency:      "USD",
		CurrentStatus: "active",
	}

	if customerID != nil && *customerID != "" {
		id, err := uuid.Parse(*customerID)
		if err != nil {
			return nil, fmt.Errorf("invalid customer ID: %w", err)
		}
		cart.CustomerID = &id
	}

	if err := s.repo.CreateCart(ctx, cart); err != nil {
		return nil, fmt.Errorf("failed to create cart: %w", err)
	}

	return cart, nil
}

func (s *CartService) GetCart(ctx context.Context, cartID string) (*Cart, error) {
	cart, err := s.repo.GetCartByID(ctx, cartID)
	if err != nil {
		if errors.Is(err, ErrCartNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}
	return cart, nil
}

func (s *CartService) DeleteCart(ctx context.Context, cartID string) error {
	if err := s.repo.DeleteCart(ctx, cartID); err != nil {
		if errors.Is(err, ErrCartNotFound) {
			return err
		}
		return fmt.Errorf("failed to delete cart: %w", err)
	}
	return nil
}

func (s *CartService) AddItem(ctx context.Context, cartID string, productID string, quantity int) (*CartItem, error) {

	log.Printf("[DEBUG] CartService: Adding item to cart %s: product_id=%s, quantity=%d", cartID, productID, quantity)

	if quantity <= 0 {
		return nil, errors.New("quantity must be positive")
	}

	cart, err := s.repo.GetCartByID(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	if cart.CurrentStatus != "active" {
		return nil, errors.New("cannot add items to non-active cart")
	}

	log.Printf("[DEBUG] CartService: Validating product %s for cart %s", productID, cartID)
	product, err := s.infrastructure.ProductClient.GetProduct(ctx, productID)
	if err != nil {
		log.Printf("[DEBUG] CartService: failed to validate product %s for cart %s: %v", productID, cartID, err)
		return nil, fmt.Errorf("failed to validate product: %w", err)
	}

	if !product.InStock {
		log.Printf("[DEBUG] CartService: product %s is out of stock for cart %s", productID, cartID)
		return nil, errors.New("product is out of stock")
	}

	item := &CartItem{
		ProductID:   productID,
		ProductName: product.Name,
		UnitPrice:   product.FinalPrice,
		Quantity:    quantity,
	}
	item.CalculateLineTotal()

	log.Printf("[DEBUG] CartService: Adding item to repository for cart %s: %+v", cartID, item)
	if err := s.repo.AddItem(ctx, cartID, item); err != nil {
		log.Printf("[DEBUG] CartService: failed to add item to repository for cart %s: %v", cartID, err)
		return nil, fmt.Errorf("failed to add item: %w", err)
	}

	cart.Items = append(cart.Items, *item)
	cart.CalculateTotals()

	log.Printf("[DEBUG] CartService: Updating cart totals for cart %s after adding item", cartID)
	if err := s.repo.UpdateCart(ctx, cart); err != nil {
		log.Printf("[DEBUG] CartService: failed to update cart totals for cart %s after adding item: %v", cartID, err)
		return nil, fmt.Errorf("failed to update cart totals: %w", err)
	}

	return item, nil
}

func (s *CartService) UpdateItemQuantity(ctx context.Context, cartID string, lineNumber string, quantity int) error {
	if quantity <= 0 {
		return errors.New("quantity must be positive")
	}

	cart, err := s.repo.GetCartByID(ctx, cartID)
	if err != nil {
		return fmt.Errorf("failed to get cart: %w", err)
	}

	if cart.CurrentStatus != "active" {
		return errors.New("cannot modify items in non-active cart")
	}

	if err := s.repo.UpdateItemQuantity(ctx, cartID, lineNumber, quantity); err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}

	for i := range cart.Items {
		if cart.Items[i].LineNumber == lineNumber {
			cart.Items[i].Quantity = quantity
			break
		}
	}
	cart.CalculateTotals()

	if err := s.repo.UpdateCart(ctx, cart); err != nil {
		return fmt.Errorf("failed to update cart totals: %w", err)
	}

	return nil
}

func (s *CartService) RemoveItem(ctx context.Context, cartID string, lineNumber string) error {
	cart, err := s.repo.GetCartByID(ctx, cartID)
	if err != nil {
		return fmt.Errorf("failed to get cart: %w", err)
	}

	if cart.CurrentStatus != "active" {
		return errors.New("cannot remove items from non-active cart")
	}

	if err := s.repo.RemoveItem(ctx, cartID, lineNumber); err != nil {
		return fmt.Errorf("failed to remove item: %w", err)
	}

	var newItems []CartItem
	for _, item := range cart.Items {
		if item.LineNumber != lineNumber {
			newItems = append(newItems, item)
		}
	}
	cart.Items = newItems
	cart.CalculateTotals()

	if err := s.repo.UpdateCart(ctx, cart); err != nil {
		return fmt.Errorf("failed to update cart totals: %w", err)
	}

	return nil
}

func (s *CartService) SetContact(ctx context.Context, cartID string, contact *Contact) error {
	log.Printf("[DEBUG] CartService: Setting contact for cart %s: %+v", cartID, contact)
	if err := contact.Validate(); err != nil {
		log.Printf("[DEBUG] CartService: invalid contact for cart %s: %v", cartID, err)
		return fmt.Errorf("invalid contact: %w", err)
	}

	cart, err := s.repo.GetCartByID(ctx, cartID)
	if err != nil {
		log.Printf("[DEBUG] CartService: failed to get cart %s for setting contact: %v", cartID, err)
		return fmt.Errorf("failed to get cart: %w", err)
	}

	if cart.CurrentStatus != "active" {
		log.Printf("[DEBUG] CartService: cannot set contact for non-active cart %s", cartID)
		return errors.New("cannot modify contact for non-active cart")
	}

	if err := s.repo.SetContact(ctx, cartID, contact); err != nil {
		log.Printf("[DEBUG] CartService: failed to set contact for cart %s: %v", cartID, err)
		return fmt.Errorf("failed to set contact: %w", err)
	}

	return nil
}

func (s *CartService) AddAddress(ctx context.Context, cartID string, address *Address) error {
	if err := address.Validate(); err != nil {
		return fmt.Errorf("invalid address: %w", err)
	}

	cart, err := s.repo.GetCartByID(ctx, cartID)
	if err != nil {
		return fmt.Errorf("failed to get cart: %w", err)
	}

	if cart.CurrentStatus != "active" {
		return errors.New("cannot add address to non-active cart")
	}

	if err := s.repo.AddAddress(ctx, cartID, address); err != nil {
		return fmt.Errorf("failed to add address: %w", err)
	}

	return nil
}

func (s *CartService) SetCreditCard(ctx context.Context, cartID string, card *CreditCard) error {
	log.Printf("[DEBUG] CartService: Setting credit card for cart %s: %+v", cartID, card)
	if err := card.Validate(); err != nil {
		return fmt.Errorf("invalid credit card: %w", err)
	}

	cart, err := s.repo.GetCartByID(ctx, cartID)
	if err != nil {
		log.Printf("[DEBUG] CartService: failed to get cart %s for setting credit card: %v", cartID, err)
		return fmt.Errorf("failed to get cart: %w", err)
	}

	if cart.CurrentStatus != "active" {
		log.Printf("[DEBUG] CartService: cannot set credit card for non-active cart %s", cartID)
		return errors.New("cannot modify payment for non-active cart")
	}

	if err := s.repo.SetCreditCard(ctx, cartID, card); err != nil {

		return fmt.Errorf("failed to set credit card: %w", err)
	}

	return nil
}

func (s *CartService) Checkout(ctx context.Context, cartID string) (*Cart, error) {
	cart, err := s.repo.GetCartByID(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart: %w", err)
	}

	cart.CalculateTotals()

	if err := s.repo.UpdateCart(ctx, cart); err != nil {
		return nil, fmt.Errorf("failed to update cart totals: %w", err)
	}

	checkedOutCart, err := s.repo.CheckoutCart(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("checkout failed: %w", err)
	}

	return checkedOutCart, nil
}

func (s *CartService) CalculateTax(ctx context.Context, netPrice float64) (float64, error) {
	return netPrice * 0.03, nil
}

func (s *CartService) CalculateShipping(ctx context.Context, cart *Cart) (float64, error) {
	return 0.0, nil
}
