# MinIO Storage Client

This package provides a generic, reusable MinIO/S3-compatible object storage client for the Go shopping platform. It implements clean architecture principles by providing infrastructure-level storage operations that can be used by any service requiring object storage.

## Features

- **Generic Interface**: Implements `ObjectStorage` interface for maximum reusability
- **Bucket Management**: Create, check existence, and delete buckets
- **Object Operations**: Upload, download, stat, copy, and delete objects
- **File Operations**: Convenient file upload/download methods
- **Presigned URLs**: Generate temporary access URLs for secure sharing
- **Content Type Detection**: Automatic content type detection based on file extensions
- **Error Handling**: Comprehensive error handling with context
- **Retries**: Configurable retry logic for resilience
- **Metadata Support**: Full support for object metadata and tags

## Usage

### Basic Setup

```go
import "go-shopping-poc/internal/platform/storage/minio"

// Create client configuration
config := &minio.Config{
    Endpoint:    "minio.minio.svc.cluster.local:9000",
    AccessKey:   "your-access-key",
    SecretKey:   "your-secret-key",
    Secure:      false, // Set to true for HTTPS
    Region:      "us-east-1",
    MaxRetries:  3,
}

// Create client
client, err := minio.NewClient(config)
if err != nil {
    log.Fatal(err)
}
```

### Bucket Operations

```go
ctx := context.Background()

// Create bucket
err := client.CreateBucket(ctx, "my-bucket")
if err != nil {
    log.Printf("Failed to create bucket: %v", err)
}

// Check if bucket exists
exists, err := client.BucketExists(ctx, "my-bucket")
if err != nil {
    log.Printf("Failed to check bucket: %v", err)
}

// Delete bucket
err = client.DeleteBucket(ctx, "my-bucket")
if err != nil {
    log.Printf("Failed to delete bucket: %v", err)
}
```

### Object Operations

```go
// Upload object from reader
reader := strings.NewReader("Hello, World!")
info, err := client.PutObject(ctx, "my-bucket", "hello.txt", reader, int64(reader.Len()), minio.PutObjectOptions{
    ContentType: "text/plain",
    Metadata: map[string]string{
        "author": "system",
    },
})

// Upload file directly
info, err := client.FPutObject(ctx, "my-bucket", "image.jpg", "/path/to/image.jpg", minio.PutObjectOptions{
    ContentType: minio.GetContentType("/path/to/image.jpg"),
})

// Download object
reader, err := client.GetObject(ctx, "my-bucket", "hello.txt")
if err == nil {
    defer reader.Close()
    content, _ := io.ReadAll(reader)
    fmt.Println(string(content))
}

// Download to file
err = client.FGetObject(ctx, "my-bucket", "image.jpg", "/tmp/downloaded.jpg")

// Get object info
info, err := client.StatObject(ctx, "my-bucket", "hello.txt")
if err == nil {
    fmt.Printf("Size: %d, Content-Type: %s\n", info.Size, info.ContentType)
}

// Delete object
err = client.RemoveObject(ctx, "my-bucket", "hello.txt")

// Copy object
err = client.CopyObject(ctx, "dest-bucket", "copied.txt", "src-bucket", "original.txt", minio.CopyObjectOptions{})
```

### Presigned URLs

```go
// Generate presigned GET URL (valid for 1 hour)
getURL, err := client.PresignedGetObject(ctx, "my-bucket", "private-file.pdf", 3600)

// Generate presigned PUT URL for uploads
putURL, err := client.PresignedPutObject(ctx, "my-bucket", "upload-target.jpg", 3600)
```

### List Objects

```go
// List all objects in bucket
objectCh := client.ListObjects(ctx, "my-bucket", minio.ListObjectsOptions{
    Prefix:    "images/",
    Recursive: true,
})

for object := range objectCh {
    if object.Err != nil {
        log.Printf("Error listing objects: %v", object.Err)
        break
    }
    fmt.Printf("Object: %s, Size: %d\n", object.Key, object.Size)
}
```

## Configuration

The client can be configured using the platform config system:

```go
import "go-shopping-poc/internal/platform/config"

// Load MinIO configuration
minioConfig, err := config.LoadMinIOConfig()
if err != nil {
    log.Fatal(err)
}

// Convert to storage config
storageConfig := &minio.Config{
    Endpoint:   minioConfig.Endpoint,
    AccessKey:  minioConfig.AccessKey,
    SecretKey:  minioConfig.SecretKey,
    Secure:     !minioConfig.TLSVerify, // Note: inverted logic
}

client, err := minio.NewClient(storageConfig)
```

## Content Types

The package includes automatic content type detection:

```go
contentType := minio.GetContentType("image.jpg") // "image/jpeg"
contentType := minio.GetContentType("document.pdf") // "application/pdf"
contentType := minio.GetContentType("unknown.xyz") // "application/octet-stream"
```

## Error Handling

All methods return descriptive errors with context:

```go
_, err := client.PutObject(ctx, "", "object", reader, size, opts)
if err != nil {
    // Error: "bucket name cannot be empty"
}

_, err = client.GetObject(ctx, "bucket", "nonexistent")
if err != nil {
    // Error: "failed to get object bucket/nonexistent: The specified key does not exist"
}
```

## Testing

Run the tests:

```bash
# Unit tests
go test ./internal/platform/storage/minio/

# Integration tests (requires MinIO server)
go test -tags=integration ./internal/platform/storage/minio/
```

## Architecture Notes

This package follows clean architecture principles:

- **Platform Layer**: Provides generic storage infrastructure
- **Service Layer**: Domain services use this for storage operations
- **Interface-Based**: `ObjectStorage` interface allows for different implementations
- **Dependency Injection**: Services receive the storage client as dependency

## Migration from Product Loader

This package replaces the specific MinIO implementation in `temp/product-loader/internal/storage/minio.go` with a generic, reusable version. The original implementation can be updated to use this new client:

```go
// Before (specific implementation)
storage := &MinIOStorage{client: minioClient, bucket: bucket}

// After (generic interface)
var storage minio.ObjectStorage
storage, _ = minio.NewClient(config)
```

This allows any service to use MinIO storage without duplicating code.