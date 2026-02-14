# MCP Server Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add an `oastools mcp` subcommand that exposes all oastools capabilities as MCP tools over stdio, plus a Claude Code plugin for easy installation.

**Architecture:** Internal `internal/mcpserver/` package using the official Go MCP SDK (`github.com/modelcontextprotocol/go-sdk/mcp`). Each tool has its own file with a typed input struct (JSON schema auto-generated from struct tags) and a handler function. A shared `specInput` type handles file/URL/inline spec resolution. The server runs over stdio transport.

**Tech Stack:** Go 1.24+, `github.com/modelcontextprotocol/go-sdk` (v1.2.0+), existing oastools library packages.

**Design Doc:** `docs/plans/2026-02-13-mcp-server-design.md`

---

### Task 1: Foundation — SDK Dependency & Server Scaffold

**Files:**
- Modify: `go.mod` (add MCP SDK dependency)
- Create: `internal/mcpserver/server.go`
- Modify: `cmd/oastools/main.go` (add `mcp` case to switch)
- Modify: `cmd/oastools/commands/common.go` (if `validCommands` list exists, add "mcp")

**Step 1: Add the MCP SDK dependency**

Run: `go get github.com/modelcontextprotocol/go-sdk@latest`
Expected: `go.mod` and `go.sum` updated with new dependency

**Step 2: Create the server scaffold**

Create `internal/mcpserver/server.go`:

```go
package mcpserver

import (
	"context"
	"log"

	oastools "github.com/erraggy/oastools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Run starts the MCP server over stdio and blocks until the client disconnects.
func Run() {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "oastools", Version: oastools.Version()},
		nil,
	)
	registerAllTools(server)
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}

func registerAllTools(server *mcp.Server) {
	// Tools will be registered here as they are implemented.
}
```

**Step 3: Wire up the `mcp` subcommand in main.go**

Add to the switch statement in `cmd/oastools/main.go`, alongside existing cases:

```go
case "mcp":
	mcpserver.Run()
```

Add the import: `"github.com/erraggy/oastools/internal/mcpserver"`

Add `"mcp"` to the `validCommands` slice.

**Step 4: Verify it compiles**

Run: `go build ./cmd/oastools/`
Expected: Successful build, no errors

**Step 5: Run gopls diagnostics**

Run gopls `go_diagnostics` on modified files.
Expected: No errors

**Step 6: Commit**

```bash
git add go.mod go.sum internal/mcpserver/server.go cmd/oastools/main.go
git commit -m "feat(mcp): add MCP server scaffold with oastools mcp subcommand"
```

---

### Task 2: Shared Input Model

**Files:**
- Create: `internal/mcpserver/input.go`
- Create: `internal/mcpserver/input_test.go`

**Step 1: Write failing tests for specInput resolution**

Create `internal/mcpserver/input_test.go`:

```go
package mcpserver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpecInput_ResolveFile(t *testing.T) {
	// Use an existing testdata file from the repo
	input := specInput{File: "../../testdata/petstore-v3.yaml"}
	result, err := input.resolve()
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.Version)
}

func TestSpecInput_ResolveContent(t *testing.T) {
	content := `openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`
	input := specInput{Content: content}
	result, err := input.resolve()
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "3.0.0", result.Version)
}

func TestSpecInput_ResolveNoneProvided(t *testing.T) {
	input := specInput{}
	_, err := input.resolve()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one of file, url, or content must be provided")
}

func TestSpecInput_ResolveMultipleProvided(t *testing.T) {
	input := specInput{File: "foo.yaml", Content: "bar"}
	_, err := input.resolve()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exactly one of file, url, or content must be provided")
}

func TestSpecInput_ResolveFileNotFound(t *testing.T) {
	input := specInput{File: "/nonexistent/path.yaml"}
	_, err := input.resolve()
	assert.Error(t, err)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/mcpserver/ -v -run TestSpecInput`
Expected: FAIL — `specInput` type not defined

**Step 3: Implement specInput**

Create `internal/mcpserver/input.go`:

