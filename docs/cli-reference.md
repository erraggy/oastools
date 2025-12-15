# CLI Reference

Complete command-line reference for oastools.

## Global Usage

```bash
oastools <command> [options] [arguments]
```

## Commands Overview

| Command | Description |
|---------|-------------|
| `validate` | Validate an OpenAPI specification file or URL |
| `fix` | Automatically fix common validation errors |
| `parse` | Parse and display an OpenAPI specification |
| `convert` | Convert between OpenAPI specification versions |
| `join` | Join multiple OpenAPI specifications |
| `diff` | Compare two OpenAPI specifications |
| `generate` | Generate Go code from an OpenAPI specification |
| `overlay` | Apply OpenAPI Overlay transformations |
| `version` | Show version information |
| `help` | Show help information |

## validate

Validate an OpenAPI specification file, URL, or stdin against the specification version it declares.

### Synopsis

```bash
oastools validate [flags] <file|url|->
```

### Flags

| Flag | Description |
|------|-------------|
| `--strict` | Enable stricter validation beyond spec requirements |
| `--no-warnings` | Suppress warning messages (only show errors) |
| `-s, --source-map` | Include line numbers in output (IDE-friendly format) |
| `-q, --quiet` | Quiet mode: only output validation result, no diagnostic messages |
| `--format` | Output format: text, json, or yaml (default: "text") |
| `-h, --help` | Display help for validate command |

### Examples

```bash
# Validate a local YAML file
oastools validate openapi.yaml

# Validate a local JSON file
oastools validate openapi.json

# Validate from a URL
oastools validate https://example.com/api/openapi.yaml

# Enable strict mode (treats warnings as errors)
oastools validate --strict openapi.yaml

# Show only errors, suppress warnings
oastools validate --no-warnings openapi.yaml

# Combine flags
oastools validate --strict --no-warnings openapi.yaml

# Output as JSON for programmatic processing
oastools validate --format json openapi.yaml | jq '.valid'

# Output as YAML
oastools validate --format yaml openapi.yaml

# Read from stdin (for pipelines)
cat openapi.yaml | oastools validate -

# Pipeline with quiet mode
cat openapi.yaml | oastools validate -q -
```

### Pipelining

The validate command supports shell pipelines:

- Use `-` as the file path to read from stdin
- Use `--quiet/-q` to suppress diagnostic output for clean pipelining
- Use `--format json/yaml` for structured output that can be parsed by other tools

```bash
# Validate and extract just the valid field
cat openapi.yaml | oastools validate --format json - | jq '.valid'

# Chain with other tools
curl -s https://example.com/openapi.yaml | oastools validate -q -
```

### Output Format

```
OpenAPI Specification Validator
================================

oastools version: v1.17.1
Specification: openapi.yaml
OAS Version: 3.0.3
Source Size: 2.5 KB
Paths: 5
Operations: 12
Schemas: 8
Load Time: 125ms
Total Time: 140ms

Errors (2):
  ✗ paths./users.get.responses: missing required field '200' or 'default'
    Spec: https://spec.openapis.org/oas/v3.0.3.html#responses-object
  ✗ components.schemas.User.properties.id: missing required 'type' field
    Spec: https://spec.openapis.org/oas/v3.0.3.html#schema-object

Warnings (1):
  ⚠ paths./users/{id}.get: Operation should have a description or summary
    Spec: https://spec.openapis.org/oas/v3.0.3.html#operation-object

✗ Validation failed: 2 error(s), 1 warning(s)
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Validation passed |
| 1 | Validation failed (errors found) |

---

## fix

Automatically fix common validation errors in an OpenAPI specification file, URL, or stdin.

### Synopsis

```bash
oastools fix [flags] <file|url|->
```

### Description

The fix command automatically corrects common validation errors in OpenAPI specifications. Currently supported fixes:

- **Missing path parameters**: Adds missing path parameters (e.g., `{userId}`) that are referenced in the path but not declared in the parameters list

### Flags

| Flag | Description |
|------|-------------|
| `--infer` | Infer parameter types from naming conventions (e.g., `userId` → integer) |
| `-s, --source-map` | Include line numbers in output (IDE-friendly format) |
| `-o, --output` | Output file path (default: stdout) |
| `-q, --quiet` | Quiet mode: only output the fixed document, no diagnostic messages |
| `-h, --help` | Display help for fix command |

### Type Inference

When `--infer` is enabled, parameter types are inferred from naming conventions:

| Pattern | Type | Format |
|---------|------|--------|
| Names ending in `id`, `Id`, `ID` | `integer` | - |
| Names containing `uuid`, `guid` | `string` | `uuid` |
| All other names | `string` | - |

### Examples

```bash
# Fix a local file and output to stdout
oastools fix openapi.yaml

