package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type CfgctlConfig struct {
	DefaultRegion     string                   `yaml:"default_region"`
	DefaultStage      string                   `yaml:"default_stage"`
	PathPrefix        string                   `yaml:"path_prefix"`
	KMSKeys           map[string]RegionKMSKeys `yaml:"kms_keys"`
	StagePrefixBlanks []string                 `yaml:"stage_prefix_blanks"`
}

type RegionKMSKeys struct {
	Stages map[string]string `yaml:"stages"`
}

var globalConfig *CfgctlConfig

func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".cfgctl.config"), nil
}

func loadConfig() (*CfgctlConfig, error) {
	if globalConfig != nil {
		return globalConfig, nil
	}

	configPath, err := getConfigFilePath()
	if err != nil {
		return nil, err
	}

	// If config file doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		globalConfig = &CfgctlConfig{
			DefaultRegion:     "us-east-1",
			DefaultStage:      "dev",
			PathPrefix:        "/app",
			KMSKeys:           make(map[string]RegionKMSKeys),
			StagePrefixBlanks: []string{"prod"},
		}
		return globalConfig, nil
	}

	// Read and parse config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config CfgctlConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Initialize maps if nil
	if config.KMSKeys == nil {
		config.KMSKeys = make(map[string]RegionKMSKeys)
	}

	globalConfig = &config
	return globalConfig, nil
}

func saveConfig(config *CfgctlConfig) error {
	configPath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	globalConfig = config
	return nil
}

func getKMSKeyForStageAndRegion(stage, region string) string {
	config, err := loadConfig()
	if err != nil {
		return ""
	}

	if regionKeys, exists := config.KMSKeys[region]; exists {
		if kmsKey, exists := regionKeys.Stages[stage]; exists {
			return kmsKey
		}
	}

	return ""
}

func setKMSKeyForStageAndRegion(stage, region, kmsKey string) error {
	config, err := loadConfig()
	if err != nil {
		return err
	}

	if config.KMSKeys == nil {
		config.KMSKeys = make(map[string]RegionKMSKeys)
	}

	if _, exists := config.KMSKeys[region]; !exists {
		config.KMSKeys[region] = RegionKMSKeys{
			Stages: make(map[string]string),
		}
	}

	regionKeys := config.KMSKeys[region]
	if regionKeys.Stages == nil {
		regionKeys.Stages = make(map[string]string)
	}

	regionKeys.Stages[stage] = kmsKey
	config.KMSKeys[region] = regionKeys

	return saveConfig(config)
}

func getDefaultRegionAndStage() (string, string) {
	config, err := loadConfig()
	if err != nil {
		return "us-east-1", "dev"
	}

	defaultRegion := config.DefaultRegion
	if defaultRegion == "" {
		defaultRegion = "us-east-1"
	}

	defaultStage := config.DefaultStage
	if defaultStage == "" {
		defaultStage = "dev"
	}

	return defaultRegion, defaultStage
}

func getPathPrefix() string {
	config, err := loadConfig()
	if err != nil {
		return "/app"
	}

	if config.PathPrefix == "" {
		return "/app"
	}

	return config.PathPrefix
}

func setPathPrefix(prefix string) error {
	config, err := loadConfig()
	if err != nil {
		return err
	}

	config.PathPrefix = prefix
	return saveConfig(config)
}

func getStagePrefixBlanks() []string {
	config, err := loadConfig()
	if err != nil {
		return []string{"prod"}
	}

	if config.StagePrefixBlanks == nil {
		return []string{"prod"}
	}

	return config.StagePrefixBlanks
}

func setStagePrefixBlanks(stages []string) error {
	config, err := loadConfig()
	if err != nil {
		return err
	}

	config.StagePrefixBlanks = stages
	return saveConfig(config)
}
