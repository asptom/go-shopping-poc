// Package product provides business logic for product management operations.
//
// This package implements the service layer for product domain operations including
// CRUD operations, validation, product ingestion workflow, and business rule enforcement.
// It acts as an intermediary between HTTP handlers and the data repository layer.
package product

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/csv"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/downloader"
	"go-shopping-poc/internal/platform/service"
	"go-shopping-poc/internal/platform/storage/minio"

	"github.com/google/uuid"
)

// ProductIngestionRequest represents a request to ingest products from CSV
type ProductIngestionRequest struct {
	CSVPath    string `json:"csv_path" validate:"required"`
	BatchID    string `json:"batch_id,omitempty"`
	UseCache   bool   `json:"use_cache"`
	ResetCache bool   `json:"reset_cache"`
}

// ProductIngestionResult represents the result of a product ingestion operation
type ProductIngestionResult struct {
	BatchID           string    `json:"batch_id"`
	TotalProducts     int       `json:"total_products"`
	ProcessedProducts int       `json:"processed_products"`
	TotalImages       int       `json:"total_images"`
	SuccessfulImages  int       `json:"successful_images"`
	FailedProducts    int       `json:"failed_products"`
	FailedImages      int       `json:"failed_images"`
	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
	Duration          string    `json:"duration"`
	Errors            []string  `json:"errors,omitempty"`
}

// ProductService orchestrates product business operations.
//
// ProductService acts as the service layer, coordinating between
// the HTTP handlers and the repository layer. It contains business
// logic, validation, and data transformation for product operations.
type ProductService struct {
	*service.BaseService
	repo           ProductRepository
	config         *Config
	infrastructure *ProductInfrastructure
}

// NewProductService creates a new product service instance.
func NewProductService(repo ProductRepository, config *Config, infrastructure *ProductInfrastructure) *ProductService {
	return &ProductService{
		BaseService:    service.NewBaseService("product"),
		repo:           repo,
		config:         config,
		infrastructure: infrastructure,
	}
}

// GetProductByID retrieves a product by its ID
func (s *ProductService) GetProductByID(ctx context.Context, productID int64) (*Product, error) {
	log.Printf("[INFO] ProductService: Fetching product by ID: %d", productID)

	product, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		log.Printf("[ERROR] ProductService: Failed to get product %d: %v", productID, err)
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	return product, nil
}

// CreateProduct creates a new product with validation and event publishing
func (s *ProductService) CreateProduct(ctx context.Context, product *Product) error {
	log.Printf("[INFO] ProductService: Creating new product")

	// Validate the product entity
	if err := product.Validate(); err != nil {
		return fmt.Errorf("product validation failed: %w", err)
	}

	// Generate ID if not provided
	if product.ID == 0 {
		// Use current timestamp as ID for simplicity (in production, use proper ID generation)
		product.ID = time.Now().Unix()
	}

	// Set timestamps
	now := time.Now()
	product.CreatedAt = now
	product.UpdatedAt = now

	// Insert product
	if err := s.repo.InsertProduct(ctx, product); err != nil {
		log.Printf("[ERROR] ProductService: Failed to insert product: %v", err)
		return fmt.Errorf("failed to insert product: %w", err)
	}

	// Publish product created event
	if err := s.publishProductCreatedEvent(ctx, product); err != nil {
		log.Printf("[WARN] ProductService: Failed to publish product created event: %v", err)
		// Don't fail the operation for event publishing errors
	}

	log.Printf("[INFO] ProductService: Successfully created product %d", product.ID)
	return nil
}

// UpdateProduct updates an existing product
func (s *ProductService) UpdateProduct(ctx context.Context, product *Product) error {
	log.Printf("[INFO] ProductService: Updating product %d", product.ID)

	// Validate the product entity
	if err := product.Validate(); err != nil {
		return fmt.Errorf("product validation failed: %w", err)
	}

	if product.ID == 0 {
		return fmt.Errorf("product ID is required for update")
	}

	// Update timestamp
	product.UpdatedAt = time.Now()

	// Update product
	if err := s.repo.UpdateProduct(ctx, product); err != nil {
		log.Printf("[ERROR] ProductService: Failed to update product %d: %v", product.ID, err)
		return fmt.Errorf("failed to update product: %w", err)
	}

	// Publish product updated event
	if err := s.publishProductUpdatedEvent(ctx, product); err != nil {
		log.Printf("[WARN] ProductService: Failed to publish product updated event: %v", err)
	}

	log.Printf("[INFO] ProductService: Successfully updated product %d", product.ID)
	return nil
}

