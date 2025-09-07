package cfg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/reiki4040/cfg/aws"
)

type Loader struct {
	awsClient     *aws.ParameterStoreClient
	stageResolver *StageResolver
	resolver      *Resolver
	parser        *Parser
}

type LoadOptions struct {
	Stage             string
	AWSRegion         string
	Profile           string
	StagePrefixBlanks []string
}

type ConfigError struct {
	Type    string // "missing_parameter", "aws_error", "parse_error"
	Path    string
	Message string
	Cause   error
}

func (e *ConfigError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func New(opts ...LoadOptions) *Loader {
	return NewWithPrefix("", opts...)
}

func NewWithPrefix(pathPrefix string, opts ...LoadOptions) *Loader {
	var options LoadOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	stageResolver := NewStageResolverWithPrefixBlanks(options.Stage, options.StagePrefixBlanks)
	
	// Resolve {stage} placeholder in path prefix
	resolvedPathPrefix := stageResolver.ResolvePath(pathPrefix)
	
	var awsClient *aws.ParameterStoreClient
	var err error
	
	// Only initialize AWS client if region is provided or can be determined
	if options.AWSRegion != "" {
		awsClient, err = aws.NewParameterStoreClient(context.Background(), options.AWSRegion)
		if err != nil {
			// Don't fail initialization if AWS is not available
			awsClient = nil
		}
	}

	resolver := NewResolverWithPrefix(awsClient, stageResolver, resolvedPathPrefix)
	parser := NewParser(resolver)

	return &Loader{
		awsClient:     awsClient,
		stageResolver: stageResolver,
		resolver:      resolver,
		parser:        parser,
	}
}

func (l *Loader) LoadFromFile(path string, target interface{}) error {
	// Validate file path for security
	if err := validateConfigFilePath(path); err != nil {
		return &ConfigError{
			Type:    "parse_error",
			Path:    path,
			Message: "invalid file path",
			Cause:   err,
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return &ConfigError{
			Type:    "parse_error",
			Path:    path,
			Message: "failed to read config file",
			Cause:   err,
		}
	}

	return l.LoadFromBytes(data, target)
}

func (l *Loader) LoadFromBytes(data []byte, target interface{}) error {
	ctx := context.Background()

	// Check if Parameter Store references exist and AWS client is available
	if l.resolver.HasParameterStoreReferences(string(data)) && l.awsClient == nil {
		return &ConfigError{
			Type:    "aws_error",
			Message: "Parameter Store references found but AWS client not initialized",
		}
	}

	// Parse and interpolate the YAML
	configData, err := l.parser.ParseAndInterpolate(ctx, data)
	if err != nil {
		return &ConfigError{
			Type:    "parse_error",
			Message: "failed to parse and interpolate config",
			Cause:   err,
		}
	}

	// Map to struct
	if err := l.parser.MapToStruct(configData, target); err != nil {
		return &ConfigError{
			Type:    "parse_error",
			Message: "failed to map config to struct",
			Cause:   err,
		}
	}

	return nil
}

func (l *Loader) SetStage(stage string) {
	// Preserve existing stagePrefixBlanks configuration
	var stagePrefixBlanks []string
	for s := range l.stageResolver.stagePrefixBlanks {
		stagePrefixBlanks = append(stagePrefixBlanks, s)
	}
	l.stageResolver = NewStageResolverWithPrefixBlanks(stage, stagePrefixBlanks)
	l.resolver = NewResolver(l.awsClient, l.stageResolver)
	l.parser = NewParser(l.resolver)
}

func getStagePrefixBlanksFromOptions(stagePrefixBlanks []string) []string {
	if stagePrefixBlanks == nil {
		return []string{"prod"}
	}
	return stagePrefixBlanks
}

func (l *Loader) GetStage() string {
	return l.stageResolver.GetStage()
}

func (l *Loader) SetAWSClient(awsClient *aws.ParameterStoreClient) {
	l.awsClient = awsClient
	l.resolver = NewResolver(l.awsClient, l.stageResolver)
	l.parser = NewParser(l.resolver)
}

func validateConfigFilePath(path string) error {
	// Clean and validate the path
	cleanPath := filepath.Clean(path)
	
	// Check for path traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("path traversal not allowed")
	}
	
	// Ensure absolute path or relative path in current directory
	if !filepath.IsAbs(cleanPath) {
		// Convert to absolute path to validate
		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			return fmt.Errorf("failed to resolve absolute path: %w", err)
		}
		cleanPath = absPath
	}
	
	// Check file extension for additional safety
	ext := filepath.Ext(cleanPath)
	allowedExts := []string{".yaml", ".yml", ".json"}
	validExt := false
	for _, allowed := range allowedExts {
		if strings.EqualFold(ext, allowed) {
			validExt = true
			break
		}
	}
	
	if !validExt {
		return fmt.Errorf("unsupported file extension: %s (allowed: .yaml, .yml, .json)", ext)
	}
	
	return nil
}