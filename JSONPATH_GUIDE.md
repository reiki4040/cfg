# JSONPath Support Guide

## Overview

cfg now supports JSONPath for extracting specific values from JSON stored in Parameter Store. This feature enables:

- **Reduced API Calls**: Store multiple configuration values in a single JSON parameter
- **Nested Structure Support**: Navigate through deeply nested JSON objects
- **Type Conversion**: Automatic conversion of JSON types to YAML-compatible values
- **Automatic Deduplication**: Multiple references to the same parameter are deduplicated

## Basic Syntax

JSONPath references use the format:

```
${ps:/parameter/path:json.path.to.key}
```

- `/parameter/path`: The Parameter Store parameter name
- `json.path.to.key`: The JSONPath using dot-notation to navigate the JSON

## Examples

### Simple Key Extraction

```yaml
# Parameter: /app/config contains {"host": "localhost", "port": 8080}
server:
  host: ${ps:/app/config:host}          # Extracts "localhost"
  port: ${ps:/app/config:port}          # Extracts 8080
```

### Nested Object Access

```yaml
# Parameter: /db/config contains:
# {
#   "primary": {
#     "host": "db.primary.example.com",
#     "port": 5432
#   }
# }

database:
  host: ${ps:/db/config:primary.host}   # Extracts "db.primary.example.com"
  port: ${ps:/db/config:primary.port}   # Extracts 5432
```

### Deeply Nested Paths

```yaml
# Parameter: /app/secrets contains:
# {
#   "services": {
#     "api": {
#       "v1": {
#         "endpoints": {
#           "auth": {
#             "client_id": "abc123",
#             "client_secret": "secret"
#           }
#         }
#       }
#     }
#   }
# }

api:
  client_id: ${ps:/app/secrets:services.api.v1.endpoints.auth.client_id}
  client_secret: ${ps:/app/secrets:services.api.v1.endpoints.auth.client_secret}
```

## Features

### Automatic Deduplication

When multiple references point to the same parameter, cfg automatically deduplicates them:

```yaml
# Parameter: /app/db-config
database:
  host: ${ps:/app/db-config:primary.host}    # API call #1
  port: ${ps:/app/db-config:primary.port}    # Same parameter
  user: ${ps:/app/db-config:primary.user}    # Same parameter
  password: ${ps:/app/db-config:primary.password}  # Same parameter
```

All four references result in only **one API call** to Parameter Store.

### Caching

Extracted JSON objects are cached with a 5-minute TTL (Time To Live):

- Subsequent references to the same parameter use the cached version
- Cache is automatically invalidated after TTL expires
- Cache can be manually cleared if needed

### Type Conversion

JSON types are automatically converted to YAML-compatible values:

| JSON Type | YAML Value | Example |
|-----------|-----------|---------|
| String | String | `"hello"` → `hello` |
| Number | Number | `42` → `42` |
| Boolean | Boolean | `true` → `true` |
| null | String (empty) | `null` → `` |
| Object | String (JSON) | `{"key": "value"}` → `{"key":"value"}` |
| Array | String (JSON) | `[1, 2, 3]` → `[1,2,3]` |

## Error Handling

### Common Errors

#### Key Not Found

```
json_error: key not found in JSON (JSONPath: database.host, available keys: [primary, replica, pool])
```

**Solution**: Check the exact key names in your JSON parameter. Use the available keys list to find the correct path.

#### Type Mismatch

```
json_error: cannot traverse key 'host' on string type (JSONPath: primary.host.address)
```

**Solution**: The value at `primary.host` is a string, not an object. You cannot navigate further into it.

#### Invalid JSONPath Format

```
json_error: invalid JSONPath format: ..invalid
```

**Solution**: JSONPath must be dot-separated identifiers. Valid characters are: `a-z`, `A-Z`, `0-9`, `_`. No double dots, leading/trailing dots allowed.

### Debug Logging

Enable debug logging to trace JSONPath resolution:

```go
resolver.SetLogLevel(cfg.LogLevelDebug)

// Output:
// [cfg] Fetching parameter: /app/db-config (JSONPath: primary.host)
// [cfg] Cache miss for parameter: /app/db-config
// [cfg] Parsing JSON from parameter /app/db-config (size: 452 bytes)
// [cfg] JSONPath traversal: at key 'primary' found type object
// [cfg] Successfully extracted value from /app/db-config::primary.host
```

## CLI Usage

### Get Command with JSONPath

```bash
# Extract a single value
cfgctl get /app/config --jsonpath=database.host

# Extract with JSON formatting
cfgctl get /app/config --jsonpath=server --format=json
```

### List Parameters

```bash
# List parameters with values
cfgctl list /app --values

# Show parameter structure (for JSON parameters)
cfgctl list /app --values --format=json
```

## Performance Considerations

### API Call Reduction

JSONPath feature significantly reduces API calls:

- **Traditional**: 8 references → 8 API calls
- **JSONPath**: 8 references to same parameter → 1 API call (87.5% reduction)

### Extraction Performance

Benchmark results on typical configurations:

- Reference extraction: ~20µs per YAML
- JSONPath extraction: ~800ns per path
- Cache lookup: ~51ns per entry

### Memory Usage

JSON caching adds minimal memory overhead:

- Cache size increases by ~5-10% for typical configurations
- Cached objects are automatically expired after 5 minutes
- Cache can be manually cleared when needed

## Backward Compatibility

Traditional Parameter Store references (without JSONPath) are fully supported and unchanged:

```yaml
# Traditional reference (still works)
api_key: ${ps:/app/secrets/api-key}

# JSONPath reference (new feature)
db_host: ${ps:/app/config:database.host}

# Mixed usage is supported
config:
  traditional: ${ps:/app/value}
  jsonpath: ${ps:/app/config:nested.key}
  env: ${env:VARIABLE}
```

## Stage Placeholder Support

JSONPath works seamlessly with stage placeholders:

```yaml
# Parameter: /app/{stage}/config:database.host
# At runtime {stage} is replaced with current stage
database:
  host: ${ps:/app/{stage}/config:database.host}
```

## Troubleshooting

### Q: Why am I getting "key not found" error?

**A**: The JSON key name doesn't exist. Check:
1. Exact key names (case-sensitive)
2. Available keys listed in error message
3. JSON structure in Parameter Store

### Q: Can I extract array elements?

**A**: Not with dot-notation. JSONPath currently supports object navigation only.

**Solution**: Store arrays as JSON string and parse manually if needed.

### Q: Why is extraction slow?

**A**: Common causes:
1. **Cache miss**: First access to a parameter always requires parsing. Subsequent accesses use cache.
2. **Disabled caching**: Verify cache is enabled in Resolver.
3. **Large JSON**: Very large JSON objects may take longer to parse.

**Solution**: Use debug logging to identify bottlenecks.

### Q: How do I clear the cache?

**A**: Programmatically:
```go
resolver.jsonCache.Clear()
```

Automatic clearing: Cache expires after 5 minutes (TTL).

## Advanced Usage

### Multiple Path Extraction

Extract multiple paths from the same JSON:

```go
helper := cfg.NewJSONPathHelper()
paths := []string{"database.host", "database.port", "server.endpoint"}
results, err := helper.ExtractMultiplePaths(jsonStr, paths, "/param")
```

### JSON Formatting

Pretty-print JSON values:

```go
formatted, err := cfg.FormatJSONOutput(jsonString)
```

### Available Keys Discovery

Find all available keys in a JSON object:

```go
keys, err := cfg.GetAvailableKeysFromJSON(jsonString)
// Returns: ["database", "server", "cache", ...]
```

## Best Practices

1. **Use JSONPath for related values**: Store related configuration values in one JSON parameter
2. **Cache management**: Let automatic TTL manage cache expiration in most cases
3. **Error handling**: Always check errors when using JSONPath
4. **Documentation**: Document your JSON schema in parameter store descriptions
5. **Logging**: Enable debug logging during troubleshooting
6. **Testing**: Test JSONPath references with various JSON structures

## See Also

- [Configuration Examples](examples/config-jsonpath.yaml)
- [API Reference](JSONPATH_API.md) (if available)
- [Performance Benchmarks](#performance-considerations)
