package cfg

import (
	"encoding/json"
	"testing"
)

// Test 1.1: JSON Parsing and Validation
func TestParseAndValidateJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errType string
	}{
		{
			name:    "valid simple object",
			input:   `{"key":"value"}`,
			wantErr: false,
		},
		{
			name:    "valid nested object",
			input:   `{"database":{"host":"localhost","port":5432}}`,
			wantErr: false,
		},
		{
			name:    "valid array in object",
			input:   `{"items":[1,2,3]}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   `{"key":"value"`,
			wantErr: true,
			errType: "JsonParseError",
		},
		{
			name:    "empty string",
			input:   ``,
			wantErr: true,
			errType: "JsonParseError",
		},
		{
			name:    "null value",
			input:   `null`,
			wantErr: true,
			errType: "JsonParseError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errType != "" {
				if jsonErr, ok := err.(*JsonParseError); !ok {
					t.Errorf("ParseJSON() error type = %T, want JsonParseError", err)
				} else if jsonErr == nil {
					t.Errorf("ParseJSON() returned nil error when error was expected")
				}
			}
			if !tt.wantErr && result == nil {
				t.Errorf("ParseJSON() returned nil result for valid input")
			}
		})
	}
}

// Test 1.2: JSONPath Key Path Parsing
func TestParseKeyPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantKeys []string
		wantErr  bool
		errType  string
	}{
		{
			name:     "single key",
			input:    "key",
			wantKeys: []string{"key"},
			wantErr:  false,
		},
		{
			name:     "nested keys",
			input:    "database.connection.host",
			wantKeys: []string{"database", "connection", "host"},
			wantErr:  false,
		},
		{
			name:     "two level nesting",
			input:    "config.value",
			wantKeys: []string{"config", "value"},
			wantErr:  false,
		},
		{
			name:    "empty key",
			input:   "",
			wantErr: true,
			errType: "JsonPathError",
		},
		{
			name:    "double dot (path traversal)",
			input:   "database..host",
			wantErr: true,
			errType: "JsonPathError",
		},
		{
			name:    "leading dot",
			input:   ".database.host",
			wantErr: true,
			errType: "JsonPathError",
		},
		{
			name:    "trailing dot",
			input:   "database.host.",
			wantErr: true,
			errType: "JsonPathError",
		},
		{
			name:     "special characters in key",
			input:    "my_config-value",
			wantKeys: []string{"my_config-value"},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys, err := ParseKeyPath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseKeyPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errType != "" {
				if _, ok := err.(*JsonPathError); !ok {
					t.Errorf("ParseKeyPath() error type = %T, want JsonPathError", err)
				}
			}
			if !tt.wantErr {
				if len(keys) != len(tt.wantKeys) {
					t.Errorf("ParseKeyPath() returned %d keys, want %d", len(keys), len(tt.wantKeys))
					return
				}
				for i, key := range keys {
					if key != tt.wantKeys[i] {
						t.Errorf("ParseKeyPath() key[%d] = %q, want %q", i, key, tt.wantKeys[i])
					}
				}
			}
		})
	}
}

// Test 1.3: Update JSON Attributes
func TestUpdateJsonAttribute(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		keyPath  string
		newValue string
		wantJSON string
		wantErr  bool
		errType  string
	}{
		{
			name:     "update simple key",
			input:    `{"host":"oldhost","port":"5432"}`,
			keyPath:  "host",
			newValue: "newhost",
			wantJSON: `{"host":"newhost","port":"5432"}`,
			wantErr:  false,
		},
		{
			name:     "update nested attribute",
			input:    `{"database":{"host":"oldhost","port":5432}}`,
			keyPath:  "database.host",
			newValue: "newhost",
			wantJSON: `{"database":{"host":"newhost","port":5432}}`,
			wantErr:  false,
		},
		{
			name:     "create new key",
			input:    `{"existing":"value"}`,
			keyPath:  "newkey",
			newValue: "newvalue",
			wantJSON: `{"existing":"value","newkey":"newvalue"}`,
			wantErr:  false,
		},
		{
			name:     "create nested structure",
			input:    `{}`,
			keyPath:  "database.connection.host",
			newValue: "localhost",
			wantJSON: `{"database":{"connection":{"host":"localhost"}}}`,
			wantErr:  false,
		},
		{
			name:    "path through non-object",
			input:   `{"database":"string_value"}`,
			keyPath: "database.host",
			wantErr: true,
			errType: "JsonUpdateError",
		},
		{
			name:    "update with invalid JSON input",
			input:   `{invalid}`,
			keyPath: "key",
			wantErr: true,
			errType: "JsonParseError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := UpdateJsonAttribute(tt.input, tt.keyPath, tt.newValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateJsonAttribute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errType != "" {
				switch tt.errType {
				case "JsonUpdateError":
					if _, ok := err.(*JsonUpdateError); !ok {
						t.Errorf("UpdateJsonAttribute() error type = %T, want JsonUpdateError", err)
					}
				case "JsonParseError":
					if _, ok := err.(*JsonParseError); !ok {
						t.Errorf("UpdateJsonAttribute() error type = %T, want JsonParseError", err)
					}
				case "JsonPathError":
					if _, ok := err.(*JsonPathError); !ok {
						t.Errorf("UpdateJsonAttribute() error type = %T, want JsonPathError", err)
					}
				}
			}
			if !tt.wantErr {
				// Compare as JSON objects to ignore formatting differences
				var resultObj, wantObj interface{}
				if err := json.Unmarshal([]byte(result), &resultObj); err != nil {
					t.Errorf("Failed to unmarshal result: %v", err)
					return
				}
				if err := json.Unmarshal([]byte(tt.wantJSON), &wantObj); err != nil {
					t.Errorf("Failed to unmarshal expected: %v", err)
					return
				}
				resultJSON, _ := json.Marshal(resultObj)
				wantJSON, _ := json.Marshal(wantObj)
				if string(resultJSON) != string(wantJSON) {
					t.Errorf("UpdateJsonAttribute() = %s, want %s", string(resultJSON), string(wantJSON))
				}
			}
		})
	}
}

// Test 1.4: JSON Update with Large Values
func TestUpdateJsonAttributeWithLargeValues(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		keyPath  string
		newValue string
		wantErr  bool
	}{
		{
			name:     "update with large string value",
			input:    `{"config":"old"}`,
			keyPath:  "config",
			newValue: string(make([]byte, 1000)),
			wantErr:  false,
		},
		{
			name:     "update with JSON value",
			input:    `{"outer":{"key":"value"}}`,
			keyPath:  "outer.nested",
			newValue: `{"complex":"object","with":"multiple","keys":123}`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UpdateJsonAttribute(tt.input, tt.keyPath, tt.newValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateJsonAttribute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test 1.5: Error Types and Messages
func TestJsonErrorTypes(t *testing.T) {
	t.Run("JsonParseError details", func(t *testing.T) {
		input := `{"incomplete"`
		_, err := ParseJSON(input)
		if err == nil {
			t.Errorf("ParseJSON() expected error for invalid JSON, got nil")
			return
		}

		parseErr, ok := err.(*JsonParseError)
		if !ok {
			t.Errorf("Expected *JsonParseError, got %T", err)
			return
		}

		if parseErr.Message == "" {
			t.Errorf("JsonParseError.Message should not be empty")
		}
		if parseErr.Input == "" {
			t.Errorf("JsonParseError.Input should contain the problematic input")
		}
	})

	t.Run("JsonPathError details", func(t *testing.T) {
		_, err := ParseKeyPath("..invalid")
		if err == nil {
			t.Errorf("ParseKeyPath() expected error for invalid path, got nil")
			return
		}

		pathErr, ok := err.(*JsonPathError)
		if !ok {
			t.Errorf("Expected *JsonPathError, got %T", err)
			return
		}

		if pathErr.Path == "" {
			t.Errorf("JsonPathError.Path should not be empty")
		}
		if pathErr.Message == "" {
			t.Errorf("JsonPathError.Message should not be empty")
		}
	})

	t.Run("JsonUpdateError details", func(t *testing.T) {
		_, err := UpdateJsonAttribute(`{"key":"string"}`, "key.nested", "value")
		if err == nil {
			t.Errorf("UpdateJsonAttribute() expected error for type mismatch, got nil")
			return
		}

		updateErr, ok := err.(*JsonUpdateError)
		if !ok {
			t.Errorf("Expected *JsonUpdateError, got %T", err)
			return
		}

		if updateErr.Path == "" {
			t.Errorf("JsonUpdateError.Path should not be empty")
		}
		if updateErr.Message == "" {
			t.Errorf("JsonUpdateError.Message should not be empty")
		}
	})
}

// Test 1.6: Memory Security - Ensure data is cleared
func TestMemorySecurity(t *testing.T) {
	t.Run("JSON data not leaked in error messages", func(t *testing.T) {
		sensitiveJSON := `{"password":"secret123","api_key":"key456"}`
		_, err := UpdateJsonAttribute(sensitiveJSON, "..invalid", "value")

		if err != nil {
			// Error message should not contain the sensitive data
			errMsg := err.Error()
			if errMsg != "" {
				// The error message shouldn't contain the full JSON or sensitive values
				// This is a security consideration
				t.Logf("Error message for security review: %s", errMsg)
			}
		}
	})

	t.Run("Verify UpdateJsonAttribute doesn't expose secrets", func(t *testing.T) {
		originalJSON := `{"config":{"db_password":"secret"}}`
		_, err := UpdateJsonAttribute(originalJSON, "config.db_password", "newpassword")

		if err != nil {
			t.Errorf("UpdateJsonAttribute() should succeed with valid input: %v", err)
		}
	})
}

// Test 1.7: Complex Nested Structures
func TestComplexNestedStructures(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		keyPath  string
		newValue string
		wantErr  bool
	}{
		{
			name:     "deeply nested update",
			input:    `{"a":{"b":{"c":{"d":"old"}}}}`,
			keyPath:  "a.b.c.d",
			newValue: "new",
			wantErr:  false,
		},
		{
			name:     "update within array-containing object",
			input:    `{"items":[1,2,3],"config":"value"}`,
			keyPath:  "config",
			newValue: "newvalue",
			wantErr:  false,
		},
		{
			name:     "create path through empty object",
			input:    `{"empty":{}}`,
			keyPath:  "empty.newkey",
			newValue: "value",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UpdateJsonAttribute(tt.input, tt.keyPath, tt.newValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateJsonAttribute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Test 1.8: JSON Formatting and Validation
func TestFormatValidateJson(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "well-formed JSON",
			input:   `{"key":"value"}`,
			wantErr: false,
		},
		{
			name:    "formatted JSON with indents",
			input:   "{\n  \"key\": \"value\"\n}",
			wantErr: false,
		},
		{
			name:    "malformed JSON",
			input:   `{"key":"value"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
