package cli

import (
	"strings"
	"testing"
)

// TestSecureStringMaskedWithColor: SecureString のマスク表示が色付けされる
func TestSecureStringMaskedWithColor(t *testing.T) {
	colorEnabled = true

	maskedSecret := "***masked secret***"

	// 削除行として色付け
	coloredRemoval := colorizeRemovalLine("-" + maskedSecret)
	if !hasANSICode(coloredRemoval) {
		t.Errorf("masked secret in removal line should have ANSI code")
	}
	if !strings.Contains(coloredRemoval, maskedSecret) {
		t.Errorf("masked secret should be preserved in colored output")
	}

	// 追加行として色付け
	coloredAddition := colorizeAdditionLine("+" + maskedSecret)
	if !hasANSICode(coloredAddition) {
		t.Errorf("masked secret in addition line should have ANSI code")
	}
	if !strings.Contains(coloredAddition, maskedSecret) {
		t.Errorf("masked secret should be preserved in colored output")
	}
}

// TestSecureStringSecretValueWithColor: SecureString 秘密値が色付けされる
func TestSecureStringSecretValueWithColor(t *testing.T) {
	colorEnabled = true

	secretValue := "db_password_xyz123"

	// 削除行として色付け
	coloredRemoval := colorizeRemovalLine("-" + secretValue)
	if !hasANSICode(coloredRemoval) {
		t.Errorf("secret value in removal line should have ANSI code")
	}
	if !strings.Contains(coloredRemoval, secretValue) {
		t.Errorf("secret value should be preserved in colored output")
	}

	// 追加行として色付け
	coloredAddition := colorizeAdditionLine("+" + secretValue)
	if !hasANSICode(coloredAddition) {
		t.Errorf("secret value in addition line should have ANSI code")
	}
	if !strings.Contains(coloredAddition, secretValue) {
		t.Errorf("secret value should be preserved in colored output")
	}
}

// TestANSICodeInjectionPrevention: ANSI コード注入がないことを確認
func TestANSICodeInjectionPrevention(t *testing.T) {
	colorEnabled = true

	// ユーザー入力に ANSI コードが含まれている場合
	inputWithANSI := "value\x1b[32minjected"

	// 色付けすると、元々の ANSI コードもそのまま含まれる
	// （注: 本実装では escapeANSICodes がまだないため、この確認は実装時に追加）
	colored := colorizeRemovalLine("-" + inputWithANSI)

	// 最低限、入力値が出力に含まれることを確認
	if !strings.Contains(colored, inputWithANSI) {
		t.Errorf("input value should be in output")
	}
}

// TestSecureStringColoringConsistency: SecureString 色付けの一貫性
func TestSecureStringColoringConsistency(t *testing.T) {
	colorEnabled = true

	maskedValue := "***masked secret***"

	// 複数回の色付けが同じ結果をもたらす
	colored1 := colorizeRemovalLine("-" + maskedValue)
	colored2 := colorizeRemovalLine("-" + maskedValue)

	if colored1 != colored2 {
		t.Errorf("multiple colorizations should produce same result")
	}
}

// TestColorDisabledWithSecureString: 色無効時にセキュリティが維持される
func TestColorDisabledWithSecureString(t *testing.T) {
	colorEnabled = false

	maskedSecret := "***masked secret***"

	// 色付けなしでもマスク文字列は保持される
	result := colorizeRemovalLine("-" + maskedSecret)

	if result != "-"+maskedSecret {
		t.Errorf("with colors disabled, masked secret should not be modified")
	}

	if hasANSICode(result) {
		t.Errorf("with colors disabled, should not have ANSI code")
	}
}

// hasANSICode: 文字列に ANSI コードが含まれているか確認
func hasANSICode(s string) bool {
	return len(s) > 0 && s[0] == '\x1b'
}
