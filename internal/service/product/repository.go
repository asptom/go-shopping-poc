// Package product provides data access operations for product entities.
//
// This package implements the repository pattern for product domain objects,
// handling database operations including CRUD operations, transactions, and
// event publishing through the outbox pattern.
//
// Note: This repository now uses direct sqlx.DB operations instead of the
// database.Database abstraction layer to remove overhead and improve performance.
package product

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// Custom error types for different repository failure scenarios
var (
	ErrProductNotFound      = errors.New("product not found")
	ErrProductImageNotFound = errors.New("product image not found")
	ErrInvalidProductID     = errors.New("invalid product ID")
	ErrDatabaseOperation    = errors.New("database operation failed")
	ErrTransactionFailed    = errors.New("transaction failed")
	ErrDuplicateImage       = errors.New("duplicate image URL for product")
)

// ProductRepository defines the contract for product data access operations.
//
// This interface abstracts database operations for product entities,
// providing a clean separation between business logic and data persistence.
// All methods accept a context for proper request tracing and cancellation.
type ProductRepository interface {
	// Product CRUD operations
	InsertProduct(ctx context.Context, product *Product) error
	GetProductByID(ctx context.Context, productID int64) (*Product, error)
	UpdateProduct(ctx context.Context, product *Product) error
	DeleteProduct(ctx context.Context, productID int64) error

	// Product queries
	GetProductsByCategory(ctx context.Context, category string, limit, offset int) ([]*Product, error)
	GetProductsByBrand(ctx context.Context, brand string, limit, offset int) ([]*Product, error)
	SearchProducts(ctx context.Context, query string, limit, offset int) ([]*Product, error)
	GetProductsInStock(ctx context.Context, limit, offset int) ([]*Product, error)

	// Product image operations
	AddProductImage(ctx context.Context, image *ProductImage) error
	UpdateProductImage(ctx context.Context, image *ProductImage) error
	DeleteProductImage(ctx context.Context, imageID int64) error
	GetProductImages(ctx context.Context, productID int64) ([]ProductImage, error)
	SetMainImage(ctx context.Context, productID int64, imageID int64) error

	// Bulk operations
	BulkInsertProducts(ctx context.Context, products []*Product) error
	BulkInsertProductImages(ctx context.Context, images []*ProductImage) error
}

// productRepository implements ProductRepository using direct sqlx.DB operations.
//
// This struct provides the concrete implementation of product data access
// operations using direct sqlx.DB for database interactions.
type productRepository struct {
	db *sqlx.DB
}

