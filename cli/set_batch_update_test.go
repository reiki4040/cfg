package cli

import (
	"encoding/json"
	"testing"
)

// TestParseKeysOption tests parsing of the --keys option
func TestParseKeysOption(t *testing.T) {
	testCases := []struct {
		name          string
		keysStr       string
		expected      map[string]string
		shouldErr     bool
		errContains   string
		description   string
	}{
		{
			name:        "single_key_value_pair",
			keysStr:     "host=localhost",
			expected:    map[string]string{"host": "localhost"},
			shouldErr:   false,
			description: "Single key=value pair",
		},
		{
			name:    "multiple_key_value_pairs",
			keysStr: "host=localhost,port=5432,user=admin",
			expected: map[string]string{
				"host": "localhost",
				"port": "5432",
				"user": "admin",
			},
			shouldErr:   false,
			description: "Multiple key=value pairs",
		},
		{
			name:    "nested_keys",
			keysStr: "database.host=localhost,database.port=5432",
			expected: map[string]string{
				"database.host": "localhost",
				"database.port": "5432",
			},
			shouldErr:   false,
			description: "Nested JSON path keys",
		},
		{
			name:        "key_with_empty_value",
			keysStr:     "emptyKey=",
			expected:    map[string]string{"emptyKey": ""},
			shouldErr:   false,
			description: "Key with empty value",
		},
		{
			name:        "whitespace_around_keys",
			keysStr:     "  host  =  localhost  ,  port  =  5432  ",
			expected:    map[string]string{"host": "localhost", "port": "5432"},
			shouldErr:   false,
			description: "Whitespace around keys and values",
		},
		{
			name:        "empty_string",
			keysStr:     "",
			expected:    map[string]string{},
			shouldErr:   false,
			description: "Empty string returns empty map",
		},
		{
			name:        "invalid_no_equals",
			keysStr:     "host:localhost",
			shouldErr:   true,
			errContains: "invalid key=value pair format",
			description: "Invalid format without equals sign",
		},
		{
			name:        "invalid_empty_key",
			keysStr:     "=value",
			shouldErr:   true,
			errContains: "empty key path",
			description: "Empty key path",
		},
		{
			name:        "invalid_multiple_equals",
			keysStr:     "key=value=extra",
			expected:    map[string]string{"key": "value=extra"},
			shouldErr:   false,
			description: "Multiple equals signs (value contains equals)",
		},
		{
			name:        "values_with_special_chars",
			keysStr:     "url=http://example.com:8080,path=/api/v1/endpoint",
			expected:    map[string]string{"url": "http://example.com:8080", "path": "/api/v1/endpoint"},
			shouldErr:   false,
			description: "Values with special characters (colons, slashes)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseKeysOption(tc.keysStr)

			if tc.shouldErr {
				if err == nil {
					t.Errorf("Expected error, but got nil")
				}
				if tc.errContains != "" && err != nil {
					errMsg := err.Error()
					if !stringContains(errMsg, tc.errContains) {
						t.Errorf("Expected error containing '%s', got '%s'", tc.errContains, errMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !mapEqual(result, tc.expected) {
					t.Errorf("Expected %v, got %v", tc.expected, result)
				}
			}
		})
	}
}

// TestBatchUpdateAttributes tests batch updating of JSON attributes
func TestBatchUpdateAttributes(t *testing.T) {
	testCases := []struct {
		name        string
		originalJSON string
		updates     map[string]string
		expected    string
		shouldErr   bool
		errContains string
		description string
	}{
		{
			name:         "single_update",
			originalJSON: `{"host":"oldhost","port":5432}`,
			updates:      map[string]string{"host": "newhost"},
			expected:     `{"host":"newhost","port":5432}`,
			shouldErr:    false,
			description:  "Single attribute update",
		},
		{
			name:    "multiple_updates",
			originalJSON: `{"host":"oldhost","port":5432,"user":"admin"}`,
			updates: map[string]string{
				"host": "newhost",
				"port": "3306",
			},
			expected:    `{"host":"newhost","port":"3306","user":"admin"}`,
			shouldErr:   false,
			description: "Multiple attribute updates",
		},
		{
			name:         "nested_attribute_update",
			originalJSON: `{"database":{"host":"localhost","port":5432}}`,
			updates:      map[string]string{"database.host": "remotehost"},
			expected:     `{"database":{"host":"remotehost","port":5432}}`,
			shouldErr:    false,
			description:  "Nested attribute update",
		},
		{
			name:    "multiple_nested_updates",
			originalJSON: `{"database":{"host":"localhost","port":5432,"user":"admin"}}`,
			updates: map[string]string{
				"database.host": "remotehost",
				"database.port": "3306",
			},
			expected:    `{"database":{"host":"remotehost","port":"3306","user":"admin"}}`,
			shouldErr:   false,
			description: "Multiple nested attribute updates",
		},
		{
			name:         "new_key_creation",
			originalJSON: `{"host":"localhost"}`,
			updates:      map[string]string{"port": "5432"},
			expected:     `{"host":"localhost","port":"5432"}`,
			shouldErr:    false,
			description:  "Creating new attribute",
		},
		{
			name:    "mixed_updates_and_creation",
			originalJSON: `{"host":"localhost","user":"admin"}`,
			updates: map[string]string{
				"host": "newhost",
				"port": "5432",
			},
			expected:    `{"host":"newhost","user":"admin","port":"5432"}`,
			shouldErr:   false,
			description: "Mixed updates and new attributes",
		},
		{
			name:        "empty_updates",
			originalJSON: `{"host":"localhost"}`,
			updates:     map[string]string{},
			expected:    `{"host":"localhost"}`,
			shouldErr:   false,
			description: "No updates (empty map)",
		},
		{
			name:         "invalid_json",
			originalJSON: `{invalid json}`,
			updates:      map[string]string{"host": "newhost"},
			shouldErr:    true,
			errContains:  "failed to update attribute",
			description:  "Invalid JSON input",
		},
		{
			name:         "invalid_keypath",
			originalJSON: `{"host":"localhost"}`,
			updates:      map[string]string{"..invalid": "value"},
			shouldErr:    true,
			errContains:  "failed to update attribute",
			description:  "Invalid key path (path traversal)",
		},
		{
			name:         "overwrite_with_empty_value",
			originalJSON: `{"host":"localhost","port":5432}`,
			updates:      map[string]string{"host": ""},
			expected:     `{"host":"","port":5432}`,
			shouldErr:    false,
			description:  "Overwrite attribute with empty value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := batchUpdateAttributes(tc.originalJSON, tc.updates)

			if tc.shouldErr {
				if err == nil {
					t.Errorf("Expected error, but got nil")
				}
				if tc.errContains != "" && err != nil {
					errMsg := err.Error()
					if !stringContains(errMsg, tc.errContains) {
						t.Errorf("Expected error containing '%s', got '%s'", tc.errContains, errMsg)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				// Compare JSON objects rather than strings to handle formatting differences
				if !jsonObjectsEqual(result, tc.expected) {
					t.Errorf("Expected %s, got %s", tc.expected, result)
				}
			}
		})
	}
}

// Helper functions

// stringContains checks if a string contains a substring
func stringContains(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// mapEqual compares two maps for equality
func mapEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

// jsonObjectsEqual compares two JSON strings for semantic equality
func jsonObjectsEqual(json1, json2 string) bool {
	var obj1, obj2 interface{}
	if err := json.Unmarshal([]byte(json1), &obj1); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(json2), &obj2); err != nil {
		return false
	}
	return jsonEqual(obj1, obj2)
}

// jsonEqual compares two JSON objects for semantic equality
func jsonEqual(a, b interface{}) bool {
	// Compare JSON objects recursively
	switch av := a.(type) {
	case map[string]interface{}:
		if bv, ok := b.(map[string]interface{}); !ok {
			return false
		} else {
			if len(av) != len(bv) {
				return false
			}
			for k, v := range av {
				if bval, ok := bv[k]; !ok || !jsonEqual(v, bval) {
					return false
				}
			}
			return true
		}
	case []interface{}:
		if bv, ok := b.([]interface{}); !ok {
			return false
		} else {
			if len(av) != len(bv) {
				return false
			}
			for i, v := range av {
				if !jsonEqual(v, bv[i]) {
					return false
				}
			}
			return true
		}
	default:
		return a == b
	}
}
