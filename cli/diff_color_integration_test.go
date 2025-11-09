package cli

import (
	"testing"
)

// TestDisplayDiffColorOutput: 2ステージ diff での色出力確認
func TestDisplayDiffColorOutput(t *testing.T) {
	// このテストは実装時点では確認用です
	// 実際のParameter Store テストは別途実施

	tests := []struct {
		name     string
		mode     string
		expected bool // colorEnabled が期待値と一致するか
	}{
		{
			name:     "color always enabled",
			mode:     "always",
			expected: true,
		},
		{
			name:     "color never disabled",
			mode:     "never",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setColorMode(tt.mode)
			if shouldUseColor() != tt.expected {
				t.Errorf("setColorMode(%q): got shouldUseColor()=%v, want %v",
					tt.mode, shouldUseColor(), tt.expected)
			}
		})
	}
}

// TestColorOutputWithDiffTypes: 異なる diff タイプでの色出力
func TestColorOutputWithDiffTypes(t *testing.T) {
	colorEnabled = true

	tests := []struct {
		name             string
		content          string
		colorizer        func(string) string
		shouldContainESC bool
	}{
		{
			name:             "removal line contains ANSI code",
			content:          "removed parameter",
			colorizer:        colorizeRemovalLine,
			shouldContainESC: true,
		},
		{
			name:             "addition line contains ANSI code",
			content:          "added parameter",
			colorizer:        colorizeAdditionLine,
			shouldContainESC: true,
		},
		{
			name:             "header line contains ANSI code",
			content:          "---",
			colorizer:        colorizeHeaderLine,
			shouldContainESC: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.colorizer(tt.content)
			hasESC := len(result) > len(tt.content) // ANSI コード追加で文字数が増える
			if hasESC != tt.shouldContainESC {
				t.Errorf("got ANSI code present=%v, want %v", hasESC, tt.shouldContainESC)
			}
		})
	}
}

// TestColorModeSwitch: 色モードの切り替え
func TestColorModeSwitch(t *testing.T) {
	tests := []struct {
		name    string
		modes   []string
		expects []bool
	}{
		{
			name:    "toggle color modes",
			modes:   []string{"always", "never", "always"},
			expects: []bool{true, false, true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, mode := range tt.modes {
				setColorMode(mode)
				if shouldUseColor() != tt.expects[i] {
					t.Errorf("step %d: setColorMode(%q): got %v, want %v",
						i, mode, shouldUseColor(), tt.expects[i])
				}
			}
		})
	}
}

// TestColoredRemovalAndAddition: 削除と追加の色付け出力
func TestColoredRemovalAndAddition(t *testing.T) {
	colorEnabled = true

	removal := colorizeRemovalLine("-oldvalue")
	addition := colorizeAdditionLine("+newvalue")

	// Red ANSI code (31) should be in removal
	if !containsANSICode(removal, "31") {
		t.Errorf("removal line should contain red ANSI code (31)")
	}

	// Green ANSI code (32) should be in addition
	if !containsANSICode(addition, "32") {
		t.Errorf("addition line should contain green ANSI code (32)")
	}
}

// TestColoredHeader: ヘッダー行の色付け
func TestColoredHeader(t *testing.T) {
	colorEnabled = true

	header := colorizeHeaderLine("--- file1 (dev)")

	// Cyan ANSI code (36) should be in header
	if !containsANSICode(header, "36") {
		t.Errorf("header line should contain cyan ANSI code (36)")
	}
}

// containsANSICode: ANSI コードが含まれているか確認
func containsANSICode(s string, code string) bool {
	return len(s) > 0 && s[0] == '\x1b' && containsSubstring(s, code)
}

// containsSubstring: 文字列が部分文字列を含むか確認
func containsSubstring(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
