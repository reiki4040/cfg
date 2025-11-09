package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage cfgctl configuration",
	Long: `Manage cfgctl configuration settings including default regions, stages, and KMS keys.

Configuration is stored in ~/.cfgctl.config`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE:  runConfigShowCommand,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> [value]",
	Short: "Set configuration value",
	Long: `Set configuration values.

Available keys:
  default_region     - Default AWS region
  default_stage      - Default stage
  path_prefix        - Default Parameter Store path prefix
  kms_key            - KMS key for current region/stage

Examples:
  cfgctl config set default_region ap-northeast-1
  cfgctl config set default_stage prod
  cfgctl config set path_prefix /myapp
  cfgctl config set kms_key alias/myapp-key --region=ap-northeast-1 --stage=prod
  cfgctl config set kms_key --interactive --region=ap-northeast-1 --stage=prod`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runConfigSetCommand,
}

var configUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "Unset configuration value",
	Long: `Unset configuration values.

Examples:
  cfgctl config unset kms_key --region=ap-northeast-1 --stage=prod`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigUnsetCommand,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration interactively",
	Long: `Initialize cfgctl configuration through interactive prompts.
This will guide you through setting up default region, stage, and KMS keys.`,
	RunE: runConfigInitCommand,
}

var (
	configInteractive bool
)

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configUnsetCmd)
	configCmd.AddCommand(configInitCmd)

	configSetCmd.Flags().BoolVar(&configInteractive, "interactive", false, "Interactive mode for value input")
}

func runConfigShowCommand(cmd *cobra.Command, args []string) error {
	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("Configuration (~/.cfgctl.config):\n\n")
	fmt.Printf("Default Region:  %s\n", config.DefaultRegion)
	fmt.Printf("Default Stage:   %s\n", config.DefaultStage)
	fmt.Printf("Path Prefix:     %s\n", config.PathPrefix)
	fmt.Println()

	if len(config.KMSKeys) > 0 {
		fmt.Println("KMS Keys by Region/Stage:")
		for region, regionKeys := range config.KMSKeys {
			fmt.Printf("  %s:\n", region)
			for stage, kmsKey := range regionKeys.Stages {
				fmt.Printf("    %s: %s\n", stage, kmsKey)
			}
		}
	} else {
		fmt.Println("No KMS keys configured.")
	}

	return nil
}

func runConfigSetCommand(cmd *cobra.Command, args []string) error {
	key := args[0]
	var value string

	// Handle interactive mode or value from args
	if configInteractive || len(args) < 2 {
		var err error
		value, err = promptForConfigValue(key)
		if err != nil {
			return fmt.Errorf("failed to get value interactively: %w", err)
		}
	} else {
		value = args[1]
	}

	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	switch key {
	case "default_region":
		config.DefaultRegion = value
		fmt.Printf("Set default region to: %s\n", value)
	case "default_stage":
		config.DefaultStage = value
		fmt.Printf("Set default stage to: %s\n", value)
	case "path_prefix":
		if !strings.HasPrefix(value, "/") {
			return fmt.Errorf("path prefix must start with '/'. Example: /app")
		}
		config.PathPrefix = value
		fmt.Printf("Set path prefix to: %s\n", value)
	case "kms_key":
		stageResolver := createStageResolver()
		err := setKMSKeyForStageAndRegion(stageResolver.GetStage(), awsRegion, value)
		if err != nil {
			return fmt.Errorf("failed to set KMS key: %w", err)
		}
		fmt.Printf("Set KMS key for %s/%s to: %s\n", awsRegion, stageResolver.GetStage(), value)
		return nil
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	return saveConfig(config)
}

func runConfigInitCommand(cmd *cobra.Command, args []string) error {
	fmt.Println("Initializing cfgctl configuration...")
	fmt.Println()

	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load existing config: %w", err)
	}

	// Set default region
	fmt.Printf("Default AWS region [%s]: ", config.DefaultRegion)
	if region, err := promptForInput(); err == nil && region != "" {
		config.DefaultRegion = region
	}

	// Set default stage
	fmt.Printf("Default stage [%s]: ", config.DefaultStage)
	if stage, err := promptForInput(); err == nil && stage != "" {
		config.DefaultStage = stage
	}

	// Set path prefix
	fmt.Printf("Parameter Store path prefix [%s]: ", config.PathPrefix)
	if prefix, err := promptForInput(); err == nil && prefix != "" {
		if !strings.HasPrefix(prefix, "/") {
			fmt.Printf("Error: path prefix must start with '/'. Example: /app\n")
			return fmt.Errorf("invalid path prefix format")
		}
		config.PathPrefix = prefix
	}

	// Ask about KMS key configuration
	fmt.Print("Configure KMS keys for stages? (y/N): ")
	if response, err := promptForInput(); err == nil && (response == "y" || response == "yes") {
		err := configureKMSKeysInteractively(config)
		if err != nil {
			return fmt.Errorf("failed to configure KMS keys: %w", err)
		}
	}

	// Save configuration
	if err := saveConfig(config); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Println()
	fmt.Printf("Configuration saved to: %s\n", getConfigFilePathString())
	return nil
}

