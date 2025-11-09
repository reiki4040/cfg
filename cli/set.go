package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/reiki4040/cfg"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var setCmd = &cobra.Command{
	Use:   "set <parameter-path> [value]",
	Short: "Set a parameter value in Parameter Store",
	Long: `Set a parameter value in AWS Parameter Store.
The parameter path supports {stage} placeholder which will be replaced with the current stage.
JSON values can be set using --json flag, either from file or as inline JSON string.
JSON values can be stored as either String (default) or SecureString (--SS) for encrypted storage.
Note: JSON values cannot be stored as StringList. Use --S (String) or --SS (SecureString) instead.

Use --dry-run flag to preview changes without actually modifying Parameter Store.

Examples:
  cfgctl set /app/{stage}/db/password --stage=prod --SS
  cfgctl set /app/prod/api/timeout "30" -S --no-interactive
  cfgctl set /app/{stage}/db/password --SS
  cfgctl set /app/prod/config --json='{"host":"localhost","port":5432}'
  cfgctl set /app/prod/config --json='{"host":"localhost","port":5432}' --SS
  cfgctl set /app/prod/config --json-file=config.json
  cfgctl set /app/prod/config --json-file=config.json --SS
  cfgctl set /app/prod/config --json='{"host":"newhost"}' --dry-run`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runSetCommand,
}

var (
	setType         string
	setOverwrite    bool
	setInteractive  bool
	setKMSKeyID     string
	setTypeString   bool
	setTypeSecure   bool
	setTypeList     bool
	setJsonValue    string
	setJsonFile     string
	setJsonValidate bool
	setDryRun       bool
	setColorFlag    string
)

func init() {
	rootCmd.AddCommand(setCmd)
	setCmd.Flags().StringVar(&setType, "type", "String", "Parameter type (String, SecureString, StringList)")
	setCmd.Flags().BoolVarP(&setTypeString, "string", "S", false, "Set parameter type to String")
	setCmd.Flags().BoolVar(&setTypeSecure, "SS", false, "Set parameter type to SecureString")
	setCmd.Flags().BoolVar(&setTypeList, "SL", false, "Set parameter type to StringList")
	setCmd.Flags().BoolVar(&setOverwrite, "overwrite", false, "Overwrite existing parameter without confirmation")
	setCmd.Flags().BoolVar(&setInteractive, "no-interactive", false, "Disable interactive mode (default: interactive)")
	setCmd.Flags().StringVar(&setKMSKeyID, "kms-key", "", "KMS key ID for SecureString parameters (default: alias/aws/ssm)")
	setCmd.Flags().StringVar(&setJsonValue, "json", "", "Set value as JSON string (inline JSON)")
	setCmd.Flags().StringVar(&setJsonFile, "json-file", "", "Set value as JSON from file")
	setCmd.Flags().BoolVar(&setJsonValidate, "json-validate", true, "Validate JSON format (default: true)")
	setCmd.Flags().BoolVar(&setDryRun, "dry-run", false, "Preview changes without actually setting the parameter (show diff and exit)")
	setCmd.Flags().StringVar(&setColorFlag, "color", "auto", "Color output mode: auto (default), always, never")

	// Mark flags as mutually exclusive
	setCmd.MarkFlagsMutuallyExclusive("type", "string", "SS", "SL")
}

