package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourusername/cfg/pkg/aws"
	"github.com/yourusername/cfg/pkg/cfg"
)

var (
	stage     string
	awsRegion string
	awsProfile string
)

var rootCmd = &cobra.Command{
	Use:   "cfgctl",
	Short: "Configuration management tool for AWS Parameter Store and YAML configs",
	Long: `cfgctl is a CLI tool for managing application configuration across environments.
It supports AWS Parameter Store integration, environment-specific settings,
and YAML-based configuration management.`,
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
}

func createAWSClient() (*aws.ParameterStoreClient, error) {
	return aws.NewParameterStoreClientWithProfile(context.Background(), awsRegion, awsProfile)
}

func createStageResolver() *cfg.StageResolver {
	return cfg.NewStageResolver(stage)
}

func resolveParameterPath(path string, stageResolver *cfg.StageResolver) string {
	// Apply path prefix if path doesn't start with /
	if !strings.HasPrefix(path, "/") {
		pathPrefix := getPathPrefix()
		if pathPrefix != "" {
			// Ensure path prefix starts with /
			if !strings.HasPrefix(pathPrefix, "/") {
				pathPrefix = "/" + pathPrefix
			}
			// Ensure path prefix doesn't end with /
			pathPrefix = strings.TrimSuffix(pathPrefix, "/")
			
			// If path prefix contains {stage} placeholder, resolve it first
			if strings.Contains(pathPrefix, "{stage}") {
				pathPrefix = stageResolver.ResolvePath(pathPrefix)
			}
			
			path = pathPrefix + "/" + path
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