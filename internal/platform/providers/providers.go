// Package providers defines the provider interfaces for infrastructure components.
// This package implements the providers pattern for dependency injection and
// clean architecture, allowing services to access platform infrastructure
// through well-defined interfaces.
//
// Key interfaces:
//   - DatabaseProvider: Provides database connectivity
//   - EventBusProvider: Provides event messaging infrastructure
//   - OutboxProvider: Provides outbox pattern components
//   - CORSProvider: Provides CORS middleware
//   - StorageProvider: Provides object storage
//   - DownloaderProvider: Provides HTTP downloading with caching
//
// Usage patterns:
//   - Services depend on provider interfaces, not concrete implementations
//   - Platform provides concrete implementations of these interfaces
//   - Enables clean separation between business logic and infrastructure
//
// Example usage:
//
//	type MyService struct {
//	    dbProvider DatabaseProvider
//	    busProvider EventBusProvider
//	}
//
//	func NewMyService(dbProvider DatabaseProvider, busProvider EventBusProvider) *MyService {
//	    return &MyService{
//	        dbProvider: dbProvider,
//	        busProvider: busProvider,
//	    }
//	}
//
//	func (s *MyService) DoSomething(ctx context.Context) error {
//	    db := s.dbProvider.GetDatabase()
//	    bus := s.busProvider.GetEventBus()
//	    // Use db and bus...
//	}
package providers

import (
	"net/http"

	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/downloader"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/outbox"
	"go-shopping-poc/internal/platform/storage/minio"
)

// DatabaseProvider defines the interface for providing database connectivity.
// Implementations should return a configured database instance that services
// can use for data persistence operations.
type DatabaseProvider interface {
	// GetDatabase returns a configured database instance
	GetDatabase() database.Database
}

// EventBusProvider defines the interface for providing event messaging infrastructure.
// Implementations should return a configured event bus that services can use
// for publishing and consuming events.
type EventBusProvider interface {
	// GetEventBus returns a configured event bus instance
	GetEventBus() bus.Bus
}

// OutboxProvider defines the interface for providing outbox pattern components.
// The outbox pattern ensures reliable event publishing by storing events in
// the database before publishing them to external systems.
type OutboxProvider interface {
	// GetOutboxWriter returns a configured outbox writer for storing events
	GetOutboxWriter() *outbox.Writer

	// GetOutboxPublisher returns a configured outbox publisher for publishing events
	GetOutboxPublisher() *outbox.Publisher
}

// CORSProvider defines the interface for providing CORS middleware.
// Implementations should return a CORS handler function that can be used
// as middleware in HTTP servers.
type CORSProvider interface {
	// GetCORSHandler returns a CORS middleware handler function
	GetCORSHandler() func(http.Handler) http.Handler
}

// StorageProvider defines the interface for providing object storage.
// Implementations should return a configured object storage client that
// services can use for storing and retrieving files.
type StorageProvider interface {
	// GetObjectStorage returns a configured object storage instance
	GetObjectStorage() minio.ObjectStorage
}

// DownloaderProvider defines the interface for providing HTTP downloading with caching.
// Implementations should return a configured HTTP downloader that services
// can use for downloading files with intelligent caching.
type DownloaderProvider interface {
	// GetHTTPDownloader returns a configured HTTP downloader instance
	GetHTTPDownloader() downloader.HTTPDownloader
}
