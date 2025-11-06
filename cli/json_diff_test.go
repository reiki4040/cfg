package cli

import (
	"reflect"
	"testing"
)

// TestFlattenJSON_SingleLevel は単一階層の JSON オブジェクトをフラット化するテスト
func TestFlattenJSON_SingleLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		prefix   string
		expected map[string]interface{}
	}{
		{
			name: "single level with strings",
			input: map[string]interface{}{
				"host": "localhost",
				"port": "5432",
			},
			prefix: "",
			expected: map[string]interface{}{
				"host": "localhost",
				"port": "5432",
			},
		},
		{
			name: "single level with mixed types",
			input: map[string]interface{}{
				"host":    "localhost",
				"port":    float64(5432),
				"enabled": true,
			},
			prefix: "",
			expected: map[string]interface{}{
				"host":    "localhost",
				"port":    float64(5432),
				"enabled": true,
			},
		},
		{
			name: "single level with prefix",
			input: map[string]interface{}{
				"host": "localhost",
			},
			prefix: "database",
			expected: map[string]interface{}{
				"database.host": "localhost",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenJSON(tt.input, tt.prefix)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("flattenJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestFlattenJSON_EmptyAndNil は空の JSON オブジェクトと nil のエッジケースをテスト
func TestFlattenJSON_EmptyAndNil(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		prefix   string
		expected map[string]interface{}
	}{
		{
			name:     "empty object",
			input:    map[string]interface{}{},
			prefix:   "",
			expected: map[string]interface{}{},
		},
		{
			name:     "nil object",
			input:    nil,
			prefix:   "",
			expected: map[string]interface{}{},
		},
		{
			name: "object with null value",
			input: map[string]interface{}{
				"value": nil,
			},
			prefix: "",
			expected: map[string]interface{}{
				"value": nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenJSON(tt.input, tt.prefix)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("flattenJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestFlattenJSON_NestedObjects は 2-3 階層のネストされた JSON をフラット化するテスト
func TestFlattenJSON_NestedObjects(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		prefix   string
		expected map[string]interface{}
	}{
		{
			name: "2 levels nested",
			input: map[string]interface{}{
				"database": map[string]interface{}{
					"host": "localhost",
					"port": float64(5432),
				},
			},
			prefix: "",
			expected: map[string]interface{}{
				"database.host": "localhost",
				"database.port": float64(5432),
			},
		},
		{
			name: "3 levels nested",
			input: map[string]interface{}{
				"database": map[string]interface{}{
					"connection": map[string]interface{}{
						"host": "localhost",
						"port": float64(5432),
					},
				},
			},
			prefix: "",
			expected: map[string]interface{}{
				"database.connection.host": "localhost",
				"database.connection.port": float64(5432),
			},
		},
		{
			name: "mixed nested and flat",
			input: map[string]interface{}{
				"app_name": "myapp",
				"database": map[string]interface{}{
					"host": "localhost",
				},
			},
			prefix: "",
			expected: map[string]interface{}{
				"app_name":      "myapp",
				"database.host": "localhost",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenJSON(tt.input, tt.prefix)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("flattenJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestFlattenJSON_DeepNesting は 5 階層の深いネストをテスト
func TestFlattenJSON_DeepNesting(t *testing.T) {
	input := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"level3": map[string]interface{}{
					"level4": map[string]interface{}{
						"level5": "deep_value",
					},
				},
			},
		},
	}

	expected := map[string]interface{}{
		"level1.level2.level3.level4.level5": "deep_value",
	}

	result := flattenJSON(input, "")
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("flattenJSON() = %v, want %v", result, expected)
	}
}

// TestFlattenJSON_WithArrays は配列を含む JSON のフラット化をテスト
func TestFlattenJSON_WithArrays(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		prefix   string
		expected map[string]interface{}
	}{
		{
			name: "array of strings",
			input: map[string]interface{}{
				"tags": []interface{}{"tag1", "tag2", "tag3"},
			},
			prefix: "",
			expected: map[string]interface{}{
				"tags[0]": "tag1",
				"tags[1]": "tag2",
				"tags[2]": "tag3",
			},
		},
		{
			name: "array of objects",
			input: map[string]interface{}{
				"servers": []interface{}{
					map[string]interface{}{"host": "server1"},
					map[string]interface{}{"host": "server2"},
				},
			},
			prefix: "",
			expected: map[string]interface{}{
				"servers[0].host": "server1",
				"servers[1].host": "server2",
			},
		},
		{
			name: "nested object with array",
			input: map[string]interface{}{
				"config": map[string]interface{}{
					"ports": []interface{}{float64(8080), float64(8081)},
				},
			},
			prefix: "",
			expected: map[string]interface{}{
				"config.ports[0]": float64(8080),
				"config.ports[1]": float64(8081),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenJSON(tt.input, tt.prefix)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("flattenJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestFlattenJSON_SpecialCharactersInKeys はキーに特殊文字を含む JSON のテスト
func TestFlattenJSON_SpecialCharactersInKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		prefix   string
		expected map[string]interface{}
	}{
		{
			name: "key with dot",
			input: map[string]interface{}{
				"server.name": "localhost",
			},
			prefix: "",
			expected: map[string]interface{}{
				"server.name": "localhost",
			},
		},
		{
			name: "key with slash",
			input: map[string]interface{}{
				"path/to/file": "/usr/bin",
			},
			prefix: "",
			expected: map[string]interface{}{
				"path/to/file": "/usr/bin",
			},
		},
		{
			name: "nested with special chars",
			input: map[string]interface{}{
				"config": map[string]interface{}{
					"db.host": "localhost",
				},
			},
			prefix: "",
			expected: map[string]interface{}{
				"config.db.host": "localhost",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenJSON(tt.input, tt.prefix)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("flattenJSON() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestFlattenJSON_VeryDeepNesting は 10 階層を超えるネストをテスト
func TestFlattenJSON_VeryDeepNesting(t *testing.T) {
	// 12 階層のネスト構造を作成
	input := map[string]interface{}{
		"l1": map[string]interface{}{
			"l2": map[string]interface{}{
				"l3": map[string]interface{}{
					"l4": map[string]interface{}{
						"l5": map[string]interface{}{
							"l6": map[string]interface{}{
								"l7": map[string]interface{}{
									"l8": map[string]interface{}{
										"l9": map[string]interface{}{
											"l10": map[string]interface{}{
												"l11": map[string]interface{}{
													"l12": "very_deep_value",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	expected := map[string]interface{}{
		"l1.l2.l3.l4.l5.l6.l7.l8.l9.l10.l11.l12": "very_deep_value",
	}

	result := flattenJSON(input, "")
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("flattenJSON() = %v, want %v", result, expected)
	}
}

// TestCompareJSONObjects_Added は追加された属性を検出するテスト
func TestCompareJSONObjects_Added(t *testing.T) {
	obj1 := map[string]interface{}{
		"host": "localhost",
	}
	obj2 := map[string]interface{}{
		"host": "localhost",
		"port": float64(5432),
	}

	diff := CompareJSONObjects(obj1, obj2, "")

	// Added に port が含まれることを確認
	if len(diff.Added) != 1 {
		t.Errorf("Expected 1 added attribute, got %d", len(diff.Added))
	}
	if diff.Added["port"] != float64(5432) {
		t.Errorf("Expected port=5432, got %v", diff.Added["port"])
	}

	// Removed と Modified は空
	if len(diff.Removed) != 0 {
		t.Errorf("Expected 0 removed attributes, got %d", len(diff.Removed))
	}
	if len(diff.Modified) != 0 {
		t.Errorf("Expected 0 modified attributes, got %d", len(diff.Modified))
	}
}

// TestCompareJSONObjects_Removed は削除された属性を検出するテスト
func TestCompareJSONObjects_Removed(t *testing.T) {
	obj1 := map[string]interface{}{
		"host": "localhost",
		"port": float64(5432),
	}
	obj2 := map[string]interface{}{
		"host": "localhost",
	}

	diff := CompareJSONObjects(obj1, obj2, "")

	// Removed に port が含まれることを確認
	if len(diff.Removed) != 1 {
		t.Errorf("Expected 1 removed attribute, got %d", len(diff.Removed))
	}
	if diff.Removed["port"] != float64(5432) {
		t.Errorf("Expected port=5432, got %v", diff.Removed["port"])
	}

	// Added と Modified は空
	if len(diff.Added) != 0 {
		t.Errorf("Expected 0 added attributes, got %d", len(diff.Added))
	}
	if len(diff.Modified) != 0 {
		t.Errorf("Expected 0 modified attributes, got %d", len(diff.Modified))
	}
}

// TestCompareJSONObjects_Modified は変更された属性を検出するテスト
func TestCompareJSONObjects_Modified(t *testing.T) {
	obj1 := map[string]interface{}{
		"host": "localhost",
		"port": float64(5432),
	}
	obj2 := map[string]interface{}{
		"host": "remotehost",
		"port": float64(5432),
	}

	diff := CompareJSONObjects(obj1, obj2, "")

	// Modified に host が含まれることを確認
	if len(diff.Modified) != 1 {
		t.Errorf("Expected 1 modified attribute, got %d", len(diff.Modified))
	}
	if pair, ok := diff.Modified["host"]; !ok {
		t.Error("Expected host in Modified")
	} else {
		if pair.Old != "localhost" {
			t.Errorf("Expected old value=localhost, got %v", pair.Old)
		}
		if pair.New != "remotehost" {
			t.Errorf("Expected new value=remotehost, got %v", pair.New)
		}
	}

	// Added と Removed は空
	if len(diff.Added) != 0 {
		t.Errorf("Expected 0 added attributes, got %d", len(diff.Added))
	}
	if len(diff.Removed) != 0 {
		t.Errorf("Expected 0 removed attributes, got %d", len(diff.Removed))
	}
}

// TestCompareJSONObjects_NestedObjects はネストされたオブジェクトの比較をテスト
func TestCompareJSONObjects_NestedObjects(t *testing.T) {
	obj1 := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "localhost",
			"port": float64(5432),
		},
	}
	obj2 := map[string]interface{}{
		"database": map[string]interface{}{
			"host": "remotehost",
			"port": float64(5432),
		},
	}

	diff := CompareJSONObjects(obj1, obj2, "")

	// Modified に database.host が含まれることを確認
	if len(diff.Modified) != 1 {
		t.Errorf("Expected 1 modified attribute, got %d", len(diff.Modified))
	}
	if pair, ok := diff.Modified["database.host"]; !ok {
		t.Error("Expected database.host in Modified")
	} else {
		if pair.Old != "localhost" {
			t.Errorf("Expected old value=localhost, got %v", pair.Old)
		}
		if pair.New != "remotehost" {
			t.Errorf("Expected new value=remotehost, got %v", pair.New)
		}
	}
}

// TestCompareJSONObjects_Complex は複数の変更を含む複雑なケースをテスト
func TestCompareJSONObjects_Complex(t *testing.T) {
	obj1 := map[string]interface{}{
		"host":     "localhost",
		"port":     float64(5432),
		"username": "admin",
	}
	obj2 := map[string]interface{}{
		"host":     "remotehost",
		"port":     float64(5432),
		"password": "secret",
	}

	diff := CompareJSONObjects(obj1, obj2, "")

	// Modified: host
	if len(diff.Modified) != 1 {
		t.Errorf("Expected 1 modified attribute, got %d", len(diff.Modified))
	}
	if _, ok := diff.Modified["host"]; !ok {
		t.Error("Expected host in Modified")
	}

	// Added: password
	if len(diff.Added) != 1 {
		t.Errorf("Expected 1 added attribute, got %d", len(diff.Added))
	}
	if diff.Added["password"] != "secret" {
		t.Errorf("Expected password=secret, got %v", diff.Added["password"])
	}

	// Removed: username
	if len(diff.Removed) != 1 {
		t.Errorf("Expected 1 removed attribute, got %d", len(diff.Removed))
	}
	if diff.Removed["username"] != "admin" {
		t.Errorf("Expected username=admin, got %v", diff.Removed["username"])
	}
}

// TestCompareJSONObjects_TypeComparison は異なる型の値比較をテスト
func TestCompareJSONObjects_TypeComparison(t *testing.T) {
	tests := []struct {
		name     string
		obj1     map[string]interface{}
		obj2     map[string]interface{}
		expected JSONDiff
	}{
		{
			name: "string values",
			obj1: map[string]interface{}{"key": "value1"},
			obj2: map[string]interface{}{"key": "value2"},
			expected: JSONDiff{
				Added:   make(map[string]interface{}),
				Removed: make(map[string]interface{}),
				Modified: map[string]DiffPair{
					"key": {Old: "value1", New: "value2"},
				},
			},
		},
		{
			name: "numeric values",
			obj1: map[string]interface{}{"count": float64(10)},
			obj2: map[string]interface{}{"count": float64(20)},
			expected: JSONDiff{
				Added:   make(map[string]interface{}),
				Removed: make(map[string]interface{}),
				Modified: map[string]DiffPair{
					"count": {Old: float64(10), New: float64(20)},
				},
			},
		},
		{
			name: "boolean values",
			obj1: map[string]interface{}{"enabled": true},
			obj2: map[string]interface{}{"enabled": false},
			expected: JSONDiff{
				Added:   make(map[string]interface{}),
				Removed: make(map[string]interface{}),
				Modified: map[string]DiffPair{
					"enabled": {Old: true, New: false},
				},
			},
		},
		{
			name: "null values",
			obj1: map[string]interface{}{"value": "something"},
			obj2: map[string]interface{}{"value": nil},
			expected: JSONDiff{
				Added:   make(map[string]interface{}),
				Removed: make(map[string]interface{}),
				Modified: map[string]DiffPair{
					"value": {Old: "something", New: nil},
				},
			},
		},
		{
			name: "same values - no diff",
			obj1: map[string]interface{}{"key": "value"},
			obj2: map[string]interface{}{"key": "value"},
			expected: JSONDiff{
				Added:    make(map[string]interface{}),
				Removed:  make(map[string]interface{}),
				Modified: make(map[string]DiffPair),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := CompareJSONObjects(tt.obj1, tt.obj2, "")

			if len(diff.Added) != len(tt.expected.Added) {
				t.Errorf("Added: got %d items, want %d", len(diff.Added), len(tt.expected.Added))
			}
			if len(diff.Removed) != len(tt.expected.Removed) {
				t.Errorf("Removed: got %d items, want %d", len(diff.Removed), len(tt.expected.Removed))
			}
			if len(diff.Modified) != len(tt.expected.Modified) {
				t.Errorf("Modified: got %d items, want %d", len(diff.Modified), len(tt.expected.Modified))
			}

			// Modified の内容を検証
			for key, expectedPair := range tt.expected.Modified {
				actualPair, ok := diff.Modified[key]
				if !ok {
					t.Errorf("Expected key %s in Modified", key)
					continue
				}
				if !reflect.DeepEqual(actualPair.Old, expectedPair.Old) {
					t.Errorf("Old value for %s: got %v, want %v", key, actualPair.Old, expectedPair.Old)
				}
				if !reflect.DeepEqual(actualPair.New, expectedPair.New) {
					t.Errorf("New value for %s: got %v, want %v", key, actualPair.New, expectedPair.New)
				}
			}
		})
	}
}

// TestCompareJSONObjects_SameValues は同一値の属性が差分に含まれないことをテスト
func TestCompareJSONObjects_SameValues(t *testing.T) {
	obj1 := map[string]interface{}{
		"host":    "localhost",
		"port":    float64(5432),
		"enabled": true,
	}
	obj2 := map[string]interface{}{
		"host":    "localhost",
		"port":    float64(5432),
		"enabled": true,
	}

	diff := CompareJSONObjects(obj1, obj2, "")

	// すべて同一のため、差分は空
	if len(diff.Added) != 0 {
		t.Errorf("Expected 0 added attributes, got %d: %v", len(diff.Added), diff.Added)
	}
	if len(diff.Removed) != 0 {
		t.Errorf("Expected 0 removed attributes, got %d: %v", len(diff.Removed), diff.Removed)
	}
	if len(diff.Modified) != 0 {
		t.Errorf("Expected 0 modified attributes, got %d: %v", len(diff.Modified), diff.Modified)
	}
}

// TestCompareJSONObjects_SortedOutput は差分がソートされていることをテスト
func TestCompareJSONObjects_SortedOutput(t *testing.T) {
	obj1 := map[string]interface{}{
		"zebra": "z",
		"apple": "a",
		"mango": "m",
	}
	obj2 := map[string]interface{}{
		"zebra": "Z",
		"apple": "A",
		"mango": "M",
	}

	diff := CompareJSONObjects(obj1, obj2, "")

	// Modified に 3 つの属性があることを確認
	if len(diff.Modified) != 3 {
		t.Fatalf("Expected 3 modified attributes, got %d", len(diff.Modified))
	}

	// キーを取得してソートを確認
	keys := make([]string, 0, len(diff.Modified))
	for key := range diff.Modified {
		keys = append(keys, key)
	}

	// ソート済みであることを期待
	expectedOrder := []string{"apple", "mango", "zebra"}
	
	// Go の map は順序を保証しないため、
	// ここではソート機能が実装されているかを確認するために
	// 後続の GetSortedDiffKeys 関数を使用する想定
	// 今は diff.Modified にすべてのキーが含まれていることのみ確認
	for _, expectedKey := range expectedOrder {
		if _, ok := diff.Modified[expectedKey]; !ok {
			t.Errorf("Expected key %s in Modified", expectedKey)
		}
	}
}

// TestGetSortedDiffKeys はソート済みのキーリストを取得する関数のテスト
func TestGetSortedDiffKeys(t *testing.T) {
	diff := JSONDiff{
		Added: map[string]interface{}{
			"zoo":   "value",
			"apple": "value",
		},
		Removed: map[string]interface{}{
			"banana": "value",
		},
		Modified: map[string]DiffPair{
			"cherry": {Old: "old", New: "new"},
			"date":   {Old: "old", New: "new"},
		},
	}

	addedKeys := getSortedKeys(diff.Added)
	removedKeys := getSortedKeys(diff.Removed)
	modifiedKeys := getSortedModifiedKeys(diff.Modified)

	// Added のソート順を確認
	expectedAdded := []string{"apple", "zoo"}
	if !reflect.DeepEqual(addedKeys, expectedAdded) {
		t.Errorf("Added keys: got %v, want %v", addedKeys, expectedAdded)
	}

	// Removed のソート順を確認
	expectedRemoved := []string{"banana"}
	if !reflect.DeepEqual(removedKeys, expectedRemoved) {
		t.Errorf("Removed keys: got %v, want %v", removedKeys, expectedRemoved)
	}

	// Modified のソート順を確認
	expectedModified := []string{"cherry", "date"}
	if !reflect.DeepEqual(modifiedKeys, expectedModified) {
		t.Errorf("Modified keys: got %v, want %v", modifiedKeys, expectedModified)
	}
}
