# Walk Tool Aggregation & walk_headers Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add `group_by` aggregation to all four collector-based walk tools and introduce a `walk_headers` tool.

**Architecture:** Each walk tool gets a `group_by` string parameter that switches output from item lists to `[{key, count}]` groups. A shared `groupCount` type and `groupAndSort` helper in `server.go` handle the common logic. `walk_headers` uses raw `Walk()` + `WithHeaderHandler`. Filters apply before grouping (WHERE + GROUP BY).

**Tech Stack:** Go 1.24, MCP go-sdk, walker package, testify

---

### Task 1: Shared group_by infrastructure + walk_operations group_by

This task establishes the pattern that all subsequent tasks follow.

**Files:**
- Modify: `internal/mcpserver/server.go` (add `groupCount` type, `groupAndSort` helper, `validateGroupBy` helper)
- Modify: `internal/mcpserver/tools_walk_operations.go` (add `GroupBy` to input, `Groups` to output, grouping logic)
- Modify: `internal/mcpserver/tools_walk_operations_test.go` (add group_by tests)

**Step 1: Write failing tests**

Add these tests to `internal/mcpserver/tools_walk_operations_test.go`:

```go
func TestWalkOperations_GroupByTag(t *testing.T) {
	input := walkOperationsInput{
		Spec:    specInput{Content: walkOperationsTestSpec},
		GroupBy: "tag",
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 5, output.Total)
	assert.Equal(t, 5, output.Matched)
	require.NotEmpty(t, output.Groups)
	assert.Nil(t, output.Summaries)

	// pets tag has 3 ops (listPets, createPet, getPet), admin has 1 (deletePet), stores has 1.
	groupMap := make(map[string]int)
	for _, g := range output.Groups {
		groupMap[g.Key] = g.Count
	}
	assert.Equal(t, 3, groupMap["pets"])
	assert.Equal(t, 1, groupMap["admin"])
	assert.Equal(t, 1, groupMap["stores"])

	// Sorted descending by count.
	assert.Equal(t, "pets", output.Groups[0].Key)
}

func TestWalkOperations_GroupByMethod(t *testing.T) {
	input := walkOperationsInput{
		Spec:    specInput{Content: walkOperationsTestSpec},
		GroupBy: "method",
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 5, output.Total)
	require.NotEmpty(t, output.Groups)
	assert.Nil(t, output.Summaries)

	groupMap := make(map[string]int)
	for _, g := range output.Groups {
		groupMap[g.Key] = g.Count
	}
	assert.Equal(t, 3, groupMap["GET"])
	assert.Equal(t, 1, groupMap["POST"])
	assert.Equal(t, 1, groupMap["DELETE"])
}

func TestWalkOperations_GroupByWithFilter(t *testing.T) {
	// group_by=method, filtered to tag=pets: should show method distribution within pets tag.
	input := walkOperationsInput{
		Spec:    specInput{Content: walkOperationsTestSpec},
		Tag:     "pets",
		GroupBy: "method",
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 5, output.Total)
	assert.Equal(t, 3, output.Matched) // 3 pets-tagged ops
	groupMap := make(map[string]int)
	for _, g := range output.Groups {
		groupMap[g.Key] = g.Count
	}
	assert.Equal(t, 2, groupMap["GET"])  // listPets + getPet
	assert.Equal(t, 1, groupMap["POST"]) // createPet
}

func TestWalkOperations_GroupByAndDetailError(t *testing.T) {
	input := walkOperationsInput{
		Spec:    specInput{Content: walkOperationsTestSpec},
		GroupBy: "tag",
		Detail:  true,
	}
	result, _ := callWalkOperations(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestWalkOperations_GroupByInvalid(t *testing.T) {
	input := walkOperationsInput{
		Spec:    specInput{Content: walkOperationsTestSpec},
		GroupBy: "invalid",
	}
	result, _ := callWalkOperations(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/mcpserver/ -run "TestWalkOperations_GroupBy" -v`
Expected: FAIL — `Groups` field does not exist on `walkOperationsOutput`

**Step 3: Add shared infrastructure to server.go**

Add after the `errResult` function in `internal/mcpserver/server.go`:

```go
// groupCount represents a single group in group_by results.
type groupCount struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

// groupAndSort groups items by key, sorts by count descending (ties
// broken alphabetically by key), and returns the sorted groups.
func groupAndSort[T any](items []T, keyFn func(T) []string) []groupCount {
	counts := make(map[string]int)
	for _, item := range items {
		for _, key := range keyFn(item) {
			counts[key]++
		}
	}
	groups := make([]groupCount, 0, len(counts))
	for key, count := range counts {
		groups = append(groups, groupCount{Key: key, Count: count})
	}
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Count != groups[j].Count {
			return groups[i].Count > groups[j].Count
		}
		return groups[i].Key < groups[j].Key
	})
	return groups
}

// validateGroupBy checks that group_by is a valid value and is not combined with detail.
func validateGroupBy(groupBy string, detail bool, allowed []string) error {
	if groupBy == "" {
		return nil
	}
	if detail {
		return fmt.Errorf("cannot use both group_by and detail")
	}
	for _, a := range allowed {
		if strings.EqualFold(groupBy, a) {
			return nil
		}
	}
	return fmt.Errorf("invalid group_by value %q; valid values: %s", groupBy, strings.Join(allowed, ", "))
}
```

