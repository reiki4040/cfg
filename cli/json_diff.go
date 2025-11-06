package cli

import (
	"fmt"
)

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
