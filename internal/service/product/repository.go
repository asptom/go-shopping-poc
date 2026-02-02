// Package product provides data access operations for product entities.
//
// This package implements the repository pattern for product domain objects,
// handling database operations including CRUD operations, transactions, and
// event publishing through the outbox pattern.
//
// The repository is split into multiple files:
// - repository.go: Interface and struct definitions
// - repository_crud.go: CRUD operations
// - repository_query.go: Query operations
// - repository_image.go: Image operations
// - repository_bulk.go: Bulk operations
// - repository_util.go: Utility functions
package product

import (
	"context"
	"errors"

	"github.com/jmoiron/sqlx"
)

var (
	ErrProductNotFound      = errors.New("product not found")
	ErrProductImageNotFound = errors.New("product image not found")
	ErrInvalidProductID     = errors.New("invalid product ID")
	ErrDatabaseOperation    = errors.New("database operation failed")
	ErrTransactionFailed    = errors.New("transaction failed")
	ErrDuplicateImage       = errors.New("duplicate image URL for product")
)

type ProductRepository interface {
	InsertProduct(ctx context.Context, product *Product) error
	GetProductByID(ctx context.Context, productID int64) (*Product, error)
	UpdateProduct(ctx context.Context, product *Product) error
	DeleteProduct(ctx context.Context, productID int64) error

	GetProductsByCategory(ctx context.Context, category string, limit, offset int) ([]*Product, error)
	GetProductsByBrand(ctx context.Context, brand string, limit, offset int) ([]*Product, error)
	SearchProducts(ctx context.Context, query string, limit, offset int) ([]*Product, error)
	GetProductsInStock(ctx context.Context, limit, offset int) ([]*Product, error)
	GetAllProducts(ctx context.Context, limit, offset int) ([]*Product, error)

	AddProductImage(ctx context.Context, image *ProductImage) error
	UpdateProductImage(ctx context.Context, image *ProductImage) error
	DeleteProductImage(ctx context.Context, imageID int64) error
	GetProductImages(ctx context.Context, productID int64) ([]ProductImage, error)
	GetProductImageByID(ctx context.Context, imageID int64) (*ProductImage, error)
	// UPDATED: SetMainImage renamed to SetMainImageFlag - no longer updates products table
	SetMainImageFlag(ctx context.Context, productID int64, imageID int64) error

	BulkInsertProducts(ctx context.Context, products []*Product) error
	BulkInsertProductImages(ctx context.Context, images []*ProductImage) error
}

type productRepository struct {
	db *sqlx.DB
}

func NewProductRepository(db *sqlx.DB) ProductRepository {
	return &productRepository{db: db}
}
