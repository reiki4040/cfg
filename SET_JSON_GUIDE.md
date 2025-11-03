# cfgctl set - JSON Support Guide

## Overview

The `cfgctl set` command now supports setting JSON values in Parameter Store with validation and formatting capabilities.

## Basic Usage

### Set JSON from inline string

```bash
# Simple JSON object
cfgctl set /app/config --json='{"host":"localhost","port":5432}'

# Complex nested structure
cfgctl set /app/db-config --json='{"primary":{"host":"db.example.com","port":5432}}'
```

### Set JSON from file

```bash
# Read JSON from file
cfgctl set /app/config --json-file=config.json

# With stage placeholder
cfgctl set /app/{stage}/config --json-file=config.json --stage=prod
```

## Features

### Automatic JSON Validation

By default, JSON is automatically validated before being sent to Parameter Store:

```bash
# Valid JSON - succeeds
cfgctl set /app/config --json='{"key":"value"}'

# Invalid JSON - fails with error
cfgctl set /app/config --json='{invalid}'
# Error: JSON validation failed: invalid character 'i' looking for beginning of object key string
```

### Disable Validation (if needed)

```bash
# Skip JSON validation
cfgctl set /app/config --json='{"key":"value"}' --json-validate=false
```

### Pretty-Printed Diff

When updating an existing parameter with JSON values, the tool displays a nicely formatted diff:

```bash
cfgctl set /app/config --json-file=new-config.json

# Output:
# Parameter /app/config already exists.
# Current value (JSON):
# {
#   "host": "localhost",
#   "port": 5432
# }
#
# New value (JSON):
# {
#   "host": "db.example.com",
#   "port": 5432,
#   "replicas": [
#     "replica1.example.com",
#     "replica2.example.com"
#   ]
# }
# Are you sure you want to overwrite parameter '/app/config'? (y/N):
```

## Examples

### Database Configuration

Store database configuration as JSON:

```bash
cfgctl set /app/prod/db-config --json='{
  "primary": {
    "host": "primary.db.example.com",
    "port": 5432,
    "username": "dbuser",
    "password": "secret123"
  },
  "replica": {
    "host": "replica.db.example.com",
    "port": 5432
  },
  "pool": {
    "min_size": 5,
    "max_size": 20,
    "timeout": 30
  }
}'
```

Then extract values using JSONPath:

```bash
# Extract primary host
cfgctl get /app/prod/db-config --jsonpath=primary.host
# Output: primary.db.example.com

# Extract replica host
cfgctl get /app/prod/db-config --jsonpath=replica.host
# Output: replica.db.example.com

# Extract pool size
cfgctl get /app/prod/db-config --jsonpath=pool.max_size
# Output: 20
```

### Server Configuration from File

```bash
# Create config.json
cat > config.json << 'EOF'
{
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "timeout": 30,
    "tls": {
      "enabled": true,
      "cert_path": "/etc/certs/server.crt",
      "key_path": "/etc/certs/server.key"
    }
  },
  "logging": {
    "level": "info",
    "format": "json"
  }
}
EOF

# Set in Parameter Store
cfgctl set /app/prod/server-config --json-file=config.json --overwrite

# Extract TLS config
cfgctl get /app/prod/server-config --jsonpath=server.tls.enabled
# Output: true
```

### Multi-stage Configuration

```bash
# Use stage placeholder with JSON file
cfgctl set /app/{stage}/config --json-file=config.json --stage=dev
cfgctl set /app/{stage}/config --json-file=config.json --stage=prod
```

## Supported JSON Types

| JSON Type | Example | YAML Equivalent |
|-----------|---------|-----------------|
| String | `"hello"` | `hello` |
| Number | `42` | `42` |
| Float | `3.14` | `3.14` |
| Boolean | `true` / `false` | `true` / `false` |
| null | `null` | empty/nil |
| Object | `{"key":"value"}` | Map |
| Array | `[1, 2, 3]` | List |

## Best Practices

1. **Validate JSON locally first**
   ```bash
   # Use jq to validate JSON before uploading
   jq . config.json > /dev/null && cfgctl set /app/config --json-file=config.json
   ```

2. **Use files for complex structures**
   - Files are better for large or complex JSON
   - Easier to version control
   - Simpler to manage indentation

