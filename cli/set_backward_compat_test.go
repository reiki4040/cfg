package cli

import (
	"testing"
)

// TestBackwardCompatibilityParseJsonPath tests that parseJsonPath correctly handles non-JSONPath cases
// This verifies backward compatibility with standard parameter paths
func TestBackwardCompatibilityParseJsonPath(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectPath     string
		expectJsonPath string
		description    string
	}{
		{
			name:           "standard_param_path_simple",
			input:          "/config",
			expectPath:     "/config",
			expectJsonPath: "",
			description:    "Simple parameter path without JSONPath should work as before",
		},
		{
			name:           "standard_param_path_nested",
			input:          "/app/prod/config",
			expectPath:     "/app/prod/config",
			expectJsonPath: "",
			description:    "Nested parameter path without JSONPath should work as before",
		},
		{
			name:           "standard_param_path_with_stage",
			input:          "/app/{stage}/db/password",
			expectPath:     "/app/{stage}/db/password",
			expectJsonPath: "",
			description:    "Parameter path with stage placeholder should work as before",
		},
		{
			name:           "standard_param_path_complex",
			input:          "/myapp/stage-prod/config/db/primary",
			expectPath:     "/myapp/stage-prod/config/db/primary",
			expectJsonPath: "",
			description:    "Complex parameter path should work as before",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path, jsonPath, err := parseJsonPath(tc.input)
			if err != nil {
				t.Errorf("parseJsonPath() unexpected error: %v", err)
				return
			}
			if path != tc.expectPath {
				t.Errorf("%s: parseJsonPath() path = %q, want %q", tc.description, path, tc.expectPath)
			}
			if jsonPath != tc.expectJsonPath {
				t.Errorf("%s: parseJsonPath() jsonPath = %q, want %q", tc.description, jsonPath, tc.expectJsonPath)
			}
		})
	}
}

// TestNormalizeParameterType tests that parameter type normalization still works correctly
func TestNormalizeParameterType(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expected    string
		shouldErr   bool
		description string
	}{
		{
			name:        "lowercase_string",
			input:       "string",
			expected:    "String",
			shouldErr:   false,
			description: "Lowercase 'string' should normalize to 'String'",
		},
		{
			name:        "lowercase_securestring",
			input:       "securestring",
			expected:    "SecureString",
			shouldErr:   false,
			description: "Lowercase 'securestring' should normalize to 'SecureString'",
		},
		{
			name:        "lowercase_stringlist",
			input:       "stringlist",
			expected:    "StringList",
			shouldErr:   false,
			description: "Lowercase 'stringlist' should normalize to 'StringList'",
		},
		{
			name:        "already_normalized_string",
			input:       "String",
			expected:    "String",
			shouldErr:   false,
			description: "Already normalized 'String' should remain unchanged",
		},
		{
			name:        "already_normalized_securestring",
			input:       "SecureString",
			expected:    "SecureString",
			shouldErr:   false,
			description: "Already normalized 'SecureString' should remain unchanged",
		},
		{
			name:        "invalid_type",
			input:       "InvalidType",
			expected:    "",
			shouldErr:   true,
			description: "Invalid type should return error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := normalizeParameterType(tc.input)
			if (err != nil) != tc.shouldErr {
				t.Errorf("%s: normalizeParameterType() error = %v, shouldErr %v", tc.description, err, tc.shouldErr)
				return
			}
			if result != tc.expected {
				t.Errorf("%s: normalizeParameterType() = %q, want %q", tc.description, result, tc.expected)
			}
		})
	}
}

// TestIsValidParameterType tests parameter type validation
func TestIsValidParameterType(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expected    bool
		description string
	}{
		{
			name:        "lowercase_string",
			input:       "string",
			expected:    true,
			description: "Lowercase 'string' should be valid",
		},
		{
			name:        "uppercase_string",
			input:       "String",
			expected:    true,
			description: "Uppercase 'String' is considered valid (case-insensitive check)",
		},
		{
			name:        "securestring",
			input:       "securestring",
			expected:    true,
			description: "Lowercase 'securestring' should be valid",
		},
		{
			name:        "stringlist",
			input:       "stringlist",
			expected:    true,
			description: "Lowercase 'stringlist' should be valid",
		},
		{
			name:        "invalid_type",
			input:       "InvalidType",
			expected:    false,
			description: "Invalid type should return false",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isValidParameterType(tc.input)
			if result != tc.expected {
				t.Errorf("%s: isValidParameterType() = %v, want %v", tc.description, result, tc.expected)
			}
		})
	}
}

