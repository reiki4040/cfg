package cli

import (
	"os"
	"testing"
)

// TestIsTerminal: ターミナル判定機能
func TestIsTerminal(t *testing.T) {
	// 標準出力がターミナルか判定
	// (このテスト実行環境では通常 false になる)
	result := isTerminal()

	// テスト環境ではパイプ出力なので、通常 false
	// 実際の使用時にはターミナル出力で true になる
	if result && !isTTY() {
		t.Logf("isTerminal() returned true (likely in TTY environment)")
	} else if !result {
		t.Logf("isTerminal() returned false (likely piped/redirected output)")
	}
}

// TestColorModeAuto: auto モード時のターミナル判定
func TestColorModeAuto(t *testing.T) {
	// auto モード: ターミナル判定に依存
	setColorMode("auto")

	// テスト環境ではターミナルではないので、色無効
	if isTTY() {
		if !shouldUseColor() {
			t.Logf("in TTY but color disabled (unexpected)")
		}
	} else {
		if shouldUseColor() {
			t.Errorf("not in TTY but color enabled (expected disabled)")
		}
	}
}

// TestColorModeAlways: always モード時の強制有効化
func TestColorModeAlways(t *testing.T) {
	setColorMode("always")

	if !shouldUseColor() {
		t.Errorf("color should be enabled with 'always' mode")
	}
}

// TestColorModeNever: never モード時の強制無効化
func TestColorModeNever(t *testing.T) {
	setColorMode("never")

	if shouldUseColor() {
		t.Errorf("color should be disabled with 'never' mode")
	}
}

// TestColorModePipeDetection: パイプ出力時の自動判定
func TestColorModePipeDetection(t *testing.T) {
	setColorMode("auto")

	// パイプ出力（非TTY）ではターミナル判定が false
	if isTTY() {
		t.Logf("running in TTY environment, auto mode enables color")
	} else {
		t.Logf("running with piped/redirected output, auto mode disables color")
		if shouldUseColor() {
			t.Errorf("color should be disabled when output is piped")
		}
	}
}

// TestColorModeOverride: フラグでの色モード制御
func TestColorModeOverride(t *testing.T) {
	tests := []struct {
		mode    string
		expect  bool
		name    string
	}{
		{"always", true, "always mode overrides pipe detection"},
		{"never", false, "never mode overrides TTY detection"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setColorMode(tt.mode)
			if shouldUseColor() != tt.expect {
				t.Errorf("setColorMode(%q): got %v, want %v",
					tt.mode, shouldUseColor(), tt.expect)
			}
		})
	}
}

// TestANSIColorCodesAreStandard: ANSI カラーコードが標準
func TestANSIColorCodesAreStandard(t *testing.T) {
	// 標準的な ANSI コード定義を確認
	tests := []struct {
		code     string
		expected string
		name     string
	}{
		{ColorRed, "\x1b[31m", "Red (31)"},
		{ColorGreen, "\x1b[32m", "Green (32)"},
		{ColorYellow, "\x1b[33m", "Yellow (33)"},
		{ColorCyan, "\x1b[36m", "Cyan (36)"},
		{ColorGray, "\x1b[90m", "Gray (90)"},
		{ColorReset, "\x1b[0m", "Reset (0)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.expected {
				t.Errorf("got %q, want %q", tt.code, tt.expected)
			}
		})
	}
}

// TestColorOutputFallback: 古いターミナル環境での色出力無効化
func TestColorOutputFallback(t *testing.T) {
	// never モードで色を完全に無効化
	setColorMode("never")

	removal := colorizeRemovalLine("-value")
	if removal != "-value" {
		t.Errorf("with never mode, got %q, want %q", removal, "-value")
	}
}

// isTTY: TTY 環境かどうかを判定（テスト用ヘルパー）
func isTTY() bool {
	// Stdout がターミナルであるか確認
	stat, _ := os.Stdout.Stat()
	// os.ModeCharDevice check
	return (stat.Mode() & 0x00002000) != 0
}
