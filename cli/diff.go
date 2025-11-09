package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/reiki4040/cfg"
	"github.com/reiki4040/cfg/aws"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare Parameter Store between stages",
	Long: `Compare Parameter Store parameters between stages.

Color Output:
By default, diff output includes ANSI color codes for better readability:
  - Removed parameters are shown in RED
  - Added parameters are shown in GREEN
  - Changed parameters are shown in YELLOW
  - Unchanged parameters are shown in DEFAULT color

Color Mode Control (--color flag):
  auto   (default): Automatically detect if output is a terminal
  always: Force color output even when piping
  never:  Disable color output (useful for logs and CI/CD)

Examples:
  cfgctl diff --stage=dev --compare-stage=stg --path=/app/
  cfgctl diff --stages=dev,stg,prod --path=/app/
  cfgctl diff --stage=dev --compare-stage=stg --keys-only
  cfgctl diff --stage=dev --compare-stage=stg --show-secrets
  cfgctl diff --stage=dev --compare-stage=stg --compare-profile=prod-profile
  cfgctl diff --stage=dev --compare-stage=stg --color=always
  cfgctl diff --stage=dev --compare-stage=stg --color=never`,
	RunE: runDiffPsCommand,
}

var (
	diffCompareStage   string
	diffStages         string
	diffPath           string
	diffKeysOnly       bool
	diffShowSecrets    bool
	diffCompareProfile string
	diffNoJSONDiff     bool
	diffColorMode      string
)

func init() {
	rootCmd.AddCommand(diffCmd)

	diffCmd.Flags().StringVar(&diffCompareStage, "compare-stage", "", "Stage to compare with")
	diffCmd.Flags().StringVar(&diffStages, "stages", "", "Comma-separated list of stages to compare")
	diffCmd.Flags().StringVar(&diffPath, "path", "", "Parameter path to compare (default: path prefix)")
	diffCmd.Flags().BoolVar(&diffKeysOnly, "keys-only", false, "Show only parameter names without values")
	diffCmd.Flags().BoolVar(&diffShowSecrets, "show-secrets", false, "Show SecureString parameter values (DANGEROUS)")
	diffCmd.Flags().StringVar(&diffCompareProfile, "compare-profile", "", "AWS profile for comparison target")
	diffCmd.Flags().BoolVar(&diffNoJSONDiff, "no-json-diff", false, "Disable JSON attribute-level diff (use string comparison)")
	diffCmd.Flags().StringVar(&diffColorMode, "color", "auto", "Color output mode: auto (default), always, never")
}

func runDiffPsCommand(cmd *cobra.Command, args []string) error {
	// 色出力モードを初期化
	setColorMode(diffColorMode)

	if diffStages != "" {
		return runMultiStageDiff()
	}

	if diffCompareStage == "" {
		return fmt.Errorf("either --compare-stage or --stages must be specified")
	}

	// Determine comparison path
	comparePath := diffPath
	if comparePath == "" {
		// Use path prefix from config
		comparePath = getPathPrefix()
	}

	// Create AWS client
	awsClient, err := createAWSClient()
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Create second AWS client for comparison if different profile specified
	var compareAWSClient *aws.ParameterStoreClient
	if diffCompareProfile != "" {
		compareAWSClient, err = aws.NewParameterStoreClientWithProfile(context.Background(), awsRegion, diffCompareProfile)
		if err != nil {
			return fmt.Errorf("failed to create comparison AWS client with profile %s: %w", diffCompareProfile, err)
		}
	} else {
		compareAWSClient = awsClient
	}

	// Create stage resolvers
	stageResolver1 := cfg.NewStageResolver(stage)
	stageResolver2 := cfg.NewStageResolver(diffCompareStage)

	// 比較パスにステージプレースホルダーがない場合、追加する
	// これにより、diff での stage 別パラメータ取得が正しく機能する
	pathForDiff := comparePath
	if !strings.Contains(pathForDiff, "{stage}") {
		// パスに {stage} が含まれていない場合、パスの末尾に応じて追加
		if strings.HasSuffix(pathForDiff, "/") {
			pathForDiff = pathForDiff + "{stage}/"
		} else {
			pathForDiff = pathForDiff + "/{stage}"
		}
	}

	resolvedPath1 := stageResolver1.ResolvePath(pathForDiff)
	resolvedPath2 := stageResolver2.ResolvePath(pathForDiff)

	// Get parameter infos for both stages
	ctx := context.Background()
	// Decrypt parameters if we want to show secrets
	shouldDecrypt := diffShowSecrets
	paramInfos1, err := awsClient.GetParameterInfosByPath(ctx, resolvedPath1, true, shouldDecrypt)
	if err != nil {
		paramInfos1 = []aws.ParameterInfo{}
	}

	paramInfos2, err := compareAWSClient.GetParameterInfosByPath(ctx, resolvedPath2, true, shouldDecrypt)
	if err != nil {
		paramInfos2 = []aws.ParameterInfo{}
	}

	// Display comparison
	displayParameterStoreDiffWithTypes(stageResolver1.GetStage(), stageResolver2.GetStage(), paramInfos1, paramInfos2, comparePath, diffKeysOnly, diffShowSecrets)

	return nil
}

