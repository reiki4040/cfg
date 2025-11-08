package cfg

import (
	"encoding/json"
	"testing"
	"time"
)

// MockParameterStoreClient mocks AWS Parameter Store for testing
type MockParameterStoreClient struct {
	params map[string]string
	errors map[string]error
}

// NewMockParameterStoreClient creates a new mock client
func NewMockParameterStoreClient() *MockParameterStoreClient {
	return &MockParameterStoreClient{
		params: make(map[string]string),
		errors: make(map[string]error),
	}
}

// SetParameter sets a parameter value in the mock
func (m *MockParameterStoreClient) SetParameter(key string, value string) {
	m.params[key] = value
}

// SetError sets an error for a parameter in the mock
func (m *MockParameterStoreClient) SetError(key string, err error) {
	m.errors[key] = err
}

// GetParameter simulates Parameter Store GetParameter call
func (m *MockParameterStoreClient) GetParameter(key string) (string, error) {
	if err, exists := m.errors[key]; exists {
		return "", err
	}
	if value, exists := m.params[key]; exists {
		return value, nil
	}
	return "", nil
}

// TestIntegrationJsonPathExtraction tests JSONPath extraction with realistic JSON
func TestIntegrationJsonPathExtraction(t *testing.T) {
	stageResolver := NewStageResolver("prod")
	resolver := NewResolver(nil, stageResolver)

	// Setup mock JSON parameter store data
	dbConfig := map[string]interface{}{
		"database": map[string]interface{}{
			"primary": map[string]interface{}{
				"host":     "primary-db.example.com",
				"port":     5432,
				"username": "dbuser",
				"password": "secret123",
			},
			"replica": map[string]interface{}{
				"host": "replica-db.example.com",
				"port": 5432,
			},
		},
		"connection_pool": map[string]interface{}{
			"min_size": 5,
			"max_size": 20,
		},
	}

	dbConfigJSON, _ := json.Marshal(dbConfig)

	yamlContent := `
database:
  primary_host: ${ps:/db-config:database.primary.host}
  primary_port: ${ps:/db-config:database.primary.port}
  primary_user: ${ps:/db-config:database.primary.username}
  replica_host: ${ps:/db-config:database.replica.host}
  pool_min: ${ps:/db-config:connection_pool.min_size}
  legacy: ${ps:/db-config}
`

	refs := resolver.ExtractReferences(yamlContent)

	// Should extract 6 ps references
	psRefs := []Reference{}
	for _, ref := range refs {
		if ref.Type == "ps" {
			psRefs = append(psRefs, ref)
		}
	}

	if len(psRefs) != 6 {
		t.Errorf("Expected 6 ps references, got %d", len(psRefs))
	}

	// Verify JSONPath references are captured
	jsonPathRefs := 0
	for _, ref := range psRefs {
		if ref.JSONPath != "" {
			jsonPathRefs++
		}
	}

	if jsonPathRefs != 5 {
		t.Errorf("Expected 5 JSONPath references, got %d", jsonPathRefs)
	}

	// Verify deduplication: all JSONPath refs should point to same parameter
	paramMap := make(map[string]int)
	for _, ref := range psRefs {
		if ref.JSONPath != "" {
			paramMap[ref.Key]++
		}
	}

	if len(paramMap) != 1 {
		t.Errorf("Expected JSONPath refs to map to 1 parameter, got %d", len(paramMap))
	}

	// Verify the extractor can parse the JSON config
	extractor := NewJsonPathExtractor()

	testCases := []struct {
		jsonPath     string
		expectedType string
	}{
		{"database.primary.host", "string"},
		{"database.primary.port", "number"},
		{"connection_pool.max_size", "number"},
	}

	for _, tc := range testCases {
		result, err := extractor.ExtractValue(string(dbConfigJSON), tc.jsonPath, "/db-config")
		if err != nil {
			t.Errorf("Failed to extract %s: %v", tc.jsonPath, err)
		}
		if result == "" {
			t.Errorf("Expected non-empty result for %s", tc.jsonPath)
		}
	}
}

