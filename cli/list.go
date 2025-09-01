package cli

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [parameter-path]",
	Short: "List parameters from Parameter Store",
	Long: `List parameters from AWS Parameter Store by path.
If no path is specified, lists parameters from the configured path prefix.
The parameter path supports {stage} placeholder which will be replaced with the current stage.

Examples:
  cfgctl list --stage=prod
  cfgctl list {stage}/db --stage=dev  
  cfgctl list /app/prod/ --recursive --values`,
	Args: cobra.MaximumNArgs(1),
	RunE: runListCommand,
}

var (
	listRecursive   bool
	listDecrypt     bool
	listValues      bool
	listShowSecrets bool
)

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&listRecursive, "recursive", true, "List parameters recursively")
	listCmd.Flags().BoolVar(&listDecrypt, "decrypt", false, "Decrypt SecureString parameters")
	listCmd.Flags().BoolVar(&listValues, "values", false, "Show parameter values (use with caution for secrets)")
	listCmd.Flags().BoolVar(&listShowSecrets, "show-secrets", false, "Show secret parameter values (DANGEROUS - use with extreme caution)")
}

func runListCommand(cmd *cobra.Command, args []string) error {
	// Determine list path
	var parameterPath string
	if len(args) > 0 {
		parameterPath = args[0]
	} else {
		// Use path prefix from config directly (already resolved)
		pathPrefix := getPathPrefix()
		stageResolver := createStageResolver()
		parameterPath = stageResolver.ResolvePath(pathPrefix)
	}

	// Create AWS client
	awsClient, err := createAWSClient()
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	// Create stage resolver and resolve path (skip prefix application for direct paths)
	stageResolver := createStageResolver()
	var resolvedPath string
	if len(args) > 0 {
		// User provided path - apply normal resolution
		resolvedPath = resolveParameterPath(parameterPath, stageResolver)
	} else {
		// Default path - already resolved above
		resolvedPath = parameterPath
	}

	// Ensure path ends with / for consistency
	if !strings.HasSuffix(resolvedPath, "/") {
		resolvedPath += "/"
	}

	// Get parameters by path with type information
	ctx := context.Background()
	// Always decrypt if we want to show secret values
	shouldDecrypt := listDecrypt || (listValues && listShowSecrets)
	parameterInfos, err := awsClient.GetParameterInfosByPath(ctx, resolvedPath, listRecursive, shouldDecrypt)
	if err != nil {
		return fmt.Errorf("failed to list parameters for path %s: %w", resolvedPath, err)
	}

	if len(parameterInfos) == 0 {
		fmt.Printf("No parameters found for path: %s\n", resolvedPath)
		return nil
	}

	// Sort parameter names for consistent output
	sort.Slice(parameterInfos, func(i, j int) bool {
		return parameterInfos[i].Name < parameterInfos[j].Name
	})

	// Display results
	fmt.Printf("Parameters for path: %s\n", resolvedPath)
	fmt.Println(strings.Repeat("-", 50))

	for _, param := range parameterInfos {
		if listValues {
			// Show values based on parameter type
			if param.Type == "SecureString" && !listShowSecrets {
				fmt.Printf("%s = ***masked secret*** (SecureString)\n", param.Name)
			} else {
				fmt.Printf("%s = %s (%s)\n", param.Name, param.Value, param.Type)
			}
		} else {
			fmt.Printf("%s (%s)\n", param.Name, param.Type)
		}
	}

	fmt.Printf("\nTotal parameters: %d\n", len(parameterInfos))
	return nil
}