// Package product provides data access operations for product entities.
//
// This file contains CRUD (Create, Read, Update, Delete) operations
// for product entities, handling database operations with proper transaction
// management and validation.
//

package product

import (
	"context"
	"fmt"
	"log"
	"time"

	events "go-shopping-poc/internal/contracts/events"

	"go-shopping-poc/internal/platform/database"
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

// insertProductWithImages handles the complete product creation process
func (r *productRepository) insertProductWithImages(ctx context.Context, product *Product) error {
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

	if err := r.insertProductRecord(ctx, tx, product); err != nil {
		return err
	}

	if len(product.Images) > 0 {
		if err := r.insertProductImages(ctx, tx, product.Images); err != nil {
			return err
		}
	}

	// Publish product created event to outbox
	evt := events.NewProductCreatedEvent(fmt.Sprintf("%d", product.ID), map[string]string{
		"name":   product.Name,
		"brand":  product.Brand,
		"price":  product.FormattedPrice(),
		"images": fmt.Sprintf("%d", len(product.Images)),
	})

	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return fmt.Errorf("%w: failed to write product created event: %v", ErrEventWriteFailed, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}

// insertProductRecord inserts the product record within a transaction
func (r *productRepository) insertProductRecord(ctx context.Context, tx database.Tx, product *Product) error {

	query := `
		INSERT INTO products.products (
			id, name, description, initial_price, final_price, currency, in_stock,
			color, size, country_code, image_count, model_number,
			root_category, category, brand, other_attributes, all_available_sizes,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
		) ON CONFLICT (id) DO NOTHING`

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

// UpdateProduct updates an existing product record
func (r *productRepository) UpdateProduct(ctx context.Context, product *Product) error {
	log.Printf("[DEBUG] Repository: Updating product %d", product.ID)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := product.Validate(); err != nil {
		return fmt.Errorf("product validation failed: %w", err)
	}

	if product.ID <= 0 {
		return fmt.Errorf("%w: product ID must be positive", ErrInvalidProductID)
	}

	product.UpdatedAt = time.Now()

	query := `
		UPDATE products.products SET
			name = $2, description = $3, initial_price = $4, final_price = $5,
			currency = $6, in_stock = $7, color = $8, size = $9,
			country_code = $10, image_count = $11, model_number = $12,
			root_category = $13, category = $14, brand = $15,
			other_attributes = $16, all_available_sizes = $17, updated_at = $18
		WHERE id = $1`

	result, err := tx.ExecContext(ctx, query,
		product.ID, product.Name, product.Description, product.InitialPrice, product.FinalPrice,
		product.Currency, product.InStock, product.Color, product.Size,
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

	// Publish product updated event to outbox
	evt := events.NewProductUpdatedEvent(fmt.Sprintf("%d", product.ID), map[string]string{
		"name":  product.Name,
		"brand": product.Brand,
	})

	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return fmt.Errorf("%w: failed to write product updated event: %v", ErrEventWriteFailed, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}

// DeleteProduct removes a product and all its associated images
func (r *productRepository) DeleteProduct(ctx context.Context, productID int64) error {
	log.Printf("[DEBUG] Repository: Deleting product %d", productID)

	if productID <= 0 {
		return fmt.Errorf("%w: product ID must be positive", ErrInvalidProductID)
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

	// Publish product deleted event to outbox
	evt := events.NewProductDeletedEvent(fmt.Sprintf("%d", productID), map[string]string{})

	if err := r.outboxWriter.WriteEvent(ctx, tx, evt); err != nil {
		return fmt.Errorf("%w: failed to write product deleted event: %v", ErrEventWriteFailed, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: failed to commit transaction: %v", ErrTransactionFailed, err)
	}
	committed = true

	return nil
}