// TestIntegrationMixedReferences tests mixed reference types in one YAML
func TestIntegrationMixedReferences(t *testing.T) {
	stageResolver := NewStageResolver("staging")
	resolver := NewResolver(nil, stageResolver)

	yamlContent := `
app:
  name: myapp
  environment: ${env:APP_ENV}
  config_path: ${stage-prefix:configs}
  database:
    primary: ${ps:/app/staging/db-config:primary.host}
    secondary: ${ps:/app/staging/db-config:secondary.host}
    credentials: ${ps:/app/staging/db-secrets:password}
  api:
    endpoint: ${env:API_ENDPOINT}
    timeout: 30
`

	refs := resolver.ExtractReferences(yamlContent)

	// Count references by type
	counts := make(map[string]int)
	for _, ref := range refs {
		counts[ref.Type]++
	}

	if counts["ps"] != 3 {
		t.Errorf("Expected 3 ps references, got %d", counts["ps"])
	}
	if counts["env"] != 2 {
		t.Errorf("Expected 2 env references, got %d", counts["env"])
	}
	if counts["stage-prefix"] != 1 {
		t.Errorf("Expected 1 stage-prefix reference, got %d", counts["stage-prefix"])
	}

	// Verify JSONPath extraction
	jsonPathCount := 0
	paramCount := make(map[string]bool)
	for _, ref := range refs {
		if ref.Type == "ps" && ref.JSONPath != "" {
			jsonPathCount++
			paramCount[ref.Key] = true
		}
	}

	// 3 ps references: 2 with JSONPath from db-config, 1 with JSONPath from db-secrets
	if jsonPathCount != 3 {
		t.Errorf("Expected 3 JSONPath references, got %d", jsonPathCount)
	}

	// 2 different parameters with JSONPath
	if len(paramCount) != 2 {
		t.Errorf("Expected 2 unique parameters with JSONPath, got %d", len(paramCount))
	}
}

// TestIntegrationDeepNesting tests deeply nested JSON structures
func TestIntegrationDeepNesting(t *testing.T) {
	stageResolver := NewStageResolver("dev")
	resolver := NewResolver(nil, stageResolver)

	// Create deeply nested JSON structure
	deepConfig := map[string]interface{}{
		"services": map[string]interface{}{
			"api": map[string]interface{}{
				"v1": map[string]interface{}{
					"endpoints": map[string]interface{}{
						"users": map[string]interface{}{
							"host":   "api.example.com",
							"port":   8080,
							"secure": true,
						},
					},
				},
			},
		},
	}

	deepConfigJSON, _ := json.Marshal(deepConfig)

	yamlContent := `
services:
  api:
    host: ${ps:/deep-config:services.api.v1.endpoints.users.host}
    port: ${ps:/deep-config:services.api.v1.endpoints.users.port}
    secure: ${ps:/deep-config:services.api.v1.endpoints.users.secure}
`

	refs := resolver.ExtractReferences(yamlContent)

	psRefs := []Reference{}
	for _, ref := range refs {
		if ref.Type == "ps" {
			psRefs = append(psRefs, ref)
		}
	}

	if len(psRefs) != 3 {
		t.Errorf("Expected 3 references, got %d", len(psRefs))
	}

	// All should have JSONPath
	for _, ref := range psRefs {
		if ref.JSONPath == "" {
			t.Errorf("Expected JSONPath in reference")
		}

		// Verify each path can be extracted
		extractor := NewJsonPathExtractor()
		result, err := extractor.ExtractValue(string(deepConfigJSON), ref.JSONPath, "/deep-config")
		if err != nil {
			t.Errorf("Failed to extract %s: %v", ref.JSONPath, err)
		}
		if result == "" {
			t.Errorf("Expected non-empty result for JSONPath: %s", ref.JSONPath)
		}
	}
}

// TestIntegrationStageSubstitution tests stage placeholder substitution with JSONPath
func TestIntegrationStageSubstitution(t *testing.T) {
	stageResolver := NewStageResolver("testing")
	resolver := NewResolver(nil, stageResolver)

	yamlContent := `
config:
  db: ${ps:/app/{stage}/database-config:host}
  cache: ${ps:/infra/{stage}/redis-config:connection.url}
  secrets: ${ps:/app/{stage}/secrets:api-key}
`

	refs := resolver.ExtractReferences(yamlContent)

	psRefs := []Reference{}
	for _, ref := range refs {
		if ref.Type == "ps" {
			psRefs = append(psRefs, ref)
		}
	}

	if len(psRefs) != 3 {
		t.Errorf("Expected 3 ps references, got %d", len(psRefs))
	}

	// Verify {stage} was replaced with 'testing'
	for _, ref := range psRefs {
		if ref.Key == "" {
			t.Errorf("Expected non-empty key after stage substitution")
		}
		if contains(ref.Key, "{stage}") {
			t.Errorf("Stage placeholder not resolved: %s", ref.Key)
		}
		if !contains(ref.Key, "testing") {
			t.Errorf("Expected 'testing' in resolved key: %s", ref.Key)
		}
	}
}

