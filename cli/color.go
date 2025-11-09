package cli

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// ANSI カラーコード定数
const (
	ColorRed    = "\x1b[31m"
	ColorGreen  = "\x1b[32m"
	ColorYellow = "\x1b[33m"
	ColorCyan   = "\x1b[36m"
	ColorGray   = "\x1b[90m"
	ColorReset  = "\x1b[0m"
)

// 色出力制御用の変数
var (
	// colorEnabled: 実際に色を使用するかどうか
	colorEnabled = false
)

// isTerminal: 標準出力がターミナルであるかを判定
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// shouldUseColor: 色を使用すべきかを判定
func shouldUseColor() bool {
	return colorEnabled
}

// setColorMode: 色出力モードを設定（diff.go からの呼び出し）
func setColorMode(mode string) {
	if mode == "always" {
		colorEnabled = true
	} else if mode == "never" {
		colorEnabled = false
	} else {
		// "auto": ターミナル判定
		colorEnabled = isTerminal()
	}
}

// colorizeRemovalLine: 削除行を赤色で色付け
func colorizeRemovalLine(content string) string {
	if !shouldUseColor() {
		return content
	}
	return fmt.Sprintf("%s%s%s", ColorRed, content, ColorReset)
}

// colorizeAdditionLine: 追加行を緑色で色付け
func colorizeAdditionLine(content string) string {
	if !shouldUseColor() {
		return content
	}
	return fmt.Sprintf("%s%s%s", ColorGreen, content, ColorReset)
}

// colorizeChangeLine: 変更行を黄色で色付け
func colorizeChangeLine(content string) string {
	if !shouldUseColor() {
		return content
	}
	return fmt.Sprintf("%s%s%s", ColorYellow, content, ColorReset)
}

// colorizeHeaderLine: ヘッダー行（---、+++、@@）をシアンで色付け
func colorizeHeaderLine(content string) string {
	if !shouldUseColor() {
		return content
	}
	return fmt.Sprintf("%s%s%s", ColorCyan, content, ColorReset)
}

// colorizeUnchangedLine: 変更がない行をデフォルト色で表示
func colorizeUnchangedLine(content string) string {
	return content
}

// colorizeTableValue: テーブル値を対応する色で色付け
func colorizeTableValue(value string, diffType string) string {
	if !shouldUseColor() {
		return value
	}
	switch diffType {
	case "added":
		return fmt.Sprintf("%s%s%s", ColorGreen, value, ColorReset)
	case "removed":
		return fmt.Sprintf("%s%s%s", ColorRed, value, ColorReset)
	case "changed":
		return fmt.Sprintf("%s%s%s", ColorYellow, value, ColorReset)
	case "missing":
		return fmt.Sprintf("%s%s%s", ColorGray, value, ColorReset)
	default:
		return value
	}
}

// colorizeJsonDiffLine: JSON 差分の行を色付け
// 行の先頭パターンで判定して色を決定
func colorizeJsonDiffLine(line string) string {
	if !shouldUseColor() {
		return line
	}

	if len(line) == 0 {
		return line
	}

	// ヘッダー行（---、+++、@@）を優先的に判定
	if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "@@") {
		return colorizeHeaderLine(line)
	}

	// 行の先頭文字で判定
	if line[0] == '-' {
		return colorizeRemovalLine(line)
	} else if line[0] == '+' {
		return colorizeAdditionLine(line)
	}

	return line
}

// colorizeJsonDiffOutput: JSON 差分出力全体を色付け
// 複数行の出力を改行で分割して、各行を色付け
func colorizeJsonDiffOutput(output string) string {
	if !shouldUseColor() {
		return output
	}

	lines := strings.Split(output, "\n")
	for i, line := range lines {
		lines[i] = colorizeJsonDiffLine(line)
	}
	return strings.Join(lines, "\n")
}
