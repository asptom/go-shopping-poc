package product

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"go-shopping-poc/internal/platform/database"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: Config{
				DatabaseURL: "postgres://user:pass@localhost/db",
				ServicePort: "8080",
				CacheDir:    "/tmp/cache",
				MinIOBucket: "productimages",
			},
			wantErr: false,
		},
		{
			name: "missing database URL",
			config: Config{
				ServicePort: "8080",
				MinIOBucket: "productimages",
			},
			wantErr: true,
			errMsg:  "database URL is required",
		},
		{
			name: "missing service port",
			config: Config{
				DatabaseURL: "postgres://user:pass@localhost/db",
				MinIOBucket: "productimages",
			},
			wantErr: true,
			errMsg:  "service port is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Config.Validate() expected error, got nil")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Config.Validate() error = %v, want %v", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Config.Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestFilterImageURLs(t *testing.T) {
	service := &ProductService{infrastructure: &ProductInfrastructure{}} // Create minimal service for testing

	urls := []string{
		"http://example.com/image.jpg",
		"http://example.com/image.jpeg",
		"http://example.com/image.png",
		"http://example.com/image.gif",
		"http://example.com/image.webp",
		"http://example.com/document.pdf",
		"http://example.com/image",
		"http://example.com/image.txt",
	}

	filtered := service.filterImageURLs(urls)

	expected := []string{
		"http://example.com/image.jpg",
		"http://example.com/image.jpeg",
		"http://example.com/image.png",
		"http://example.com/image.gif",
		"http://example.com/image.webp",
	}

	if len(filtered) != len(expected) {
		t.Errorf("filterImageURLs() returned %d URLs, expected %d", len(filtered), len(expected))
	}

	for i, url := range expected {
		if i >= len(filtered) || filtered[i] != url {
			t.Errorf("filterImageURLs()[%d] = %v, want %v", i, filtered[i], url)
		}
	}
}

func TestGetContentTypeFromPath(t *testing.T) {
	service := &ProductService{infrastructure: &ProductInfrastructure{}} // Create minimal service for testing

	tests := []struct {
		path     string
		expected string
	}{
		{"/path/to/image.jpg", "image/jpeg"},
		{"/path/to/image.jpeg", "image/jpeg"},
		{"/path/to/image.png", "image/png"},
		{"/path/to/image.gif", "image/gif"},
		{"/path/to/image.webp", "image/webp"},
		{"/path/to/image.unknown", "image/jpeg"}, // Default
		{"/path/to/image", "image/jpeg"},         // Default
	}

	for _, test := range tests {
		result := service.getContentTypeFromPath(test.path)
		if result != test.expected {
			t.Errorf("getContentTypeFromPath(%s) = %v, want %v", test.path, result, test.expected)
		}
	}
}

func TestConvertCSVRecordToProduct(t *testing.T) {
	service := &ProductService{infrastructure: &ProductInfrastructure{}} // Create minimal service for testing

	record := ProductCSVRecord{
		Name:              "Test Product",
		Description:       "Test Description",
		InitialPrice:      100.0,
		FinalPrice:        90.0,
		Currency:          "USD",
		InStock:           "true",
		Color:             "Red",
		Size:              "M",
		MainImage:         "http://example.com/main.jpg",
		CountryCode:       "US",
		ImageCount:        "2",
		ModelNumber:       "TEST123",
		RootCategory:      "Clothing",
		Category:          "T-Shirts",
		Brand:             "Test Brand",
		AllAvailableSizes: database.JSON{Data: []string{"S", "M", "L"}},
		ImageURLs:         []string{"http://example.com/image1.jpg", "http://example.com/image2.png"},
	}

	product, err := service.convertCSVRecordToProduct(record)
	if err != nil {
		t.Fatalf("convertCSVRecordToProduct() error = %v", err)
	}

	if product.Name != "Test Product" {
		t.Errorf("Name = %v, want %v", product.Name, "Test Product")
	}
	if product.InitialPrice != 100.0 {
		t.Errorf("InitialPrice = %v, want %v", product.InitialPrice, 100.0)
	}
	if product.FinalPrice != 90.0 {
		t.Errorf("FinalPrice = %v, want %v", product.FinalPrice, 90.0)
	}
	if product.Currency != "USD" {
		t.Errorf("Currency = %v, want %v", product.Currency, "USD")
	}
	if !product.InStock {
		t.Errorf("InStock = %v, want %v", product.InStock, true)
	}
	if product.Color != "Red" {
		t.Errorf("Color = %v, want %v", product.Color, "Red")
	}
	if product.ImageCount != 2 {
		t.Errorf("ImageCount = %v, want %v", product.ImageCount, 2)
	}
	if sizes, ok := product.AllAvailableSizes.Data.([]string); !ok || len(sizes) != 3 {
		t.Errorf("AllAvailableSizes length = %v, want %v", len(sizes), 3)
	}
	if len(product.ImageURLs) != 2 {
		t.Errorf("ImageURLs length = %v, want %v", len(product.ImageURLs), 2)
	}
}

