package cfg

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// psJsonPathReferenceRegex matches Parameter Store references with optional JSONPath
// Pattern: ${ps:path[:jsonpath]}
// Handles: ${ps:/path}, ${ps:/path:}, ${ps:/path:key}, ${ps:/path:key.nested}
// The challenge is distinguishing between path (can contain :) and jsonpath (after final :)
// Strategy: match everything up to the last : if it exists and followed by non-} chars
var psJsonPathReferenceRegex = regexp.MustCompile(`\$\{ps:([^}]*?):([^}]*)\}|\$\{ps:([^}]*)\}`)

// For simpler matching, we'll use a two-step approach in the Match function

// JsonPathMatcher handles extraction of path and jsonpath from references
type JsonPathMatcher struct {
	regex *regexp.Regexp
}

// NewJsonPathMatcher creates a new JsonPathMatcher instance
func NewJsonPathMatcher() *JsonPathMatcher {
	return &JsonPathMatcher{
		regex: psJsonPathReferenceRegex,
	}
}

// Match extracts parameter path and JSONPath from a reference string
// Returns (path, jsonpath, matched)
func (m *JsonPathMatcher) Match(input string) (string, string, bool) {
	// Use a simpler approach: find the ps reference and manually parse
	if !strings.HasPrefix(input, "${ps:") || !strings.HasSuffix(input, "}") {
		return "", "", false
	}

	// Extract content between ${ps: and }
	content := input[5 : len(input)-1] // Remove "${ps:" and "}"

	// Find the last colon that separates path from jsonpath
	// But we need to be careful: {stage} in path can contain colons
	// So we look for : that is NOT inside {...}

	path, jsonPath := splitPathAndJsonPath(content)
	return path, jsonPath, true
}

// splitPathAndJsonPath splits parameter path and jsonpath
// Handles: /path, /path:key, /path:key.nested, /path{stage}:key
func splitPathAndJsonPath(content string) (string, string) {
	// Find the last unescaped colon that is not part of {...}
	braceDepth := 0
	lastColonIdx := -1

	for i := 0; i < len(content); i++ {
		ch := content[i]
		if ch == '{' {
			braceDepth++
		} else if ch == '}' {
			braceDepth--
		} else if ch == ':' && braceDepth == 0 {
			lastColonIdx = i
		}
	}

	if lastColonIdx == -1 {
		// No colon outside of braces, entire content is path
		return content, ""
	}

	// Split at the last colon
	path := content[:lastColonIdx]
	jsonPath := content[lastColonIdx+1:]
	return path, jsonPath
}

// ValidateJsonPath checks if a JSONPath string is valid
// Valid format: alphanumeric and underscore characters separated by dots
// Examples: "password", "db.password", "db.primary.host"
// Empty string is NOT valid for validation purposes (should be caught at extraction level)
func ValidateJsonPath(jsonPath string) *JSONPathError {
	if jsonPath == "" {
		// Empty JSONPath should be rejected in validation context
		// (At extraction level, this means no JSONPath, which is fine)
		return InvalidJSONPathFormatError(jsonPath, "", "empty jsonpath not allowed")
	}

	// Check for invalid patterns
	if strings.Contains(jsonPath, "..") {
		return InvalidJSONPathFormatError(jsonPath, "", "double dots not allowed")
	}

	if strings.HasPrefix(jsonPath, ".") {
		return InvalidJSONPathFormatError(jsonPath, "", "leading dot not allowed")
	}

	if strings.HasSuffix(jsonPath, ".") {
		return InvalidJSONPathFormatError(jsonPath, "", "trailing dot not allowed")
	}

	// Check for array index notation
	if strings.ContainsAny(jsonPath, "[]") {
		return InvalidJSONPathFormatError(jsonPath, "", "array index notation not supported")
	}

	// Validate each segment
	segments := strings.Split(jsonPath, ".")
	for _, segment := range segments {
		if !isValidKeySegment(segment) {
			return InvalidJSONPathFormatError(jsonPath, "", fmt.Sprintf("invalid key segment: %q", segment))
		}
	}

	return nil
}

// isValidKeySegment checks if a key segment is valid
// Valid: alphanumeric and underscore, must start with letter or underscore
func isValidKeySegment(segment string) bool {
	if len(segment) == 0 {
		return false
	}

	// First character must be letter or underscore
	first := segment[0]
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}

	// Rest must be alphanumeric or underscore
	for _, ch := range segment {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
			return false
		}
	}

	return true
}

// JsonPathExtractor handles extraction of values from JSON using JSONPath
type JsonPathExtractor struct {
	matcher *JsonPathMatcher
}

// NewJsonPathExtractor creates a new JsonPathExtractor
func NewJsonPathExtractor() *JsonPathExtractor {
	return &JsonPathExtractor{
		matcher: NewJsonPathMatcher(),
	}
}

// ExtractValue extracts a value from JSON using JSONPath notation
// Returns the value as a string and any error encountered
// If jsonPath is empty, returns the whole JSON string as-is
func (e *JsonPathExtractor) ExtractValue(jsonStr string, jsonPath string, parameterName string) (string, *JSONPathError) {
	if jsonPath == "" {
		// No JSONPath specified, return the whole JSON string
		return jsonStr, nil
	}

	// Validate JSONPath format (non-empty JSONPath must be valid)
	if err := ValidateJsonPath(jsonPath); err != nil {
		return "", err
	}

	// Parse JSON
	var jsonObj map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &jsonObj); err != nil {
		return "", JSONParseError(parameterName, err)
	}

	// Traverse the path and extract value
	value, err := e.traversePath(jsonObj, jsonPath, parameterName)
	if err != nil {
		return "", err
	}

	// Convert value to string
	strValue := valueToString(value)

	return strValue, nil
}

// traversePath recursively traverses the JSON object following the JSONPath
func (e *JsonPathExtractor) traversePath(obj interface{}, jsonPath string, parameterName string) (interface{}, *JSONPathError) {
	if jsonPath == "" {
		return obj, nil
	}

	// Split the path into segments
	segments := strings.Split(jsonPath, ".")
	return e.traverseSegments(obj, segments, 0, parameterName)
}

// traverseSegments recursively traverses path segments
func (e *JsonPathExtractor) traverseSegments(current interface{}, segments []string, index int, parameterName string) (interface{}, *JSONPathError) {
	if index >= len(segments) {
		return current, nil
	}

	segment := segments[index]
	objMap, ok := current.(map[string]interface{})
	if !ok {
		// Current value is not an object, cannot traverse further
		actualType := getTypeName(current)
		return nil, TypeMismatchError(strings.Join(segments[:index+1], "."), parameterName, segment, actualType)
	}

	value, exists := objMap[segment]
	if !exists {
		// Key not found, return all available keys
		keys := make([]string, 0, len(objMap))
		for k := range objMap {
			keys = append(keys, k)
		}
		return nil, KeyNotFoundError(strings.Join(segments, "."), parameterName, keys)
	}

	// Continue traversal or return value
	if index == len(segments)-1 {
		return value, nil
	}

	return e.traverseSegments(value, segments, index+1, parameterName)
}

// valueToString converts a JSON value to string representation
func valueToString(value interface{}) string {
	if value == nil {
		return "" // null becomes empty string
	}

	switch v := value.(type) {
	case string:
		return v
	case float64:
		// Handle both int and float
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%f", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		// For complex types (objects, arrays), marshal back to JSON
		jsonBytes, _ := json.Marshal(value)
		return string(jsonBytes)
	}
}

// getTypeName returns a human-readable type name for error messages
func getTypeName(value interface{}) string {
	switch value.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	default:
		return "unknown"
	}
}
