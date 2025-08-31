package cfg

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/yourusername/cfg/pkg/aws"
)

type Loader struct {
	awsClient     *aws.ParameterStoreClient
	stageResolver *StageResolver
	resolver      *Resolver
	parser        *Parser
}

type LoadOptions struct {
	Stage     string
	AWSRegion string
	Profile   string
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
	var options LoadOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	stageResolver := NewStageResolver(options.Stage)
	
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

	resolver := NewResolver(awsClient, stageResolver)
	parser := NewParser(resolver)

	return &Loader{
		awsClient:     awsClient,
		stageResolver: stageResolver,
		resolver:      resolver,
		parser:        parser,
	}
}

func (l *Loader) LoadFromFile(path string, target interface{}) error {
	data, err := ioutil.ReadFile(path)
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
	l.stageResolver = NewStageResolver(stage)
	l.resolver = NewResolver(l.awsClient, l.stageResolver)
	l.parser = NewParser(l.resolver)
}

func (l *Loader) GetStage() string {
	return l.stageResolver.GetStage()
}

func (l *Loader) SetAWSClient(awsClient *aws.ParameterStoreClient) {
	l.awsClient = awsClient
	l.resolver = NewResolver(l.awsClient, l.stageResolver)
	l.parser = NewParser(l.resolver)
}