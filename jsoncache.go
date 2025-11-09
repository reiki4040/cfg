package cfg

import (
	"sync"
	"time"
)

// JsonCache stores parsed JSON objects with TTL-based expiration
type JsonCache struct {
	entries     map[string]map[string]interface{}
	ttl         map[string]time.Time
	ttlDuration time.Duration
	mu          sync.RWMutex
}

// NewJsonCache creates a new JSON cache with the specified TTL duration
func NewJsonCache(ttlDuration time.Duration) *JsonCache {
	return &JsonCache{
		entries:     make(map[string]map[string]interface{}),
		ttl:         make(map[string]time.Time),
		ttlDuration: ttlDuration,
	}
}

// Get retrieves a cached JSON object by parameter name
// Returns (value, exists)
func (c *JsonCache) Get(parameterName string) (map[string]interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if entry exists and is not expired
	if expiryTime, hasExpiry := c.ttl[parameterName]; hasExpiry {
		if time.Now().After(expiryTime) {
			// Entry has expired
			return nil, false
		}
	}

	if entry, exists := c.entries[parameterName]; exists {
		return entry, true
	}

	return nil, false
}

// Set stores or updates a JSON object in the cache
func (c *JsonCache) Set(parameterName string, jsonObj map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[parameterName] = jsonObj
	c.ttl[parameterName] = time.Now().Add(c.ttlDuration)
}

// Invalidate removes a specific entry from the cache
func (c *JsonCache) Invalidate(parameterName string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, parameterName)
	delete(c.ttl, parameterName)
}

// Clear removes all entries from the cache
func (c *JsonCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]map[string]interface{})
	c.ttl = make(map[string]time.Time)
}

// CleanupExpired removes all expired entries from the cache
// This can be called periodically to clean up memory
func (c *JsonCache) CleanupExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for parameterName, expiryTime := range c.ttl {
		if now.After(expiryTime) {
			delete(c.entries, parameterName)
			delete(c.ttl, parameterName)
		}
	}
}

// SetTTL updates the TTL duration for future entries
func (c *JsonCache) SetTTL(ttlDuration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ttlDuration = ttlDuration
}

// Size returns the number of entries currently in the cache
func (c *JsonCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}
