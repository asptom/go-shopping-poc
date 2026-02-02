// Package product provides data access operations for product entities.
//
// This file contains utility functions for product repository operations,
// including helper methods for validation, default values, and error handling.
package product

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// prepareProductDefaults sets default values for missing product fields
func (r *productRepository) prepareProductDefaults(product *Product) {
	if product.CreatedAt.IsZero() {
		product.CreatedAt = time.Now()
	}
	if product.UpdatedAt.IsZero() {
		product.UpdatedAt = time.Now()
	}
	if product.Currency == "" {
		product.Currency = "USD"
	}
	if !product.InStock {
		product.InStock = true
	}
}

// insertProductWithImages handles the complete product creation process
func (r *productRepository) insertProductWithImages(ctx context.Context, product *Product) error {
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

	if err := r.insertProductRecord(ctx, tx, product); err != nil {
		return err
	}

	if len(product.Images) > 0 {
		if err := r.insertProductImages(ctx, tx, product.Images); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}

// insertProductRecord inserts the product record within a transaction
func (r *productRepository) insertProductRecord(ctx context.Context, tx *sqlx.Tx, product *Product) error {
	// UPDATED: Removed main_image from query
	query := `
		INSERT INTO products.products (
			id, name, description, initial_price, final_price, currency, in_stock,
			color, size, country_code, image_count, model_number,
			root_category, category, brand, other_attributes, all_available_sizes,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
		) ON CONFLICT (id) DO NOTHING`

	// UPDATED: Removed product.MainImage from parameters
	_, err := tx.ExecContext(ctx, query,
		product.ID, product.Name, product.Description, product.InitialPrice, product.FinalPrice,
		product.Currency, product.InStock, product.Color, product.Size,
		product.CountryCode, product.ImageCount, product.ModelNumber, product.RootCategory,
		product.Category, product.Brand, product.OtherAttributes, product.AllAvailableSizes,
		product.CreatedAt, product.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("%w: failed to insert product record: %v", ErrDatabaseOperation, err)
	}
	return nil
}

// insertProductImages inserts product images within a transaction
func (r *productRepository) insertProductImages(ctx context.Context, tx *sqlx.Tx, images []ProductImage) error {
	// UPDATED: Removed image_url from query
	query := `
		INSERT INTO products.product_images (
			product_id, minio_object_name, is_main, image_order,
			file_size, content_type, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)`

	for _, image := range images {
		if err := image.Validate(); err != nil {
			return fmt.Errorf("image validation failed: %w", err)
		}

		// UPDATED: Removed image.ImageURL from parameters
		_, err := tx.ExecContext(ctx, query,
			image.ProductID, image.MinioObjectName, image.IsMain,
			image.ImageOrder, image.FileSize, image.ContentType, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("%w: failed to insert product image: %v", ErrDatabaseOperation, err)
		}
	}
	return nil
}

// insertProductImageRecord inserts a single product image record within a transaction
func (r *productRepository) insertProductImageRecord(ctx context.Context, tx *sqlx.Tx, image *ProductImage) error {
	// UPDATED: Removed image_url from query
	query := `
		INSERT INTO products.product_images (
			product_id, minio_object_name, is_main, image_order,
			file_size, content_type, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)`

	// UPDATED: Removed image.ImageURL from parameters
	_, err := tx.ExecContext(ctx, query,
		image.ProductID, image.MinioObjectName, image.IsMain,
		image.ImageOrder, image.FileSize, image.ContentType, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("%w: failed to insert product image: %v", ErrDatabaseOperation, err)
	}
	return nil
}

// isDuplicateError checks if an error is a duplicate key violation
func isDuplicateError(err error) bool {
	errStr := fmt.Sprintf("%v", err)
	return err != nil &&
		(strings.Contains(errStr, "duplicate") ||
			strings.Contains(errStr, "unique"))
}