```go
package mcpserver

import (
	"fmt"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// specInput represents the three ways an OAS spec can be provided to a tool.
// Exactly one of File, URL, or Content must be set.
type specInput struct {
	File    string `json:"file,omitempty"    jsonschema:"description=Path to an OAS file on disk"`
	URL     string `json:"url,omitempty"     jsonschema:"description=URL to fetch an OAS document from"`
	Content string `json:"content,omitempty" jsonschema:"description=Inline OAS document content (JSON or YAML)"`
}

// resolve parses the spec from whichever input was provided.
func (s specInput) resolve(extraOpts ...parser.Option) (*parser.ParseResult, error) {
	count := 0
	if s.File != "" {
		count++
	}
	if s.URL != "" {
		count++
	}
	if s.Content != "" {
		count++
	}
	if count != 1 {
		return nil, fmt.Errorf("exactly one of file, url, or content must be provided (got %d)", count)
	}

	var opts []parser.Option
	switch {
	case s.File != "":
		opts = append(opts, parser.WithFilePath(s.File))
	case s.URL != "":
		opts = append(opts, parser.WithFilePath(s.URL))
	case s.Content != "":
		opts = append(opts, parser.WithReader(strings.NewReader(s.Content)))
	}
	opts = append(opts, extraOpts...)

	return parser.ParseWithOptions(opts...)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/mcpserver/ -v -run TestSpecInput`
Expected: All PASS

Note: Adjust the testdata file path in `TestSpecInput_ResolveFile` if needed — find an existing OAS 3.x YAML file in `testdata/`. Check `testdata/` for available files.

**Step 5: Run gopls diagnostics**

Run gopls `go_diagnostics` on `internal/mcpserver/input.go`.

**Step 6: Commit**

```bash
git add internal/mcpserver/input.go internal/mcpserver/input_test.go
git commit -m "feat(mcp): add shared specInput type for file/URL/inline resolution"
```

---

### Task 3: First Tool — validate

This task establishes the pattern that all subsequent tools follow.

**Files:**
- Create: `internal/mcpserver/tools_validate.go`
- Create: `internal/mcpserver/tools_validate_test.go`
- Modify: `internal/mcpserver/server.go` (register the tool)

**Step 1: Write failing test**

Create `internal/mcpserver/tools_validate_test.go`:

```go
package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTool_ValidSpec(t *testing.T) {
	content := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
`
	input := validateInput{
		Spec: specInput{Content: content},
	}
	_, output, err := handleValidate(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.True(t, output.Valid)
	assert.Empty(t, output.Errors)
}

