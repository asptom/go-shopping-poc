package cart

import "errors"

var (
	ErrCartNotReadyForCheckout        = errors.New("cart not ready for checkout")
	ErrCartMustBeActiveForCheckout    = errors.New("cart must be active to checkout")
	ErrCartMustHaveItemsForCheckout   = errors.New("cart must have at least one item")
	ErrCartContactRequiredForCheckout = errors.New("contact information required")
	ErrCartPaymentRequiredForCheckout = errors.New("payment method required")
	ErrCartItemsPendingValidation     = errors.New("cannot checkout: some items are still being validated, please wait")
)
