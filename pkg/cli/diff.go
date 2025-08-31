package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourusername/cfg/pkg/aws"
	"github.com/yourusername/cfg/pkg/cfg"
)

var diffCmd = &cobra.Command{
	Use:   "diff <subcommand>",
	Short: "Compare configurations",
	Long: `Compare configurations between Parameter Store, files, or stages.

Available subcommands:
  ps     - Compare Parameter Store between stages
  config - Compare config file with Parameter Store
  file   - Compare two config files`,
}

var diffPsCmd = &cobra.Command{
	Use:   "ps",
	Short: "Compare Parameter Store between stages",
	Long: `Compare Parameter Store parameters between stages.

Examples:
  cfgctl diff ps --stage=dev --compare-stage=stg --path=/app/
  cfgctl diff ps --stages=dev,stg,prod --path=/app/
  cfgctl diff ps --stage=dev --compare-stage=stg --keys-only
  cfgctl diff ps --stage=dev --compare-stage=stg --show-secrets
  cfgctl diff ps --stage=dev --compare-stage=stg --compare-profile=prod-profile`,
	RunE: runDiffPsCommand,
}

var diffConfigCmd = &cobra.Command{
	Use:   "config <config-file>",
	Short: "Compare config file with Parameter Store",
	Long: `Compare configuration file with current Parameter Store values.

Examples:
  cfgctl diff config config.yaml --stage=prod`,
	Args: cobra.ExactArgs(1),
	RunE: runDiffConfigCommand,
}

var diffFileCmd = &cobra.Command{
	Use:   "file <file1> <file2>",
	Short: "Compare two config files",
	Long: `Compare two configuration files.

Examples:
  cfgctl diff file config.dev.yaml config.prod.yaml`,
	Args: cobra.ExactArgs(2),
	RunE: runDiffFileCommand,
}

var (
	diffCompareStage string
	diffStages       string
	diffPath         string
	diffKeysOnly     bool
	diffShowSecrets  bool
	diffCompareProfile string
)

func init() {
	rootCmd.AddCommand(diffCmd)
	diffCmd.AddCommand(diffPsCmd)
	diffCmd.AddCommand(diffConfigCmd)
	diffCmd.AddCommand(diffFileCmd)

	diffPsCmd.Flags().StringVar(&diffCompareStage, "compare-stage", "", "Stage to compare with")
	diffPsCmd.Flags().StringVar(&diffStages, "stages", "", "Comma-separated list of stages to compare")
	diffPsCmd.Flags().StringVar(&diffPath, "path", "", "Parameter path to compare (default: path prefix)")
	diffPsCmd.Flags().BoolVar(&diffKeysOnly, "keys-only", false, "Show only parameter names without values")
	diffPsCmd.Flags().BoolVar(&diffShowSecrets, "show-secrets", false, "Show SecureString parameter values (DANGEROUS)")
	diffPsCmd.Flags().StringVar(&diffCompareProfile, "compare-profile", "", "AWS profile for comparison target")
}

func runDiffPsCommand(cmd *cobra.Command, args []string) error {
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

	resolvedPath1 := stageResolver1.ResolvePath(comparePath)
	resolvedPath2 := stageResolver2.ResolvePath(comparePath)

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

	// Create AWS client
	awsClient, err := createAWSClient()
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	ctx := context.Background()
	stageParamInfos := make(map[string][]aws.ParameterInfo)

	// Get parameters for each stage
	for _, s := range stages {
		stageResolver := cfg.NewStageResolver(strings.TrimSpace(s))
		resolvedPath := stageResolver.ResolvePath(comparePath)
		
		// Decrypt parameters if we want to show secrets
		shouldDecrypt := diffShowSecrets
		paramInfos, err := awsClient.GetParameterInfosByPath(ctx, resolvedPath, true, shouldDecrypt)
		if err != nil {
			paramInfos = []aws.ParameterInfo{}
		}
		stageParamInfos[stageResolver.GetStage()] = paramInfos
	}

	// Display multi-stage comparison
	displayMultiStageDiffWithTypes(stages, stageParamInfos, comparePath, diffKeysOnly, diffShowSecrets)

	return nil
}