# Fix and write to a specific file
oastools fix openapi.yaml -o fixed.yaml

# Fix with type inference
oastools fix --infer openapi.yaml -o fixed.yaml

# Fix from a URL
oastools fix https://example.com/api/openapi.yaml -o fixed.yaml

# Fix from stdin (for pipelines)
cat openapi.yaml | oastools fix - > fixed.yaml

# Pipeline: fix then validate
oastools fix api.yaml | oastools validate -q -

# Pipeline with quiet mode
cat openapi.yaml | oastools fix -q - | oastools validate -q -
```

### Pipelining

The fix command supports shell pipelines:

- Use `-` as the file path to read from stdin
- Use `--quiet/-q` to suppress diagnostic output for clean pipelining
- Output goes to stdout by default (use `-o` for file output)

```bash
# Fix and pipe to validate
cat openapi.yaml | oastools fix -q - | oastools validate -q -

# Chain with curl
curl -s https://example.com/openapi.yaml | oastools fix -q - > fixed.yaml
```

### Output Format

```
OpenAPI Specification Fixer
===========================

oastools version: v1.17.1
Specification: openapi.yaml
OAS Version: 3.0.3
Source Size: 2.5 KB
Paths: 5
Operations: 12
Schemas: 8
Load Time: 125ms
Total Time: 140ms

Fixes Applied (3):
  ✓ paths./users/{userId}.get.parameters: Added missing path parameter 'userId' (type: string)
  ✓ paths./projects/{projectId}.get.parameters: Added missing path parameter 'projectId' (type: integer)
  ✓ paths./docs/{documentUuid}.get.parameters: Added missing path parameter 'documentUuid' (type: string, format: uuid)

✓ Fixed: 3 issue(s) corrected
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Fix completed successfully |
| 1 | Fix failed (parse error or invalid input) |

---

## parse

Parse and output OpenAPI document structure and metadata.

### Synopsis

```bash
oastools parse [flags] <file|url|->
```

### Flags

| Flag | Description |
|------|-------------|
| `--resolve-refs` | Resolve external $ref references |
| `--validate-structure` | Validate document structure during parsing |
| `-q, --quiet` | Quiet mode: only output the document, no diagnostic messages |
| `-h, --help` | Display help for parse command |

### Examples

```bash
# Parse a local file
oastools parse openapi.yaml

# Parse from a URL
oastools parse https://petstore.swagger.io/v2/swagger.yaml

# Parse with external reference resolution
oastools parse --resolve-refs openapi.yaml

# Parse with structure validation
oastools parse --validate-structure openapi.yaml

# Combine both options
oastools parse --resolve-refs --validate-structure openapi.yaml

# Read from stdin (for pipelines)
cat openapi.yaml | oastools parse -

# Pipeline with quiet mode (output only JSON)
cat openapi.yaml | oastools parse -q -
```

### Pipelining

The parse command supports shell pipelines:

- Use `-` as the file path to read from stdin
- Use `--quiet/-q` to suppress diagnostic output and get clean JSON output

```bash
# Parse and pipe to jq for processing
cat openapi.yaml | oastools parse -q - | jq '.info.title'

# Chain with other tools
curl -s https://example.com/openapi.yaml | oastools parse -q -
```

### Output Format

