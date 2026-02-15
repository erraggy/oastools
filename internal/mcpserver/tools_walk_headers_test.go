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

	// 6 response headers + 1 component header = 7 total.
	assert.Equal(t, 7, output.Total)
	assert.Equal(t, 7, output.Matched)
	require.Len(t, output.Summaries, 7)
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
	require.Len(t, output.Summaries, 1)
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
	require.Len(t, output.Summaries, 1)
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
	assert.Equal(t, 2, groupMap["X-Rate-Limit"]) // appears in 2 responses
	assert.Equal(t, 1, groupMap["Location"])     // appears in 1 response
	assert.Equal(t, 1, groupMap["TraceId"])      // component header

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
	assert.Equal(t, 3, groupMap["200"]) // GET /pets 200 (2) + GET /stores 200 (1)
	assert.Equal(t, 2, groupMap["201"]) // POST /pets 201 (2)
	assert.Equal(t, 1, groupMap["404"]) // GET /pets 404 (1)
	// Component header (TraceId) has no status code -- excluded from status_code grouping.
}

func TestWalkHeaders_Pagination(t *testing.T) {
	input := walkHeadersInput{
		Spec:  specInput{Content: walkHeadersTestSpec},
		Limit: 2,
	}
	_, output := callWalkHeaders(t, input)

	assert.Equal(t, 7, output.Total)
	assert.Equal(t, 7, output.Matched)
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

func TestWalkHeaders_FilterByExtension(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Extension Test
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
              x-internal: true
              schema:
                type: integer
            X-Request-Id:
              description: Request identifier
              schema:
                type: string
`
	input := walkHeadersInput{
		Spec:      specInput{Content: spec},
		Extension: "x-internal=true",
	}
	_, output := callWalkHeaders(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "X-Rate-Limit", output.Summaries[0].Name)
}

func TestWalkHeaders_FilterByExtensionInvalid(t *testing.T) {
	input := walkHeadersInput{
		Spec:      specInput{Content: walkHeadersTestSpec},
		Extension: "not-an-extension=true",
	}
	result, _ := callWalkHeaders(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}
