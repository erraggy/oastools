# MCP Walk Tool Usability Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix path glob matching, add schema name glob, add walk_refs tool, and audit all tool descriptions for AI agent usability.

**Architecture:** All changes are in `internal/mcpserver/`. The glob upgrade replaces `matchWalkPath()` with `**`-aware matching. Schema name glob adds a `matchGlobName()` helper. `walk_refs` is a new tool file following the existing walk tool pattern (input struct → handler → collector → filter → paginate → output). Descriptions are string-only changes in `server.go` and jsonschema tags.

**Tech Stack:** Go stdlib (`strings`, `path/filepath`), `walker.WithRefHandler` + `walker.WithMapRefTracking`, `stretchr/testify`

**Design doc:** `docs/plans/2026-02-14-mcp-walk-usability-design.md`

---

### Task 1: Upgrade `matchWalkPath` — Write Failing Tests

**Files:**
- Modify: `internal/mcpserver/tools_walk_operations_test.go`

**Step 1: Write failing tests for `**` glob support**

Add these tests after the existing `TestWalkOperations_FilterByPath` test (line ~139):

```go
func TestMatchWalkPath_DoubleStarMiddle(t *testing.T) {
	assert.True(t, matchWalkPath("/drives/{drive-id}/items/{driveItem-id}/workbook/functions/abs", "/drives/**/workbook/**"))
	assert.True(t, matchWalkPath("/drives/{drive-id}/items/{driveItem-id}/workbook/names/{id}/range()", "/drives/**/workbook/**"))
}

func TestMatchWalkPath_DoubleStarLeading(t *testing.T) {
	assert.True(t, matchWalkPath("/users/{id}/posts/{postId}", "/**/posts/*"))
	assert.True(t, matchWalkPath("/posts/{postId}", "/**/posts/*"))
}

func TestMatchWalkPath_DoubleStarTrailing(t *testing.T) {
	assert.True(t, matchWalkPath("/users/{id}", "/users/**"))
	assert.True(t, matchWalkPath("/users/{id}/posts", "/users/**"))
	assert.True(t, matchWalkPath("/users", "/users/**"))
}

func TestMatchWalkPath_DoubleStarMatchesZeroSegments(t *testing.T) {
	// ** can match zero segments.
	assert.True(t, matchWalkPath("/users/{id}", "/**/users/*"))
}

func TestMatchWalkPath_DoubleStarNoMatch(t *testing.T) {
	assert.False(t, matchWalkPath("/pets/{petId}", "/users/**"))
	assert.False(t, matchWalkPath("/stores", "/users/**/stores"))
}

func TestMatchWalkPath_SingleStarUnchanged(t *testing.T) {
	// Existing behavior: * matches exactly one segment.
	assert.True(t, matchWalkPath("/pets/{petId}", "/pets/*"))
	assert.False(t, matchWalkPath("/pets/{petId}/toys", "/pets/*"))
}

func TestMatchWalkPath_ExactMatchUnchanged(t *testing.T) {
	assert.True(t, matchWalkPath("/pets", "/pets"))
	assert.False(t, matchWalkPath("/pets", "/stores"))
}

func TestMatchWalkPath_EmptyPattern(t *testing.T) {
	assert.True(t, matchWalkPath("/anything", ""))
}

func TestMatchWalkPath_MixedStars(t *testing.T) {
	// Mix of * and ** in one pattern.
	assert.True(t, matchWalkPath("/sites/{id}/termStore/groups/{gid}/sets/{sid}/children/{tid}", "/sites/*/termStore/**"))
	assert.False(t, matchWalkPath("/sites/{id}/other/groups", "/sites/*/termStore/**"))
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/mcpserver/ -run "TestMatchWalkPath_DoubleStar|TestMatchWalkPath_Mixed" -v`
Expected: FAIL — `matchWalkPath` doesn't handle `**`

**Step 3: Commit failing tests**

```bash
git add internal/mcpserver/tools_walk_operations_test.go
git commit -m "test(mcp): add failing tests for ** glob path matching"
```

---

### Task 2: Implement `**` Glob Matching in `matchWalkPath`

**Files:**
- Modify: `internal/mcpserver/tools_walk_operations.go:150-173`

**Step 1: Replace `matchWalkPath` implementation**

