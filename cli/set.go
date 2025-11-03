package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var setCmd = &cobra.Command{
	Use:   "set <parameter-path> [value]",
	Short: "Set a parameter value in Parameter Store",
	Long: `Set a parameter value in AWS Parameter Store.
The parameter path supports {stage} placeholder which will be replaced with the current stage.
JSON values can be set using --json flag, either from file or as inline JSON string.

Examples:
  cfgctl set /app/{stage}/db/password --stage=prod --SS
  cfgctl set /app/prod/api/timeout "30" -S --no-interactive
  cfgctl set /app/{stage}/db/password --SS
  cfgctl set /app/prod/config --json='{"host":"localhost","port":5432}'
  cfgctl set /app/prod/config --json-file=config.json`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runSetCommand,
}

var (
	setType        string
	setOverwrite   bool
	setInteractive bool
	setKMSKeyID    string
	setTypeString  bool
	setTypeSecure  bool
	setTypeList    bool
	setJsonValue   string
	setJsonFile    string
	setJsonValidate bool
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

	// Mark flags as mutually exclusive
	setCmd.MarkFlagsMutuallyExclusive("type", "string", "SS", "SL")
}

func runSetCommand(cmd *cobra.Command, args []string) error {
	parameterPath := args[0]
	var value string

	// Handle JSON input first
	if setJsonValue != "" || setJsonFile != "" {
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
		// Force type to String for JSON values
		setType = "String"
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
			value, err = promptForValue(parameterPath, setType == "SecureString")
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

	// Create stage resolver and resolve path
	stageResolver := createStageResolver()
	resolvedPath := resolveParameterPath(parameterPath, stageResolver)


	ctx := context.Background()

	// Check if parameter already exists and show diff
	existingValue, err := awsClient.GetParameter(ctx, resolvedPath, true)
	parameterExists := (err == nil)
	
	if parameterExists {
		fmt.Printf("Parameter %s already exists.\n", resolvedPath)

		// Show diff (hide values for secrets)
		if setType == "SecureString" {
			fmt.Println("Current value: [HIDDEN]")
			fmt.Println("New value:     [HIDDEN]")
		} else {
			// Try to format as JSON for better readability
			currentFormatted, err1 := formatJsonForDisplay(existingValue)
			newFormatted, err2 := formatJsonForDisplay(value)

			if err1 == nil && err2 == nil {
				// Both are valid JSON, show formatted
				fmt.Println("Current value (JSON):")
				fmt.Println(currentFormatted)
				fmt.Println("\nNew value (JSON):")
				fmt.Println(newFormatted)
			} else {
				// Show as plain text
				if len(existingValue) > 200 {
					fmt.Printf("Current value: %s...\n", existingValue[:200])
				} else {
					fmt.Printf("Current value: %s\n", existingValue)
				}

				if len(value) > 200 {
					fmt.Printf("New value: %s...\n", value[:200])
				} else {
					fmt.Printf("New value: %s\n", value)
				}
			}
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