// DeleteProduct removes a product and publishes deletion event
func (s *ProductService) DeleteProduct(ctx context.Context, productID int64) error {
	log.Printf("[INFO] ProductService: Deleting product %d", productID)

	if productID == 0 {
		return fmt.Errorf("product ID is required for deletion")
	}

	// Get product before deletion for event
	product, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		log.Printf("[ERROR] ProductService: Failed to get product %d for deletion: %v", productID, err)
		return fmt.Errorf("failed to get product for deletion: %w", err)
	}

	// Delete product
	if err := s.repo.DeleteProduct(ctx, productID); err != nil {
		log.Printf("[ERROR] ProductService: Failed to delete product %d: %v", productID, err)
		return fmt.Errorf("failed to delete product: %w", err)
	}

	// Publish product deleted event
	if err := s.publishProductDeletedEvent(ctx, product); err != nil {
		log.Printf("[WARN] ProductService: Failed to publish product deleted event: %v", err)
	}

	log.Printf("[INFO] ProductService: Successfully deleted product %d", productID)
	return nil
}

// IngestProductsFromCSV orchestrates the complete product ingestion workflow
func (s *ProductService) IngestProductsFromCSV(ctx context.Context, req *ProductIngestionRequest) (*ProductIngestionResult, error) {
	log.Printf("[INFO] ProductService: Starting product ingestion from CSV: %s", req.CSVPath)

	startTime := time.Now()
	batchID := req.BatchID
	if batchID == "" {
		batchID = uuid.New().String()
	}

	result := &ProductIngestionResult{
		BatchID:   batchID,
		StartTime: startTime,
		Errors:    make([]string, 0),
	}

	// Publish ingestion started event
	if err := s.publishIngestionStartedEvent(ctx, batchID, req); err != nil {
		log.Printf("[WARN] ProductService: Failed to publish ingestion started event: %v", err)
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime).String()

		// Publish completion or failure event
		if len(result.Errors) > 0 {
			if err := s.publishIngestionFailedEvent(ctx, batchID, result); err != nil {
				log.Printf("[WARN] ProductService: Failed to publish ingestion failed event: %v", err)
			}
		} else {
			if err := s.publishIngestionCompletedEvent(ctx, batchID, result); err != nil {
				log.Printf("[WARN] ProductService: Failed to publish ingestion completed event: %v", err)
			}
		}
	}()

	// Validate CSV file exists and is readable
	parser := csv.NewParser[ProductCSVRecord](req.CSVPath)
	if err := parser.ValidateHeaders(); err != nil {
		errMsg := fmt.Sprintf("CSV validation failed: %v", err)
		result.Errors = append(result.Errors, errMsg)
		return result, fmt.Errorf("CSV validation failed: %w", err)
	}

	// Parse CSV records
	records, err := parser.Parse()
	if err != nil {
		errMsg := fmt.Sprintf("Failed to parse CSV: %v", err)
		result.Errors = append(result.Errors, errMsg)
		return result, fmt.Errorf("failed to parse CSV: %w", err)
	}

	result.TotalProducts = len(records)
	log.Printf("[INFO] ProductService: Parsed %d products from CSV", result.TotalProducts)

	// Reset cache if requested
	if req.ResetCache {
		if err := s.infrastructure.HTTPDownloader.ClearCache(); err != nil {
			log.Printf("[WARN] ProductService: Failed to clear cache: %v", err)
		} else {
			log.Printf("[INFO] ProductService: Cache cleared successfully")
		}
	}

	// Set cache policy
	if s.infrastructure.HTTPDownloader != nil {
		cachePolicy := downloader.CachePolicy{
			MaxAge:  s.config.CacheMaxAge,
			MaxSize: s.config.CacheMaxSize,
		}
		s.infrastructure.HTTPDownloader.SetCachePolicy(cachePolicy)
	}

	// Process products in batches
	batchSize := s.config.CSVBatchSize
	if batchSize <= 0 {
		batchSize = 100
	}

	for i := 0; i < len(records); i += batchSize {
		end := i + batchSize
		if end > len(records) {
			end = len(records)
		}

		batch := records[i:end]
		log.Printf("[INFO] ProductService: Processing batch %d-%d of %d products", i+1, end, len(records))

		if err := s.processProductBatch(ctx, batch, req.UseCache, result); err != nil {
			errMsg := fmt.Sprintf("Failed to process batch %d-%d: %v", i+1, end, err)
			result.Errors = append(result.Errors, errMsg)
			log.Printf("[ERROR] ProductService: %s", errMsg)
			// Continue with next batch instead of failing completely
		}
	}

	log.Printf("[INFO] ProductService: Ingestion completed. Processed: %d/%d products, %d/%d images",
		result.ProcessedProducts, result.TotalProducts, result.SuccessfulImages, result.TotalImages)

	return result, nil
}

