package cfg

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/yourusername/cfg/pkg/aws"
)

var (
	psReferenceRegex  = regexp.MustCompile(`\$\{ps:([^}]+)\}`)
	envReferenceRegex = regexp.MustCompile(`\$\{env:([^}]+)\}`)
)

type Reference struct {
	Type string // "ps" or "env"
	Key  string
	Raw  string
}

type Resolver struct {
	psClient     *aws.ParameterStoreClient
	envVars      map[string]string
	cache        map[string]string
	stageResolver *StageResolver
	pathPrefix   string
}

func NewResolver(psClient *aws.ParameterStoreClient, stageResolver *StageResolver) *Resolver {
	return NewResolverWithPrefix(psClient, stageResolver, "")
}

func NewResolverWithPrefix(psClient *aws.ParameterStoreClient, stageResolver *StageResolver, pathPrefix string) *Resolver {
	return &Resolver{
		psClient:      psClient,
		envVars:       getEnvironmentVariables(),
		cache:         make(map[string]string),
		stageResolver: stageResolver,
		pathPrefix:    pathPrefix,
	}
}

func (r *Resolver) ExtractReferences(yamlContent string) []Reference {
	var references []Reference

	// Extract Parameter Store references
	psMatches := psReferenceRegex.FindAllStringSubmatch(yamlContent, -1)
	for _, match := range psMatches {
		if len(match) >= 2 {
			path := match[1]
			// Apply path prefix if path doesn't start with /
			if r.pathPrefix != "" && !strings.HasPrefix(path, "/") {
				// Ensure path prefix starts with /
				prefix := r.pathPrefix
				if !strings.HasPrefix(prefix, "/") {
					prefix = "/" + prefix
				}
				// Ensure path prefix doesn't end with /
				prefix = strings.TrimSuffix(prefix, "/")
				path = prefix + "/" + path
			}
			key := r.stageResolver.ResolvePath(path)
			references = append(references, Reference{
				Type: "ps",
				Key:  key,
				Raw:  match[0],
			})
		}
	}

	// Extract environment variable references
	envMatches := envReferenceRegex.FindAllStringSubmatch(yamlContent, -1)
	for _, match := range envMatches {
		if len(match) >= 2 {
			references = append(references, Reference{
				Type: "env",
				Key:  match[1],
				Raw:  match[0],
			})
		}
	}

	return references
}

func (r *Resolver) ResolveReferences(ctx context.Context, refs []Reference) (map[string]string, error) {
	values := make(map[string]string)

	// Group Parameter Store references for batch retrieval
	var psKeys []string
	psRefMap := make(map[string]Reference)
	
	for _, ref := range refs {
		switch ref.Type {
		case "ps":
			if _, exists := r.cache[ref.Key]; !exists {
				psKeys = append(psKeys, ref.Key)
			}
			psRefMap[ref.Key] = ref
		case "env":
			if envValue, exists := r.envVars[ref.Key]; exists {
				values[ref.Raw] = envValue
			} else {
				return nil, fmt.Errorf("environment variable %s not found", ref.Key)
			}
		}
	}

	// Batch retrieve Parameter Store values
	if len(psKeys) > 0 && r.psClient != nil {
		psValues, err := r.psClient.GetParameters(ctx, psKeys, true)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve Parameter Store references: %w", err)
		}

		// Cache and map the values
		for key, value := range psValues {
			r.cache[key] = value
		}
	}

	// Map Parameter Store values to raw references
	for key, ref := range psRefMap {
		if cachedValue, exists := r.cache[key]; exists {
			values[ref.Raw] = cachedValue
		} else if r.psClient != nil {
			return nil, fmt.Errorf("parameter %s not found in Parameter Store", key)
		}
	}

	return values, nil
}

func (r *Resolver) InterpolateString(input string, values map[string]string) string {
	result := input
	for placeholder, value := range values {
		result = strings.ReplaceAll(result, placeholder, value)
	}
	
	// Resolve stage placeholders
	result = r.stageResolver.ResolveString(result)
	
	return result
}

func (r *Resolver) HasParameterStoreReferences(content string) bool {
	return psReferenceRegex.MatchString(content)
}

func getEnvironmentVariables() map[string]string {
	envVars := make(map[string]string)
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envVars[parts[0]] = parts[1]
		}
	}
	return envVars
}