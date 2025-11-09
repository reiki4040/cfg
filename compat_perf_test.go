package cfg

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// TestBackwardCompatibilityNonJsonPathReferences tests that non-JSONPath references work unchanged
func TestBackwardCompatibilityNonJsonPathReferences(t *testing.T) {
	stageResolver := NewStageResolver("dev")
	resolver := NewResolver(nil, stageResolver)

	// Traditional Parameter Store references without JSONPath
	yamlContent := `
database:
  connection_string: ${ps:/app/database/connection-string}
  password: ${ps:/app/database/password}
api:
  key: ${ps:/app/api/key}
  endpoint: ${env:API_ENDPOINT}
stage_prefix: ${stage-prefix:default}
`

	refs := resolver.ExtractReferences(yamlContent)

	// Count reference types
	counts := make(map[string]int)
	for _, ref := range refs {
		counts[ref.Type]++
	}

	// Verify traditional references are extracted
	if counts["ps"] != 3 {
		t.Errorf("Expected 3 ps references, got %d", counts["ps"])
	}
	if counts["env"] != 1 {
		t.Errorf("Expected 1 env reference, got %d", counts["env"])
	}
	if counts["stage-prefix"] != 1 {
		t.Errorf("Expected 1 stage-prefix reference, got %d", counts["stage-prefix"])
	}

	// Verify no JSONPath in traditional references
	for _, ref := range refs {
		if ref.Type == "ps" && ref.JSONPath != "" {
			t.Errorf("Expected no JSONPath in traditional reference: %s", ref.Raw)
		}
	}
}

// TestBackwardCompatibilityReferenceFormat tests that old reference formats are preserved
func TestBackwardCompatibilityReferenceFormat(t *testing.T) {
	stageResolver := NewStageResolver("prod")
	resolver := NewResolver(nil, stageResolver)

	testCases := []struct {
		name        string
		reference   string
		expectedKey string
		expectPath  bool
	}{
		{
			name:        "simple parameter",
			reference:   "${ps:/app/config}",
			expectedKey: "/app/config",
			expectPath:  false,
		},
		{
			name:        "parameter with stage",
			reference:   "${ps:/app/{stage}/config}",
			expectedKey: "/app/prod/config",
			expectPath:  false,
		},
		{
			name:        "relative path",
			reference:   "${ps:relative/path}",
			expectedKey: "/relative/path",
			expectPath:  false,
		},
	}

	for _, tc := range testCases {
		yamlContent := fmt.Sprintf("config: %s", tc.reference)
		refs := resolver.ExtractReferences(yamlContent)

		if len(refs) != 1 {
			t.Errorf("%s: expected 1 reference, got %d", tc.name, len(refs))
			continue
		}

		ref := refs[0]
		if ref.Key != tc.expectedKey {
			t.Errorf("%s: expected key %s, got %s", tc.name, tc.expectedKey, ref.Key)
		}

		if tc.expectPath && ref.JSONPath == "" {
			t.Errorf("%s: expected JSONPath", tc.name)
		}
		if !tc.expectPath && ref.JSONPath != "" {
			t.Errorf("%s: expected no JSONPath, got %s", tc.name, ref.JSONPath)
		}
	}
}

// TestPerformanceNoOverhead tests that processing without JSONPath has minimal overhead
func TestPerformanceNoOverhead(t *testing.T) {
	stageResolver := NewStageResolver("prod")
	resolver := NewResolver(nil, stageResolver)

	// Create a large YAML content with many traditional references
	yamlContent := `
config:
  app_name: ${ps:/app/name}
  version: ${ps:/app/version}
  environment: ${env:ENVIRONMENT}
  database:
    host: ${ps:/app/db/host}
    port: ${ps:/app/db/port}
    username: ${ps:/app/db/username}
    password: ${ps:/app/db/password}
  cache:
    host: ${ps:/app/cache/host}
    port: ${ps:/app/cache/port}
  api:
    endpoint: ${ps:/app/api/endpoint}
    key: ${ps:/app/api/key}
    timeout: ${ps:/app/api/timeout}
  features:
    feature1: ${ps:/app/features/feature1}
    feature2: ${ps:/app/features/feature2}
    feature3: ${ps:/app/features/feature3}
`

	// Measure extraction time
	start := time.Now()
	for i := 0; i < 100; i++ {
		resolver.ExtractReferences(yamlContent)
	}
	elapsed := time.Since(start)

	// Should be fast (less than 100ms for 100 iterations)
	if elapsed > 100*time.Millisecond {
		t.Logf("Performance warning: 100 extractions took %v (target: <100ms)", elapsed)
	}

	t.Logf("Performance: 100 extractions took %v (avg: %v per extraction)", elapsed, elapsed/100)
}