func runMultiStageDiff() error {
	stages := strings.Split(diffStages, ",")
	if len(stages) < 2 {
		return fmt.Errorf("at least 2 stages required for comparison")
	}

	// Determine comparison path
	comparePath := diffPath
	if comparePath == "" {
		// Use path prefix from config
		comparePath = getPathPrefix()
	}

	// Add {stage} placeholder if not present
	pathForDiff := comparePath
	if !strings.Contains(pathForDiff, "{stage}") {
		// パスに {stage} が含まれていない場合、追加する
		if strings.HasSuffix(pathForDiff, "/") {
			pathForDiff = pathForDiff + "{stage}/"
		} else {
			pathForDiff = pathForDiff + "/{stage}"
		}
	}

	// Create AWS client
	awsClient, err := createAWSClient()
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	ctx := context.Background()

	// stageParamInfosMapは、正規化されたキー -> stage -> ParameterInfo のマップ
	stageParamInfosMap := make(map[string]map[string]aws.ParameterInfo)

	// normalizationMap: 各stageでの正規化方法を保持
	normalizeKey := func(paramName string, stageToNormalize string) string {
		// Remove stage-specific prefix to get comparable key
		return strings.Replace(paramName, "/"+stageToNormalize+"/", "/{stage}/", 1)
	}

	// Get parameters for each stage
	for _, s := range stages {
		stageResolver := cfg.NewStageResolver(strings.TrimSpace(s))
		resolvedPath := stageResolver.ResolvePath(pathForDiff)
		stageName := stageResolver.GetStage()

		// Decrypt parameters if we want to show secrets
		shouldDecrypt := diffShowSecrets
		paramInfos, err := awsClient.GetParameterInfosByPath(ctx, resolvedPath, true, shouldDecrypt)
		if err != nil {
			paramInfos = []aws.ParameterInfo{}
		}

		// Normalize parameter names and build the map
		for _, param := range paramInfos {
			normalizedKey := normalizeKey(param.Name, stageName)

			if stageParamInfosMap[normalizedKey] == nil {
				stageParamInfosMap[normalizedKey] = make(map[string]aws.ParameterInfo)
			}
			stageParamInfosMap[normalizedKey][stageName] = param
		}
	}

	// Display multi-stage comparison
	displayMultiStageDiffWithTypes(stages, stageParamInfosMap, comparePath, diffKeysOnly, diffShowSecrets)

	return nil
}

