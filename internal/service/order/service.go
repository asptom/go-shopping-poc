package order

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/logging"
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
	logger         *slog.Logger
	repo           OrderRepository
	infrastructure *OrderInfrastructure
	config         *Config
}

func NewOrderService(logger *slog.Logger, infrastructure *OrderInfrastructure, config *Config) *OrderService {
	if logger == nil {
		logger = logging.FromContext(context.Background())
	}
	repo := NewOrderRepository(infrastructure.Database, infrastructure.OutboxWriter)

	return &OrderService{
		EventServiceBase: service.NewEventServiceBase("order", infrastructure.EventBus, logger),
		logger:           logger.With("component", "order_service"),
		repo:             repo,
		infrastructure:   infrastructure,
		config:           config,
	}
}

func (s *OrderService) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	s.logger.Debug("Getting order", "order_id", orderID)

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
	s.logger.Debug("Getting orders for customer", "customer_id", customerID)

	orders, err := s.repo.GetOrdersByCustomerID(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	return orders, nil
}

func (s *OrderService) CreateOrderFromSnapshot(ctx context.Context, cartID string, snapshot *events.CartSnapshot) (*Order, error) {
	s.logger.Debug("Creating order from cart snapshot", "cart_id", cartID)

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

	if snapshot.CustomerID != nil {
		customerUUID, err := uuid.Parse(*snapshot.CustomerID)
		if err != nil {
			s.logger.Warn("Invalid customer ID in snapshot", "customer_id", *snapshot.CustomerID)
		} else {
			order.CustomerID = &customerUUID
		}
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
			ImageURL:       item.ImageURL,
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
				s.logger.Warn("Failed to trigger immediate outbox processing", "error", err.Error())
			}
		}()
	}

	return order, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID string) error {
	s.logger.Debug("Cancelling order", "order_id", orderID)

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
	s.logger.Debug("Updating order status", "order_id", orderID, "new_status", newStatus)

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
