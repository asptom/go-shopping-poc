package providers

import (
	"log/slog"

	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/outbox"
)

// WriterOption is a functional option for configuring WriterProviderImpl.
type WriterOption func(*WriterProviderImpl)

// WithWriterLogger sets the logger for the WriterProviderImpl.
func WithWriterLogger(logger *slog.Logger) WriterOption {
	return func(p *WriterProviderImpl) {
		p.logger = logger
	}
}

// WriterProvider provides outbox writer functionality
type WriterProvider interface {
	GetWriter() *outbox.Writer
}

// WriterProviderImpl implements writer-only provider
type WriterProviderImpl struct {
	writer *outbox.Writer
	logger *slog.Logger
}

// NewWriterProvider creates a writer-only provider
//
// Parameters:
//   - db: Database instance for the outbox writer
//   - opts: Optional functional options for configuring the provider
//
// Usage:
//
//	provider := providers.NewWriterProvider(db)
//	// or with custom logger
//	provider := providers.NewWriterProvider(db, providers.WithLogger(logger))
func NewWriterProvider(db database.Database, opts ...WriterOption) WriterProvider {
	p := &WriterProviderImpl{}

	for _, opt := range opts {
		opt(p)
	}

	if p.logger == nil {
		p.logger = Logger()
	}

	outboxLogger := p.logger.With("platform", "outbox", "component", "writer")
	p.logger.Debug("WriterProvider: Creating writer-only provider")
	return &WriterProviderImpl{
		writer: outbox.NewWriter(db, outbox.WithLogger(outboxLogger)),
		logger: p.logger,
	}
}

// GetWriter returns the outbox writer instance
func (p *WriterProviderImpl) GetWriter() *outbox.Writer {
	return p.writer
}
