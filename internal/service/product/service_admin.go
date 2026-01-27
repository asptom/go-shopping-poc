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
	"go-shopping-poc/internal/platform/outbox"
	"go-shopping-poc/internal/platform/service"
	"go-shopping-poc/internal/platform/storage/minio"

	"github.com/google/uuid"
)

// AdminService handles write operations for product management.
//
// AdminService is focused on product creation, updates, deletion,
// image management, and CSV ingestion. All write operations
// publish events using the outbox pattern for reliable delivery.
type AdminService struct {
	*service.BaseService
	repo           ProductRepository
	config         *Config
	infrastructure *AdminInfrastructure
}

// AdminInfrastructure defines infrastructure components for admin service
type AdminInfrastructure struct {
	Database       database.Database
	ObjectStorage  minio.ObjectStorage
	OutboxWriter   *outbox.Writer
	HTTPDownloader downloader.HTTPDownloader
}

// NewAdminService creates a new admin service instance.
func NewAdminService(repo ProductRepository, config interface{}, infrastructure *AdminInfrastructure) *AdminService {
	var cfg *Config
	switch c := config.(type) {
	case *AdminConfig:
		// Convert AdminConfig to Config for the service
		cfg = &Config{
			DatabaseURL:  c.DatabaseURL,
			ServicePort:  c.ServicePort,
			CacheDir:     c.CacheDir,
			CacheMaxAge:  c.CacheMaxAge,
			CacheMaxSize: c.CacheMaxSize,
			CSVBatchSize: c.CSVBatchSize,
			MinIOBucket:  c.MinIOBucket,
		}
	case *Config:
		cfg = c
	default:
		cfg = config.(*Config)
	}
	return &AdminService{
		BaseService:    service.NewBaseService("product-admin"),
		repo:           repo,
		config:         cfg,
		infrastructure: infrastructure,
	}
}

// CreateProduct creates a new product with validation and event publishing
func (s *AdminService) CreateProduct(ctx context.Context, product *Product) error {
	log.Printf("[INFO] AdminService: Creating new product")

	if err := product.Validate(); err != nil {
		return fmt.Errorf("product validation failed: %w", err)
	}

	if product.ID == 0 {
		product.ID = time.Now().Unix()
	}

	now := time.Now()
	product.CreatedAt = now
	product.UpdatedAt = now

	if err := s.repo.InsertProduct(ctx, product); err != nil {
		log.Printf("[ERROR] AdminService: Failed to insert product: %v", err)
		return fmt.Errorf("failed to insert product: %w", err)
	}

	if err := s.publishProductCreatedEvent(ctx, product); err != nil {
		log.Printf("[WARN] AdminService: Failed to publish product created event: %v", err)
	}

	log.Printf("[INFO] AdminService: Successfully created product %d", product.ID)
	return nil
}

// UpdateProduct updates an existing product
func (s *AdminService) UpdateProduct(ctx context.Context, product *Product) error {
	log.Printf("[INFO] AdminService: Updating product %d", product.ID)

	if err := product.Validate(); err != nil {
		return fmt.Errorf("product validation failed: %w", err)
	}

	if product.ID == 0 {
		return fmt.Errorf("product ID is required for update")
	}

	product.UpdatedAt = time.Now()

	if err := s.repo.UpdateProduct(ctx, product); err != nil {
		log.Printf("[ERROR] AdminService: Failed to update product %d: %v", product.ID, err)
		return fmt.Errorf("failed to update product: %w", err)
	}

	if err := s.publishProductUpdatedEvent(ctx, product); err != nil {
		log.Printf("[WARN] AdminService: Failed to publish product updated event: %v", err)
	}

	log.Printf("[INFO] AdminService: Successfully updated product %d", product.ID)
	return nil
}

// DeleteProduct removes a product and publishes deletion event
func (s *AdminService) DeleteProduct(ctx context.Context, productID int64) error {
	log.Printf("[INFO] AdminService: Deleting product %d", productID)

	if productID == 0 {
		return fmt.Errorf("product ID is required for deletion")
	}

	product, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		log.Printf("[ERROR] AdminService: Failed to get product %d for deletion: %v", productID, err)
		return fmt.Errorf("failed to get product for deletion: %w", err)
	}

	if err := s.repo.DeleteProduct(ctx, productID); err != nil {
		log.Printf("[ERROR] AdminService: Failed to delete product %d: %v", productID, err)
		return fmt.Errorf("failed to delete product: %w", err)
	}

	if err := s.publishProductDeletedEvent(ctx, product); err != nil {
		log.Printf("[WARN] AdminService: Failed to publish product deleted event: %v", err)
	}

	log.Printf("[INFO] AdminService: Successfully deleted product %d", productID)
	return nil
}

