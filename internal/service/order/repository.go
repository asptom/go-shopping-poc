package order

import (
	"context"
	"errors"
	"log/slog"

	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/outbox"
)

var (
	ErrOrderNotFound           = errors.New("order not found")
	ErrOrderItemNotFound       = errors.New("order item not found")
	ErrInvalidUUID             = errors.New("invalid UUID format")
	ErrDatabaseOperation       = errors.New("database operation failed")
	ErrTransactionFailed       = errors.New("transaction failed")
	ErrInvalidStatusTransition = errors.New("invalid order status transition")
	ErrOrderCannotBeCancelled  = errors.New("order cannot be cancelled in current status")
)

type OrderRepository interface {
	CreateOrder(ctx context.Context, order *Order) error
	GetOrderByID(ctx context.Context, orderID string) (*Order, error)
	GetOrdersByCustomerID(ctx context.Context, customerID string) ([]Order, error)
	GetOrdersByCartID(ctx context.Context, cartID string) ([]Order, error)
	UpdateOrderStatus(ctx context.Context, orderID string, newStatus string, notes string) error

	GetStatusHistory(ctx context.Context, orderID string) ([]OrderStatus, error)
	AddStatusEntry(ctx context.Context, orderID string, status string, notes string) error
}

type orderRepository struct {
	db           database.Database
	outboxWriter *outbox.Writer
	logger       *slog.Logger
}

func NewOrderRepository(db database.Database, outbox *outbox.Writer) OrderRepository {
	return &orderRepository{
		db:           db,
		outboxWriter: outbox,
		logger:       slog.Default().With("component", "order_repository"),
	}
}

var _ OrderRepository = (*orderRepository)(nil)
