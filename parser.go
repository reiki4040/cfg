package cfg

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Parser struct {
	resolver *Resolver
}

func NewParser(resolver *Resolver) *Parser {
	return &Parser{
		resolver: resolver,
	}
}

func (p *Parser) ParseAndInterpolate(ctx context.Context, yamlData []byte) (map[string]interface{}, error) {
	// Validate input size to prevent YAML bomb attacks
	const maxYAMLSize = 10 * 1024 * 1024 // 10MB limit
	if len(yamlData) > maxYAMLSize {
		return nil, fmt.Errorf("YAML input too large: %d bytes (max: %d)", len(yamlData), maxYAMLSize)
	}

	// First parse YAML into map
	var data map[string]interface{}
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Convert back to string for reference extraction
	yamlString := string(yamlData)

	// Extract all references
	refs := p.resolver.ExtractReferences(yamlString)
	
	// Resolve references if any exist
	var resolvedValues map[string]string
	if len(refs) > 0 {
		var err error
		resolvedValues, err = p.resolver.ResolveReferences(ctx, refs)
		if err != nil {
			return nil, err
		}
	} else {
		resolvedValues = make(map[string]string)
	}

	// Interpolate the resolved values
	interpolatedYAML := p.resolver.InterpolateString(yamlString, resolvedValues)

	// Parse the interpolated YAML
	var result map[string]interface{}
	if err := yaml.Unmarshal([]byte(interpolatedYAML), &result); err != nil {
		return nil, fmt.Errorf("failed to parse interpolated YAML: %w", err)
	}

	return result, nil
}

func (p *Parser) MapToStruct(data map[string]interface{}, target interface{}) error {
	return mapToStruct(data, target, "")
}

func mapToStruct(data map[string]interface{}, target interface{}, prefix string) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer to struct")
	}

	targetValue = targetValue.Elem()
	if targetValue.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}

	targetType := targetValue.Type()

	for i := 0; i < targetValue.NumField(); i++ {
		field := targetValue.Field(i)
		fieldType := targetType.Field(i)

		if !field.CanSet() {
			continue
		}

		// Get yaml tag (prioritize yaml over cfg for backward compatibility)
		yamlTag := fieldType.Tag.Get("yaml")
		cfgTag := fieldType.Tag.Get("cfg")
		
		var tag string
		if yamlTag != "" {
			tag = yamlTag
		} else if cfgTag != "" {
			tag = cfgTag
		} else {
			// If no tag, use field name in lowercase
			tag = strings.ToLower(fieldType.Name)
		}
		
		// Apply prefix if needed
		if prefix != "" && !strings.Contains(tag, ".") {
			tag = prefix + "." + tag
		}

		// Handle nested structs
		if field.Kind() == reflect.Struct {
			if err := mapToStruct(data, field.Addr().Interface(), tag); err != nil {
				return err
			}
			continue
		}

		// Get value from data map
		value, err := getValueFromPath(data, tag)
		if err != nil {
			continue // Skip missing values
		}

		// Set value based on field type
		if err := setFieldValue(field, value); err != nil {
			return fmt.Errorf("failed to set field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

func getValueFromPath(data map[string]interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		if current == nil {
			return nil, fmt.Errorf("path not found: %s", path)
		}

		if val, ok := current[part]; ok {
			if nextMap, ok := val.(map[string]interface{}); ok {
				current = nextMap
			} else {
				// This is the final value
				return val, nil
			}
		} else {
			return nil, fmt.Errorf("key not found: %s in path %s", part, path)
		}
	}

	return current, nil
}

func setFieldValue(field reflect.Value, value interface{}) error {
	switch field.Kind() {
	case reflect.String:
		if str, ok := value.(string); ok {
			field.SetString(str)
		} else {
			field.SetString(fmt.Sprintf("%v", value))
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intVal, err := toInt64(value); err == nil {
			field.SetInt(intVal)
		} else {
			return fmt.Errorf("cannot convert %v to int", value)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if uintVal, err := toUint64(value); err == nil {
			field.SetUint(uintVal)
		} else {
			return fmt.Errorf("cannot convert %v to uint", value)
		}
	case reflect.Float32, reflect.Float64:
		if floatVal, err := toFloat64(value); err == nil {
			field.SetFloat(floatVal)
		} else {
			return fmt.Errorf("cannot convert %v to float", value)
		}
	case reflect.Bool:
		if boolVal, err := toBool(value); err == nil {
			field.SetBool(boolVal)
		} else {
			return fmt.Errorf("cannot convert %v to bool", value)
		}
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}

func toInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int8:
		return int64(v), nil
	case int16:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case uint:
		return int64(v), nil
	case uint8:
		return int64(v), nil
	case uint16:
		return int64(v), nil
	case uint32:
		return int64(v), nil
	case uint64:
		return int64(v), nil
	case float32:
		return int64(v), nil
	case float64:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to int64", value)
	}
}

func toUint64(value interface{}) (uint64, error) {
	switch v := value.(type) {
	case uint:
		return uint64(v), nil
	case uint8:
		return uint64(v), nil
	case uint16:
		return uint64(v), nil
	case uint32:
		return uint64(v), nil
	case uint64:
		return v, nil
	case int:
		return uint64(v), nil
	case int8:
		return uint64(v), nil
	case int16:
		return uint64(v), nil
	case int32:
		return uint64(v), nil
	case int64:
		return uint64(v), nil
	case float32:
		return uint64(v), nil
	case float64:
		return uint64(v), nil
	case string:
		return strconv.ParseUint(v, 10, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to uint64", value)
	}
}

func toFloat64(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", value)
	}
}

func toBool(value interface{}) (bool, error) {
	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	case int:
		return v != 0, nil
	case int8:
		return v != 0, nil
	case int16:
		return v != 0, nil
	case int32:
		return v != 0, nil
	case int64:
		return v != 0, nil
	case uint:
		return v != 0, nil
	case uint8:
		return v != 0, nil
	case uint16:
		return v != 0, nil
	case uint32:
		return v != 0, nil
	case uint64:
		return v != 0, nil
	case float32:
		return v != 0, nil
	case float64:
		return v != 0, nil
	default:
		return false, fmt.Errorf("cannot convert %T to bool", value)
	}
}