func displayParameterStoreDiffWithTypes(stage1, stage2 string, paramInfos1, paramInfos2 []aws.ParameterInfo, path string, keysOnly bool, showSecrets bool) {
	// Create maps organized by normalized key (removing stage-specific parts)
	params1 := make(map[string]aws.ParameterInfo)
	params2 := make(map[string]aws.ParameterInfo)

	// Helper function to normalize parameter name for comparison
	normalizeKey := func(paramName string, stageToRemove string) string {
		// Remove stage-specific prefix to get comparable key
		return strings.Replace(paramName, "/"+stageToRemove+"/", "/{stage}/", 1)
	}

	for _, param := range paramInfos1 {
		normalizedKey := normalizeKey(param.Name, stage1)
		params1[normalizedKey] = param
	}
	for _, param := range paramInfos2 {
		normalizedKey := normalizeKey(param.Name, stage2)
		params2[normalizedKey] = param
	}

	fmt.Printf("Parameter Store Comparison: %s vs %s (path: %s)\n\n", stage1, stage2, path)

	// Find keys that exist in only one stage
	allKeys := make(map[string]bool)
	for key := range params1 {
		allKeys[key] = true
	}
	for key := range params2 {
		allKeys[key] = true
	}

	var onlyIn1, onlyIn2, different, maskedNoChange []string

	for key := range allKeys {
		param1, exists1 := params1[key]
		param2, exists2 := params2[key]

		if exists1 && !exists2 {
			onlyIn1 = append(onlyIn1, key)
		} else if !exists1 && exists2 {
			onlyIn2 = append(onlyIn2, key)
		} else if exists1 && exists2 {
			// Mask状態のときは値を比較せず、show secretsの時だけ差分判定する
			if (param1.Type == "SecureString" || param2.Type == "SecureString") && !showSecrets {
				// SecureStringがmask状態の場合は、比較なしで「存在する」として扱う
				// 色分けなしで出力するため maskedNoChange カテゴリに追加
				maskedNoChange = append(maskedNoChange, key)
			} else if param1.Value != param2.Value {
				different = append(different, key)
			}
		}
	}

	// Sort for consistent output
	sort.Strings(onlyIn1)
	sort.Strings(onlyIn2)
	sort.Strings(different)
	sort.Strings(maskedNoChange)

	hasChanges := len(onlyIn1) > 0 || len(onlyIn2) > 0 || len(different) > 0 || len(maskedNoChange) > 0

	if !hasChanges {
		fmt.Println("No differences found.")
		return
	}

	if len(onlyIn1) > 0 {
		for _, key := range onlyIn1 {
			param := params1[key]
			if keysOnly {
				fmt.Printf("%s\n", colorizeRemovalLine("- "+key))
			} else {
				// Show actual resolved parameter paths for each stage
				param1Path := param.Name
				param2Path := strings.Replace(key, "/{stage}/", "/"+stage2+"/", 1)
				fmt.Printf("%s\n", colorizeHeaderLine("--- "+param1Path+" ("+stage1+")"))
				fmt.Printf("%s\n", colorizeHeaderLine("+++ "+param2Path+" ("+stage2+")"))
				fmt.Printf("%s\n", colorizeHeaderLine("@@ -1 +0,0 @@"))
				if param.Type == "SecureString" && !showSecrets {
					fmt.Printf("%s\n", colorizeRemovalLine("-***masked secret***"))
				} else {
					fmt.Printf("%s\n", colorizeRemovalLine("-"+param.Value))
				}
				fmt.Println()
			}
		}
	}

	if len(onlyIn2) > 0 {
		for _, key := range onlyIn2 {
			param := params2[key]
			if keysOnly {
				fmt.Printf("%s\n", colorizeAdditionLine("+ "+key))
			} else {
				// Show actual resolved parameter paths for each stage
				param1Path := strings.Replace(key, "/{stage}/", "/"+stage1+"/", 1)
				param2Path := param.Name
				fmt.Printf("%s\n", colorizeHeaderLine("--- "+param1Path+" ("+stage1+")"))
				fmt.Printf("%s\n", colorizeHeaderLine("+++ "+param2Path+" ("+stage2+")"))
				fmt.Printf("%s\n", colorizeHeaderLine("@@ -0,0 +1 @@"))
				if param.Type == "SecureString" && !showSecrets {
					fmt.Printf("%s\n", colorizeAdditionLine("+***masked secret***"))
				} else {
					fmt.Printf("%s\n", colorizeAdditionLine("+"+param.Value))
				}
				fmt.Println()
			}
		}
	}

	if len(different) > 0 {
		for _, key := range different {
			param1 := params1[key]
			param2 := params2[key]
			if keysOnly {
				fmt.Printf("%s\n", colorizeChangeLine("~ "+key))
			} else {
				// Try JSON diff if not disabled and not keys-only mode
				var jsonDiffOutput string
				if !diffNoJSONDiff {
					diff, err := CompareParameters(param1, param2, diffNoJSONDiff)
					if err == nil && diff != nil {
						// JSON diff succeeded
						if param1.Type == "SecureString" || param2.Type == "SecureString" {
							jsonDiffOutput = FormatJSONDiffOutputSecure(param1.Name, stage1, stage2, *diff, showSecrets, param1.Type)
						} else {
							jsonDiffOutput = FormatJSONDiffOutput(param1.Name, stage1, stage2, *diff, showSecrets)
						}
						// JSON diff output に色付けを適用
						jsonDiffOutput = colorizeJsonDiffOutput(jsonDiffOutput)
					}
				}

				// If JSON diff was successful, use it; otherwise fall back to string comparison
				if jsonDiffOutput != "" {
					fmt.Print(jsonDiffOutput)
				} else {
					// Fall back to string comparison
					param1Path := param1.Name
					param2Path := param2.Name
					fmt.Printf("%s\n", colorizeHeaderLine("--- "+param1Path+" ("+stage1+")"))
					fmt.Printf("%s\n", colorizeHeaderLine("+++ "+param2Path+" ("+stage2+")"))
					fmt.Printf("%s\n", colorizeHeaderLine("@@ -1 +1 @@"))
					if (param1.Type == "SecureString" || param2.Type == "SecureString") && !showSecrets {
						fmt.Printf("%s\n", colorizeRemovalLine("-***masked secret***"))
						fmt.Printf("%s\n", colorizeAdditionLine("+***masked secret***"))
					} else {
						fmt.Printf("%s\n", colorizeRemovalLine("-"+param1.Value))
						fmt.Printf("%s\n", colorizeAdditionLine("+"+param2.Value))
					}
					fmt.Println()
				}
			}
		}
	}

	if len(maskedNoChange) > 0 {
		for _, key := range maskedNoChange {
			param1 := params1[key]
			param2 := params2[key]
			if keysOnly {
				// Keys onlyモードではmask状態のパラメータも表示
				fmt.Printf("%s\n", key)
			} else {
				// Mask状態のSecureString：色分けなしで表示
				// 通常文字色で「存在する」ことを示す
				param1Path := param1.Name
				param2Path := param2.Name
				fmt.Printf("--- %s (%s)\n", param1Path, stage1)
				fmt.Printf("+++ %s (%s)\n", param2Path, stage2)
				fmt.Printf("***masked secret***\n")
				fmt.Println()
			}
		}
	}
}

