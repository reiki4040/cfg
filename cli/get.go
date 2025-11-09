package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <parameter-path> [--jsonpath <path>]",
	Short: "Get a parameter value from Parameter Store",
	Long: `Get a parameter value from AWS Parameter Store.
The parameter path supports {stage} placeholder which will be replaced with the current stage.
Use --jsonpath to extract a specific key from JSON parameters (e.g., --jsonpath=database.host).

Examples:
  cfgctl get /app/{stage}/db/password --stage=prod
  cfgctl get /app/prod/api/key
  cfgctl get /app/prod/config --jsonpath=database.host
  cfgctl get /app/prod/config --jsonpath=server.port --format=json`,
	Args: cobra.ExactArgs(1),
	RunE: runGetCommand,
}

var (
	getDecrypt  bool
	getJsonPath string
	getFormat   string
)

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().BoolVar(&getDecrypt, "decrypt", true, "Decrypt SecureString parameters")
	getCmd.Flags().StringVar(&getJsonPath, "jsonpath", "", "JSONPath to extract specific key from JSON parameters")
	getCmd.Flags().StringVar(&getFormat, "format", "text", "Output format: text, json (default: text)")
}

func runGetCommand(cmd *cobra.Command, args []string) error {
	parameterPath := args[0]

	// Create AWS client
	awsClient, err := createAWSClient()
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Create stage resolver and resolve path
	stageResolver := createStageResolver()
	resolvedPath := resolveParameterPath(parameterPath, stageResolver)

	// Get parameter value
	ctx := context.Background()
	value, err := awsClient.GetParameter(ctx, resolvedPath, getDecrypt)
	if err != nil {
		return fmt.Errorf("failed to get parameter %s: %w", resolvedPath, err)
	}

	// If JSONPath is specified, extract from JSON
	if getJsonPath != "" {
		value, err = FilterJSONByPath(value, getJsonPath, resolvedPath)
		if err != nil {
			return fmt.Errorf("failed to extract JSONPath %s: %w", getJsonPath, err)
		}
	} else if getFormat == "json" {
		// If format is JSON but no JSONPath, try to format as pretty JSON
		formatted, err := FormatJSONOutput(value)
		if err == nil {
			value = formatted
		}
	}

	fmt.Println(value)
	return nil
}
