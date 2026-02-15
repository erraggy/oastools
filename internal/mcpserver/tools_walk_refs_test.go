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

func TestMatchRefGlob(t *testing.T) {
	// Glob * crosses / boundaries in ref targets.
	assert.True(t, matchRefGlob("#/components/schemas/Pet", "*schemas/Pet"))
	assert.True(t, matchRefGlob("#/components/schemas/Pet", "*Pet"))
	assert.True(t, matchRefGlob("#/components/responses/NotFound", "*responses/*"))

	// Case-insensitive.
	assert.True(t, matchRefGlob("#/components/schemas/Pet", "*SCHEMAS/PET"))

	// ? matches single character.
	assert.True(t, matchRefGlob("#/components/schemas/Pet", "*schemas/P?t"))
	assert.False(t, matchRefGlob("#/components/schemas/Pet", "*schemas/P?"))

	// Exact match without glob chars (case-insensitive).
	assert.True(t, matchRefGlob("#/components/schemas/Pet", "#/components/schemas/Pet"))
	assert.True(t, matchRefGlob("#/components/schemas/Pet", "#/components/schemas/pet"))
	assert.False(t, matchRefGlob("#/components/schemas/Pet", "#/components/schemas/Error"))

	// No match.
	assert.False(t, matchRefGlob("#/components/schemas/Pet", "*responses/*"))
}

func TestWalkRefs_InvalidSpec(t *testing.T) {
	input := walkRefsInput{
		Spec: specInput{Content: "not valid yaml: ["},
	}
	result, _ := callWalkRefs(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}
