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

// CompareJSONObjects は 2 つの JSON オブジェクトを再帰的に比較する
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

// FormatJSONDiffLine は差分の 1 行を unified diff 形式でフォーマットする
func FormatJSONDiffLine(op string, path string, value interface{}) string {
	// 値を文字列化
	var valueStr string
	switch v := value.(type) {
	case nil:
		valueStr = "null"
	case string:
		valueStr = v
	case bool:
		if v {
			valueStr = "true"
		} else {
			valueStr = "false"
		}
	case float64:
		// JSON の数値は float64 として扱われる
		// 整数の場合は小数点なしで表示
		if v == float64(int64(v)) {
			valueStr = fmt.Sprintf("%d", int64(v))
		} else {
			valueStr = fmt.Sprintf("%g", v)
		}
	default:
		valueStr = fmt.Sprintf("%v", v)
	}

	// unified diff 形式：{op} {path}: {value}
	return fmt.Sprintf("%s %s: %s", op, path, valueStr)
}

// FormatJSONDiffHeader は unified diff のヘッダー行を生成する
func FormatJSONDiffHeader(paramName, stage1, stage2 string) string {
	// unified diff 形式のヘッダー
	// --- /path [JSON] (stage1)
	// +++ /path [JSON] (stage2)
	removed := fmt.Sprintf("--- %s [JSON] (%s)", paramName, stage1)
	added := fmt.Sprintf("+++ %s [JSON] (%s)", paramName, stage2)
	return removed + "\n" + added
}

// formatSecureValue は SecureString の値をマスクまたは表示する
func formatSecureValue(value interface{}, showSecrets bool) string {
	if !showSecrets {
		return "***masked***"
	}

	// showSecrets が true の場合は実際の値を表示
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
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// FormatJSONDiffOutput は JSON 差分全体を unified diff 形式で出力する
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

// FormatJSONDiffOutputSecure は SecureString パラメータ用の差分出力（マスキング対応）
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

// CompareParameters は 2 つの Parameter を比較し、JSON 差分または nil を返す
// JSON 検出失敗時は nil を返し、呼び出し元は文字列比較にフォールバックする
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
	if err1 != nil && err2 != nil {
		// 両方パース失敗
		fmt.Fprintf(os.Stderr, "Warning: Failed to parse JSON for %s: both stages have invalid JSON\n", param1.Name)
	} else if err1 != nil {
		// param1 のみパース失敗
		fmt.Fprintf(os.Stderr, "Warning: Failed to parse JSON for %s (stage 1): %v\n", param1.Name, err1)
	} else if err2 != nil {
		// param2 のみパース失敗
		fmt.Fprintf(os.Stderr, "Warning: Failed to parse JSON for %s (stage 2): %v\n", param2.Name, err2)
	}

	// フォールバック：nil を返して文字列比較を促す
	return nil, nil
}
