// Package product provides data access operations for product entities.
//
// This file contains image-related operations including adding, updating,
// deleting, and managing product images.
package product

import (
	"context"
	"fmt"
	"log"
	"time"

	events "go-shopping-poc/internal/contracts/events"
	"go-shopping-poc/internal/platform/database"
)

// AddProductImage adds a new image to a product
func (r *productRepository) AddProductImage(ctx context.Context, image *ProductImage) error {
	log.Printf("[DEBUG] Repository: Adding image for product %d", image.ProductID)

	if err := image.Validate(); err != nil {
		return fmt.Errorf("image validation failed: %w", err)
	}

	_, err := r.ProductExists(ctx, image.ProductID)
	if err != nil {
		return fmt.Errorf("cannot add image to non-existent product: %w", err)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", ErrTransactionFailed, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	query := `
		INSERT INTO products.product_images (
			product_id, minio_object_name, is_main, image_order,
			file_size, content_type, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)`

	_, err = tx.ExecContext(ctx, query,
		image.ProductID, image.MinioObjectName, image.IsMain,
		image.ImageOrder, image.FileSize, image.ContentType, image.CreatedAt,
	)
	if err != nil {
		if isDuplicateError(err) {
			return fmt.Errorf("%w: minio object name already exists for product", ErrDuplicateImage)
		}
		return fmt.Errorf("%w: failed to add product image: %v", ErrDatabaseOperation, err)
	}

	// Publish product created event to outbox
	evt := events.NewProductImageAddedEvent(fmt.Sprintf("%d", image.ProductID), fmt.Sprintf("%d", image.ID), map[string]string{
		"name":   image.MinioObjectName,
		"brand":  image.ContentType,
		"price":  fmt.Sprintf("%d", image.FileSize),
		"images": fmt.Sprintf("%d", image.ImageOrder),
	})

	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return fmt.Errorf("%w: failed to write product image added event: %v", ErrEventWriteFailed, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil

}

// insertProductImages inserts product images within a transaction
func (r *productRepository) insertProductImages(ctx context.Context, tx database.Tx, images []ProductImage) error {

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
func (r *productRepository) insertProductImageRecord(ctx context.Context, tx database.Tx, image *ProductImage) error {

	query := `
		INSERT INTO products.product_images (
			product_id, minio_object_name, is_main, image_order,
			file_size, content_type, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)`

	_, err := tx.ExecContext(ctx, query,
		image.ProductID, image.MinioObjectName, image.IsMain,
		image.ImageOrder, image.FileSize, image.ContentType, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("%w: failed to insert product image: %v", ErrDatabaseOperation, err)
	}
	return nil
}

// UpdateProductImage updates an existing product image
func (r *productRepository) UpdateProductImage(ctx context.Context, image *ProductImage) error {
	log.Printf("[DEBUG] Repository: Updating image %d", image.ID)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", ErrTransactionFailed, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if image.ID <= 0 {
		return fmt.Errorf("%w: image ID must be positive", ErrInvalidProductID)
	}

	if err := image.Validate(); err != nil {
		return fmt.Errorf("image validation failed: %w", err)
	}

	query := `
		UPDATE products.product_images SET
			minio_object_name = $2, is_main = $3, image_order = $4,
			file_size = $5, content_type = $6
		WHERE id = $1`

	result, err := tx.ExecContext(ctx, query,
		image.ID, image.MinioObjectName, image.IsMain,
		image.ImageOrder, image.FileSize, image.ContentType,
	)
	if err != nil {
		return fmt.Errorf("%w: failed to update product image: %v", ErrDatabaseOperation, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: failed to get rows affected: %v", ErrDatabaseOperation, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: product image %d not found", ErrProductImageNotFound, image.ID)
	}

	// Publish product updated event to outbox
	evt := events.NewProductImageUpdatedEvent(fmt.Sprintf("%d", image.ProductID), fmt.Sprintf("%d", image.ID), map[string]string{
		"name":   image.MinioObjectName,
		"brand":  image.ContentType,
		"price":  fmt.Sprintf("%d", image.FileSize),
		"images": fmt.Sprintf("%d", image.ImageOrder),
	})

	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return fmt.Errorf("%w: failed to write product image updated event: %v", ErrEventWriteFailed, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}

// DeleteProductImage removes a product image
func (r *productRepository) DeleteProductImage(ctx context.Context, image *ProductImage) error {
	log.Printf("[DEBUG] Repository: Deleting image %d", image.ID)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", ErrTransactionFailed, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if image.ID <= 0 {
		return fmt.Errorf("%w: image ID must be positive", ErrInvalidProductID)
	}

	query := `DELETE FROM products.product_images WHERE id = $1`
	result, err := tx.ExecContext(ctx, query, image.ID)
	if err != nil {
		return fmt.Errorf("%w: failed to delete product image: %v", ErrDatabaseOperation, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: failed to get rows affected: %v", ErrDatabaseOperation, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: product image %d not found", ErrProductImageNotFound, image.ID)
	}

	// Publish product deleted event to outbox
	evt := events.NewProductImageDeletedEvent(fmt.Sprintf("%d", image.ProductID), fmt.Sprintf("%d", image.ID), map[string]string{
		"name":   image.MinioObjectName,
		"brand":  image.ContentType,
		"images": fmt.Sprintf("%d", image.ImageOrder),
	})

	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return fmt.Errorf("%w: failed to write product image deleted event: %v", ErrEventWriteFailed, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}

// SetMainImageFlag sets the is_main flag for an image without updating products.main_image
func (r *productRepository) SetMainImageFlag(ctx context.Context, productID int64, imageID int64) error {
	log.Printf("[DEBUG] Repository: Setting main image flag %d for product %d", imageID, productID)

	if productID <= 0 || imageID <= 0 {
		return fmt.Errorf("%w: product ID and image ID must be positive", ErrInvalidProductID)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: failed to begin transaction: %v", ErrTransactionFailed, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// Unset all main images for this product
	unsetQuery := `UPDATE products.product_images SET is_main = false WHERE product_id = $1`
	_, err = tx.Exec(ctx, unsetQuery, productID)
	if err != nil {
		return fmt.Errorf("%w: failed to unset main images: %v", ErrDatabaseOperation, err)
	}

	// Set the specified image as main
	setQuery := `UPDATE products.product_images SET is_main = true WHERE id = $1 AND product_id = $2`
	result, err := tx.Exec(ctx, setQuery, imageID, productID)
	if err != nil {
		return fmt.Errorf("%w: failed to set main image flag: %v", ErrDatabaseOperation, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: failed to get rows affected: %v", ErrDatabaseOperation, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: image %d not found for product %d", ErrProductImageNotFound, imageID, productID)
	}

	// Publish product updated event to outbox
	evt := events.NewProductImageUpdatedEvent(fmt.Sprintf("%d", productID), fmt.Sprintf("%d", imageID), map[string]string{
		"image_id": fmt.Sprintf("%d", imageID),
	})

	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return fmt.Errorf("%w: failed to write product image updated event: %v", ErrEventWriteFailed, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}

// unsetAllMainImages unsets the is_main flag for all images of a product
// ADDED: Helper method for AddSingleImage
func (s *AdminService) unsetAllMainImages(ctx context.Context, productID int64) error {
	query := `UPDATE products.product_images SET is_main = false WHERE product_id = $1`
	// Use DB() to get underlying sqlx.DB for ExecContext
	_, err := s.infrastructure.Database.DB().ExecContext(ctx, query, productID)
	return err
}