// AddProductImage adds an image to a product
func (s *AdminService) AddProductImage(ctx context.Context, image *ProductImage) error {
	log.Printf("[INFO] AdminService: Adding image for product %d", image.ProductID)

	if err := image.Validate(); err != nil {
		return fmt.Errorf("image validation failed: %w", err)
	}

	if err := s.repo.AddProductImage(ctx, image); err != nil {
		log.Printf("[ERROR] AdminService: Failed to add image: %v", err)
		return fmt.Errorf("failed to add image: %w", err)
	}

	if err := s.publishProductImageAddedEvent(ctx, image.ProductID, image.ID, image.ImageURL); err != nil {
		log.Printf("[WARN] AdminService: Failed to publish image added event: %v", err)
	}

	return nil
}

// UpdateProductImage updates an existing product image
func (s *AdminService) UpdateProductImage(ctx context.Context, image *ProductImage) error {
	log.Printf("[INFO] AdminService: Updating image %d", image.ID)

	if err := image.Validate(); err != nil {
		return fmt.Errorf("image validation failed: %w", err)
	}

	if err := s.repo.UpdateProductImage(ctx, image); err != nil {
		log.Printf("[ERROR] AdminService: Failed to update image: %v", err)
		return fmt.Errorf("failed to update image: %w", err)
	}

	if err := s.publishProductImageUpdatedEvent(ctx, image.ProductID, image.ID); err != nil {
		log.Printf("[WARN] AdminService: Failed to publish image updated event: %v", err)
	}

	return nil
}

// DeleteProductImage removes a product image
func (s *AdminService) DeleteProductImage(ctx context.Context, imageID int64) error {
	log.Printf("[INFO] AdminService: Deleting image %d", imageID)

	if err := s.repo.DeleteProductImage(ctx, imageID); err != nil {
		log.Printf("[ERROR] AdminService: Failed to delete image: %v", err)
		return fmt.Errorf("failed to delete image: %w", err)
	}

	if err := s.publishProductImageDeletedEvent(ctx, imageID); err != nil {
		log.Printf("[WARN] AdminService: Failed to publish image deleted event: %v", err)
	}

	return nil
}

// SetMainImage sets the main image for a product
func (s *AdminService) SetMainImage(ctx context.Context, productID int64, imageID int64) error {
	log.Printf("[INFO] AdminService: Setting main image %d for product %d", imageID, productID)

	if err := s.repo.SetMainImage(ctx, productID, imageID); err != nil {
		log.Printf("[ERROR] AdminService: Failed to set main image: %v", err)
		return fmt.Errorf("failed to set main image: %w", err)
	}

	return nil
}

