package product

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/csv"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/downloader"
	"go-shopping-poc/internal/platform/logging"
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
// AdminInfrastructure defines infrastructure components for admin service

type AdminInfrastructure struct {
	Database       database.Database
	ObjectStorage  minio.ObjectStorage
	OutboxWriter   *outbox.Writer
	HTTPDownloader downloader.HTTPDownloader
}

func NewAdminInfrastructure(
	db database.Database,
	storage minio.ObjectStorage,
	writer *outbox.Writer,
	downloader downloader.HTTPDownloader) *AdminInfrastructure {
	return &AdminInfrastructure{
		Database:       db,
		ObjectStorage:  storage,
		OutboxWriter:   writer,
		HTTPDownloader: downloader,
	}
}

type AdminService struct {
	*service.BaseService
	logger         *slog.Logger
	repo           ProductRepository
	infrastructure *AdminInfrastructure
	config         *Config
}

// NewAdminService creates a new admin service instance.
func NewAdminService(logger *slog.Logger, config interface{}, infrastructure *AdminInfrastructure) *AdminService {
	if logger == nil {
		logger = logging.FromContext(context.Background())
	}
	repo := NewProductRepository(infrastructure.Database, infrastructure.OutboxWriter, logger)

	var cfg *Config
	switch c := config.(type) {
	case *AdminConfig:
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
		logger:         logger.With("component", "admin_service"),
		repo:           repo,
		config:         cfg,
		infrastructure: infrastructure,
	}
}

// CreateProduct creates a new product with validation and event publishing
func (s *AdminService) CreateProduct(ctx context.Context, product *Product) error {
	s.logger.Info("Creating new product")

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
		s.logger.Error("Failed to insert product", "error", err.Error())
		return fmt.Errorf("failed to insert product: %w", err)
	}

	s.logger.Info("Successfully created product", "product_id", product.ID)
	return nil
}

// UpdateProduct updates an existing product
func (s *AdminService) UpdateProduct(ctx context.Context, product *Product) error {
	s.logger.Info("Updating product", "product_id", product.ID)

	if err := product.Validate(); err != nil {
		return fmt.Errorf("product validation failed: %w", err)
	}

	if product.ID == 0 {
		return fmt.Errorf("product ID is required for update")
	}

	product.UpdatedAt = time.Now()

	if err := s.repo.UpdateProduct(ctx, product); err != nil {
		s.logger.Error("Failed to update product", "product_id", product.ID, "error", err.Error())
		return fmt.Errorf("failed to update product: %w", err)
	}

	s.logger.Info("Successfully updated product", "product_id", product.ID)
	return nil
}

// DeleteProduct removes a product and publishes deletion event
func (s *AdminService) DeleteProduct(ctx context.Context, productID int64) error {
	s.logger.Info("Deleting product", "product_id", productID)

	if productID == 0 {
		return fmt.Errorf("product ID is required for deletion")
	}

	product, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		s.logger.Error("Failed to get product for deletion", "product_id", productID, "error", err.Error())
		return fmt.Errorf("failed to get product for deletion: %w", err)
	}

	if err := s.repo.DeleteProduct(ctx, product.ID); err != nil {
		s.logger.Error("Failed to delete product", "product_id", productID, "error", err.Error())
		return fmt.Errorf("failed to delete product: %w", err)
	}

	s.logger.Info("Successfully deleted product", "product_id", productID)
	return nil
}

// AddProductImage adds an image to a product
func (s *AdminService) AddProductImage(ctx context.Context, image *ProductImage) error {
	s.logger.Info("Adding image for product", "product_id", image.ProductID)

	if err := image.Validate(); err != nil {
		return fmt.Errorf("image validation failed: %w", err)
	}

	if err := s.repo.AddProductImage(ctx, image); err != nil {
		s.logger.Error("Failed to add image", "error", err.Error())
		return fmt.Errorf("failed to add image: %w", err)
	}

	s.logger.Info("Successfully added image for product", "product_id", image.ProductID)
	return nil
}

