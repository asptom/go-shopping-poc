package downloader

import (
	"context"
	"time"
)

// HTTPDownloader defines the interface for HTTP download operations with caching
type HTTPDownloader interface {
	// Download downloads a file from the given URL and returns the local cached path
	Download(ctx context.Context, url string) (string, error)

	// IsCached checks if a URL is already cached
	IsCached(url string) bool

	// GetCachePath returns the cache path for a given URL without downloading
	GetCachePath(url string) string

	// ClearCache removes all cached files
	ClearCache() error

	// GetCacheStats returns statistics about the cache
	GetCacheStats() (*CacheStats, error)

	// SetCachePolicy sets the cache policy for the downloader
	SetCachePolicy(policy CachePolicy)
}

// CachePolicy defines caching behavior
type CachePolicy struct {
	// MaxAge defines how long cached files are considered valid
	MaxAge time.Duration

	// MaxSize defines the maximum cache size in bytes (0 = unlimited)
	MaxSize int64
}

// CacheStats provides information about the cache state
type CacheStats struct {
	TotalFiles     int
	TotalSize      int64
	OldestFile     time.Time
	NewestFile     time.Time
	CacheDirectory string
}

// DownloadOptions contains options for download operations
type DownloadOptions struct {
	// Timeout for the HTTP request
	Timeout time.Duration

	// UserAgent for the HTTP request
	UserAgent string

	// Headers to include in the request
	Headers map[string]string
}