Replace the function at lines 150-173 with:

```go
// matchWalkPath checks if a path template matches a pattern.
// Supports glob matching: * matches one path segment, ** matches zero or more segments.
func matchWalkPath(pathTemplate, pattern string) bool {
	if pattern == "" {
		return true
	}
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(pathTemplate, "/")
	return matchPathParts(pathParts, patternParts)
}

// matchPathParts recursively matches path segments against pattern segments.
// * matches exactly one segment, ** matches zero or more segments.
func matchPathParts(path, pattern []string) bool {
	for len(pattern) > 0 {
		seg := pattern[0]
		pattern = pattern[1:]

		if seg == "**" {
			// If ** is the last pattern segment, it matches everything remaining.
			if len(pattern) == 0 {
				return true
			}
			// Try matching the rest of the pattern at every possible position.
			for i := 0; i <= len(path); i++ {
				if matchPathParts(path[i:], pattern) {
					return true
				}
			}
			return false
		}

		if len(path) == 0 {
			return false
		}

		if seg != "*" && seg != path[0] {
			return false
		}

		path = path[1:]
	}
	return len(path) == 0
}
```

**Step 2: Run all matchWalkPath tests**

Run: `go test ./internal/mcpserver/ -run "TestMatchWalkPath|TestWalkOperations_FilterByPath" -v`
Expected: ALL PASS

**Step 3: Run full mcpserver tests to check for regressions**

Run: `go test ./internal/mcpserver/ -v -count=1`
Expected: ALL PASS

**Step 4: Run gopls diagnostics**

Run: `go_diagnostics` on `internal/mcpserver/tools_walk_operations.go`

**Step 5: Commit**

```bash
git add internal/mcpserver/tools_walk_operations.go
git commit -m "feat(mcp): add ** glob support for path matching in walk tools

Replaces single-segment-only * matching with recursive ** support.
** matches zero or more path segments, enabling patterns like
/drives/**/workbook/** that work across path depths.

Affects: walk_operations, walk_paths, walk_parameters, walk_responses."
```

---

### Task 3: Add Integration Test for `**` via `walk_operations`

**Files:**
- Modify: `internal/mcpserver/tools_walk_operations_test.go`

**Step 1: Write integration test using the walk tool directly**

Add a test spec with nested paths and test `**` filtering through the handler:

```go
const walkOperationsDeepPathSpec = `openapi: "3.0.0"
info:
  title: Deep Path Test
  version: "1.0.0"
paths:
  /drives/{driveId}/items/{itemId}/workbook/functions/abs:
    post:
      operationId: workbook.functions.abs
      summary: Invoke abs
      responses:
        "200":
          description: OK
  /drives/{driveId}/items/{itemId}/workbook/worksheets/{wsId}/range:
    get:
      operationId: workbook.worksheets.range
      summary: Get range
      responses:
        "200":
          description: OK
  /users/{userId}/posts:
    get:
      operationId: users.listPosts
      summary: List posts
      responses:
        "200":
          description: OK
`

func TestWalkOperations_FilterByPathDoubleStar(t *testing.T) {
	input := walkOperationsInput{
		Spec: specInput{Content: walkOperationsDeepPathSpec},
		Path: "/drives/**/workbook/**",
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 3, output.Total)
	assert.Equal(t, 2, output.Matched)
	ids := make([]string, 0, len(output.Summaries))
	for _, s := range output.Summaries {
		ids = append(ids, s.OperationID)
	}
	assert.Contains(t, ids, "workbook.functions.abs")
	assert.Contains(t, ids, "workbook.worksheets.range")
}

func TestWalkOperations_FilterByPathDoubleStarTrailing(t *testing.T) {
	input := walkOperationsInput{
		Spec: specInput{Content: walkOperationsDeepPathSpec},
		Path: "/users/**",
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 1, output.Matched)
	assert.Equal(t, "users.listPosts", output.Summaries[0].OperationID)
}
```

**Step 2: Run the tests**

Run: `go test ./internal/mcpserver/ -run "TestWalkOperations_FilterByPathDoubleStar" -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/mcpserver/tools_walk_operations_test.go
git commit -m "test(mcp): add integration tests for ** path glob in walk_operations"
```

---

