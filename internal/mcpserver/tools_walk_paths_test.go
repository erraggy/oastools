package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const walkPathsTestSpec = `openapi: "3.0.0"
info:
  title: Walk Paths Test
  version: "1.0.0"
paths:
  /pets:
    summary: Pet operations
    get:
      summary: List pets
      responses:
        "200":
          description: OK
    post:
      summary: Create a pet
      responses:
        "201":
          description: Created
  /pets/{petId}:
    get:
      summary: Get a pet
      responses:
        "200":
          description: OK
    put:
      summary: Update a pet
      responses:
        "200":
          description: OK
    delete:
      summary: Delete a pet
      responses:
        "204":
          description: Deleted
  /stores:
    get:
      summary: List stores
      responses:
        "200":
          description: OK
`

func callWalkPaths(t *testing.T, input walkPathsInput) (*mcp.CallToolResult, walkPathsOutput) {
	t.Helper()
	result, out, err := handleWalkPaths(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	if out == nil {
		return result, walkPathsOutput{}
	}
	wo, ok := out.(walkPathsOutput)
	require.True(t, ok, "expected walkPathsOutput, got %T", out)
	return result, wo
}

func TestWalkPaths_AllPaths(t *testing.T) {
	input := walkPathsInput{
		Spec: specInput{Content: walkPathsTestSpec},
	}
	_, output := callWalkPaths(t, input)

	assert.Equal(t, 3, output.Total)
	assert.Equal(t, 3, output.Matched)
	assert.Equal(t, 3, output.Returned)
	require.Len(t, output.Summaries, 3)
}

func TestWalkPaths_FilterByPath(t *testing.T) {
	input := walkPathsInput{
		Spec: specInput{Content: walkPathsTestSpec},
		Path: "/pets/*",
	}
	_, output := callWalkPaths(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "/pets/{petId}", output.Summaries[0].Path)
	assert.Equal(t, 3, output.Summaries[0].MethodCount)
}

func TestWalkPaths_FilterByExactPath(t *testing.T) {
	input := walkPathsInput{
		Spec: specInput{Content: walkPathsTestSpec},
		Path: "/pets",
	}
	_, output := callWalkPaths(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "/pets", output.Summaries[0].Path)
	assert.Equal(t, 2, output.Summaries[0].MethodCount)
	assert.Equal(t, "Pet operations", output.Summaries[0].Summary)
}

func TestWalkPaths_MethodCount(t *testing.T) {
	input := walkPathsInput{
		Spec: specInput{Content: walkPathsTestSpec},
	}
	_, output := callWalkPaths(t, input)

	// Build a map of path to method count for easy lookup.
	counts := make(map[string]int)
	for _, s := range output.Summaries {
		counts[s.Path] = s.MethodCount
	}

	assert.Equal(t, 2, counts["/pets"])
	assert.Equal(t, 3, counts["/pets/{petId}"])
	assert.Equal(t, 1, counts["/stores"])
}

func TestWalkPaths_DetailMode(t *testing.T) {
	input := walkPathsInput{
		Spec:   specInput{Content: walkPathsTestSpec},
		Path:   "/stores",
		Detail: true,
	}
	_, output := callWalkPaths(t, input)

	assert.Equal(t, 1, output.Matched)
	assert.Nil(t, output.Summaries)
	require.Len(t, output.Paths, 1)
	assert.Equal(t, "/stores", output.Paths[0].Path)
	assert.NotNil(t, output.Paths[0].PathItem)
	assert.NotNil(t, output.Paths[0].PathItem.Get)
	assert.Nil(t, output.Paths[0].PathItem.Post)
}

func TestWalkPaths_Limit(t *testing.T) {
	input := walkPathsInput{
		Spec:  specInput{Content: walkPathsTestSpec},
		Limit: 1,
	}
	_, output := callWalkPaths(t, input)

	assert.Equal(t, 3, output.Total)
	assert.Equal(t, 3, output.Matched)
	assert.Equal(t, 1, output.Returned)
	assert.Len(t, output.Summaries, 1)
}

func TestWalkPaths_InvalidSpec(t *testing.T) {
	input := walkPathsInput{
		Spec: specInput{Content: "not valid yaml: ["},
	}
	result, _ := callWalkPaths(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestWalkPaths_NoMatches(t *testing.T) {
	input := walkPathsInput{
		Spec: specInput{Content: walkPathsTestSpec},
		Path: "/users",
	}
	_, output := callWalkPaths(t, input)

	assert.Equal(t, 0, output.Matched)
	assert.Nil(t, output.Summaries)
}

func TestWalkPaths_FilterByExtension(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Extension Test
  version: "1.0.0"
paths:
  /internal:
    x-internal: true
    get:
      summary: Internal endpoint
      responses:
        "200":
          description: OK
  /public:
    get:
      summary: Public endpoint
      responses:
        "200":
          description: OK
`
	input := walkPathsInput{
		Spec:      specInput{Content: spec},
		Extension: "x-internal=true",
	}
	_, output := callWalkPaths(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "/internal", output.Summaries[0].Path)
}
