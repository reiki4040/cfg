// +build ignore

package main

import (
	"fmt"
	"log"

	"github.com/reiki4040/cfg"
)

// Config structure for JSONPath example
// This demonstrates how to use JSONPath to extract values from JSON stored in Parameter Store
type JSONPathExampleConfig struct {
	App struct {
		Name    string `yaml:"app.name"`
		Version string `yaml:"app.version"`
	}

	Database struct {
		Primary struct {
			Host     string `yaml:"database.primary_host"`
			Port     int    `yaml:"database.primary_port"`
			Username string `yaml:"database.primary_user"`
			Password string `yaml:"database.primary_user"` // Note: would be masked in real usage
		}
		Replica struct {
			Host string `yaml:"database.replica_host"`
			Port int    `yaml:"database.replica_port"`
		}
		Pool struct {
			MinConnections int `yaml:"database.pool_min"`
			MaxConnections int `yaml:"database.pool_max"`
			IdleTimeout    int `yaml:"database.pool_idle"`
		}
	}

	Server struct {
		Host            string `yaml:"server.host"`
		Port            int    `yaml:"server.port"`
		TimeoutSeconds  int    `yaml:"server.timeout_seconds"`
		TLSEnabled      bool   `yaml:"server.tls.enabled"`
		TLSCertPath     string `yaml:"server.tls.cert_path"`
		TLSKeyPath      string `yaml:"server.tls.key_path"`
	}

	Logging struct {
		Level  string `yaml:"logging.level"`
		Format string `yaml:"logging.format"`
	}

	Secrets struct {
		APIKey          string `yaml:"secrets.api_key"`
		DatabasePassword string `yaml:"secrets.database_password"`
	}

	Environment struct {
		Stage           string `yaml:"environment.stage"`
		EnvironmentType string `yaml:"environment.environment_type"`
		Region          string `yaml:"environment.region"`
	}
}

func main() {
	var config JSONPathExampleConfig

	// Create loader for the dev stage
	// This will resolve {stage} placeholder to "dev" in all references
	loader := cfg.New(cfg.LoadOptions{
		Stage:     "dev",
		AWSRegion: "ap-northeast-1",
		StagePrefixBlanks: []string{"prod"},
	})

	// Load configuration from config-jsonpath.yaml
	// This example demonstrates JSONPath usage by extracting values from JSON parameters
	err := loader.LoadFromFile("config-jsonpath.yaml", &config)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Display application configuration
	fmt.Println("=== Application Configuration ===")
	fmt.Printf("App Name: %s\n", config.App.Name)
	fmt.Printf("App Version: %s\n", config.App.Version)

	// Display database configuration
	// Note: These values are extracted from a single JSON parameter using JSONPath
	// For example: ${ps:/app/{stage}/database-config:primary.host}
	fmt.Println("\n=== Database Configuration ===")
	fmt.Printf("Primary DB Host: %s\n", config.Database.Primary.Host)
	fmt.Printf("Primary DB Port: %d\n", config.Database.Primary.Port)
	fmt.Printf("Primary DB User: %s\n", config.Database.Primary.Username)
	fmt.Printf("Replica DB Host: %s\n", config.Database.Replica.Host)
	fmt.Printf("Replica DB Port: %d\n", config.Database.Replica.Port)
	fmt.Printf("Pool Min Connections: %d\n", config.Database.Pool.MinConnections)
	fmt.Printf("Pool Max Connections: %d\n", config.Database.Pool.MaxConnections)
	fmt.Printf("Pool Idle Timeout: %d\n", config.Database.Pool.IdleTimeout)

	// Display server configuration
	// These values are extracted from JSON using JSONPath
	fmt.Println("\n=== Server Configuration ===")
	fmt.Printf("Server Host: %s\n", config.Server.Host)
	fmt.Printf("Server Port: %d\n", config.Server.Port)
	fmt.Printf("Server Timeout: %d seconds\n", config.Server.TimeoutSeconds)
	fmt.Printf("TLS Enabled: %t\n", config.Server.TLSEnabled)
	fmt.Printf("TLS Cert Path: %s\n", config.Server.TLSCertPath)
	fmt.Printf("TLS Key Path: %s\n", config.Server.TLSKeyPath)

	// Display logging configuration
	fmt.Println("\n=== Logging Configuration ===")
	fmt.Printf("Log Level: %s\n", config.Logging.Level)
	fmt.Printf("Log Format: %s\n", config.Logging.Format)

	// Display environment information
	fmt.Println("\n=== Environment ===")
	fmt.Printf("Stage: %s\n", config.Environment.Stage)
	fmt.Printf("Environment Type: %s\n", config.Environment.EnvironmentType)
	fmt.Printf("Region: %s\n", config.Environment.Region)

	// Display secrets summary (values hidden in production)
	fmt.Println("\n=== Secrets (Summary) ===")
	fmt.Printf("API Key: [%d chars]\n", len(config.Secrets.APIKey))
	fmt.Printf("Database Password: [%d chars]\n", len(config.Secrets.DatabasePassword))

	// --- Key Benefits Demonstrated ---
	//
	// 1. JSONPath Deduplication:
	//    All database values come from a single /app/{stage}/database-config parameter
	//    This results in only 1 API call to Parameter Store instead of 7
	//
	// 2. Nested Structure Navigation:
	//    JSONPath allows navigating nested JSON objects with dot notation
	//    Example: database.primary.host, server.tls.cert_path
	//
	// 3. Stage Placeholder Resolution:
	//    {stage} is automatically replaced with "dev" at runtime
	//    Enables easy multi-stage configuration management
	//
	// 4. Type Conversion:
	//    JSON values (strings, numbers, booleans) are automatically converted
	//    to Go types based on struct tags
	//
	// 5. Environment Variable Support:
	//    ${env:VAR_NAME} references environment variables
	//    ${stage-prefix:default} generates stage-specific prefixes
	//
	fmt.Println("\n=== JSONPath Benefits ===")
	fmt.Println("✓ API Call Reduction: Multiple values from 1 parameter = 1 API call")
	fmt.Println("✓ Nested JSON Navigation: Dot notation for deep object traversal")
	fmt.Println("✓ Stage Placeholder: {stage} automatically replaced with current stage")
	fmt.Println("✓ Type Conversion: JSON types converted to Go types automatically")
	fmt.Println("✓ Backward Compatible: Works alongside traditional references")
	fmt.Println("✓ Automatic Caching: Parsed JSON cached for 5 minutes")
}