Add `"fmt"`, `"sort"`, and `"strings"` to the imports if not already present (check — `sort` is likely new).

**Step 4: Add group_by to walk_operations**

In `internal/mcpserver/tools_walk_operations.go`:

1. Add to `walkOperationsInput`:
```go
GroupBy string `json:"group_by,omitempty" jsonschema:"Group results and return counts instead of individual items. Values: tag\\, method"`
```

2. Add to `walkOperationsOutput`:
```go
Groups []groupCount `json:"groups,omitempty"`
```

3. In `handleWalkOperations`, add validation after `input.Spec.resolve()`:
```go
if err := validateGroupBy(input.GroupBy, input.Detail, []string{"tag", "method"}); err != nil {
    return errResult(err), nil, nil
}
```

4. After filtering, add the group_by branch before the existing summary/detail logic:
```go
if input.GroupBy != "" {
    groups := groupAndSort(matched, func(op *walker.OperationInfo) []string {
        switch strings.ToLower(input.GroupBy) {
        case "tag":
            if len(op.Operation.Tags) == 0 {
                return nil
            }
            return op.Operation.Tags
        case "method":
            return []string{strings.ToUpper(op.Method)}
        default:
            return nil
        }
    })
    paged := paginate(groups, input.Offset, input.Limit)
    output := walkOperationsOutput{
        Total:    len(collector.All),
        Matched:  len(matched),
        Returned: len(paged),
        Groups:   paged,
    }
    return nil, output, nil
}
```

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/mcpserver/ -run "TestWalkOperations" -v`
Expected: ALL PASS

**Step 6: Commit**

```bash
git add internal/mcpserver/server.go internal/mcpserver/tools_walk_operations.go internal/mcpserver/tools_walk_operations_test.go
git commit -m "feat(mcp): add group_by aggregation to walk_operations"
```

---

### Task 2: walk_schemas group_by

**Files:**
- Modify: `internal/mcpserver/tools_walk_schemas.go` (add `GroupBy` to input, `Groups` to output)
- Modify: `internal/mcpserver/tools_walk_schemas_test.go` (add group_by tests)

**Step 1: Write failing tests**

Add to `internal/mcpserver/tools_walk_schemas_test.go`:

```go
func TestWalkSchemas_GroupByType(t *testing.T) {
	input := walkSchemasInput{
		Spec:      specInput{Content: walkSchemasTestSpec},
		Component: true,
		GroupBy:   "type",
	}
	_, output := callWalkSchemas(t, input)

	assert.Equal(t, 8, output.Matched)
	require.NotEmpty(t, output.Groups)
	assert.Nil(t, output.Summaries)

	groupMap := make(map[string]int)
	for _, g := range output.Groups {
		groupMap[g.Key] = g.Count
	}
	// Pet(object) + Error(object) = 2 object schemas, Tag(string) = 1 string schema,
	// plus property schemas: id(integer)*2, name(string)*1, tag(string)*1,
	// code(integer)*1, message(string)*1. Total: object=2, string=3, integer=3
	assert.Greater(t, groupMap["object"], 0)
	assert.Greater(t, groupMap["string"], 0)
	assert.Greater(t, groupMap["integer"], 0)
}

func TestWalkSchemas_GroupByLocation(t *testing.T) {
	input := walkSchemasInput{
		Spec:    specInput{Content: walkSchemasTestSpec},
		GroupBy: "location",
	}
	_, output := callWalkSchemas(t, input)

	require.NotEmpty(t, output.Groups)
	assert.Nil(t, output.Summaries)

	groupMap := make(map[string]int)
	for _, g := range output.Groups {
		groupMap[g.Key] = g.Count
	}
	assert.Greater(t, groupMap["component"], 0)
	assert.Greater(t, groupMap["inline"], 0)
}