// processProductBatch processes a batch of product records
func (s *ProductService) processProductBatch(ctx context.Context, records []ProductCSVRecord, useCache bool, result *ProductIngestionResult) error {
	// Convert CSV records to products
	products := make([]*Product, 0, len(records))

	for _, record := range records {
		product, err := s.convertCSVRecordToProduct(record)
		if err != nil {
			log.Printf("[WARN] ProductService: Failed to convert CSV record: %v", err)
			result.FailedProducts++
			continue
		}
		products = append(products, product)
	}

	// Bulk insert products
	if len(products) > 0 {
		if s.repo != nil {
			err := s.repo.BulkInsertProducts(ctx, products)
			if err != nil {
				return fmt.Errorf("failed to bulk insert products: %w", err)
			}
		}

		result.ProcessedProducts += len(products)

		// Process images for each product
		for _, product := range products {
			if err := s.processProductImages(ctx, product, useCache, result); err != nil {
				log.Printf("[WARN] ProductService: Failed to process images for product %d: %v", product.ID, err)
				// Continue with other products
			}
		}
	}

	return nil
}

// processProductImages downloads and uploads images for a product
func (s *ProductService) processProductImages(ctx context.Context, product *Product, useCache bool, result *ProductIngestionResult) error {
	if len(product.ImageURLs) == 0 {
		return nil // No images to process
	}

	// Filter to image URLs only
	imageURLs := s.filterImageURLs(product.ImageURLs)
	result.TotalImages += len(imageURLs)

	log.Printf("[DEBUG] ProductService: Processing %d images for product %d", len(imageURLs), product.ID)

	for i, imageURL := range imageURLs {
		if err := s.processSingleImage(ctx, product, imageURL, i, useCache, result); err != nil {
			log.Printf("[WARN] ProductService: Failed to process image %d for product %d: %v", i, product.ID, err)
			result.FailedImages++
			continue
		}
		result.SuccessfulImages++
	}

	return nil
}

// processSingleImage downloads, uploads, and stores metadata for a single image
func (s *ProductService) processSingleImage(ctx context.Context, product *Product, imageURL string, imageIndex int, useCache bool, result *ProductIngestionResult) error {
	var localPath string
	var err error

	// Download or use cached image
	if s.infrastructure.HTTPDownloader != nil {
		if useCache && s.infrastructure.HTTPDownloader.IsCached(imageURL) {
			localPath = s.infrastructure.HTTPDownloader.GetCachePath(imageURL)
			log.Printf("[DEBUG] ProductService: Using cached image for product %d, index %d", product.ID, imageIndex)
		} else {
			localPath, err = s.infrastructure.HTTPDownloader.Download(ctx, imageURL)
			if err != nil {
				return fmt.Errorf("failed to download image: %w", err)
			}
			log.Printf("[DEBUG] ProductService: Downloaded image for product %d, index %d", product.ID, imageIndex)
		}
	} else {
		// Skip download in test environment
		localPath = "/tmp/test.jpg"
	}

	// Upload to MinIO
	minioObjectName, err := s.uploadImageToMinIO(ctx, localPath, product.ID, imageIndex)
	if err != nil {
		return fmt.Errorf("failed to upload image to MinIO: %w", err)
	}

	// Get image metadata
	fileSize, contentType, err := s.getImageInfo(localPath)
	if err != nil {
		return fmt.Errorf("failed to get image info: %w", err)
	}

	// Create product image record
	productImage := &ProductImage{
		ProductID:       product.ID,
		ImageURL:        imageURL,
		MinioObjectName: minioObjectName,
		IsMain:          imageURL == product.MainImage,
		ImageOrder:      imageIndex,
		FileSize:        fileSize,
		ContentType:     contentType,
	}

	// Insert image metadata
	if s.repo != nil {
		if err := s.repo.AddProductImage(ctx, productImage); err != nil {
			return fmt.Errorf("failed to insert product image: %w", err)
		}
	}

	// Publish image added event
	if err := s.publishProductImageAddedEvent(ctx, product.ID, productImage.ID, imageURL); err != nil {
		log.Printf("[WARN] ProductService: Failed to publish image added event: %v", err)
	}

	log.Printf("[DEBUG] ProductService: Successfully processed image %d for product %d", imageIndex, product.ID)
	return nil
}

