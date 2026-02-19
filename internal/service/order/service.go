package order

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/outbox"
	"go-shopping-poc/internal/platform/service"
)

type OrderInfrastructure struct {
	Database        database.Database
	EventBus        bus.Bus
	OutboxWriter    *outbox.Writer
	OutboxPublisher *outbox.Publisher
	CORSHandler     func(http.Handler) http.Handler
}

func NewOrderInfrastructure(
	db database.Database,
	eventBus bus.Bus,
	outboxWriter *outbox.Writer,
	outboxPublisher *outbox.Publisher,
	corsHandler func(http.Handler) http.Handler,
) *OrderInfrastructure {
	return &OrderInfrastructure{
		Database:        db,
		EventBus:        eventBus,
		OutboxWriter:    outboxWriter,
		OutboxPublisher: outboxPublisher,
		CORSHandler:     corsHandler,
	}
}

type Service interface {
	service.Service
}

func RegisterHandler[T events.Event](s Service, factory events.EventFactory[T], handler bus.HandlerFunc[T]) error {
	return service.RegisterHandler(s, factory, handler)
}

type OrderService struct {
	*service.EventServiceBase
	repo           OrderRepository
	infrastructure *OrderInfrastructure
	config         *Config
}

func NewOrderService(infrastructure *OrderInfrastructure, config *Config) *OrderService {
	repo := NewOrderRepository(infrastructure.Database, infrastructure.OutboxWriter)

	return &OrderService{
		EventServiceBase: service.NewEventServiceBase("order", infrastructure.EventBus),
		repo:             repo,
		infrastructure:   infrastructure,
		config:           config,
	}
}

func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	log.Printf("[DEBUG] OrderService: Getting order %s", orderID)

	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return order, nil
}

func (s *OrderService) GetOrdersByCustomer(ctx context.Context, customerID string) ([]Order, error) {
	log.Printf("[DEBUG] OrderService: Getting orders for customer %s", customerID)

	orders, err := s.repo.GetOrdersByCustomerID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	return orders, nil
}

func (s *OrderService) CreateOrderFromSnapshot(ctx context.Context, cartID string, snapshot *events.CartSnapshot) (*Order, error) {
	log.Printf("[DEBUG] OrderService: Creating order from cart snapshot %s", cartID)

	cartIDUUID, err := uuid.Parse(cartID)
	if err != nil {
		return nil, fmt.Errorf("invalid cart ID: %w", err)
	}

	order := &Order{
		CartID:        cartIDUUID,
		Currency:      snapshot.Currency,
		NetPrice:      snapshot.NetPrice,
		Tax:           snapshot.Tax,
		Shipping:      snapshot.Shipping,
		TotalPrice:    snapshot.TotalPrice,
		CurrentStatus: "created",
		Items:         make([]OrderItem, len(snapshot.Items)),
	}

	if snapshot.Contact != nil {
		order.Contact = &Contact{
			Email:     snapshot.Contact.Email,
			FirstName: snapshot.Contact.FirstName,
			LastName:  snapshot.Contact.LastName,
			Phone:     snapshot.Contact.Phone,
		}
	}

	if snapshot.CreditCard != nil {
		order.CreditCard = &CreditCard{
			CardType:       snapshot.CreditCard.CardType,
			CardNumber:     snapshot.CreditCard.CardNumber,
			CardHolderName: snapshot.CreditCard.CardHolderName,
			CardExpires:    snapshot.CreditCard.CardExpires,
			CardCVV:        snapshot.CreditCard.CardCVV,
		}
	}

	for i, item := range snapshot.Items {
		order.Items[i] = OrderItem{
			LineNumber:     i + 1,
			ProductID:      item.ProductID,
			ProductName:    item.ProductName,
			UnitPrice:      item.UnitPrice,
			Quantity:       item.Quantity,
			TotalPrice:     item.TotalPrice,
			ItemStatus:     "pending",
			ItemStatusDate: time.Now(),
		}
	}

	for _, addr := range snapshot.Addresses {
		order.Addresses = append(order.Addresses, Address{
			AddressType: addr.AddressType,
			FirstName:   addr.FirstName,
			LastName:    addr.LastName,
			Address1:    addr.Address1,
			Address2:    addr.Address2,
			City:        addr.City,
			State:       addr.State,
			Zip:         addr.Zip,
		})
	}

	if err := s.repo.CreateOrder(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Trigger immediate outbox processing for low latency
	if s.infrastructure.OutboxPublisher != nil {
		go func() {
			if err := s.infrastructure.OutboxPublisher.ProcessNow(); err != nil {
				log.Printf("[WARN] Order: Failed to trigger immediate outbox processing: %v", err)
			}
		}()
	}

	return order, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID string) error {
	log.Printf("[DEBUG] OrderService: Cancelling order %s", orderID)

	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if err := order.CanCancel(); err != nil {
		return fmt.Errorf("cannot cancel order: %w", err)
	}

	if err := s.repo.UpdateOrderStatus(ctx, orderID, "cancelled", "Order cancelled by customer"); err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	return nil
}

func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderID string, newStatus string) error {
	log.Printf("[DEBUG] OrderService: Updating order %s status to %s", orderID, newStatus)

	order, err := s.repo.GetOrderByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	if err := order.SetStatus(newStatus); err != nil {
		return fmt.Errorf("invalid status transition: %w", err)
	}

	if err := s.repo.UpdateOrderStatus(ctx, orderID, newStatus, ""); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}
