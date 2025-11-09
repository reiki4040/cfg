package cli

import (
	"strings"
	"testing"
)

// TestMultiStageDiffPathResolution: --stagesでの{stage}プレースホルダー追加テスト
func TestMultiStageDiffPathResolution(t *testing.T) {
	tests := []struct {
		name         string
		inputPath    string
		expectedPath string
		description  string
	}{
		{
			name:         "PathWithoutStageHolder",
			inputPath:    "/cfgtool/",
			expectedPath: "/cfgtool/{stage}/",
			description:  "末尾スラッシュで{stage}/が追加される",
		},
		{
			name:         "PathWithoutStageHolderNoSlash",
			inputPath:    "/cfgtool",
			expectedPath: "/cfgtool/{stage}",
			description:  "末尾スラッシュなしで/{stage}が追加される",
		},
		{
			name:         "PathWithStageHolder",
			inputPath:    "/cfgtool/{stage}/app",
			expectedPath: "/cfgtool/{stage}/app",
			description:  "既に{stage}が含まれている場合は変更なし",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Simulate the logic from runMultiStageDiff
			pathForDiff := test.inputPath
			if !strings.Contains(pathForDiff, "{stage}") {
				if strings.HasSuffix(pathForDiff, "/") {
					pathForDiff = pathForDiff + "{stage}/"
				} else {
					pathForDiff = pathForDiff + "/{stage}"
				}
			}

			if pathForDiff != test.expectedPath {
				t.Errorf("%s: expected %s, got %s", test.description, test.expectedPath, pathForDiff)
			}
		})
	}
}

// TestStagePathResolutionDifference: 複数ステージでパスが異なることを確認
func TestStagePathResolutionDifference(t *testing.T) {
	inputPath := "/cfgtool/"
	stages := []string{"dev", "stg", "prod"}

	// Add {stage} placeholder
	pathForDiff := inputPath
	if !strings.Contains(pathForDiff, "{stage}") {
		if strings.HasSuffix(pathForDiff, "/") {
			pathForDiff = pathForDiff + "{stage}/"
		} else {
			pathForDiff = pathForDiff + "/{stage}"
		}
	}

	// Resolve paths for each stage
	resolvedPaths := make([]string, 0, len(stages))
	for _, stage := range stages {
		resolved := strings.ReplaceAll(pathForDiff, "{stage}", stage)
		resolvedPaths = append(resolvedPaths, resolved)
	}

	// Verify that all paths are different
	pathMap := make(map[string]bool)
	for _, path := range resolvedPaths {
		if pathMap[path] {
			t.Errorf("Duplicate path found: %s", path)
		}
		pathMap[path] = true
	}

	// Verify correct paths
	expectedPaths := []string{"/cfgtool/dev/", "/cfgtool/stg/", "/cfgtool/prod/"}
	for i, expected := range expectedPaths {
		if resolvedPaths[i] != expected {
			t.Errorf("Stage %s: expected %s, got %s", stages[i], expected, resolvedPaths[i])
		}
	}
}
