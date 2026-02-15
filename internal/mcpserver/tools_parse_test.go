package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSpecYAML = `openapi: "3.0.0"
info:
  title: Pet Store
  description: A sample pet store API
  version: "1.0.0"
servers:
  - url: https://api.example.com
    description: Production
tags:
  - name: pets
  - name: store
paths:
  /pets:
    get:
      summary: List pets
      operationId: listPets
      tags:
        - pets
      responses:
        "200":
          description: OK
  /pets/{id}:
    get:
      summary: Get a pet
      operationId: getPet
      tags:
        - pets
      responses:
        "200":
          description: OK
`

func TestParseTool_Summary(t *testing.T) {
	input := parseInput{
		Spec: specInput{Content: testSpecYAML},
	}
	_, output, err := handleParse(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.Equal(t, "3.0.0", output.Version)
	assert.Equal(t, "Pet Store", output.Title)
	assert.Equal(t, "A sample pet store API", output.Description)
	assert.Equal(t, "yaml", output.Format)
	assert.Equal(t, 2, output.PathCount)
	assert.Equal(t, 2, output.OperationCount)
	assert.Equal(t, []string{"pets", "store"}, output.Tags)
	assert.Empty(t, output.FullDocument)

	require.Len(t, output.Servers, 1)
	assert.Equal(t, "https://api.example.com", output.Servers[0].URL)
	assert.Equal(t, "Production", output.Servers[0].Description)
}

func TestParseTool_Full(t *testing.T) {
	input := parseInput{
		Spec: specInput{Content: testSpecYAML},
		Full: true,
	}
	_, output, err := handleParse(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.Equal(t, "3.0.0", output.Version)
	assert.Equal(t, "Pet Store", output.Title)
	assert.NotEmpty(t, output.FullDocument)
	assert.Contains(t, output.FullDocument, "Pet Store")
	assert.Contains(t, output.FullDocument, "/pets")
}

func TestParseTool_InvalidSpec(t *testing.T) {
	input := parseInput{
		Spec: specInput{Content: "not valid yaml: ["},
	}
	result, output, err := handleParse(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.Version)
}

func TestParseTool_ResolveRefs(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Ref Test
  version: "1.0.0"
paths:
  /items:
    get:
      summary: List items
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Item"
components:
  schemas:
    Item:
      type: object
      properties:
        name:
          type: string
`
	input := parseInput{
		Spec:        specInput{Content: spec},
		ResolveRefs: true,
		Full:        true,
	}
	_, output, err := handleParse(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.Equal(t, "3.0.0", output.Version)
	assert.Equal(t, "Ref Test", output.Title)
	assert.Equal(t, 1, output.SchemaCount)
	assert.NotEmpty(t, output.FullDocument)
}

func TestParseTool_TruncatesLongDescription(t *testing.T) {
	// Create a spec with a very long description.
	longDesc := strings.Repeat("A", 500)
	spec := `openapi: "3.0.0"
info:
  title: Long Desc Test
  description: "` + longDesc + `"
  version: "1.0.0"
servers:
  - url: https://api.example.com
    description: "` + longDesc + `"
paths: {}
`
	input := parseInput{
		Spec: specInput{Content: spec},
	}
	_, output, err := handleParse(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	// Summary mode: description should be truncated.
	assert.LessOrEqual(t, len(output.Description), 203) // 200 + "..."
	assert.True(t, strings.HasSuffix(output.Description, "..."))
	// Server description should also be truncated.
	require.Len(t, output.Servers, 1)
	assert.LessOrEqual(t, len(output.Servers[0].Description), 203)
	assert.True(t, strings.HasSuffix(output.Servers[0].Description, "..."))
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"truncated", "hello world", 5, "hello..."},
		{"empty", "", 5, ""},
		{"multi-byte UTF-8", "café résumé", 5, "café ..."},
		{"zero maxLen", "hello", 0, "..."},
		{"negative maxLen", "hello", -1, "..."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, truncateText(tt.input, tt.maxLen))
		})
	}
}