// UpdateProductImage updates an existing product image
func (s *AdminService) UpdateProductImage(ctx context.Context, image *ProductImage) error {
	s.logger.Info("Updating image", "image_id", image.ID)

	if err := image.Validate(); err != nil {
		return fmt.Errorf("image validation failed: %w", err)
	}

	if err := s.repo.UpdateProductImage(ctx, image); err != nil {
		s.logger.Error("Failed to update image", "image_id", image.ID, "error", err.Error())
		return fmt.Errorf("failed to update image: %w", err)
	}

	s.logger.Info("Successfully updated image", "image_id", image.ID)
	return nil
}

// DeleteProductImage removes a product image
func (s *AdminService) DeleteProductImage(ctx context.Context, imageID int64) error {
	s.logger.Info("Deleting image", "image_id", imageID)

	// Get image details first (for Minio deletion)
	// ADDED: Need to fetch image before deleting to get MinioObjectName
	image, err := s.repo.GetProductImageByID(ctx, imageID)
	if err != nil {
		s.logger.Error("Failed to get image for deletion", "image_id", imageID, "error", err.Error())
		return fmt.Errorf("failed to get image for deletion: %w", err)
	}

	// Delete from database first
	if err := s.repo.DeleteProductImage(ctx, image); err != nil {
		s.logger.Error("Failed to delete image", "image_id", imageID, "error", err.Error())
		return fmt.Errorf("failed to delete image: %w", err)
	}

	// ADDED: Delete from Minio
	if s.infrastructure.ObjectStorage != nil && image.MinioObjectName != "" {
		if err := s.infrastructure.ObjectStorage.RemoveObject(ctx, s.config.MinIOBucket, image.MinioObjectName); err != nil {
			s.logger.Warn("Failed to delete image from Minio", "error", err.Error())
			// Don't fail the operation if Minio deletion fails
		}
	}

	return nil
}

// AddSingleImage adds a single image to a product by downloading from URL and uploading to Minio
// ADDED: New method for admin API to add individual images
func (s *AdminService) AddSingleImage(ctx context.Context, productID int64, imageURL string, isMain bool) (*ProductImage, error) {
	s.logger.Info("Adding single image for product from URL", "product_id", productID)

	if productID <= 0 {
		return nil, fmt.Errorf("product ID must be positive")
	}

	if imageURL == "" {
		return nil, fmt.Errorf("image URL is required")
	}

	// Download image
	localPath, err := s.infrastructure.HTTPDownloader.Download(ctx, imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}

	// Get existing image count for ordering
	existingImages, err := s.repo.GetProductImages(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing images: %w", err)
	}
	imageOrder := len(existingImages)

	// Upload to Minio
	minioObjectName, err := s.uploadImageToMinIO(ctx, localPath, productID, imageOrder)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image to MinIO: %w", err)
	}

	// Get file info
	fileSize, contentType, err := s.getImageInfo(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get image info: %w", err)
	}

	// Handle is_main logic - if this is main, unset others
	if isMain {
		if err := s.unsetAllMainImages(ctx, productID); err != nil {
			s.logger.Warn("Failed to unset existing main images", "error", err.Error())
		}
	}

	// Create image record
	productImage := &ProductImage{
		ProductID:       productID,
		MinioObjectName: minioObjectName,
		IsMain:          isMain,
		ImageOrder:      imageOrder,
		FileSize:        fileSize,
		ContentType:     contentType,
	}

	if err := s.repo.AddProductImage(ctx, productImage); err != nil {
		return nil, fmt.Errorf("failed to add product image: %w", err)
	}

	s.logger.Info("Successfully added image to product", "image_id", productImage.ID, "product_id", productID)
	return productImage, nil
}