```
OpenAPI Specification Parser
============================

oastools version: v1.17.1
Specification: petstore.yaml
OAS Version: 3.0.3
Source Size: 15.2 KB
Paths: 8
Operations: 15
Schemas: 12
Load Time: 45ms

Document Type: OpenAPI 3.x
Title: Petstore API
Description: A sample API for a pet store
Version: 1.0.0
Servers: 2
Paths: 8

Raw Data (JSON):
{
  "openapi": "3.0.3",
  "info": {
    "title": "Petstore API",
    ...
  }
}

Parsing completed successfully!
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Parsing succeeded |
| 1 | Parsing failed (errors found) |

---

## convert

Convert an OpenAPI specification from one version to another.

### Synopsis

```bash
oastools convert [flags] <file|url|->
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--target` | `-t` | Target OAS version (required). Examples: "3.0.3", "2.0", "3.1.0" |
| `--output` | `-o` | Output file path (default: stdout) |
| `--strict` | | Fail on any conversion issues (even warnings) |
| `--no-warnings` | | Suppress warning and info messages |
| `--source-map` | `-s` | Include line numbers in output (IDE-friendly format) |
| `-q, --quiet` | | Quiet mode: only output the document, no diagnostic messages |
| `-h, --help` | | Display help for convert command |

### Supported Conversions

| Source | Target | Notes |
|--------|--------|-------|
| OAS 2.0 | OAS 3.0.x | Full support |
| OAS 2.0 | OAS 3.1.x | Full support |
| OAS 2.0 | OAS 3.2.x | Full support |
| OAS 3.x | OAS 2.0 | Some features cannot be converted |
| OAS 3.x | OAS 3.y | Version updates supported |

### Examples

```bash
# Convert Swagger 2.0 to OpenAPI 3.0.3
oastools convert -t 3.0.3 swagger.yaml -o openapi.yaml

# Convert from URL
oastools convert -t 3.0.3 https://example.com/swagger.yaml -o openapi.yaml

# Convert to stdout (for piping)
oastools convert -t 3.0.3 swagger.yaml > openapi.yaml

# Convert OpenAPI 3.x back to Swagger 2.0
oastools convert -t 2.0 openapi.yaml -o swagger.yaml

# Strict mode: fail on any conversion issues
oastools convert --strict -t 3.0.3 swagger.yaml -o openapi.yaml

# Suppress informational messages
oastools convert --no-warnings -t 3.0.3 swagger.yaml -o openapi.yaml

# Update OpenAPI 3.0.x to 3.1.0
oastools convert -t 3.1.0 openapi-3.0.yaml -o openapi-3.1.yaml

# Read from stdin (for pipelines)
cat swagger.yaml | oastools convert -t 3.0.3 - -o openapi.yaml

# Pipeline with quiet mode (output to stdout)
cat swagger.yaml | oastools convert -q -t 3.0.3 - > openapi.yaml
```

### Pipelining

The convert command supports shell pipelines:

- Use `-` as the file path to read from stdin
- Use `--quiet/-q` to suppress diagnostic output for clean pipelining
- Output goes to stdout by default (use `-o` for file output)

```bash
# Convert and write to file
cat swagger.yaml | oastools convert -q -t 3.0.3 - > openapi.yaml

# Chain conversions
curl -s https://example.com/swagger.yaml | oastools convert -q -t 3.0.3 - > openapi.yaml
```

### Output Format

```
OpenAPI Specification Converter
===============================

oastools version: v1.17.1
Specification: swagger.yaml
Source Version: 2.0
Target Version: 3.0.3
Source Size: 8.5 KB
Paths: 5
Operations: 12
Schemas: 8
Load Time: 85ms
Total Time: 95ms

Conversion Issues (3):
  [INFO] servers: Converted host 'api.example.com' to server URL 'https://api.example.com/v1'
  [WARNING] parameters.filter.allowEmptyValue: OAS 3.x does not support allowEmptyValue; dropped
  [INFO] securityDefinitions: Converted to components.securitySchemes

✓ Conversion successful (2 info, 1 warnings)

Output written to: openapi.yaml
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Conversion successful |
| 1 | Conversion failed (critical issues) |

---

## join

Join multiple OpenAPI specification files into a single document.

### Synopsis

```bash
oastools join [flags] <file1> <file2> [file3...]
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--output` | `-o` | Output file path (required) |
| `--path-strategy` | | Collision strategy for paths |
| `--schema-strategy` | | Collision strategy for schemas/definitions |
| `--component-strategy` | | Collision strategy for other components |
| `--semantic-dedup` | | Enable semantic deduplication to consolidate identical schemas |
| `--no-merge-arrays` | | Don't merge arrays (servers, security, etc.) |
| `--no-dedup-tags` | | Don't deduplicate tags by name |
| `--source-map` | `-s` | Include line numbers in output (IDE-friendly format) |
| `-h, --help` | | Display help for join command |

### Collision Strategies

