package downloader

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Default values
const (
	DefaultTimeout   = 30 * time.Second
	DefaultUserAgent = "go-shopping-poc/1.0"
)

// HTTPDownloaderImpl implements the HTTPDownloader interface
type HTTPDownloaderImpl struct {
	cacheDir    string
	cachePolicy CachePolicy
	httpClient  *http.Client
	userAgent   string
	headers     map[string]string
}

// NewHTTPDownloader creates a new HTTP downloader with caching
func NewHTTPDownloader(cacheDir string, options ...Option) (HTTPDownloader, error) {
	// Apply default options
	opts := &optionsStruct{
		timeout:   DefaultTimeout,
		userAgent: DefaultUserAgent,
		headers:   make(map[string]string),
		cachePolicy: CachePolicy{
			MaxAge:  24 * time.Hour, // 24 hours
			MaxSize: 0,              // unlimited
		},
	}

	// Apply user options
	for _, option := range options {
		option(opts)
	}

	// Create HTTP client
	httpClient := &http.Client{
		Timeout: opts.timeout,
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory %s: %w", cacheDir, err)
	}

	downloader := &HTTPDownloaderImpl{
		cacheDir:    cacheDir,
		cachePolicy: opts.cachePolicy,
		httpClient:  httpClient,
		userAgent:   opts.userAgent,
		headers:     opts.headers,
	}

	return downloader, nil
}

// Option defines functional options for the downloader
type Option func(*optionsStruct)

type optionsStruct struct {
	timeout     time.Duration
	userAgent   string
	headers     map[string]string
	cachePolicy CachePolicy
}

// WithTimeout sets the HTTP timeout
func WithTimeout(timeout time.Duration) Option {
	return func(o *optionsStruct) {
		o.timeout = timeout
	}
}

// WithUserAgent sets the User-Agent header
func WithUserAgent(userAgent string) Option {
	return func(o *optionsStruct) {
		o.userAgent = userAgent
	}
}

// WithHeaders sets additional HTTP headers
func WithHeaders(headers map[string]string) Option {
	return func(o *optionsStruct) {
		o.headers = headers
	}
}

// WithCachePolicy sets the cache policy
func WithCachePolicy(policy CachePolicy) Option {
	return func(o *optionsStruct) {
		o.cachePolicy = policy
	}
}

// Download downloads a file from the given URL and returns the local cached path
func (d *HTTPDownloaderImpl) Download(ctx context.Context, url string) (string, error) {
	cachePath := d.GetCachePath(url)

	// Check if already cached and valid
	if d.IsCached(url) && d.isCacheValid(cachePath) {
		return cachePath, nil
	}

	// Download the file
	err := d.downloadFile(ctx, url, cachePath)
	if err != nil {
		return "", fmt.Errorf("failed to download %s: %w", url, err)
	}

	return cachePath, nil
}

// IsCached checks if a URL is already cached
func (d *HTTPDownloaderImpl) IsCached(url string) bool {
	cachePath := d.GetCachePath(url)
	_, err := os.Stat(cachePath)
	return err == nil
}

// GetCachePath returns the cache path for a given URL without downloading
func (d *HTTPDownloaderImpl) GetCachePath(url string) string {
	// Create hash of URL for uniqueness
	hash := sha256.Sum256([]byte(url))
	hashStr := fmt.Sprintf("%x", hash)[:16]

	// Extract extension from URL
	ext := d.extractExtension(url)

	return filepath.Join(d.cacheDir, fmt.Sprintf("%s%s", hashStr, ext))
}

// ClearCache removes all cached files
func (d *HTTPDownloaderImpl) ClearCache() error {
	files, err := os.ReadDir(d.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	var lastErr error
	for _, file := range files {
		if !file.IsDir() {
			filePath := filepath.Join(d.cacheDir, file.Name())
			if err := os.Remove(filePath); err != nil {
				lastErr = fmt.Errorf("failed to remove %s: %w", filePath, err)
			}
		}
	}

	return lastErr
}

// GetCacheStats returns statistics about the cache
func (d *HTTPDownloaderImpl) GetCacheStats() (*CacheStats, error) {
	files, err := os.ReadDir(d.cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache directory: %w", err)
	}

	stats := &CacheStats{
		CacheDirectory: d.cacheDir,
		TotalFiles:     0,
		TotalSize:      0,
	}

	var oldest, newest time.Time
	hasFiles := false

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		stats.TotalFiles++
		stats.TotalSize += info.Size()

		modTime := info.ModTime()
		if !hasFiles {
			oldest = modTime
			newest = modTime
			hasFiles = true
		} else {
			if modTime.Before(oldest) {
				oldest = modTime
			}
			if modTime.After(newest) {
				newest = modTime
			}
		}
	}

	if hasFiles {
		stats.OldestFile = oldest
		stats.NewestFile = newest
	}

	return stats, nil
}

// SetCachePolicy sets the cache policy for the downloader
func (d *HTTPDownloaderImpl) SetCachePolicy(policy CachePolicy) {
	d.cachePolicy = policy
}

// downloadFile performs the actual HTTP download
func (d *HTTPDownloaderImpl) downloadFile(ctx context.Context, url, cachePath string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", d.userAgent)
	for key, value := range d.headers {
		req.Header.Set(key, value)
	}

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	// Create temporary file first
	tempPath := cachePath + ".tmp"
	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = file.Close() }()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		_ = os.Remove(tempPath) // Clean up on error
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Close file before renaming
	_ = file.Close()

	// Atomic rename
	if err := os.Rename(tempPath, cachePath); err != nil {
		_ = os.Remove(tempPath) // Clean up on error
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// isCacheValid checks if a cached file is still valid according to cache policy
func (d *HTTPDownloaderImpl) isCacheValid(cachePath string) bool {
	if d.cachePolicy.MaxAge <= 0 {
		return true // No age limit
	}

	info, err := os.Stat(cachePath)
	if err != nil {
		return false
	}

	return time.Since(info.ModTime()) <= d.cachePolicy.MaxAge
}

// extractExtension extracts file extension from URL
func (d *HTTPDownloaderImpl) extractExtension(url string) string {
	parts := strings.Split(url, ".")
	if len(parts) <= 1 {
		return "" // No extension
	}

	lastPart := strings.ToLower(parts[len(parts)-1])

	// Remove query parameters and fragment from extension
	if idx := strings.Index(lastPart, "?"); idx != -1 {
		lastPart = lastPart[:idx]
	}
	if idx := strings.Index(lastPart, "#"); idx != -1 {
		lastPart = lastPart[:idx]
	}

	// Check if extension is empty or invalid
	if lastPart == "" || len(lastPart) > 10 || strings.Contains(lastPart, "/") {
		return "" // Invalid extension
	}

	return "." + lastPart
}
