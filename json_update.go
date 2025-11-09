package cfg

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseJSON parses a JSON string and validates it as a JSON object.
// It ensures the input is a valid JSON object (not null or other types).
// Returns a map[string]interface{} for the parsed JSON object, or an error if parsing fails.
//
// Example:
//
//	obj, err := ParseJSON(`{"key":"value","nested":{"inner":"data"}}`)
func ParseJSON(jsonStr string) (map[string]interface{}, error) {
	if jsonStr == "" {
		return nil, &JsonParseError{
			Message: "empty JSON string",
			Input:   jsonStr,
		}
	}

	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, &JsonParseError{
			Message: "invalid JSON format",
			Input:   jsonStr,
			Cause:   err,
		}
	}

	// Validate that we got an object, not other JSON types
	if result == nil {
		return nil, &JsonParseError{
			Message: "JSON must be an object, not null or other type",
			Input:   jsonStr,
		}
	}

	return result, nil
}

// ParseKeyPath parses a dot-notation key path into individual keys.
// It validates the path format to prevent path traversal attacks.
// Valid format: "key" or "parent.child.grandchild"
// Invalid formats: "" (empty), ".." (double dots), ".key" (leading dot), "key." (trailing dot)
//
// Example:
//
//	keys, err := ParseKeyPath("database.connection.host")
//	// keys = ["database", "connection", "host"]
func ParseKeyPath(keyPath string) ([]string, error) {
	if keyPath == "" {
		return nil, &JsonPathError{
			Path:    keyPath,
			Message: "key path cannot be empty",
		}
	}

	// Check for invalid patterns (security: prevent path traversal)
	if strings.Contains(keyPath, "..") {
		return nil, &JsonPathError{
			Path:    keyPath,
			Message: "double dots (..) not allowed - possible path traversal attempt",
			Details: "use single dots to separate nested keys",
		}
	}

	if strings.HasPrefix(keyPath, ".") {
		return nil, &JsonPathError{
			Path:    keyPath,
			Message: "key path cannot start with a dot",
		}
	}

	if strings.HasSuffix(keyPath, ".") {
		return nil, &JsonPathError{
			Path:    keyPath,
			Message: "key path cannot end with a dot",
		}
	}

	keys := strings.Split(keyPath, ".")
	for i, key := range keys {
		if key == "" {
			return nil, &JsonPathError{
				Path:    keyPath,
				Message: fmt.Sprintf("empty key at position %d", i),
			}
		}
	}

	return keys, nil
}

// UpdateJsonAttribute updates or creates an attribute in a JSON object.
// It safely parses the JSON, updates the specified key path, and returns the modified JSON string.
//
// Parameters:
//
//	jsonStr: the original JSON string to modify
//	keyPath: the dot-notation path to the attribute (e.g., "database.connection.host")
//	newValue: the new value as a string (will be stored as a string in JSON)
//
// Returns the updated JSON string, or an error if:
//   - JSON parsing fails
//   - keyPath format is invalid
//   - attempting to traverse through non-object values
//   - JSON serialization fails
//
// Example:
//
//	result, err := UpdateJsonAttribute(
//	    `{"database":{"host":"localhost"}}`,
//	    "database.host",
//	    "newhost",
//	)
//	// result = `{"database":{"host":"newhost"}}`
func UpdateJsonAttribute(jsonStr, keyPath, newValue string) (string, error) {
	// Parse the JSON input
	obj, err := ParseJSON(jsonStr)
	if err != nil {
		return "", err
	}

	// Parse the key path
	keys, err := ParseKeyPath(keyPath)
	if err != nil {
		return "", err
	}

	// Update the object with the new value
	if err := updateObjectWithKeys(obj, keys, newValue); err != nil {
		return "", err
	}

	// Marshal back to JSON string
	updatedJSON, err := json.Marshal(obj)
	if err != nil {
		return "", &JsonUpdateError{
			Path:    keyPath,
			Message: "failed to serialize updated JSON",
			Reason:  "marshal_error",
		}
	}

	return string(updatedJSON), nil
}

// updateObjectWithKeys recursively updates an object following the key path.
// It creates intermediate objects as needed and validates that non-leaf keys point to objects.
//
// Algorithm:
// 1. Extract the first key from the path
// 2. If it's the final key, set its value directly
// 3. If more keys remain, traverse deeper:
//   - Create intermediate object if key doesn't exist
//   - Validate that the value at this key is an object (map)
//   - Recursively call with remaining keys
//
// Error cases:
// - Type mismatch: attempting to traverse through non-object values (strings, numbers, etc.)
func updateObjectWithKeys(obj map[string]interface{}, keys []string, newValue string) error {
	if len(keys) == 0 {
		return nil
	}

	// Separate the first key from remaining keys
	currentKey := keys[0]
	remainingKeys := keys[1:]

	// Base case: this is the final key - set the value directly
	if len(remainingKeys) == 0 {
		obj[currentKey] = newValue
		return nil
	}

	// Recursive case: need to traverse deeper
	// Check if key exists and get its value
	nestedValue, exists := obj[currentKey]

	// If key doesn't exist, create an empty nested object
	if !exists {
		obj[currentKey] = make(map[string]interface{})
		nestedValue = obj[currentKey]
	}

	// Type assertion: ensure the value is a map (JSON object)
	// This prevents attempting to traverse through primitive values (strings, numbers, etc.)
	nestedObj, ok := nestedValue.(map[string]interface{})
	if !ok {
		// The key exists but is not an object - cannot traverse
		return &JsonUpdateError{
			Path:    currentKey,
			Message: fmt.Sprintf("cannot traverse through non-object value of type %T", nestedValue),
			Reason:  "type_mismatch",
		}
	}

	// Recursively update the nested object with remaining path
	return updateObjectWithKeys(nestedObj, remainingKeys, newValue)
}

// FormatJsonForDisplay returns pretty-printed JSON for display purposes.
// It indents the JSON with 2 spaces for better readability.
//
// Example:
//
//	formatted, err := FormatJsonForDisplay(`{"key":"value","nested":{"inner":"data"}}`)
//	// formatted = "{\n  \"key\": \"value\",\n  \"nested\": {\n    \"inner\": \"data\"\n  }\n}"
func FormatJsonForDisplay(jsonStr string) (string, error) {
	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return "", err
	}

	formatted, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", err
	}

	return string(formatted), nil
}

// ClearSensitiveData securely clears sensitive data from memory by zero-filling the byte slice.
// This is a security consideration to prevent sensitive data from lingering in memory
// and being recoverable through memory dumps or forensic analysis.
//
// Example:
//
//	sensitiveBytes := []byte("password123")
//	defer ClearSensitiveData(sensitiveBytes)
func ClearSensitiveData(data []byte) {
	if len(data) > 0 {
		for i := range data {
			data[i] = 0
		}
	}
}
