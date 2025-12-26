// Package product defines the domain entities for the product bounded context.
//
// This package contains the core business objects (entities) that represent
// products, product images, and related domain concepts. Entities
// include validation methods and business logic.
package product

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go-shopping-poc/internal/platform/database"
)

// Product represents a product entity in the e-commerce system.
type Product struct {
	ID                int64         `json:"id" db:"id"`
	Name              string        `json:"name" db:"name"`
	Description       string        `json:"description" db:"description"`
	InitialPrice      float64       `json:"initial_price" db:"initial_price"`
	FinalPrice        float64       `json:"final_price" db:"final_price"`
	Currency          string        `json:"currency" db:"currency"`
	InStock           bool          `json:"in_stock" db:"in_stock"`
	Color             string        `json:"color" db:"color"`
	Size              string        `json:"size" db:"size"`
	MainImage         string        `json:"main_image" db:"main_image"`
	CountryCode       string        `json:"country_code" db:"country_code"`
	ImageCount        int           `json:"image_count" db:"image_count"`
	ModelNumber       string        `json:"model_number" db:"model_number"`
	OtherAttributes   string        `json:"other_attributes" db:"other_attributes"`
	RootCategory      string        `json:"root_category" db:"root_category"`
	Category          string        `json:"category" db:"category"`
	Brand             string        `json:"brand" db:"brand"`
	AllAvailableSizes database.JSON `json:"all_available_sizes" db:"all_available_sizes"`
	CreatedAt         time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at" db:"updated_at"`

	// Domain relationships (not persisted)
	Images []ProductImage `json:"images,omitempty"`

	// Temporary fields for ingestion (not persisted)
	ImageURLs []string `json:"image_urls,omitempty"`
}

// Validate performs domain validation on the Product entity
func (p *Product) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("product name is required")
	}
	if len(p.Name) > 500 {
		return errors.New("product name must be 500 characters or less")
	}

	if p.InitialPrice < 0 {
		return errors.New("initial price cannot be negative")
	}
	if p.FinalPrice < 0 {
		return errors.New("final price cannot be negative")
	}
	if p.FinalPrice > p.InitialPrice {
		return errors.New("final price cannot be greater than initial price")
	}

	if p.Currency != "" && len(p.Currency) != 3 {
		return errors.New("currency must be a 3-letter ISO code")
	}

	if p.CountryCode != "" && len(p.CountryCode) != 2 {
		return errors.New("country code must be a 2-letter ISO code")
	}

	if p.ModelNumber != "" && len(p.ModelNumber) > 100 {
		return errors.New("model number must be 100 characters or less")
	}

	if p.RootCategory != "" && len(p.RootCategory) > 100 {
		return errors.New("root category must be 100 characters or less")
	}

	if p.Category != "" && len(p.Category) > 100 {
		return errors.New("category must be 100 characters or less")
	}

	if p.Brand != "" && len(p.Brand) > 100 {
		return errors.New("brand must be 100 characters or less")
	}

	if p.Color != "" && len(p.Color) > 100 {
		return errors.New("color must be 100 characters or less")
	}

	if p.Size != "" && len(p.Size) > 100 {
		return errors.New("size must be 100 characters or less")
	}

	if p.ImageCount < 0 {
		return errors.New("image count cannot be negative")
	}

	return nil
}

// IsOnSale returns true if the product has a discounted price
func (p *Product) IsOnSale() bool {
	return p.FinalPrice < p.InitialPrice && p.InitialPrice > 0
}

// DiscountPercentage returns the discount percentage if the product is on sale
func (p *Product) DiscountPercentage() float64 {
	if !p.IsOnSale() {
		return 0
	}
	return ((p.InitialPrice - p.FinalPrice) / p.InitialPrice) * 100
}

// FormattedPrice returns a formatted price string with currency
func (p *Product) FormattedPrice() string {
	currency := p.Currency
	if currency == "" {
		currency = "USD"
	}
	return fmt.Sprintf("%.2f %s", p.FinalPrice, currency)
}

