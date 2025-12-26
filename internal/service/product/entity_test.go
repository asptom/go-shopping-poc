package product

import (
	"testing"
	"time"
)

func TestProduct_Validate(t *testing.T) {
	tests := []struct {
		name    string
		product Product
		wantErr bool
	}{
		{
			name: "valid product",
			product: Product{
				ID:           1,
				Name:         "Test Product",
				InitialPrice: 100.0,
				FinalPrice:   90.0,
				Currency:     "USD",
				InStock:      true,
				CountryCode:  "US",
				ImageCount:   2,
			},
			wantErr: false,
		},
		{
			name: "missing name",
			product: Product{
				ID:           1,
				InitialPrice: 100.0,
				FinalPrice:   90.0,
			},
			wantErr: true,
		},
		{
			name: "negative initial price",
			product: Product{
				ID:           1,
				Name:         "Test Product",
				InitialPrice: -10.0,
				FinalPrice:   90.0,
			},
			wantErr: true,
		},
		{
			name: "final price higher than initial",
			product: Product{
				ID:           1,
				Name:         "Test Product",
				InitialPrice: 90.0,
				FinalPrice:   100.0,
			},
			wantErr: true,
		},
		{
			name: "invalid currency code",
			product: Product{
				ID:           1,
				Name:         "Test Product",
				InitialPrice: 100.0,
				FinalPrice:   90.0,
				Currency:     "INVALID",
			},
			wantErr: true,
		},
		{
			name: "invalid country code",
			product: Product{
				ID:           1,
				Name:         "Test Product",
				InitialPrice: 100.0,
				FinalPrice:   90.0,
				CountryCode:  "USA",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.product.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Product.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProduct_IsOnSale(t *testing.T) {
	tests := []struct {
		name     string
		product  Product
		expected bool
	}{
		{
			name: "on sale",
			product: Product{
				InitialPrice: 100.0,
				FinalPrice:   80.0,
			},
			expected: true,
		},
		{
			name: "not on sale - same price",
			product: Product{
				InitialPrice: 100.0,
				FinalPrice:   100.0,
			},
			expected: false,
		},
		{
			name: "not on sale - zero initial price",
			product: Product{
				InitialPrice: 0.0,
				FinalPrice:   80.0,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.product.IsOnSale()
			if result != tt.expected {
				t.Errorf("Product.IsOnSale() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestProductImage_Validate(t *testing.T) {
	tests := []struct {
		name         string
		productImage ProductImage
		wantErr      bool
	}{
		{
			name: "valid product image",
			productImage: ProductImage{
				ID:              1,
				ProductID:       1,
				ImageURL:        "https://example.com/image.jpg",
				MinioObjectName: "product-1/image.jpg",
				IsMain:          true,
				ImageOrder:      1,
				FileSize:        1024000,
				ContentType:     "image/jpeg",
				CreatedAt:       time.Now(),
			},
			wantErr: false,
		},
		{
			name: "missing product ID",
			productImage: ProductImage{
				ImageURL: "https://example.com/image.jpg",
			},
			wantErr: true,
		},
		{
			name: "missing image URL",
			productImage: ProductImage{
				ProductID: 1,
			},
			wantErr: true,
		},
		{
			name: "negative image order",
			productImage: ProductImage{
				ProductID:  1,
				ImageURL:   "https://example.com/image.jpg",
				ImageOrder: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.productImage.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ProductImage.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProductImage_IsImage(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		expected    bool
	}{
		{"jpeg image", "image/jpeg", true},
		{"png image", "image/png", true},
		{"gif image", "image/gif", true},
		{"text file", "text/plain", false},
		{"json file", "application/json", false},
		{"empty content type", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pi := ProductImage{ContentType: tt.contentType}
			result := pi.IsImage()
			if result != tt.expected {
				t.Errorf("ProductImage.IsImage() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
