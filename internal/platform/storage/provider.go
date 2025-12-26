package storage

import (
	"fmt"
	"log"
	"os"

	"go-shopping-poc/internal/platform/config"
	"go-shopping-poc/internal/platform/storage/minio"
)

// StorageProviderImpl implements the StorageProvider interface.
// It encapsulates MinIO object storage setup and provides a configured
// MinIO client to services.
type StorageProviderImpl struct {
	storage minio.ObjectStorage
}

// StorageProvider defines the interface for providing object storage.
// This interface is implemented by StorageProviderImpl.
type StorageProvider interface {
	GetObjectStorage() minio.ObjectStorage
}

// NewStorageProvider creates a new storage provider.
// It loads platform MinIO configuration, determines the appropriate endpoint
// based on the environment (Kubernetes vs Local), creates a MinIO client,
// and establishes a connection to the object storage.
//
// Returns:
//   - A configured StorageProvider that provides object storage connectivity
//   - An error if configuration loading, client creation, or connection fails
//
// Usage:
//
//	provider, err := storage.NewStorageProvider()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	storage := provider.GetObjectStorage()
func NewStorageProvider() (StorageProvider, error) {
	log.Printf("[INFO] StorageProvider: Initializing storage provider")

	// Load platform MinIO configuration
	platformCfg, err := config.LoadConfig[minio.PlatformConfig]("platform-minio")
	if err != nil {
		log.Printf("[ERROR] StorageProvider: Failed to load MinIO config: %v", err)
		return nil, fmt.Errorf("failed to load MinIO config: %w", err)
	}

	log.Printf("[DEBUG] StorageProvider: MinIO config loaded successfully")

	// Determine endpoint based on environment
	endpoint := platformCfg.EndpointLocal
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		endpoint = platformCfg.EndpointKubernetes
		log.Printf("[DEBUG] StorageProvider: Using Kubernetes endpoint: %s", endpoint)
	} else {
		log.Printf("[DEBUG] StorageProvider: Using local endpoint: %s", endpoint)
	}

	// Create MinIO client configuration
	minioCfg := &minio.Config{
		Endpoint:  endpoint,
		AccessKey: platformCfg.AccessKey,
		SecretKey: platformCfg.SecretKey,
		Secure:    platformCfg.TLSVerify,
	}

	// Create MinIO client
	storageClient, err := minio.NewClient(minioCfg)
	if err != nil {
		log.Printf("[ERROR] StorageProvider: Failed to create MinIO client: %v", err)
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	log.Printf("[INFO] StorageProvider: Storage provider initialized successfully")

	return &StorageProviderImpl{
		storage: storageClient,
	}, nil
}

// GetObjectStorage returns the configured object storage instance.
// The storage client is already initialized and ready for use.
//
// Returns:
//   - A ObjectStorage interface implementation that can be used for
//     bucket operations, object uploads/downloads, etc.
//
// Usage:
//
//	storage := provider.GetObjectStorage()
//	err := storage.PutObject(ctx, bucket, objectName, reader, size, opts)
func (p *StorageProviderImpl) GetObjectStorage() minio.ObjectStorage {
	return p.storage
}
