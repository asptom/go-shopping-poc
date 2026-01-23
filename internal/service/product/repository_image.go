// Package product provides data access operations for product entities.
//
// This file contains image-related operations including adding, updating,
// deleting, and managing product images.
package product

import (
	"context"
	"fmt"
	"log"
)

// AddProductImage adds a new image to a product
func (r *productRepository) AddProductImage(ctx context.Context, image *ProductImage) error {
	log.Printf("[DEBUG] Repository: Adding image for product %d", image.ProductID)

	if err := image.Validate(); err != nil {
		return fmt.Errorf("image validation failed: %w", err)
	}

	_, err := r.GetProductByID(ctx, image.ProductID)
	if err != nil {
		return fmt.Errorf("cannot add image to non-existent product: %w", err)
	}

	query := `
		INSERT INTO products.product_images (
			product_id, image_url, minio_object_name, is_main, image_order,
			file_size, content_type, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)`

	_, err = r.db.ExecContext(ctx, query,
		image.ProductID, image.ImageURL, image.MinioObjectName, image.IsMain,
		image.ImageOrder, image.FileSize, image.ContentType, image.CreatedAt,
	)
	if err != nil {
		if isDuplicateError(err) {
			return fmt.Errorf("%w: image URL already exists for product", ErrDuplicateImage)
		}
		return fmt.Errorf("%w: failed to add product image: %v", ErrDatabaseOperation, err)
	}

	return nil
}

// UpdateProductImage updates an existing product image
func (r *productRepository) UpdateProductImage(ctx context.Context, image *ProductImage) error {
	log.Printf("[DEBUG] Repository: Updating image %d", image.ID)

	if image.ID <= 0 {
		return fmt.Errorf("%w: image ID must be positive", ErrInvalidProductID)
	}

	if err := image.Validate(); err != nil {
		return fmt.Errorf("image validation failed: %w", err)
	}

	query := `
		UPDATE products.product_images SET
			image_url = $2, minio_object_name = $3, is_main = $4, image_order = $5,
			file_size = $6, content_type = $7
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		image.ID, image.ImageURL, image.MinioObjectName, image.IsMain,
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

	return nil
}

// DeleteProductImage removes a product image
func (r *productRepository) DeleteProductImage(ctx context.Context, imageID int64) error {
	log.Printf("[DEBUG] Repository: Deleting image %d", imageID)

	if imageID <= 0 {
		return fmt.Errorf("%w: image ID must be positive", ErrInvalidProductID)
	}

	query := `DELETE FROM products.product_images WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, imageID)
	if err != nil {
		return fmt.Errorf("%w: failed to delete product image: %v", ErrDatabaseOperation, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: failed to get rows affected: %v", ErrDatabaseOperation, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: product image %d not found", ErrProductImageNotFound, imageID)
	}

	return nil
}

// GetProductImages retrieves all images for a product
func (r *productRepository) GetProductImages(ctx context.Context, productID int64) ([]ProductImage, error) {
	log.Printf("[DEBUG] Repository: Fetching images for product %d", productID)

	if productID <= 0 {
		return nil, fmt.Errorf("%w: product ID must be positive", ErrInvalidProductID)
	}

	query := `
		SELECT id, product_id, image_url, minio_object_name, is_main, image_order,
			   file_size, content_type, created_at
		FROM products.product_images
		WHERE product_id = $1
		ORDER BY image_order, created_at`

	rows, err := r.db.QueryContext(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to query product images: %v", ErrDatabaseOperation, err)
	}
	defer func() { _ = rows.Close() }()

	var images []ProductImage
	for rows.Next() {
		var image ProductImage
		err := rows.Scan(
			&image.ID, &image.ProductID, &image.ImageURL, &image.MinioObjectName,
			&image.IsMain, &image.ImageOrder, &image.FileSize, &image.ContentType, &image.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to scan image row: %v", ErrDatabaseOperation, err)
		}
		images = append(images, image)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: error iterating image rows: %v", ErrDatabaseOperation, err)
	}

	return images, nil
}

// SetMainImage sets the main image for a product
func (r *productRepository) SetMainImage(ctx context.Context, productID int64, imageID int64) error {
	log.Printf("[DEBUG] Repository: Setting main image %d for product %d", imageID, productID)

	if productID <= 0 || imageID <= 0 {
		return fmt.Errorf("%w: product ID and image ID must be positive", ErrInvalidProductID)
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

	unsetQuery := `UPDATE products.product_images SET is_main = false WHERE product_id = $1`
	_, err = tx.ExecContext(ctx, unsetQuery, productID)
	if err != nil {
		return fmt.Errorf("%w: failed to unset main images: %v", ErrDatabaseOperation, err)
	}

	setQuery := `UPDATE products.product_images SET is_main = true WHERE id = $1 AND product_id = $2`
	result, err := tx.ExecContext(ctx, setQuery, imageID, productID)
	if err != nil {
		return fmt.Errorf("%w: failed to set main image: %v", ErrDatabaseOperation, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: failed to get rows affected: %v", ErrDatabaseOperation, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: image %d not found for product %d", ErrProductImageNotFound, imageID, productID)
	}

	updateProductQuery := `
		UPDATE products SET main_image = (
			SELECT image_url FROM products.product_images WHERE id = $1
		) WHERE id = $2`
	_, err = tx.ExecContext(ctx, updateProductQuery, imageID, productID)
	if err != nil {
		return fmt.Errorf("%w: failed to update product main image: %v", ErrDatabaseOperation, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}
