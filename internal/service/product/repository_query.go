// Package product provides data access operations for product entities.
//
// This file contains query operations for searching and filtering products,
// including category, brand, search, and in-stock queries.
package product

import (
	"context"
	"errors"
	"fmt"
	"log"
)

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
			   color, size, main_image, country_code, image_count, model_number,
			   root_category, category, brand, other_attributes, all_available_sizes,
			   created_at, updated_at
		FROM products.products
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to query products: %v", ErrDatabaseOperation, err)
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