| Strategy | Description |
|----------|-------------|
| `accept-left` | Keep the first value when collisions occur |
| `accept-right` | Keep the last value when collisions occur (overwrite) |
| `fail` | Fail with an error on any collision |
| `fail-on-paths` | Fail only on path collisions, allow schema collisions |

### Examples

```bash
# Basic join of two files
oastools join -o merged.yaml base.yaml extension.yaml

# Join multiple files
oastools join -o api.yaml users.yaml posts.yaml comments.yaml

# Keep first value on collision
oastools join --path-strategy accept-left -o merged.yaml base.yaml ext.yaml

# Keep last value on collision (overwrite)
oastools join --path-strategy accept-right -o merged.yaml base.yaml ext.yaml

# Fail on any collision
oastools join --path-strategy fail -o merged.yaml base.yaml ext.yaml

# Different strategies for different components
oastools join \
  --path-strategy fail \
  --schema-strategy accept-left \
  --component-strategy accept-right \
  -o merged.yaml base.yaml ext.yaml

# Don't merge arrays
oastools join --no-merge-arrays -o merged.yaml base.yaml ext.yaml

# Don't deduplicate tags
oastools join --no-dedup-tags -o merged.yaml base.yaml ext.yaml

# Enable semantic deduplication to consolidate identical schemas
oastools join --semantic-dedup -o merged.yaml api1.yaml api2.yaml
```

### Output Format

```
OpenAPI Specification Joiner
============================

oastools version: v1.17.1
Successfully joined 3 specification files
Output: merged.yaml
OAS Version: 3.0.3
Paths: 12
Operations: 28
Schemas: 15
Total Time: 250ms

Collisions resolved: 2

Warnings (1):
  - Schema 'User' collision resolved with accept-left strategy

✓ Join completed successfully!
```

**With Semantic Deduplication:**

When `--semantic-dedup` is enabled, the output includes deduplication information:

```
Warnings (2):
  - Schema 'User' collision resolved with accept-left strategy
  - semantic deduplication: consolidated 3 duplicate definition(s)
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Join successful |
| 1 | Join failed (collision with fail strategy, version mismatch, etc.) |

### Notes

- All input files must be the same major OAS version (2.0 or 3.x)
- The output uses the version and format (JSON/YAML) of the first input file
- Info section is taken from the first document
- Output file is written with restrictive permissions (0600) for security
- Warning is displayed if output file already exists (will be overwritten)
- Semantic deduplication identifies structurally identical schemas and consolidates them, reducing document size

---

## diff

Compare two OpenAPI specification files or URLs and report differences.

### Synopsis

```bash
oastools diff [flags] <source> <target>
```

### Flags

| Flag | Description |
|------|-------------|
| `--breaking` | Enable breaking change detection and categorization |
| `--no-info` | Exclude informational changes from output |
| `-s, --source-map` | Include line numbers in output (IDE-friendly format) |
| `--format` | Output format: text, json, or yaml (default: "text") |
| `-h, --help` | Display help for diff command |

### Examples

```bash
# Simple diff (all changes)
oastools diff api-v1.yaml api-v2.yaml

# Breaking change detection
oastools diff --breaking api-v1.yaml api-v2.yaml

# Exclude informational changes
oastools diff --breaking --no-info api-v1.yaml api-v2.yaml

# Compare from URLs
oastools diff \
  https://example.com/api/v1/openapi.yaml \
  https://example.com/api/v2/openapi.yaml

# Compare local with remote
oastools diff local-api.yaml https://example.com/api/openapi.yaml

# Output as JSON for programmatic processing
oastools diff --format json --breaking api-v1.yaml api-v2.yaml | jq '.HasBreakingChanges'

# Output as YAML
oastools diff --format yaml api-v1.yaml api-v2.yaml
```

### Output Format (Simple Mode)

The diff output includes document statistics showing paths, operations, and schemas for both source and target. The layout adapts automatically: side-by-side columns for short paths, single column for longer paths.

```
OpenAPI Specification Diff
==========================

oastools version: v1.22.0

Source: api-v1.yaml                      Target: api-v2.yaml
  OAS Version: 3.0.3                       OAS Version: 3.0.3
  Source Size: 12.5 KB                     Target Size: 14.2 KB
  Paths: 5                                 Paths: 6
  Operations: 12                           Operations: 15
  Schemas: 8                               Schemas: 10

Total Time: 125ms

