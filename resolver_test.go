package cfg

import (
	"regexp"
	"strings"
	"testing"
)

func TestPsReferenceRegex(t *testing.T) {
	testCases := []struct {
		input       string
		expected    string
		shouldMatch bool
	}{
		{"${ps:/app/simple}", "/app/simple", true},
		{"${ps:/cfgtool/{stage}/app/api_url}", "/cfgtool/{stage}/app/api_url", true},
		{"${ps:/app/{stage}/db/host}", "/app/{stage}/db/host", true},
		{"${ps:/complex/{stage}/path/{env}/test}", "/complex/{stage}/path/{env}/test", true},
		{"${ps:relative/path}", "relative/path", true},
		{"${ps:}", "", true},
		{"${env:TEST}", "", false}, // Should not match ps regex
		{"${invalid", "", false},   // Should not match
	}

	for _, tc := range testCases {
		matches := psReferenceRegex.FindStringSubmatch(tc.input)
		if tc.shouldMatch {
			if len(matches) < 2 {
				t.Errorf("Expected %s to match, but it didn't", tc.input)
				continue
			}
			if matches[1] != tc.expected {
				t.Errorf("For %s, expected %s but got %s", tc.input, tc.expected, matches[1])
			}
		} else {
			if len(matches) >= 2 {
				t.Errorf("Expected %s not to match ps regex, but it did", tc.input)
			}
		}
	}
}

// BenchmarkPsReferenceRegex benchmarks the parameter store reference regex
func BenchmarkPsReferenceRegex(b *testing.B) {
	testCases := []string{
		"${ps:/app/simple}",
		"${ps:/cfgtool/{stage}/app/api_url}",
		"${ps:/app/{stage}/db/host}",
		"${ps:/complex/{stage}/path/{env}/test}",
		"${ps:relative/path}",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, testCase := range testCases {
			psReferenceRegex.FindStringSubmatch(testCase)
		}
	}
}

// TestPsReferenceRegexInteractive provides interactive testing similar to the standalone version
func TestPsReferenceRegexInteractive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping interactive test in short mode")
	}

	psReferenceRegex := regexp.MustCompile(`\$\{ps:((?:[^{}]|\{[^}]*\})*)\}`)

	testCases := []string{
		"${ps:/app/simple}",
		"${ps:/cfgtool/{stage}/app/api_url}",
		"${ps:/app/{stage}/db/host}",
		"${ps:/complex/{stage}/path/{env}/test}",
		"${ps:relative/path}",
	}

	for _, testCase := range testCases {
		matches := psReferenceRegex.FindStringSubmatch(testCase)
		if len(matches) >= 2 {
			t.Logf("✅ %s -> %s", testCase, matches[1])
		} else {
			t.Errorf("❌ %s -> NO MATCH", testCase)
		}
	}
}

// Test resolver's JSONPath reference extraction
func TestResolverExtractJsonPathReferences(t *testing.T) {
	stageResolver := NewStageResolver("dev")
	resolver := NewResolver(nil, stageResolver)

	yamlContent := `
database:
  config: ${ps:/app-config:db.host}
  password: ${ps:/app-config:db.password}
  port: ${ps:/app-port}
api:
  key: ${ps:/api-config:key.secret}
  endpoint: ${env:API_ENDPOINT}
`

	refs := resolver.ExtractReferences(yamlContent)

	// Should extract 4 ps references total
	psRefs := make([]Reference, 0)
	for _, ref := range refs {
		if ref.Type == "ps" {
			psRefs = append(psRefs, ref)
		}
	}

	if len(psRefs) != 4 {
		t.Errorf("Expected 4 ps references, got %d", len(psRefs))
	}

	// Check that JSONPath references are properly captured
	hasJsonPathRef := false
	for _, ref := range psRefs {
		if ref.JSONPath != "" {
			hasJsonPathRef = true
			break
		}
	}

	if !hasJsonPathRef {
		t.Errorf("Expected to find JSONPath references in extracted references")
	}
}

// TestResolverExtractMultipleReferences tests extraction of mixed reference types
func TestResolverExtractMultipleReferences(t *testing.T) {
	stageResolver := NewStageResolver("prod")
	resolver := NewResolver(nil, stageResolver)

	yamlContent := `
config:
  db:
    host: ${ps:/app-config:database.host}
    user: ${ps:/app-config:database.user}
    port: ${ps:/app-port}
  api:
    endpoint: ${env:API_ENDPOINT}
    timeout: 30
  stage: ${stage-prefix:default}
`

	refs := resolver.ExtractReferences(yamlContent)

	// Count by type
	psCount := 0
	envCount := 0
	stagePrefixCount := 0

	for _, ref := range refs {
		switch ref.Type {
		case "ps":
			psCount++
		case "env":
			envCount++
		case "stage-prefix":
			stagePrefixCount++
		}
	}

	if psCount != 3 {
		t.Errorf("Expected 3 ps references, got %d", psCount)
	}
	if envCount != 1 {
		t.Errorf("Expected 1 env reference, got %d", envCount)
	}
	if stagePrefixCount != 1 {
		t.Errorf("Expected 1 stage-prefix reference, got %d", stagePrefixCount)
	}

	// Verify JSONPath references
	jsonPathCount := 0
	for _, ref := range refs {
		if ref.Type == "ps" && ref.JSONPath != "" {
			jsonPathCount++
		}
	}

	if jsonPathCount != 2 {
		t.Errorf("Expected 2 JSONPath references, got %d", jsonPathCount)
	}
}

