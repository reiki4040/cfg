package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"time"

	"github.com/reiki4040/cfg/aws"
)

// JSONDiff は JSON オブジェクトの差分を表現する
type JSONDiff struct {
	Added    map[string]interface{} // 追加された属性 (path -> value)
	Removed  map[string]interface{} // 削除された属性 (path -> value)
	Modified map[string]DiffPair     // 変更された属性 (path -> (old, new))
}

// DiffPair は変更前後の値ペア
type DiffPair struct {
	Old interface{}
	New interface{}
}

// flattenJSON は JSON オブジェクトをフラットなマップに変換する
// 例: {"db": {"host": "localhost"}} -> {"db.host": "localhost"}
// 配列は [0], [1] のようにインデックスで表現される
func flattenJSON(obj map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})

	// nil チェック
	if obj == nil {
		return result
	}

	for key, value := range obj {
		// フルパスを構築
		var fullPath string
		if prefix == "" {
			fullPath = key
		} else {
			fullPath = prefix + "." + key
		}

		// 値の型に応じて処理を分岐
		switch v := value.(type) {
		case map[string]interface{}:
			// ネストされたオブジェクトを再帰的にフラット化
			nested := flattenJSON(v, fullPath)
			for k, val := range nested {
				result[k] = val
			}
		case []interface{}:
			// 配列を展開
			for i, item := range v {
				arrayPath := fmt.Sprintf("%s[%d]", fullPath, i)
				if nestedMap, ok := item.(map[string]interface{}); ok {
					// 配列内のオブジェクトを再帰的にフラット化
					nested := flattenJSON(nestedMap, arrayPath)
					for k, val := range nested {
						result[k] = val
					}
				} else {
					// プリミティブ値はそのまま格納
					result[arrayPath] = item
				}
			}
		default:
			// プリミティブ値はそのまま格納
			result[fullPath] = value
		}
	}

	return result
}

// CompareJSONObjects は 2 つの JSON オブジェクトを再帰的に比較し、差分を返します。
// ネストされたオブジェクトと配列は自動的にフラット化され、ドット記法のパスで表現されます。
// 例: {"database": {"host": "localhost"}} -> "database.host"
//
// Parameters:
//   - obj1: 比較元の JSON オブジェクト
//   - obj2: 比較先の JSON オブジェクト
//   - basePath: 属性パスのプレフィックス（通常は空文字列）
//
// Returns:
//   - JSONDiff: 追加・削除・変更された属性を含む差分オブジェクト
func CompareJSONObjects(obj1, obj2 map[string]interface{}, basePath string) JSONDiff {
	// 両方のオブジェクトをフラット化
	flat1 := flattenJSON(obj1, basePath)
	flat2 := flattenJSON(obj2, basePath)

	// 差分を格納する構造体を初期化
	diff := JSONDiff{
		Added:    make(map[string]interface{}),
		Removed:  make(map[string]interface{}),
		Modified: make(map[string]DiffPair),
	}

	// すべてのキーを収集
	allKeys := make(map[string]bool)
	for key := range flat1 {
		allKeys[key] = true
	}
	for key := range flat2 {
		allKeys[key] = true
	}

	// 各キーについて差分を分類
	for key := range allKeys {
		val1, exists1 := flat1[key]
		val2, exists2 := flat2[key]

		if exists1 && !exists2 {
			// obj1 にのみ存在 → Removed
			diff.Removed[key] = val1
		} else if !exists1 && exists2 {
			// obj2 にのみ存在 → Added
			diff.Added[key] = val2
		} else if exists1 && exists2 {
			// 両方に存在 → 値が異なる場合は Modified
			if !valuesEqual(val1, val2) {
				diff.Modified[key] = DiffPair{
					Old: val1,
					New: val2,
				}
			}
		}
	}

	return diff
}

// valuesEqual は 2 つの値が等しいかを判定する
func valuesEqual(v1, v2 interface{}) bool {
	// reflect.DeepEqual を使用して値と型を厳密に比較
	// JSON unmarshal の結果として、数値は float64、真偽値は bool として扱われる
	return reflect.DeepEqual(v1, v2)
}

