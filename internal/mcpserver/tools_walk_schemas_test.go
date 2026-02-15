package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const walkSchemasTestSpec = `openapi: "3.0.0"
info:
  title: Walk Schemas Test
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
              type: object
              properties:
                name:
                  type: string
                age:
                  type: integer
              required:
                - name
      responses:
        "201":
          description: Created
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
        tag:
          type: string
      required:
        - id
        - name
    Error:
      type: object
      properties:
        code:
          type: integer
        message:
          type: string
      required:
        - code
        - message
    Tag:
      type: string
`

func callWalkSchemas(t *testing.T, input walkSchemasInput) (*mcp.CallToolResult, walkSchemasOutput) {
	t.Helper()
	result, out, err := handleWalkSchemas(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	if out == nil {
		return result, walkSchemasOutput{}
	}
	wo, ok := out.(walkSchemasOutput)
	require.True(t, ok, "expected walkSchemasOutput, got %T", out)
	return result, wo
}

func TestWalkSchemas_AllSchemas(t *testing.T) {
	input := walkSchemasInput{
		Spec: specInput{Content: walkSchemasTestSpec},
	}
	_, output := callWalkSchemas(t, input)

	// Should include both component and inline schemas.
	assert.Greater(t, output.Total, 3)
	assert.Equal(t, output.Total, output.Matched)
	assert.NotEmpty(t, output.Summaries)
}

func TestWalkSchemas_ComponentFilter(t *testing.T) {
	input := walkSchemasInput{
		Spec:      specInput{Content: walkSchemasTestSpec},
		Component: true,
	}
	_, output := callWalkSchemas(t, input)

	// Component schemas include top-level (Pet, Error, Tag) and their nested
	// property schemas (id, name, tag, code, message) since the walker marks
	// all schemas under components as IsComponent=true.
	assert.Equal(t, 8, output.Matched)
	require.Len(t, output.Summaries, 8)

	for _, s := range output.Summaries {
		assert.Equal(t, "component", s.Location)
	}

	names := make([]string, 0, len(output.Summaries))
	for _, s := range output.Summaries {
		names = append(names, s.Name)
	}
	assert.Contains(t, names, "Pet")
	assert.Contains(t, names, "Error")
	assert.Contains(t, names, "Tag")
}

func TestWalkSchemas_InlineFilter(t *testing.T) {
	input := walkSchemasInput{
		Spec:   specInput{Content: walkSchemasTestSpec},
		Inline: true,
	}
	_, output := callWalkSchemas(t, input)

	assert.Greater(t, output.Matched, 0)
	for _, s := range output.Summaries {
		assert.Equal(t, "inline", s.Location)
	}
}

func TestWalkSchemas_FilterByType(t *testing.T) {
	input := walkSchemasInput{
		Spec:      specInput{Content: walkSchemasTestSpec},
		Component: true,
		Type:      "object",
	}
	_, output := callWalkSchemas(t, input)

	// Pet and Error are object types; Tag is string.
	assert.Equal(t, 2, output.Matched)
	for _, s := range output.Summaries {
		assert.Equal(t, "object", s.Type)
	}
}

func TestWalkSchemas_FilterByName(t *testing.T) {
	input := walkSchemasInput{
		Spec: specInput{Content: walkSchemasTestSpec},
		Name: "Pet",
	}
	_, output := callWalkSchemas(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "Pet", output.Summaries[0].Name)
	assert.Equal(t, "object", output.Summaries[0].Type)
	assert.Equal(t, 3, output.Summaries[0].PropertyCount)
	assert.Equal(t, []string{"id", "name"}, output.Summaries[0].Required)
}

func TestWalkSchemas_DetailMode(t *testing.T) {
	input := walkSchemasInput{
		Spec:   specInput{Content: walkSchemasTestSpec},
		Name:   "Error",
		Detail: true,
	}
	_, output := callWalkSchemas(t, input)

	assert.Equal(t, 1, output.Matched)
	assert.Nil(t, output.Summaries)
	require.Len(t, output.Schemas, 1)
	assert.Equal(t, "Error", output.Schemas[0].Name)
	assert.True(t, output.Schemas[0].IsComponent)
	assert.NotNil(t, output.Schemas[0].Schema)
	assert.Equal(t, 2, len(output.Schemas[0].Schema.Properties))
}

func TestWalkSchemas_Limit(t *testing.T) {
	input := walkSchemasInput{
		Spec:      specInput{Content: walkSchemasTestSpec},
		Component: true,
		Limit:     2,
	}
	_, output := callWalkSchemas(t, input)

	assert.Equal(t, 8, output.Matched)
	assert.Equal(t, 2, output.Returned)
	assert.Len(t, output.Summaries, 2)
}

func TestWalkSchemas_ComponentAndInlineMutuallyExclusive(t *testing.T) {
	input := walkSchemasInput{
		Spec:      specInput{Content: walkSchemasTestSpec},
		Component: true,
		Inline:    true,
	}
	result, _ := callWalkSchemas(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestWalkSchemas_FilterByExtension(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Extension Test
  version: "1.0.0"
paths: {}
components:
  schemas:
    InternalModel:
      type: object
      x-internal: true
      properties:
        id:
          type: integer
    PublicModel:
      type: object
      properties:
        name:
          type: string
`
	input := walkSchemasInput{
		Spec:      specInput{Content: spec},
		Name:      "InternalModel",
		Extension: "x-internal=true",
	}
	_, output := callWalkSchemas(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "InternalModel", output.Summaries[0].Name)
}

func TestWalkSchemas_SchemaTypeMatchesArrayAny(t *testing.T) {
	// OAS 3.1+ can have type as []any (e.g., ["string", "null"]).
	assert.True(t, schemaTypeMatches([]any{"string", "null"}, "string"))
	assert.True(t, schemaTypeMatches([]any{"string", "null"}, "null"))
	assert.False(t, schemaTypeMatches([]any{"string", "null"}, "integer"))
	assert.False(t, schemaTypeMatches(nil, "string"))
	assert.True(t, schemaTypeMatches("object", "object"))
	assert.False(t, schemaTypeMatches("object", "string"))
	assert.True(t, schemaTypeMatches([]string{"string", "null"}, "null"))
	assert.False(t, schemaTypeMatches([]string{"string", "null"}, "integer"))
}

func TestWalkSchemas_InvalidSpec(t *testing.T) {
	input := walkSchemasInput{
		Spec: specInput{Content: "not valid yaml: ["},
	}
	result, _ := callWalkSchemas(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestWalkSchemas_Offset(t *testing.T) {
	input := walkSchemasInput{
		Spec:      specInput{Content: walkSchemasTestSpec},
		Component: true,
		Offset:    3,
	}
	_, output := callWalkSchemas(t, input)

	assert.Equal(t, 8, output.Matched)
	assert.Equal(t, 5, output.Returned)
	assert.Len(t, output.Summaries, 5)
}

func TestWalkSchemas_OffsetAndLimit(t *testing.T) {
	input := walkSchemasInput{
		Spec:      specInput{Content: walkSchemasTestSpec},
		Component: true,
		Offset:    3,
		Limit:     2,
	}
	_, output := callWalkSchemas(t, input)

	assert.Equal(t, 8, output.Matched)
	assert.Equal(t, 2, output.Returned)
	assert.Len(t, output.Summaries, 2)
}