// isValuesVarying: 複数ステージでの値が異なるかを判定
func isValuesVarying(valuesByStage map[string]string, existenceByStage map[string]bool) bool {
	// ステージ間で存在の有無が異なる場合
	existCount := 0
	for _, exists := range existenceByStage {
		if exists {
			existCount++
		}
	}
	if existCount != len(existenceByStage) && existCount > 0 {
		return true // 一部ステージにのみ存在
	}

	// 値が異なる場合
	if len(valuesByStage) == 0 {
		return false
	}

	var firstValue string
	for _, value := range valuesByStage {
		if firstValue == "" {
			firstValue = value
		} else if value != firstValue {
			return true // 値が異なる
		}
	}

	return false // すべてのステージで同一の値
}

func displayParameterStoreDiff(stage1, stage2 string, params1, params2 map[string]string, path string) {
	fmt.Printf("Parameter Store Comparison: %s vs %s (path: %s)\n\n", stage1, stage2, path)

	// Find keys that exist in only one stage
	allKeys := make(map[string]bool)
	for key := range params1 {
		allKeys[key] = true
	}
	for key := range params2 {
		allKeys[key] = true
	}

	var onlyIn1, onlyIn2, different []string

	for key := range allKeys {
		val1, exists1 := params1[key]
		val2, exists2 := params2[key]

		if exists1 && !exists2 {
			onlyIn1 = append(onlyIn1, key)
		} else if !exists1 && exists2 {
			onlyIn2 = append(onlyIn2, key)
		} else if exists1 && exists2 && val1 != val2 {
			different = append(different, key)
		}
	}

	// Sort for consistent output
	sort.Strings(onlyIn1)
	sort.Strings(onlyIn2)
	sort.Strings(different)

	hasChanges := len(onlyIn1) > 0 || len(onlyIn2) > 0 || len(different) > 0

	if !hasChanges {
		fmt.Println("No differences found.")
		return
	}

	if len(onlyIn1) > 0 {
		for _, key := range onlyIn1 {
			fmt.Printf("--- %s\t(%s)\n", key, stage1)
			fmt.Printf("+++ %s\t(%s)\n", key, stage2)
			fmt.Printf("@@ -1 +0,0 @@\n")
			if isSecretParameter(key) {
				fmt.Printf("-*****\n")
			} else {
				fmt.Printf("-%s\n", params1[key])
			}
			fmt.Println()
		}
	}

	if len(onlyIn2) > 0 {
		for _, key := range onlyIn2 {
			fmt.Printf("--- %s\t(%s)\n", key, stage1)
			fmt.Printf("+++ %s\t(%s)\n", key, stage2)
			fmt.Printf("@@ -0,0 +1 @@\n")
			if isSecretParameter(key) {
				fmt.Printf("+*****\n")
			} else {
				fmt.Printf("+%s\n", params2[key])
			}
			fmt.Println()
		}
	}

	if len(different) > 0 {
		for _, key := range different {
			if isSecretParameter(key) {
				fmt.Printf("--- %s\t(%s)\n", key, stage1)
				fmt.Printf("+++ %s\t(%s)\n", key, stage2)
				fmt.Printf("@@ -1 +1 @@\n")
				fmt.Printf("-*****\n")
				fmt.Printf("+*****\n")
				fmt.Println()
			} else {
				fmt.Printf("--- %s\t(%s)\n", key, stage1)
				fmt.Printf("+++ %s\t(%s)\n", key, stage2)
				fmt.Printf("@@ -1 +1 @@\n")
				fmt.Printf("-%s\n", params1[key])
				fmt.Printf("+%s\n", params2[key])
				fmt.Println()
			}
		}
	}
}

