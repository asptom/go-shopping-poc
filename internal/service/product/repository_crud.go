// Package product provides data access operations for product entities.
//
// This file contains CRUD (Create, Read, Update, Delete) operations
// for product entities, handling database operations with proper transaction
// management and validation.
package product

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"
)

// InsertProduct creates a new product record in the database.
//
// This method handles the complete product creation process including:
// - Setting default values for missing fields
// - Inserting the product record
// - Inserting associated product images if provided
//
// The product ID must be provided and unique.
func (r *productRepository) InsertProduct(ctx context.Context, product *Product) error {
	log.Printf("[DEBUG] Repository: Inserting new product...")

	r.prepareProductDefaults(product)

	if err := product.Validate(); err != nil {
		return fmt.Errorf("product validation failed: %w", err)
	}

	return r.insertProductWithImages(ctx, product)
}

// GetProductByID retrieves a product by its ID with all associated images
func (r *productRepository) GetProductByID(ctx context.Context, productID int64) (*Product, error) {
	log.Printf("[DEBUG] Repository: Fetching product by ID: %d", productID)

	if productID <= 0 {
		return nil, fmt.Errorf("%w: product ID must be positive", ErrInvalidProductID)
	}

	query := `
		SELECT id, name, description, initial_price, final_price, currency, in_stock,
			   color, size, main_image, country_code, image_count, model_number,
			   root_category, category, brand, other_attributes, all_available_sizes,
			   created_at, updated_at
		FROM products.products WHERE id = $1`

	var product Product
	err := r.db.QueryRowContext(ctx, query, productID).Scan(
		&product.ID, &product.Name, &product.Description, &product.InitialPrice, &product.FinalPrice,
		&product.Currency, &product.InStock, &product.Color, &product.Size, &product.MainImage,
		&product.CountryCode, &product.ImageCount, &product.ModelNumber, &product.RootCategory,
		&product.Category, &product.Brand, &product.OtherAttributes, &product.AllAvailableSizes,
		&product.CreatedAt, &product.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%w: product %d not found", ErrProductNotFound, productID)
		}
		log.Printf("[ERROR] Error fetching product by ID: %v", err)
		return nil, fmt.Errorf("%w: failed to fetch product %d: %v", ErrDatabaseOperation, productID, err)
	}

	images, err := r.GetProductImages(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to load images for product %d: %w", productID, err)
	}
	product.Images = images

	return &product, nil
}

// UpdateProduct updates an existing product record
func (r *productRepository) UpdateProduct(ctx context.Context, product *Product) error {
	log.Printf("[DEBUG] Repository: Updating product %d", product.ID)

	if err := product.Validate(); err != nil {
		return fmt.Errorf("product validation failed: %w", err)
	}

	if product.ID <= 0 {
		return fmt.Errorf("%w: product ID must be positive", ErrInvalidProductID)
	}

	product.UpdatedAt = time.Now()

	query := `
		UPDATE products SET
			name = $2, description = $3, initial_price = $4, final_price = $5,
			currency = $6, in_stock = $7, color = $8, size = $9, main_image = $10,
			country_code = $11, image_count = $12, model_number = $13,
			root_category = $14, category = $15, brand = $16,
			other_attributes = $17, all_available_sizes = $18, updated_at = $19
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		product.ID, product.Name, product.Description, product.InitialPrice, product.FinalPrice,
		product.Currency, product.InStock, product.Color, product.Size, product.MainImage,
		product.CountryCode, product.ImageCount, product.ModelNumber, product.RootCategory,
		product.Category, product.Brand, product.OtherAttributes, product.AllAvailableSizes,
		product.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("%w: failed to update product: %v", ErrDatabaseOperation, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: failed to get rows affected: %v", ErrDatabaseOperation, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: product %d not found", ErrProductNotFound, product.ID)
	}

	return nil
}

// DeleteProduct removes a product and all its associated images
func (r *productRepository) DeleteProduct(ctx context.Context, productID int64) error {
	log.Printf("[DEBUG] Repository: Deleting product %d", productID)

	if productID <= 0 {
		return fmt.Errorf("%w: product ID must be positive", ErrInvalidProductID)
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

	query := `DELETE FROM products.products WHERE id = $1`
	result, err := tx.ExecContext(ctx, query, productID)
	if err != nil {
		return fmt.Errorf("%w: failed to delete product: %v", ErrDatabaseOperation, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: failed to get rows affected: %v", ErrDatabaseOperation, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("%w: product %d not found", ErrProductNotFound, productID)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}
