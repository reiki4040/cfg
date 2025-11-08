package cli

import (
	"encoding/json"
	"fmt"

	"github.com/reiki4040/cfg"
)

// JSONPathHelper provides JSONPath utilities for CLI operations
type JSONPathHelper struct {
	extractor *cfg.JsonPathExtractor
}

// NewJSONPathHelper creates a new JSONPath helper
func NewJSONPathHelper() *JSONPathHelper {
	return &JSONPathHelper{
		extractor: cfg.NewJsonPathExtractor(),
	}
}

// ExtractFromParameter extracts a value from a Parameter Store JSON value
func (h *JSONPathHelper) ExtractFromParameter(paramValue string, jsonPath string, paramName string) (string, error) {
	if jsonPath == "" {
		// No JSONPath specified, return the whole value
		return paramValue, nil
	}

	result, err := h.extractor.ExtractValue(paramValue, jsonPath, paramName)
	if err != nil {
		return "", fmt.Errorf("failed to extract JSONPath %s from parameter %s: %w", jsonPath, paramName, err)
	}

	return result, nil
}

// FormatJSONOutput formats JSON output with proper indentation
func FormatJSONOutput(jsonStr string) (string, error) {
	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// Pretty print with indentation
	formatted, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(formatted), nil
}

// FilterJSONByPath extracts specific keys from JSON using JSONPath
func FilterJSONByPath(jsonStr string, jsonPath string, paramName string) (string, error) {
	helper := NewJSONPathHelper()

	// Extract the value
	value, err := helper.ExtractFromParameter(jsonStr, jsonPath, paramName)
	if err != nil {
		return "", err
	}

	// Try to format as JSON if it looks like JSON
	if value != "" && (value[0] == '{' || value[0] == '[') {
		formatted, err := FormatJSONOutput(value)
		if err == nil {
			return formatted, nil
		}
	}

	return value, nil
}

// ExtractMultiplePaths extracts multiple JSONPath values from a single JSON parameter
func ExtractMultiplePaths(jsonStr string, paths []string, paramName string) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	helper := NewJSONPathHelper()

	for _, path := range paths {
		value, err := helper.ExtractFromParameter(jsonStr, path, paramName)
		if err != nil {
			results[path] = fmt.Sprintf("error: %v", err)
			continue
		}
		results[path] = value
	}

	return results, nil
}

// GetAvailableKeysFromJSON returns the available keys in a JSON object
func GetAvailableKeysFromJSON(jsonStr string) ([]string, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	keys := make([]string, 0, len(obj))
	for key := range obj {
		keys = append(keys, key)
	}

	return keys, nil
}

// ValidateJSONPath checks if a JSONPath is valid
func ValidateJSONPath(jsonPath string, paramName string) error {
	if jsonPath == "" {
		return nil // Empty JSONPath is valid (means no path)
	}

	// Create a simple JSON object to test validation
	testJSON := `{"test": "value"}`

	helper := NewJSONPathHelper()
	_, err := helper.ExtractFromParameter(testJSON, jsonPath, paramName)
	if err != nil {
		// Check if it's a key_not_found error (which is OK for validation)
		if _, ok := err.(*cfg.JSONPathError); !ok {
			return fmt.Errorf("invalid JSONPath format: %w", err)
		}
	}

	return nil
}