func runDiffConfigCommand(cmd *cobra.Command, args []string) error {
	configFile := args[0]

	fmt.Printf("Config vs Parameter Store comparison not fully implemented yet.\n")
	fmt.Printf("Config file: %s, Stage: %s\n", configFile, stage)
	
	return nil
}

func runDiffFileCommand(cmd *cobra.Command, args []string) error {
	file1 := args[0]
	file2 := args[1]

	fmt.Printf("File comparison not fully implemented yet.\n")
	fmt.Printf("Comparing: %s vs %s\n", file1, file2)
	
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

	var onlyIn1, onlyIn2, different []string

	for key := range allKeys {
		param1, exists1 := params1[key]
		param2, exists2 := params2[key]

		if exists1 && !exists2 {
			onlyIn1 = append(onlyIn1, key)
		} else if !exists1 && exists2 {
			onlyIn2 = append(onlyIn2, key)
		} else if exists1 && exists2 && param1.Value != param2.Value {
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
			param := params1[key]
			if keysOnly {
				fmt.Printf("- %s\n", key)
			} else {
				// Show actual resolved parameter paths for each stage
				param1Path := param.Name
				param2Path := strings.Replace(key, "/{stage}/", "/"+stage2+"/", 1)
				fmt.Printf("--- %s (%s)\n", param1Path, stage1)
				fmt.Printf("+++ %s (%s)\n", param2Path, stage2)
				fmt.Printf("@@ -1 +0,0 @@\n")
				if param.Type == "SecureString" && !showSecrets {
					fmt.Printf("-***masked secret***\n")
				} else {
					fmt.Printf("-%s\n", param.Value)
				}
				fmt.Println()
			}
		}
	}

	if len(onlyIn2) > 0 {
		for _, key := range onlyIn2 {
			param := params2[key]
			if keysOnly {
				fmt.Printf("+ %s\n", key)
			} else {
				// Show actual resolved parameter paths for each stage
				param1Path := strings.Replace(key, "/{stage}/", "/"+stage1+"/", 1)
				param2Path := param.Name
				fmt.Printf("--- %s (%s)\n", param1Path, stage1)
				fmt.Printf("+++ %s (%s)\n", param2Path, stage2)
				fmt.Printf("@@ -0,0 +1 @@\n")
				if param.Type == "SecureString" && !showSecrets {
					fmt.Printf("+***masked secret***\n")
				} else {
					fmt.Printf("+%s\n", param.Value)
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
				fmt.Printf("~ %s\n", key)
			} else {
				// Show actual resolved parameter paths for each stage
				param1Path := param1.Name
				param2Path := param2.Name
				fmt.Printf("--- %s (%s)\n", param1Path, stage1)
				fmt.Printf("+++ %s (%s)\n", param2Path, stage2)
				fmt.Printf("@@ -1 +1 @@\n")
				if (param1.Type == "SecureString" || param2.Type == "SecureString") && !showSecrets {
					fmt.Printf("-***masked secret***\n")
					fmt.Printf("+***masked secret***\n")
				} else {
					fmt.Printf("-%s\n", param1.Value)
					fmt.Printf("+%s\n", param2.Value)
				}
				fmt.Println()
			}
		}
	}
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

func displayMultiStageDiffWithTypes(stages []string, stageParamInfos map[string][]aws.ParameterInfo, path string, keysOnly bool, showSecrets bool) {
	// Convert to map format for easier lookup
	stageParams := make(map[string]map[string]aws.ParameterInfo)
	for stage, paramInfos := range stageParamInfos {
		params := make(map[string]aws.ParameterInfo)
		for _, param := range paramInfos {
			params[param.Name] = param
		}
		stageParams[stage] = params
	}

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
			fmt.Printf("%-50s", key)
			
			for _, stage := range stages {
				params := stageParams[stage]
				if param, exists := params[key]; exists {
					if param.Type == "SecureString" && !showSecrets {
						fmt.Printf(" %-32s", "***masked secret***")
					} else {
						truncatedValue := param.Value
						if len(param.Value) > 29 {
							truncatedValue = param.Value[:29] + "..."
						}
						fmt.Printf(" %-32s", truncatedValue)
					}
				} else {
					fmt.Printf(" %-32s", "-")
				}
			}
			fmt.Println()
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