func displayMultiStageDiffWithTypes(stages []string, stageParamInfos map[string]map[string]aws.ParameterInfo, path string, keysOnly bool, showSecrets bool) {
	fmt.Printf("Multi-stage Parameter Store Comparison (path: %s)\n", path)
	fmt.Printf("Stages: %s\n\n", strings.Join(stages, ", "))

	// Collect all unique parameter keys (正規化されたキー)
	allKeys := make(map[string]bool)
	for key := range stageParamInfos {
		allKeys[key] = true
	}

	if len(allKeys) == 0 {
		fmt.Println("No parameters found.")
		return
	}

	// Sort keys
	var sortedKeys []string
	for key := range allKeys {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	if keysOnly {
		// Keys only mode - just show parameter names
		fmt.Printf("Parameters (path: %s)\n", path)
		fmt.Printf("Stages: %s\n\n", strings.Join(stages, ", "))

		for _, key := range sortedKeys {
			fmt.Println(key)
		}
	} else {
		// Display comparison table
		fmt.Printf("%-50s", "Parameter")
		for _, stage := range stages {
			fmt.Printf(" %-32s", stage)
		}
		fmt.Println()
		fmt.Println(strings.Repeat("-", 50+len(stages)*33))

		for _, key := range sortedKeys {
			// 各ステージのパラメータを取得（正規化されたキーから）
			paramsByStage := stageParamInfos[key]

			// 各ステージの表示値を取得
			displayValues := make([]string, len(stages))
			maxLines := 1 // 表示行数

			for i, stage := range stages {
				param, exists := paramsByStage[stage]

				if !exists {
					// パラメータが存在しないステージは赤で-を表示（削除）
					missingValue := "-"
					if shouldUseColor() {
						// 前のステージに存在する場合は削除（赤）
						if i > 0 {
							prevStage := stages[i-1]
							if _, prevExists := paramsByStage[prevStage]; prevExists {
								missingValue = colorizeTableValue(missingValue, "removed")
							}
						}
					}
					displayValues[i] = missingValue
					continue
				}

				displayValue := param.Value

				// JSON の場合の処理
				isJSON := isJSONValue(param.Value)
				if isJSON {
					// JSON値は常にJSONPath形式で展開表示
					displayValue = flattenJSONForDisplay(param.Value)
				} else if len(param.Value) > 1000 {
					// JSON でない場合はサマリー表示
					summary := FormatJSONSummary(param.Value)
					if len(summary) < len(param.Value) {
						displayValue = summary
					}

					// 改行文字をスペースに置換してテーブルの崩れ防止
					displayValue = strings.ReplaceAll(displayValue, "\n", " ")
					displayValue = strings.ReplaceAll(displayValue, "\r", "")
					displayValue = strings.ReplaceAll(displayValue, "\t", " ")

					// それでも長い場合は truncate
					if len(displayValue) > 29 {
						displayValue = displayValue[:29] + "..."
					}
				} else {
					// 短い値の場合は改行文字をスペースに置換
					displayValue = strings.ReplaceAll(displayValue, "\n", " ")
					displayValue = strings.ReplaceAll(displayValue, "\r", "")
					displayValue = strings.ReplaceAll(displayValue, "\t", " ")
				}

				// 前のステージとの比較で色分けを判定
				if shouldUseColor() && i > 0 {
					prevStage := stages[i-1]
					if prevParam, prevExists := paramsByStage[prevStage]; prevExists {
						// 前のステージに存在する場合
						if param.Type == "SecureString" && !showSecrets {
							// SecureStringの場合、mask状態では差分なしとして扱う
							// showSecretsの時だけ差分判定を行う
							displayValue = "***masked secret***"
						} else {
							// 前のステージの値と比較
							if param.Value != prevParam.Value {
								// JSON値の場合は属性ごとに色分け、それ以外は全体を色分け
								if isJSON {
									displayValue = colorizeJSONAttributeDiff(param.Value, prevParam.Value, displayValue)
								} else {
									// 値が異なる場合は黄色（変更）
									displayValue = colorizeTableValue(displayValue, "changed")
								}
							}
						}
					} else {
						// 前のステージに存在しない場合は緑色（追加）
						if param.Type == "SecureString" && !showSecrets {
							// Mask状態では色分けなしで表示
							displayValue = "***masked secret***"
						} else {
							displayValue = colorizeTableValue(displayValue, "added")
						}
					}
				} else if shouldUseColor() && i == 0 && param.Type == "SecureString" && !showSecrets {
					// 最初のステージでSecureStringの場合はマスク値を表示
					displayValue = "***masked secret***"
				}

				displayValues[i] = displayValue

				// JSON値は常に複数行表示の準備
				if isJSON {
					lines := strings.Split(displayValue, " | ")
					if len(lines) > maxLines {
						maxLines = len(lines)
					}
				}
			}

			// 複数行表示の場合
			if maxLines > 1 {
				// 最初の行：正規化キー（{stage}プレースホルダー） + 各ステージの最初の行
				fmt.Printf("%-50s", key)
				for _, displayValue := range displayValues {
					parts := strings.Split(displayValue, " | ")
					if len(parts) > 0 {
						fmt.Printf(" %-32s", truncateValue(parts[0], 32))
					} else {
						fmt.Printf(" %-32s", truncateValue(displayValue, 32))
					}
				}
				fmt.Println()

				// 2行目以降：パディング + 各ステージの残りの行
				for lineIdx := 1; lineIdx < maxLines; lineIdx++ {
					fmt.Printf("%-50s", "") // パラメータ名の部分は空欄
					for _, displayValue := range displayValues {
						parts := strings.Split(displayValue, " | ")
						if lineIdx < len(parts) {
							fmt.Printf(" %-32s", truncateValue(parts[lineIdx], 32))
						} else {
							fmt.Printf(" %-32s", "")
						}
					}
					fmt.Println()
				}
			} else {
				// 通常の1行表示
				fmt.Printf("%-50s", key)
				for _, displayValue := range displayValues {
					fmt.Printf(" %-32s", truncateValue(displayValue, 32))
				}
				fmt.Println()
			}
		}
	}
}

func displayMultiStageDiff(stages []string, stageParams map[string]map[string]string, path string) {
	fmt.Printf("Multi-stage Parameter Store Comparison (path: %s)\n", path)
	fmt.Printf("Stages: %s\n\n", strings.Join(stages, ", "))

	// Collect all unique parameter keys
	allKeys := make(map[string]bool)
	for _, params := range stageParams {
		for key := range params {
			allKeys[key] = true
		}
	}

	if len(allKeys) == 0 {
		fmt.Println("No parameters found.")
		return
	}

	// Sort keys
	var sortedKeys []string
	for key := range allKeys {
		sortedKeys = append(sortedKeys, key)
	}
	sort.Strings(sortedKeys)

	// Display comparison table
	fmt.Printf("%-50s", "Parameter")
	for _, stage := range stages {
		fmt.Printf(" %-15s", stage)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 50+len(stages)*16))

	for _, key := range sortedKeys {
		fmt.Printf("%-50s", key)

		for _, stage := range stages {
			params := stageParams[stage]
			if value, exists := params[key]; exists {
				if isSecretParameter(key) {
					fmt.Printf(" %-15s", "[HIDDEN]")
				} else {
					truncatedValue := value
					if len(value) > 12 {
						truncatedValue = value[:12] + "..."
					}
					fmt.Printf(" %-15s", truncatedValue)
				}
			} else {
				fmt.Printf(" %-15s", "-")
			}
		}
		fmt.Println()
	}
}