func TestWalkSchemas_GroupByAndDetailError(t *testing.T) {
	input := walkSchemasInput{
		Spec:    specInput{Content: walkSchemasTestSpec},
		GroupBy: "type",
		Detail:  true,
	}
	result, _ := callWalkSchemas(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestWalkSchemas_GroupByInvalid(t *testing.T) {
	input := walkSchemasInput{
		Spec:    specInput{Content: walkSchemasTestSpec},
		GroupBy: "invalid",
	}
	result, _ := callWalkSchemas(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/mcpserver/ -run "TestWalkSchemas_GroupBy" -v`
Expected: FAIL

**Step 3: Implement**

In `internal/mcpserver/tools_walk_schemas.go`:

1. Add to `walkSchemasInput`:
```go
GroupBy string `json:"group_by,omitempty" jsonschema:"Group results and return counts instead of individual items. Values: type\\, location"`
```

2. Add to `walkSchemasOutput`:
```go
Groups []groupCount `json:"groups,omitempty"`
```

3. In `handleWalkSchemas`, add validation right after the `component && inline` check:
```go
if err := validateGroupBy(input.GroupBy, input.Detail, []string{"type", "location"}); err != nil {
    return errResult(err), nil, nil
}
```

4. After filtering, add the group_by branch:
```go
if input.GroupBy != "" {
    groups := groupAndSort(filtered, func(info *walker.SchemaInfo) []string {
        switch strings.ToLower(input.GroupBy) {
        case "type":
            t := schemaTypeString(info.Schema.Type)
            if t == "" {
                return []string{""}
            }
            return []string{t}
        case "location":
            return []string{schemaLocation(info.IsComponent)}
        default:
            return nil
        }
    })
    paged := paginate(groups, input.Offset, input.Limit)
    output := walkSchemasOutput{
        Total:    len(collector.All),
        Matched:  len(filtered),
        Returned: len(paged),
        Groups:   paged,
    }
    return nil, output, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/mcpserver/ -run "TestWalkSchemas" -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/mcpserver/tools_walk_schemas.go internal/mcpserver/tools_walk_schemas_test.go
git commit -m "feat(mcp): add group_by aggregation to walk_schemas"
```

---

### Task 3: walk_parameters group_by

**Files:**
- Modify: `internal/mcpserver/tools_walk_parameters.go`
- Modify: `internal/mcpserver/tools_walk_parameters_test.go`

**Step 1: Write failing tests**

Add to `internal/mcpserver/tools_walk_parameters_test.go`:

```go
func TestWalkParameters_GroupByLocation(t *testing.T) {
	input := walkParametersInput{
		Spec:    specInput{Content: walkParametersTestSpec},
		GroupBy: "location",
	}
	_, output := callWalkParameters(t, input)

	assert.Equal(t, 4, output.Total)
	require.NotEmpty(t, output.Groups)
	assert.Nil(t, output.Summaries)

	groupMap := make(map[string]int)
	for _, g := range output.Groups {
		groupMap[g.Key] = g.Count
	}
	assert.Equal(t, 2, groupMap["query"])  // limit, offset
	assert.Equal(t, 1, groupMap["header"]) // X-Request-Id
	assert.Equal(t, 1, groupMap["path"])   // petId
}

func TestWalkParameters_GroupByName(t *testing.T) {
	input := walkParametersInput{
		Spec:    specInput{Content: walkParametersTestSpec},
		GroupBy: "name",
	}
	_, output := callWalkParameters(t, input)

	require.NotEmpty(t, output.Groups)
	// Each parameter has a unique name in test spec, so each group has count 1.
	for _, g := range output.Groups {
		assert.Equal(t, 1, g.Count)
	}
}

func TestWalkParameters_GroupByWithFilter(t *testing.T) {
	input := walkParametersInput{
		Spec:    specInput{Content: walkParametersTestSpec},
		Path:    "/pets",
		GroupBy: "location",
	}
	_, output := callWalkParameters(t, input)

	assert.Equal(t, 3, output.Matched)
	groupMap := make(map[string]int)
	for _, g := range output.Groups {
		groupMap[g.Key] = g.Count
	}
	assert.Equal(t, 2, groupMap["query"])
	assert.Equal(t, 1, groupMap["header"])
}

func TestWalkParameters_GroupByAndDetailError(t *testing.T) {
	input := walkParametersInput{
		Spec:    specInput{Content: walkParametersTestSpec},
		GroupBy: "location",
		Detail:  true,
	}
	result, _ := callWalkParameters(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestWalkParameters_GroupByInvalid(t *testing.T) {
	input := walkParametersInput{
		Spec:    specInput{Content: walkParametersTestSpec},
		GroupBy: "invalid",
	}
	result, _ := callWalkParameters(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/mcpserver/ -run "TestWalkParameters_GroupBy" -v`
Expected: FAIL

**Step 3: Implement**

In `internal/mcpserver/tools_walk_parameters.go`:

1. Add to `walkParametersInput`:
```go
GroupBy string `json:"group_by,omitempty" jsonschema:"Group results and return counts instead of individual items. Values: location\\, name"`
```

2. Add to `walkParametersOutput`:
```go
Groups []groupCount `json:"groups,omitempty"`
```

3. In `handleWalkParameters`, add validation after `input.Spec.resolve()`:
```go
if err := validateGroupBy(input.GroupBy, input.Detail, []string{"location", "name"}); err != nil {
    return errResult(err), nil, nil
}
```

4. After filtering, add group_by branch:
```go
if input.GroupBy != "" {
    groups := groupAndSort(matched, func(info *walker.ParameterInfo) []string {
        switch strings.ToLower(input.GroupBy) {
        case "location":
            return []string{info.In}
        case "name":
            return []string{info.Name}
        default:
            return nil
        }
    })
    paged := paginate(groups, input.Offset, input.Limit)
    output := walkParametersOutput{
        Total:    len(collector.All),
        Matched:  len(matched),
        Returned: len(paged),
        Groups:   paged,
    }
    return nil, output, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/mcpserver/ -run "TestWalkParameters" -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/mcpserver/tools_walk_parameters.go internal/mcpserver/tools_walk_parameters_test.go
git commit -m "feat(mcp): add group_by aggregation to walk_parameters"
```

---

### Task 4: walk_responses group_by

**Files:**
- Modify: `internal/mcpserver/tools_walk_responses.go`
- Modify: `internal/mcpserver/tools_walk_responses_test.go`

**Step 1: Write failing tests**

Add to `internal/mcpserver/tools_walk_responses_test.go`. The test spec for responses (`walkResponsesTestSpec`) should be checked first — read the file to confirm the fixture. Use the same `walkOperationsTestSpec` which has responses at 200, 201, 204.

Actually, use `walkResponsesTestSpec` if it exists, or define a local spec. Check the test file first. The response test fixture likely has several status codes. Use this test:

```go
func TestWalkResponses_GroupByStatusCode(t *testing.T) {
	input := walkResponsesInput{
		Spec:    specInput{Content: walkResponsesTestSpec},
		GroupBy: "status_code",
	}
	_, output := callWalkResponses(t, input)

	require.NotEmpty(t, output.Groups)
	assert.Nil(t, output.Summaries)

	// Verify groups are sorted descending by count.
	for i := 1; i < len(output.Groups); i++ {
		assert.GreaterOrEqual(t, output.Groups[i-1].Count, output.Groups[i].Count)
	}
}

func TestWalkResponses_GroupByMethod(t *testing.T) {
	input := walkResponsesInput{
		Spec:    specInput{Content: walkResponsesTestSpec},
		GroupBy: "method",
	}
	_, output := callWalkResponses(t, input)

	require.NotEmpty(t, output.Groups)
	assert.Nil(t, output.Summaries)

	// All keys should be uppercase methods.
	for _, g := range output.Groups {
		assert.Equal(t, strings.ToUpper(g.Key), g.Key)
	}
}

func TestWalkResponses_GroupByWithFilter(t *testing.T) {
	input := walkResponsesInput{
		Spec:    specInput{Content: walkResponsesTestSpec},
		Status:  "2xx",
		GroupBy: "method",
	}
	_, output := callWalkResponses(t, input)

	assert.Greater(t, output.Matched, 0)
	require.NotEmpty(t, output.Groups)
}

func TestWalkResponses_GroupByAndDetailError(t *testing.T) {
	input := walkResponsesInput{
		Spec:    specInput{Content: walkResponsesTestSpec},
		GroupBy: "status_code",
		Detail:  true,
	}
	result, _ := callWalkResponses(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestWalkResponses_GroupByInvalid(t *testing.T) {
	input := walkResponsesInput{
		Spec:    specInput{Content: walkResponsesTestSpec},
		GroupBy: "invalid",
	}
	result, _ := callWalkResponses(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}
```

Note: If `walkResponsesTestSpec` is not defined in the test file, check the file and use whatever constant is defined. If needed, use `walkOperationsTestSpec` which also has responses.

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/mcpserver/ -run "TestWalkResponses_GroupBy" -v`
Expected: FAIL

**Step 3: Implement**

In `internal/mcpserver/tools_walk_responses.go`:

1. Add to `walkResponsesInput`:
```go
GroupBy string `json:"group_by,omitempty" jsonschema:"Group results and return counts instead of individual items. Values: status_code\\, method"`
```

2. Add to `walkResponsesOutput`:
```go
Groups []groupCount `json:"groups,omitempty"`
```

3. In `handleWalkResponses`, add validation after `input.Spec.resolve()`:
```go
if err := validateGroupBy(input.GroupBy, input.Detail, []string{"status_code", "method"}); err != nil {
    return errResult(err), nil, nil
}
```

4. After filtering, add group_by branch:
```go
if input.GroupBy != "" {
    groups := groupAndSort(matched, func(info *walker.ResponseInfo) []string {
        switch strings.ToLower(input.GroupBy) {
        case "status_code":
            return []string{info.StatusCode}
        case "method":
            return []string{strings.ToUpper(info.Method)}
        default:
            return nil
        }
    })
    paged := paginate(groups, input.Offset, input.Limit)
    output := walkResponsesOutput{
        Total:    len(collector.All),
        Matched:  len(matched),
        Returned: len(paged),
        Groups:   paged,
    }
    return nil, output, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/mcpserver/ -run "TestWalkResponses" -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add internal/mcpserver/tools_walk_responses.go internal/mcpserver/tools_walk_responses_test.go
git commit -m "feat(mcp): add group_by aggregation to walk_responses"
```

---

### Task 5: walk_headers tool

**Files:**
- Create: `internal/mcpserver/tools_walk_headers.go`
- Create: `internal/mcpserver/tools_walk_headers_test.go`
- Modify: `internal/mcpserver/server.go` (register tool)
- Modify: `internal/mcpserver/integration_test.go` (16 -> 17 tools, add walk_headers)

**Step 1: Write failing tests**

Create `internal/mcpserver/tools_walk_headers_test.go`:

```go
package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const walkHeadersTestSpec = `openapi: "3.0.0"
info:
  title: Walk Headers Test
  version: "1.0.0"
paths:
  /pets:
    get:
      summary: List pets
      responses:
        "200":
          description: OK
          headers:
            X-Rate-Limit:
              description: Rate limit per hour
              schema:
                type: integer
            X-Request-Id:
              description: Request identifier
              schema:
                type: string
        "404":
          description: Not found
          headers:
            X-Request-Id:
              description: Request identifier
              schema:
                type: string
    post:
      summary: Create a pet
      responses:
        "201":
          description: Created
          headers:
            X-Request-Id:
              description: Request identifier
              schema:
                type: string
            Location:
              description: URL of created resource
              required: true
              schema:
                type: string
  /stores:
    get:
      summary: List stores
      responses:
        "200":
          description: OK
          headers:
            X-Rate-Limit:
              description: Rate limit per hour
              schema:
                type: integer
components:
  headers:
    TraceId:
      description: Distributed tracing identifier
      schema:
        type: string
`

func callWalkHeaders(t *testing.T, input walkHeadersInput) (*mcp.CallToolResult, walkHeadersOutput) {
	t.Helper()
	result, out, err := handleWalkHeaders(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	if out == nil {
		return result, walkHeadersOutput{}
	}
	wo, ok := out.(walkHeadersOutput)
	require.True(t, ok, "expected walkHeadersOutput, got %T", out)
	return result, wo
}

func TestWalkHeaders_AllHeaders(t *testing.T) {
	input := walkHeadersInput{
		Spec: specInput{Content: walkHeadersTestSpec},
	}
	_, output := callWalkHeaders(t, input)

	// 5 response headers + 1 component header = 6 total.
	assert.Equal(t, 6, output.Total)
	assert.Equal(t, 6, output.Matched)
	require.Len(t, output.Summaries, 6)
}

func TestWalkHeaders_FilterByName(t *testing.T) {
	input := walkHeadersInput{
		Spec: specInput{Content: walkHeadersTestSpec},
		Name: "X-Rate-Limit",
	}
	_, output := callWalkHeaders(t, input)

	assert.Equal(t, 2, output.Matched)
	for _, s := range output.Summaries {
		assert.Equal(t, "X-Rate-Limit", s.Name)
	}
}

func TestWalkHeaders_FilterByNameGlob(t *testing.T) {
	input := walkHeadersInput{
		Spec: specInput{Content: walkHeadersTestSpec},
		Name: "X-*",
	}
	_, output := callWalkHeaders(t, input)

	// X-Rate-Limit (2) + X-Request-Id (3) = 5
	assert.Equal(t, 5, output.Matched)
}

func TestWalkHeaders_FilterByPath(t *testing.T) {
	input := walkHeadersInput{
		Spec: specInput{Content: walkHeadersTestSpec},
		Path: "/pets",
	}
	_, output := callWalkHeaders(t, input)

	// /pets GET 200 has 2 headers, GET 404 has 1, POST 201 has 2 = 5.
	assert.Equal(t, 5, output.Matched)
	for _, s := range output.Summaries {
		assert.Equal(t, "/pets", s.Path)
	}
}

func TestWalkHeaders_FilterByMethod(t *testing.T) {
	input := walkHeadersInput{
		Spec:   specInput{Content: walkHeadersTestSpec},
		Method: "post",
	}
	_, output := callWalkHeaders(t, input)

	assert.Equal(t, 2, output.Matched) // POST /pets 201 has X-Request-Id + Location
	for _, s := range output.Summaries {
		assert.Equal(t, "POST", s.Method)
	}
}

func TestWalkHeaders_FilterByStatus(t *testing.T) {
	input := walkHeadersInput{
		Spec:   specInput{Content: walkHeadersTestSpec},
		Status: "404",
	}
	_, output := callWalkHeaders(t, input)

	assert.Equal(t, 1, output.Matched)
	assert.Equal(t, "X-Request-Id", output.Summaries[0].Name)
	assert.Equal(t, "404", output.Summaries[0].Status)
}

func TestWalkHeaders_ComponentFilter(t *testing.T) {
	input := walkHeadersInput{
		Spec:      specInput{Content: walkHeadersTestSpec},
		Component: true,
	}
	_, output := callWalkHeaders(t, input)

	assert.Equal(t, 1, output.Matched)
	assert.Equal(t, "TraceId", output.Summaries[0].Name)
}

func TestWalkHeaders_DetailMode(t *testing.T) {
	input := walkHeadersInput{
		Spec:   specInput{Content: walkHeadersTestSpec},
		Name:   "Location",
		Detail: true,
	}
	_, output := callWalkHeaders(t, input)

	assert.Equal(t, 1, output.Matched)
	assert.Nil(t, output.Summaries)
	require.Len(t, output.Headers, 1)
	assert.Equal(t, "Location", output.Headers[0].Name)
	assert.NotNil(t, output.Headers[0].Header)
	assert.True(t, output.Headers[0].Header.Required)
}

func TestWalkHeaders_GroupByName(t *testing.T) {
	input := walkHeadersInput{
		Spec:    specInput{Content: walkHeadersTestSpec},
		GroupBy: "name",
	}
	_, output := callWalkHeaders(t, input)

	require.NotEmpty(t, output.Groups)
	assert.Nil(t, output.Summaries)

	groupMap := make(map[string]int)
	for _, g := range output.Groups {
		groupMap[g.Key] = g.Count
	}
	assert.Equal(t, 3, groupMap["X-Request-Id"]) // appears in 3 responses
	assert.Equal(t, 2, groupMap["X-Rate-Limit"])  // appears in 2 responses
	assert.Equal(t, 1, groupMap["Location"])       // appears in 1 response
	assert.Equal(t, 1, groupMap["TraceId"])        // component header

	// Most-referenced first.
	assert.Equal(t, "X-Request-Id", output.Groups[0].Key)
}

func TestWalkHeaders_GroupByStatusCode(t *testing.T) {
	input := walkHeadersInput{
		Spec:    specInput{Content: walkHeadersTestSpec},
		GroupBy: "status_code",
	}
	_, output := callWalkHeaders(t, input)

	require.NotEmpty(t, output.Groups)
	groupMap := make(map[string]int)
	for _, g := range output.Groups {
		groupMap[g.Key] = g.Count
	}
	assert.Equal(t, 3, groupMap["200"])  // GET /pets 200 (2) + GET /stores 200 (1)
	assert.Equal(t, 2, groupMap["201"])  // POST /pets 201 (2)
	assert.Equal(t, 1, groupMap["404"])  // GET /pets 404 (1)
	// Component header (TraceId) has no status code — excluded from status_code grouping.
}

func TestWalkHeaders_Pagination(t *testing.T) {
	input := walkHeadersInput{
		Spec:  specInput{Content: walkHeadersTestSpec},
		Limit: 2,
	}
	_, output := callWalkHeaders(t, input)

	assert.Equal(t, 6, output.Total)
	assert.Equal(t, 6, output.Matched)
	assert.Equal(t, 2, output.Returned)
	assert.Len(t, output.Summaries, 2)
}

func TestWalkHeaders_GroupByAndDetailError(t *testing.T) {
	input := walkHeadersInput{
		Spec:    specInput{Content: walkHeadersTestSpec},
		GroupBy: "name",
		Detail:  true,
	}
	result, _ := callWalkHeaders(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestWalkHeaders_InvalidSpec(t *testing.T) {
	input := walkHeadersInput{
		Spec: specInput{Content: "not valid yaml: ["},
	}
	result, _ := callWalkHeaders(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/mcpserver/ -run "TestWalkHeaders" -v`
Expected: FAIL — `walkHeadersInput` type does not exist

**Step 3: Implement walk_headers**

Create `internal/mcpserver/tools_walk_headers.go`:

```go
package mcpserver

import (
	"context"
	"strings"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type walkHeadersInput struct {
	Spec        specInput `json:"spec"                     jsonschema:"The OAS document to walk"`
	Name        string    `json:"name,omitempty"           jsonschema:"Filter by header name (exact match\\, or glob with * and ? for pattern matching\\, e.g. X-Rate-* or *Token*)"`
	Path        string    `json:"path,omitempty"           jsonschema:"Filter by path pattern (* = one segment\\, ** = zero or more segments)"`
	Method      string    `json:"method,omitempty"         jsonschema:"Filter by HTTP method (get\\, post\\, put\\, delete\\, patch\\, etc.)"`
	Status      string    `json:"status,omitempty"         jsonschema:"Filter by status code: exact (200\\, 404)\\, wildcard (2xx\\, 4xx)\\, or default"`
	Component   bool      `json:"component,omitempty"      jsonschema:"Only show component headers (defined in components/headers)"`
	ResolveRefs bool      `json:"resolve_refs,omitempty"   jsonschema:"Resolve $ref pointers in output. Inlines referenced objects instead of showing $ref strings."`
	Detail      bool      `json:"detail,omitempty"         jsonschema:"Return full header objects instead of summaries"`
	GroupBy     string    `json:"group_by,omitempty"       jsonschema:"Group results and return counts instead of individual items. Values: name\\, status_code"`
	Limit       int       `json:"limit,omitempty"          jsonschema:"Maximum results (default 100)"`
	Offset      int       `json:"offset,omitempty"         jsonschema:"Skip the first N results (for pagination)"`
}

type headerInfo struct {
	Header       *parser.Header
	Name         string
	JSONPath     string
	PathTemplate string
	Method       string
	StatusCode   string
	IsComponent  bool
}

type headerSummary struct {
	Name        string `json:"name"`
	Path        string `json:"path,omitempty"`
	Method      string `json:"method,omitempty"`
	Status      string `json:"status,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Deprecated  bool   `json:"deprecated,omitempty"`
}

type headerDetail struct {
	Name     string         `json:"name"`
	Path     string         `json:"path,omitempty"`
	Method   string         `json:"method,omitempty"`
	Status   string         `json:"status,omitempty"`
	Header   *parser.Header `json:"header"`
}

type walkHeadersOutput struct {
	Total     int             `json:"total"`
	Matched   int             `json:"matched"`
	Returned  int             `json:"returned"`
	Summaries []headerSummary `json:"summaries,omitempty"`
	Headers   []headerDetail  `json:"headers,omitempty"`
	Groups    []groupCount    `json:"groups,omitempty"`
}

func handleWalkHeaders(_ context.Context, _ *mcp.CallToolRequest, input walkHeadersInput) (*mcp.CallToolResult, any, error) {
	if err := validateGroupBy(input.GroupBy, input.Detail, []string{"name", "status_code"}); err != nil {
		return errResult(err), nil, nil
	}

	var extraOpts []parser.Option
	if input.ResolveRefs {
		extraOpts = append(extraOpts, parser.WithResolveRefs(true))
	}

	result, err := input.Spec.resolve(extraOpts...)
	if err != nil {
		return errResult(err), nil, nil
	}

	// Collect all headers via the walker.
	var allHeaders []*headerInfo
	err = walker.Walk(result,
		walker.WithHeaderHandler(func(wc *walker.WalkContext, header *parser.Header) walker.Action {
			allHeaders = append(allHeaders, &headerInfo{
				Header:       header,
				Name:         wc.Name,
				JSONPath:     wc.JSONPath,
				PathTemplate: wc.PathTemplate,
				Method:       wc.Method,
				StatusCode:   wc.StatusCode,
				IsComponent:  wc.IsComponent,
			})
			return walker.Continue
		}),
	)
	if err != nil {
		return errResult(err), nil, nil
	}

	// Filter headers.
	filtered := filterWalkHeaders(allHeaders, input)

	// Group-by mode.
	if input.GroupBy != "" {
		groups := groupAndSort(filtered, func(h *headerInfo) []string {
			switch strings.ToLower(input.GroupBy) {
			case "name":
				return []string{h.Name}
			case "status_code":
				if h.StatusCode == "" {
					return nil // component headers have no status code
				}
				return []string{h.StatusCode}
			default:
				return nil
			}
		})
		paged := paginate(groups, input.Offset, input.Limit)
		output := walkHeadersOutput{
			Total:    len(allHeaders),
			Matched:  len(filtered),
			Returned: len(paged),
			Groups:   paged,
		}
		return nil, output, nil
	}

	// Apply pagination.
	returned := paginate(filtered, input.Offset, input.Limit)

	output := walkHeadersOutput{
		Total:    len(allHeaders),
		Matched:  len(filtered),
		Returned: len(returned),
	}

	if input.Detail {
		output.Headers = makeSlice[headerDetail](len(returned))
		for _, h := range returned {
			output.Headers = append(output.Headers, headerDetail{
				Name:   h.Name,
				Path:   h.PathTemplate,
				Method: strings.ToUpper(h.Method),
				Status: h.StatusCode,
				Header: h.Header,
			})
		}
	} else {
		output.Summaries = makeSlice[headerSummary](len(returned))
		for _, h := range returned {
			output.Summaries = append(output.Summaries, headerSummary{
				Name:        h.Name,
				Path:        h.PathTemplate,
				Method:      strings.ToUpper(h.Method),
				Status:      h.StatusCode,
				Description: h.Header.Description,
				Required:    h.Header.Required,
				Deprecated:  h.Header.Deprecated,
			})
		}
	}

	return nil, output, nil
}

// filterWalkHeaders applies name, path, method, status, and component filters.
func filterWalkHeaders(headers []*headerInfo, input walkHeadersInput) []*headerInfo {
	if input.Name == "" && input.Path == "" && input.Method == "" && input.Status == "" && !input.Component {
		return headers
	}
	var filtered []*headerInfo
	for _, h := range headers {
		if input.Name != "" && !matchGlobName(h.Name, input.Name) {
			continue
		}
		if input.Path != "" && !matchWalkPath(h.PathTemplate, input.Path) {
			continue
		}
		if input.Method != "" && !strings.EqualFold(h.Method, input.Method) {
			continue
		}
		if input.Status != "" && !statusCodeMatches(h.StatusCode, input.Status) {
			continue
		}
		if input.Component && !h.IsComponent {
			continue
		}
		filtered = append(filtered, h)
	}
	return filtered
}
```

**Step 4: Register in server.go**

Add to `registerAllTools` in `internal/mcpserver/server.go`:

```go
mcp.AddTool(server, &mcp.Tool{
    Name:        "walk_headers",
    Description: "Walk and query response headers and component headers in an OpenAPI Specification document. Filter by name, path, method, status code, or component location. Returns summaries (name, path, method, status, description) by default or full header objects with detail=true. Use group_by=name to find the most commonly used headers across the API.",
}, handleWalkHeaders)
```

**Step 5: Update integration test**

In `internal/mcpserver/integration_test.go`:
- Change `assert.Len(t, result.Tools, 16` to `assert.Len(t, result.Tools, 17`
- Change comment from `16 tools: 9 core + 7 walk` to `17 tools: 9 core + 8 walk`
- Add `"walk_headers"` to the `expectedTools` slice

**Step 6: Run tests**

Run: `go test ./internal/mcpserver/ -run "TestWalkHeaders|TestIntegration_ListTools" -v`
Expected: ALL PASS

**Step 7: Commit**

```bash
git add internal/mcpserver/tools_walk_headers.go internal/mcpserver/tools_walk_headers_test.go internal/mcpserver/server.go internal/mcpserver/integration_test.go
git commit -m "feat(mcp): add walk_headers tool with group_by support"
```

---

### Task 6: Update tool descriptions for group_by

**Files:**
- Modify: `internal/mcpserver/server.go` (update 4 walk tool descriptions to mention group_by)

**Step 1: Update descriptions**

Update the description for each tool that now has `group_by`:

- **walk_operations**: Append to description: `Use group_by (tag or method) to get distribution counts instead of individual items.`
- **walk_schemas**: Append to description: `Use group_by (type or location) to get distribution counts instead of individual items.`
- **walk_parameters**: Append to description: `Use group_by (location or name) to get distribution counts instead of individual items.`
- **walk_responses**: Append to description: `Use group_by (status_code or method) to get distribution counts instead of individual items.`

**Step 2: Run integration test**

Run: `go test ./internal/mcpserver/ -run "TestIntegration" -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/mcpserver/server.go
git commit -m "docs(mcp): update walk tool descriptions to document group_by"
```

---

### Task 7: Plugin guidance updates

**Files:**
- Modify: `plugin/CLAUDE.md`
- Modify: `plugin/skills/explore-api/SKILL.md`

**Step 1: Update plugin/CLAUDE.md**

1. Change tool counts: `16` -> `17` everywhere
2. Change Walk tool list: add `walk_headers`
3. After best practice #5 about filtering, add: `8. **Aggregate with `group_by`.** The walk tools (`walk_operations`, `walk_schemas`, `walk_parameters`, `walk_responses`, `walk_headers`) support a `group_by` parameter that returns `{key, count}` groups instead of individual items. Use this for distribution questions ("how many operations per tag?") instead of paging through all results.`
4. In the "Explore a large API" workflow, add between steps 3 and 4: `3.5. `walk_operations` with `group_by: "tag"` — operation count per tag at a glance`

**Step 2: Update explore-api SKILL.md**

1. In Step 2 (List endpoints), add before the filtering guidance:
```markdown
For a quick overview of a large API, use `group_by` first:

```json
{"spec": {"file": "<path>"}, "group_by": "tag"}
```

This returns operation counts per tag — the fastest way to understand API scope.

```json
{"spec": {"file": "<path>"}, "group_by": "method"}
```

This shows the HTTP method distribution (e.g., 60% GET, 25% POST, etc.).
```

2. In Step 3 (List data models), add:
```markdown
Get schema type distribution:

```json
{"spec": {"file": "<path>"}, "group_by": "type", "component": true}
```
```

3. In Step 4 (Drill into specifics), add a section for headers:
```markdown
**Response headers:**

```json
{"spec": {"file": "<path>"}, "group_by": "name"}
```

(using `walk_headers` — shows which headers are most common across the API)

**Headers for a specific endpoint:**

```json
{"spec": {"file": "<path>"}, "path": "/users", "method": "get"}
```

(using `walk_headers`)
```

**Step 3: Commit**

```bash
git add plugin/CLAUDE.md plugin/skills/explore-api/SKILL.md
git commit -m "docs(plugin): add group_by and walk_headers guidance to plugin skills"
```

---

### Task 8: make check

**Files:** None (validation only)

**Step 1: Run make check**

Run: `make check`
Expected: 0 issues, all tests pass

**Step 2: Fix any issues**

If `go fmt` or linter issues are found, fix them. Common issues:
- Missing imports (e.g., `"sort"` in server.go)
- Unused variables from copy-paste
- `unparam` warnings if any test helpers have unused return values

**Step 3: Commit fixes if needed**

```bash
git add -A
git commit -m "fix: address make check findings for walk aggregation"
```

---

### Task 9: Smoke test against MS Graph corpus

**Files:** None (manual verification)

**Step 1: Run group_by tests against MS Graph**

Write a temporary test in `internal/mcpserver/` (delete after):

```go
func TestCorpusSmoke_GroupBy(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus smoke test")
	}
	spec := specInput{File: "../../corpus/microsoft-graph.yaml"}

	// walk_operations group_by=tag: should return tag distribution.
	_, opOut := callWalkOperations(t, walkOperationsInput{Spec: spec, GroupBy: "tag"})
	t.Logf("Operations: %d total, %d tag groups, top: %s (%d)",
		opOut.Total, len(opOut.Groups), opOut.Groups[0].Key, opOut.Groups[0].Count)
	assert.Greater(t, len(opOut.Groups), 10)

	// walk_operations group_by=method: should return method distribution.
	_, methodOut := callWalkOperations(t, walkOperationsInput{Spec: spec, GroupBy: "method"})
	t.Logf("Methods: %d groups", len(methodOut.Groups))
	assert.Greater(t, len(methodOut.Groups), 2)

	// walk_schemas group_by=type: schema type distribution.
	_, schemaOut := callWalkSchemas(t, walkSchemasInput{Spec: spec, Component: true, GroupBy: "type"})
	t.Logf("Schema types: %d groups, top: %s (%d)",
		len(schemaOut.Groups), schemaOut.Groups[0].Key, schemaOut.Groups[0].Count)

	// walk_headers group_by=name: header distribution.
	_, headerOut := callWalkHeaders(t, walkHeadersInput{Spec: spec, GroupBy: "name"})
	t.Logf("Headers: %d total, %d unique names, top: %s (%d)",
		headerOut.Total, len(headerOut.Groups), headerOut.Groups[0].Key, headerOut.Groups[0].Count)
}
```

Run: `go test ./internal/mcpserver/ -run "TestCorpusSmoke_GroupBy" -v -timeout 120s`

Verify all pass and numbers look reasonable. Then delete the temporary test file.

**Step 2: No commit needed** (temp file deleted)
