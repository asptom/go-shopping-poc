// Package product provides the product service implementation.
// This package contains the domain logic, entities, and handlers for product management.
package product

import (
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/downloader"
	"go-shopping-poc/internal/platform/outbox"
	"go-shopping-poc/internal/platform/storage/minio"
)

// ProductInfrastructure defines the infrastructure components required by the product service.
// This struct encapsulates all external dependencies that the product service needs to function,
// following clean architecture principles by keeping infrastructure concerns separate from domain logic.
//
// The product service requires these infrastructure components for:
// - Database: Persistent storage and transactions for product data
// - ObjectStorage: Storing and retrieving product images (MinIO)
// - OutboxWriter: Writing product events to the outbox table within transactions
// - HTTPDownloader: Downloading and caching product images from external URLs
type ProductInfrastructure struct {
	// Database provides PostgreSQL connectivity for data persistence and transactions
	Database database.Database

	// ObjectStorage manages product image storage in MinIO
	ObjectStorage minio.ObjectStorage

	// OutboxWriter writes product events to the outbox table within database transactions
	OutboxWriter *outbox.Writer

	// HTTPDownloader downloads and caches product images from external URLs
	HTTPDownloader downloader.HTTPDownloader
}