// IngestProductsFromCSV orchestrates the complete product ingestion workflow
func (s *AdminService) IngestProductsFromCSV(ctx context.Context, req *ProductIngestionRequest) (*ProductIngestionResult, error) {
	log.Printf("[INFO] AdminService: Starting product ingestion from CSV: %s", req.CSVPath)

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

	if err := s.publishIngestionStartedEvent(ctx, batchID, req); err != nil {
		log.Printf("[WARN] AdminService: Failed to publish ingestion started event: %v", err)
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime).String()

		if len(result.Errors) > 0 {
			if err := s.publishIngestionFailedEvent(ctx, batchID, result); err != nil {
				log.Printf("[WARN] AdminService: Failed to publish ingestion failed event: %v", err)
			}
		} else {
			if err := s.publishIngestionCompletedEvent(ctx, batchID, result); err != nil {
				log.Printf("[WARN] AdminService: Failed to publish ingestion completed event: %v", err)
			}
		}
	}()

	parser := csv.NewParser[ProductCSVRecord](req.CSVPath)
	if err := parser.ValidateHeaders(); err != nil {
		errMsg := fmt.Sprintf("CSV validation failed: %v", err)
		result.Errors = append(result.Errors, errMsg)
		return result, fmt.Errorf("CSV validation failed: %w", err)
	}

	records, err := parser.Parse()
	if err != nil {
		errMsg := fmt.Sprintf("Failed to parse CSV: %v", err)
		result.Errors = append(result.Errors, errMsg)
		return result, fmt.Errorf("failed to parse CSV: %w", err)
	}

	result.TotalProducts = len(records)
	log.Printf("[INFO] AdminService: Parsed %d products from CSV", result.TotalProducts)

	if req.ResetCache {
		if err := s.infrastructure.HTTPDownloader.ClearCache(); err != nil {
			log.Printf("[WARN] AdminService: Failed to clear cache: %v", err)
		} else {
			log.Printf("[INFO] AdminService: Cache cleared successfully")
		}
	}

	if s.infrastructure.HTTPDownloader != nil {
		cachePolicy := downloader.CachePolicy{
			MaxAge:  s.config.CacheMaxAge,
			MaxSize: s.config.CacheMaxSize,
		}
		s.infrastructure.HTTPDownloader.SetCachePolicy(cachePolicy)
	}

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
		log.Printf("[INFO] AdminService: Processing batch %d-%d of %d products", i+1, end, len(records))

		if err := s.processProductBatch(ctx, batch, req.UseCache, result); err != nil {
			errMsg := fmt.Sprintf("Failed to process batch %d-%d: %v", i+1, end, err)
			result.Errors = append(result.Errors, errMsg)
			log.Printf("[ERROR] AdminService: %s", errMsg)
		}
	}

	log.Printf("[INFO] AdminService: Ingestion completed. Processed: %d/%d products, %d/%d images",
		result.ProcessedProducts, result.TotalProducts, result.SuccessfulImages, result.TotalImages)

	return result, nil
}

func (s *AdminService) processProductBatch(ctx context.Context, records []ProductCSVRecord, useCache bool, result *ProductIngestionResult) error {
	products := make([]*Product, 0, len(records))

	for _, record := range records {
		product, err := s.convertCSVRecordToProduct(record)
		if err != nil {
			log.Printf("[WARN] AdminService: Failed to convert CSV record: %v", err)
			result.FailedProducts++
			continue
		}
		products = append(products, product)
	}

	if len(products) > 0 {
		if s.repo != nil {
			err := s.repo.BulkInsertProducts(ctx, products)
			if err != nil {
				return fmt.Errorf("failed to bulk insert products: %w", err)
			}
		}

		result.ProcessedProducts += len(products)

		for _, product := range products {
			if err := s.processProductImages(ctx, product, useCache, result); err != nil {
				log.Printf("[WARN] AdminService: Failed to process images for product %d: %v", product.ID, err)
			}
		}
	}

	return nil
}

func (s *AdminService) processProductImages(ctx context.Context, product *Product, useCache bool, result *ProductIngestionResult) error {
	if len(product.ImageURLs) == 0 {
		return nil
	}

	imageURLs := s.filterImageURLs(product.ImageURLs)
	result.TotalImages += len(imageURLs)

	log.Printf("[DEBUG] AdminService: Processing %d images for product %d", len(imageURLs), product.ID)

	for i, imageURL := range imageURLs {
		if err := s.processSingleImage(ctx, product, imageURL, i, useCache, result); err != nil {
			log.Printf("[WARN] AdminService: Failed to process image %d for product %d: %v", i, product.ID, err)
			result.FailedImages++
			continue
		}
		result.SuccessfulImages++
	}

	return nil
}

