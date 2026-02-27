package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-shopping-poc/internal/platform/config"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/downloader"
	"go-shopping-poc/internal/platform/logging"
	"go-shopping-poc/internal/platform/outbox/providers"
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
func runProductLoader(ctx context.Context, cliConfig *CLIConfig, logger *slog.Logger) error {
	logger.Info("Processing CSV file", "csv_path", cliConfig.CSVPath)
	logger.Info("Batch ID", "batch_id", cliConfig.BatchID)
	logger.Info("Use cache", "use_cache", cliConfig.UseCache)
	logger.Info("Reset cache", "reset_cache", cliConfig.ResetCache)

	// Load loader-specific configuration
	loaderCfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	logger.Debug("Configuration loaded successfully")

	// If CSV path not provided via CLI, use from configuration
	if cliConfig.CSVPath == "" {
		cliConfig.CSVPath = loaderCfg.CSVPath
		logger.Info("Using default CSV path from config", "csv_path", cliConfig.CSVPath)
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

	logger.Debug("Database connection config loaded")

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
			logger.Error("Error closing database connection", "error", err.Error())
		}
	}()

	logger.Debug("Database connection established")

	// Initialize HTTP downloader
	httpDownloader, err := downloader.NewHTTPDownloader(cfg.CacheDir)
	if err != nil {
		return fmt.Errorf("failed to create HTTP downloader: %w", err)
	}

	logger.Debug("HTTP downloader initialized")

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

	logger.Debug("MinIO storage initialized")

	// Initialize outbox writer
	writerProvider := providers.NewWriterProvider(platformDB)
	outboxWriter := writerProvider.GetWriter()
	logger.Debug("Outbox writer initialized")

	// Create admin infrastructure
	infrastructure := &product.AdminInfrastructure{
		Database:       platformDB,
		ObjectStorage:  minioStorage,
		OutboxWriter:   outboxWriter,
		HTTPDownloader: httpDownloader,
	}

	// Create admin service
	adminService := product.NewAdminService(logger, cfg, infrastructure)
	logger.Debug("Admin service created")

	// Create service wrapper for lifecycle management
	loaderService := &ProductLoaderService{
		BaseService: service.NewBaseService("product-loader"),
		service:     adminService,
		csvPath:     cliConfig.CSVPath,
		batchID:     cliConfig.BatchID,
		useCache:    cliConfig.UseCache,
		resetCache:  cliConfig.ResetCache,
		logger:      logger,
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the ingestion process
	logger.Info("Starting product ingestion process...")

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
		logger.Info("Ingestion completed successfully!")
		logger.Info("Batch ID", "batch_id", result.BatchID)
		logger.Info("Total Products", "count", result.TotalProducts)
		logger.Info("Processed Products", "count", result.ProcessedProducts)
		logger.Info("Total Images", "count", result.TotalImages)
		logger.Info("Successful Images", "count", result.SuccessfulImages)
		logger.Info("Failed Products", "count", result.FailedProducts)
		logger.Info("Failed Images", "count", result.FailedImages)
		logger.Info("Duration", "duration", result.Duration)

		if len(result.Errors) > 0 {
			logger.Warn("Errors occurred during ingestion", "count", len(result.Errors))
			for i, errMsg := range result.Errors {
				logger.Warn("Error", "index", i+1, "message", errMsg)
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
		logger.Info("Product loader completed successfully")
		return nil
	case sig := <-sigChan:
		logger.Info("Received signal, initiating graceful shutdown", "signal", sig)
		ingestionCancel()

		// Wait for ingestion to finish or timeout
		select {
		case <-done:
			logger.Info("Ingestion completed during shutdown")
		case <-time.After(30 * time.Second):
			logger.Warn("Ingestion did not complete within timeout")
		}

		logger.Info("Shutdown complete")
		return nil
	}
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			slog.Default().Error("Panic recovered in product-loader", "panic", r)
			os.Exit(1)
		}
	}()

	loggerProvider, err := logging.NewLoggerProvider(logging.DefaultLoggerConfig("product-loader"))
	if err != nil {
		slog.Default().Error("Failed to create logger provider", "error", err.Error())
		os.Exit(1)
	}
	logger := loggerProvider.Logger()

	logger.Info("Starting product loader service...")

	// Parse command line arguments
	cliConfig, err := parseFlags()
	if err != nil {
		logger.Error("Failed to parse flags", "error", err.Error())
		flag.Usage()
		os.Exit(1)
	}

	// Run the product loader
	ctx := context.Background()
	if err := runProductLoader(ctx, cliConfig, logger); err != nil {
		logger.Error("Product loader failed", "error", err.Error())
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
	logger     *slog.Logger
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
	s.logger.Info("Starting ingestion", "csv_path", s.csvPath)

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