func configureKMSKeysInteractively(config *CfgctlConfig) error {
	regions := []string{"us-east-1", "us-west-2", "ap-northeast-1", "eu-west-1"}
	stages := []string{"dev", "stg", "prod"}

	fmt.Println()
	fmt.Println("Configure KMS keys by region and stage:")
	fmt.Println("(Leave blank to use default aws/ssm key)")
	fmt.Println()

	for _, region := range regions {
		fmt.Printf("Configure KMS keys for region %s? (y/N): ", region)
		response, err := promptForInput()
		if err != nil || (response != "y" && response != "yes") {
			continue
		}

		for _, stage := range stages {
			currentKey := getKMSKeyForStageAndRegion(stage, region)
			prompt := fmt.Sprintf("  KMS key for %s/%s", region, stage)
			if currentKey != "" {
				prompt += fmt.Sprintf(" [%s]", currentKey)
			}
			prompt += ": "

			fmt.Print(prompt)
			kmsKey, err := promptForInput()
			if err != nil {
				continue
			}

			if kmsKey != "" {
				err := setKMSKeyForStageAndRegion(stage, region, kmsKey)
				if err != nil {
					fmt.Printf("    Error setting KMS key: %v\n", err)
				} else {
					fmt.Printf("    Set KMS key for %s/%s\n", region, stage)
				}
			}
		}
		fmt.Println()
	}

	return nil
}

func promptForConfigValue(key string) (string, error) {
	switch key {
	case "default_region":
		fmt.Print("Enter default AWS region: ")
	case "default_stage":
		fmt.Print("Enter default stage: ")
	case "path_prefix":
		fmt.Print("Enter Parameter Store path prefix (must start with /): ")
	case "kms_key":
		stageResolver := createStageResolver()
		fmt.Printf("Enter KMS key for %s/%s: ", awsRegion, stageResolver.GetStage())
	default:
		fmt.Printf("Enter value for %s: ", key)
	}

	return promptForInput()
}

func promptForInput() (string, error) {
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func getConfigFilePathString() string {
	path, _ := getConfigFilePath()
	return path
}

func runConfigUnsetCommand(cmd *cobra.Command, args []string) error {
	key := args[0]

	config, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	switch key {
	case "default_region":
		config.DefaultRegion = ""
		fmt.Println("Unset default region")
	case "default_stage":
		config.DefaultStage = ""
		fmt.Println("Unset default stage")
	case "path_prefix":
		config.PathPrefix = ""
		fmt.Println("Unset path prefix")
	case "kms_key":
		stageResolver := createStageResolver()
		if regionKeys, exists := config.KMSKeys[awsRegion]; exists {
			delete(regionKeys.Stages, stageResolver.GetStage())
			config.KMSKeys[awsRegion] = regionKeys
			fmt.Printf("Unset KMS key for %s/%s\n", awsRegion, stageResolver.GetStage())
		} else {
			fmt.Printf("No KMS key configured for %s/%s\n", awsRegion, stageResolver.GetStage())
		}
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	return saveConfig(config)
}
