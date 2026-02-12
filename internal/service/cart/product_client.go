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

// ProductID accepts either a JSON string or a number and always stores it as a string.
type ProductID string

func (p *ProductID) UnmarshalJSON(data []byte) error {
	// allow null
	if string(data) == "null" {
		*p = ""
		return nil
	}

	// try string first (e.g. "123")
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*p = ProductID(s)
		return nil
	}

	// then try number (e.g. 123)
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*p = ProductID(n.String())
		return nil
	}

	return fmt.Errorf("invalid product id: %s", string(data))
}

type ProductInfo struct {
	ID         ProductID `json:"id"`
	Name       string    `json:"name"`
	FinalPrice float64   `json:"final_price"`
	Currency   string    `json:"currency"`
	InStock    bool      `json:"in_stock"`
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
