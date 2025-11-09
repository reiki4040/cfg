package cli

import (
	"testing"
)

func TestColorizeRemovalLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		enabled  bool
		expected string
	}{
		{
			name:     "color disabled",
			input:    "test",
			enabled:  false,
			expected: "test",
		},
		{
			name:     "color enabled",
			input:    "test",
			enabled:  true,
			expected: "\x1b[31mtest\x1b[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			colorEnabled = tt.enabled
			result := colorizeRemovalLine(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestColorizeAdditionLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		enabled  bool
		expected string
	}{
		{
			name:     "color disabled",
			input:    "test",
			enabled:  false,
			expected: "test",
		},
		{
			name:     "color enabled",
			input:    "test",
			enabled:  true,
			expected: "\x1b[32mtest\x1b[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			colorEnabled = tt.enabled
			result := colorizeAdditionLine(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestColorizeHeaderLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		enabled  bool
		expected string
	}{
		{
			name:     "color disabled",
			input:    "---",
			enabled:  false,
			expected: "---",
		},
		{
			name:     "color enabled",
			input:    "---",
			enabled:  true,
			expected: "\x1b[36m---\x1b[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			colorEnabled = tt.enabled
			result := colorizeHeaderLine(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestColorizeTableValue(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		diffType string
		enabled  bool
		expected string
	}{
		{
			name:     "color disabled",
			value:    "value",
			diffType: "changed",
			enabled:  false,
			expected: "value",
		},
		{
			name:     "added type",
			value:    "value",
			diffType: "added",
			enabled:  true,
			expected: "\x1b[32mvalue\x1b[0m",
		},
		{
			name:     "removed type",
			value:    "value",
			diffType: "removed",
			enabled:  true,
			expected: "\x1b[31mvalue\x1b[0m",
		},
		{
			name:     "changed type",
			value:    "value",
			diffType: "changed",
			enabled:  true,
			expected: "\x1b[33mvalue\x1b[0m",
		},
		{
			name:     "missing type",
			value:    "-",
			diffType: "missing",
			enabled:  true,
			expected: "\x1b[90m-\x1b[0m",
		},
		{
			name:     "unknown type",
			value:    "value",
			diffType: "unknown",
			enabled:  true,
			expected: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			colorEnabled = tt.enabled
			result := colorizeTableValue(tt.value, tt.diffType)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestColorizeJsonDiffLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		enabled  bool
		expected string
	}{
		{
			name:     "empty line",
			input:    "",
			enabled:  true,
			expected: "",
		},
		{
			name:     "removal line",
			input:    "-value",
			enabled:  true,
			expected: "\x1b[31m-value\x1b[0m",
		},
		{
			name:     "addition line",
			input:    "+value",
			enabled:  true,
			expected: "\x1b[32m+value\x1b[0m",
		},
		{
			name:     "header line",
			input:    "@@ -1 +1 @@",
			enabled:  true,
			expected: "\x1b[36m@@ -1 +1 @@\x1b[0m",
		},
		{
			name:     "normal line",
			input:    " value",
			enabled:  true,
			expected: " value",
		},
		{
			name:     "color disabled",
			input:    "-value",
			enabled:  false,
			expected: "-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			colorEnabled = tt.enabled
			result := colorizeJsonDiffLine(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestColorizeJsonDiffOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		enabled  bool
		expected string
	}{
		{
			name:     "color disabled",
			input:    "-val1\n+val2",
			enabled:  false,
			expected: "-val1\n+val2",
		},
		{
			name:     "multi-line diff",
			input:    "-val1\n+val2",
			enabled:  true,
			expected: "\x1b[31m-val1\x1b[0m\n\x1b[32m+val2\x1b[0m",
		},
		{
			name:     "with header",
			input:    "--- file\n+++ file\n@@ -1 +1 @@\n-old\n+new",
			enabled:  true,
			expected: "\x1b[36m--- file\x1b[0m\n\x1b[36m+++ file\x1b[0m\n\x1b[36m@@ -1 +1 @@\x1b[0m\n\x1b[31m-old\x1b[0m\n\x1b[32m+new\x1b[0m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			colorEnabled = tt.enabled
			result := colorizeJsonDiffOutput(tt.input)
			if result != tt.expected {
				t.Errorf("got %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestSetColorMode(t *testing.T) {
	tests := []struct {
		name string
		mode string
		want bool
	}{
		{
			name: "always",
			mode: "always",
			want: true,
		},
		{
			name: "never",
			mode: "never",
			want: false,
		},
		{
			name: "auto (in terminal)",
			mode: "auto",
			// ターミナルかどうかは isTerminal() に委譲するため、テストでは判定不可
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mode == "auto" {
				// auto モードはターミナル判定に依存するためスキップ
				return
			}
			setColorMode(tt.mode)
			if colorEnabled != tt.want {
				t.Errorf("got colorEnabled=%v, want %v", colorEnabled, tt.want)
			}
		})
	}
}

func TestIsValuesVarying(t *testing.T) {
	tests := []struct {
		name             string
		valuesByStage    map[string]string
		existenceByStage map[string]bool
		expected         bool
	}{
		{
			name:             "same values",
			valuesByStage:    map[string]string{"dev": "val", "stg": "val", "prod": "val"},
			existenceByStage: map[string]bool{"dev": true, "stg": true, "prod": true},
			expected:         false,
		},
		{
			name:             "different values",
			valuesByStage:    map[string]string{"dev": "val1", "stg": "val2", "prod": "val1"},
			existenceByStage: map[string]bool{"dev": true, "stg": true, "prod": true},
			expected:         true,
		},
		{
			name:             "missing stage",
			valuesByStage:    map[string]string{"dev": "val", "stg": "val"},
			existenceByStage: map[string]bool{"dev": true, "stg": true, "prod": false},
			expected:         true,
		},
		{
			name:             "empty values",
			valuesByStage:    map[string]string{},
			existenceByStage: map[string]bool{},
			expected:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValuesVarying(tt.valuesByStage, tt.existenceByStage)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}
