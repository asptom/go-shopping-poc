// Package product provides business logic for product catalog operations.
//
// This file contains the catalog service which handles read-only operations
// for product browsing and searching. The catalog service publishes
// view events for analytics and marketing purposes using the outbox pattern.
package product

import (
	"context"
	"fmt"
	"log/slog"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/database"
	"go-shopping-poc/internal/platform/event/bus"
	"go-shopping-poc/internal/platform/logging"
	"go-shopping-poc/internal/platform/outbox"
	"go-shopping-poc/internal/platform/service"
)

type CatalogInfrastructure struct {
	Database        database.Database
	OutboxWriter    *outbox.Writer
	OutboxPublisher *outbox.Publisher
	EventBus        bus.Bus
}

type Service interface {
	service.Service
}

func RegisterHandler[T events.Event](s Service, factory events.EventFactory[T], handler bus.HandlerFunc[T]) error {
	return service.RegisterHandler(s, factory, handler)
}

type CatalogService struct {
	*service.EventServiceBase
	logger         *slog.Logger
	repo           ProductRepository
	infrastructure *CatalogInfrastructure
	config         *Config
}

func NewCatalogService(logger *slog.Logger, infrastructure *CatalogInfrastructure, config *Config) *CatalogService {
	if logger == nil {
		logger = logging.FromContext(context.Background())
	}
	repo := NewProductRepository(infrastructure.Database, infrastructure.OutboxWriter, logger)

	return &CatalogService{
		EventServiceBase: service.NewEventServiceBase("product", infrastructure.EventBus, logger),
		logger:           logger.With("component", "catalog_service"),
		repo:             repo,
		infrastructure:   infrastructure,
		config:           config,
	}
}

func (s *CatalogService) GetInfrastructure() *CatalogInfrastructure {
	return s.infrastructure
}

func (s *CatalogService) GetProductByID(ctx context.Context, productID int64) (*Product, error) {
	s.logger.Debug("Fetching product by ID", "product_id", productID)

	product, err := s.repo.GetProductByID(ctx, productID)
	if err != nil {
		s.logger.Error("Failed to get product", "product_id", productID, "error", err.Error())
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	return product, nil
}

func (s *CatalogService) GetAllProducts(ctx context.Context, limit, offset int) ([]*Product, error) {
	s.logger.Debug("Fetching all products", "limit", limit, "offset", offset)

	products, err := s.repo.GetAllProducts(ctx, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get all products", "error", err.Error())
		return nil, fmt.Errorf("failed to get all products: %w", err)
	}

	return products, nil
}

func (s *CatalogService) GetProductsByCategory(ctx context.Context, category string, limit, offset int) ([]*Product, error) {
	s.logger.Debug("Fetching products by category", "category", category, "limit", limit, "offset", offset)

	products, err := s.repo.GetProductsByCategory(ctx, category, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get products by category", "category", category, "error", err.Error())
		return nil, fmt.Errorf("failed to get products by category: %w", err)
	}

	if s.infrastructure.OutboxWriter != nil && len(products) > 0 {
		if err := s.publishCategoryViewedEvent(ctx, category, len(products)); err != nil {
			s.logger.Warn("Failed to publish category viewed event", "error", err.Error())
		}
	}

	return products, nil
}

func (s *CatalogService) GetProductsByBrand(ctx context.Context, brand string, limit, offset int) ([]*Product, error) {
	s.logger.Debug("Fetching products by brand", "brand", brand, "limit", limit, "offset", offset)

	products, err := s.repo.GetProductsByBrand(ctx, brand, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get products by brand", "brand", brand, "error", err.Error())
		return nil, fmt.Errorf("failed to get products by brand: %w", err)
	}

	return products, nil
}

func (s *CatalogService) SearchProducts(ctx context.Context, query string, limit, offset int) ([]*Product, error) {
	s.logger.Debug("Searching products", "query", query, "limit", limit, "offset", offset)

	products, err := s.repo.SearchProducts(ctx, query, limit, offset)
	if err != nil {
		s.logger.Error("Failed to search products", "query", query, "error", err.Error())
		return nil, fmt.Errorf("failed to search products: %w", err)
	}

	if s.infrastructure.OutboxWriter != nil && len(products) > 0 {
		if err := s.publishSearchExecutedEvent(ctx, query, len(products)); err != nil {
			s.logger.Warn("Failed to publish search executed event", "error", err.Error())
		}
	}

	return products, nil
}

func (s *CatalogService) GetProductsInStock(ctx context.Context, limit, offset int) ([]*Product, error) {
	s.logger.Debug("Fetching in-stock products", "limit", limit, "offset", offset)

	products, err := s.repo.GetProductsInStock(ctx, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get in-stock products", "error", err.Error())
		return nil, fmt.Errorf("failed to get in-stock products: %w", err)
	}

	return products, nil
}

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
