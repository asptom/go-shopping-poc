package downloader

import (
	"fmt"
	"log/slog"
	"time"
)

// ProviderOption is a functional option for configuring DownloaderProviderImpl.
type ProviderOption func(*DownloaderProviderImpl)

// WithLogger sets the logger for the DownloaderProviderImpl.
func WithLogger(logger *slog.Logger) ProviderOption {
	return func(p *DownloaderProviderImpl) {
		p.logger = logger
	}
}

// DownloaderProviderImpl implements the DownloaderProvider interface.
// It encapsulates HTTP downloader setup and provides a configured
// HTTP downloader instance to services.
type DownloaderProviderImpl struct {
	downloader HTTPDownloader
	logger     *slog.Logger
}

// DownloaderProvider defines the interface for providing HTTP downloading with caching.
// This interface matches the one defined in the providers package.
type DownloaderProvider interface {
	// GetHTTPDownloader returns a configured HTTP downloader instance
	GetHTTPDownloader() HTTPDownloader
}

// DownloaderProviderConfig defines the configuration for creating a downloader provider.
type DownloaderProviderConfig struct {
	// CacheDir is the directory where downloaded files will be cached
	CacheDir string

	// CacheMaxAge defines how long cached files are considered valid (optional)
	CacheMaxAge time.Duration

	// CacheMaxSize defines the maximum cache size in bytes (0 = unlimited, optional)
	CacheMaxSize int64
}

// NewDownloaderProvider creates a new downloader provider with the given configuration.
// It creates an HTTP downloader with caching capabilities configured according
// to the provided settings.
//
// Parameters:
//   - config: Configuration specifying cache directory and policy
//   - opts: Optional functional options for configuring the provider
//
// Returns:
//   - A configured DownloaderProvider that provides HTTP downloading with caching
//   - An error if downloader creation fails
//
// Usage:
//
//	config := downloader.DownloaderProviderConfig{
//	    CacheDir: "/tmp/product-images",
//	    CacheMaxAge: 24 * time.Hour,
//	    CacheMaxSize: 100 * 1024 * 1024, // 100MB
//	}
//	provider, err := downloader.NewDownloaderProvider(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	downloader := provider.GetHTTPDownloader()
//
// Or with custom logger:
//
//	provider, err := downloader.NewDownloaderProvider(config, downloader.WithLogger(logger))
func NewDownloaderProvider(config DownloaderProviderConfig, opts ...ProviderOption) (DownloaderProvider, error) {
	p := &DownloaderProviderImpl{}

	for _, opt := range opts {
		opt(p)
	}

	if p.logger == nil {
		p.logger = Logger()
	}

	p.logger = p.logger.With("platform", "downloader")

	if config.CacheDir == "" {
		return nil, fmt.Errorf("cache directory is required")
	}

	p.logger.Info("DownloaderProvider: Initializing downloader provider with cache dir", "cacheDir", config.CacheDir)

	// Set default cache policy if not specified
	cachePolicy := CachePolicy{
		MaxAge:  24 * time.Hour, // Default to 24 hours
		MaxSize: 0,              // Default to unlimited
	}

	if config.CacheMaxAge > 0 {
		cachePolicy.MaxAge = config.CacheMaxAge
	}

	if config.CacheMaxSize > 0 {
		cachePolicy.MaxSize = config.CacheMaxSize
	}

	p.logger.Debug("DownloaderProvider: Cache policy", "maxAge", cachePolicy.MaxAge, "maxSize", cachePolicy.MaxSize)

	// Create HTTP downloader with cache policy
	httpDownloader, err := NewHTTPDownloader(config.CacheDir, WithCachePolicy(cachePolicy))
	if err != nil {
		p.logger.Error("DownloaderProvider: Failed to create HTTP downloader", "error", err)
		return nil, fmt.Errorf("failed to create HTTP downloader: %w", err)
	}

	p.logger.Info("DownloaderProvider: Downloader provider initialized successfully")

	return &DownloaderProviderImpl{
		downloader: httpDownloader,
		logger:     p.logger,
	}, nil
}

// GetHTTPDownloader returns the configured HTTP downloader instance.
// The downloader is ready for downloading files with caching enabled.
//
// Returns:
//   - A HTTPDownloader interface implementation that can be used for
//     downloading files from URLs with intelligent caching
//
// Usage:
//
//	downloader := provider.GetHTTPDownloader()
//	localPath, err := downloader.Download(ctx, "https://example.com/image.jpg")
func (p *DownloaderProviderImpl) GetHTTPDownloader() HTTPDownloader {
	return p.downloader
}