// NewProductRepository creates a new product repository instance.
//
// Parameters:
//   - db: Database connection using sqlx.DB
//
// Returns a configured product repository ready for use.
func NewProductRepository(db *sqlx.DB) ProductRepository {
	return &productRepository{db: db}
}

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

	// Prepare product with defaults
	r.prepareProductDefaults(product)

	// Validate product before insertion
	if err := product.Validate(); err != nil {
		return fmt.Errorf("product validation failed: %w", err)
	}

	// Insert product and related images in a transaction
	return r.insertProductWithImages(ctx, product)
}

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
		product.InStock = true // Default to in stock
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

	// Insert basic product record
	if err := r.insertProductRecord(ctx, tx, product); err != nil {
		return err
	}

	// Insert product images if provided
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
	query := `
		INSERT INTO products.products (
			id, name, description, initial_price, final_price, currency, in_stock,
			color, size, main_image, country_code, image_count, model_number,
			root_category, category, brand, other_attributes, all_available_sizes,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		) ON CONFLICT (id) DO NOTHING`

	_, err := tx.ExecContext(ctx, query,
		product.ID, product.Name, product.Description, product.InitialPrice, product.FinalPrice,
		product.Currency, product.InStock, product.Color, product.Size, product.MainImage,
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
	query := `
		INSERT INTO products.product_images (
			product_id, image_url, minio_object_name, is_main, image_order,
			file_size, content_type, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)`

	for _, image := range images {
		// Validate image before insertion
		if err := image.Validate(); err != nil {
			return fmt.Errorf("image validation failed: %w", err)
		}

		_, err := tx.ExecContext(ctx, query,
			image.ProductID, image.ImageURL, image.MinioObjectName, image.IsMain,
			image.ImageOrder, image.FileSize, image.ContentType, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("%w: failed to insert product image: %v", ErrDatabaseOperation, err)
		}
	}
	return nil
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

	// Load associated images
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

	// Validate product
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

	// Use transaction to ensure atomicity (cascade delete will handle images)
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

// GetProductsByCategory retrieves products by category with pagination
func (r *productRepository) GetProductsByCategory(ctx context.Context, category string, limit, offset int) ([]*Product, error) {
	log.Printf("[DEBUG] Repository: Fetching products by category: %s", category)

	if category == "" {
		return nil, errors.New("category cannot be empty")
	}
	if limit <= 0 {
		limit = 50 // Default limit
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, name, description, initial_price, final_price, currency, in_stock,
			   color, size, main_image, country_code, image_count, model_number,
			   root_category, category, brand, other_attributes, all_available_sizes,
			   created_at, updated_at
		FROM products.products
		WHERE category = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, category, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to query products by category: %v", ErrDatabaseOperation, err)
	}
	defer func() { _ = rows.Close() }()

	var products []*Product
	for rows.Next() {
		var product Product
		err := rows.Scan(
			&product.ID, &product.Name, &product.Description, &product.InitialPrice, &product.FinalPrice,
			&product.Currency, &product.InStock, &product.Color, &product.Size, &product.MainImage,
			&product.CountryCode, &product.ImageCount, &product.ModelNumber, &product.RootCategory,
			&product.Category, &product.Brand, &product.OtherAttributes, &product.AllAvailableSizes,
			&product.CreatedAt, &product.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to scan product row: %v", ErrDatabaseOperation, err)
		}
		products = append(products, &product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: error iterating product rows: %v", ErrDatabaseOperation, err)
	}

	return products, nil
}

// GetProductsByBrand retrieves products by brand with pagination
func (r *productRepository) GetProductsByBrand(ctx context.Context, brand string, limit, offset int) ([]*Product, error) {
	log.Printf("[DEBUG] Repository: Fetching products by brand: %s", brand)

	if brand == "" {
		return nil, errors.New("brand cannot be empty")
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, name, description, initial_price, final_price, currency, in_stock,
			   color, size, main_image, country_code, image_count, model_number,
			   root_category, category, brand, other_attributes, all_available_sizes,
			   created_at, updated_at
		FROM products.products
		WHERE brand = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, brand, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to query products by brand: %v", ErrDatabaseOperation, err)
	}
	defer func() { _ = rows.Close() }()

	var products []*Product
	for rows.Next() {
		var product Product
		err := rows.Scan(
			&product.ID, &product.Name, &product.Description, &product.InitialPrice, &product.FinalPrice,
			&product.Currency, &product.InStock, &product.Color, &product.Size, &product.MainImage,
			&product.CountryCode, &product.ImageCount, &product.ModelNumber, &product.RootCategory,
			&product.Category, &product.Brand, &product.OtherAttributes, &product.AllAvailableSizes,
			&product.CreatedAt, &product.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to scan product row: %v", ErrDatabaseOperation, err)
		}
		products = append(products, &product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: error iterating product rows: %v", ErrDatabaseOperation, err)
	}

	return products, nil
}

// SearchProducts performs a text search on product names and descriptions
func (r *productRepository) SearchProducts(ctx context.Context, query string, limit, offset int) ([]*Product, error) {
	log.Printf("[DEBUG] Repository: Searching products with query: %s", query)

	if query == "" {
		return nil, errors.New("search query cannot be empty")
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	searchQuery := `
		SELECT id, name, description, initial_price, final_price, currency, in_stock,
			   color, size, main_image, country_code, image_count, model_number,
			   root_category, category, brand, other_attributes, all_available_sizes,
			   created_at, updated_at
		FROM products.products
		WHERE name ILIKE $1 OR description ILIKE $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	searchPattern := "%" + query + "%"
	rows, err := r.db.QueryContext(ctx, searchQuery, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to search products: %v", ErrDatabaseOperation, err)
	}
	defer func() { _ = rows.Close() }()

	var products []*Product
	for rows.Next() {
		var product Product
		err := rows.Scan(
			&product.ID, &product.Name, &product.Description, &product.InitialPrice, &product.FinalPrice,
			&product.Currency, &product.InStock, &product.Color, &product.Size, &product.MainImage,
			&product.CountryCode, &product.ImageCount, &product.ModelNumber, &product.RootCategory,
			&product.Category, &product.Brand, &product.OtherAttributes, &product.AllAvailableSizes,
			&product.CreatedAt, &product.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to scan product row: %v", ErrDatabaseOperation, err)
		}
		products = append(products, &product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: error iterating product rows: %v", ErrDatabaseOperation, err)
	}

	return products, nil
}

// GetProductsInStock retrieves products that are currently in stock
func (r *productRepository) GetProductsInStock(ctx context.Context, limit, offset int) ([]*Product, error) {
	log.Printf("[DEBUG] Repository: Fetching in-stock products")

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, name, description, initial_price, final_price, currency, in_stock,
			   color, size, main_image, country_code, image_count, model_number,
			   root_category, category, brand, other_attributes, all_available_sizes,
			   created_at, updated_at
		FROM products.products
		WHERE in_stock = true
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to query in-stock products: %v", ErrDatabaseOperation, err)
	}
	defer func() { _ = rows.Close() }()

	var products []*Product
	for rows.Next() {
		var product Product
		err := rows.Scan(
			&product.ID, &product.Name, &product.Description, &product.InitialPrice, &product.FinalPrice,
			&product.Currency, &product.InStock, &product.Color, &product.Size, &product.MainImage,
			&product.CountryCode, &product.ImageCount, &product.ModelNumber, &product.RootCategory,
			&product.Category, &product.Brand, &product.OtherAttributes, &product.AllAvailableSizes,
			&product.CreatedAt, &product.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to scan product row: %v", ErrDatabaseOperation, err)
		}
		products = append(products, &product)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: error iterating product rows: %v", ErrDatabaseOperation, err)
	}

	return products, nil
}

// AddProductImage adds a new image to a product
func (r *productRepository) AddProductImage(ctx context.Context, image *ProductImage) error {
	log.Printf("[DEBUG] Repository: Adding image for product %d", image.ProductID)

	// Validate image
	if err := image.Validate(); err != nil {
		return fmt.Errorf("image validation failed: %w", err)
	}

	// Check if product exists
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
		image.ImageOrder, image.FileSize, image.ContentType, time.Now(),
	)
	if err != nil {
		// Check for duplicate key violation (unique constraint on product_id, image_url)
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
		return errors.New("image ID must be positive")
	}

	// Validate image
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
		return errors.New("image ID must be positive")
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
		return errors.New("product ID and image ID must be positive")
	}

	// Use transaction to ensure consistency
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

	// First, unset all main images for this product
	unsetQuery := `UPDATE products.product_images SET is_main = false WHERE product_id = $1`
	_, err = tx.ExecContext(ctx, unsetQuery, productID)
	if err != nil {
		return fmt.Errorf("%w: failed to unset main images: %v", ErrDatabaseOperation, err)
	}

	// Then, set the specified image as main
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

	// Update the product's main_image field
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

// insertProductImageRecord inserts a single product image record within a transaction
func (r *productRepository) insertProductImageRecord(ctx context.Context, tx *sqlx.Tx, image *ProductImage) error {
	query := `
		INSERT INTO products.product_images (
			product_id, image_url, minio_object_name, is_main, image_order,
			file_size, content_type, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)`

	_, err := tx.ExecContext(ctx, query,
		image.ProductID, image.ImageURL, image.MinioObjectName, image.IsMain,
		image.ImageOrder, image.FileSize, image.ContentType, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("%w: failed to insert product image: %v", ErrDatabaseOperation, err)
	}
	return nil
}

// isDuplicateError checks if an error is a duplicate key violation
func isDuplicateError(err error) bool {
	// This is a simplified check - in a real implementation, you'd check for specific
	// PostgreSQL error codes (e.g., "23505" for unique_violation)
	errStr := fmt.Sprintf("%v", err)
	return err != nil && !errors.Is(err, sql.ErrNoRows) &&
		(strings.Contains(errStr, "duplicate") ||
			strings.Contains(errStr, "unique"))
}