3. **Keep JSON organized**
   ```json
   {
     "database": {
       "primary": {...},
       "replica": {...}
     },
     "cache": {...},
     "logging": {...}
   }
   ```

4. **Use JSONPath for multiple values**
   - Store related values in one parameter
   - Reference with JSONPath to reduce API calls
   - Automatic deduplication of multiple references

5. **Leverage overwrite flag**
   ```bash
   # Auto-confirm overwrite in scripts
   cfgctl set /app/config --json-file=config.json --overwrite --no-interactive
   ```

## Error Handling

### Invalid JSON Format

```bash
cfgctl set /app/config --json='{invalid}'
# Error: JSON validation failed: invalid character 'i' looking for beginning of object key string
```

### File Not Found

```bash
cfgctl set /app/config --json-file=nonexistent.json
# Error: failed to get JSON value: failed to read JSON file nonexistent.json: no such file or directory
```

### JSON File is Too Large

Parameter Store has a 4KB limit for String parameters:

```bash
# Check file size before uploading
ls -lh config.json

# For large configurations, consider:
# 1. Splitting into multiple parameters
# 2. Using StringList type
# 3. Compressing the JSON
```

## Integration with JSONPath

Once JSON is stored in Parameter Store, use JSONPath to extract specific values:

```bash
# Set complex config
cfgctl set /app/config --json-file=complex-config.json

# Extract specific values
cfgctl get /app/config --jsonpath=database.primary.host
cfgctl get /app/config --jsonpath=server.port
cfgctl get /app/config --jsonpath=logging.level
```

## Scripting Examples

### Batch Set Multiple Parameters

```bash
#!/bin/bash

# Array of parameters
declare -A params=(
  ["/app/dev/config"]="config.dev.json"
  ["/app/prod/config"]="config.prod.json"
  ["/app/staging/config"]="config.staging.json"
)

# Set all parameters
for param in "${!params[@]}"; do
  file="${params[$param]}"
  echo "Setting $param from $file..."
  cfgctl set "$param" --json-file="$file" --overwrite --no-interactive
done
```

### Validate and Update

```bash
#!/bin/bash

CONFIG_FILE="config.json"
PARAM_PATH="/app/config"

# Validate JSON
if ! jq . "$CONFIG_FILE" > /dev/null 2>&1; then
  echo "Error: Invalid JSON in $CONFIG_FILE"
  exit 1
fi

# Set parameter
cfgctl set "$PARAM_PATH" --json-file="$CONFIG_FILE" --overwrite

# Verify by retrieving
echo "Verifying parameter..."
cfgctl get "$PARAM_PATH" --format=json
```

### Extract and Compare

```bash
#!/bin/bash

PARAM_PATH="/app/prod/config"

# Get current config
cfgctl get "$PARAM_PATH" --format=json > current.json

# Compare with new config
diff -u current.json config.json

# If differences are acceptable, update
cfgctl set "$PARAM_PATH" --json-file=config.json --overwrite
```

## CLI Reference

```
cfgctl set <parameter-path> [value]

FLAGS:
  --json string              Set value as JSON string (inline JSON)
  --json-file string         Set value as JSON from file
  --json-validate            Validate JSON format (default: true)
  --type string              Parameter type (String, SecureString, StringList)
  -S, --string              Set parameter type to String
  --SS                      Set parameter type to SecureString
  --SL                      Set parameter type to StringList
  --overwrite               Overwrite existing parameter without confirmation
  --no-interactive          Disable interactive mode
  --kms-key string          KMS key ID for SecureString parameters
  --stage string            Stage placeholder value
```

## Limitations and Notes

- **4KB limit**: Parameter Store String type has 4KB size limit
- **Type fixed to String**: JSON values are always stored as String type
- **No compression**: Large JSON objects may hit the 4KB limit
- **Validation is optional**: Can be disabled with `--json-validate=false`
- **File encoding**: JSON files must be UTF-8 encoded

## See Also

- [JSONPath Support Guide](JSONPATH_GUIDE.md)
- [cfgctl get command](README.md)
- [Parameter Store Documentation](https://docs.aws.amazon.com/systems-manager/latest/userguide/systems-manager-parameter-store.html)
