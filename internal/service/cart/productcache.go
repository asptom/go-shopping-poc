package cart

import (
	"sync"
)

// ProductEntry holds the data needed for product validation in the cart service.
// Only the fields required for validation are stored — no full product aggregate.
type ProductEntry struct {
	ProductID  string  `json:"product_id"`
	InStock    bool    `json:"in_stock"`
	FinalPrice float64 `json:"final_price"`
	Name       string  `json:"name"`
}

// ProductCache provides a thread-safe in-memory cache of product identity data.
// It is populated by consuming ProductCreated, ProductUpdated, and ProductDeleted
// events from the ProductEvents topic. The cache is used to accelerate product
// validation when items are added to a cart, eliminating the event round-trip
// for products already in the cache.
type ProductCache struct {
	mu    sync.RWMutex
	items map[string]ProductEntry // product_id → ProductEntry
}

// NewProductCache creates a new empty product cache.
func NewProductCache() *ProductCache {
	return &ProductCache{
		items: make(map[string]ProductEntry),
	}
}

// Get returns the ProductEntry for a given product_id and whether it was found.
func (c *ProductCache) Get(productID string) (ProductEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.items[productID]
	return entry, ok
}

// Set upserts a product entry into the cache.
// Use this for ProductCreated and ProductUpdated events.
func (c *ProductCache) Set(productID string, entry ProductEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[productID] = entry
}

// Delete removes a product entry from the cache.
// Use this for ProductDeleted events.
func (c *ProductCache) Delete(productID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, productID)
}

// Count returns the number of entries in the cache.
func (c *ProductCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}