// truncateValue: 文字列を指定の長さで truncate
func truncateValue(value string, maxLen int) string {
	if len(value) > maxLen {
		return value[:maxLen-3] + "..."
	}
	return value
}

// isJSONValue: 文字列がJSON形式かどうかを判定
func isJSONValue(value string) bool {
	if len(value) == 0 {
		return false
	}

	value = strings.TrimSpace(value)
	// JSONは {} または [] で始まる必要がある
	if !((strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}")) ||
		(strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]"))) {
		return false
	}

	// JSONのパース試行
	var obj interface{}
	err := json.Unmarshal([]byte(value), &obj)
	return err == nil
}

// flattenJSONForDisplay: JSON値をJSONPath形式でフラット化して表示用文字列に変換
// 例: {"key1":{"sub1":"value1"},"key2":"value2"}
//
//	-> "key1.sub1: value1 | key2: value2"
func flattenJSONForDisplay(jsonValue string) string {
	var obj interface{}
	err := json.Unmarshal([]byte(jsonValue), &obj)
	if err != nil {
		// JSONパースエラーの場合は元の値を返す（改行削除）
		cleaned := strings.ReplaceAll(jsonValue, "\n", " ")
		cleaned = strings.ReplaceAll(cleaned, "\r", "")
		cleaned = strings.ReplaceAll(cleaned, "\t", " ")
		return cleaned
	}

	flatMap := make(map[string]interface{})

	// オブジェクト型の場合
	if objMap, ok := obj.(map[string]interface{}); ok {
		flatMap = flattenJSONObject(objMap, "")
	} else if objArray, ok := obj.([]interface{}); ok {
		// 配列型の場合
		flatMap = flattenJSONArray(objArray, "")
	}

	// フラット化されたマップを文字列に変換
	return formatFlattenedJSON(flatMap)
}