// TestValidateJsonStringBackwardCompat tests that validateJsonString continues to work as before
func TestValidateJsonStringBackwardCompat(t *testing.T) {
	testCases := []struct {
		name        string
		json        string
		shouldErr   bool
		description string
	}{
		{
			name:        "simple_object",
			json:        `{"key":"value"}`,
			shouldErr:   false,
			description: "Simple JSON object should validate",
		},
		{
			name:        "empty_object",
			json:        `{}`,
			shouldErr:   false,
			description: "Empty JSON object should validate",
		},
		{
			name:        "array",
			json:        `[1,2,3]`,
			shouldErr:   false,
			description: "JSON array should validate",
		},
		{
			name:        "invalid_json",
			json:        `{invalid}`,
			shouldErr:   true,
			description: "Invalid JSON should error",
		},
		{
			name:        "unclosed_brace",
			json:        `{"key":"value"`,
			shouldErr:   true,
			description: "Unclosed JSON should error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateJsonString(tc.json)
			if (err != nil) != tc.shouldErr {
				t.Errorf("%s: validateJsonString() error = %v, shouldErr %v", tc.description, err, tc.shouldErr)
			}
		})
	}
}

// TestFormatJsonForDisplayBackwardCompat tests that formatJsonForDisplay continues to work
func TestFormatJsonForDisplayBackwardCompat(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		shouldErr    bool
		shouldFormat bool
		description  string
	}{
		{
			name:         "compact_json",
			input:        `{"host":"localhost","port":5432}`,
			shouldErr:    false,
			shouldFormat: true,
			description:  "Compact JSON should be formatted with indentation",
		},
		{
			name:         "nested_json",
			input:        `{"database":{"host":"localhost","port":5432}}`,
			shouldErr:    false,
			shouldFormat: true,
			description:  "Nested JSON should be formatted",
		},
		{
			name:         "already_formatted",
			input:        "{\n  \"host\": \"localhost\"\n}",
			shouldErr:    false,
			shouldFormat: true,
			description:  "Already formatted JSON should remain valid",
		},
		{
			name:         "invalid_json",
			input:        `{invalid}`,
			shouldErr:    true,
			shouldFormat: false,
			description:  "Invalid JSON should error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := formatJsonForDisplay(tc.input)
			if (err != nil) != tc.shouldErr {
				t.Errorf("%s: formatJsonForDisplay() error = %v, shouldErr %v", tc.description, err, tc.shouldErr)
				return
			}
			if tc.shouldFormat && result == "" {
				t.Errorf("%s: formatJsonForDisplay() returned empty string", tc.description)
			}
			if !tc.shouldErr {
				// Verify the output is valid JSON
				if err := validateJsonString(result); err != nil {
					t.Errorf("%s: formatJsonForDisplay() output is not valid JSON: %v", tc.description, err)
				}
			}
		})
	}
}

// TestNonJsonPathParameterOperations tests that non-JSONPath parameter operations are unaffected
// by the JSONPath implementation
func TestNonJsonPathParameterOperations(t *testing.T) {
	testCases := []struct {
		name               string
		paramPath          string
		shouldHaveJsonPath bool
		description        string
	}{
		{
			name:               "simple_param_set",
			paramPath:          "/config",
			shouldHaveJsonPath: false,
			description:        "Simple parameter path should not have JSONPath",
		},
		{
			name:               "nested_param_set",
			paramPath:          "/app/prod/config",
			shouldHaveJsonPath: false,
			description:        "Nested parameter should not have JSONPath",
		},
		{
			name:               "with_underscore",
			paramPath:          "/my_app/prod_config",
			shouldHaveJsonPath: false,
			description:        "Parameter with underscore should not have JSONPath",
		},
		{
			name:               "with_hyphen",
			paramPath:          "/my-app/prod-config",
			shouldHaveJsonPath: false,
			description:        "Parameter with hyphen should not have JSONPath",
		},
		{
			name:               "with_numbers",
			paramPath:          "/app1/config2/value3",
			shouldHaveJsonPath: false,
			description:        "Parameter with numbers should not have JSONPath",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path, jsonPath, err := parseJsonPath(tc.paramPath)
			if err != nil {
				t.Errorf("%s: parseJsonPath() unexpected error: %v", tc.description, err)
				return
			}
			if path != tc.paramPath {
				t.Errorf("%s: path changed from %q to %q", tc.description, tc.paramPath, path)
			}
			hasJsonPath := jsonPath != ""
			if hasJsonPath != tc.shouldHaveJsonPath {
				t.Errorf("%s: jsonPath presence = %v, want %v", tc.description, hasJsonPath, tc.shouldHaveJsonPath)
			}
		})
	}
}
