package cfg

import "fmt"

// JsonParseError represents an error during JSON parsing.
// It includes the original input and any underlying cause for debugging.
type JsonParseError struct {
	// Message describes the parse error
	Message string
	// Input contains the JSON string that failed to parse
	Input string
	// Cause is the underlying error from json.Unmarshal, if any
	Cause error
}

// Error implements the error interface and returns a formatted error message
func (e *JsonParseError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("JSON parse error: %s (cause: %v)", e.Message, e.Cause)
	}
	return fmt.Sprintf("JSON parse error: %s", e.Message)
}

// JsonPathError represents an error in JSONPath format or parsing.
// It includes the problematic path and contextual details.
type JsonPathError struct {
	// Path is the key path that caused the error
	Path string
	// Message describes the path error
	Message string
	// Details provides additional context or guidance
	Details string
}

// Error implements the error interface and returns a formatted error message
func (e *JsonPathError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("JSONPath error in %q: %s (%s)", e.Path, e.Message, e.Details)
	}
	return fmt.Sprintf("JSONPath error in %q: %s", e.Path, e.Message)
}

// JsonUpdateError represents an error during JSON attribute update.
// It provides information about which path failed and why.
type JsonUpdateError struct {
	// Path is the key path where the update failed
	Path string
	// Message describes the update error
	Message string
	// Reason categorizes the error type:
	// - "type_mismatch": attempted to traverse through non-object value
	// - "missing_key": key not found and cannot create
	// - "invalid_path": malformed key path
	// - "marshal_error": failed to serialize updated JSON
	Reason string
}

// Error implements the error interface and returns a formatted error message
func (e *JsonUpdateError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("JSON update error at path %q: %s (reason: %s)", e.Path, e.Message, e.Reason)
	}
	return fmt.Sprintf("JSON update error at path %q: %s", e.Path, e.Message)
}
