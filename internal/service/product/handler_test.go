package product

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestProductHandler_GetProduct_Validation(t *testing.T) {
	tests := []struct {
		name           string
		productID      string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "missing product ID",
			productID:      "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing product ID in path",
		},
		{
			name:           "invalid product ID format",
			productID:      "abc",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid product ID format",
		},
		{
			name:           "zero product ID",
			productID:      "0",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Product ID must be positive",
		},
		{
			name:           "negative product ID",
			productID:      "-1",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Product ID must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/products/"+tt.productID, nil)

			// Add chi URL param for valid cases
			if tt.productID != "" {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("id", tt.productID)
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			}

			w := httptest.NewRecorder()
			handler := &ProductHandler{}

			handler.GetProduct(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("Expected body to contain '%s', got '%s'", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestProductHandler_CreateProduct_Validation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid JSON in request body",
		},
		{
			name: "missing name",
			requestBody: Product{
				Description:  "A product without name",
				InitialPrice: 29.99,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "product name is required",
		},
		{
			name: "invalid price",
			requestBody: Product{
				Name:         "Product with invalid price",
				InitialPrice: -10.0,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "initial price must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			if str, ok := tt.requestBody.(string); ok {
				body.WriteString(str)
			} else {
				require.NoError(t, json.NewEncoder(&body).Encode(tt.requestBody))
			}

			req := httptest.NewRequest(http.MethodPost, "/products", &body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler := &ProductHandler{}
			handler.CreateProduct(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("Expected body to contain '%s', got '%s'", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestProductHandler_UpdateProduct_Validation(t *testing.T) {
	tests := []struct {
		name           string
		productID      string
		requestBody    interface{}
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "missing product ID",
			productID:      "",
			requestBody:    Product{Name: "Test"},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing product ID in path",
		},
		{
			name:           "invalid product ID format",
			productID:      "abc",
			requestBody:    Product{Name: "Test"},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid product ID format",
		},
		{
			name:      "PUT with missing name",
			productID: "123",
			requestBody: Product{
				InitialPrice: 29.99,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "PUT requires complete product record with name",
		},
		{
			name:      "PUT with invalid price",
			productID: "123",
			requestBody: Product{
				Name:         "Test Product",
				InitialPrice: 0,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "PUT requires complete product record with valid initial price",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			require.NoError(t, json.NewEncoder(&body).Encode(tt.requestBody))

			req := httptest.NewRequest(http.MethodPut, "/products/"+tt.productID, &body)
			req.Header.Set("Content-Type", "application/json")

			// Add chi URL param
			if tt.productID != "" {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("id", tt.productID)
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			}

			w := httptest.NewRecorder()
			handler := &ProductHandler{}

			handler.UpdateProduct(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("Expected body to contain '%s', got '%s'", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestProductHandler_DeleteProduct_Validation(t *testing.T) {
	tests := []struct {
		name           string
		productID      string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "missing product ID",
			productID:      "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing product ID in path",
		},
		{
			name:           "invalid product ID format",
			productID:      "abc",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid product ID format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/products/"+tt.productID, nil)

			// Add chi URL param
			if tt.productID != "" {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("id", tt.productID)
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			}

			w := httptest.NewRecorder()
			handler := &ProductHandler{}

			handler.DeleteProduct(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("Expected body to contain '%s', got '%s'", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestProductHandler_IngestProducts_Validation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "invalid JSON",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid JSON in request body",
		},
		{
			name: "missing CSV path",
			requestBody: ProductIngestionRequest{
				UseCache: true,
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "csv_path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			if str, ok := tt.requestBody.(string); ok {
				body.WriteString(str)
			} else {
				require.NoError(t, json.NewEncoder(&body).Encode(tt.requestBody))
			}

			req := httptest.NewRequest(http.MethodPost, "/products/ingest", &body)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler := &ProductHandler{}
			handler.IngestProducts(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("Expected body to contain '%s', got '%s'", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestProductHandler_GetProductsByCategory_Validation(t *testing.T) {
	tests := []struct {
		name           string
		category       string
		limit          string
		offset         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "missing category",
			category:       "",
			limit:          "50",
			offset:         "0",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing category in path",
		},
		{
			name:           "invalid limit",
			category:       "electronics",
			limit:          "invalid",
			offset:         "0",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid limit parameter",
		},
		{
			name:           "limit too high",
			category:       "electronics",
			limit:          "2000",
			offset:         "0",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid limit parameter",
		},
		{
			name:           "negative offset",
			category:       "electronics",
			limit:          "50",
			offset:         "-1",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid offset parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/products/category/"+tt.category+"?limit="+tt.limit+"&offset="+tt.offset, nil)

			// Add chi URL param
			if tt.category != "" {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("category", tt.category)
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			}

			w := httptest.NewRecorder()
			handler := &ProductHandler{}

			handler.GetProductsByCategory(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("Expected body to contain '%s', got '%s'", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestProductHandler_GetProductsByBrand_Validation(t *testing.T) {
	tests := []struct {
		name           string
		brand          string
		limit          string
		offset         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "missing brand",
			brand:          "",
			limit:          "50",
			offset:         "0",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing brand in path",
		},
		{
			name:           "invalid limit",
			brand:          "Nike",
			limit:          "abc",
			offset:         "0",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid limit parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/products/brand/"+tt.brand+"?limit="+tt.limit+"&offset="+tt.offset, nil)

			// Add chi URL param
			if tt.brand != "" {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("brand", tt.brand)
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			}

			w := httptest.NewRecorder()
			handler := &ProductHandler{}

			handler.GetProductsByBrand(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("Expected body to contain '%s', got '%s'", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestProductHandler_SearchProducts_Validation(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		limit          string
		offset         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "missing query",
			query:          "",
			limit:          "50",
			offset:         "0",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing search query parameter 'q'",
		},
		{
			name:           "invalid limit",
			query:          "laptop",
			limit:          "0",
			offset:         "0",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid limit parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/products/search?q="+tt.query+"&limit="+tt.limit+"&offset="+tt.offset, nil)
			w := httptest.NewRecorder()

			handler := &ProductHandler{}
			handler.SearchProducts(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("Expected body to contain '%s', got '%s'", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestProductHandler_GetProductsInStock_Validation(t *testing.T) {
	tests := []struct {
		name           string
		limit          string
		offset         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "invalid limit",
			limit:          "-1",
			offset:         "0",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid limit parameter",
		},
		{
			name:           "invalid offset",
			limit:          "50",
			offset:         "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid offset parameter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/products/in-stock?limit="+tt.limit+"&offset="+tt.offset, nil)
			w := httptest.NewRecorder()

			handler := &ProductHandler{}
			handler.GetProductsInStock(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("Expected body to contain '%s', got '%s'", tt.expectedBody, w.Body.String())
			}
		})
	}
}