// getSortedKeys はマップのキーをソート済みのスライスとして返す
func getSortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// getSortedModifiedKeys は Modified マップのキーをソート済みのスライスとして返す
func getSortedModifiedKeys(m map[string]DiffPair) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// formatValueAsString は JSON 値を文字列に変換します。
// 数値、真偽値、null を適切な形式で文字列化します。
func formatValueAsString(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return "null"
	case string:
		return v
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		// JSON の数値は float64 として扱われる
		// 整数の場合は小数点なしで表示
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// FormatJSONDiffLine は差分の 1 行を unified diff 形式でフォーマットします。
// 出力形式: "{op} {path}: {value}" (例: "+ database.host: localhost")
//
// Parameters:
//   - op: 操作タイプ（"+" は追加、"-" は削除）
//   - path: 属性パス（ドット記法）
//   - value: 属性値（文字列、数値、真偽値、null など）
//
// Returns:
//   - string: フォーマットされた差分行
func FormatJSONDiffLine(op string, path string, value interface{}) string {
	valueStr := formatValueAsString(value)
	// unified diff 形式：{op} {path}: {value}
	return fmt.Sprintf("%s %s: %s", op, path, valueStr)
}

// FormatJSONDiffHeader は unified diff のヘッダー行を生成します。
// [JSON] タグを含むヘッダーを生成し、JSON 差分であることを明示します。
//
// Parameters:
//   - paramName: パラメータ名（Parameter Store のパス）
//   - stage1: 比較元の Stage 名（例: "dev"）
//   - stage2: 比較先の Stage 名（例: "prod"）
//
// Returns:
//   - string: unified diff 形式のヘッダー（--- と +++ の2行）
func FormatJSONDiffHeader(paramName, stage1, stage2 string) string {
	// unified diff 形式のヘッダー
	// --- /path [JSON] (stage1)
	// +++ /path [JSON] (stage2)
	removed := fmt.Sprintf("--- %s [JSON] (%s)", paramName, stage1)
	added := fmt.Sprintf("+++ %s [JSON] (%s)", paramName, stage2)
	return removed + "\n" + added
}

// formatSecureValue は SecureString の値をマスクまたは表示します。
// showSecrets が false の場合は "***masked***" を返し、true の場合は実際の値を表示します。
func formatSecureValue(value interface{}, showSecrets bool) string {
	if !showSecrets {
		return "***masked***"
	}
	// showSecrets が true の場合は実際の値を表示
	return formatValueAsString(value)
}

// FormatJSONDiffOutput は JSON 差分全体を unified diff 形式で出力します。
// 追加・削除・変更された属性をアルファベット順にソートして表示します。
//
// Parameters:
//   - paramName: パラメータ名（Parameter Store のパス）
//   - stage1: 比較元の Stage 名
//   - stage2: 比較先の Stage 名
//   - diff: CompareJSONObjects() から返された差分オブジェクト
//   - showSecrets: true の場合、SecureString の実際の値を表示（未使用）
//
// Returns:
//   - string: unified diff 形式の完全な差分出力
func FormatJSONDiffOutput(paramName, stage1, stage2 string, diff JSONDiff, showSecrets bool) string {
	var output string

	// ヘッダー
	output += FormatJSONDiffHeader(paramName, stage1, stage2) + "\n"

	// チャンクヘッダー
	output += "@@ JSON Attribute Differences @@\n"

	// Removed（削除された属性）
	removedKeys := getSortedKeys(diff.Removed)
	for _, key := range removedKeys {
		output += FormatJSONDiffLine("-", key, diff.Removed[key]) + "\n"
	}

	// Modified（変更された属性）- 削除と追加のペアで表示
	modifiedKeys := getSortedModifiedKeys(diff.Modified)
	for _, key := range modifiedKeys {
		pair := diff.Modified[key]
		output += FormatJSONDiffLine("-", key, pair.Old) + "\n"
		output += FormatJSONDiffLine("+", key, pair.New) + "\n"
	}

	// Added（追加された属性）
	addedKeys := getSortedKeys(diff.Added)
	for _, key := range addedKeys {
		output += FormatJSONDiffLine("+", key, diff.Added[key]) + "\n"
	}

	return output
}

// FormatJSONDiffOutputSecure は SecureString パラメータ用の差分出力を生成します。
// showSecrets が false の場合、すべての値を "***masked***" でマスキングします。
//
// Parameters:
//   - paramName: パラメータ名（Parameter Store のパス）
//   - stage1: 比較元の Stage 名
//   - stage2: 比較先の Stage 名
//   - diff: CompareJSONObjects() から返された差分オブジェクト
//   - showSecrets: true の場合、実際の値を表示；false の場合、"***masked***" でマスキング
//   - paramType: パラメータタイプ（"SecureString" の場合のみマスキング適用）
//
// Returns:
//   - string: unified diff 形式の差分出力（SecureString の場合はマスキング済み）
func FormatJSONDiffOutputSecure(paramName, stage1, stage2 string, diff JSONDiff, showSecrets bool, paramType string) string {
	// SecureString でない場合は通常の出力
	if paramType != "SecureString" {
		return FormatJSONDiffOutput(paramName, stage1, stage2, diff, showSecrets)
	}

	var output string

	// ヘッダー
	output += FormatJSONDiffHeader(paramName, stage1, stage2) + "\n"

	// チャンクヘッダー
	output += "@@ JSON Attribute Differences @@\n"

	// Removed（削除された属性）
	removedKeys := getSortedKeys(diff.Removed)
	for _, key := range removedKeys {
		maskedValue := formatSecureValue(diff.Removed[key], showSecrets)
		output += fmt.Sprintf("- %s: %s\n", key, maskedValue)
	}

	// Modified（変更された属性）
	modifiedKeys := getSortedModifiedKeys(diff.Modified)
	for _, key := range modifiedKeys {
		pair := diff.Modified[key]
		maskedOld := formatSecureValue(pair.Old, showSecrets)
		maskedNew := formatSecureValue(pair.New, showSecrets)
		output += fmt.Sprintf("- %s: %s\n", key, maskedOld)
		output += fmt.Sprintf("+ %s: %s\n", key, maskedNew)
	}

	// Added（追加された属性）
	addedKeys := getSortedKeys(diff.Added)
	for _, key := range addedKeys {
		maskedValue := formatSecureValue(diff.Added[key], showSecrets)
		output += fmt.Sprintf("+ %s: %s\n", key, maskedValue)
	}

	return output
}

// FormatJSONSummary は JSON 値のサマリーを生成します。
// マルチステージ diff 表示で長い JSON 値を簡潔に表示するために使用します。
// フラット化後のキー数を [JSON: N keys] 形式で表示します。
//
// Parameters:
//   - jsonStr: JSON 文字列（有効・無効どちらも受け付ける）
//
// Returns:
//   - string: "[JSON: N keys]" 形式のサマリー、または無効な JSON の場合は truncate された文字列
func FormatJSONSummary(jsonStr string) string {
	// JSON としてパースを試行
	var obj map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &obj)

	if err != nil {
		// 無効な JSON の場合は truncate して返す
		if len(jsonStr) > 50 {
			return jsonStr[:50] + "..."
		}
		return jsonStr
	}

	// JSON をフラット化してキー数をカウント
	flattened := flattenJSON(obj, "")
	keyCount := len(flattened)

	// キー数に応じて単数/複数形を使い分け
	if keyCount == 1 {
		return "[JSON: 1 key]"
	}
	return fmt.Sprintf("[JSON: %d keys]", keyCount)
}