// SetMainImage sets the main image for a product
func (s *AdminService) SetMainImage(ctx context.Context, productID int64, imageID int64) error {
	s.logger.Info("Setting main image for product", "image_id", imageID, "product_id", productID)

	// CHANGED: Use SetMainImageFlag instead of SetMainImage (repository method renamed)
	if err := s.repo.SetMainImageFlag(ctx, productID, imageID); err != nil {
		s.logger.Error("Failed to set main image", "error", err.Error())
		return fmt.Errorf("failed to set main image: %w", err)
	}

	return nil
}

// IngestProductsFromCSV orchestrates the complete product ingestion workflow
func (s *AdminService) IngestProductsFromCSV(ctx context.Context, req *ProductIngestionRequest) (*ProductIngestionResult, error) {
	s.logger.Info("Starting product ingestion from CSV", "csv_path", req.CSVPath)

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
		s.logger.Warn("Failed to publish ingestion started event", "error", err.Error())
	}

	defer func() {
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime).String()

		if len(result.Errors) > 0 {
			if err := s.publishIngestionFailedEvent(ctx, batchID, result); err != nil {
				s.logger.Warn("Failed to publish ingestion failed event", "error", err.Error())
			}
		} else {
			if err := s.publishIngestionCompletedEvent(ctx, batchID, result); err != nil {
				s.logger.Warn("Failed to publish ingestion completed event", "error", err.Error())
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
	s.logger.Info("Parsed products from CSV", "count", result.TotalProducts)

	if req.ResetCache {
		if err := s.infrastructure.HTTPDownloader.ClearCache(); err != nil {
			s.logger.Warn("Failed to clear cache", "error", err.Error())
		} else {
			s.logger.Info("Cache cleared successfully")
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
		s.logger.Info("Processing batch of products", "start", i+1, "end", end, "total", len(records))

		if err := s.processProductBatch(ctx, batch, req.UseCache, result); err != nil {
			errMsg := fmt.Sprintf("Failed to process batch %d-%d: %v", i+1, end, err)
			result.Errors = append(result.Errors, errMsg)
			s.logger.Error("Batch processing failed", "error", err.Error())
		}
	}

	s.logger.Info("Ingestion completed", "processed_products", result.ProcessedProducts, "total_products", result.TotalProducts, "successful_images", result.SuccessfulImages, "total_images", result.TotalImages)

	return result, nil
}

func (s *AdminService) processProductBatch(ctx context.Context, records []ProductCSVRecord, useCache bool, result *ProductIngestionResult) error {
	products := make([]*Product, 0, len(records))

	for _, record := range records {
		product, err := s.convertCSVRecordToProduct(record)
		if err != nil {
			s.logger.Warn("Failed to convert CSV record", "error", err.Error())
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
			if images, err := s.processProductImages(ctx, product, useCache, result); err != nil {
				s.logger.Warn("Failed to process images for product", "product_id", product.ID, "error", err.Error())
			} else {
				// If images were processed successfully, update the product with the image list
				if s.repo != nil {
					err := s.repo.BulkInsertProductImages(ctx, images)
					if err != nil {
						return fmt.Errorf("failed to bulk insert product images: %w", err)
					}
				}
			}
		}
	}

	return nil
}

func (s *AdminService) processProductImages(ctx context.Context, product *Product, useCache bool, result *ProductIngestionResult) ([]*ProductImage, error) {
	if len(product.ImageURLs) == 0 {
		return nil, nil
	}

	productImages := make([]*ProductImage, 0, len(product.ImageURLs))
	imageURLs := s.filterImageURLs(product.ImageURLs)
	result.TotalImages += len(imageURLs)

	s.logger.Debug("Processing images for product", "count", len(imageURLs), "product_id", product.ID)

	for i, imageURL := range imageURLs {
		if productImage, err := s.processSingleImage(ctx, product, imageURL, i, useCache); err != nil {
			s.logger.Warn("Failed to process image for product", "image_index", i, "product_id", product.ID, "error", err.Error())
			result.FailedImages++
			continue
		} else {
			result.SuccessfulImages++
			productImages = append(productImages, productImage)
		}
	}

	return productImages, nil
}

func (s *AdminService) processSingleImage(ctx context.Context, product *Product, imageURL string, imageIndex int, useCache bool) (*ProductImage, error) {
	var localPath string
	var err error

	if s.infrastructure.HTTPDownloader != nil {
		if useCache && s.infrastructure.HTTPDownloader.IsCached(imageURL) {
			localPath = s.infrastructure.HTTPDownloader.GetCachePath(imageURL)
			s.logger.Debug("Using cached image for product", "product_id", product.ID, "image_index", imageIndex)
		} else {
			localPath, err = s.infrastructure.HTTPDownloader.Download(ctx, imageURL)
			if err != nil {
				return nil, fmt.Errorf("failed to download image: %w", err)
			}
			s.logger.Debug("Downloaded image for product", "product_id", product.ID, "image_index", imageIndex)
		}
	} else {
		localPath = "/tmp/test.jpg"
	}

	minioObjectName, err := s.uploadImageToMinIO(ctx, localPath, product.ID, imageIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to upload image to MinIO: %w", err)
	}

	fileSize, contentType, err := s.getImageInfo(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get image info: %w", err)
	}

	// IsMain is determined by matching against MainImageURL from CSV
	productImage := &ProductImage{
		ProductID:       product.ID,
		MinioObjectName: minioObjectName,
		IsMain:          imageURL == product.MainImageURL, // Match against CSV main_image column
		ImageOrder:      imageIndex,
		FileSize:        fileSize,
		ContentType:     contentType,
	}

	//if s.repo != nil {
	//	if err := s.repo.AddProductImage(ctx, productImage); err != nil {
	//		return fmt.Errorf("failed to insert product image: %w", err)
	//	}
	//}

	//s.logger.Debug("AdminService: Successfully processed image %d for product %d", imageIndex, product.ID)
	return productImage, nil
}

func (s *AdminService) convertCSVRecordToProduct(record ProductCSVRecord) (*Product, error) {
	product := &Product{
		Name:         record.Name,
		Description:  record.Description,
		InitialPrice: record.InitialPrice,
		FinalPrice:   record.FinalPrice,
		Currency:     record.Currency,
		Color:        record.Color,
		Size:         record.Size,
		// REMOVED: MainImage field - now stored in MainImageURL (temporary field)
		CountryCode:       record.CountryCode,
		ModelNumber:       record.ModelNumber,
		RootCategory:      record.RootCategory,
		Category:          record.Category,
		Brand:             record.Brand,
		AllAvailableSizes: record.AllAvailableSizes,
		ImageURLs:         record.ImageURLs,
		MainImageURL:      record.MainImage, // ADDED: Store main_image from CSV for IsMain determination
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
			s.logger.Warn("Failed to parse other_attributes JSON", "error", err.Error())
		} else {
			if jsonBytes, err := json.Marshal(otherAttrs); err != nil {
				s.logger.Warn("Failed to marshal other_attributes JSON", "error", err.Error())
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

		s.logger.Debug("Uploaded image to MinIO", "object_name", objectName, "size", info.Size)
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
		s.logger.Info("Created MinIO bucket", "bucket", bucketName)
	} else {
		s.logger.Debug("MinIO bucket already exists", "bucket", bucketName)
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

// Non-transactional Oriduct Event publishing methods

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

// GetProductImageByID retrieves a single image by its ID
func (s *AdminService) GetProductImageByID(ctx context.Context, imageID int64) (*ProductImage, error) {
	s.logger.Debug("Fetching image", "image_id", imageID)

	if imageID <= 0 {
		return nil, fmt.Errorf("image ID must be positive")
	}

	image, err := s.repo.GetProductImageByID(ctx, imageID)
	if err != nil {
		s.logger.Error("Failed to get image", "image_id", imageID, "error", err.Error())
		return nil, fmt.Errorf("failed to get image: %w", err)
	}

	return image, nil
}
