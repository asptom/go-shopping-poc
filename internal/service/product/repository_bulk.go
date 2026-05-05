// Package product provides data access operations for product entities.
//
// This file contains bulk operations for efficient batch processing
// of products and product images.
package product

import (
	"context"
	"fmt"
	events "go-shopping-poc/internal/contracts/events"
)

// BulkInsertProducts inserts multiple products efficiently
func (r *productRepository) BulkInsertProducts(ctx context.Context, products []*Product) error {
	r.logger.Debug("Bulk inserting products", "count", len(products))

	if len(products) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %w", ErrTransactionFailed, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for _, product := range products {
		r.prepareProductDefaults(product)
		if err := product.Validate(); err != nil {
			return fmt.Errorf("product validation failed for ID %d: %w", product.ID, err)
		}

		if err := r.insertProductRecord(ctx, tx, product); err != nil {
			return fmt.Errorf("failed to insert product %d: %w", product.ID, err)
		}

		// Publish product created event to outbox
		// This allows other services to initialize their product cache (if they have one)
		evt := events.NewProductCreatedEvent(fmt.Sprintf("%d", product.ID), map[string]string{
			"name":        product.Name,
			"brand":       product.Brand,
			"final_price": product.FormattedPrice(),
			"in_stock":    fmt.Sprintf("%t", product.InStock),
			"images":      fmt.Sprintf("%d", len(product.Images)),
		})

		if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
			return fmt.Errorf("%w: failed to write product created event: %w", ErrEventWriteFailed, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit bulk insert: %w", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}

// BulkInsertProductImages inserts multiple product images efficiently
func (r *productRepository) BulkInsertProductImages(ctx context.Context, images []*ProductImage) error {
	r.logger.Debug("Bulk inserting product images", "count", len(images))

	if len(images) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %w", ErrTransactionFailed, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	for _, image := range images {
		if err := image.Validate(); err != nil {
			return fmt.Errorf("image validation failed for product %d: %w", image.ProductID, err)
		}

		if err := r.insertProductImageRecord(ctx, tx, image); err != nil {
			return fmt.Errorf("failed to insert image for product %d: %w", image.ProductID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit bulk insert: %w", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}
