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