func runSetCommand(cmd *cobra.Command, args []string) error {
	// Initialize color mode
	setColorMode(setColorFlag)

	parameterPath := args[0]
	var value string

	// Parse JSONPath from parameter path (e.g., "/config:database.host")
	resolvedParamPath, jsonPath, err := parseJsonPath(parameterPath)
	if err != nil {
		return fmt.Errorf("failed to parse parameter path: %w", err)
	}

	// Create stage resolver for path resolution
	stageResolver := createStageResolver()
	resolvedPath := resolveParameterPath(resolvedParamPath, stageResolver)

	// Handle JSONPath-based attribute update
	if jsonPath != "" {
		// JSONPath update flow
		if len(args) < 2 {
			return fmt.Errorf("value is required for JSONPath attribute update")
		}
		newValue := args[1]

		// Create AWS client
		awsClient, err := createAWSClient()
		if err != nil {
			return fmt.Errorf("failed to create AWS client: %w", err)
		}

		ctx := context.Background()

		// Retrieve existing parameter
		existingValue, err := awsClient.GetParameter(ctx, resolvedPath, true)
		if err != nil {
			return fmt.Errorf("failed to retrieve parameter %s: %w", resolvedPath, err)
		}

		// Update JSON attribute
		updatedValue, err := cfg.UpdateJsonAttribute(existingValue, jsonPath, newValue)
		if err != nil {
			return fmt.Errorf("failed to update JSON attribute: %w", err)
		}

		// Show diff
		fmt.Printf("Parameter %s - JSONPath update: %s\n", resolvedPath, jsonPath)
		if err := showJsonAttributeDiff(existingValue, updatedValue, jsonPath); err != nil {
			return fmt.Errorf("failed to show diff: %w", err)
		}

		// If dry-run mode, exit here without asking for confirmation or making actual changes
		if setDryRun {
			fmt.Println("\n[DRY RUN MODE] - No actual changes were made to Parameter Store")
			return nil
		}

		// Ask for confirmation unless --overwrite flag is used
		if !setOverwrite {
			confirmed, err := confirmOverwrite(resolvedPath)
			if err != nil {
				return fmt.Errorf("failed to get confirmation: %w", err)
			}
			if !confirmed {
				fmt.Println("Operation cancelled.")
				return nil
			}
		}

		// Determine KMS key and parameter type
		paramType := "String"
		if setTypeSecure {
			paramType = "SecureString"
		} else if setTypeString {
			paramType = "String"
		}

		kmsKey := setKMSKeyID
		if kmsKey == "" && paramType == "SecureString" {
			kmsKey = getKMSKeyForStageAndRegion(stageResolver.GetStage(), awsRegion)
		}

		// Update parameter
		if kmsKey != "" && paramType == "SecureString" {
			err = awsClient.PutParameterWithKey(ctx, resolvedPath, updatedValue, paramType, true, kmsKey)
		} else {
			err = awsClient.PutParameter(ctx, resolvedPath, updatedValue, paramType, true)
		}
		if err != nil {
			return fmt.Errorf("failed to update parameter %s: %w", resolvedPath, err)
		}

		fmt.Printf("Successfully updated JSON attribute in parameter: %s\n", resolvedPath)
		return nil
	}

	// Standard value update flow (non-JSONPath)
	// Handle JSON input first
	if setJsonValue != "" || setJsonFile != "" {
		// Validate that JSON is not being stored as StringList
		if setTypeList {
			return fmt.Errorf("JSON values cannot be stored as StringList. Use --S (String) or --SS (SecureString) instead.")
		}

		jsonValue, err := getJsonValue()
		if err != nil {
			return fmt.Errorf("failed to get JSON value: %w", err)
		}

		// Validate JSON if requested
		if setJsonValidate {
			if err := validateJsonString(jsonValue); err != nil {
				return fmt.Errorf("JSON validation failed: %w", err)
			}
		}

		value = jsonValue
		// Determine type for JSON values based on flags
		if setTypeSecure {
			setType = "SecureString"
		} else if setTypeString {
			setType = "String"
		} else {
			// Default to String if no type is explicitly specified
			setType = "String"
		}
	} else {
		// Determine parameter type from flags
		if setTypeString {
			setType = "String"
		} else if setTypeSecure {
			setType = "SecureString"
		} else if setTypeList {
			setType = "StringList"
		} else {
			// Normalize the type from --type flag
			normalizedType, err := normalizeParameterType(setType)
			if err != nil {
				return err
			}
			setType = normalizedType
		}

		// Handle interactive mode or value from args
		if !setInteractive && len(args) < 2 {
			// Interactive mode by default
			var err error
			value, err = promptForValue(resolvedPath, setType == "SecureString")
			if err != nil {
				return fmt.Errorf("failed to get value interactively: %w", err)
			}
		} else if len(args) >= 2 {
			value = args[1]
		} else {
			return fmt.Errorf("value is required when using --no-interactive mode")
		}
	}

	// Create AWS client
	awsClient, err := createAWSClient()
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}

	ctx := context.Background()

	// Check if parameter already exists and show diff
	existingValue, err := awsClient.GetParameter(ctx, resolvedPath, true)
	parameterExists := (err == nil)

	if parameterExists {
		fmt.Printf("Parameter %s already exists.\n", resolvedPath)

		// Show diff (hide values for secrets)
		if setType == "SecureString" {
			fmt.Println(colorizeHeaderLine("Current value: [HIDDEN]"))
			fmt.Println(colorizeHeaderLine("New value:     [HIDDEN]"))
		} else {
			// Try to format as JSON for better readability
			currentFormatted, err1 := formatJsonForDisplay(existingValue)
			newFormatted, err2 := formatJsonForDisplay(value)

			if err1 == nil && err2 == nil {
				// Both are valid JSON, show formatted with color
				fmt.Println(colorizeHeaderLine("Current value (JSON):"))
				for _, line := range strings.Split(currentFormatted, "\n") {
					fmt.Println(colorizeRemovalLine(fmt.Sprintf("- %s", line)))
				}
				fmt.Println()
				fmt.Println(colorizeHeaderLine("New value (JSON):"))
				for _, line := range strings.Split(newFormatted, "\n") {
					fmt.Println(colorizeAdditionLine(fmt.Sprintf("+ %s", line)))
				}
			} else {
				// Show as plain text with color
				if len(existingValue) > 200 {
					fmt.Printf("%s\n", colorizeRemovalLine(fmt.Sprintf("- Current value: %s...", existingValue[:200])))
				} else {
					fmt.Printf("%s\n", colorizeRemovalLine(fmt.Sprintf("- Current value: %s", existingValue)))
				}

				if len(value) > 200 {
					fmt.Printf("%s\n", colorizeAdditionLine(fmt.Sprintf("+ New value: %s...", value[:200])))
				} else {
					fmt.Printf("%s\n", colorizeAdditionLine(fmt.Sprintf("+ New value: %s", value)))
				}
			}
		}

		// If dry-run mode, exit here without asking for confirmation
		if setDryRun {
			fmt.Println("\n[DRY RUN MODE] - No actual changes were made to Parameter Store")
			return nil
		}

		// Ask for confirmation unless --overwrite flag is used
		if !setOverwrite {
			confirmed, err := confirmOverwrite(resolvedPath)
			if err != nil {
				return fmt.Errorf("failed to get confirmation: %w", err)
			}
			if !confirmed {
				fmt.Println("Operation cancelled.")
				return nil
			}
		}
	}

	// If dry-run mode, exit here without making actual changes
	if setDryRun {
		fmt.Println("\n[DRY RUN MODE] - No actual changes were made to Parameter Store")
		return nil
	}

	// Determine KMS key to use
	kmsKey := setKMSKeyID
	if kmsKey == "" && setType == "SecureString" {
		// Try to get KMS key from config file
		kmsKey = getKMSKeyForStageAndRegion(stageResolver.GetStage(), awsRegion)
	}

	// Set parameter value (always use overwrite=true for PutParameter API)
	if kmsKey != "" && setType == "SecureString" {
		err = awsClient.PutParameterWithKey(ctx, resolvedPath, value, setType, true, kmsKey)
	} else {
		err = awsClient.PutParameter(ctx, resolvedPath, value, setType, true)
	}
	if err != nil {
		return fmt.Errorf("failed to set parameter %s: %w", resolvedPath, err)
	}

	if existingValue != "" {
		fmt.Printf("Successfully updated parameter: %s\n", resolvedPath)
	} else {
		fmt.Printf("Successfully created parameter: %s\n", resolvedPath)
	}
	return nil
}

