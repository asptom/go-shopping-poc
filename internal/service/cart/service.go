package cart

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"net/http"

	"github.com/google/uuid"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/logging"
	"go-shopping-poc/internal/platform/outbox"
	"go-shopping-poc/internal/platform/service"
	"go-shopping-poc/internal/platform/sse"
)

type CartInfrastructure struct {
	Database        database.Database
	EventBus        bus.Bus
	OutboxWriter    *outbox.Writer
	OutboxPublisher *outbox.Publisher
	CORSHandler     func(http.Handler) http.Handler
	SSEProvider      *sse.Provider
}

func NewCartInfrastructure(
	db database.Database,
	eventBus bus.Bus,
	outboxWriter *outbox.Writer,
	outboxPublisher *outbox.Publisher,
	corsHandler func(http.Handler) http.Handler,
	sseProvider *sse.Provider,
) *CartInfrastructure {
	return &CartInfrastructure{
		Database:        db,
		EventBus:        eventBus,
		OutboxWriter:    outboxWriter,
		OutboxPublisher: outboxPublisher,
		CORSHandler:     corsHandler,
		SSEProvider:     sseProvider,
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
	logger       *slog.Logger
	repo         CartRepository
	infrastructure *CartInfrastructure
	config       *Config
	productCache *ProductCache
}

func NewCartService(logger *slog.Logger, infrastructure *CartInfrastructure, config *Config) *CartService {
	if logger == nil {
		logger = logging.FromContext(context.Background())
	}
	repo := NewCartRepository(infrastructure.Database, infrastructure.OutboxWriter)

	return &CartService{
		EventServiceBase: service.NewEventServiceBase("cart", infrastructure.EventBus, logger),
		logger:           logger.With("component", "cart_service"),
		repo:             repo,
		infrastructure:   infrastructure,
		config:           config,
		productCache:     NewProductCache(),
	}
}

func NewCartServiceWithRepo(logger *slog.Logger, repo CartRepository, infrastructure *CartInfrastructure, config *Config) *CartService {
	if logger == nil {
		logger = logging.FromContext(context.Background())
	}
	return &CartService{
		EventServiceBase: service.NewEventServiceBase("cart", infrastructure.EventBus, logger),
		logger:           logger.With("component", "cart_service"),
		repo:             repo,
		infrastructure:   infrastructure,
		config:           config,
		productCache:     NewProductCache(),
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
			s.logger.Warn("Invalid customer ID provided when creating cart", "customer_id", *customerID, "error", err.Error())
			return nil, fmt.Errorf("invalid customer ID: %w", err)
		}
		cart.CustomerID = &id
		s.logger.Debug("Creating cart with customer ID", "customer_id", *customerID)
	} else {
		s.logger.Debug("Creating cart without customer ID")
	}

	if err := s.repo.CreateCart(ctx, cart); err != nil {
		return nil, fmt.Errorf("failed to create cart: %w", err)
	}

	s.logger.Info("Created new cart", "cart_id", cart.CartID)
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

func (s *CartService) AddItem(ctx context.Context, cartID string, productID string, quantity int, imageURL string) (*CartItem, error) {
	s.logger.Debug("Adding item to cart",
		"cart_id", cartID,
		"product_id", productID,
		"quantity", quantity,
		"image_url", imageURL,
	)

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

	// Check if product already exists in cart (prevent duplicates during validation)
	existingItem, err := s.repo.GetItemByProductID(ctx, cartID, productID)
	if err == nil && existingItem != nil {
		if existingItem.IsPendingValidation() {
			return nil, errors.New("product is already being added to cart, please wait for validation")
		}
		if existingItem.IsConfirmed() {
			return nil, errors.New("product already exists in cart, use update quantity instead")
		}
		// If backorder, allow adding again (will create new validation attempt)
	}

	// Fast path: check product cache before emitting event.
	// If the product is in the cache, we can validate synchronously
	// without the event round-trip.
	cacheEntry, cacheHit := s.productCache.Get(productID)
	if cacheHit {
		if !cacheEntry.InStock {
			s.logger.Debug("Product out of stock (cache)", "product_id", productID)
			// Create a backorder item — same as the event-driven path's ProductUnavailable handling
			validationID := uuid.New().String()
			reason := "product_out_of_stock"
			item := &CartItem{
				ProductID:    productID,
				Quantity:     quantity,
				ImageURL:     imageURL,
				Status:       "backorder",
				ValidationID: &validationID,
				BackorderReason: &reason,
			}

			tx, err := s.infrastructure.Database.BeginTx(ctx, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to begin transaction: %w", err)
			}

			committed := false
			defer func() {
				if !committed {
					_ = tx.Rollback()
				}
			}()

			if err := s.repo.AddItemTx(ctx, tx, cartID, item); err != nil {
				return nil, fmt.Errorf("failed to add item: %w", err)
			}

			if err := tx.Commit(); err != nil {
				return nil, fmt.Errorf("failed to commit transaction: %w", err)
			}
			committed = true

			s.logger.Debug("Updating cart totals with backorder item", "cart_id", cartID)
			if err := s.repo.UpdateCart(ctx, cart); err != nil {
				s.logger.Warn("Failed to update cart totals", "cart_id", cartID, "error", err.Error())
			}

			if s.infrastructure.SSEProvider != nil {
				s.infrastructure.SSEProvider.GetHub().Publish(cartID, "cart.item.backorder", map[string]interface{}{
					"line_number":      item.LineNumber,
					"product_id":       productID,
					"status":           "backorder",
					"backorder_reason": reason,
				})
			}

			s.logger.Info("Added backorder item (cache miss - out of stock)",
				"cart_id", cartID,
				"product_id", productID,
				"quantity", quantity,
			)
			return item, nil
		}

		// Product is in stock — fast path: confirm immediately without event round-trip.
		validationID := uuid.New().String()
		item := &CartItem{
			ProductID:    productID,
			Quantity:     quantity,
			ImageURL:     imageURL,
			Status:       "confirmed",
			ValidationID: &validationID,
			ProductName:  cacheEntry.Name,
			UnitPrice:    cacheEntry.FinalPrice,
		}

		tx, err := s.infrastructure.Database.BeginTx(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}

		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback()
			}
		}()

		if err := s.repo.AddItemTx(ctx, tx, cartID, item); err != nil {
			return nil, fmt.Errorf("failed to add item: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}
		committed = true

		cart.Items = append(cart.Items, *item)
		cart.CalculateTotals()

		s.logger.Debug("Updating cart totals after confirmed item", "cart_id", cartID)
		if err := s.repo.UpdateCart(ctx, cart); err != nil {
			s.logger.Warn("Failed to update cart totals", "cart_id", cartID, "error", err.Error())
		}

		item.CalculateLineTotal()

		if s.infrastructure.SSEProvider != nil {
			s.infrastructure.SSEProvider.GetHub().Publish(cartID, "cart.item.validated", map[string]interface{}{
				"line_number":  item.LineNumber,
				"product_id":   productID,
				"product_name": cacheEntry.Name,
				"unit_price":   cacheEntry.FinalPrice,
				"quantity":     item.Quantity,
				"total_price":  item.TotalPrice,
				"status":       "validated",
			})
		}

		s.logger.Info("Added item to cart (fast path - cache hit)",
			"cart_id", cartID,
			"product_id", productID,
			"quantity", quantity,
		)
		return item, nil
	}

	// Slow path: product not in cache. Fall back to event-driven validation.
	// This is the existing behavior — emit CartItemAdded and wait for
	// ProductValidated/ProductUnavailable events from the product service.

	s.logger.Debug("Product not in cache, emitting validation event", "product_id", productID)

	validationID := uuid.New().String()
	item := &CartItem{
		ProductID:    productID,
		Quantity:     quantity,
		ImageURL:     imageURL,
		Status:       "pending_validation",
		ValidationID: &validationID,
		// LineNumber will be assigned by repository
		// ProductName and UnitPrice will be updated after validation
	}

	// Begin transaction to add item and write event
	tx, err := s.infrastructure.Database.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// Add item to cart within transaction
	if err := s.repo.AddItemTx(ctx, tx, cartID, item); err != nil {
		return nil, fmt.Errorf("failed to add item: %w", err)
	}

	// Emit CartItemAdded event to outbox (transactional)
	// This notifies other services (like product) that an item was added
	cartItemEvent := events.NewCartItemAddedEvent(cartID, item.LineNumber, productID, quantity, validationID)
	if err := s.infrastructure.OutboxWriter.WriteEvent(ctx, tx, cartItemEvent); err != nil {
		return nil, fmt.Errorf("failed to write cart item event: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true

	// Trigger immediate outbox processing for low latency
	if s.infrastructure.OutboxPublisher != nil {
		go func() {
			if err := s.infrastructure.OutboxPublisher.ProcessNow(); err != nil {
				s.logger.Warn("Failed to trigger immediate outbox processing",
					"error", err.Error(),
				)
			}
		}()
	}

	// Update cart totals with pending item (best effort, not transactional)
	cart.Items = append(cart.Items, *item)
	cart.CalculateTotals()

	s.logger.Debug("Updating cart totals after adding pending item", "cart_id", cartID)
	if err := s.repo.UpdateCart(ctx, cart); err != nil {
		s.logger.Warn("Failed to update cart totals",
			"cart_id", cartID,
			"error", err.Error(),
		)
	}

	s.logger.Debug("Added pending item to cart (cache miss, awaiting validation)",
		"cart_id", cartID,
		"product_id", productID,
		"quantity", quantity,
		"item_line_number", item.LineNumber,
	)
	s.logger.Info("Added item to cart, pending validation",
		"cart_id", cartID,
		"product_id", productID,
		"quantity", quantity,
		"item_line_number", item.LineNumber,
	)
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
	s.logger.Debug("Setting contact for cart",
		"cart_id", cartID,
		"contact", contact,
	)
	if err := contact.Validate(); err != nil {
		s.logger.Debug("Invalid contact for cart",
			"cart_id", cartID,
			"error", err.Error(),
		)
		return fmt.Errorf("invalid contact: %w", err)
	}

	cart, err := s.repo.GetCartByID(ctx, cartID)
	if err != nil {
		s.logger.Debug("Failed to get cart for setting contact",
			"cart_id", cartID,
			"error", err.Error(),
		)
		return fmt.Errorf("failed to get cart: %w", err)
	}

	if cart.CurrentStatus != "active" {
		s.logger.Debug("Cannot set contact for non-active cart",
			"cart_id", cartID,
		)
		return errors.New("cannot modify contact for non-active cart")
	}

	if err := s.repo.SetContact(ctx, cartID, contact); err != nil {
		s.logger.Debug("Failed to set contact for cart",
			"cart_id", cartID,
			"error", err.Error(),
		)
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
	s.logger.Debug("Setting credit card for cart",
		"cart_id", cartID,
		"card", card,
	)
	if err := card.Validate(); err != nil {
		return fmt.Errorf("invalid credit card: %w", err)
	}

	cart, err := s.repo.GetCartByID(ctx, cartID)
	if err != nil {
		s.logger.Debug("Failed to get cart for setting credit card",
			"cart_id", cartID,
			"error", err.Error(),
		)
		return fmt.Errorf("failed to get cart: %w", err)
	}

	if cart.CurrentStatus != "active" {
		s.logger.Debug("Cannot set credit card for non-active cart",
			"cart_id", cartID,
		)
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

	// Check for pending validation items - cannot checkout until all items are validated
	for _, item := range cart.Items {
		if item.IsPendingValidation() {
			return nil, ErrCartItemsPendingValidation
		}
	}

	cart.CalculateTotals()

	if err := s.repo.UpdateCart(ctx, cart); err != nil {
		return nil, fmt.Errorf("failed to update cart totals: %w", err)
	}

	checkedOutCart, err := s.repo.CheckoutCart(ctx, cartID)
	if err != nil {
		return nil, fmt.Errorf("checkout failed: %w", err)
	}

	s.logger.Info("Cart checked out successfully", "cart_id", cartID)

	return checkedOutCart, nil
}

func (s *CartService) CalculateTax(ctx context.Context, netPrice float64) (float64, error) {
	return netPrice * 0.03, nil
}

func (s *CartService) CalculateShipping(ctx context.Context, cart *Cart) (float64, error) {
	return 0.0, nil
}

// GetRepository returns the cart repository for use by event handlers
func (s *CartService) GetRepository() CartRepository {
	return s.repo
}

// GetInfrastructure returns the cart infrastructure for use by event handlers
func (s *CartService) GetInfrastructure() *CartInfrastructure {
	return s.infrastructure
}

// GetProductCache returns the product cache for use by event handlers.
func (s *CartService) GetProductCache() *ProductCache {
	return s.productCache
}
