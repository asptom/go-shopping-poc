// Package product provides business logic for product catalog operations.
//
// This file contains the catalog service which handles read-only operations
// for product browsing and searching. The catalog service publishes
// view events for analytics and marketing purposes using the outbox pattern.
package product

import (
	"context"
	"fmt"
	"log"

	"go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/outbox"
	"go-shopping-poc/internal/platform/service"
)

// CatalogService handles read-only product operations and event publishing.
//
// CatalogService is focused on product browsing and retrieval with
// optional event publishing for analytics. All view events are
// published using the outbox pattern for reliable delivery.

// CatalogInfrastructure defines infrastructure components for catalog service
type CatalogInfrastructure struct {
	Database     database.Database
	OutboxWriter *outbox.Writer
}

type CatalogService struct {
	*service.BaseService
	repo           ProductRepository
	infrastructure *CatalogInfrastructure
	config         *Config
}

// NewCatalogService creates a new catalog service instance.
func NewCatalogService(infrastructure *CatalogInfrastructure, config *Config) *CatalogService {

	repo := NewProductRepository(infrastructure.Database, infrastructure.OutboxWriter)

	return &CatalogService{
		BaseService:    service.NewBaseService("product"),
		repo:           repo,
		infrastructure: infrastructure,
		config:         config,
	}
}

// GetProductByID retrieves a product by ID and publishes a view event
func (s *CatalogService) GetProductByID(ctx context.Context, productID int64) (*Product, error) {
	log.Printf("[INFO] CatalogService: Fetching product by ID: %d", productID)

	product, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		log.Printf("[ERROR] CatalogService: Failed to get product %d: %v", productID, err)
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	if s.infrastructure.OutboxWriter != nil {
		if err := s.publishProductViewedEvent(ctx, product); err != nil {
			log.Printf("[WARN] CatalogService: Failed to publish product viewed event: %v", err)
		}
	}

	return product, nil
}

// GetAllProducts retrieves all products with pagination
func (s *CatalogService) GetAllProducts(ctx context.Context, limit, offset int) ([]*Product, error) {
	log.Printf("[INFO] CatalogService: Fetching all products (limit: %d, offset: %d)", limit, offset)

	products, err := s.repo.GetAllProducts(ctx, limit, offset)
	if err != nil {
		log.Printf("[ERROR] CatalogService: Failed to get all products: %v", err)
		return nil, fmt.Errorf("failed to get all products: %w", err)
	}

	return products, nil
}

// GetProductsByCategory retrieves products by category and publishes a category view event
func (s *CatalogService) GetProductsByCategory(ctx context.Context, category string, limit, offset int) ([]*Product, error) {
	log.Printf("[INFO] CatalogService: Fetching products by category: %s (limit: %d, offset: %d)", category, limit, offset)

	products, err := s.repo.GetProductsByCategory(ctx, category, limit, offset)
	if err != nil {
		log.Printf("[ERROR] CatalogService: Failed to get products by category %s: %v", category, err)
		return nil, fmt.Errorf("failed to get products by category: %w", err)
	}

	if s.infrastructure.OutboxWriter != nil && len(products) > 0 {
		if err := s.publishCategoryViewedEvent(ctx, category, len(products)); err != nil {
			log.Printf("[WARN] CatalogService: Failed to publish category viewed event: %v", err)
		}
	}

	return products, nil
}

// GetProductsByBrand retrieves products by brand
func (s *CatalogService) GetProductsByBrand(ctx context.Context, brand string, limit, offset int) ([]*Product, error) {
	log.Printf("[INFO] CatalogService: Fetching products by brand: %s (limit: %d, offset: %d)", brand, limit, offset)

	products, err := s.repo.GetProductsByBrand(ctx, brand, limit, offset)
	if err != nil {
		log.Printf("[ERROR] CatalogService: Failed to get products by brand %s: %v", brand, err)
		return nil, fmt.Errorf("failed to get products by brand: %w", err)
	}

	return products, nil
}

// SearchProducts searches products and publishes a search event
func (s *CatalogService) SearchProducts(ctx context.Context, query string, limit, offset int) ([]*Product, error) {
	log.Printf("[INFO] CatalogService: Searching products with query: %s (limit: %d, offset: %d)", query, limit, offset)

	products, err := s.repo.SearchProducts(ctx, query, limit, offset)
	if err != nil {
		log.Printf("[ERROR] CatalogService: Failed to search products with query %s: %v", query, err)
		return nil, fmt.Errorf("failed to search products: %w", err)
	}

	if s.infrastructure.OutboxWriter != nil && len(products) > 0 {
		if err := s.publishSearchExecutedEvent(ctx, query, len(products)); err != nil {
			log.Printf("[WARN] CatalogService: Failed to publish search executed event: %v", err)
		}
	}

	return products, nil
}

// GetProductsInStock retrieves products that are in stock
func (s *CatalogService) GetProductsInStock(ctx context.Context, limit, offset int) ([]*Product, error) {
	log.Printf("[INFO] CatalogService: Fetching in-stock products (limit: %d, offset: %d)", limit, offset)

	products, err := s.repo.GetProductsInStock(ctx, limit, offset)
	if err != nil {
		log.Printf("[ERROR] CatalogService: Failed to get in-stock products: %v", err)
		return nil, fmt.Errorf("failed to get in-stock products: %w", err)
	}

	return products, nil
}

// Event publishing methods using outbox pattern

func (s *CatalogService) publishProductViewedEvent(ctx context.Context, product *Product) error {
	tx, err := s.infrastructure.Database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	event := events.NewProductViewedEvent(fmt.Sprintf("%d", product.ID), map[string]string{
		"product_id": fmt.Sprintf("%d", product.ID),
		"name":       product.Name,
		"brand":      product.Brand,
		"category":   product.Category,
	})

	if err := s.infrastructure.OutboxWriter.WriteEvent(ctx, tx, event); err != nil {
		return fmt.Errorf("failed to write event to outbox: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true

	return nil
}

func (s *CatalogService) publishSearchExecutedEvent(ctx context.Context, query string, resultCount int) error {
	tx, err := s.infrastructure.Database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	event := events.NewProductSearchExecutedEvent(query, map[string]string{
		"query":        query,
		"result_count": fmt.Sprintf("%d", resultCount),
	})

	if err := s.infrastructure.OutboxWriter.WriteEvent(ctx, tx, event); err != nil {
		return fmt.Errorf("failed to write event to outbox: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true

	return nil
}

func (s *CatalogService) publishCategoryViewedEvent(ctx context.Context, category string, resultCount int) error {
	tx, err := s.infrastructure.Database.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	event := events.NewProductCategoryViewedEvent(category, map[string]string{
		"category":     category,
		"result_count": fmt.Sprintf("%d", resultCount),
	})

	if err := s.infrastructure.OutboxWriter.WriteEvent(ctx, tx, event); err != nil {
		return fmt.Errorf("failed to write event to outbox: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	committed = true

	return nil
}
