package cfg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/reiki4040/cfg/aws"
)

var (
	psReferenceRegex          = regexp.MustCompile(`\$\{ps:((?:[^{}]|\{[^}]*\})*)\}`)
	envReferenceRegex         = regexp.MustCompile(`\$\{env:([^}]+)\}`)
	stagePrefixReferenceRegex = regexp.MustCompile(`\$\{stage-prefix:([^}]+)\}`)
)

type Reference struct {
	Type     string // "ps", "env", or "stage-prefix"
	Key      string // parameter path (for ps), var name (for env)
	JSONPath string // JSONPath for ps references (optional, e.g., "parent.child.key")
	Raw      string // original reference string
}

type Resolver struct {
	psClient      *aws.ParameterStoreClient
	envVars       map[string]string
	cache         map[string]string
	stageResolver *StageResolver
	pathPrefix    string
	jsonCache     *JsonCache
	jsonExtractor *JsonPathExtractor
	logger        *Logger
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
		jsonCache:     NewJsonCache(5 * time.Minute),
		jsonExtractor: NewJsonPathExtractor(),
		logger:        NewLogger(LogLevelWarn), // Default to warn level
	}
}

// SetLogLevel sets the logging level for this resolver
func (r *Resolver) SetLogLevel(level LogLevel) {
	r.logger = NewLogger(level)
}

