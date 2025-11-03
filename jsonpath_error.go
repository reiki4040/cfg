package cfg

import "fmt"

// JSONPathError represents errors that occur during JSONPath parsing and extraction
type JSONPathError struct {
	Type          string   // "invalid_format", "parse_error", "key_not_found", "type_mismatch"
	JSONPath      string   // The JSONPath that caused the error
	Parameter     string   // The Parameter Store parameter name
	Message       string   // Descriptive error message
	Cause         error    // Underlying error, if any
	AvailableKeys []string // Available keys in the JSON object (for key_not_found)
	ExpectedType  string   // Expected type in JSON traversal (for type_mismatch)
	ActualType    string   // Actual type found (for type_mismatch)
}

// Error implements the error interface
func (e *JSONPathError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("jsonpath error (%s): %s at %s:%s (caused by: %v)", e.Type, e.Message, e.Parameter, e.JSONPath, e.Cause)
	}
	return fmt.Sprintf("jsonpath error (%s): %s at %s:%s", e.Type, e.Message, e.Parameter, e.JSONPath)
}

// InvalidJSONPathFormatError creates an error for invalid JSONPath format
func InvalidJSONPathFormatError(jsonPath string, parameter string, reason string) *JSONPathError {
	return &JSONPathError{
		Type:      "invalid_format",
		JSONPath:  jsonPath,
		Parameter: parameter,
		Message:   fmt.Sprintf("invalid JSONPath format: %s", reason),
	}
}

// JSONParseError creates an error for JSON parsing failures
func JSONParseError(parameter string, cause error) *JSONPathError {
	return &JSONPathError{
		Type:      "parse_error",
		Parameter: parameter,
		Message:   "failed to parse JSON parameter value",
		Cause:     cause,
	}
}

// KeyNotFoundError creates an error when a JSONPath key is not found in the JSON object
func KeyNotFoundError(jsonPath string, parameter string, availableKeys []string) *JSONPathError {
	keysStr := fmt.Sprintf("%v", availableKeys)
	return &JSONPathError{
		Type:          "key_not_found",
		JSONPath:      jsonPath,
		Parameter:     parameter,
		Message:       fmt.Sprintf("key not found in JSON object. Available keys: %s", keysStr),
		AvailableKeys: availableKeys,
	}
}

// TypeMismatchError creates an error when path traversal encounters a non-object type
func TypeMismatchError(jsonPath string, parameter string, pathSegment string, foundType string) *JSONPathError {
	return &JSONPathError{
		Type:       "type_mismatch",
		JSONPath:   jsonPath,
		Parameter:  parameter,
		Message:    fmt.Sprintf("cannot traverse key '%s' on %s type", pathSegment, foundType),
		ExpectedType: "object",
		ActualType: foundType,
	}
}
