package order

import (
	"sync"
)

// CustomerIdentity holds resolved customer identity for authorization
type CustomerIdentity struct {
	CustomerID  string
	Email       string
	KeycloakSub string
}

// IdentityCache provides thread-safe lookup of customer identities by keycloak_sub.
// It is bootstrapped from Kafka events at startup and kept current via subscription.
type IdentityCache struct {
	mu    sync.RWMutex
	cache map[string]CustomerIdentity // keycloak_sub → CustomerIdentity
}

// NewIdentityCache creates a new empty cache
func NewIdentityCache() *IdentityCache {
	return &IdentityCache{
		cache: make(map[string]CustomerIdentity),
	}
}

// Get returns the CustomerIdentity for a given keycloak_sub
func (c *IdentityCache) Get(keycloakSub string) (CustomerIdentity, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	identity, ok := c.cache[keycloakSub]
	return identity, ok
}

// Set upserts a customer identity into the cache
func (c *IdentityCache) Set(keycloakSub string, identity CustomerIdentity) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[keycloakSub] = identity
}

// Count returns the number of entries in the cache
func (c *IdentityCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}
