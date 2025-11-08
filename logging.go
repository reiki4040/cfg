package cfg

import (
	"fmt"
	"log"
	"strings"
)

// LogLevel defines the verbosity of logging
type LogLevel int

const (
	LogLevelOff LogLevel = iota
	LogLevelError
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

// Logger provides structured logging for JSONPath resolution
type Logger struct {
	level LogLevel
	debug bool
}

// NewLogger creates a new logger with the specified log level
func NewLogger(level LogLevel) *Logger {
	return &Logger{
		level: level,
		debug: level >= LogLevelDebug,
	}
}

// IsDebugEnabled returns whether debug logging is enabled
func (l *Logger) IsDebugEnabled() bool {
	return l.debug
}

// LogReferenceFetch logs the start of parameter fetch
func (l *Logger) LogReferenceFetch(paramPath string, jsonPath string) {
	if l.level < LogLevelDebug {
		return
	}

	if jsonPath != "" {
		log.Printf("[cfg] Fetching parameter: %s (JSONPath: %s)", paramPath, jsonPath)
	} else {
		log.Printf("[cfg] Fetching parameter: %s", paramPath)
	}
}

// LogJsonParsing logs JSON parsing attempt
func (l *Logger) LogJsonParsing(paramPath string, jsonSize int) {
	if l.level < LogLevelDebug {
		return
	}
	log.Printf("[cfg] Parsing JSON from parameter %s (size: %d bytes)", paramPath, jsonSize)
}

// LogJsonStructure logs the structure of parsed JSON (metadata only, no values)
func (l *Logger) LogJsonStructure(paramPath string, keys []string) {
	if l.level < LogLevelDebug || len(keys) == 0 {
		return
	}
	log.Printf("[cfg] JSON object structure at %s: keys=%v", paramPath, keys)
}

// LogJsonPathTraversal logs each step of JSONPath traversal
func (l *Logger) LogJsonPathTraversal(paramPath string, path string, foundType string) {
	if l.level < LogLevelDebug {
		return
	}
	log.Printf("[cfg] JSONPath traversal: at key '%s' found type %s (param: %s)", path, foundType, paramPath)
}

// LogJsonValueExtracted logs successful value extraction
func (l *Logger) LogJsonValueExtracted(paramPath string, jsonPath string) {
	if l.level < LogLevelDebug {
		return
	}
	log.Printf("[cfg] Successfully extracted value from %s::%s", paramPath, jsonPath)
}

// LogCacheHit logs cache hit
func (l *Logger) LogCacheHit(paramPath string) {
	if l.level < LogLevelDebug {
		return
	}
	log.Printf("[cfg] Cache hit for parameter: %s", paramPath)
}

// LogCacheMiss logs cache miss
func (l *Logger) LogCacheMiss(paramPath string) {
	if l.level < LogLevelDebug {
		return
	}
	log.Printf("[cfg] Cache miss for parameter: %s", paramPath)
}

// LogError logs an error with context
func (l *Logger) LogError(paramPath string, jsonPath string, err error) {
	if l.level < LogLevelError {
		return
	}

	var context string
	if jsonPath != "" {
		context = fmt.Sprintf("%s::%s", paramPath, jsonPath)
	} else {
		context = paramPath
	}

	log.Printf("[cfg] ERROR resolving %s: %v", context, err)
}

// LogKeyNotFound logs when a key is not found in JSON
func (l *Logger) LogKeyNotFound(paramPath string, jsonPath string, availableKeys []string) {
	if l.level < LogLevelWarn {
		return
	}

	log.Printf("[cfg] Key not found in %s for JSONPath: %s (available keys: %v)", paramPath, jsonPath, availableKeys)
}

// LogTypeMismatch logs type mismatch during traversal
func (l *Logger) LogTypeMismatch(paramPath string, jsonPath string, expectedType string, foundType string) {
	if l.level < LogLevelWarn {
		return
	}

	log.Printf("[cfg] Type mismatch in %s::%s (expected %s, got %s)", paramPath, jsonPath, expectedType, foundType)
}

// LogParseFailure logs JSON parsing failure
func (l *Logger) LogParseFailure(paramPath string, err error) {
	if l.level < LogLevelWarn {
		return
	}

	log.Printf("[cfg] Failed to parse JSON from parameter %s: %v", paramPath, err)
}

// MaskParameterValue returns a masked version of a parameter value for logging
func MaskParameterValue(value string, maxLen int) string {
	if len(value) <= maxLen {
		return "***"
	}
	// Show first and last few chars, mask the middle
	first := maxLen / 4
	last := maxLen / 4
	middle := strings.Repeat("*", maxLen/2)
	return value[:first] + middle + value[len(value)-last:]
}

// LogParameterValue logs a parameter value (with masking for large values)
func (l *Logger) LogParameterValue(paramPath string, value string) {
	if l.level < LogLevelDebug {
		return
	}

	maskedValue := MaskParameterValue(value, 50)
	log.Printf("[cfg] Parameter %s value (masked): %s", paramPath, maskedValue)
}