Changes (8):
  Path '/users/{id}' removed
  Path '/posts' added
  Operation GET '/users' modified
  Parameter 'limit' added to GET '/users'
  Schema 'User' modified
  ...
```

### Output Format (Breaking Mode)

```
OpenAPI Specification Diff
==========================

oastools version: v1.22.0

Source: api-v1.yaml                      Target: api-v2.yaml
  OAS Version: 3.0.3                       OAS Version: 3.0.3
  Source Size: 12.5 KB                     Target Size: 14.2 KB
  Paths: 5                                 Paths: 6
  Operations: 12                           Operations: 15
  Schemas: 8                               Schemas: 10

Total Time: 125ms

Endpoint Changes (2):
  [CRITICAL] /users/{id}: Endpoint removed
  [INFO] /posts: Endpoint added

Operation Changes (1):
  [WARNING] GET /users: Operation deprecated

Parameter Changes (2):
  [ERROR] GET /users: Required parameter 'limit' added
  [INFO] GET /users: Optional parameter 'filter' added

Summary:
  Total changes: 5
  ⚠️  Breaking changes: 2
  Warnings: 1
  Info: 2
```

### Severity Levels (Breaking Mode)

| Severity | Impact | Examples |
|----------|--------|----------|
| CRITICAL | API consumers WILL break | Removed endpoints, operations |
| ERROR | API consumers MAY break | Type changes, new required parameters |
| WARNING | Consumers SHOULD be aware | Deprecated operations, new optional fields |
| INFO | Non-breaking changes | Added endpoints, documentation updates |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | No differences found (or no breaking changes in `--breaking` mode) |
| 1 | Differences found (or breaking changes in `--breaking` mode) |

### Notes

- Both specifications must be valid OpenAPI documents
- Cross-version comparison (2.0 vs 3.x) is supported with limitations
- Breaking change detection helps identify backward compatibility issues
- Use in CI/CD pipelines to prevent accidental breaking changes

---

## generate

Generate idiomatic Go code (clients, servers, or types) from an OpenAPI specification.

### Synopsis

```bash
oastools generate [flags] <file|url|->
```

### Description

The generate command creates Go code from OpenAPI specifications. It can generate:
- **HTTP clients** with methods for each API operation
- **Server interfaces** defining the endpoints your implementation must satisfy
- **Type definitions** for all schemas in the specification

Generated code follows Go idioms, includes proper error handling, and is suitable for production use.

### Flags

| Flag | Description |
|------|-------------|
| `-o, --output string` | Output directory for generated files **(required)** |
| `-p, --package string` | Go package name for generated code (default: "api") |
| `--client` | Generate HTTP client code |
| `--server` | Generate server interface code |
| `--types` | Generate type definitions from schemas (default: true) |
| `--no-pointers` | Don't use pointer types for optional fields |
| `--no-validation` | Don't include validation tags in generated code |
| `-s, --source-map` | Include line numbers in output (IDE-friendly format) |
| `--strict` | Fail on any generation issues (even warnings) |
| `--no-warnings` | Suppress warning and info messages |
| `-h, --help` | Display help for generate command |

**Security Generation Flags:**

| Flag | Description |
|------|-------------|
| `--no-security` | Don't generate security helper functions (default: false, security is generated) |
| `--oauth2-flows` | Generate full OAuth2 token flow helpers |
| `--credential-mgmt` | Generate credential management interfaces |
| `--security-enforce` | Generate security enforcement middleware |
| `--oidc-discovery` | Generate OpenID Connect discovery client |
| `--no-readme` | Don't generate README.md documentation (default: false, README is generated) |

**File Splitting Flags (for large APIs):**

| Flag | Description |
|------|-------------|
| `--max-lines-per-file int` | Maximum lines per generated file (default: 2000) |
| `--max-types-per-file int` | Maximum types per generated file (default: 200) |
| `--max-ops-per-file int` | Maximum operations per generated file (default: 100) |
| `--split-by-tag` | Split files by operation tag (default: true) |
| `--split-by-path` | Split files by path prefix (default: true) |

### Examples

**Generate HTTP client:**

```bash
oastools generate --client -o ./client -p petstore openapi.yaml
```

**Generate server interface:**

```bash
oastools generate --server -o ./server -p petstore openapi.yaml
```

**Generate both client and server:**

```bash
oastools generate --client --server -o ./generated -p myapi openapi.yaml
```

**Generate types only:**

```bash
oastools generate --types -o ./models openapi.yaml
```

**Generate from URL:**

```bash
oastools generate --client -o ./client https://example.com/api/openapi.yaml
```

**Generate client with OAuth2 flows:**

```bash
oastools generate --client --oauth2-flows -o ./client -p api openapi.yaml
```

**Generate with all security features:**

```bash
oastools generate --client --oauth2-flows --credential-mgmt --oidc-discovery --readme \
  -o ./client -p api openapi.yaml
