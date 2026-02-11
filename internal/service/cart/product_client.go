package cart

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ProductClient interface {
	GetProduct(ctx context.Context, productID string) (*ProductInfo, error)
}

type ProductInfo struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	FinalPrice float64 `json:"final_price"`
	Currency   string  `json:"currency"`
	InStock    bool    `json:"in_stock"`
}

type HTTPProductClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewProductClient(baseURL string) ProductClient {
	return &HTTPProductClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *HTTPProductClient) GetProduct(ctx context.Context, productID string) (*ProductInfo, error) {
	url := fmt.Sprintf("%s/api/v1/products/%s", c.baseURL, productID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call product service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("product not found: %s", productID)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("product service returned status %d", resp.StatusCode)
	}

	var product ProductInfo
	if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
		return nil, fmt.Errorf("failed to decode product response: %w", err)
	}

	return &product, nil
}