// TestResolverComplexJsonPath tests complex nested JSONPath extraction
func TestResolverComplexJsonPath(t *testing.T) {
	stageResolver := NewStageResolver("dev")
	resolver := NewResolver(nil, stageResolver)

	yamlContent := `
app:
  database:
    primary: ${ps:/app-config:connections.primary.host}
    replica: ${ps:/app-config:connections.replica.host}
    credentials:
      user: ${ps:/app-secrets:db.user}
      password: ${ps:/app-secrets:db.password}
`

	refs := resolver.ExtractReferences(yamlContent)

	psRefs := []Reference{}
	for _, ref := range refs {
		if ref.Type == "ps" {
			psRefs = append(psRefs, ref)
		}
	}

	if len(psRefs) != 4 {
		t.Errorf("Expected 4 ps references, got %d", len(psRefs))
	}

	// Verify specific JSONPath extractions
	expectedPaths := map[string]bool{
		"connections.primary.host": false,
		"connections.replica.host": false,
		"db.user":                  false,
		"db.password":              false,
	}

	for _, ref := range psRefs {
		if _, exists := expectedPaths[ref.JSONPath]; exists {
			expectedPaths[ref.JSONPath] = true
		}
	}

	for path, found := range expectedPaths {
		if !found {
			t.Errorf("Expected to find JSONPath: %s", path)
		}
	}
}

// TestResolverStagePlaceholder tests stage placeholder resolution with JSONPath
func TestResolverStageJsonPathCombination(t *testing.T) {
	stageResolver := NewStageResolver("staging")
	resolver := NewResolver(nil, stageResolver)

	yamlContent := `
config:
  db: ${ps:/app/{stage}/config:database.primary}
  cache: ${ps:/data/{stage}/cache:server}
`

	refs := resolver.ExtractReferences(yamlContent)

	if len(refs) != 2 {
		t.Errorf("Expected 2 references, got %d", len(refs))
	}

	// Check that stage placeholder was resolved
	for _, ref := range refs {
		if ref.Type == "ps" {
			// Should contain /staging/ not /{stage}/
			if strings.Contains(ref.Key, "{stage}") {
				t.Errorf("Stage placeholder not resolved in key: %s", ref.Key)
			}
			if !strings.Contains(ref.Key, "staging") {
				t.Errorf("Expected 'staging' in resolved key: %s", ref.Key)
			}
		}
	}
}

// TestResolverDeduplication tests that multiple JSONPath references to the same parameter are detected
func TestResolverDeduplication(t *testing.T) {
	stageResolver := NewStageResolver("dev")
	resolver := NewResolver(nil, stageResolver)

	yamlContent := `
app:
  db:
    host: ${ps:/app-config:database.host}
    port: ${ps:/app-config:database.port}
    user: ${ps:/app-config:database.user}
`

	refs := resolver.ExtractReferences(yamlContent)

	// All three references should point to the same parameter
	paramKeys := make(map[string]int)
	for _, ref := range refs {
		if ref.Type == "ps" {
			paramKeys[ref.Key]++
		}
	}

	// Should have all references to the same parameter
	if len(paramKeys) != 1 {
		t.Errorf("Expected all references to map to 1 parameter, got %d unique parameters", len(paramKeys))
	}

	// Verify the parameter name is correct
	if _, hasKey := paramKeys["/app-config"]; !hasKey {
		t.Errorf("Expected parameter '/app-config' in deduplication map")
	}
}

// TestResolverJsonPathValidation tests that invalid JSONPath formats are handled
func TestResolverJsonPathValidation(t *testing.T) {
	stageResolver := NewStageResolver("dev")
	resolver := NewResolver(nil, stageResolver)

	// These should not cause extraction errors, but JSONPath should be empty or invalid
	yamlContent := `
app:
  config: ${ps:/app-config}
  double_colon: ${ps:/app::invalid}
`

	refs := resolver.ExtractReferences(yamlContent)

	if len(refs) != 2 {
		t.Errorf("Expected 2 references, got %d", len(refs))
	}

	// First one should have no JSONPath
	if refs[0].JSONPath != "" {
		t.Errorf("Expected no JSONPath for traditional reference, got: %s", refs[0].JSONPath)
	}
}