func promptForValue(parameterPath string, isSecret bool) (string, error) {
	fmt.Printf("Enter value for parameter %s: ", parameterPath)

	if isSecret {
		// Validate terminal state for secure input
		if !term.IsTerminal(int(syscall.Stdin)) {
			return "", fmt.Errorf("secure input requires an interactive terminal")
		}

		// Hide input for secure strings
		byteValue, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return "", fmt.Errorf("failed to read secure input: %w", err)
		}
		fmt.Println() // Print newline after hidden input

		// Validate minimum length for security
		value := string(byteValue)
		if len(value) == 0 {
			return "", fmt.Errorf("secure parameter value cannot be empty")
		}

		return value, nil
	} else {
		// Normal input for non-secure strings
		reader := bufio.NewReader(os.Stdin)
		value, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}
		return strings.TrimSpace(value), nil
	}
}

func confirmOverwrite(parameterPath string) (bool, error) {
	fmt.Printf("Are you sure you want to overwrite parameter '%s'? (y/N): ", parameterPath)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

func isValidParameterType(paramType string) bool {
	validTypes := []string{"string", "securestring", "stringlist"}
	lowerType := strings.ToLower(paramType)
	for _, validType := range validTypes {
		if lowerType == validType {
			return true
		}
	}
	return false
}

func normalizeParameterType(paramType string) (string, error) {
	lowerType := strings.ToLower(paramType)
	switch lowerType {
	case "string":
		return "String", nil
	case "securestring":
		return "SecureString", nil
	case "stringlist":
		return "StringList", nil
	default:
		return "", fmt.Errorf("invalid parameter type: %s. Valid types: String, SecureString, StringList", paramType)
	}
}

// getJsonValue retrieves JSON value from file or inline string
func getJsonValue() (string, error) {
	if setJsonFile != "" {
		// Read from file
		content, err := os.ReadFile(setJsonFile)
		if err != nil {
			return "", fmt.Errorf("failed to read JSON file %s: %w", setJsonFile, err)
		}
		return string(content), nil
	} else if setJsonValue != "" {
		// Use inline JSON
		return setJsonValue, nil
	}
	return "", fmt.Errorf("no JSON value provided (use --json or --json-file)")
}

// validateJsonString validates that a string is valid JSON
func validateJsonString(jsonStr string) error {
	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return nil
}

// formatJsonForDisplay returns pretty-printed JSON for display purposes
func formatJsonForDisplay(jsonStr string) (string, error) {
	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return "", err
	}

	formatted, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", err
	}

	return string(formatted), nil
}

