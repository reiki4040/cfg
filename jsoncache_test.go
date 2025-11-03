package cfg

import (
	"testing"
	"time"
)

// Test JSON cache storage and retrieval
func TestJsonCacheBasicOperations(t *testing.T) {
	cache := NewJsonCache(5 * time.Minute)

	// Test storing a parsed JSON object
	jsonObj := map[string]interface{}{
		"password": "secret123",
		"host":     "localhost",
	}

	cache.Set("/config", jsonObj)

	// Test retrieval
	retrieved, exists := cache.Get("/config")
	if !exists {
		t.Errorf("Expected cache entry to exist, but it doesn't")
	}

	if len(retrieved) != len(jsonObj) {
		t.Errorf("Expected %d keys, got %d", len(jsonObj), len(retrieved))
	}

	// Test retrieval of non-existent key
	_, exists = cache.Get("/nonexistent")
	if exists {
		t.Errorf("Expected non-existent entry to not exist")
	}
}

// Test cache TTL and expiration
func TestJsonCacheTTL(t *testing.T) {
	// Create cache with very short TTL
	cache := NewJsonCache(10 * time.Millisecond)

	jsonObj := map[string]interface{}{
		"key": "value",
	}

	cache.Set("/config", jsonObj)

	// Should exist immediately
	_, exists := cache.Get("/config")
	if !exists {
		t.Errorf("Expected cache entry to exist immediately after set")
	}

	// Wait for TTL to expire
	time.Sleep(20 * time.Millisecond)

	// Should not exist after TTL
	_, exists = cache.Get("/config")
	if exists {
		t.Errorf("Expected cache entry to be expired after TTL")
	}
}

// Test manual cache invalidation
func TestJsonCacheInvalidation(t *testing.T) {
	cache := NewJsonCache(5 * time.Minute)

	jsonObj := map[string]interface{}{
		"key": "value",
	}

	cache.Set("/config", jsonObj)

	// Verify it exists
	_, exists := cache.Get("/config")
	if !exists {
		t.Errorf("Expected cache entry to exist")
	}

	// Invalidate it
	cache.Invalidate("/config")

	// Verify it's gone
	_, exists = cache.Get("/config")
	if exists {
		t.Errorf("Expected cache entry to be invalidated")
	}
}

// Test cache clearing
func TestJsonCacheClear(t *testing.T) {
	cache := NewJsonCache(5 * time.Minute)

	// Add multiple entries
	cache.Set("/config1", map[string]interface{}{"key": "value1"})
	cache.Set("/config2", map[string]interface{}{"key": "value2"})
	cache.Set("/config3", map[string]interface{}{"key": "value3"})

	// Verify all exist
	if _, exists := cache.Get("/config1"); !exists {
		t.Errorf("Expected /config1 to exist")
	}
	if _, exists := cache.Get("/config2"); !exists {
		t.Errorf("Expected /config2 to exist")
	}
	if _, exists := cache.Get("/config3"); !exists {
		t.Errorf("Expected /config3 to exist")
	}

	// Clear cache
	cache.Clear()

	// Verify all are gone
	if _, exists := cache.Get("/config1"); exists {
		t.Errorf("Expected /config1 to be cleared")
	}
	if _, exists := cache.Get("/config2"); exists {
		t.Errorf("Expected /config2 to be cleared")
	}
	if _, exists := cache.Get("/config3"); exists {
		t.Errorf("Expected /config3 to be cleared")
	}
}

// Test cache update (overwrite existing entry)
func TestJsonCacheUpdate(t *testing.T) {
	cache := NewJsonCache(5 * time.Minute)

	// Set initial value
	initialObj := map[string]interface{}{
		"key": "initial",
	}
	cache.Set("/config", initialObj)

	// Update value
	updatedObj := map[string]interface{}{
		"key": "updated",
	}
	cache.Set("/config", updatedObj)

	// Verify updated value is returned
	retrieved, exists := cache.Get("/config")
	if !exists {
		t.Errorf("Expected cache entry to exist")
	}

	if retrieved["key"] != "updated" {
		t.Errorf("Expected key to be 'updated', got %v", retrieved["key"])
	}
}