### Task 4: Add Schema Name Glob — Write Failing Test

**Files:**
- Modify: `internal/mcpserver/tools_walk_schemas_test.go`

**Step 1: Write failing tests**

Add after the existing `TestWalkSchemas_FilterByName` test:

```go
func TestWalkSchemas_FilterByNameGlob(t *testing.T) {
	input := walkSchemasInput{
		Spec: specInput{Content: walkSchemasTestSpec},
		Name: "*et*",
	}
	_, output := callWalkSchemas(t, input)

	// Should match "Pet" (case-insensitive glob).
	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "Pet", output.Summaries[0].Name)
}

func TestWalkSchemas_FilterByNameGlobStar(t *testing.T) {
	input := walkSchemasInput{
		Spec:      specInput{Content: walkSchemasTestSpec},
		Name:      "*",
		Component: true,
	}
	_, output := callWalkSchemas(t, input)

	// Glob * matches all names — should return all component schemas.
	assert.Equal(t, 8, output.Matched)
}

func TestWalkSchemas_FilterByNameGlobPrefix(t *testing.T) {
	input := walkSchemasInput{
		Spec:      specInput{Content: walkSchemasTestSpec},
		Name:      "P*",
		Component: true,
	}
	_, output := callWalkSchemas(t, input)

	// Should match "Pet" only (not "Error" or "Tag").
	assert.GreaterOrEqual(t, output.Matched, 1)
	for _, s := range output.Summaries {
		assert.True(t, strings.HasPrefix(strings.ToLower(s.Name), "p"),
			"expected name starting with P, got %q", s.Name)
	}
}

func TestWalkSchemas_FilterByNameExactUnchanged(t *testing.T) {
	// No glob chars → exact match (case-insensitive), unchanged behavior.
	input := walkSchemasInput{
		Spec: specInput{Content: walkSchemasTestSpec},
		Name: "pet",
	}
	_, output := callWalkSchemas(t, input)

	assert.Equal(t, 1, output.Matched)
	assert.Equal(t, "Pet", output.Summaries[0].Name)
}
```

