# MCP Server & Claude Code Plugin Design

**Date**: 2026-02-13
**Status**: Approved

## Overview

Add an MCP (Model Context Protocol) server to oastools, exposing all CLI capabilities as LLM agent tools. Ship a Claude Code plugin for easy installation.

## Goals

1. Expose the full oastools toolkit (validate, fix, convert, diff, join, parse, overlay, generate, walk) as MCP tools
2. Run as an `oastools mcp` subcommand over stdio transport
3. Provide a Claude Code plugin with guided workflow skills
4. Keep the MCP server internal — no exported Go API

## Architecture

### Server Location

New subcommand: `oastools mcp`

- Uses library internals directly (parser, validator, walker, etc.) — not CLI subprocess calls
- Runs over stdio transport, the standard for local MCP servers
- Zero extra install — users who have oastools already have the MCP server

### Package Structure

```
cmd/oastools/commands/mcp.go                    # HandleMCP() entry point
internal/mcpserver/                              # unexported — internal only
├── server.go                                    # newServer(), tool registration
├── input.go                                     # specInput type, resolution
├── tools_validate.go                            # validate tool
├── tools_parse.go                               # parse tool
├── tools_fix.go                                 # fix tool
├── tools_convert.go                             # convert tool
├── tools_diff.go                                # diff tool
├── tools_join.go                                # join tool
├── tools_overlay.go                             # overlay_apply, overlay_validate
├── tools_generate.go                            # generate tool
├── tools_walk_operations.go                     # walk_operations tool
├── tools_walk_schemas.go                        # walk_schemas tool
├── tools_walk_parameters.go                     # walk_parameters tool
├── tools_walk_responses.go                      # walk_responses tool
├── tools_walk_security.go                       # walk_security tool
└── tools_walk_paths.go                          # walk_paths tool
```

### SDK

Official Go MCP SDK: `github.com/modelcontextprotocol/go-sdk/mcp`

Server setup:

```go
func newServer() *mcp.Server {
    server := mcp.NewServer(
        &mcp.Implementation{Name: "oastools", Version: version.Version},
        nil,
    )
    registerAllTools(server)
    return server
}
```

## Input Model

### Shared specInput

Every tool that operates on a spec embeds `specInput`:

```go
type specInput struct {
    File    string `json:"file,omitempty"    jsonschema:"description=Path to an OAS file on disk"`
    URL     string `json:"url,omitempty"     jsonschema:"description=URL to fetch an OAS document from"`
    Content string `json:"content,omitempty" jsonschema:"description=Inline OAS document content (JSON or YAML)"`
}
```

Exactly one of `file`, `url`, or `content` must be provided. The `resolve()` method parses the spec from whichever input was given.

### Special Cases

- **diff**: Two specInput fields (`spec_base`, `spec_revision`)
- **join**: Array of specInputs (`specs`)
- **overlay_apply**: Two specInputs (`spec`, `overlay`)

## Tool Catalog

### Core Tools (9)

| Tool | Description | Key Params | Library API |
|------|-------------|------------|-------------|
| `validate` | Validate an OAS document | `strict`, `no_warnings` | `validator.Validate()` |
| `parse` | Parse and display OAS structure | `resolve_refs` | `parser.Parse()` |
| `fix` | Auto-fix common OAS issues | `fix_schema_names`, `fix_duplicate_operationids`, `prune`, `stub_missing_refs`, `dry_run` | `fixer.Fix()` |
| `convert` | Convert between OAS versions | `target` (required) | `converter.Convert()` |
| `diff` | Compare two specs, detect breaking changes | `breaking_only` | `differ.Diff()` |
| `join` | Merge multiple OAS documents | `path_strategy`, `schema_strategy`, `semantic_dedup` | `joiner.Join()` |
| `overlay_apply` | Apply Overlay to a spec | `dry_run` | `overlay.Apply()` |
| `overlay_validate` | Validate an Overlay document | — | `overlay.Validate()` |
| `generate` | Generate Go client/server code | `client`, `server`, `types`, `package_name`, `output_dir` | `generator.Generate()` |