func TestValidateTool_InvalidSpec(t *testing.T) {
	content := `openapi: "3.0.0"
info:
  title: Test API
paths: {}
`
	input := validateInput{
		Spec: specInput{Content: content},
	}
	_, output, err := handleValidate(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.False(t, output.Valid)
	assert.NotEmpty(t, output.Errors)
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/mcpserver/ -v -run TestValidateTool`
Expected: FAIL — `validateInput` not defined

**Step 3: Implement the validate tool**

Create `internal/mcpserver/tools_validate.go`:

```go
package mcpserver

import (
	"context"

	"github.com/erraggy/oastools/validator"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type validateInput struct {
	Spec       specInput `json:"spec"                    jsonschema:"description=The OAS document to validate"`
	Strict     bool      `json:"strict,omitempty"        jsonschema:"description=Enable strict validation mode"`
	NoWarnings bool      `json:"no_warnings,omitempty"   jsonschema:"description=Suppress warnings from output"`
}

type validateIssue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

type validateOutput struct {
	Valid        bool            `json:"valid"`
	Version      string          `json:"version"`
	ErrorCount   int             `json:"error_count"`
	WarningCount int             `json:"warning_count"`
	Errors       []validateIssue `json:"errors,omitempty"`
	Warnings     []validateIssue `json:"warnings,omitempty"`
}

func handleValidate(ctx context.Context, req *mcp.CallToolRequest, input validateInput) (*mcp.CallToolResult, validateOutput, error) {
	parseResult, err := input.Spec.resolve()
	if err != nil {
		return errResult(err), validateOutput{}, nil
	}

	var opts []validator.Option
	opts = append(opts, validator.WithParsed(*parseResult))
	if input.Strict {
		opts = append(opts, validator.WithStrictMode(true))
	}

	result, err := validator.ValidateWithOptions(opts...)
	if err != nil {
		return errResult(err), validateOutput{}, nil
	}

	output := validateOutput{
		Valid:        result.Valid,
		Version:      result.Version,
		ErrorCount:   result.ErrorCount,
		WarningCount: result.WarningCount,
	}

	for _, e := range result.Errors {
		output.Errors = append(output.Errors, validateIssue{
			Path:    e.Path,
			Message: e.Message,
			Field:   e.Field,
		})
	}
	if !input.NoWarnings {
		for _, w := range result.Warnings {
			output.Warnings = append(output.Warnings, validateIssue{
				Path:    w.Path,
				Message: w.Message,
				Field:   w.Field,
			})
		}
	}

	return nil, output, nil
}
```

**Step 4: Add the errResult helper to server.go**

Add to `internal/mcpserver/server.go`:

```go
// errResult creates an MCP error result from an error.
func errResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{mcp.NewTextContent(err.Error())},
	}
}
```

**Step 5: Register the validate tool**

Update `registerAllTools` in `server.go`:

```go
func registerAllTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "validate",
		Description: "Validate an OpenAPI Specification document against its version schema. Returns validation errors and warnings.",
	}, handleValidate)
}
```

**Step 6: Run tests**

Run: `go test ./internal/mcpserver/ -v -run TestValidateTool`
Expected: All PASS

Note: The exact validator option names may differ. Check `validator.WithParsed`, `validator.WithStrictMode` — if these don't exist, find the correct option names via gopls. The `ValidationError` fields (`Path`, `Message`, `Field`) may also have slightly different names — verify with `go_package_api`.

**Step 7: Run gopls diagnostics**

Run gopls `go_diagnostics` on all modified files.

**Step 8: Commit**

```bash
git add internal/mcpserver/tools_validate.go internal/mcpserver/tools_validate_test.go internal/mcpserver/server.go
git commit -m "feat(mcp): add validate tool with errResult helper"
```

---

### Task 4: parse Tool

**Files:**
- Create: `internal/mcpserver/tools_parse.go`
- Create: `internal/mcpserver/tools_parse_test.go`
- Modify: `internal/mcpserver/server.go` (register)

**Step 1: Write failing test**

Test that parse returns document info and stats for a valid spec. Test both summary mode (default) and full mode.

```go
func TestParseTool_Summary(t *testing.T) {
	input := parseInput{
		Spec: specInput{Content: minimalOAS3},
	}
	_, output, err := handleParse(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotEmpty(t, output.Version)
}
```

**Step 2: Run test to verify failure, then implement**

The parse tool calls `parser.ParseWithOptions()` and returns either a summary (version, paths count, schemas count, servers) or the full parsed document depending on the `Full` input flag.

```go
type parseInput struct {
	Spec        specInput `json:"spec"                     jsonschema:"description=The OAS document to parse"`
	ResolveRefs bool      `json:"resolve_refs,omitempty"   jsonschema:"description=Resolve $ref pointers before returning"`
	Full        bool      `json:"full,omitempty"           jsonschema:"description=Return full parsed document instead of summary"`
}
```

**Step 3: Run tests, run diagnostics, commit**

```bash
git commit -m "feat(mcp): add parse tool"
```

---

### Task 5: fix Tool

**Files:**
- Create: `internal/mcpserver/tools_fix.go`
- Create: `internal/mcpserver/tools_fix_test.go`
- Modify: `internal/mcpserver/server.go` (register)

**Step 1: Write failing test**

Test that fix returns a list of applied fixes. Test dry_run mode.

**Step 2: Implement**

Key: `fixer.Fix()` internally parses, so we need to handle input differently. Check if `FixWithOptions` accepts a file path directly or needs parsed input. The fix tool should return the list of fixes applied (not the full document by default).

```go
type fixInput struct {
	Spec                    specInput `json:"spec"                                  jsonschema:"description=The OAS document to fix"`
	FixSchemaNames          bool      `json:"fix_schema_names,omitempty"            jsonschema:"description=Rename generic schema names (Object1, Model2) to meaningful names"`
	FixDuplicateOperationIds bool     `json:"fix_duplicate_operationids,omitempty"  jsonschema:"description=Fix duplicate operationId values"`
	Prune                   bool      `json:"prune,omitempty"                       jsonschema:"description=Remove empty paths and unused schemas"`
	StubMissingRefs         bool      `json:"stub_missing_refs,omitempty"           jsonschema:"description=Create stub schemas for missing $ref targets"`
	DryRun                  bool      `json:"dry_run,omitempty"                     jsonschema:"description=Preview fixes without applying them"`
	IncludeDocument         bool      `json:"include_document,omitempty"            jsonschema:"description=Include the full corrected document in output"`
}
```

Important: Remember from MEMORY.md — `Fix()` owns its document and sets `MutableInput=true` internally.

**Step 3: Run tests, diagnostics, commit**

```bash
git commit -m "feat(mcp): add fix tool"
```

---

### Task 6: convert Tool

**Files:**
- Create: `internal/mcpserver/tools_convert.go`
- Create: `internal/mcpserver/tools_convert_test.go`
- Modify: `internal/mcpserver/server.go` (register)

**Step 1: Write test, then implement**

```go
type convertInput struct {
	Spec   specInput `json:"spec"               jsonschema:"description=The OAS document to convert"`
	Target string    `json:"target"             jsonschema:"description=Target OAS version: 2.0 or 3.0 or 3.1,enum=2.0|3.0|3.1"`
	Strict bool      `json:"strict,omitempty"   jsonschema:"description=Enable strict conversion mode"`
	Output string    `json:"output,omitempty"   jsonschema:"description=File path to write converted document. If omitted the converted document is returned inline."`
}
```

The convert tool calls `converter.ConvertWithOptions()`. If `Output` is set, write to file and return a summary. Otherwise return the converted document inline.

**Step 2: Run tests, diagnostics, commit**

```bash
git commit -m "feat(mcp): add convert tool"
```

---

### Task 7: diff Tool (Two Inputs)

**Files:**
- Create: `internal/mcpserver/tools_diff.go`
- Create: `internal/mcpserver/tools_diff_test.go`
- Modify: `internal/mcpserver/server.go` (register)

**Step 1: Write test**

This tool is special — it takes two specs (base and revision). Test with two inline specs that have a known difference.

**Step 2: Implement**

```go
type diffInput struct {
	Base         specInput `json:"base"                   jsonschema:"description=The base/original OAS document"`
	Revision     specInput `json:"revision"               jsonschema:"description=The revised OAS document to compare against the base"`
	BreakingOnly bool      `json:"breaking_only,omitempty" jsonschema:"description=Only show breaking changes"`
	NoInfo       bool      `json:"no_info,omitempty"       jsonschema:"description=Suppress informational changes"`
}
```

The handler resolves both specs, then calls `differ.DiffWithOptions()` with both parsed results. Returns the list of changes.

**Step 3: Run tests, diagnostics, commit**

```bash
git commit -m "feat(mcp): add diff tool with dual-spec input"
```

---

### Task 8: join Tool (Array Input)

**Files:**
- Create: `internal/mcpserver/tools_join.go`
- Create: `internal/mcpserver/tools_join_test.go`
- Modify: `internal/mcpserver/server.go` (register)

**Step 1: Write test**

Test with two minimal inline specs that should merge cleanly.

**Step 2: Implement**

```go
type joinInput struct {
	Specs             []specInput `json:"specs"                         jsonschema:"description=Array of OAS documents to join (minimum 2)"`
	PathStrategy      string      `json:"path_strategy,omitempty"       jsonschema:"description=Strategy for path collisions: accept_left or accept_right or fail"`
	SchemaStrategy    string      `json:"schema_strategy,omitempty"     jsonschema:"description=Strategy for schema collisions: accept_left or accept_right or fail or rename"`
	SemanticDedup     bool        `json:"semantic_dedup,omitempty"      jsonschema:"description=Enable semantic deduplication of equivalent schemas"`
	Output            string      `json:"output,omitempty"              jsonschema:"description=File path to write joined document. If omitted the result is returned inline."`
}
```

Resolve all specs in the array, pass parsed results to `joiner.JoinWithOptions()`.

**Step 3: Run tests, diagnostics, commit**

```bash
git commit -m "feat(mcp): add join tool with array spec input"
```

---

### Task 9: overlay Tools (apply + validate)

**Files:**
- Create: `internal/mcpserver/tools_overlay.go`
- Create: `internal/mcpserver/tools_overlay_test.go`
- Modify: `internal/mcpserver/server.go` (register both)

**Step 1: Write tests for both overlay_apply and overlay_validate**

**Step 2: Implement two tools in one file**

```go
type overlayApplyInput struct {
	Spec    specInput `json:"spec"               jsonschema:"description=The OAS document to apply the overlay to"`
	Overlay specInput `json:"overlay"            jsonschema:"description=The Overlay document to apply"`
	DryRun  bool      `json:"dry_run,omitempty"  jsonschema:"description=Preview changes without applying"`
	Output  string    `json:"output,omitempty"   jsonschema:"description=File path to write result. If omitted the result is returned inline."`
}

type overlayValidateInput struct {
	Overlay specInput `json:"overlay" jsonschema:"description=The Overlay document to validate"`
}
```

Register both as `overlay_apply` and `overlay_validate`.

**Step 3: Run tests, diagnostics, commit**

```bash
git commit -m "feat(mcp): add overlay_apply and overlay_validate tools"
```

---

### Task 10: generate Tool

**Files:**
- Create: `internal/mcpserver/tools_generate.go`
- Create: `internal/mcpserver/tools_generate_test.go`
- Modify: `internal/mcpserver/server.go` (register)

**Step 1: Write test**

Test that generate produces files and returns a manifest.

**Step 2: Implement**

```go
type generateInput struct {
	Spec        specInput `json:"spec"                    jsonschema:"description=The OAS document to generate code from"`
	Client      bool      `json:"client,omitempty"        jsonschema:"description=Generate client code"`
	Server      bool      `json:"server,omitempty"        jsonschema:"description=Generate server code"`
	Types       bool      `json:"types,omitempty"         jsonschema:"description=Generate type definitions only"`
	PackageName string    `json:"package_name,omitempty"  jsonschema:"description=Go package name for generated code (default: api)"`
	OutputDir   string    `json:"output_dir"              jsonschema:"description=Directory to write generated files to"`
}
```

Key: `OutputDir` is required. The tool writes files and returns a manifest listing generated files, types count, and operations count.

**Step 3: Run tests, diagnostics, commit**

```bash
git commit -m "feat(mcp): add generate tool"
```

---

### Task 11: walk_operations + walk_schemas Tools

These two tools establish the walk tool pattern.

**Files:**
- Create: `internal/mcpserver/tools_walk_operations.go`
- Create: `internal/mcpserver/tools_walk_operations_test.go`
- Create: `internal/mcpserver/tools_walk_schemas.go`
- Create: `internal/mcpserver/tools_walk_schemas_test.go`
- Modify: `internal/mcpserver/server.go` (register both)

**Step 1: Write failing test for walk_operations**

```go
func TestWalkOperationsTool(t *testing.T) {
	content := `openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /pets:
    get:
      summary: List pets
      operationId: listPets
      tags: [pets]
      responses:
        "200":
          description: OK
    post:
      summary: Create pet
      operationId: createPet
      tags: [pets]
      responses:
        "201":
          description: Created
`
	input := walkOperationsInput{
		Spec:   specInput{Content: content},
		Method: "get",
	}
	_, output, err := handleWalkOperations(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.Len(t, output.Operations, 1)
	assert.Equal(t, "get", output.Operations[0].Method)
}
```

**Step 2: Implement walk_operations**

```go
type walkOperationsInput struct {
	Spec        specInput `json:"spec"                       jsonschema:"description=The OAS document to walk"`
	Method      string    `json:"method,omitempty"           jsonschema:"description=Filter by HTTP method (get, post, put, delete, patch, etc.)"`
	Path        string    `json:"path,omitempty"             jsonschema:"description=Filter by path pattern (supports * glob, e.g. /pets/*)"`
	Tag         string    `json:"tag,omitempty"              jsonschema:"description=Filter by tag name"`
	Deprecated  bool      `json:"deprecated,omitempty"       jsonschema:"description=Only show deprecated operations"`
	OperationID string    `json:"operation_id,omitempty"     jsonschema:"description=Select by operationId"`
	Extension   string    `json:"extension,omitempty"        jsonschema:"description=Filter by extension (e.g. x-internal=true)"`
	ResolveRefs bool      `json:"resolve_refs,omitempty"     jsonschema:"description=Resolve $ref pointers before output"`
	Detail      bool      `json:"detail,omitempty"           jsonschema:"description=Return full operation objects instead of summaries"`
	Limit       int       `json:"limit,omitempty"            jsonschema:"description=Maximum number of results to return (default 100)"`
}
```

The handler calls `walker.CollectOperations()`, filters results by the provided params, and returns either summaries or full operation objects. Use the same filtering logic from `cmd/oastools/commands/walk_operations.go` — reuse or reimplement the path glob matching and extension filtering.

**Step 3: Run test, verify pass**

**Step 4: Write test + implement walk_schemas (same pattern)**

```go
type walkSchemasInput struct {
	Spec        specInput `json:"spec"                     jsonschema:"description=The OAS document to walk"`
	Name        string    `json:"name,omitempty"           jsonschema:"description=Filter by schema name"`
	Type        string    `json:"type,omitempty"           jsonschema:"description=Filter by schema type (object, array, string, integer, etc.)"`
	Component   bool      `json:"component,omitempty"      jsonschema:"description=Only show component schemas"`
	Inline      bool      `json:"inline,omitempty"         jsonschema:"description=Only show inline schemas"`
	Extension   string    `json:"extension,omitempty"      jsonschema:"description=Filter by extension"`
	ResolveRefs bool      `json:"resolve_refs,omitempty"   jsonschema:"description=Resolve $ref pointers"`
	Detail      bool      `json:"detail,omitempty"         jsonschema:"description=Return full schema objects"`
	Limit       int       `json:"limit,omitempty"          jsonschema:"description=Maximum results (default 100)"`
}
```

**Step 5: Run all tests, diagnostics, commit**

```bash
git commit -m "feat(mcp): add walk_operations and walk_schemas tools"
```

---

### Task 12: Remaining Walk Tools

**Files:**
- Create: `internal/mcpserver/tools_walk_parameters.go`
- Create: `internal/mcpserver/tools_walk_parameters_test.go`
- Create: `internal/mcpserver/tools_walk_responses.go`
- Create: `internal/mcpserver/tools_walk_responses_test.go`
- Create: `internal/mcpserver/tools_walk_security.go`
- Create: `internal/mcpserver/tools_walk_security_test.go`
- Create: `internal/mcpserver/tools_walk_paths.go`
- Create: `internal/mcpserver/tools_walk_paths_test.go`
- Modify: `internal/mcpserver/server.go` (register all four)

All four follow the exact same pattern as walk_operations/walk_schemas. Each:
1. Defines an input struct with specInput + filter fields + `resolve_refs`, `detail`, `limit`
2. Calls the corresponding `walker.CollectXxx()` function
3. Filters results by the provided params
4. Returns summaries or detail objects

**walk_parameters filters:** `in` (query/header/path/cookie), `name`, `path`, `method`, `extension`

**walk_responses filters:** `status` (200, 4xx, default), `path`, `method`, `extension`

**walk_security filters:** `name`, `type` (apiKey/http/oauth2/openIdConnect), `extension`

**walk_paths filters:** `path` (glob pattern), `extension`

**Step 1: Implement and test walk_parameters**
**Step 2: Implement and test walk_responses**
**Step 3: Implement and test walk_security**
**Step 4: Implement and test walk_paths**

Run tests after each. Once all pass:

**Step 5: Run full test suite and diagnostics**

Run: `go test ./internal/mcpserver/ -v`
Expected: All tests pass

**Step 6: Commit**

```bash
git commit -m "feat(mcp): add walk_parameters, walk_responses, walk_security, walk_paths tools"
```

---

### Task 13: Integration Test

**Files:**
- Create: `internal/mcpserver/integration_test.go`

**Step 1: Write integration test using MCP client SDK**

This test starts the full MCP server, connects a client, and invokes tools end-to-end.

```go
package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_ValidateTool(t *testing.T) {
	server := mcp.NewServer(
		&mcp.Implementation{Name: "oastools-test", Version: "test"},
		nil,
	)
	registerAllTools(server)

	// Create an in-process client-server pair
	clientTransport, serverTransport := mcp.NewInMemoryTransport()

	go func() {
		_ = server.Run(context.Background(), serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "test"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	require.NoError(t, err)
	defer session.Close()

	// List tools
	tools, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(tools.Tools), 15) // 9 core + 6 walk

	// Call validate
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "validate",
		Arguments: map[string]any{
			"spec": map[string]any{
				"content": `openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}`,
			},
		},
	})
	require.NoError(t, err)
	assert.False(t, result.IsError)
}
```

Note: Check the MCP SDK for in-memory transport. If `mcp.NewInMemoryTransport()` doesn't exist, check for `mcp.NewInProcessTransport()` or similar. Alternatively, use `io.Pipe()` to create a stdio pair. Adjust based on what the SDK provides.

**Step 2: Run integration test**

Run: `go test ./internal/mcpserver/ -v -run TestIntegration`
Expected: PASS

**Step 3: Commit**

```bash
git commit -m "test(mcp): add integration test with in-process MCP client"
```

---

### Task 14: Claude Code Plugin

**Files:**
- Create: `plugin/.claude-plugin/plugin.json`
- Create: `plugin/.mcp.json`
- Create: `plugin/CLAUDE.md`
- Create: `plugin/skills/validate-spec.md`
- Create: `plugin/skills/fix-spec.md`
- Create: `plugin/skills/explore-api.md`
- Create: `plugin/skills/diff-specs.md`
- Create: `plugin/skills/generate-code.md`

**Step 1: Create plugin.json**

```json
{
  "name": "oastools",
  "description": "OpenAPI Specification tools — validate, fix, convert, diff, walk, and generate from OAS 2.0-3.2 documents",
  "version": "1.0.0",
  "author": "erraggy"
}
```

**Step 2: Create .mcp.json**

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

**Step 3: Create CLAUDE.md with tool usage guidance**

Document how agents should use the tools, best practices (prefer `file` over `content` for large specs, use walk tools for exploration before making changes, validate after fixing/converting).

**Step 4: Create skills**

Each skill is a markdown file that guides agents through a workflow:

- `validate-spec.md`: Validate, explain each error in context, suggest fixes
- `fix-spec.md`: Dry-run first → review changes → apply
- `explore-api.md`: Use walk_operations/schemas/paths to build API understanding
- `diff-specs.md`: Compare versions, highlight breaking changes with impact analysis
- `generate-code.md`: Guided code generation with option selection

Skills should reference the MCP tool names and guide agents through multi-step workflows.

**Step 5: Commit**

```bash
git add plugin/
git commit -m "feat(plugin): add Claude Code plugin with MCP config and skills"
```

---

### Task 15: Final Verification

**Step 1: Run make check**

Run: `make check`
Expected: All checks pass (lint, format, tests, trailing whitespace)

**Step 2: Fix any issues found by make check**

**Step 3: Run the full test suite**

Run: `go test ./... -count=1`
Expected: All tests pass

**Step 4: Manual smoke test**

Run: `echo '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}},"id":1}' | go run ./cmd/oastools/ mcp`
Expected: JSON-RPC response with server capabilities and tool list

**Step 5: Final commit if any fixes were needed**

```bash
git commit -m "chore(mcp): fix issues from make check"
```