// HasImages returns true if the product has associated images
func (p *Product) HasImages() bool {
	return p.ImageCount > 0 || len(p.Images) > 0
}

// GetMainImage returns the main image URL, checking both the field and images slice
func (p *Product) GetMainImage() string {
	if p.MainImage != "" {
		return p.MainImage
	}

	// Check images slice for main image
	for _, img := range p.Images {
		if img.IsMain {
			return img.ImageURL
		}
	}

	return ""
}

// IsAvailable returns true if the product is in stock
func (p *Product) IsAvailable() bool {
	return p.InStock
}

// GetDisplayName returns a formatted display name including brand and model
func (p *Product) GetDisplayName() string {
	parts := []string{}
	if p.Brand != "" {
		parts = append(parts, p.Brand)
	}
	parts = append(parts, p.Name)
	if p.ModelNumber != "" {
		parts = append(parts, fmt.Sprintf("(%s)", p.ModelNumber))
	}
	return strings.Join(parts, " ")
}

// GetOtherAttributesJSON returns the OtherAttributes field as parsed JSON data
func (p *Product) GetOtherAttributesJSON() (interface{}, error) {
	if p.OtherAttributes == "" {
		return nil, nil
	}
	var data interface{}
	if err := json.Unmarshal([]byte(p.OtherAttributes), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal other_attributes JSON: %w", err)
	}
	return data, nil
}

// SetOtherAttributesJSON sets the OtherAttributes field from JSON data
func (p *Product) SetOtherAttributesJSON(data interface{}) error {
	if data == nil {
		p.OtherAttributes = ""
		return nil
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal other_attributes JSON: %w", err)
	}
	p.OtherAttributes = string(jsonBytes)
	return nil
}

// ProductImage represents an image associated with a product
type ProductImage struct {
	ID              int64     `json:"id" db:"id"`
	ProductID       int64     `json:"product_id" db:"product_id"`
	ImageURL        string    `json:"image_url" db:"image_url"`
	MinioObjectName string    `json:"minio_object_name" db:"minio_object_name"`
	IsMain          bool      `json:"is_main" db:"is_main"`
	ImageOrder      int       `json:"image_order" db:"image_order"`
	FileSize        int64     `json:"file_size" db:"file_size"`
	ContentType     string    `json:"content_type" db:"content_type"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// Validate performs domain validation on the ProductImage entity
func (pi *ProductImage) Validate() error {
	if pi.ProductID <= 0 {
		return errors.New("product ID is required and must be positive")
	}

	if strings.TrimSpace(pi.ImageURL) == "" {
		return errors.New("image URL is required")
	}

	if len(pi.ImageURL) > 2000 {
		return errors.New("image URL must be 2000 characters or less")
	}

	if pi.MinioObjectName != "" && len(pi.MinioObjectName) > 500 {
		return errors.New("MinIO object name must be 500 characters or less")
	}

	if pi.ImageOrder < 0 {
		return errors.New("image order cannot be negative")
	}

	if pi.FileSize < 0 {
		return errors.New("file size cannot be negative")
	}

	if pi.ContentType != "" && len(pi.ContentType) > 100 {
		return errors.New("content type must be 100 characters or less")
	}

	return nil
}

// IsImage returns true if the content type indicates an image
func (pi *ProductImage) IsImage() bool {
	return strings.HasPrefix(pi.ContentType, "image/")
}

// FormattedFileSize returns a human-readable file size
func (pi *ProductImage) FormattedFileSize() string {
	const unit = 1024
	if pi.FileSize < unit {
		return fmt.Sprintf("%d B", pi.FileSize)
	}
	div, exp := int64(unit), 0
	for n := pi.FileSize / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(pi.FileSize)/float64(div), "KMGTPE"[exp])
}

// GetFileExtension returns the file extension from the content type
func (pi *ProductImage) GetFileExtension() string {
	switch pi.ContentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	default:
		return ""
	}
}
