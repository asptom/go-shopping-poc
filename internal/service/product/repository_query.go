// Package product provides data access operations for product entities.
//
// This file contains query operations for searching and filtering products,
// including category, brand, search, and in-stock queries.
package product

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
)

// ProductExists checks if a product with the given ID exists
func (r *productRepository) ProductExists(ctx context.Context, productID int64) (bool, error) {
	log.Printf("[DEBUG] Repository: Checking existence of product %d", productID)

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM products.products WHERE id = $1)`

	err := r.db.QueryRow(ctx, query, productID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%w: failed to check product existence: %v", ErrDatabaseOperation, err)
	}

	return exists, nil
}

// GetProductByID retrieves a product by its ID with all associated images
func (r *productRepository) GetProductByID(ctx context.Context, productID int64) (*Product, error) {
	log.Printf("[DEBUG] Repository: Fetching product by ID: %d", productID)

	if productID <= 0 {
		return nil, fmt.Errorf("%w: product ID must be positive", ErrInvalidProductID)
	}

	query := `
		SELECT id, name, description, initial_price, final_price, currency, in_stock,
			   color, size, country_code, image_count, model_number,
			   root_category, category, brand, other_attributes, all_available_sizes,
			   created_at, updated_at
		FROM products.products WHERE id = $1`

	var product Product

	err := r.db.QueryRow(ctx, query, productID).Scan(
		&product.ID, &product.Name, &product.Description, &product.InitialPrice, &product.FinalPrice,
		&product.Currency, &product.InStock, &product.Color, &product.Size,
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

// GetProductsByCategory retrieves products by category with pagination
func (r *productRepository) GetProductsByCategory(ctx context.Context, category string, limit, offset int) ([]*Product, error) {
	log.Printf("[DEBUG] Repository: Fetching products by category: %s", category)

	if category == "" {
		return nil, errors.New("category cannot be empty")
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, name, description, initial_price, final_price, currency, in_stock,
			   color, size, country_code, image_count, model_number,
			   root_category, category, brand, other_attributes, all_available_sizes,
			   created_at, updated_at
		FROM products.products
		WHERE category = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, category, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to query products by category: %v", ErrDatabaseOperation, err)
	}
	defer func() { _ = rows.Close() }()

	var products []*Product
	for rows.Next() {
		var product Product

		err := rows.Scan(
			&product.ID, &product.Name, &product.Description, &product.InitialPrice, &product.FinalPrice,
			&product.Currency, &product.InStock, &product.Color, &product.Size,
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
			   color, size, country_code, image_count, model_number,
			   root_category, category, brand, other_attributes, all_available_sizes,
			   created_at, updated_at
		FROM products.products
		WHERE brand = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, brand, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to query products by brand: %v", ErrDatabaseOperation, err)
	}
	defer func() { _ = rows.Close() }()

	var products []*Product
	for rows.Next() {
		var product Product

		err := rows.Scan(
			&product.ID, &product.Name, &product.Description, &product.InitialPrice, &product.FinalPrice,
			&product.Currency, &product.InStock, &product.Color, &product.Size,
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
			   color, size, country_code, image_count, model_number,
			   root_category, category, brand, other_attributes, all_available_sizes,
			   created_at, updated_at
		FROM products.products
		WHERE name ILIKE $1 OR description ILIKE $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	searchPattern := "%" + query + "%"
	rows, err := r.db.Query(ctx, searchQuery, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to search products: %v", ErrDatabaseOperation, err)
	}
	defer func() { _ = rows.Close() }()

	var products []*Product
	for rows.Next() {
		var product Product

		err := rows.Scan(
			&product.ID, &product.Name, &product.Description, &product.InitialPrice, &product.FinalPrice,
			&product.Currency, &product.InStock, &product.Color, &product.Size,
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
			   color, size, country_code, image_count, model_number,
			   root_category, category, brand, other_attributes, all_available_sizes,
			   created_at, updated_at
		FROM products.products
		WHERE in_stock = true
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to query in-stock products: %v", ErrDatabaseOperation, err)
	}
	defer func() { _ = rows.Close() }()

	var products []*Product
	for rows.Next() {
		var product Product

		err := rows.Scan(
			&product.ID, &product.Name, &product.Description, &product.InitialPrice, &product.FinalPrice,
			&product.Currency, &product.InStock, &product.Color, &product.Size,
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

// GetAllProducts retrieves all products with pagination
func (r *productRepository) GetAllProducts(ctx context.Context, limit, offset int) ([]*Product, error) {
	log.Printf("[DEBUG] Repository: Fetching all products")

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, name, description, initial_price, final_price, currency, in_stock,
			   color, size, country_code, image_count, model_number,
			   root_category, category, brand, other_attributes, all_available_sizes,
			   created_at, updated_at
		FROM products.products
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to query products: %v", ErrDatabaseOperation, err)
	}
	defer func() { _ = rows.Close() }()

	var products []*Product
	for rows.Next() {
		var product Product

		err := rows.Scan(
			&product.ID, &product.Name, &product.Description, &product.InitialPrice, &product.FinalPrice,
			&product.Currency, &product.InStock, &product.Color, &product.Size,
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

// GetProductImages retrieves all images for a product
func (r *productRepository) GetProductImages(ctx context.Context, productID int64) ([]ProductImage, error) {
	log.Printf("[DEBUG] Repository: Fetching images for product %d", productID)

	if productID <= 0 {
		return nil, fmt.Errorf("%w: product ID must be positive", ErrInvalidProductID)
	}

	query := `
		SELECT id, product_id, minio_object_name, is_main, image_order,
			   file_size, content_type, created_at
		FROM products.product_images
		WHERE product_id = $1
		ORDER BY image_order, created_at`

	rows, err := r.db.Query(ctx, query, productID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to query product images: %v", ErrDatabaseOperation, err)
	}
	defer func() { _ = rows.Close() }()

	var images []ProductImage
	for rows.Next() {
		var image ProductImage

		err := rows.Scan(
			&image.ID, &image.ProductID, &image.MinioObjectName,
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

// GetProductImageByID retrieves a single image by its ID
func (r *productRepository) GetProductImageByID(ctx context.Context, imageID int64) (*ProductImage, error) {
	log.Printf("[DEBUG] Repository: Fetching image %d", imageID)

	if imageID <= 0 {
		return nil, fmt.Errorf("%w: image ID must be positive", ErrInvalidProductID)
	}

	query := `
		SELECT id, product_id, minio_object_name, is_main, image_order,
			   file_size, content_type, created_at
		FROM products.product_images
		WHERE id = $1`

	var image ProductImage
	err := r.db.QueryRow(ctx, query, imageID).Scan(
		&image.ID, &image.ProductID, &image.MinioObjectName,
		&image.IsMain, &image.ImageOrder, &image.FileSize, &image.ContentType, &image.CreatedAt,
	)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, fmt.Errorf("%w: image %d not found", ErrProductImageNotFound, imageID)
		}
		return nil, fmt.Errorf("%w: failed to query product image: %v", ErrDatabaseOperation, err)
	}

	return &image, nil
}
