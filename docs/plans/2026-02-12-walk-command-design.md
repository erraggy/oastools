# Walk Command Design

> **Date:** 2026-02-12
> **Status:** Approved

## Overview

A structured query CLI command for exploring OAS documents. Provides subcommand-per-node-type with semantic filters, extension-aware output, and version-agnostic behavior (OAS 2.0–3.2).

## Command Structure

```
oastools walk <subcommand> [flags] <spec>
```

Spec can be a file path, URL, or `-` for stdin.

### Subcommands

| Subcommand   | What it queries                              |
|--------------|----------------------------------------------|
| `operations` | Operations across all paths                  |
| `schemas`    | Schemas (components + inline)                |
| `parameters` | Parameters (path, query, header, cookie)     |
| `responses`  | Responses                                    |
| `security`   | Security schemes                             |
| `paths`      | Path items                                   |

### Common Flags

| Flag               | Description                                                     |
|--------------------|-----------------------------------------------------------------|
| `--format`         | Output format: `text` (default summary), `json`, `yaml`        |
| `-q, --quiet`      | Strip headers/decoration for piping                             |
| `--detail`         | Show full node instead of summary table                         |
| `--extension`      | Filter by extension (see Extension Filter Syntax)               |
| `--resolve-refs`   | Resolve `$ref` pointers in detail output                        |

### Subcommand-Specific Flags

**`operations`:**
- `--method <method>` — Filter by HTTP method
- `--path <pattern>` — Filter by path (supports glob)
- `--tag <tag>` — Filter by tag
- `--deprecated` — Only show deprecated operations
- `--operationId <id>` — Select by operationId

**`schemas`:**
- `--name <name>` — Select by schema name
- `--component` / `--inline` — Only component or inline schemas
- `--type <type>` — Filter by schema type (object, array, string, etc.)

**`parameters`:**
- `--in <location>` — Filter by location (path, query, header, cookie)
- `--name <name>` — Filter by parameter name
- `--path <pattern>` — Filter by owning path
- `--method <method>` — Filter by owning operation

**`responses`:**
- `--status <code>` — Filter by status code (200, 4xx, etc.)
- `--path <pattern>` — Filter by owning path
- `--method <method>` — Filter by owning operation

**`security`:** name filter

**`paths`:** path pattern filter

## Extension Filter Syntax

Extensions appear in summary table output by default and are filterable on all subcommands.

### Grammar

```
FILTER = EXPR ( ("," | "+") EXPR )*     # , = OR, + = AND
EXPR   = "!"? KEY (("=" | "!=") VALUE)?  # negation, existence, value match
KEY    = "x-" IDENTIFIER
VALUE  = string (no unescaped , or +)
```

### Modes

| Syntax                       | Meaning                           |
|------------------------------|-----------------------------------|
| `--extension x-foo`          | Node **has** the extension        |
| `--extension x-foo=bar`      | Extension **equals** `bar`        |
| `--extension '!x-foo'`       | Node does **not** have extension  |
| `--extension 'x-foo!=bar'`   | Extension does **not** equal      |

### Operators

- `+` = AND: `--extension x-audited-by+x-internal=true`
- `,` = OR: `--extension x-audited-by,x-internal=true`

### Shell Safety

- `,`, `+`, `=` are safe unquoted in bash
- `!` requires single quotes (bash history expansion)
- Common case (existence + value checks) works without quoting

## Output Modes

### Summary (default)

Table with columns relevant to each node type. Extensions column included by default:

```
METHOD  PATH        SUMMARY         TAGS    EXTENSIONS
GET     /pets       List all pets   pets    x-audited-by=u123, x-internal=true
POST    /pets       Create a pet    pets    x-internal=true
DELETE  /pets/{id}  Delete a pet    pets
```

### Detail (`--detail`)

Full YAML/JSON of matched node(s), re-marshaled from the parsed model.

### Quiet (`-q`)

