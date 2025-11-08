package cfg

import (
	"strings"
	"testing"
)

// TestRemoveYamlComments tests the removal of YAML comments
func TestRemoveYamlComments(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple_comment",
			input:    "key: value # this is a comment",
			expected: "key: value ",
		},
		{
			name:     "full_line_comment",
			input:    "# This is a full line comment\nkey: value",
			expected: "\nkey: value",
		},
		{
			name: "reference_in_comment",
			input: `# JSONPath references use the format: ${ps:/path/to/parameter:json.path.to.key}
key: ${ps:/actual/parameter}`,
			expected: `
key: ${ps:/actual/parameter}`,
		},
		{
			name:     "no_comment",
			input:    "key: value",
			expected: "key: value",
		},
		{
			name: "multiple_references_with_comments",
			input: `# Example: ${ps:/example:json.path}
database:
  host: ${ps:/app/config:database.host}  # Primary host
  port: ${ps:/app/config:database.port}  # Port number`,
			expected: "\ndatabase:\n  host: ${ps:/app/config:database.host}  \n  port: ${ps:/app/config:database.port}  ",
		},
		{
			name:     "hash_in_quoted_string",
			input:    `password: "mypass#word"  # this is comment`,
			expected: `password: "mypass#word"  `,
		},
		{
			name:     "hash_in_single_quoted_string",
			input:    `password: 'mypass#word'  # this is comment`,
			expected: `password: 'mypass#word'  `,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := removeYamlComments(tc.input)
			if result != tc.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tc.expected, result)
			}
		})
	}
}

// TestFindCommentStart tests the findCommentStart function
func TestFindCommentStart(t *testing.T) {
	testCases := []struct {
		name     string
		line     string
		expected int
	}{
		{
			name:     "comment_at_end",
			line:     "key: value # comment",
			expected: 11, // Position of '#' (0-indexed)
		},
		{
			name:     "no_comment",
			line:     "key: value",
			expected: -1,
		},
		{
			name:     "full_line_comment",
			line:     "# This is a comment",
			expected: 0,
		},
		{
			name:     "hash_in_double_quotes",
			line:     `password: "pass#word" # comment`,
			expected: 22, // Position of comment '#' (not the one inside quotes)
		},
		{
			name:     "hash_in_single_quotes",
			line:     `password: 'pass#word' # comment`,
			expected: 22, // Position of comment '#' (not the one inside quotes)
		},
		{
			name:     "escaped_hash_in_quotes",
			line:     `text: "some\#text" # comment`,
			expected: 19, // Position of comment '#'
		},
		{
			name:     "only_hash",
			line:     "#",
			expected: 0,
		},
		{
			name:     "reference_with_hash",
			line:     "${ps:/path:json.hash} # comment",
			expected: 22, // Position of comment '#' (after the space)
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findCommentStart(tc.line)
			if result != tc.expected {
				t.Errorf("expected %d, got %d for line: %q", tc.expected, result, tc.line)
			}
		})
	}
}

// TestResolverIgnoresCommentedReferences tests that ExtractReferences ignores commented references
func TestResolverIgnoresCommentedReferences(t *testing.T) {
	stageResolver := NewStageResolver("dev")
	resolver := NewResolver(nil, stageResolver)

	yamlContent := `# Example: ${ps:/example/parameter}
# Another example: ${env:COMMENTED_VAR}
app:
  name: myapp
  database:
    # Config from: ${ps:/app/config:database.host}
    host: ${ps:/app/config:host}  # Real reference
    port: 5432  # ${ps:/app/config:port} - this is commented
  api:
    endpoint: ${env:API_ENDPOINT}  # Real env reference
    # timeout: ${ps:/app/timeout}  - commented
`

	refs := resolver.ExtractReferences(yamlContent)

	// Should find exactly 2 references: one ps and one env
	psRefs := 0
	envRefs := 0

	for _, ref := range refs {
		switch ref.Type {
		case "ps":
			psRefs++
			// Should find /app/config (resolved stage)
			if !strings.Contains(ref.Key, "/config") {
				t.Errorf("unexpected ps reference key: %s", ref.Key)
			}
		case "env":
			envRefs++
			// Should find API_ENDPOINT
			if ref.Key != "API_ENDPOINT" {
				t.Errorf("unexpected env reference key: %s", ref.Key)
			}
		}
	}

	if psRefs != 1 {
		t.Errorf("expected 1 ps reference, got %d", psRefs)
	}
	if envRefs != 1 {
		t.Errorf("expected 1 env reference, got %d", envRefs)
	}
}

// TestConfigJsonPathYamlWithComments tests the actual config-jsonpath.yaml use case
func TestConfigJsonPathYamlWithComments(t *testing.T) {
	stageResolver := NewStageResolver("dev")
	resolver := NewResolver(nil, stageResolver)

	// Simulate the config-jsonpath.yaml file with comments
	yamlContent := `# JSONPath Example Configuration
#
# This example demonstrates how to use JSONPath to extract specific values
# from JSON stored in Parameter Store parameters.
#
# JSONPath references use the format: ${ps:/path/to/parameter:json.path.to.key}
#
# Benefits:
# - Reduce API calls by storing multiple values in one JSON parameter
# - Extract specific keys from nested JSON structures
# - Maintain backward compatibility with traditional references

# Example Parameter Store Setup:
#
# /app/{stage}/database-config contains:
# {
#   "primary": {
#     "host": "primary-db.example.com",
#     "port": 5432
#   }
# }

app:
  name: cfg-example-app
  version: 1.0.0

database:
  # Using JSONPath to extract specific keys from the same parameter
  # Multiple references to the same parameter are deduplicated
  primary:
    host: ${ps:/app/{stage}/database-config:primary.host}
    port: ${ps:/app/{stage}/database-config:primary.port}
    username: ${ps:/app/{stage}/database-config:primary.username}
`

	refs := resolver.ExtractReferences(yamlContent)

	// Should extract 3 ps references with JSONPath
	psRefs := 0
	jsonPathRefs := 0

	for _, ref := range refs {
		if ref.Type == "ps" {
			psRefs++
			if ref.JSONPath != "" {
				jsonPathRefs++
			}
		}
	}

	if psRefs != 3 {
		t.Errorf("expected 3 ps references, got %d", psRefs)
	}
	if jsonPathRefs != 3 {
		t.Errorf("expected 3 JSONPath references, got %d", jsonPathRefs)
	}

	// All references should point to the same parameter (deduplication)
	uniqueParams := make(map[string]bool)
	for _, ref := range refs {
		if ref.Type == "ps" {
			uniqueParams[ref.Key] = true
		}
	}

	if len(uniqueParams) != 1 {
		t.Errorf("expected 1 unique parameter, got %d", len(uniqueParams))
	}
}

// TestMultipleCommentsPerLine tests lines with multiple potential comment starts
func TestMultipleCommentsPerLine(t *testing.T) {
	testCases := []struct {
		name     string
		line     string
		expected int
	}{
		{
			name:     "two_hashes_first_in_quote",
			line:     `text: "value#1" # real comment`,
			expected: 16, // Position of comment '#'
		},
		{
			name:     "url_with_port_then_comment",
			line:     `url: "http://host:8080" # comment`,
			expected: 24, // Position of comment '#'
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := findCommentStart(tc.line)
			if result != tc.expected {
				t.Errorf("expected %d, got %d for line: %q", tc.expected, result, tc.line)
			}
		})
	}
}
