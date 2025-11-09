package cfg

import (
	"testing"
)

// Test 1.1: Regex pattern matching for JSONPath references
func TestJsonPathReferenceRegexExtraction(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		shouldMatch  bool
		expectedPath string
		expectedJSON string
	}{
		{
			name:         "simple single-level key",
			input:        "${ps:/config:password}",
			shouldMatch:  true,
			expectedPath: "/config",
			expectedJSON: "password",
		},
		{
			name:         "nested key path",
			input:        "${ps:/config:database.password}",
			shouldMatch:  true,
			expectedPath: "/config",
			expectedJSON: "database.password",
		},
		{
			name:         "deeply nested key path",
			input:        "${ps:/app-config:db.primary.host}",
			shouldMatch:  true,
			expectedPath: "/app-config",
			expectedJSON: "db.primary.host",
		},
		{
			name:         "with stage placeholder",
			input:        "${ps:/app-{stage}-config:api.key}",
			shouldMatch:  true,
			expectedPath: "/app-{stage}-config",
			expectedJSON: "api.key",
		},
		{
			name:         "traditional ps reference without jsonpath",
			input:        "${ps:/simple/path}",
			shouldMatch:  true,
			expectedPath: "/simple/path",
			expectedJSON: "",
		},
		{
			name:         "empty jsonpath after colon",
			input:        "${ps:/config:}",
			shouldMatch:  true,
			expectedPath: "/config",
			expectedJSON: "",
		},
		{
			name:         "jsonpath with underscores",
			input:        "${ps:/config:db_password}",
			shouldMatch:  true,
			expectedPath: "/config",
			expectedJSON: "db_password",
		},
		{
			name:         "jsonpath with numbers",
			input:        "${ps:/config:key123.sub456}",
			shouldMatch:  true,
			expectedPath: "/config",
			expectedJSON: "key123.sub456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test will use the extended psReferenceRegex
			// The regex should be updated in resolver.go to support JSONPath
			// Pattern: ${ps:(path)(?::jsonpath)?}
			pathValue, jsonPath, ok := extractJsonPathReference(tt.input)

			if !ok && tt.shouldMatch {
				t.Errorf("Expected to match JSONPath reference, but didn't")
				return
			}

			if ok && !tt.shouldMatch {
				t.Errorf("Expected not to match JSONPath reference, but did")
				return
			}

			if ok {
				if pathValue != tt.expectedPath {
					t.Errorf("Expected path %q, got %q", tt.expectedPath, pathValue)
				}
				if jsonPath != tt.expectedJSON {
					t.Errorf("Expected jsonpath %q, got %q", tt.expectedJSON, jsonPath)
				}
			}
		})
	}
}

// Test 1.5: JSONPath format validation
func TestJsonPathFormatValidation(t *testing.T) {
	tests := []struct {
		name     string
		jsonPath string
		isValid  bool
		reason   string
	}{
		{
			name:     "simple key",
			jsonPath: "password",
			isValid:  true,
		},
		{
			name:     "nested keys",
			jsonPath: "database.password",
			isValid:  true,
		},
		{
			name:     "deeply nested",
			jsonPath: "db.primary.master.password",
			isValid:  true,
		},
		{
			name:     "with underscores",
			jsonPath: "db_password",
			isValid:  true,
		},
		{
			name:     "with numbers",
			jsonPath: "key123.sub456",
			isValid:  true,
		},
		{
			name:     "empty string",
			jsonPath: "",
			isValid:  false,
			reason:   "empty jsonpath",
		},
		{
			name:     "double dots",
			jsonPath: "key..nested",
			isValid:  false,
			reason:   "double dots",
		},
		{
			name:     "trailing dot",
			jsonPath: "key.nested.",
			isValid:  false,
			reason:   "trailing dot",
		},
		{
			name:     "leading dot",
			jsonPath: ".key.nested",
			isValid:  false,
			reason:   "leading dot",
		},
		{
			name:     "array index notation",
			jsonPath: "key[0]",
			isValid:  false,
			reason:   "array index not supported",
		},
		{
			name:     "special characters",
			jsonPath: "key@nested",
			isValid:  false,
			reason:   "special characters not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJsonPath(tt.jsonPath)

			if tt.isValid && err != nil {
				t.Errorf("Expected valid JSONPath, but got error: %v", err)
			}

			if !tt.isValid && err == nil {
				t.Errorf("Expected error for invalid JSONPath (%s), but got none", tt.reason)
			}
		})
	}
}

// Helper function to extract jsonpath reference (to be implemented)
func extractJsonPathReference(input string) (path string, jsonPath string, ok bool) {
	// This is a test helper that will call the actual regex matching
	// Once the regex is updated in resolver.go, this will use it
	matcher := NewJsonPathMatcher()
	return matcher.Match(input)
}

// Test 3: JSONPath value extraction engine
func TestJsonPathValueExtraction(t *testing.T) {
	extractor := NewJsonPathExtractor()

	tests := []struct {
		name        string
		jsonStr     string
		jsonPath    string
		expected    string
		shouldError bool
		errorType   string
	}{
		{
			name:     "simple string key",
			jsonStr:  `{"password": "secret123"}`,
			jsonPath: "password",
			expected: "secret123",
		},
		{
			name:     "simple number key",
			jsonStr:  `{"port": 5432}`,
			jsonPath: "port",
			expected: "5432",
		},
		{
			name:     "simple boolean key",
			jsonStr:  `{"enabled": true}`,
			jsonPath: "enabled",
			expected: "true",
		},
		{
			name:     "nested object access",
			jsonStr:  `{"database": {"host": "localhost"}}`,
			jsonPath: "database.host",
			expected: "localhost",
		},
		{
			name:     "deeply nested",
			jsonStr:  `{"db": {"primary": {"master": {"host": "db1.example.com"}}}}`,
			jsonPath: "db.primary.master.host",
			expected: "db1.example.com",
		},
		{
			name:     "null value",
			jsonStr:  `{"value": null}`,
			jsonPath: "value",
			expected: "",
		},
		{
			name:        "key not found",
			jsonStr:     `{"password": "secret"}`,
			jsonPath:    "missing",
			shouldError: true,
			errorType:   "key_not_found",
		},
		{
			name:        "type mismatch - string traversal",
			jsonStr:     `{"password": "secret"}`,
			jsonPath:    "password.nested",
			shouldError: true,
			errorType:   "type_mismatch",
		},
		{
			name:        "invalid json",
			jsonStr:     `{invalid json}`,
			jsonPath:    "key",
			shouldError: true,
			errorType:   "parse_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractor.ExtractValue(tt.jsonStr, tt.jsonPath, "test-param")

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error of type %s, but got nil", tt.errorType)
				} else if err.Type != tt.errorType {
					t.Errorf("Expected error type %s, but got %s", tt.errorType, err.Type)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

// Test empty JSONPath behavior
func TestEmptyJsonPathBehavior(t *testing.T) {
	extractor := NewJsonPathExtractor()

	jsonStr := `{"password": "secret123"}`
	result, err := extractor.ExtractValue(jsonStr, "", "test-param")

	if err != nil {
		t.Errorf("Expected no error for empty JSONPath, but got: %v", err)
	}

	if result != jsonStr {
		t.Errorf("Expected full JSON string for empty JSONPath, but got: %q", result)
	}
}