// Helper methods

func (s *ProductService) convertCSVRecordToProduct(record ProductCSVRecord) (*Product, error) {
	product := &Product{
		Name:              record.Name,
		Description:       record.Description,
		InitialPrice:      record.InitialPrice,
		FinalPrice:        record.FinalPrice,
		Currency:          record.Currency,
		Color:             record.Color,
		Size:              record.Size,
		MainImage:         record.MainImage,
		CountryCode:       record.CountryCode,
		ModelNumber:       record.ModelNumber,
		RootCategory:      record.RootCategory,
		Category:          record.Category,
		Brand:             record.Brand,
		AllAvailableSizes: record.AllAvailableSizes,
		ImageURLs:         record.ImageURLs,
	}

	// Parse ID
	if record.ID != "" {
		if id, err := strconv.ParseInt(record.ID, 10, 64); err == nil {
			product.ID = id
		} else {
			return nil, fmt.Errorf("invalid product ID '%s': %w", record.ID, err)
		}
	}

	// Set default values
	if product.Currency == "" {
		product.Currency = "USD"
	}
	if product.CountryCode == "" {
		product.CountryCode = "US"
	}

	// Parse boolean fields
	if record.InStock != "" {
		product.InStock = strings.ToLower(record.InStock) == "true" || record.InStock == "1"
	} else {
		product.InStock = true // Default to in stock
	}

	// Parse image count
	if record.ImageCount != "" {
		if count, err := strconv.Atoi(record.ImageCount); err == nil {
			product.ImageCount = count
		}
	}

	// Parse other attributes JSON
	if record.OtherAttributes != "" && record.OtherAttributes != "[]" {
		var otherAttrs []map[string]any
		if err := json.Unmarshal([]byte(record.OtherAttributes), &otherAttrs); err != nil {
			log.Printf("[WARN] ProductService: Failed to parse other_attributes JSON: %v", err)
			// Continue without other attributes
		} else {
			// Marshal back to JSON string for storage
			if jsonBytes, err := json.Marshal(otherAttrs); err != nil {
				log.Printf("[WARN] ProductService: Failed to marshal other_attributes JSON: %v", err)
			} else {
				product.OtherAttributes = string(jsonBytes)
			}
		}
	}

	return product, nil
}

func (s *ProductService) filterImageURLs(urls []string) []string {
	var filtered []string
	imageExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true,
	}

	for _, url := range urls {
		lowerURL := strings.ToLower(url)
		for ext := range imageExts {
			if strings.HasSuffix(lowerURL, ext) {
				filtered = append(filtered, url)
				break
			}
		}
	}
	return filtered
}