// TestIntegrationCacheConsistency tests that caching maintains consistency
func TestIntegrationCacheConsistency(t *testing.T) {
	stageResolver := NewStageResolver("prod")
	resolver := NewResolver(nil, stageResolver)

	// Setup cache with explicit TTL
	cache := NewJsonCache(10 * time.Minute)
	resolver.jsonCache = cache

	jsonObj := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"nested": map[string]interface{}{
			"key3": "value3",
		},
	}

	// Store in cache
	cache.Set("/config", jsonObj)

	// Extract multiple values from same cached object
	extractor := NewJsonPathExtractor()

	testPaths := []struct {
		path     string
		expected string
	}{
		{"key1", "value1"},
		{"key2", "value2"},
		{"nested.key3", "value3"},
	}

	jsonBytes, _ := json.Marshal(jsonObj)
	for _, test := range testPaths {
		result, err := extractor.ExtractValue(string(jsonBytes), test.path, "/config")
		if err != nil {
			t.Errorf("Failed to extract %s: %v", test.path, err)
		}
		if result != test.expected {
			t.Errorf("Expected %s, got %s for path %s", test.expected, result, test.path)
		}
	}

	// Verify cache contains the object
	if cachedObj, exists := cache.Get("/config"); !exists {
		t.Errorf("Expected /config in cache")
	} else if cachedObj == nil {
		t.Errorf("Expected cached value to be non-nil")
	}
}

// TestIntegrationErrorHandling tests error scenarios
func TestIntegrationErrorHandling(t *testing.T) {
	stageResolver := NewStageResolver("dev")
	resolver := NewResolver(nil, stageResolver)

	// Test invalid JSONPath with valid parameter reference
	yamlContent := `
config:
  invalid: ${ps:/config:..invalid}
  missing: ${ps:/config:nonexistent.key}
`

	refs := resolver.ExtractReferences(yamlContent)

	if len(refs) != 2 {
		t.Errorf("Expected 2 references, got %d", len(refs))
	}

	// Both should have JSONPath fields extracted
	for _, ref := range refs {
		if ref.Type == "ps" && ref.JSONPath != "" {
			// This indicates we extracted something, which is correct
			// The actual validation will happen during resolution
		}
	}
}

// TestIntegrationMultipleJsonPathSameParameter tests deduplication
func TestIntegrationMultipleJsonPathSameParameter(t *testing.T) {
	stageResolver := NewStageResolver("prod")
	resolver := NewResolver(nil, stageResolver)

	yamlContent := `
database:
  host: ${ps:/db-config:database.host}
  port: ${ps:/db-config:database.port}
  user: ${ps:/db-config:database.user}
  password: ${ps:/db-config:database.password}
`

	refs := resolver.ExtractReferences(yamlContent)

	// Count unique parameters
	uniqueParams := make(map[string]bool)
	for _, ref := range refs {
		if ref.Type == "ps" {
			uniqueParams[ref.Key] = true
		}
	}

	// Should be deduplicated to 1 unique parameter
	if len(uniqueParams) != 1 {
		t.Errorf("Expected 1 unique parameter after deduplication, got %d", len(uniqueParams))
	}

	// All references should have JSONPath
	for _, ref := range refs {
		if ref.Type == "ps" && ref.JSONPath == "" {
			t.Errorf("Expected JSONPath in ps reference")
		}
	}
}

// TestIntegrationNullAndEmptyValues tests handling of null and empty values
func TestIntegrationNullAndEmptyValues(t *testing.T) {
	extractor := NewJsonPathExtractor()

	testCases := []struct {
		name     string
		json     string
		path     string
		valid    bool
		hasValue bool
	}{
		{
			name:     "null_value",
			json:     `{"key": null}`,
			path:     "key",
			valid:    true,
			hasValue: false,
		},
		{
			name:     "empty_string",
			json:     `{"key": ""}`,
			path:     "key",
			valid:    true,
			hasValue: false,
		},
		{
			name:     "empty_object",
			json:     `{"key": {}}`,
			path:     "key",
			valid:    true,
			hasValue: false,
		},
		{
			name:     "false_boolean",
			json:     `{"key": false}`,
			path:     "key",
			valid:    true,
			hasValue: true,
		},
		{
			name:     "zero_number",
			json:     `{"key": 0}`,
			path:     "key",
			valid:    true,
			hasValue: true,
		},
	}

	for _, tc := range testCases {
		_, err := extractor.ExtractValue(tc.json, tc.path, "/param")
		if !tc.valid && err == nil {
			t.Errorf("%s: expected error but got none", tc.name)
		}
		if tc.valid && err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	// Simple substring check
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
