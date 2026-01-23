package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-shopping-poc/internal/platform/config"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/downloader"
	"go-shopping-poc/internal/platform/outbox"
	"go-shopping-poc/internal/platform/service"
	"go-shopping-poc/internal/platform/storage/minio"
	"go-shopping-poc/internal/service/product"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// CLIConfig holds the parsed command line configuration
type CLIConfig struct {
	CSVPath    string
	BatchID    string
	UseCache   bool
	ResetCache bool
}

// parseFlags parses command line flags and returns CLIConfig
func parseFlags() (*CLIConfig, error) {
	var csvPath = flag.String("csv", "", "Path to CSV file containing products (optional, uses config default if not provided)")
	var batchID = flag.String("batch-id", "", "Optional batch ID for this ingestion (auto-generated if not provided)")
	var useCache = flag.Bool("use-cache", true, "Use cached images when available")
	var resetCache = flag.Bool("reset-cache", false, "Reset image cache before processing")
	flag.Parse()

	finalCSVPath := *csvPath

	// Validate CSV file exists only if a path was provided
	if finalCSVPath != "" {
		if _, err := os.Stat(finalCSVPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("CSV file does not exist: %s", finalCSVPath)
		}
	}

	return &CLIConfig{
		CSVPath:    finalCSVPath,
		BatchID:    getBatchID(*batchID),
		UseCache:   *useCache,
		ResetCache: *resetCache,
	}, nil
}

// runProductLoader executes the main product loader logic
func runProductLoader(ctx context.Context, cliConfig *CLIConfig) error {
	log.Printf("[INFO] Product Loader: Processing CSV file: %s", cliConfig.CSVPath)
	log.Printf("[INFO] Product Loader: Batch ID: %s", cliConfig.BatchID)
	log.Printf("[INFO] Product Loader: Use cache: %t", cliConfig.UseCache)
	log.Printf("[INFO] Product Loader: Reset cache: %t", cliConfig.ResetCache)

	// Load loader-specific configuration
	loaderCfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	log.Printf("[DEBUG] Product Loader: Configuration loaded successfully")

	// If CSV path not provided via CLI, use from configuration
	if cliConfig.CSVPath == "" {
		cliConfig.CSVPath = loaderCfg.CSVPath
		log.Printf("[INFO] Using default CSV path from config: %s", cliConfig.CSVPath)
	}

	// Validate CSV file exists
	if _, err := os.Stat(cliConfig.CSVPath); os.IsNotExist(err) {
		return fmt.Errorf("CSV file does not exist: %s", cliConfig.CSVPath)
	}

	// Load platform MinIO configuration
	minioCfg, err := config.LoadConfig[minio.PlatformConfig]("platform-minio")
	if err != nil {
		return fmt.Errorf("failed to load MinIO config: %w", err)
	}

	// Convert loader config to product service config
	cfg := productConfigFromLoaderConfig(loaderCfg)

	// Load database connection configuration
	dbConnConfigPtr, err := config.LoadConfig[database.ConnectionConfig]("platform-database")
	if err != nil {
		return fmt.Errorf("failed to load database connection config: %w", err)
	}
	dbConnConfig := *dbConnConfigPtr

	// Get database URL from service config
	dbURL := cfg.DatabaseURL
	if dbURL == "" {
		return fmt.Errorf("database URL is required in service config")
	}

	log.Printf("[DEBUG] Product Loader: Database connection config loaded")

	platformDB, err := database.NewPostgreSQLClient(dbURL, dbConnConfig)
	if err != nil {
		return fmt.Errorf("failed to create database client: %w", err)
	}

	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	err = platformDB.Connect(connectCtx)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if err := platformDB.Close(); err != nil {
			log.Printf("[ERROR] Product Loader: Error closing database connection: %v", err)
		}
	}()

	log.Printf("[DEBUG] Product Loader: Database connection established")

	// Initialize HTTP downloader
	httpDownloader, err := downloader.NewHTTPDownloader(cfg.CacheDir)
	if err != nil {
		return fmt.Errorf("failed to create HTTP downloader: %w", err)
	}

	log.Printf("[DEBUG] Product Loader: HTTP downloader initialized")

	// Choose MinIO endpoint based on environment
	minioEndpoint := minioCfg.EndpointLocal
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		minioEndpoint = minioCfg.EndpointKubernetes
	}

	minioStorage, err := minio.NewClient(&minio.Config{
		Endpoint:  minioEndpoint,
		AccessKey: minioCfg.AccessKey,
		SecretKey: minioCfg.SecretKey,
		Secure:    minioCfg.TLSVerify,
	})
	if err != nil {
		return fmt.Errorf("failed to create MinIO storage: %w", err)
	}

	log.Printf("[DEBUG] Product Loader: MinIO storage initialized")

	// Initialize outbox writer
	outboxWriter := outbox.NewWriter(platformDB)
	log.Printf("[DEBUG] Product Loader: Outbox writer initialized")

	// Create product repository
	repo := product.NewProductRepository(platformDB.DB())
	log.Printf("[DEBUG] Product Loader: Product repository created")

	// Create admin infrastructure
	infrastructure := &product.AdminInfrastructure{
		Database:       platformDB,
		ObjectStorage:  minioStorage,
		OutboxWriter:   outboxWriter,
		HTTPDownloader: httpDownloader,
	}

	// Create admin service
	adminService := product.NewAdminService(repo, cfg, infrastructure)
	log.Printf("[DEBUG] Product Loader: Admin service created")

	// Create service wrapper for lifecycle management
	loaderService := &ProductLoaderService{
		BaseService: service.NewBaseService("product-loader"),
		service:     adminService,
		csvPath:     cliConfig.CSVPath,
		batchID:     cliConfig.BatchID,
		useCache:    cliConfig.UseCache,
		resetCache:  cliConfig.ResetCache,
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the ingestion process
	log.Printf("[INFO] Product Loader: Starting product ingestion process...")

	ingestionCtx, ingestionCancel := context.WithCancel(ctx)
	defer ingestionCancel()

	// Run ingestion in a goroutine
	done := make(chan error, 1)
	go func() {
		result, err := loaderService.RunIngestion(ingestionCtx)
		if err != nil {
			done <- err
			return
		}

		// Log results
		log.Printf("[INFO] Product Loader: Ingestion completed successfully!")
		log.Printf("[INFO] Product Loader: Batch ID: %s", result.BatchID)
		log.Printf("[INFO] Product Loader: Total Products: %d", result.TotalProducts)
		log.Printf("[INFO] Product Loader: Processed Products: %d", result.ProcessedProducts)
		log.Printf("[INFO] Product Loader: Total Images: %d", result.TotalImages)
		log.Printf("[INFO] Product Loader: Successful Images: %d", result.SuccessfulImages)
		log.Printf("[INFO] Product Loader: Failed Products: %d", result.FailedProducts)
		log.Printf("[INFO] Product Loader: Failed Images: %d", result.FailedImages)
		log.Printf("[INFO] Product Loader: Duration: %s", result.Duration)

		if len(result.Errors) > 0 {
			log.Printf("[WARN] Product Loader: %d errors occurred during ingestion:", len(result.Errors))
			for i, errMsg := range result.Errors {
				log.Printf("[WARN] Product Loader: Error %d: %s", i+1, errMsg)
			}
		}

		done <- nil
	}()

	// Wait for completion or shutdown signal
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("ingestion failed: %w", err)
		}
		log.Printf("[INFO] Product Loader: Product loader completed successfully")
		return nil
	case sig := <-sigChan:
		log.Printf("[INFO] Product Loader: Received signal %v, initiating graceful shutdown...", sig)
		ingestionCancel()

		// Wait for ingestion to finish or timeout
		select {
		case <-done:
			log.Printf("[INFO] Product Loader: Ingestion completed during shutdown")
		case <-time.After(30 * time.Second):
			log.Printf("[WARN] Product Loader: Ingestion did not complete within timeout")
		}

		log.Printf("[INFO] Product Loader: Shutdown complete")
		return nil
	}
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ERROR] Panic recovered in product-loader: %v", r)
			os.Exit(1)
		}
	}()
	log.SetFlags(log.LstdFlags)
	log.Printf("[INFO] Product Loader: Starting product loader service...")

	// Parse command line arguments
	cliConfig, err := parseFlags()
	if err != nil {
		log.Printf("[ERROR] Product Loader: %v", err)
		flag.Usage()
		os.Exit(1)
	}

	// Run the product loader
	ctx := context.Background()
	if err := runProductLoader(ctx, cliConfig); err != nil {
		log.Printf("[ERROR] Product Loader: %v", err)
		os.Exit(1)
	}
}

