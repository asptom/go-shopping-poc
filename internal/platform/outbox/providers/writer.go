package providers

import (
	"log"

	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/outbox"
)

// WriterProvider provides outbox writer functionality
type WriterProvider interface {
	GetWriter() *outbox.Writer
}

// WriterProviderImpl implements writer-only provider
type WriterProviderImpl struct {
	writer *outbox.Writer
}

// NewWriterProvider creates a writer-only provider
func NewWriterProvider(db database.Database) WriterProvider {
	log.Printf("[DEBUG] WriterProvider: Creating writer-only provider")
	return &WriterProviderImpl{
		writer: outbox.NewWriter(db),
	}
}

// GetWriter returns the outbox writer instance
func (p *WriterProviderImpl) GetWriter() *outbox.Writer {
	return p.writer
}
