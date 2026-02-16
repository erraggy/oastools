package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const walkParametersTestSpec = `openapi: "3.0.0"
info:
  title: Walk Params Test
  version: "1.0.0"
paths:
  /pets:
    get:
      summary: List pets
      parameters:
        - name: limit
          in: query
          required: false
          schema:
            type: integer
        - name: offset
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: OK
    post:
      summary: Create a pet
      parameters:
        - name: X-Request-Id
          in: header
          required: true
          schema:
            type: string
      responses:
        "201":
          description: Created
  /pets/{petId}:
    get:
      summary: Get a pet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: OK
`

func callWalkParameters(t *testing.T, input walkParametersInput) (*mcp.CallToolResult, walkParametersOutput) {
	t.Helper()
	result, out, err := handleWalkParameters(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	if out == nil {
		return result, walkParametersOutput{}
	}
	wo, ok := out.(walkParametersOutput)
	require.True(t, ok, "expected walkParametersOutput, got %T", out)
	return result, wo
}

func TestWalkParameters_AllParameters(t *testing.T) {
	input := walkParametersInput{
		Spec: specInput{Content: walkParametersTestSpec},
	}
	_, output := callWalkParameters(t, input)

	assert.Equal(t, 4, output.Total)
	assert.Equal(t, 4, output.Matched)
	assert.Equal(t, 4, output.Returned)
	require.Len(t, output.Summaries, 4)
}

func TestWalkParameters_FilterByIn(t *testing.T) {
	input := walkParametersInput{
		Spec: specInput{Content: walkParametersTestSpec},
		In:   "query",
	}
	_, output := callWalkParameters(t, input)

	assert.Equal(t, 2, output.Matched)
	require.Len(t, output.Summaries, 2)
	for _, s := range output.Summaries {
		assert.Equal(t, "query", s.In)
	}
}

func TestWalkParameters_FilterByName(t *testing.T) {
	input := walkParametersInput{
		Spec: specInput{Content: walkParametersTestSpec},
		Name: "petId",
	}
	_, output := callWalkParameters(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "petId", output.Summaries[0].Name)
	assert.Equal(t, "path", output.Summaries[0].In)
	assert.True(t, output.Summaries[0].Required)
	assert.Equal(t, "string", output.Summaries[0].Type)
}

func TestWalkParameters_FilterByPath(t *testing.T) {
	input := walkParametersInput{
		Spec: specInput{Content: walkParametersTestSpec},
		Path: "/pets",
	}
	_, output := callWalkParameters(t, input)

	assert.Equal(t, 3, output.Matched)
	for _, s := range output.Summaries {
		assert.Equal(t, "/pets", s.Path)
	}
}

func TestWalkParameters_FilterByMethod(t *testing.T) {
	input := walkParametersInput{
		Spec:   specInput{Content: walkParametersTestSpec},
		Method: "post",
	}
	_, output := callWalkParameters(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "X-Request-Id", output.Summaries[0].Name)
	assert.Equal(t, "POST", output.Summaries[0].Method)
}

func TestWalkParameters_DetailMode(t *testing.T) {
	input := walkParametersInput{
		Spec:   specInput{Content: walkParametersTestSpec},
		Name:   "limit",
		Detail: true,
	}
	_, output := callWalkParameters(t, input)

	assert.Equal(t, 1, output.Matched)
	assert.Nil(t, output.Summaries)
	require.Len(t, output.Parameters, 1)
	assert.Equal(t, "limit", output.Parameters[0].Name)
	assert.NotNil(t, output.Parameters[0].Parameter)
	assert.Equal(t, "query", output.Parameters[0].Parameter.In)
}

func TestWalkParameters_Limit(t *testing.T) {
	input := walkParametersInput{
		Spec:  specInput{Content: walkParametersTestSpec},
		Limit: 2,
	}
	_, output := callWalkParameters(t, input)

	assert.Equal(t, 4, output.Total)
	assert.Equal(t, 4, output.Matched)
	assert.Equal(t, 2, output.Returned)
	assert.Len(t, output.Summaries, 2)
}

func TestWalkParameters_InvalidSpec(t *testing.T) {
	input := walkParametersInput{
		Spec: specInput{Content: "not valid yaml: ["},
	}
	result, _ := callWalkParameters(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestWalkParameters_FilterByExtension(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Extension Test
  version: "1.0.0"
paths:
  /pets:
    get:
      summary: List pets
      parameters:
        - name: limit
          in: query
          x-internal: true
          schema:
            type: integer
        - name: offset
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: OK
`
	input := walkParametersInput{
		Spec:      specInput{Content: spec},
		Extension: "x-internal=true",
	}
	_, output := callWalkParameters(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "limit", output.Summaries[0].Name)
}

func TestWalkParameters_NoMatches(t *testing.T) {
	input := walkParametersInput{
		Spec: specInput{Content: walkParametersTestSpec},
		In:   "cookie",
	}
	_, output := callWalkParameters(t, input)

	assert.Equal(t, 4, output.Total)
	assert.Equal(t, 0, output.Matched)
	assert.Nil(t, output.Summaries)
}

func TestWalkParameters_Offset(t *testing.T) {
	input := walkParametersInput{
		Spec:   specInput{Content: walkParametersTestSpec},
		Offset: 2,
	}
	_, output := callWalkParameters(t, input)

	assert.Equal(t, 4, output.Total)
	assert.Equal(t, 4, output.Matched)
	assert.Equal(t, 2, output.Returned)
	assert.Len(t, output.Summaries, 2)
}

func TestWalkParameters_OffsetAndLimit(t *testing.T) {
	input := walkParametersInput{
		Spec:   specInput{Content: walkParametersTestSpec},
		Offset: 1,
		Limit:  2,
	}
	_, output := callWalkParameters(t, input)

	assert.Equal(t, 4, output.Total)
	assert.Equal(t, 4, output.Matched)
	assert.Equal(t, 2, output.Returned)
	assert.Len(t, output.Summaries, 2)
}

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

func TestWalkParameters_GroupByLocation_RefLabel(t *testing.T) {
	// Spec with a $ref parameter that has no "in" field resolved.
	spec := `openapi: "3.0.0"
info:
  title: Ref Param Test
  version: "1.0.0"
paths:
  /pets:
    get:
      summary: List pets
      parameters:
        - $ref: "#/components/parameters/LimitParam"
      responses:
        "200":
          description: OK
components:
  parameters:
    LimitParam:
      name: limit
      in: query
      schema:
        type: integer
`
	input := walkParametersInput{
		Spec:    specInput{Content: spec},
		GroupBy: "location",
	}
	_, output := callWalkParameters(t, input)

	require.NotEmpty(t, output.Groups)
	// With resolved refs, this should show "query" for the resolved parameter.
	// Without resolve_refs, the walker may report "" for unresolved $ref params.
	// Verify no empty-string key exists in either case.
	for _, g := range output.Groups {
		assert.NotEqual(t, "", g.Key, "expected no empty-string group key")
	}
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

func TestWalkParameters_InFilterAutoResolves(t *testing.T) {
	// Spec where parameters are $ref references — filtering by "in" requires resolution.
	spec := `openapi: "3.0.0"
info:
  title: Ref Param Test
  version: "1.0.0"
paths:
  /pets:
    get:
      summary: List pets
      parameters:
        - $ref: "#/components/parameters/LimitParam"
        - $ref: "#/components/parameters/TokenHeader"
      responses:
        "200":
          description: OK
components:
  parameters:
    LimitParam:
      name: limit
      in: query
      schema:
        type: integer
    TokenHeader:
      name: X-Auth-Token
      in: header
      schema:
        type: string
`
	// Without explicit resolve_refs, in=header should still find the header param.
	input := walkParametersInput{
		Spec: specInput{Content: spec},
		In:   "header",
		Path: "/pets",
	}
	_, output := callWalkParameters(t, input)

	assert.Equal(t, 1, output.Matched, "expected 1 header parameter after auto-resolution")
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "X-Auth-Token", output.Summaries[0].Name)
	assert.Equal(t, "header", output.Summaries[0].In)
}

func TestWalkParameters_NameFilterAutoResolves(t *testing.T) {
	// Spec where parameters are $ref references — filtering by "name" requires resolution.
	spec := `openapi: "3.0.0"
info:
  title: Ref Param Test
  version: "1.0.0"
paths:
  /pets:
    get:
      summary: List pets
      parameters:
        - $ref: "#/components/parameters/LimitParam"
      responses:
        "200":
          description: OK
components:
  parameters:
    LimitParam:
      name: limit
      in: query
      schema:
        type: integer
`
	// Without explicit resolve_refs, name=limit should still find the param.
	input := walkParametersInput{
		Spec: specInput{Content: spec},
		Name: "limit",
		Path: "/pets",
	}
	_, output := callWalkParameters(t, input)

	assert.Equal(t, 1, output.Matched, "expected 1 parameter after auto-resolution by name")
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "limit", output.Summaries[0].Name)
	assert.Equal(t, "query", output.Summaries[0].In)
}
