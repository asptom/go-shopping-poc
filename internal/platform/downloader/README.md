# HTTP Downloader Package

This package provides a generic HTTP downloader with caching capabilities for the go-shopping-poc platform.

## Features

- **Generic HTTP Downloads**: Download any file from HTTP/HTTPS URLs
- **Intelligent Caching**: Automatic caching with configurable policies
- **Configurable Options**: Timeout, retries, headers, user agent
- **Cache Management**: Statistics, cleanup, size limits
- **Concurrent Safe**: Thread-safe for concurrent downloads
- **Atomic Downloads**: Temporary files prevent corruption

## Usage

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "go-shopping-poc/internal/platform/downloader"
)

func main() {
    // Create downloader with default options
    dl, err := downloader.NewHTTPDownloader("/tmp/cache")
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()
    localPath, err := dl.Download(ctx, "https://example.com/image.jpg")
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Downloaded to: %s", localPath)
}
```

### Advanced Configuration

```go
// Create downloader with custom options
dl, err := downloader.NewHTTPDownloader("/tmp/cache",
    downloader.WithTimeout(10*time.Second),
    downloader.WithMaxRetries(5),
    downloader.WithRetryDelay(2*time.Second),
    downloader.WithUserAgent("my-app/1.0"),
    downloader.WithHeaders(map[string]string{
        "Authorization": "Bearer token",
        "X-API-Key": "key",
    }),
    downloader.WithCachePolicy(downloader.CachePolicy{
        MaxAge:  24 * time.Hour,
        MaxSize: 100 * 1024 * 1024, // 100MB
    }),
)
```

### Cache Management

```go
// Check if URL is cached
if dl.IsCached("https://example.com/file.zip") {
    log.Println("File is cached")
}

// Get cache statistics
stats, err := dl.GetCacheStats()
if err != nil {
    log.Fatal(err)
}
log.Printf("Cache: %d files, %d bytes", stats.TotalFiles, stats.TotalSize)

// Clear all cached files
err = dl.ClearCache()
if err != nil {
    log.Fatal(err)
}
```

## Cache Policy

The downloader supports configurable cache policies:

```go
type CachePolicy struct {
    MaxAge  time.Duration // How long files are considered valid
    MaxSize int64         // Maximum cache size in bytes (0 = unlimited)
}
```

### Cache Behavior

- Files are cached based on URL hash + extension
- Cache validity is checked before downloading
- Expired files are automatically re-downloaded
- Size limits trigger LRU-style cleanup (oldest files first)
- Background cleanup runs at configured intervals

## Interface

The package provides the `HTTPDownloader` interface for extensibility:

```go
type HTTPDownloader interface {
    Download(ctx context.Context, url string) (string, error)
    IsCached(url string) bool
    GetCachePath(url string) string
    ClearCache() error
    GetCacheStats() (*CacheStats, error)
    SetCachePolicy(policy CachePolicy)
}
```

## Error Handling

The downloader returns standard Go errors. Common error scenarios:

- Network errors: Connection timeouts, DNS failures
- HTTP errors: Non-2xx status codes
- File system errors: Permission issues, disk full
- Cache errors: Directory creation failures

## Testing

Run the tests:

```bash
go test ./internal/platform/downloader/...
```

The tests cover:
- Basic download functionality
- Caching behavior
- Error handling
- Cache management
- Concurrent safety
- Configuration options

## Migration from Legacy Downloader

This package replaces the image-specific downloader in `temp/product-loader/internal/downloader/`. Key changes:

1. **Generic**: Works with any HTTP content, not just images
2. **Interface-based**: Extensible design with interfaces
3. **Configurable**: Comprehensive options for different use cases
4. **Better caching**: Age and size-based policies
5. **Atomic operations**: Prevents corrupted downloads

### Migration Example

**Before:**
```go
// Image-specific downloader
imageDownloader := downloader.NewDownloader(cacheDir)
localPath, err := imageDownloader.DownloadImage(productID, index, imageURL)
```

**After:**
```go
// Generic downloader
httpDownloader, _ := downloader.NewHTTPDownloader(cacheDir)
localPath, err := httpDownloader.Download(ctx, imageURL)

// Domain-specific filename generation (in service layer)
filename := generateProductImageFilename(productID, index, imageURL)
```

Domain-specific logic like filename generation and URL filtering should be moved to the service layer.