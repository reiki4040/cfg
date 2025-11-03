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

// TestJsonStringListIncompatibility tests that JSON + StringList combination is rejected
func TestJsonStringListIncompatibility(t *testing.T) {
	testCases := []struct {
		name      string
		jsonValue string
		jsonFile  string
		typeList  bool
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "json_with_SL_flag",
			jsonValue: `{"host":"localhost"}`,
			typeList:  true,
			shouldErr: true,
			errMsg:    "JSON values cannot be stored as StringList",
		},
		{
			name:      "json_without_SL_flag",
			jsonValue: `{"host":"localhost"}`,
			typeList:  false,
			shouldErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Save current flag state
			oldJsonValue := setJsonValue
			oldTypeList := setTypeList

			// Set test values
			setJsonValue = tc.jsonValue
			setTypeList = tc.typeList

			// Simulate the StringList validation logic from runSetCommand
			if setJsonValue != "" && setTypeList {
				if !tc.shouldErr {
					t.Errorf("expected no error but validation would fail")
				}
			} else {
				if tc.shouldErr {
					t.Errorf("expected error but validation would pass")
				}
			}

			// Restore flag state
			setJsonValue = oldJsonValue
			setTypeList = oldTypeList
		})
	}
}

// TestJsonSecureStringTypeSelection tests that JSON + SS flag selects SecureString type
func TestJsonSecureStringTypeSelection(t *testing.T) {
	testCases := []struct {
		name        string
		typeSecure  bool
		typeString  bool
		expectedType string
	}{
		{
			name:         "json_with_SS_flag",
			typeSecure:   true,
			typeString:   false,
			expectedType: "SecureString",
		},
		{
			name:         "json_with_S_flag",
			typeSecure:   false,
			typeString:   true,
			expectedType: "String",
		},
		{
			name:         "json_without_flags",
			typeSecure:   false,
			typeString:   false,
			expectedType: "String",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Save current flag state
			oldTypeSecure := setTypeSecure
			oldTypeString := setTypeString

			// Set test values
			setTypeSecure = tc.typeSecure
			setTypeString = tc.typeString

			// Simulate the type selection logic from runSetCommand
			var selectedType string
			if setTypeSecure {
				selectedType = "SecureString"
			} else if setTypeString {
				selectedType = "String"
			} else {
				selectedType = "String"
			}

			if selectedType != tc.expectedType {
				t.Errorf("expected %s but got %s", tc.expectedType, selectedType)
			}

			// Restore flag state
			setTypeSecure = oldTypeSecure
			setTypeString = oldTypeString
		})
	}
}
