package outbox

import (
	"errors"
	"fmt"
)

var (
	// ErrWriteFailed is returned when event writing to outbox fails
	ErrWriteFailed = errors.New("outbox: write operation failed")

	// ErrPublishFailed is returned when event publishing to broker fails
	ErrPublishFailed = errors.New("outbox: publish operation failed")

	// ErrTransactionRollover is returned when transaction is incorrectly rolled back
	ErrTransactionRollover = errors.New("outbox: transaction rolled back unexpectedly")

	// ErrInvalidEvent is returned for malformed event structures
	ErrInvalidEvent = errors.New("outbox: invalid event")
)

// WrapWithContext adds context information to errors
func WrapWithContext(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}
