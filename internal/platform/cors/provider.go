package cors

import (
	"fmt"
	"log"
	"net/http"

	"go-shopping-poc/internal/platform/config"
	"go-shopping-poc/internal/platform/providers"
)

// CORSProviderImpl implements the CORSProvider interface.
// It encapsulates CORS configuration loading and middleware creation,
// providing a configured CORS handler to services.
type CORSProviderImpl struct {
	corsHandler func(http.Handler) http.Handler
}

// CORSProvider defines the interface for providing CORS middleware.
// This interface is implemented by CORSProviderImpl.
type CORSProvider interface {
	providers.CORSProvider
}

// NewCORSProvider creates a new CORS provider with loaded configuration.
// It loads the platform-cors configuration, validates it, and creates
// a CORS middleware handler that can be used in HTTP servers.
//
// Returns:
//   - A configured CORSProvider that provides CORS middleware
//   - An error if configuration loading or validation fails
//
// Usage:
//
//	provider, err := cors.NewCORSProvider()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	corsHandler := provider.GetCORSHandler()
func NewCORSProvider() (CORSProvider, error) {
	log.Printf("[INFO] CORSProvider: Initializing CORS provider")

	// Load platform CORS configuration
	corsCfg, err := config.LoadConfig[Config]("platform-cors")
	if err != nil {
		log.Printf("[ERROR] CORSProvider: Failed to load CORS config: %v", err)
		return nil, fmt.Errorf("failed to load CORS config: %w", err)
	}

	log.Printf("[DEBUG] CORSProvider: CORS config loaded successfully")

	// Validate the configuration
	if err := corsCfg.Validate(); err != nil {
		log.Printf("[ERROR] CORSProvider: CORS config validation failed: %v", err)
		return nil, fmt.Errorf("CORS config validation failed: %w", err)
	}

	log.Printf("[DEBUG] CORSProvider: CORS config validated successfully")

	// Create CORS middleware handler
	corsHandler := NewFromConfig(corsCfg)

	log.Printf("[INFO] CORSProvider: CORS provider initialized successfully")

	return &CORSProviderImpl{
		corsHandler: corsHandler,
	}, nil
}

// GetCORSHandler returns the configured CORS middleware handler function.
// The handler can be used as middleware in HTTP servers to apply CORS
// policies according to the loaded configuration.
//
// Returns:
//   - A func(http.Handler) http.Handler that applies CORS headers and policies
//
// Usage:
//
//	corsHandler := provider.GetCORSHandler()
//	wrappedHandler := corsHandler(yourHandler)
//	http.Handle("/api/", wrappedHandler)
func (p *CORSProviderImpl) GetCORSHandler() func(http.Handler) http.Handler {
	return p.corsHandler
}
