package cli

import (
	"fmt"
	"reflect"
	"sort"
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
