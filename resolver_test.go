package cfg

import (
	"regexp"
	"testing"
)

func TestPsReferenceRegex(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
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