package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/reiki4040/cfg/aws"
	"github.com/reiki4040/cfg"
)

var (
	stage     string
	awsRegion string
	awsProfile string
	showVersion bool
)

var rootCmd = &cobra.Command{
	Use:   "cfgctl",
	Short: "Configuration management tool for AWS Parameter Store and YAML configs",
	Long: `cfgctl is a CLI tool for managing application configuration across environments.
It supports AWS Parameter Store integration, environment-specific settings,
and YAML-based configuration management.`,
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			if GetVersionString != nil {
				fmt.Println(GetVersionString())
			} else {
				fmt.Println("Version information not available")
			}
			return
		}
		cmd.Help()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Load defaults from config file
	defaultRegion, defaultStage := getDefaultRegionAndStage()
	
	rootCmd.PersistentFlags().StringVar(&stage, "stage", defaultStage, "Configuration stage (dev, stg, prod)")
	rootCmd.PersistentFlags().StringVar(&awsRegion, "region", defaultRegion, "AWS region")
	rootCmd.PersistentFlags().StringVar(&awsProfile, "profile", "", "AWS profile")
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")
}

func createAWSClient() (*aws.ParameterStoreClient, error) {
	return aws.NewParameterStoreClientWithProfile(context.Background(), awsRegion, awsProfile)
}

func createStageResolver() *cfg.StageResolver {
	stagePrefixBlanks := getStagePrefixBlanks()
	return cfg.NewStageResolverWithPrefixBlanks(stage, stagePrefixBlanks)
}

func resolveParameterPath(path string, stageResolver *cfg.StageResolver) string {
	pathPrefix := getPathPrefix()
	
	// Apply path prefix if configured
	if pathPrefix != "" {
		// Normalize prefix to ensure it starts with / and doesn't end with /
		if !strings.HasPrefix(pathPrefix, "/") {
			pathPrefix = "/" + pathPrefix
		}
		pathPrefix = strings.TrimSuffix(pathPrefix, "/")
		
		// If path prefix contains {stage} placeholder, resolve it first
		if strings.Contains(pathPrefix, "{stage}") {
			pathPrefix = stageResolver.ResolvePath(pathPrefix)
		}
		
		// For paths starting with /, combine prefix + path
		// For relative paths, treat them as absolute within the prefix
		if strings.HasPrefix(path, "/") {
			path = pathPrefix + path
		} else {
			path = pathPrefix + "/" + path
		}
	} else {
		// No prefix configured, ensure path starts with /
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
	}
	
	return stageResolver.ResolvePath(path)
}

func printError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}

func isSecretParameter(path string) bool {
	lowerPath := strings.ToLower(path)
	secretKeywords := []string{"password", "secret", "key", "token", "credential"}
	
	for _, keyword := range secretKeywords {
		if strings.Contains(lowerPath, keyword) {
			return true
		}
	}
	return false
}