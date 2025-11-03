package cli

import (
	"encoding/json"
	"testing"
)

// TestValidateJsonString tests JSON validation
func TestValidateJsonString(t *testing.T) {
	testCases := []struct {
		name    string
		json    string
		valid   bool
		description string
	}{
		{
			name:        "simple_object",
			json:        `{"key":"value"}`,
			valid:       true,
			description: "Simple JSON object",
		},
		{
			name:        "nested_object",
			json:        `{"parent":{"child":"value"}}`,
			valid:       true,
			description: "Nested JSON object",
		},
		{
			name:        "array",
			json:        `[1,2,3]`,
			valid:       true,
			description: "JSON array",
		},
		{
			name:        "with_numbers",
			json:        `{"port":5432,"timeout":30}`,
			valid:       true,
			description: "JSON with numbers",
		},
		{
			name:        "with_booleans",
			json:        `{"enabled":true,"debug":false}`,
			valid:       true,
			description: "JSON with booleans",
		},
		{
			name:        "with_null",
			json:        `{"value":null}`,
			valid:       true,
			description: "JSON with null value",
		},
		{
			name:        "invalid_json",
			json:        `{invalid}`,
			valid:       false,
			description: "Invalid JSON format",
		},
		{
			name:        "unclosed_brace",
			json:        `{"key":"value"`,
			valid:       false,
			description: "Unclosed JSON brace",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateJsonString(tc.json)
			if tc.valid && err != nil {
				t.Errorf("%s: expected valid JSON but got error: %v", tc.description, err)
			}
			if !tc.valid && err == nil {
				t.Errorf("%s: expected error but got none", tc.description)
			}
		})
	}
}

// TestFormatJsonForDisplay tests JSON formatting for display
func TestFormatJsonForDisplay(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		valid    bool
	}{
		{
			name:  "compact_object",
			input: `{"host":"localhost","port":5432}`,
			valid: true,
		},
		{
			name:  "nested_structure",
			input: `{"database":{"primary":{"host":"db1.example.com"}}}`,
			valid: true,
		},
		{
			name:  "invalid_json",
			input: `{invalid}`,
			valid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			formatted, err := formatJsonForDisplay(tc.input)
			if tc.valid && err != nil {
				t.Errorf("expected valid formatting but got error: %v", err)
			}
			if !tc.valid && err == nil {
				t.Errorf("expected error but got none")
			}
			if tc.valid && formatted == "" {
				t.Errorf("expected non-empty formatted JSON")
			}
		})
	}
}

// TestComplexJsonStructure tests setting complex JSON structures
func TestComplexJsonStructure(t *testing.T) {
	// Create a complex configuration structure
	config := map[string]interface{}{
		"database": map[string]interface{}{
			"primary": map[string]interface{}{
				"host":     "primary-db.example.com",
				"port":     5432,
				"username": "dbuser",
				"password": "secret",
			},
			"replica": map[string]interface{}{
				"host": "replica-db.example.com",
				"port": 5432,
			},
			"pool": map[string]interface{}{
				"min_size": 5,
				"max_size": 20,
			},
		},
		"server": map[string]interface{}{
			"host":    "0.0.0.0",
			"port":    8080,
			"timeout": 30,
		},
		"features": []string{"auth", "api", "cache"},
	}

	// Marshal to JSON
	jsonBytes, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Validate the JSON
	if err := validateJsonString(jsonStr); err != nil {
		t.Errorf("complex structure validation failed: %v", err)
	}

	// Format for display
	formatted, err := formatJsonForDisplay(jsonStr)
	if err != nil {
		t.Errorf("formatting failed: %v", err)
	}

	if formatted == "" {
		t.Errorf("formatted JSON is empty")
	}

	// Verify structure is preserved
	var parsedConfig map[string]interface{}
	if err := json.Unmarshal([]byte(formatted), &parsedConfig); err != nil {
		t.Errorf("failed to parse formatted JSON: %v", err)
	}

	// Check key structures
	if _, hasDB := parsedConfig["database"]; !hasDB {
		t.Errorf("database key missing from parsed JSON")
	}
	if _, hasServer := parsedConfig["server"]; !hasServer {
		t.Errorf("server key missing from parsed JSON")
	}
}

// TestJsonWithUnicode tests JSON with unicode characters
func TestJsonWithUnicode(t *testing.T) {
	jsonWithUnicode := `{"description":"データベース設定","name":"アプリケーション"}`

	if err := validateJsonString(jsonWithUnicode); err != nil {
		t.Errorf("unicode JSON validation failed: %v", err)
	}

	formatted, err := formatJsonForDisplay(jsonWithUnicode)
	if err != nil {
		t.Errorf("unicode JSON formatting failed: %v", err)
	}

	if formatted == "" {
		t.Errorf("formatted unicode JSON is empty")
	}
}

// TestJsonEdgeCases tests edge cases in JSON handling
func TestJsonEdgeCases(t *testing.T) {
	testCases := []struct {
		name  string
		json  string
		valid bool
	}{
		{
			name:  "empty_object",
			json:  `{}`,
			valid: true,
		},
		{
			name:  "empty_array",
			json:  `[]`,
			valid: true,
		},
		{
			name:  "single_string",
			json:  `"just a string"`,
			valid: true,
		},
		{
			name:  "single_number",
			json:  `42`,
			valid: true,
		},
		{
			name:  "whitespace",
			json:  `{  "key"  :  "value"  }`,
			valid: true,
		},
		{
			name:  "large_numbers",
			json:  `{"big":9999999999999999999}`,
			valid: true,
		},
		{
			name:  "escaped_strings",
			json:  `{"path":"C:\\Users\\test","quote":"He said \"hello\""}`,
			valid: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateJsonString(tc.json)
			if tc.valid && err != nil {
				t.Errorf("expected valid but got error: %v", err)
			}
			if !tc.valid && err == nil {
				t.Errorf("expected error but got none")
			}
		})
	}
}
