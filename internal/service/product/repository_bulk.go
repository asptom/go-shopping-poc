// Package product provides data access operations for product entities.
//
// This file contains bulk operations for efficient batch processing
// of products and product images.
package product

import (
	"context"
	"fmt"
	"log"
)

// BulkInsertProducts inserts multiple products efficiently
func (r *productRepository) BulkInsertProducts(ctx context.Context, products []*Product) error {
	log.Printf("[DEBUG] Repository: Bulk inserting %d products", len(products))

	if len(products) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", ErrTransactionFailed, err)
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
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit bulk insert: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}

// BulkInsertProductImages inserts multiple product images efficiently
func (r *productRepository) BulkInsertProductImages(ctx context.Context, images []*ProductImage) error {
	log.Printf("[DEBUG] Repository: Bulk inserting %d product images", len(images))

	if len(images) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", ErrTransactionFailed, err)
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
		return fmt.Errorf("%w: failed to commit bulk insert: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}