func TestIngestProductsFromCSV_InvalidCSV(t *testing.T) {
	service := &ProductService{infrastructure: &ProductInfrastructure{}} // Create minimal service for testing

	req := &ProductIngestionRequest{
		CSVPath: "/nonexistent/file.csv",
	}

	result, err := service.IngestProductsFromCSV(context.TODO(), req)

	if err == nil {
		t.Error("IngestProductsFromCSV() expected error for nonexistent file")
	}
	if result == nil {
		t.Error("IngestProductsFromCSV() should return result even on error")
	} else if len(result.Errors) == 0 {
		t.Error("IngestProductsFromCSV() should populate errors on failure")
	}
}

func TestIngestProductsFromCSV_ValidCSV(t *testing.T) {
	// Create temporary CSV file
	csvContent := `product_id,product_name,description,initial_price,final_price,currency,in_stock,color,size,main_image,country_code,image_count,model_number,root_category,category,brand,all_available_sizes,image_urls,other_attributes
1,Test Product,Test Description,100.0,90.0,USD,true,Red,M,http://example.com/image.jpg,US,1,TEST123,Clothing,T-Shirts,Test Brand,"[""S"",""M"",""L""]","http://example.com/image1.jpg","{}"`

	tmpDir, err := os.MkdirTemp("", "product_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	csvPath := filepath.Join(tmpDir, "test.csv")
	err = os.WriteFile(csvPath, []byte(csvContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write CSV file: %v", err)
	}

	// Create a service with minimal config to avoid nil pointer dereference
	config := &Config{
		CacheDir:     tmpDir,
		CSVBatchSize: 10,
	}

	service := &ProductService{
		config:         config,
		infrastructure: &ProductInfrastructure{},
	} // Create minimal service for testing

	req := &ProductIngestionRequest{
		CSVPath:  csvPath,
		BatchID:  "test-batch",
		UseCache: false,
	}

	// This should succeed in parsing the CSV and processing products (even with missing dependencies)
	result, err := service.IngestProductsFromCSV(context.TODO(), req)

	// We expect success for CSV parsing, but some operations may fail due to missing dependencies
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result == nil {
		t.Error("Should return result")
	} else {
		if result.TotalProducts != 1 {
			t.Errorf("TotalProducts = %d, want 1", result.TotalProducts)
		}
	}
	if result.ProcessedProducts != 1 {
		t.Errorf("ProcessedProducts = %d, want 1", result.ProcessedProducts)
	}
	if result.BatchID != "test-batch" {
		t.Errorf("BatchID = %s, want test-batch", result.BatchID)
	}
	if result.TotalImages != 1 {
		t.Errorf("TotalImages = %d, want 1", result.TotalImages)
	}
	// Note: SuccessfulImages may be 0 due to missing dependencies, which is expected
}