func (r *Resolver) ExtractReferences(yamlContent string) []Reference {
	var references []Reference

	// Remove comments from YAML content before extracting references
	// This prevents processing references that appear in comment lines
	cleanedContent := removeYamlComments(yamlContent)

	// Extract Parameter Store references (with JSONPath support)
	psMatches := psReferenceRegex.FindAllStringSubmatch(cleanedContent, -1)
	for _, match := range psMatches {
		if len(match) >= 2 {
			rawRef := match[0]

			// Use JsonPathMatcher to parse path and JSONPath
			matcher := NewJsonPathMatcher()
			path, jsonPath, ok := matcher.Match(rawRef)
			if !ok {
				continue
			}

			// Apply path prefix if configured
			if r.pathPrefix != "" {
				// Normalize prefix to ensure it starts with / and doesn't end with /
				prefix := r.pathPrefix
				if !strings.HasPrefix(prefix, "/") {
					prefix = "/" + prefix
				}
				prefix = strings.TrimSuffix(prefix, "/")

				// For paths starting with /, combine prefix + path
				// For relative paths, treat them as absolute within the prefix
				if strings.HasPrefix(path, "/") {
					path = prefix + path
				} else {
					path = prefix + "/" + path
				}
			} else {
				// No prefix configured, ensure path starts with /
				if !strings.HasPrefix(path, "/") {
					path = "/" + path
				}
			}

			key := r.stageResolver.ResolvePath(path)
			references = append(references, Reference{
				Type:     "ps",
				Key:      key,
				JSONPath: jsonPath,
				Raw:      rawRef,
			})
		}
	}

	// Extract environment variable references (from cleaned content)
	envMatches := envReferenceRegex.FindAllStringSubmatch(cleanedContent, -1)
	for _, match := range envMatches {
		if len(match) >= 2 {
			references = append(references, Reference{
				Type: "env",
				Key:  match[1],
				Raw:  match[0],
			})
		}
	}

	// Extract stage-prefix references (from cleaned content)
	stagePrefixMatches := stagePrefixReferenceRegex.FindAllStringSubmatch(cleanedContent, -1)
	for _, match := range stagePrefixMatches {
		if len(match) >= 2 {
			references = append(references, Reference{
				Type: "stage-prefix",
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
	psRefMap := make(map[string][]Reference)

	for _, ref := range refs {
		switch ref.Type {
		case "ps":
			// ref.Key is already resolved by ExtractReferences
			if _, exists := r.cache[ref.Key]; !exists {
				psKeys = append(psKeys, ref.Key)
			}
			psRefMap[ref.Key] = append(psRefMap[ref.Key], ref)
		case "env":
			if envValue, exists := r.envVars[ref.Key]; exists {
				values[ref.Raw] = envValue
			} else {
				return nil, fmt.Errorf("environment variable %s not found", ref.Key)
			}
		case "stage-prefix":
			// Resolve stage-prefix immediately
			var stagePrefix string
			if r.stageResolver.stagePrefixBlanks[r.stageResolver.stage] {
				stagePrefix = ""
			} else {
				stagePrefix = r.stageResolver.stage + "-"
			}
			stagePrefixValue := stagePrefix + ref.Key
			values[ref.Raw] = stagePrefixValue
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

	// Map Parameter Store values to raw references (with JSONPath extraction)
	for key, refs := range psRefMap {
		if cachedValue, exists := r.cache[key]; exists {
			// Map the same value to all references with this key
			for _, ref := range refs {
				mappedValue := r.resolveParameterValue(key, cachedValue, ref)
				values[ref.Raw] = mappedValue
			}
		} else if r.psClient != nil {
			// Use empty string for missing parameters instead of failing
			for _, ref := range refs {
				values[ref.Raw] = ""
			}
		}
	}

	return values, nil
}

// resolveParameterValue resolves a parameter value, extracting JSONPath if specified
func (r *Resolver) resolveParameterValue(paramKey string, paramValue string, ref Reference) string {
	// If no JSONPath, return the full value
	if ref.JSONPath == "" {
		return paramValue
	}

	r.logger.LogReferenceFetch(paramKey, ref.JSONPath)

	// Try to extract from cached JSON first
	if jsonObj, exists := r.jsonCache.Get(paramKey); exists {
		r.logger.LogCacheHit(paramKey)
		// Extract the specific key path from cached JSON
		result, err := r.extractJsonPathFromCached(paramKey, jsonObj, ref.JSONPath)
		if err == nil {
			r.logger.LogJsonValueExtracted(paramKey, ref.JSONPath)
			return result
		}
		// If extraction fails, fall back to trying to parse the value
		r.logger.LogError(paramKey, ref.JSONPath, err)
	}

	r.logger.LogCacheMiss(paramKey)

	// Try to parse the parameter value as JSON and extract
	r.logger.LogJsonParsing(paramKey, len(paramValue))
	result, jpeErr := r.jsonExtractor.ExtractValue(paramValue, ref.JSONPath, paramKey)
	if jpeErr == nil {
		// Cache the parsed JSON for future use
		var jsonObj map[string]interface{}
		if parseErr := parseJSON(paramValue, &jsonObj); parseErr == nil {
			r.jsonCache.Set(paramKey, jsonObj)
		}
		r.logger.LogJsonValueExtracted(paramKey, ref.JSONPath)
		return result
	}

	// If JSONPath extraction fails, log error and return empty string
	switch jpeErr.Type {
	case "key_not_found":
		r.logger.LogKeyNotFound(paramKey, ref.JSONPath, jpeErr.AvailableKeys)
	case "type_mismatch":
		r.logger.LogTypeMismatch(paramKey, ref.JSONPath, jpeErr.ExpectedType, jpeErr.ActualType)
	case "parse_error":
		r.logger.LogParseFailure(paramKey, jpeErr)
	}
	r.logger.LogError(paramKey, ref.JSONPath, jpeErr)
	return ""
}

// extractJsonPathFromCached extracts a value from a cached JSON object
func (r *Resolver) extractJsonPathFromCached(paramKey string, jsonObj map[string]interface{}, jsonPath string) (string, *JSONPathError) {
	// For cached JSON, we need to convert it back to string and use the extractor
	// Or we can implement direct extraction from the map
	// For simplicity, we'll use the extractor which handles all the logic
	jsonBytes, _ := marshalJSON(jsonObj)
	return r.jsonExtractor.ExtractValue(string(jsonBytes), jsonPath, paramKey)
}

// parseJSON is a helper to parse JSON string
func parseJSON(jsonStr string, target interface{}) error {
	return json.Unmarshal([]byte(jsonStr), target)
}

// marshalJSON is a helper to marshal JSON
func marshalJSON(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

// removeYamlComments removes YAML comments from the content
// This prevents references (like ${ps:...}) in comment lines from being processed
func removeYamlComments(content string) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		// Find the position of '#' that starts a comment
		// Need to be careful not to remove '#' that appears in string values
		commentPos := findCommentStart(line)
		if commentPos >= 0 {
			// Remove the comment part
			line = line[:commentPos]
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// findCommentStart finds the position of a YAML comment start
// Returns -1 if no comment found, or the position of '#' if found
func findCommentStart(line string) int {
	inDoubleQuote := false
	inSingleQuote := false

	for i, ch := range line {
		// Check for escaped character
		if i > 0 && line[i-1] == '\\' {
			continue
		}

		// Handle quote toggling
		if ch == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
		} else if ch == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
		}

		// Check for comment start (outside of quotes)
		if ch == '#' && !inDoubleQuote && !inSingleQuote {
			return i
		}
	}

	return -1
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