// flattenJSONObject: JSONオブジェクトをフラット化
func flattenJSONObject(obj map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})

	if obj == nil {
		return result
	}

	for key, value := range obj {
		var fullPath string
		if prefix == "" {
			fullPath = key
		} else {
			fullPath = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			// ネストされたオブジェクト
			nested := flattenJSONObject(v, fullPath)
			for k, val := range nested {
				result[k] = val
			}
		case []interface{}:
			// 配列
			nested := flattenJSONArray(v, fullPath)
			for k, val := range nested {
				result[k] = val
			}
		default:
			// スカラー値
			result[fullPath] = value
		}
	}

	return result
}

// flattenJSONArray: JSON配列をフラット化
func flattenJSONArray(arr []interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})

	for i, value := range arr {
		fullPath := fmt.Sprintf("%s[%d]", prefix, i)

		switch v := value.(type) {
		case map[string]interface{}:
			// 配列内のオブジェクト
			nested := flattenJSONObject(v, fullPath)
			for k, val := range nested {
				result[k] = val
			}
		case []interface{}:
			// 配列内の配列
			nested := flattenJSONArray(v, fullPath)
			for k, val := range nested {
				result[k] = val
			}
		default:
			// スカラー値
			result[fullPath] = value
		}
	}

	return result
}

