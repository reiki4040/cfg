package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type ParameterStoreClient struct {
	client *ssm.Client
}

func NewParameterStoreClient(ctx context.Context, region string) (*ParameterStoreClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &ParameterStoreClient{
		client: ssm.NewFromConfig(cfg),
	}, nil
}

func NewParameterStoreClientWithProfile(ctx context.Context, region, profile string) (*ParameterStoreClient, error) {
	var opts []func(*config.LoadOptions) error
	opts = append(opts, config.WithRegion(region))
	
	if profile != "" {
		opts = append(opts, config.WithSharedConfigProfile(profile))
	}
	
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config with profile %s: %w", profile, err)
	}

	return &ParameterStoreClient{
		client: ssm.NewFromConfig(cfg),
	}, nil
}

func (p *ParameterStoreClient) GetParameter(ctx context.Context, name string, decrypt bool) (string, error) {
	input := &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(decrypt),
	}

	result, err := p.client.GetParameter(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get parameter %s: %w", name, err)
	}

	return *result.Parameter.Value, nil
}

func (p *ParameterStoreClient) GetParameters(ctx context.Context, names []string, decrypt bool) (map[string]string, error) {
	if len(names) == 0 {
		return make(map[string]string), nil
	}

	// AWS allows up to 10 parameters per request
	const batchSize = 10
	results := make(map[string]string)

	for i := 0; i < len(names); i += batchSize {
		end := i + batchSize
		if end > len(names) {
			end = len(names)
		}

		batch := names[i:end]
		batchResults, err := p.getParametersBatch(ctx, batch, decrypt)
		if err != nil {
			return nil, err
		}

		for k, v := range batchResults {
			results[k] = v
		}
	}

	return results, nil
}

func (p *ParameterStoreClient) getParametersBatch(ctx context.Context, names []string, decrypt bool) (map[string]string, error) {
	awsNames := make([]string, len(names))
	for i, name := range names {
		awsNames[i] = name
	}

	input := &ssm.GetParametersInput{
		Names:          awsNames,
		WithDecryption: aws.Bool(decrypt),
	}

	result, err := p.client.GetParameters(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get parameters batch: %w", err)
	}

	results := make(map[string]string)
	for _, param := range result.Parameters {
		results[*param.Name] = *param.Value
	}

	// Check for missing parameters
	if len(result.InvalidParameters) > 0 {
		return nil, fmt.Errorf("invalid parameters: %s", strings.Join(result.InvalidParameters, ", "))
	}

	return results, nil
}

func (p *ParameterStoreClient) PutParameter(ctx context.Context, name, value string, paramType string, overwrite bool) error {
	return p.PutParameterWithKey(ctx, name, value, paramType, overwrite, "")
}

func (p *ParameterStoreClient) PutParameterWithKey(ctx context.Context, name, value string, paramType string, overwrite bool, kmsKeyID string) error {
	var ssmType types.ParameterType
	switch strings.ToLower(paramType) {
	case "string":
		ssmType = types.ParameterTypeString
	case "securestring":
		ssmType = types.ParameterTypeSecureString
	case "stringlist":
		ssmType = types.ParameterTypeStringList
	default:
		ssmType = types.ParameterTypeString
	}

	input := &ssm.PutParameterInput{
		Name:      aws.String(name),
		Value:     aws.String(value),
		Type:      ssmType,
		Overwrite: aws.Bool(overwrite),
	}

	// Set KMS key ID for SecureString parameters
	if ssmType == types.ParameterTypeSecureString && kmsKeyID != "" {
		input.KeyId = aws.String(kmsKeyID)
	}

	_, err := p.client.PutParameter(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to put parameter %s: %w", name, err)
	}

	return nil
}

func (p *ParameterStoreClient) DeleteParameter(ctx context.Context, name string) error {
	input := &ssm.DeleteParameterInput{
		Name: aws.String(name),
	}

	_, err := p.client.DeleteParameter(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete parameter %s: %w", name, err)
	}

	return nil
}

type ParameterInfo struct {
	Name  string
	Value string
	Type  string
}

func (p *ParameterStoreClient) GetParametersByPath(ctx context.Context, path string, recursive bool, decrypt bool) (map[string]string, error) {
	parameterInfos, err := p.GetParameterInfosByPath(ctx, path, recursive, decrypt)
	if err != nil {
		return nil, err
	}

	results := make(map[string]string)
	for _, param := range parameterInfos {
		results[param.Name] = param.Value
	}

	return results, nil
}

func (p *ParameterStoreClient) GetParameterInfosByPath(ctx context.Context, path string, recursive bool, decrypt bool) ([]ParameterInfo, error) {
	input := &ssm.GetParametersByPathInput{
		Path:           aws.String(path),
		Recursive:      aws.Bool(recursive),
		WithDecryption: aws.Bool(decrypt),
	}

	var results []ParameterInfo
	paginator := ssm.NewGetParametersByPathPaginator(p.client, input)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get parameters by path %s: %w", path, err)
		}

		for _, param := range page.Parameters {
			results = append(results, ParameterInfo{
				Name:  *param.Name,
				Value: *param.Value,
				Type:  string(param.Type),
			})
		}
	}

	return results, nil
}