Strips headers and decoration. Summary renders as clean delimited rows. Detail renders raw YAML/JSON.

## Architecture

### Package Layout

All code in `cmd/oastools/commands/`:

```
walk.go              # Top-level router, common flags, extension filter, renderers
walk_operations.go   # operations subcommand
walk_schemas.go      # schemas subcommand
walk_parameters.go   # parameters subcommand
walk_responses.go    # responses subcommand
walk_security.go     # security subcommand
walk_paths.go        # paths subcommand
```

No new packages. Walker package used as-is.

### Three-Layer Flow

Each subcommand follows:

```
1. Collect  →  Walker collectors gather all nodes of the type
2. Filter   →  Apply --extension, --method, --path, etc.
3. Render   →  Summary table or detail YAML/JSON
```

### Shared Infrastructure (`walk.go`)

- `WalkFlags` struct with common flags
- `ParseExtensionFilter(string) (ExtensionFilter, error)`
- `ExtensionFilter.Match(extensions map[string]any) bool`
- `RenderSummaryTable(w io.Writer, headers []string, rows [][]string, quiet bool)`
- `RenderDetail(w io.Writer, node any, format string, quiet bool)`

### Extension Filter Model

```go
type ExtensionFilter struct {
    Groups [][]ExtensionExpr  // outer = OR (,), inner = AND (+)
}

type ExtensionExpr struct {
    Key     string  // e.g., "x-audited-by"
    Value   *string // nil = existence check
    Negated bool    // ! prefix
}
```

### Walker Integration

- Existing `CollectOperations` and `CollectSchemas` used directly
- New collectors added to walker package for parameters, responses, security
- Walker handles OAS 2.0/3.x normalization — no version logic in walk command

### Detail Output

Re-marshal from parsed model (strategy 1). Source-map fragment extraction deferred as future enhancement.

## Error Handling

| Condition                | Behavior                                      | Exit |
|--------------------------|-----------------------------------------------|------|
| Parse failure            | Error message to stderr                       | 1    |
| Invalid flags/filter     | Error message, fail fast before parsing spec  | 1    |
| Zero results             | Informative message to stderr; silent in `-q` | 0    |
| Unsupported OAS version  | Parser error (existing behavior)              | 1    |

## Testing Strategy

### Unit Tests

1. **Extension filter parsing** — all syntax variants, edge cases, malformed input
2. **Extension filter matching** — against maps with various extension combinations
3. **Subcommand filtering** — flags correctly filter collected nodes, using inline OAS 2.0 and 3.x documents
4. **Rendering** — summary table format, detail YAML/JSON, `--quiet` behavior

### Integration Tests

- Golden file tests against petstore OAS 2.0 and 3.0 fixtures
- Pipe validity: `-q --format json` produces valid JSON
- Version-agnostic: same queries against equivalent 2.0/3.x specs produce same output shape

## Example Usage

```bash
# List all operations
oastools walk operations api.yaml

# Get details of a specific operation
oastools walk operations --method get --path /pets --detail api.yaml

# Find operations with an extension
oastools walk operations --extension x-audited-by api.yaml

# Find unaudited operations
oastools walk operations --extension '!x-audited-by' api.yaml

# AND/OR extension filters
oastools walk operations --extension x-audited-by+x-internal=true api.yaml
oastools walk operations --extension x-audited-by,x-internal=true api.yaml

# List component schemas
oastools walk schemas --component api.yaml

# Inspect a schema in detail
oastools walk schemas --name Pet --detail api.yaml

# Parameters of a specific operation
oastools walk parameters --path /pets --method get api.yaml

# 4xx responses, piped to jq
oastools walk responses --status '4xx' -q --format json api.yaml | jq '.[]'

# Works identically on OAS 2.0 and 3.x
oastools walk schemas --name Pet --detail swagger.yaml
oastools walk schemas --name Pet --detail openapi.yaml
```