func (s *ProductService) uploadImageToMinIO(ctx context.Context, localPath string, productID int64, imageIndex int) (string, error) {
	// Ensure bucket exists before uploading
	if err := s.ensureBucketExists(ctx, s.config.MinIOBucket); err != nil {
		return "", fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	// Generate object name
	objectName := fmt.Sprintf("products/%d/image_%d%s", productID, imageIndex, filepath.Ext(localPath))

	// Upload file
	if s.infrastructure.ObjectStorage != nil {
		info, err := s.infrastructure.ObjectStorage.FPutObject(ctx, s.config.MinIOBucket, objectName, localPath, minio.PutObjectOptions{
			ContentType: s.getContentTypeFromPath(localPath),
		})
		if err != nil {
			return "", fmt.Errorf("failed to upload to MinIO: %w", err)
		}

		log.Printf("[DEBUG] ProductService: Uploaded image to MinIO: %s (size: %d)", objectName, info.Size)
	}

	return objectName, nil
}

// ensureBucketExists checks if the bucket exists and creates it if it doesn't
func (s *ProductService) ensureBucketExists(ctx context.Context, bucketName string) error {
	if s.infrastructure.ObjectStorage == nil {
		return fmt.Errorf("storage client is not available")
	}

	// Check if bucket exists
	exists, err := s.infrastructure.ObjectStorage.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		// Create the bucket
		if err := s.infrastructure.ObjectStorage.CreateBucket(ctx, bucketName); err != nil {
			return fmt.Errorf("failed to create bucket %s: %w", bucketName, err)
		}
		log.Printf("[INFO] ProductService: Created MinIO bucket: %s", bucketName)
	} else {
		log.Printf("[DEBUG] ProductService: MinIO bucket already exists: %s", bucketName)
	}

	return nil
}

func (s *ProductService) getImageInfo(localPath string) (int64, string, error) {
	// Use os.Stat instead of filepath.Stat
	stat, err := os.Stat(localPath)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get file info: %w", err)
	}

	return stat.Size(), s.getContentTypeFromPath(localPath), nil
}

func (s *ProductService) getContentTypeFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/jpeg" // Default
	}
}

// Event publishing methods

func (s *ProductService) publishProductCreatedEvent(ctx context.Context, product *Product) error {
	event := events.NewProductCreatedEvent(fmt.Sprintf("%d", product.ID), map[string]string{
		"name":  product.Name,
		"brand": product.Brand,
		"price": product.FormattedPrice(),
	})
	return s.publishEvent(ctx, event)
}

func (s *ProductService) publishProductUpdatedEvent(ctx context.Context, product *Product) error {
	event := events.NewProductUpdatedEvent(fmt.Sprintf("%d", product.ID), map[string]string{
		"name":  product.Name,
		"brand": product.Brand,
	})
	return s.publishEvent(ctx, event)
}

func (s *ProductService) publishProductDeletedEvent(ctx context.Context, product *Product) error {
	event := events.NewProductDeletedEvent(fmt.Sprintf("%d", product.ID), map[string]string{
		"name":  product.Name,
		"brand": product.Brand,
	})
	return s.publishEvent(ctx, event)
}

func (s *ProductService) publishProductImageAddedEvent(ctx context.Context, productID int64, imageID int64, imageURL string) error {
	event := events.NewProductImageAddedEvent(fmt.Sprintf("%d", productID), fmt.Sprintf("%d", imageID), map[string]string{
		"image_url": imageURL,
	})
	return s.publishEvent(ctx, event)
}

func (s *ProductService) publishIngestionStartedEvent(ctx context.Context, batchID string, req *ProductIngestionRequest) error {
	event := events.NewProductIngestionStartedEvent(batchID, map[string]string{
		"csv_path":    req.CSVPath,
		"use_cache":   fmt.Sprintf("%t", req.UseCache),
		"reset_cache": fmt.Sprintf("%t", req.ResetCache),
	})
	return s.publishEvent(ctx, event)
}

func (s *ProductService) publishIngestionCompletedEvent(ctx context.Context, batchID string, result *ProductIngestionResult) error {
	event := events.NewProductIngestionCompletedEvent(batchID, map[string]string{
		"total_products":     fmt.Sprintf("%d", result.TotalProducts),
		"processed_products": fmt.Sprintf("%d", result.ProcessedProducts),
		"total_images":       fmt.Sprintf("%d", result.TotalImages),
		"successful_images":  fmt.Sprintf("%d", result.SuccessfulImages),
		"duration":           result.Duration,
	})
	return s.publishEvent(ctx, event)
}

func (s *ProductService) publishIngestionFailedEvent(ctx context.Context, batchID string, result *ProductIngestionResult) error {
	event := events.NewProductIngestionFailedEvent(batchID, map[string]string{
		"total_products":     fmt.Sprintf("%d", result.TotalProducts),
		"processed_products": fmt.Sprintf("%d", result.ProcessedProducts),
		"errors_count":       fmt.Sprintf("%d", len(result.Errors)),
		"duration":           result.Duration,
	})
	return s.publishEvent(ctx, event)
}