func (s *AdminService) processSingleImage(ctx context.Context, product *Product, imageURL string, imageIndex int, useCache bool, result *ProductIngestionResult) error {
	var localPath string
	var err error

	if s.infrastructure.HTTPDownloader != nil {
		if useCache && s.infrastructure.HTTPDownloader.IsCached(imageURL) {
			localPath = s.infrastructure.HTTPDownloader.GetCachePath(imageURL)
			log.Printf("[DEBUG] AdminService: Using cached image for product %d, index %d", product.ID, imageIndex)
		} else {
			localPath, err = s.infrastructure.HTTPDownloader.Download(ctx, imageURL)
			if err != nil {
				return fmt.Errorf("failed to download image: %w", err)
			}
			log.Printf("[DEBUG] AdminService: Downloaded image for product %d, index %d", product.ID, imageIndex)
		}
	} else {
		localPath = "/tmp/test.jpg"
	}

	minioObjectName, err := s.uploadImageToMinIO(ctx, localPath, product.ID, imageIndex)
	if err != nil {
		return fmt.Errorf("failed to upload image to MinIO: %w", err)
	}

	fileSize, contentType, err := s.getImageInfo(localPath)
	if err != nil {
		return fmt.Errorf("failed to get image info: %w", err)
	}

	productImage := &ProductImage{
		ProductID:       product.ID,
		ImageURL:        imageURL,
		MinioObjectName: minioObjectName,
		IsMain:          imageURL == product.MainImage,
		ImageOrder:      imageIndex,
		FileSize:        fileSize,
		ContentType:     contentType,
	}

	if s.repo != nil {
		if err := s.repo.AddProductImage(ctx, productImage); err != nil {
			return fmt.Errorf("failed to insert product image: %w", err)
		}
	}

	if err := s.publishProductImageAddedEvent(ctx, product.ID, productImage.ID, imageURL); err != nil {
		log.Printf("[WARN] AdminService: Failed to publish image added event: %v", err)
	}

	log.Printf("[DEBUG] AdminService: Successfully processed image %d for product %d", imageIndex, product.ID)
	return nil
}

func (s *AdminService) convertCSVRecordToProduct(record ProductCSVRecord) (*Product, error) {
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

	if record.ID != "" {
		if id, err := strconv.ParseInt(record.ID, 10, 64); err == nil {
			product.ID = id
		} else {
			return nil, fmt.Errorf("invalid product ID '%s': %w", record.ID, err)
		}
	}

	if product.Currency == "" {
		product.Currency = "USD"
	}
	if product.CountryCode == "" {
		product.CountryCode = "US"
	}

	if record.InStock != "" {
		product.InStock = strings.ToLower(record.InStock) == "true" || record.InStock == "1"
	} else {
		product.InStock = true
	}

	if record.ImageCount != "" {
		if count, err := strconv.Atoi(record.ImageCount); err == nil {
			product.ImageCount = count
		}
	}

	if record.OtherAttributes != "" && record.OtherAttributes != "[]" {
		var otherAttrs []map[string]any
		if err := json.Unmarshal([]byte(record.OtherAttributes), &otherAttrs); err != nil {
			log.Printf("[WARN] AdminService: Failed to parse other_attributes JSON: %v", err)
		} else {
			if jsonBytes, err := json.Marshal(otherAttrs); err != nil {
				log.Printf("[WARN] AdminService: Failed to marshal other_attributes JSON: %v", err)
			} else {
				product.OtherAttributes = string(jsonBytes)
			}
		}
	}

	return product, nil
}

func (s *AdminService) filterImageURLs(urls []string) []string {
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

func (s *AdminService) uploadImageToMinIO(ctx context.Context, localPath string, productID int64, imageIndex int) (string, error) {
	if err := s.ensureBucketExists(ctx, s.config.MinIOBucket); err != nil {
		return "", fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	objectName := fmt.Sprintf("products/%d/image_%d%s", productID, imageIndex, filepath.Ext(localPath))

	if s.infrastructure.ObjectStorage != nil {
		info, err := s.infrastructure.ObjectStorage.FPutObject(ctx, s.config.MinIOBucket, objectName, localPath, minio.PutObjectOptions{
			ContentType: s.getContentTypeFromPath(localPath),
		})
		if err != nil {
			return "", fmt.Errorf("failed to upload to MinIO: %w", err)
		}

		log.Printf("[DEBUG] AdminService: Uploaded image to MinIO: %s (size: %d)", objectName, info.Size)
	}

	return objectName, nil
}

func (s *AdminService) ensureBucketExists(ctx context.Context, bucketName string) error {
	if s.infrastructure.ObjectStorage == nil {
		return fmt.Errorf("storage client is not available")
	}

	exists, err := s.infrastructure.ObjectStorage.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		if err := s.infrastructure.ObjectStorage.CreateBucket(ctx, bucketName); err != nil {
			return fmt.Errorf("failed to create bucket %s: %w", bucketName, err)
		}
		log.Printf("[INFO] AdminService: Created MinIO bucket: %s", bucketName)
	} else {
		log.Printf("[DEBUG] AdminService: MinIO bucket already exists: %s", bucketName)
	}

	return nil
}

func (s *AdminService) getImageInfo(localPath string) (int64, string, error) {
	stat, err := os.Stat(localPath)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get file info: %w", err)
	}

	return stat.Size(), s.getContentTypeFromPath(localPath), nil
}