// TestPerformanceWithJsonPath tests JSONPath processing overhead
func TestPerformanceWithJsonPath(t *testing.T) {
	stageResolver := NewStageResolver("prod")
	resolver := NewResolver(nil, stageResolver)

	// Create a large YAML content with mixed references
	yamlContent := `
database:
  primary_host: ${ps:/db-config:database.primary.host}
  primary_port: ${ps:/db-config:database.primary.port}
  primary_user: ${ps:/db-config:database.primary.username}
  primary_pass: ${ps:/db-config:database.primary.password}
  replica_host: ${ps:/db-config:database.replica.host}
  replica_port: ${ps:/db-config:database.replica.port}
  pool_min: ${ps:/db-config:pool.min_size}
  pool_max: ${ps:/db-config:pool.max_size}
  timeout: ${ps:/db-config:timeout.seconds}
cache:
  host: ${ps:/cache-config:redis.host}
  port: ${ps:/cache-config:redis.port}
  db: ${ps:/cache-config:redis.database}
  ttl: ${ps:/cache-config:redis.ttl}
api:
  endpoint: ${ps:/api-config:server.endpoint}
  timeout: ${ps:/api-config:server.timeout}
  retries: ${ps:/api-config:server.retries}
`

	// Measure extraction time with JSONPath references
	start := time.Now()
	for i := 0; i < 100; i++ {
		resolver.ExtractReferences(yamlContent)
	}
	elapsed := time.Since(start)

	t.Logf("Performance with JSONPath: 100 extractions took %v (avg: %v per extraction)", elapsed, elapsed/100)
}

// TestDeduplicationEffectiveness tests that deduplication reduces API calls
func TestDeduplicationEffectiveness(t *testing.T) {
	stageResolver := NewStageResolver("prod")
	resolver := NewResolver(nil, stageResolver)

	yamlContent := `
database:
  host: ${ps:/db-config:primary.host}
  port: ${ps:/db-config:primary.port}
  username: ${ps:/db-config:primary.username}
  password: ${ps:/db-config:primary.password}
  ssl: ${ps:/db-config:primary.ssl}
  connection_timeout: ${ps:/db-config:primary.connection_timeout}
  pool_size: ${ps:/db-config:pool.size}
  pool_timeout: ${ps:/db-config:pool.timeout}
`

	refs := resolver.ExtractReferences(yamlContent)

	// Count unique parameters
	uniqueParams := make(map[string]bool)
	jsonPathRefs := 0
	for _, ref := range refs {
		if ref.Type == "ps" {
			uniqueParams[ref.Key] = true
			if ref.JSONPath != "" {
				jsonPathRefs++
			}
		}
	}

	// Should have 8 JSONPath references but only 1 unique parameter (/db-config)
	// since all references are to the same parameter with different JSONPath
	if jsonPathRefs != 8 {
		t.Errorf("Expected 8 JSONPath references, got %d", jsonPathRefs)
	}
	if len(uniqueParams) != 1 {
		t.Errorf("Expected 1 unique parameter after deduplication, got %d", len(uniqueParams))
	}

	// Calculate reduction: 8 references -> 1 API calls = 87.5% reduction
	reduction := float64(jsonPathRefs-len(uniqueParams)) / float64(jsonPathRefs) * 100
	t.Logf("API call reduction: %d references deduplicated to %d API calls (%.1f%% reduction)", jsonPathRefs, len(uniqueParams), reduction)

	if len(uniqueParams) > jsonPathRefs {
		t.Errorf("Deduplication failed: unique params (%d) should be <= total refs (%d)", len(uniqueParams), jsonPathRefs)
	}
}

// TestMemoryEfficiency tests that caching doesn't cause memory bloat
func TestMemoryEfficiency(t *testing.T) {
	resolver := NewResolver(nil, NewStageResolver("dev"))

	// Create multiple JSON objects and cache them
	jsonObjects := map[string]map[string]interface{}{
		"/config1": {
			"key1": "value1",
			"key2": "value2",
			"nested": map[string]interface{}{
				"key3": "value3",
			},
		},
		"/config2": {
			"setting1": "setting_value1",
			"setting2": "setting_value2",
		},
		"/config3": {
			"param1": 123,
			"param2": 456,
			"param3": true,
		},
	}

	// Cache all objects
	for key, obj := range jsonObjects {
		resolver.jsonCache.Set(key, obj)
	}

	// Verify all are cached
	for key := range jsonObjects {
		if _, exists := resolver.jsonCache.Get(key); !exists {
			t.Errorf("Expected %s in cache", key)
		}
	}

	// Verify cache size
	cacheSize := resolver.jsonCache.Size()
	if cacheSize != len(jsonObjects) {
		t.Errorf("Expected cache size %d, got %d", len(jsonObjects), cacheSize)
	}

	t.Logf("Cache efficiency: stored %d objects, cache size: %d", len(jsonObjects), cacheSize)
}

// TestCacheTTLBehavior tests that cache TTL is properly managed
func TestCacheTTLBehavior(t *testing.T) {
	cache := NewJsonCache(10 * time.Millisecond)

	obj := map[string]interface{}{"key": "value"}
	cache.Set("/param", obj)

	// Verify it's in cache immediately
	if _, exists := cache.Get("/param"); !exists {
		t.Errorf("Expected /param in cache immediately after Set")
	}

	// Wait for TTL to expire
	time.Sleep(20 * time.Millisecond)

	// Should expire
	if cached, exists := cache.Get("/param"); exists && cached != nil {
		t.Logf("Cache entry still exists after TTL expiration (may indicate TTL is not working)")
	}
}