// GetProductsByCategory retrieves products by category with pagination
func (s *ProductService) GetProductsByCategory(ctx context.Context, category string, limit, offset int) ([]*Product, error) {
	log.Printf("[INFO] ProductService: Fetching products by category: %s (limit: %d, offset: %d)", category, limit, offset)

	products, err := s.repo.GetProductsByCategory(ctx, category, limit, offset)
	if err != nil {
		log.Printf("[ERROR] ProductService: Failed to get products by category %s: %v", category, err)
		return nil, fmt.Errorf("failed to get products by category: %w", err)
	}

	return products, nil
}

// GetProductsByBrand retrieves products by brand with pagination
func (s *ProductService) GetProductsByBrand(ctx context.Context, brand string, limit, offset int) ([]*Product, error) {
	log.Printf("[INFO] ProductService: Fetching products by brand: %s (limit: %d, offset: %d)", brand, limit, offset)

	products, err := s.repo.GetProductsByBrand(ctx, brand, limit, offset)
	if err != nil {
		log.Printf("[ERROR] ProductService: Failed to get products by brand %s: %v", brand, err)
		return nil, fmt.Errorf("failed to get products by brand: %w", err)
	}

	return products, nil
}

// SearchProducts searches products by query with pagination
func (s *ProductService) SearchProducts(ctx context.Context, query string, limit, offset int) ([]*Product, error) {
	log.Printf("[INFO] ProductService: Searching products with query: %s (limit: %d, offset: %d)", query, limit, offset)

	products, err := s.repo.SearchProducts(ctx, query, limit, offset)
	if err != nil {
		log.Printf("[ERROR] ProductService: Failed to search products with query %s: %v", query, err)
		return nil, fmt.Errorf("failed to search products: %w", err)
	}

	return products, nil
}

// GetProductsInStock retrieves products that are in stock with pagination
func (s *ProductService) GetProductsInStock(ctx context.Context, limit, offset int) ([]*Product, error) {
	log.Printf("[INFO] ProductService: Fetching in-stock products (limit: %d, offset: %d)", limit, offset)

	products, err := s.repo.GetProductsInStock(ctx, limit, offset)
	if err != nil {
		log.Printf("[ERROR] ProductService: Failed to get in-stock products: %v", err)
		return nil, fmt.Errorf("failed to get in-stock products: %w", err)
	}

	return products, nil
}

func (s *ProductService) publishEvent(ctx context.Context, event events.Event) error {
	// Skip event publishing if database is not available (e.g., in tests)
	if s.infrastructure.Database == nil {
		log.Printf("[DEBUG] ProductService: Skipping event publishing - database not available")
		return nil
	}

	// Create a transaction for the outbox event publishing
	tx, err := s.infrastructure.Database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for event publishing: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// Write the event to the outbox within the transaction
	if err := s.infrastructure.OutboxWriter.WriteEvent(ctx, tx, event); err != nil {
		return fmt.Errorf("failed to write event to outbox: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit outbox transaction: %w", err)
	}
	committed = true

	return nil
}

// ProductCSVRecord represents a product record from CSV
type ProductCSVRecord struct {
	ID                string        `csv:"product_id"`
	Name              string        `csv:"product_name"`
	Description       string        `csv:"description"`
	InitialPrice      float64       `csv:"initial_price"`
	FinalPrice        float64       `csv:"final_price"`
	Currency          string        `csv:"currency"`
	InStock           string        `csv:"in_stock"`
	Color             string        `csv:"color"`
	Size              string        `csv:"size"`
	MainImage         string        `csv:"main_image"`
	CountryCode       string        `csv:"country_code"`
	ImageCount        string        `csv:"image_count"`
	ModelNumber       string        `csv:"model_number"`
	RootCategory      string        `csv:"root_category"`
	Category          string        `csv:"category"`
	Brand             string        `csv:"brand"`
	AllAvailableSizes database.JSON `csv:"all_available_sizes"`
	ImageURLs         []string      `csv:"image_urls"`
	OtherAttributes   string        `csv:"other_attributes"`
}
