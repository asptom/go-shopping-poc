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

// IngestionService handles CSV product ingestion workflow.
//
// IngestionService is focused on parsing CSV files and importing products
// with images. All operations publish events using the outbox pattern for reliable delivery.
// IngestionInfrastructure defines infrastructure components for ingestion service

type IngestionInfrastructure struct {
	Database       database.Database
	ObjectStorage  minio.ObjectStorage
	OutboxWriter   *outbox.Writer
	HTTPDownloader downloader.HTTPDownloader
}

func NewIngestionInfrastructure(
	db database.Database,
	storage minio.ObjectStorage,
	writer *outbox.Writer,
	downloader downloader.HTTPDownloader) *IngestionInfrastructure {
	return &IngestionInfrastructure{
		Database:       db,
		ObjectStorage:  storage,
		OutboxWriter:   writer,
		HTTPDownloader: downloader,
	}
}

type IngestionService struct {
	*service.BaseService
	logger         *slog.Logger
	repo           ProductRepository
	infrastructure *IngestionInfrastructure
	config         *Config
}

// NewIngestionService creates a new ingestion service instance.
func NewIngestionService(logger *slog.Logger, config interface{}, infrastructure *IngestionInfrastructure) *IngestionService {
	if logger == nil {
		logger = logging.FromContext(context.Background())
	}
	repo := NewProductRepository(infrastructure.Database, infrastructure.OutboxWriter, logger)

	var cfg *Config
	switch c := config.(type) {
	case *Config:
		cfg = c
	default:
		cfg = config.(*Config)
	}
	return &IngestionService{
		BaseService:    service.NewBaseService("product-ingestion"),
		logger:         logger.With("component", "ingestion_service"),
		repo:           repo,
		config:         cfg,
		infrastructure: infrastructure,
	}
}

// IngestProductsFromCSV orchestrates the complete product ingestion workflow
func (s *IngestionService) IngestProductsFromCSV(ctx context.Context, req *ProductIngestionRequest) (*ProductIngestionResult, error) {
	s.logger.Debug("Starting product ingestion from CSV", "csv_path", req.CSVPath)

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
	s.logger.Debug("Parsed products from CSV", "count", result.TotalProducts)

	if req.ResetCache {
		if err := s.infrastructure.HTTPDownloader.ClearCache(); err != nil {
			s.logger.Warn("Failed to clear cache", "error", err.Error())
		} else {
			s.logger.Debug("Cache cleared successfully")
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
		s.logger.Debug("Processing batch of products", "start", i+1, "end", end, "total", len(records))

		if err := s.processProductBatch(ctx, batch, req.UseCache, result); err != nil {
			errMsg := fmt.Sprintf("Failed to process batch %d-%d: %v", i+1, end, err)
			result.Errors = append(result.Errors, errMsg)
			s.logger.Error("Batch processing failed", "error", err.Error())
		}
	}

	s.logger.Debug("Ingestion completed", "processed_products", result.ProcessedProducts, "total_products", result.TotalProducts, "successful_images", result.SuccessfulImages, "total_images", result.TotalImages)

	return result, nil
}

func (s *IngestionService) processProductBatch(ctx context.Context, records []ProductCSVRecord, useCache bool, result *ProductIngestionResult) error {
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

func (s *IngestionService) processProductImages(ctx context.Context, product *Product, useCache bool, result *ProductIngestionResult) ([]*ProductImage, error) {
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

func (s *IngestionService) processSingleImage(ctx context.Context, product *Product, imageURL string, imageIndex int, useCache bool) (*ProductImage, error) {
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

	//s.logger.Debug("IngestionService: Successfully processed image %d for product %d", imageIndex, product.ID)
	return productImage, nil
}

func (s *IngestionService) convertCSVRecordToProduct(record ProductCSVRecord) (*Product, error) {
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

func (s *IngestionService) filterImageURLs(urls []string) []string {
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

func (s *IngestionService) uploadImageToMinIO(ctx context.Context, localPath string, productID int64, imageIndex int) (string, error) {
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

func (s *IngestionService) ensureBucketExists(ctx context.Context, bucketName string) error {
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
		s.logger.Debug("Created MinIO bucket", "bucket", bucketName)
	} else {
		s.logger.Debug("MinIO bucket already exists", "bucket", bucketName)
	}

	return nil
}

func (s *IngestionService) getImageInfo(localPath string) (int64, string, error) {
	stat, err := os.Stat(localPath)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get file info: %w", err)
	}

	return stat.Size(), s.getContentTypeFromPath(localPath), nil
}

func (s *IngestionService) getContentTypeFromPath(path string) string {
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

func (s *IngestionService) publishIngestionStartedEvent(ctx context.Context, batchID string, req *ProductIngestionRequest) error {
	event := events.NewProductIngestionStartedEvent(batchID, map[string]string{
		"csv_path":    req.CSVPath,
		"use_cache":   fmt.Sprintf("%t", req.UseCache),
		"reset_cache": fmt.Sprintf("%t", req.ResetCache),
	})
	return s.publishEvent(ctx, event)
}

func (s *IngestionService) publishIngestionCompletedEvent(ctx context.Context, batchID string, result *ProductIngestionResult) error {
	event := events.NewProductIngestionCompletedEvent(batchID, map[string]string{
		"total_products":     fmt.Sprintf("%d", result.TotalProducts),
		"processed_products": fmt.Sprintf("%d", result.ProcessedProducts),
		"total_images":       fmt.Sprintf("%d", result.TotalImages),
		"successful_images":  fmt.Sprintf("%d", result.SuccessfulImages),
		"duration":           result.Duration,
	})
	return s.publishEvent(ctx, event)
}

func (s *IngestionService) publishIngestionFailedEvent(ctx context.Context, batchID string, result *ProductIngestionResult) error {
	event := events.NewProductIngestionFailedEvent(batchID, map[string]string{
		"total_products":     fmt.Sprintf("%d", result.TotalProducts),
		"processed_products": fmt.Sprintf("%d", result.ProcessedProducts),
		"errors_count":       fmt.Sprintf("%d", len(result.Errors)),
		"duration":           result.Duration,
	})
	return s.publishEvent(ctx, event)
}

func (s *IngestionService) publishEvent(ctx context.Context, event events.Event) error {
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