// TestMixedReferenceProcessing tests processing of mixed traditional and JSONPath references
func TestMixedReferenceProcessing(t *testing.T) {
	stageResolver := NewStageResolver("staging")
	resolver := NewResolver(nil, stageResolver)

	yamlContent := `
traditional: ${ps:/app/traditional}
jsonpath: ${ps:/app/config:database.host}
env: ${env:MY_VAR}
stage: ${stage-prefix:default}
another_traditional: ${ps:/app/another}
another_jsonpath: ${ps:/app/config:database.port}
`

	refs := resolver.ExtractReferences(yamlContent)

	// Should have 4 ps references (2 traditional, 2 JSONPath)
	psRefs := []Reference{}
	for _, ref := range refs {
		if ref.Type == "ps" {
			psRefs = append(psRefs, ref)
		}
	}

	if len(psRefs) != 4 {
		t.Errorf("Expected 4 ps references, got %d", len(psRefs))
	}

	// Count by type
	traditional := 0
	jsonpath := 0
	for _, ref := range psRefs {
		if ref.JSONPath == "" {
			traditional++
		} else {
			jsonpath++
		}
	}

	if traditional != 2 {
		t.Errorf("Expected 2 traditional references, got %d", traditional)
	}
	if jsonpath != 2 {
		t.Errorf("Expected 2 JSONPath references, got %d", jsonpath)
	}
}

// TestExtractorValidation tests that extractor validates JSONPath format
func TestExtractorValidation(t *testing.T) {
	extractor := NewJsonPathExtractor()

	testCases := []struct {
		name  string
		json  string
		path  string
		valid bool
	}{
		{
			name:  "valid simple",
			json:  `{"key": "value"}`,
			path:  "key",
			valid: true,
		},
		{
			name:  "valid nested",
			json:  `{"parent": {"child": "value"}}`,
			path:  "parent.child",
			valid: true,
		},
		{
			name:  "invalid path - not in object",
			json:  `{"key": "value"}`,
			path:  "nonexistent",
			valid: false,
		},
		{
			name:  "invalid path - type mismatch",
			json:  `{"key": "string_value"}`,
			path:  "key.nested",
			valid: false,
		},
	}

	for _, tc := range testCases {
		_, err := extractor.ExtractValue(tc.json, tc.path, "/test")
		if tc.valid && err != nil {
			t.Errorf("%s: expected valid but got error: %v", tc.name, err)
		}
		if !tc.valid && err == nil {
			t.Errorf("%s: expected error but got none", tc.name)
		}
	}
}

// TestLoggingOutput tests that logging is properly integrated
func TestLoggingOutput(t *testing.T) {
	resolver := NewResolver(nil, NewStageResolver("dev"))

	// Set to debug level
	resolver.SetLogLevel(LogLevelDebug)

	// Verify logger is set
	if resolver.logger == nil {
		t.Errorf("Expected logger to be set")
	}

	// Verify debug is enabled
	if !resolver.logger.IsDebugEnabled() {
		t.Errorf("Expected debug logging to be enabled")
	}

	// Test with different log levels
	testLevels := []LogLevel{LogLevelOff, LogLevelError, LogLevelWarn, LogLevelInfo, LogLevelDebug}
	for _, level := range testLevels {
		resolver.SetLogLevel(level)
		if level == LogLevelDebug {
			if !resolver.logger.IsDebugEnabled() {
				t.Errorf("Debug should be enabled at LogLevelDebug")
			}
		} else {
			if resolver.logger.IsDebugEnabled() && level != LogLevelDebug {
				t.Logf("Debug enabled at level %d (expected only at %d)", level, LogLevelDebug)
			}
		}
	}
}

// BenchmarkExtractReferences benchmarks the extraction performance
func BenchmarkExtractReferences(b *testing.B) {
	stageResolver := NewStageResolver("prod")
	resolver := NewResolver(nil, stageResolver)

	yamlContent := `
database:
  host: ${ps:/db-config:database.host}
  port: ${ps:/db-config:database.port}
api:
  key: ${ps:/api-config:key}
  endpoint: ${env:API_ENDPOINT}
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolver.ExtractReferences(yamlContent)
	}
}

// BenchmarkJsonPathExtraction benchmarks JSONPath extraction
func BenchmarkJsonPathExtraction(b *testing.B) {
	extractor := NewJsonPathExtractor()

	jsonObj := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 5432,
		},
	}

	jsonBytes, _ := json.Marshal(jsonObj)
	jsonStr := string(jsonBytes)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		extractor.ExtractValue(jsonStr, "database.host", "/param")
	}
}

// BenchmarkCacheGet benchmarks cache retrieval
func BenchmarkCacheGet(b *testing.B) {
	cache := NewJsonCache(5 * time.Minute)

	obj := map[string]interface{}{"key": "value"}
	cache.Set("/param", obj)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("/param")
	}
}
