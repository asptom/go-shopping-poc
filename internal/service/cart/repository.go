package cart

import (
	"context"
	"errors"

	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/outbox"
)

var (
	ErrCartNotFound        = errors.New("cart not found")
	ErrCartItemNotFound    = errors.New("cart item not found")
	ErrAddressNotFound     = errors.New("address not found")
	ErrContactNotFound     = errors.New("contact not found")
	ErrCreditCardNotFound  = errors.New("credit card not found")
	ErrInvalidUUID         = errors.New("invalid UUID format")
	ErrDatabaseOperation   = errors.New("database operation failed")
	ErrTransactionFailed   = errors.New("transaction failed")
	ErrDuplicateActiveCart = errors.New("customer already has an active cart")
)

type CartRepository interface {
	CreateCart(ctx context.Context, cart *Cart) error
	GetCartByID(ctx context.Context, cartID string) (*Cart, error)
	UpdateCart(ctx context.Context, cart *Cart) error
	DeleteCart(ctx context.Context, cartID string) error
	GetActiveCartByCustomerID(ctx context.Context, customerID string) (*Cart, error)

	AddItem(ctx context.Context, cartID string, item *CartItem) error
	UpdateItemQuantity(ctx context.Context, cartID string, lineNumber string, quantity int) error
	RemoveItem(ctx context.Context, cartID string, lineNumber string) error
	GetCartItems(ctx context.Context, cartID string) ([]CartItem, error)

	SetContact(ctx context.Context, cartID string, contact *Contact) error
	GetContact(ctx context.Context, cartID string) (*Contact, error)
	AddAddress(ctx context.Context, cartID string, address *Address) error
	GetAddresses(ctx context.Context, cartID string) ([]Address, error)
	UpdateAddress(ctx context.Context, addressID int64, address *Address) error
	RemoveAddress(ctx context.Context, addressID int64) error

	SetCreditCard(ctx context.Context, cartID string, card *CreditCard) error
	GetCreditCard(ctx context.Context, cartID string) (*CreditCard, error)
	RemoveCreditCard(ctx context.Context, cartID string) error

	GetStatusHistory(ctx context.Context, cartID string) ([]CartStatus, error)
	AddStatusEntry(ctx context.Context, cartID string, status string) error

	CheckoutCart(ctx context.Context, cartID string) (*Cart, error)
}

type cartRepository struct {
	db           database.Database
	outboxWriter *outbox.Writer
}

func NewCartRepository(db database.Database, outbox *outbox.Writer) CartRepository {
	return &cartRepository{
		db:           db,
		outboxWriter: outbox,
	}
}

var _ CartRepository = (*cartRepository)(nil)