```

**Generate server with security enforcement:**

```bash
oastools generate --server --security-enforce -o ./server -p api openapi.yaml
```

**Generate with file splitting for large APIs:**

```bash
oastools generate --client --max-lines-per-file 1500 --split-by-tag \
  -o ./client -p api large-api.yaml
```

### Output

The command generates the following files in the output directory:

- **`types.go`** - Struct definitions generated from schema definitions
  - Includes JSON marshaling/unmarshaling
  - Validation tags if `--no-validation` is not set
  - Comments from schema descriptions

- **`client.go`** (when `--client` is used)
  - HTTP client struct with configurable base URL
  - Methods for each operation in the specification
  - Automatic request/response marshaling
  - Comprehensive error handling

- **`server.go`** (when `--server` is used)
  - Server interface defining all endpoints
  - Request/response types for type safety
  - Framework-agnostic (implement the interface in your chosen framework)

- **`security_helpers.go`** (generated by default with `--client`, disable with `--no-security`)
  - ClientOption functions for each security scheme
  - API key helpers (header, query, cookie)
  - HTTP basic/bearer authentication helpers
  - OAuth2 token helpers

- **`{scheme}_oauth2.go`** (when `--oauth2-flows` is used)
  - OAuth2Config and OAuth2Token types
  - OAuth2Client with flow-specific methods
  - Authorization code, client credentials, password flows
  - PKCE support (RFC 7636) for secure authorization code flows
  - Token refresh and auto-refresh support

- **`credentials.go`** (when `--credential-mgmt` is used)
  - CredentialProvider interface
  - MemoryCredentialProvider for testing
  - EnvCredentialProvider for environment variables
  - CredentialChain for fallback providers

- **`security_enforce.go`** (when `--security-enforce` is used)
  - SecurityRequirement struct
  - OperationSecurityRequirements map
  - SecurityValidator for request validation
  - RequireSecurityMiddleware

- **`oidc_discovery.go`** (when `--oidc-discovery` is used)
  - OIDCConfiguration struct
  - OIDCDiscoveryClient for .well-known discovery
  - NewOAuth2ClientFromOIDC helper

- **`README.md`** (generated by default, disable with `--no-readme`)
  - API overview and version info
  - Generated file descriptions
  - Security configuration examples
  - Regeneration command

### Type Mapping

OpenAPI types are mapped to Go types as follows:

| OpenAPI Type | Go Type | Notes |
|--------------|---------|-------|
| `string` | `string` | Respects format hints (uuid, email, date-time, etc.) |
| `integer` (format: int32) | `int32` | |
| `integer` | `int64` | Default for integers |
| `number` (format: float) | `float32` | |
| `number` | `float64` | Default for numbers |
| `boolean` | `bool` | |
| `array` | `[]T` | T depends on item type |
| `object` | `struct` | Generated with fields from properties |
| `null` (OAS 3.1+) | `*T` | Using pointers for optional fields |

Optional fields (not in required array) use pointer types when `--no-pointers` is not set.

### Notes

- **Format Preservation**: Input files determine output format (JSON → JSON, YAML → YAML)
- **At least one generation mode required**: If none of `--client`, `--server`, or `--types` are specified, types generation is enabled by default
- **Package naming**: Go package names must be valid identifiers (lowercase, no hyphens)
- **Schema support**: Generates code for all OAS versions (2.0, 3.0.x, 3.1.x, 3.2.0)
- **Validation tags**: Generated structs include `validate` struct tags for integration with validation libraries

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Code generation successful |
| 1 | Output directory is required |
| 2 | At least one generation mode must be enabled |
| 3 | Generation failed (file read, parsing, or code generation error) |

---

## overlay

Apply OpenAPI Overlay Specification transformations to OpenAPI documents.

### Synopsis

```bash
oastools overlay <subcommand> [flags]
```

### Subcommands

| Subcommand | Description |
|------------|-------------|
| `apply` | Apply an overlay to an OpenAPI specification |
| `validate` | Validate an overlay document |

### overlay apply

Apply an overlay transformation to an OpenAPI specification.

```bash
oastools overlay apply [flags]
```

#### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--spec` | `-s` | Path to the OpenAPI specification file (required) |
| `--overlay` | | Path to the overlay file (or positional argument) |
| `--output` | `-o` | Output file path (default: stdout) |
| `--strict` | | Fail if any target matches nothing |
| `--dry-run` | | Preview changes without applying |
| `--quiet` | `-q` | Suppress diagnostic output |
| `-h, --help` | | Display help |

