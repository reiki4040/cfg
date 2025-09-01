package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <parameter-path>",
	Short: "Get a parameter value from Parameter Store",
	Long: `Get a parameter value from AWS Parameter Store.
The parameter path supports {stage} placeholder which will be replaced with the current stage.

Examples:
  cfgctl get /app/{stage}/db/password --stage=prod
  cfgctl get /app/prod/api/key`,
	Args: cobra.ExactArgs(1),
	RunE: runGetCommand,
}

var (
	getDecrypt bool
)

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.Flags().BoolVar(&getDecrypt, "decrypt", true, "Decrypt SecureString parameters")
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

	fmt.Println(value)
	return nil
}