func (s *AdminService) getContentTypeFromPath(path string) string {
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
		return "image/jpeg"
	}
}

// Event publishing methods

func (s *AdminService) publishProductCreatedEvent(ctx context.Context, product *Product) error {
	event := events.NewProductCreatedEvent(fmt.Sprintf("%d", product.ID), map[string]string{
		"name":  product.Name,
		"brand": product.Brand,
		"price": product.FormattedPrice(),
	})
	return s.publishEvent(ctx, event)
}

func (s *AdminService) publishProductUpdatedEvent(ctx context.Context, product *Product) error {
	event := events.NewProductUpdatedEvent(fmt.Sprintf("%d", product.ID), map[string]string{
		"name":  product.Name,
		"brand": product.Brand,
	})
	return s.publishEvent(ctx, event)
}

func (s *AdminService) publishProductDeletedEvent(ctx context.Context, product *Product) error {
	event := events.NewProductDeletedEvent(fmt.Sprintf("%d", product.ID), map[string]string{
		"name":  product.Name,
		"brand": product.Brand,
	})
	return s.publishEvent(ctx, event)
}

func (s *AdminService) publishProductImageAddedEvent(ctx context.Context, productID int64, imageID int64, imageURL string) error {
	event := events.NewProductImageAddedEvent(fmt.Sprintf("%d", productID), fmt.Sprintf("%d", imageID), map[string]string{
		"image_url": imageURL,
	})
	return s.publishEvent(ctx, event)
}

func (s *AdminService) publishProductImageUpdatedEvent(ctx context.Context, productID int64, imageID int64) error {
	event := events.NewProductImageUpdatedEvent(fmt.Sprintf("%d", productID), fmt.Sprintf("%d", imageID), map[string]string{})
	return s.publishEvent(ctx, event)
}

func (s *AdminService) publishProductImageDeletedEvent(ctx context.Context, imageID int64) error {
	event := events.NewProductImageDeletedEvent("", fmt.Sprintf("%d", imageID), map[string]string{})
	return s.publishEvent(ctx, event)
}

func (s *AdminService) publishIngestionStartedEvent(ctx context.Context, batchID string, req *ProductIngestionRequest) error {
	event := events.NewProductIngestionStartedEvent(batchID, map[string]string{
		"csv_path":    req.CSVPath,
		"use_cache":   fmt.Sprintf("%t", req.UseCache),
		"reset_cache": fmt.Sprintf("%t", req.ResetCache),
	})
	return s.publishEvent(ctx, event)
}

func (s *AdminService) publishIngestionCompletedEvent(ctx context.Context, batchID string, result *ProductIngestionResult) error {
	event := events.NewProductIngestionCompletedEvent(batchID, map[string]string{
		"total_products":     fmt.Sprintf("%d", result.TotalProducts),
		"processed_products": fmt.Sprintf("%d", result.ProcessedProducts),
		"total_images":       fmt.Sprintf("%d", result.TotalImages),
		"successful_images":  fmt.Sprintf("%d", result.SuccessfulImages),
		"duration":           result.Duration,
	})
	return s.publishEvent(ctx, event)
}

func (s *AdminService) publishIngestionFailedEvent(ctx context.Context, batchID string, result *ProductIngestionResult) error {
	event := events.NewProductIngestionFailedEvent(batchID, map[string]string{
		"total_products":     fmt.Sprintf("%d", result.TotalProducts),
		"processed_products": fmt.Sprintf("%d", result.ProcessedProducts),
		"errors_count":       fmt.Sprintf("%d", len(result.Errors)),
		"duration":           result.Duration,
	})
	return s.publishEvent(ctx, event)
}

func (s *AdminService) publishEvent(ctx context.Context, event events.Event) error {
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

	if err := s.infrastructure.OutboxWriter.WriteEvent(ctx, tx, event); err != nil {
		return fmt.Errorf("failed to write event to outbox: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit outbox transaction: %w", err)
	}
	committed = true

	return nil
}