#### Examples

```bash
# Apply overlay and write to file
oastools overlay apply --spec openapi.yaml --overlay changes.yaml -o result.yaml

# Apply overlay to stdout
oastools overlay apply --spec openapi.yaml --overlay changes.yaml

# Preview changes without applying
oastools overlay apply --spec openapi.yaml --overlay changes.yaml --dry-run

# Strict mode (fail if targets don't match)
oastools overlay apply --spec openapi.yaml --overlay changes.yaml --strict

# From stdin
cat openapi.yaml | oastools overlay apply --spec - --overlay changes.yaml

# Quiet mode for pipelines
oastools overlay apply -q --spec openapi.yaml --overlay changes.yaml > result.yaml
```

#### Output Format

```
OpenAPI Overlay Application
============================

oastools version: v1.24.0
Overlay: changes.yaml
Spec: openapi.yaml
OAS Version: 3.0.3

Actions applied: 3
Actions skipped: 0

Changes:
  - update 1 nodes at $.info
  - update 5 nodes at $.paths.*.get
  - remove 2 nodes at $.paths[?@.x-internal==true]

✓ Overlay applied successfully
```

#### Dry-Run Output

```
OpenAPI Overlay Dry Run
=======================

oastools version: v1.24.0
Overlay: changes.yaml
Spec: openapi.yaml
OAS Version: 3.0.3

Would apply: 3 actions
Would skip: 0 actions

Proposed changes:
  - update 1 nodes at $.info
  - update 5 nodes at $.paths.*.get
  - remove 2 nodes at $.paths[?@.x-internal==true]
```

### overlay validate

Validate an overlay document against the OpenAPI Overlay Specification.

```bash
oastools overlay validate [flags] <overlay-file>
```

#### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--quiet` | `-q` | Suppress diagnostic output |
| `-h, --help` | | Display help |

#### Examples

```bash
# Validate an overlay file
oastools overlay validate overlay.yaml

# Quiet mode
oastools overlay validate -q overlay.yaml
```

#### Output Format

```
OpenAPI Overlay Validator
=========================

oastools version: v1.24.0
Overlay: overlay.yaml
Title: Update API Metadata
Version: 1.0.0
Actions: 5

✓ Overlay is valid
```

#### Validation Errors

```
OpenAPI Overlay Validator
=========================

oastools version: v1.24.0
Overlay: invalid-overlay.yaml

Errors (2):
  ✗ info.version: version is required
  ✗ actions: at least one action is required

✗ Validation failed: 2 error(s)
```

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success (overlay applied or validated) |
| 1 | Failure (errors in overlay or application) |

### JSONPath Support

The overlay package supports these JSONPath expressions:

| Expression | Description | Example |
|------------|-------------|---------|
| `$.field` | Root field access | `$.info` |
| `$.a.b.c` | Nested field access | `$.info.title` |
| `$['field']` | Bracket notation | `$.paths['/users']` |
| `$.*` | Wildcard (all children) | `$.paths.*` |
| `$[0]` | Array index | `$.servers[0]` |
| `$..field` | Recursive descent | `$..description` |
| `$[?@.x==y]` | Filter expression | `$.paths[?@.x-internal==true]` |
| `$[?@ && @]` | Compound AND filter | `$.paths[?@.deprecated==true && @.x-internal==true]` |
| `$[?@ \|\| @]` | Compound OR filter | `$.paths[?@.deprecated==true \|\| @.x-obsolete==true]` |

---

## version

Display oastools version and build information.

### Synopsis

```bash
oastools version
oastools -v
oastools --version
```

### Output