NOTE: You'll need `"strings"` in the import block.

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/mcpserver/ -run "TestWalkSchemas_FilterByNameGlob" -v`
Expected: FAIL — exact match doesn't support `*et*`

**Step 3: Commit**

```bash
git add internal/mcpserver/tools_walk_schemas_test.go
git commit -m "test(mcp): add failing tests for schema name glob matching"
```

---

### Task 5: Implement Schema Name Glob Matching

**Files:**
- Modify: `internal/mcpserver/tools_walk_schemas.go:1-10` (imports), `tools_walk_schemas.go:118-146` (filterWalkSchemas)

**Step 1: Add `matchGlobName` helper and update filter**

Add the helper function after `schemaTypeString` (at the end of the file):

```go
// matchGlobName matches a name against a pattern. If the pattern contains
// glob characters (* or ?), it uses case-insensitive filepath.Match.
// Otherwise, it falls back to case-insensitive exact match.
func matchGlobName(name, pattern string) bool {
	if strings.ContainsAny(pattern, "*?") {
		matched, err := filepath.Match(strings.ToLower(pattern), strings.ToLower(name))
		return err == nil && matched
	}
	return strings.EqualFold(name, pattern)
}
```

Add `"path/filepath"` to the imports.

Then update line 134 in `filterWalkSchemas`:

Change:
```go
if input.Name != "" && !strings.EqualFold(info.Name, input.Name) {
```

To:
```go
if input.Name != "" && !matchGlobName(info.Name, input.Name) {
```

**Step 2: Run tests**

Run: `go test ./internal/mcpserver/ -run "TestWalkSchemas_FilterByName" -v`
Expected: ALL PASS (both old exact and new glob tests)

**Step 3: Run gopls diagnostics**

Run: `go_diagnostics` on `internal/mcpserver/tools_walk_schemas.go`

**Step 4: Commit**

```bash
git add internal/mcpserver/tools_walk_schemas.go
git commit -m "feat(mcp): add glob pattern matching for schema name filter

Schema name filter now supports * and ? glob patterns via filepath.Match
(case-insensitive). Non-glob names use exact match (backwards-compatible)."
```

---

### Task 6: Add `walk_refs` Tool — Write Failing Test

**Files:**
- Create: `internal/mcpserver/tools_walk_refs_test.go`

**Step 1: Write tests for both summary and detail modes**

```go
package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const walkRefsTestSpec = `openapi: "3.0.0"
info:
  title: Walk Refs Test
  version: "1.0.0"
paths:
  /pets:
    get:
      summary: List pets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/Pet"
    post:
      summary: Create a pet
      requestBody:
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Pet"
      responses:
        "201":
          description: Created
        "400":
          $ref: "#/components/responses/BadRequest"
  /pets/{petId}:
    get:
      summary: Get a pet
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Pet"
        "404":
          $ref: "#/components/responses/NotFound"
  /errors:
    get:
      summary: List errors
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Error"
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
    Error:
      type: object
      properties:
        code:
          type: integer
        message:
          type: string
  responses:
    BadRequest:
      description: Bad request
    NotFound:
      description: Not found
`

func callWalkRefs(t *testing.T, input walkRefsInput) (*mcp.CallToolResult, walkRefsOutput) {
	t.Helper()
	result, out, err := handleWalkRefs(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	if out == nil {
		return result, walkRefsOutput{}
	}
	wo, ok := out.(walkRefsOutput)
	require.True(t, ok, "expected walkRefsOutput, got %T", out)
	return result, wo
}

func TestWalkRefs_SummaryMode(t *testing.T) {
	input := walkRefsInput{
		Spec: specInput{Content: walkRefsTestSpec},
	}
	_, output := callWalkRefs(t, input)

	assert.Greater(t, output.Total, 0)
	assert.Greater(t, output.Matched, 0)
	require.NotEmpty(t, output.Summaries)

	// Pet should be the most-referenced schema (3 refs).
	assert.Equal(t, "#/components/schemas/Pet", output.Summaries[0].Ref)
	assert.Equal(t, 3, output.Summaries[0].Count)

	// Verify sorted descending by count.
	for i := 1; i < len(output.Summaries); i++ {
		assert.GreaterOrEqual(t, output.Summaries[i-1].Count, output.Summaries[i].Count)
	}
}

func TestWalkRefs_FilterByTarget(t *testing.T) {
	input := walkRefsInput{
		Spec:   specInput{Content: walkRefsTestSpec},
		Target: "*schemas/Pet",
	}
	_, output := callWalkRefs(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "#/components/schemas/Pet", output.Summaries[0].Ref)
	assert.Equal(t, 3, output.Summaries[0].Count)
}

func TestWalkRefs_FilterByNodeType(t *testing.T) {
	input := walkRefsInput{
		Spec:     specInput{Content: walkRefsTestSpec},
		NodeType: "response",
	}
	_, output := callWalkRefs(t, input)

	// Only response refs: BadRequest and NotFound.
	assert.Equal(t, 2, output.Matched)
	for _, s := range output.Summaries {
		assert.Contains(t, s.Ref, "#/components/responses/")
	}
}

func TestWalkRefs_DetailMode(t *testing.T) {
	input := walkRefsInput{
		Spec:   specInput{Content: walkRefsTestSpec},
		Target: "*schemas/Pet",
		Detail: true,
	}
	_, output := callWalkRefs(t, input)

	assert.Nil(t, output.Summaries)
	require.Len(t, output.Details, 3)
	for _, d := range output.Details {
		assert.Equal(t, "#/components/schemas/Pet", d.Ref)
		assert.NotEmpty(t, d.SourcePath)
		assert.Equal(t, "schema", d.NodeType)
	}
}

func TestWalkRefs_Pagination(t *testing.T) {
	input := walkRefsInput{
		Spec:  specInput{Content: walkRefsTestSpec},
		Limit: 2,
	}
	_, output := callWalkRefs(t, input)

	assert.Equal(t, 2, output.Returned)
	require.Len(t, output.Summaries, 2)
}

func TestWalkRefs_TargetGlob(t *testing.T) {
	input := walkRefsInput{
		Spec:   specInput{Content: walkRefsTestSpec},
		Target: "*schemas/*",
	}
	_, output := callWalkRefs(t, input)

	// Should match Pet and Error schemas.
	assert.Equal(t, 2, output.Matched)
}
```

**Step 2: Run tests — expect compile errors**

Run: `go test ./internal/mcpserver/ -run "TestWalkRefs" -v`
Expected: FAIL — `handleWalkRefs`, `walkRefsInput`, `walkRefsOutput` undefined

**Step 3: Commit**

```bash
git add internal/mcpserver/tools_walk_refs_test.go
git commit -m "test(mcp): add tests for new walk_refs tool"
```

---

### Task 7: Implement `walk_refs` Tool

**Files:**
- Create: `internal/mcpserver/tools_walk_refs.go`
- Modify: `internal/mcpserver/server.go:23-98` (register the tool)

**Step 1: Create `tools_walk_refs.go`**

```go
package mcpserver

import (
	"context"
	"sort"
	"strings"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type walkRefsInput struct {
	Spec     specInput `json:"spec"                   jsonschema:"The OAS document to walk"`
	Target   string    `json:"target,omitempty"        jsonschema:"Filter by ref target (supports * and ? glob, e.g. *schemas/Pet or *responses/*)"`
	NodeType string    `json:"node_type,omitempty"     jsonschema:"Filter by ref node type: schema, parameter, response, requestBody, header, pathItem"`
	Detail   bool      `json:"detail,omitempty"        jsonschema:"Return individual source locations instead of aggregated counts"`
	Limit    int       `json:"limit,omitempty"         jsonschema:"Maximum number of results to return (default 100)"`
	Offset   int       `json:"offset,omitempty"        jsonschema:"Skip the first N results (for pagination)"`
}

type refSummary struct {
	Ref   string `json:"ref"`
	Count int    `json:"count"`
}

type refDetail struct {
	Ref        string `json:"ref"`
	SourcePath string `json:"source_path"`
	NodeType   string `json:"node_type"`
}

type walkRefsOutput struct {
	Total     int          `json:"total"`
	Matched   int          `json:"matched"`
	Returned  int          `json:"returned"`
	Summaries []refSummary `json:"refs,omitempty"`
	Details   []refDetail  `json:"details,omitempty"`
}

func handleWalkRefs(_ context.Context, _ *mcp.CallToolRequest, input walkRefsInput) (*mcp.CallToolResult, any, error) {
	result, err := input.Spec.resolve()
	if err != nil {
		return errResult(err), nil, nil
	}

	// Collect all refs via the walker.
	var allRefs []*walker.RefInfo
	err = walker.Walk(result,
		walker.WithMapRefTracking(),
		walker.WithRefHandler(func(wc *walker.WalkContext, ref *walker.RefInfo) walker.Action {
			allRefs = append(allRefs, ref)
			return walker.Continue
		}),
	)
	if err != nil {
		return errResult(err), nil, nil
	}

	// Filter refs.
	filtered := filterRefs(allRefs, input)

	totalUnique := countUniqueRefs(allRefs)
	matchedUnique := countUniqueRefs(filtered)

	if input.Detail {
		// Detail mode: return individual ref locations.
		paged := paginate(filtered, input.Offset, input.Limit)
		output := walkRefsOutput{
			Total:    len(allRefs),
			Matched:  len(filtered),
			Returned: len(paged),
			Details:  makeSlice[refDetail](len(paged)),
		}
		for _, ref := range paged {
			output.Details = append(output.Details, refDetail{
				Ref:        ref.Ref,
				SourcePath: ref.SourcePath,
				NodeType:   string(ref.NodeType),
			})
		}
		return nil, output, nil
	}

	// Summary mode: aggregate by ref target, sort by count desc.
	counts := make(map[string]int)
	for _, ref := range filtered {
		counts[ref.Ref]++
	}

	summaries := make([]refSummary, 0, len(counts))
	for ref, count := range counts {
		summaries = append(summaries, refSummary{Ref: ref, Count: count})
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Count != summaries[j].Count {
			return summaries[i].Count > summaries[j].Count
		}
		return summaries[i].Ref < summaries[j].Ref
	})

	paged := paginate(summaries, input.Offset, input.Limit)
	output := walkRefsOutput{
		Total:     totalUnique,
		Matched:   matchedUnique,
		Returned:  len(paged),
		Summaries: paged,
	}
	return nil, output, nil
}

// filterRefs applies target and node_type filters to refs.
func filterRefs(refs []*walker.RefInfo, input walkRefsInput) []*walker.RefInfo {
	if input.Target == "" && input.NodeType == "" {
		return refs
	}
	var filtered []*walker.RefInfo
	for _, ref := range refs {
		if input.Target != "" && !matchGlobName(ref.Ref, input.Target) {
			continue
		}
		if input.NodeType != "" && !strings.EqualFold(string(ref.NodeType), input.NodeType) {
			continue
		}
		filtered = append(filtered, ref)
	}
	return filtered
}

// countUniqueRefs returns the number of distinct ref targets.
func countUniqueRefs(refs []*walker.RefInfo) int {
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		seen[ref.Ref] = struct{}{}
	}
	return len(seen)
}
```

**Step 2: Register the tool in `server.go`**

In `registerAllTools`, add before the closing `}` (after walk_paths registration):

```go
	mcp.AddTool(server, &mcp.Tool{
		Name:        "walk_refs",
		Description: "Walk and count $ref references in an OpenAPI Specification document. By default, returns unique ref targets ranked by reference count (most-referenced first). Use target to filter to a specific ref (supports * glob, e.g. *schemas/microsoft.graph.*). Use detail=true to see individual source locations instead of counts. Filter by node_type to narrow to schema, parameter, response, requestBody, header, or pathItem refs.",
	}, handleWalkRefs)
```

NOTE: The `matchGlobName` helper was created in Task 5 in `tools_walk_schemas.go`. To keep it accessible from `tools_walk_refs.go`, it's already in the same package (`mcpserver`) — no move needed.

**Step 3: Run walk_refs tests**

Run: `go test ./internal/mcpserver/ -run "TestWalkRefs" -v`
Expected: ALL PASS

**Step 4: Run gopls diagnostics**

Run: `go_diagnostics` on `internal/mcpserver/tools_walk_refs.go` and `internal/mcpserver/server.go`

**Step 5: Run full mcpserver test suite**

Run: `go test ./internal/mcpserver/ -v -count=1`
Expected: ALL PASS

**Step 6: Commit**

```bash
git add internal/mcpserver/tools_walk_refs.go internal/mcpserver/server.go
git commit -m "feat(mcp): add walk_refs tool for $ref counting and cross-referencing

New tool walks all \$ref occurrences using walker.WithRefHandler and
WithMapRefTracking. Summary mode returns unique targets ranked by
frequency. Detail mode shows individual source locations. Supports
target glob filtering and node_type filtering."
```

---

### Task 8: Update All Tool Descriptions

**Files:**
- Modify: `internal/mcpserver/server.go:24-97` (tool descriptions)
- Modify: `internal/mcpserver/tools_walk_operations.go:14-26` (jsonschema tags)
- Modify: `internal/mcpserver/tools_walk_schemas.go:14-24` (jsonschema tags)
- Modify: `internal/mcpserver/tools_walk_parameters.go:14-24` (jsonschema tags)
- Modify: `internal/mcpserver/tools_walk_responses.go:13-22` (jsonschema tags)
- Modify: `internal/mcpserver/tools_walk_paths.go:12-18` (jsonschema tags)
- Modify: `internal/mcpserver/tools_walk_security.go:13-20` (jsonschema tags)
- Modify: `internal/mcpserver/tools_validate.go:11-15` (jsonschema tags)
- Modify: `internal/mcpserver/tools_parse.go:11-13` (jsonschema tags)
- Modify: `internal/mcpserver/tools_fix.go:15-24` (jsonschema tags)
- Modify: `internal/mcpserver/tools_diff.go:12-17` (jsonschema tags)
- Modify: `internal/mcpserver/tools_join.go:15-19` (jsonschema tags)
- Modify: `internal/mcpserver/tools_generate.go:12-17` (jsonschema tags)

**Step 1: Update all tool descriptions in `server.go`**

Use the exact description text from the design doc section "Tool Description Audit". Apply every `Description:` string change. The full list is in `docs/plans/2026-02-14-mcp-walk-usability-design.md` sections "Core Tool Description Changes" and "Walk Tool Description Changes".

**Step 2: Update all jsonschema tags**

Apply parameter description changes from the design doc section "Parameter Description Standardization":

- **Path filter** in walk_operations, walk_paths, walk_parameters, walk_responses: `"Filter by path pattern (* = one segment, ** = zero or more segments, e.g. /users/* or /drives/**/workbook/**)"`
- **Schema name** in walk_schemas: `"Filter by schema name (exact match, or glob with * and ? for pattern matching, e.g. *workbook* or microsoft.graph.*)"`
- **Tag** in walk_operations: `"Filter by tag name (exact match, case-sensitive)"`
- **operation_id** in walk_operations: `"Select a single operation by operationId (exact match)"`
- **Parameter name** in walk_parameters: `"Filter by parameter name (case-insensitive exact match)"`
- **Status** in walk_responses: `"Filter by status code: exact (200, 404), wildcard (2xx, 4xx, 5xx), or default (case-insensitive)"`
- **resolve_refs** in all walk tools: `"Resolve $ref pointers in output. Inlines referenced objects instead of showing $ref strings."`
- **component** in walk_schemas: `"Only show component schemas (defined in components/schemas or definitions). Mutually exclusive with inline."`
- **inline** in walk_schemas: `"Only show inline schemas (embedded in operations, not in components). Mutually exclusive with component."`
- **detail** in walk_schemas: `"Return full schema objects. WARNING: produces large output without name/type filters on big specs."`
- **full** in tools_parse: `"Return full parsed document instead of summary. WARNING: produces very large output for big specs — prefer walk_* tools instead."`

**Step 3: Run gopls diagnostics on all modified files**

**Step 4: Run full mcpserver tests**

Run: `go test ./internal/mcpserver/ -v -count=1`
Expected: ALL PASS (description-only changes shouldn't break tests)

**Step 5: Commit**

```bash
git add internal/mcpserver/
git commit -m "docs(mcp): audit and improve all tool descriptions for AI agent usability

Updates all 16 tool descriptions and parameter jsonschema tags with:
- Strategy hints (filter by tag first for large APIs)
- Examples in parameter descriptions (glob patterns, status wildcards)
- Explicit limitations (WARNING on full/detail for large specs)
- Consistent wording across shared parameters (resolve_refs, limit, detail)
- Accurate behavior documentation (exact match vs glob, case sensitivity)"
```

---

### Task 9: Update `explore-api` Skill with walk_refs Guidance

**Files:**
- Modify: `plugin/skills/explore-api/SKILL.md`

**Step 1: Add walk_refs to the explore-api skill**

Add a new step between the existing step 4 (drill into specifics) and step 5 (summarize):

```markdown
5. **Analyze references** with walk_refs
   - Use walk_refs (no filters) to see which schemas/responses are most-referenced
   - Use walk_refs with target filter to trace a specific schema's usage
   - Use walk_refs with node_type to narrow to schema vs response vs parameter refs
   - Use walk_refs with detail=true + target to see exact source locations
```

Also update the path pattern examples in the skill to show `**` usage.

**Step 2: Commit**

```bash
git add plugin/skills/explore-api/SKILL.md
git commit -m "docs(plugin): add walk_refs guidance to explore-api skill"
```

---

### Task 10: Run `make check` and Fix Any Issues

**Files:** Any files that fail lint/format checks

**Step 1: Run make check**

Run: `make check`
Expected: ALL PASS

**Step 2: Fix any issues found**

Common issues: trailing whitespace, import ordering, formatting.

**Step 3: Commit fixes if needed**

```bash
git commit -am "fix: address make check findings"
```

---

### Task 11: Manual Smoke Test with MS Graph Corpus

This is a verification task, not TDD.

**Step 1: Build and test walk_refs on MS Graph**

Run: `go run ./cmd/oastools mcp` in a test harness, or run the MCP integration test.

Verify that `walk_refs` on `testdata/corpus/msgraph-openapi.yaml`:
- Returns `BaseCollectionPaginationCountResponse` as the top ref
- Returns correct counts
- `target` glob filtering works
- `detail` mode shows source paths

**Step 2: Test `**` path matching on MS Graph via walk_operations**

Verify: `walk_operations` with `path: /drives/**/workbook/**` returns workbook operations.

**Step 3: Test schema name glob on MS Graph via walk_schemas**

Verify: `walk_schemas` with `name: microsoft.graph.workbook*` returns workbookRange, workbookFunctionResult, etc.