// CompareParameters は 2 つの Parameter Store パラメータを比較し、JSON 差分または nil を返します。
// Optimistic Parsing アプローチを使用し、JSON として有効な場合のみ属性レベルの差分を返します。
// JSON 検出失敗時やサイズ超過時は nil を返し、呼び出し元が文字列比較にフォールバックします。
//
// Parameters:
//   - param1: 比較元のパラメータ（通常は stage1）
//   - param2: 比較先のパラメータ（通常は stage2）
//   - noJSONDiff: true の場合、JSON 差分をスキップして nil を返す
//
// Returns:
//   - *JSONDiff: JSON 差分オブジェクト（JSON として有効な場合）、またはフォールバック時は nil
//   - error: 常に nil（将来の拡張用）
func CompareParameters(param1, param2 aws.ParameterInfo, noJSONDiff bool) (*JSONDiff, error) {
	// --no-json-diff フラグが指定されている場合は JSON 差分をスキップ
	if noJSONDiff {
		return nil, nil
	}

	// サイズチェック：10MB を超える JSON はフォールバック
	const maxJSONSize = 10 * 1024 * 1024 // 10MB
	if len(param1.Value) > maxJSONSize || len(param2.Value) > maxJSONSize {
		sizeMB := float64(len(param1.Value)) / (1024 * 1024)
		if len(param2.Value) > len(param1.Value) {
			sizeMB = float64(len(param2.Value)) / (1024 * 1024)
		}
		fmt.Fprintf(os.Stderr, "Warning: JSON size exceeds 10MB for %s (%.1fMB), using string comparison\n",
			param1.Name, sizeMB)
		return nil, nil
	}

	// パフォーマンス測定開始
	startTime := time.Now()

	// JSON として両方をパースしてみる（Optimistic Parsing）
	var obj1, obj2 map[string]interface{}
	err1 := json.Unmarshal([]byte(param1.Value), &obj1)
	err2 := json.Unmarshal([]byte(param2.Value), &obj2)

	// 両方とも JSON パースに成功した場合のみ JSON 差分を実行
	if err1 == nil && err2 == nil {
		diff := CompareJSONObjects(obj1, obj2, "")

		// パフォーマンス情報の出力（1000属性を超える場合）
		totalAttributes := len(diff.Added) + len(diff.Removed) + len(diff.Modified)
		const largeAttributeThreshold = 1000
		if totalAttributes > largeAttributeThreshold {
			elapsed := time.Since(startTime)
			fmt.Fprintf(os.Stderr, "Info: Large JSON (%d attributes) for %s, comparison took %v\n",
				totalAttributes, param1.Name, elapsed)
		}

		return &diff, nil
	}

	// パース失敗時は警告を出力して nil を返す（フォールバック）
	// ただし、両方のパラメータが明らかに JSON ではない場合（短い文字列）は警告を出さない
	if err1 != nil && err2 != nil {
		// 両方パース失敗の場合、少なくとも1つが JSON らしき形式の場合のみ警告を出す
		// JSON のような形式：{ または [ で始まる
		if (len(param1.Value) > 0 && (param1.Value[0] == '{' || param1.Value[0] == '[')) ||
			(len(param2.Value) > 0 && (param2.Value[0] == '{' || param2.Value[0] == '[')) {
			fmt.Fprintf(os.Stderr, "Warning: Failed to parse JSON for %s: both stages have invalid JSON\n", param1.Name)
		}
	} else if err1 != nil {
		// param1 のみパース失敗、かつ JSON のような形式の場合のみ警告を出す
		if len(param1.Value) > 0 && (param1.Value[0] == '{' || param1.Value[0] == '[') {
			fmt.Fprintf(os.Stderr, "Warning: Failed to parse JSON for %s (stage 1): %v\n", param1.Name, err1)
		}
	} else if err2 != nil {
		// param2 のみパース失敗、かつ JSON のような形式の場合のみ警告を出す
		if len(param2.Value) > 0 && (param2.Value[0] == '{' || param2.Value[0] == '[') {
			fmt.Fprintf(os.Stderr, "Warning: Failed to parse JSON for %s (stage 2): %v\n", param2.Name, err2)
		}
	}

	// フォールバック：nil を返して文字列比較を促す
	return nil, nil
}
