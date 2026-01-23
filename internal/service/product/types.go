package product

import (
	"time"

	"go-shopping-poc/internal/platform/database"
)

// ProductIngestionRequest represents a request to ingest products from CSV
type ProductIngestionRequest struct {
	CSVPath    string `json:"csv_path" validate:"required"`
	BatchID    string `json:"batch_id,omitempty"`
	UseCache   bool   `json:"use_cache"`
	ResetCache bool   `json:"reset_cache"`
}

// ProductIngestionResult represents the result of a product ingestion operation
type ProductIngestionResult struct {
	BatchID           string    `json:"batch_id"`
	TotalProducts     int       `json:"total_products"`
	ProcessedProducts int       `json:"processed_products"`
	TotalImages       int       `json:"total_images"`
	SuccessfulImages  int       `json:"successful_images"`
	FailedProducts    int       `json:"failed_products"`
	FailedImages      int       `json:"failed_images"`
	StartTime         time.Time `json:"start_time"`
	EndTime           time.Time `json:"end_time"`
	Duration          string    `json:"duration"`
	Errors            []string  `json:"errors,omitempty"`
}

// ProductCSVRecord represents a single product record from CSV
type ProductCSVRecord struct {
	ID                string        `csv:"product_id"`
	Name              string        `csv:"product_name"`
	Description       string        `csv:"description"`
	InitialPrice      float64       `csv:"initial_price"`
	FinalPrice        float64       `csv:"final_price"`
	Currency          string        `csv:"currency"`
	InStock           string        `csv:"in_stock"`
	Color             string        `csv:"color"`
	Size              string        `csv:"size"`
	MainImage         string        `csv:"main_image"`
	CountryCode       string        `csv:"country_code"`
	ImageCount        string        `csv:"image_count"`
	ModelNumber       string        `csv:"model_number"`
	RootCategory      string        `csv:"root_category"`
	Category          string        `csv:"category"`
	Brand             string        `csv:"brand"`
	AllAvailableSizes database.JSON `csv:"all_available_sizes"`
	ImageURLs         []string      `csv:"image_urls"`
	OtherAttributes   string        `csv:"other_attributes"`
}