### Walk Tools (6)

| Tool | Description | Key Params | Library API |
|------|-------------|------------|-------------|
| `walk_operations` | Query operations | `method`, `path`, `tag`, `deprecated`, `operation_id`, `extension` | `walker.CollectOperations()` |
| `walk_schemas` | Query schemas | `name`, `type`, `component`, `inline`, `extension` | `walker.CollectSchemas()` |
| `walk_parameters` | Query parameters | `in`, `name`, `path`, `method`, `extension` | `walker.CollectParameters()` |
| `walk_responses` | Query responses | `status`, `path`, `method`, `extension` | `walker.CollectResponses()` |
| `walk_security` | Query security schemes | `name`, `type`, `extension` | `walker.CollectSecuritySchemes()` |
| `walk_paths` | Query path items | `path`, `extension` | walker path collection |

All walk tools also accept: `resolve_refs` (bool), `detail` (bool — full node vs summary), `limit` (int — default 100).

Tool names are bare (e.g., `validate`, `walk_operations`) — MCP server scoping prevents collisions.

## Output Limits

Large output tools write to files and return summaries:

| Tool | Strategy |
|------|----------|
| `validate` | Return errors/warnings directly (small output) |
| `parse` | Summary by default; `full` param for complete output |
| `fix` | Return changes only; `include_document` for full output |
| `convert` | Write to file (`output` param), return summary |
| `diff` | Return changes directly |
| `join` | Write to file, return summary |
| `generate` | Write to `output_dir` (required), return file manifest |
| `walk_*` | Filtered results with `limit` param (default 100) |
| `overlay_apply` | Write to file or return changes only |

## Error Handling

Two categories:

1. **Tool-level errors** (bad input, file not found): Return `IsError: true` with descriptive message. Agent can retry.
2. **Analysis results** (spec has validation errors): Successful tool call with findings in the result. Not an MCP error.

## Claude Code Plugin

Located at `plugin/` in the oastools repo:

```
plugin/
├── .claude-plugin/
│   └── plugin.json
├── .mcp.json
├── CLAUDE.md
└── skills/
    ├── validate-spec.md
    ├── fix-spec.md
    ├── explore-api.md
    ├── diff-specs.md
    └── generate-code.md
```

### .mcp.json

```json
{
  "mcpServers": {
    "oastools": {
      "type": "stdio",
      "command": "oastools",
      "args": ["mcp"]
    }
  }
}
```

### Skills

| Skill | Purpose |
|-------|---------|
| `/validate-spec` | Validate a spec, explain errors, suggest fixes |
| `/fix-spec` | Dry-run first, review changes, then apply |
| `/explore-api` | Walk-based API structure exploration |
| `/diff-specs` | Compare versions, highlight breaking changes |
| `/generate-code` | Guided code generation with option selection |

## Testing

- **Unit tests**: Each `tools_*.go` gets `tools_*_test.go` testing handler functions directly
- **Integration tests**: Full MCP server with client SDK, send tool calls, verify responses
- **Existing testdata**: Leverage specs from `testdata/` already used by CLI tests

## Package Configuration Exposure

Each library package's configurable behaviors are exposed as tool params:

| Package | Exposed Params |
|---------|---------------|
| `parser` | `resolve_refs`, `resolve_http_refs`, `insecure` |
| `validator` | `strict`, `validate_structure`, `no_warnings` |
| `fixer` | Individual fix type booleans |
| `converter` | `target` |
| `differ` | `breaking_only`, `no_info` |
| `joiner` | `path_strategy`, `schema_strategy`, `semantic_dedup` |
| `generator` | `client`, `server`, `types`, `package_name`, `output_dir` |
| `walker` | `resolve_refs` |

CLI-only flags (`--quiet`, `--format text`, `--source-map`) are not exposed — agents always get JSON.

## Dependencies

New dependency: `github.com/modelcontextprotocol/go-sdk`