// ProductLoaderService wraps the admin service for lifecycle management
type ProductLoaderService struct {
	*service.BaseService
	service    *product.AdminService
	csvPath    string
	batchID    string
	useCache   bool
	resetCache bool
}

// productConfigFromLoaderConfig converts loader config to product config
func productConfigFromLoaderConfig(loaderCfg *Config) *product.Config {
	return &product.Config{
		DatabaseURL:  loaderCfg.DatabaseURL,
		ServicePort:  "", // not used by loader
		CacheDir:     loaderCfg.CacheDir,
		CacheMaxAge:  loaderCfg.CacheMaxAge,
		CacheMaxSize: loaderCfg.CacheMaxSize,
		CSVBatchSize: loaderCfg.CSVBatchSize,
		MinIOBucket:  loaderCfg.MinIOBucket, // Use bucket from loader config
	}
}

// RunIngestion executes the product ingestion workflow
func (s *ProductLoaderService) RunIngestion(ctx context.Context) (*product.ProductIngestionResult, error) {
	log.Printf("[INFO] Product Loader: Starting ingestion for CSV: %s", s.csvPath)

	req := &product.ProductIngestionRequest{
		CSVPath:    s.csvPath,
		BatchID:    s.batchID,
		UseCache:   s.useCache,
		ResetCache: s.resetCache,
	}

	result, err := s.service.IngestProductsFromCSV(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ingestion failed: %w", err)
	}

	return result, nil
}

// getBatchID returns the provided batch ID or generates a new one
func getBatchID(batchID string) string {
	if batchID != "" {
		return batchID
	}
	return fmt.Sprintf("batch-%d", time.Now().Unix())
}