```
oastools v1.17.1
commit: 540e27a
built: 2025-12-06T20:05:42Z
go: go1.24.0
```

The version command displays:
- **version**: The release version
- **commit**: The git commit hash of the build
- **built**: The build timestamp (RFC3339 format)
- **go**: The Go version used to compile the binary

---

## help

Display help information.

### Synopsis

```bash
oastools help
oastools -h
oastools --help
oastools <command> --help
```

### Output

```
oastools - OpenAPI Specification Tools

Usage:
  oastools <command> [options]

Commands:
  validate    Validate an OpenAPI specification file or URL
  fix         Automatically fix common validation errors
  convert     Convert between OpenAPI specification versions
  diff        Compare two OpenAPI specifications and detect changes
  generate    Generate Go client/server code from an OpenAPI specification
  join        Join multiple OpenAPI specification files
  overlay     Apply OpenAPI Overlay transformations
  parse       Parse and display an OpenAPI specification file or URL
  version     Show version information
  help        Show this help message

Examples:
  oastools validate openapi.yaml
  oastools validate https://example.com/api/openapi.yaml
  oastools fix --infer api.yaml -o fixed.yaml
  oastools convert -t 3.0.3 swagger.yaml -o openapi.yaml
  oastools diff --breaking api-v1.yaml api-v2.yaml
  oastools generate --client -o ./client openapi.yaml
  oastools join -o merged.yaml base.yaml extensions.yaml
  oastools overlay apply --spec openapi.yaml --overlay changes.yaml -o result.yaml
  oastools parse https://raw.githubusercontent.com/OAI/OpenAPI-Specification/main/examples/v3.0/petstore.yaml

Run 'oastools <command> --help' for more information on a command.
```

---

## Environment Variables

oastools does not currently use any environment variables.

---

## File Format Support

| Format | Extensions | Auto-Detection |
|--------|------------|----------------|
| YAML | `.yaml`, `.yml` | Yes |
| JSON | `.json` | Yes |

The output format matches the input format (JSON input → JSON output, YAML input → YAML output).

---

## URL Support

The following commands support loading specifications from URLs:

- `validate`
- `fix`
- `parse`
- `convert`
- `diff`
- `generate`

Supported URL schemes:
- `http://`
- `https://`

Note: When loading from URLs, relative `$ref` paths resolve against the current working directory (where the CLI is executed), not relative to the remote URL location.

---

## Stdin and Pipeline Support

The following commands support reading from stdin using `-` as the file path:

- `validate`
- `fix`
- `parse`
- `convert`
- `generate`

### Pipeline Usage

```bash
# Validate from stdin
cat openapi.yaml | oastools validate -

# Parse from stdin with quiet mode
cat openapi.yaml | oastools parse -q -

# Convert from stdin to stdout
cat swagger.yaml | oastools convert -q -t 3.0.3 - > openapi.yaml

# Chain with curl
curl -s https://example.com/openapi.yaml | oastools validate -q -

# Chain multiple operations
cat swagger.yaml | oastools convert -q -t 3.0.3 - | oastools validate -q -

# Generate client from stdin
cat openapi.yaml | oastools generate --client -o ./client -
```

### Quiet Mode

Use `-q` or `--quiet` to suppress diagnostic messages for clean pipeline output:

| Command | Quiet Mode Behavior |
|---------|---------------------|
| `validate` | Only outputs validation result (no banners/stats) |
| `fix` | Only outputs the fixed document (no banners/stats) |
| `parse` | Only outputs the document JSON (no banners/stats) |
| `convert` | Only outputs the converted document (no banners/issues) |

### Structured Output

Use `--format json` or `--format yaml` for machine-readable output:

```bash
# Get validation result as JSON
oastools validate --format json openapi.yaml | jq '.valid'

# Get diff result as JSON for CI/CD
oastools diff --format json --breaking v1.yaml v2.yaml | jq '.HasBreakingChanges'
```

---

## Security Considerations

1. **External References**: Only local file references are supported for `$ref` values. HTTP(S) references are not supported for security reasons.

2. **Path Traversal**: External file references are restricted to the base directory and subdirectories to prevent path traversal attacks.

3. **Output Permissions**: The `join` command writes output files with restrictive permissions (0600).

4. **Credential Handling**: URLs may include basic authentication, but this is not recommended. Use environment-specific configuration instead.
