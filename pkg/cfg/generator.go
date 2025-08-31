package cfg

import (
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

type Generator struct {
	useStagePlaceholders bool
}

func NewGenerator(useStagePlaceholders bool) *Generator {
	return &Generator{
		useStagePlaceholders: useStagePlaceholders,
	}
}

func (g *Generator) GenerateFromStruct(structType interface{}) ([]byte, error) {
	t := reflect.TypeOf(structType)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("input must be a struct or pointer to struct")
	}

	config := g.generateConfigFromStruct(t, "")
	
	return yaml.Marshal(config)
}

func (g *Generator) generateConfigFromStruct(t reflect.Type, prefix string) map[string]interface{} {
	config := make(map[string]interface{})

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		
		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get field name and cfg tag
		fieldName := strings.ToLower(field.Name)
		cfgTag := field.Tag.Get("cfg")
		
		var configKey string
		if cfgTag != "" {
			// Use cfg tag as the key
			configKey = cfgTag
		} else {
			// Generate key from field name
			if prefix != "" {
				configKey = prefix + "." + fieldName
			} else {
				configKey = fieldName
			}
		}

		// Handle nested structs
		if field.Type.Kind() == reflect.Struct {
			nestedConfig := g.generateConfigFromStruct(field.Type, configKey)
			
			// Merge nested config into current level
			parts := strings.Split(configKey, ".")
			current := config
			
			// Navigate to the correct nesting level
			for i, part := range parts[:len(parts)-1] {
				if _, exists := current[part]; !exists {
					current[part] = make(map[string]interface{})
				}
				if i < len(parts)-2 {
					current = current[part].(map[string]interface{})
				}
			}
			
			// Set the nested struct
			if len(parts) == 1 {
				current[parts[0]] = nestedConfig
			} else {
				parent := current[parts[len(parts)-2]].(map[string]interface{})
				parent[parts[len(parts)-1]] = nestedConfig
			}
		} else {
			// Generate appropriate placeholder value
			placeholder := g.generatePlaceholder(configKey, field.Type)
			
			// Set the value in the nested structure
			g.setNestedValue(config, configKey, placeholder)
		}
	}

	return config
}

func (g *Generator) generatePlaceholder(configKey string, fieldType reflect.Type) interface{} {
	// Check if this looks like a secret
	lowerKey := strings.ToLower(configKey)
	isSecret := strings.Contains(lowerKey, "password") || 
		strings.Contains(lowerKey, "secret") || 
		strings.Contains(lowerKey, "key") || 
		strings.Contains(lowerKey, "token")

	// Generate Parameter Store or environment variable reference
	if isSecret || strings.Contains(lowerKey, "host") || strings.Contains(lowerKey, "url") {
		// Use Parameter Store for infrastructure values and secrets
		path := g.generateParameterStorePath(configKey)
		return fmt.Sprintf("${ps:%s}", path)
	} else if strings.Contains(lowerKey, "debug") || strings.Contains(lowerKey, "port") {
		// Use environment variables for runtime config
		envVar := g.generateEnvironmentVariable(configKey)
		return fmt.Sprintf("${env:%s}", envVar)
	}

	// Generate default value based on field type
	switch fieldType.Kind() {
	case reflect.String:
		if strings.Contains(lowerKey, "name") {
			return fmt.Sprintf("\"${env:APP_NAME}-%s\"", g.getStageValue())
		}
		return "\"default-value\""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return 0
	case reflect.Float32, reflect.Float64:
		return 0.0
	case reflect.Bool:
		return false
	default:
		return "default-value"
	}
}

func (g *Generator) generateParameterStorePath(configKey string) string {
	// Convert config key to parameter store path
	parts := strings.Split(configKey, ".")
	
	var pathParts []string
	if g.useStagePlaceholders {
		pathParts = append(pathParts, "/app", "{stage}")
	} else {
		pathParts = append(pathParts, "/app", "dev") // default stage
	}
	
	pathParts = append(pathParts, parts...)
	
	return strings.Join(pathParts, "/")
}

func (g *Generator) generateEnvironmentVariable(configKey string) string {
	// Convert config key to environment variable name
	parts := strings.Split(configKey, ".")
	var envParts []string
	
	for _, part := range parts {
		envParts = append(envParts, strings.ToUpper(part))
	}
	
	return strings.Join(envParts, "_")
}

func (g *Generator) getStageValue() string {
	if g.useStagePlaceholders {
		return "{stage}"
	}
	return "dev"
}

func (g *Generator) setNestedValue(config map[string]interface{}, key string, value interface{}) {
	parts := strings.Split(key, ".")
	current := config

	// Navigate to the correct nesting level
	for _, part := range parts[:len(parts)-1] {
		if _, exists := current[part]; !exists {
			current[part] = make(map[string]interface{})
		}
		current = current[part].(map[string]interface{})
	}

	// Set the final value
	current[parts[len(parts)-1]] = value
}