// formatFlattenedJSON: フラット化されたマップを表示用文字列に変換
func formatFlattenedJSON(flatMap map[string]interface{}) string {
	if len(flatMap) == 0 {
		return ""
	}

	// キーをソート
	var keys []string
	for k := range flatMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 各キーを "key: value" 形式で出力
	var parts []string
	for _, key := range keys {
		value := flatMap[key]
		valueStr := fmt.Sprintf("%v", value)

		// 値の型に応じた整形
		switch v := value.(type) {
		case string:
			valueStr = v
		case float64:
			// 整数の場合は小数点なしで表示
			if v == float64(int64(v)) {
				valueStr = fmt.Sprintf("%.0f", v)
			} else {
				valueStr = fmt.Sprintf("%g", v)
			}
		case bool:
			valueStr = fmt.Sprintf("%v", v)
		case nil:
			valueStr = "null"
		}

		// 値が長い場合は truncate
		if len(valueStr) > 20 {
			valueStr = valueStr[:20] + "..."
		}

		parts = append(parts, fmt.Sprintf("%s:%s", key, valueStr))
	}

	// "key1:value1 | key2:value2" の形式で結合
	result := strings.Join(parts, " | ")

	// 改行を削除してテーブル崩れを防止
	result = strings.ReplaceAll(result, "\n", " ")
	result = strings.ReplaceAll(result, "\r", "")
	result = strings.ReplaceAll(result, "\t", " ")

	return result
}

// colorizeJSONAttributeDiff: JSON値の属性ごとに差分判定して色分け
// 異なる属性だけ黄色で表示
func colorizeJSONAttributeDiff(currentValue, prevValue, flattenedDisplay string) string {
	if !shouldUseColor() {
		return flattenedDisplay
	}

	// 両方のJSON値をパースしてフラット化
	var currObj, prevObj interface{}
	err1 := json.Unmarshal([]byte(currentValue), &currObj)
	err2 := json.Unmarshal([]byte(prevValue), &prevObj)

	if err1 != nil || err2 != nil {
		// パースエラーの場合は全体を黄色で表示
		return colorizeTableValue(flattenedDisplay, "changed")
	}

	// フラット化
	var currFlat, prevFlat map[string]interface{}

	if currMap, ok := currObj.(map[string]interface{}); ok {
		currFlat = flattenJSONObject(currMap, "")
	}
	if prevMap, ok := prevObj.(map[string]interface{}); ok {
		prevFlat = flattenJSONObject(prevMap, "")
	}

	if currFlat == nil || prevFlat == nil {
		// パース失敗時は全体を黄色で表示
		return colorizeTableValue(flattenedDisplay, "changed")
	}

	// 属性ごとに比較して、異なる属性を抽出
	differentKeys := make(map[string]bool)

	// currに存在するキーで比較
	for key, currVal := range currFlat {
		prevVal, exists := prevFlat[key]
		if !exists || fmt.Sprintf("%v", currVal) != fmt.Sprintf("%v", prevVal) {
			differentKeys[key] = true
		}
	}

	// prevに存在するがcurrに存在しないキー
	for key := range prevFlat {
		if _, exists := currFlat[key]; !exists {
			differentKeys[key] = true
		}
	}

	// flattenedDisplay を " | " で分割して、属性ごとに色分け
	parts := strings.Split(flattenedDisplay, " | ")
	var coloredParts []string

	for _, part := range parts {
		// "key:value" の形式から key を抽出
		if idx := strings.Index(part, ":"); idx > 0 {
			key := part[:idx]
			if differentKeys[key] {
				// 異なる属性は黄色で色付け
				coloredParts = append(coloredParts, colorizeTableValue(part, "changed"))
			} else {
				// 同じ属性はそのまま
				coloredParts = append(coloredParts, part)
			}
		} else {
			coloredParts = append(coloredParts, part)
		}
	}

	return strings.Join(coloredParts, " | ")
}