// parseJsonPath splits a parameter path into parameter path and JSONPath.
// It separates the parameter path from the JSONPath by splitting on the first colon.
// Format: "/parameter/path:json.path"
// Example: "/config:database.host" returns ("/config", "database.host")
// If no colon is present, returns the full path and empty JSONPath.
//
// This function enables JSONPath-based attribute updates while maintaining backward
// compatibility with standard parameter paths that don't contain colons.
func parseJsonPath(input string) (paramPath string, jsonPath string, err error) {
	colonIndex := strings.Index(input, ":")
	if colonIndex == -1 {
		// No JSONPath specified, just return the parameter path
		return input, "", nil
	}

	paramPath = input[:colonIndex]
	jsonPath = input[colonIndex+1:]

	return paramPath, jsonPath, nil
}

// showJsonAttributeDiff displays a formatted line-by-line diff of JSON attribute changes
// when updating a specific JSON path within a parameter.
// It formats both the old and new JSON with indentation and displays lines with markers:
// - Lines starting with "-" indicate removed content from the old JSON
// - Lines starting with "+" indicate added content in the new JSON
// - Lines starting with spaces indicate unchanged content
//
// Parameters:
//   - oldJSON: the original JSON string
//   - newJSON: the modified JSON string
//   - jsonPath: the JSONPath that was modified (displayed in the diff header)
//
// Returns an error if either JSON string cannot be parsed.
func showJsonAttributeDiff(oldJSON, newJSON string, jsonPath string) error {
	var oldObj, newObj interface{}

	// Parse both JSON values
	if err := json.Unmarshal([]byte(oldJSON), &oldObj); err != nil {
		return fmt.Errorf("failed to parse old JSON: %w", err)
	}
	if err := json.Unmarshal([]byte(newJSON), &newObj); err != nil {
		return fmt.Errorf("failed to parse new JSON: %w", err)
	}

	// Display JSON attribute differences
	fmt.Println()
	fmt.Println(colorizeHeaderLine("[JSON Attribute Diff]"))
	fmt.Println(colorizeHeaderLine(fmt.Sprintf("Path: %s", jsonPath)))

	// Format both JSON objects with consistent indentation for comparison
	oldFormatted, _ := json.MarshalIndent(oldObj, "", "  ")
	newFormatted, _ := json.MarshalIndent(newObj, "", "  ")

	// Split formatted JSON into lines for line-by-line comparison
	oldLines := strings.Split(string(oldFormatted), "\n")
	newLines := strings.Split(string(newFormatted), "\n")

	// Perform simple line-by-line diff display
	// This approach iterates through both line arrays simultaneously,
	// comparing lines at the same index position
	maxLines := len(oldLines)
	if len(newLines) > maxLines {
		maxLines = len(newLines)
	}

	// Iterate through all lines, padding with empty strings as needed
	for i := 0; i < maxLines; i++ {
		oldLine := ""
		newLine := ""

		// Get line from old JSON if available, otherwise use empty string
		if i < len(oldLines) {
			oldLine = oldLines[i]
		}
		// Get line from new JSON if available, otherwise use empty string
		if i < len(newLines) {
			newLine = newLines[i]
		}

		// Output diff markers based on line comparison:
		// - "-" for lines only in old JSON
		// - "+" for lines only in new JSON
		// - " " (space) for unchanged lines
		if oldLine != newLine {
			if oldLine != "" {
				fmt.Println(colorizeRemovalLine(fmt.Sprintf("- %s", oldLine)))
			}
			if newLine != "" {
				fmt.Println(colorizeAdditionLine(fmt.Sprintf("+ %s", newLine)))
			}
		} else if oldLine != "" {
			// Print unchanged lines with space prefix for clarity
			fmt.Printf("  %s\n", oldLine)
		}
	}

	return nil
}
