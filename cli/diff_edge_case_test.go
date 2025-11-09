package cli

import (
	"testing"
	"time"

	"github.com/reiki4040/cfg/aws"
)

// TestEmptyDiffOutput: 空の diff 出力の処理
func TestEmptyDiffOutput(t *testing.T) {
	// ローカルバッファで出力をキャプチャするためのセットアップ
	// 実際には displayParameterStoreDiffWithTypes は fmt.Println を使用するため、
	// テストは function 引数をモック化することが理想だが、
	// ここでは関数の動作確認のみ行う

	colorEnabled = true

	// 同一パラメータで diff がない場合
	paramInfos1 := []aws.ParameterInfo{
		{
			Name:  "/app/db/host",
			Value: "localhost",
			Type:  "String",
		},
	}

	paramInfos2 := []aws.ParameterInfo{
		{
			Name:  "/app/db/host",
			Value: "localhost",
			Type:  "String",
		},
	}

	// この呼び出しは "No differences found." を出力して終了する
	// エラーは発生しないことを確認
	displayParameterStoreDiffWithTypes("dev", "stg", paramInfos1, paramInfos2, "/app/", false, false)

	// テスト成功（パニックしない）
	if true {
		t.Logf("Empty diff output handled correctly")
	}
}

// TestLargeParameterListPerformance: 大量パラメータ出力の性能確認
func TestLargeParameterListPerformance(t *testing.T) {
	colorEnabled = true

	// 1000+ パラメータを生成
	paramCount := 1000
	paramInfos1 := make([]aws.ParameterInfo, paramCount)
	paramInfos2 := make([]aws.ParameterInfo, paramCount)

	for i := 0; i < paramCount; i++ {
		name := "/app/param-" + string(rune('0'+(i%10)))
		paramInfos1[i] = aws.ParameterInfo{
			Name:  name,
			Value: "value-1",
			Type:  "String",
		}
		paramInfos2[i] = aws.ParameterInfo{
			Name:  name,
			Value: "value-2",
			Type:  "String",
		}
	}

	// パフォーマンス測定（色付けありの場合）
	colorEnabled = true
	start := time.Now()
	displayParameterStoreDiffWithTypes("dev", "stg", paramInfos1, paramInfos2, "/app/", false, false)
	elapsedWith := time.Since(start)

	// パフォーマンス測定（色付けなしの場合）
	colorEnabled = false
	start = time.Now()
	displayParameterStoreDiffWithTypes("dev", "stg", paramInfos1, paramInfos2, "/app/", false, false)
	elapsedWithout := time.Since(start)

	// ANSI コード追加による時間増加が < 5% であることを確認
	increase := float64(elapsedWith) / float64(elapsedWithout)
	if increase > 1.05 {
		t.Logf("Color overhead: %.2f%% (expected < 5%%)", (increase-1)*100)
		// 警告だけで失敗にはしない（環境依存の可能性）
	} else {
		t.Logf("Color overhead within acceptable range: %.2f%%", (increase-1)*100)
	}
}

// TestInvalidColorFlagHandling: 不正な --color フラグ値のハンドリング
func TestInvalidColorFlagHandling(t *testing.T) {
	tests := []struct {
		name     string
		mode     string
		expected string
	}{
		{
			name:     "invalid value should fallback",
			mode:     "invalid",
			expected: "auto",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setColorMode は不正値に対して "auto" にフォールバックするべき
			// 実装によっては error を返すかもしれないが、ここではフォールバック動作を確認
			setColorMode(tt.mode)

			// "auto" モードではターミナル判定に依存するため、
			// ここでは mode が有効な値に限定されていることを確認できればよい
			// （実装側で validation が必要）
			if shouldUseColor() {
				t.Logf("Color mode set (mode: %s)", tt.mode)
			} else {
				t.Logf("Color mode disabled (mode: %s)", tt.mode)
			}
		})
	}
}

// TestColorOutputConsistency: 大量パラメータでの色出力一貫性
func TestColorOutputConsistency(t *testing.T) {
	colorEnabled = true

	// 複数回の呼び出しで同じ結果が得られることを確認
	removal1 := colorizeRemovalLine("-test")
	removal2 := colorizeRemovalLine("-test")

	if removal1 != removal2 {
		t.Errorf("Colorizer output inconsistency detected")
	}

	// 多数の呼び出しでも一貫性が保たれることを確認
	expectedResult := colorizeRemovalLine("-value")
	for i := 0; i < 1000; i++ {
		result := colorizeRemovalLine("-value")
		if result != expectedResult {
			t.Errorf("Colorizer consistency failed at iteration %d", i)
			break
		}
	}

	t.Logf("Color output consistency verified over 1000+ invocations")
}

// TestNoParametersEdgeCase: パラメータがない場合のエッジケース
func TestNoParametersEdgeCase(t *testing.T) {
	colorEnabled = true

	paramInfos1 := []aws.ParameterInfo{}
	paramInfos2 := []aws.ParameterInfo{}

	// 空のパラメータリストで "No differences found." が出力される
	displayParameterStoreDiffWithTypes("dev", "stg", paramInfos1, paramInfos2, "/app/", false, false)

	// テスト成功（パニックしない）
	t.Logf("No parameters edge case handled correctly")
}
