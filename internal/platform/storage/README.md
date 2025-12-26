# Object Storage Provider

This package provides a storage provider implementation that encapsulates MinIO object storage setup for the shopping platform. It implements Clean Architecture principles by providing a reusable storage infrastructure layer that handles configuration loading, environment-based endpoint selection, and client initialization.

## Features

- **Provider Pattern**: Clean abstraction for object storage dependency injection
- **Environment-Aware**: Automatic endpoint selection (Kubernetes vs Local)
- **MinIO Integration**: Full MinIO client initialization with proper configuration
- **Error Handling**: Structured error handling with comprehensive logging
- **Interface Compliance**: Implements the platform StorageProvider interface
- **Comprehensive Testing**: Full test coverage for provider functionality

## Architecture

```
internal/platform/storage/
├── minio/                    # MinIO-specific implementation
│   ├── interface.go         # ObjectStorage interface
│   ├── client.go            # MinIO client implementation
│   ├── config.go            # MinIO configuration
│   └── client_test.go       # MinIO client tests
├── provider.go              # Storage provider implementation
├── provider_test.go         # Provider tests
└── README.md               # This documentation
```

## Usage

### Basic Setup

```go
import "go-shopping-poc/internal/platform/storage"

// Create storage provider (handles config loading and client initialization)
provider, err := storage.NewStorageProvider()
if err != nil {
    log.Fatal(err)
}

// Get object storage client
storageClient := provider.GetObjectStorage()
```

### Service Integration

```go
type MyService struct {
    storage storage.StorageProvider
}

func NewMyService(storage storage.StorageProvider) *MyService {
    return &MyService{storage: storage}
}

func (s *MyService) UploadFile(ctx context.Context, bucket, key string, data io.Reader) error {
    storage := s.storage.GetObjectStorage()
    _, err := storage.PutObject(ctx, bucket, key, data, -1, minio.PutObjectOptions{})
    return err
}
```

## Configuration

The storage provider is configured via environment variables:

```env
MINIO_ENDPOINT_KUBERNETES=minio.minio.svc.cluster.local:9000
MINIO_ENDPOINT_LOCAL=api.minio.local
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadminpassword
MINIO_TLS_VERIFY=false
```

### Endpoint Selection

The provider automatically selects the appropriate endpoint based on the environment:

- **Local Development**: Uses `MINIO_ENDPOINT_LOCAL` when `KUBERNETES_SERVICE_HOST` is not set
- **Kubernetes**: Uses `MINIO_ENDPOINT_KUBERNETES` when `KUBERNETES_SERVICE_HOST` is set

## Provider Interface

```go
type StorageProvider interface {
    GetObjectStorage() minio.ObjectStorage
}
```

The provider encapsulates:
- Platform configuration loading (`config.LoadConfig[minio.PlatformConfig]("platform-minio")`)
- Environment detection and endpoint selection
- MinIO client creation (`minio.NewClient()`)
- Error handling and logging

## Error Handling

The provider includes comprehensive error handling:

```go
provider, err := storage.NewStorageProvider()
if err != nil {
    // Possible errors:
    // - Config loading failure
    // - Invalid configuration
    // - MinIO client creation failure
    log.Printf("[ERROR] StorageProvider: Failed to initialize: %v", err)
    return err
}
```

## Testing

The provider includes comprehensive tests:

```bash
# Run provider tests
go test ./internal/platform/storage/

# Run all storage package tests
go test ./internal/platform/storage/...
```

Tests cover:
- Successful provider creation
- Configuration loading errors
- Interface compliance
- Environment-based endpoint selection

## Clean Architecture Benefits

- **Separation of Concerns**: Platform handles infrastructure, services handle business logic
- **Dependency Injection**: Services depend on interfaces, not concrete implementations
- **Testability**: Provider pattern enables easy mocking and testing
- **Reusability**: Storage provider can be used by any service requiring object storage
- **Maintainability**: Centralized storage configuration and initialization

## Integration with Services

Services integrate with the storage provider through dependency injection:

```go
// In service constructor
func NewProductService(
    repo ProductRepository,
    config *Config,
    storage storage.StorageProvider,  // Provider interface
    // ... other dependencies
) *ProductService {
    return &ProductService{
        storage: storage,  // Inject provider
        // ...
    }
}

// In service usage
func (s *ProductService) UploadImage(ctx context.Context, data []byte) error {
    storage := s.storage.GetObjectStorage()  // Get client
    _, err := storage.PutObject(ctx, s.bucket, objectName, bytes.NewReader(data), int64(len(data)), opts)
    return err
}
```

This pattern ensures services remain focused on business logic while delegating infrastructure concerns to the platform layer.</content>
<parameter name="filePath">internal/platform/storage/README.md