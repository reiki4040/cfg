package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <parameter-path>",
	Short: "Delete a parameter from Parameter Store",
	Long: `Delete a parameter from AWS Parameter Store.
The parameter path supports {stage} placeholder which will be replaced with the current stage.

Examples:
  cfgctl delete /app/{stage}/db/password --stage=prod
  cfgctl delete /app/prod/api/key --force`,
	Args: cobra.ExactArgs(1),
	RunE: runDeleteCommand,
}

var (
	deleteForce bool
)

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().BoolVar(&deleteForce, "force", false, "Skip confirmation prompt")
}

func runDeleteCommand(cmd *cobra.Command, args []string) error {
	parameterPath := args[0]

	// Create AWS client
	awsClient, err := createAWSClient()
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Create stage resolver and resolve path
	stageResolver := createStageResolver()
	resolvedPath := resolveParameterPath(parameterPath, stageResolver)

	// Confirmation prompt unless force is used
	if !deleteForce {
		confirmed, err := confirmDeletion(resolvedPath)
		if err != nil {
			return fmt.Errorf("failed to get confirmation: %w", err)
		}
		if !confirmed {
			fmt.Println("Operation cancelled.")
			return nil
		}
	}

	// Delete parameter
	ctx := context.Background()
	err = awsClient.DeleteParameter(ctx, resolvedPath)
	if err != nil {
		return fmt.Errorf("failed to delete parameter %s: %w", resolvedPath, err)
	}

	fmt.Printf("Successfully deleted parameter: %s\n", resolvedPath)
	return nil
}

func confirmDeletion(parameterPath string) (bool, error) {
	fmt.Printf("Are you sure you want to delete parameter '%s'? (y/N): ", parameterPath)
	
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}