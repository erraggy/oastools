package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const walkOperationsTestSpec = `openapi: "3.0.0"
info:
  title: Walk Ops Test
  version: "1.0.0"
paths:
  /pets:
    get:
      summary: List all pets
      operationId: listPets
      tags:
        - pets
      responses:
        "200":
          description: OK
    post:
      summary: Create a pet
      operationId: createPet
      tags:
        - pets
      responses:
        "201":
          description: Created
  /pets/{petId}:
    get:
      summary: Get a pet by ID
      operationId: getPet
      tags:
        - pets
      deprecated: true
      responses:
        "200":
          description: OK
    delete:
      summary: Delete a pet
      operationId: deletePet
      tags:
        - admin
      responses:
        "204":
          description: Deleted
  /stores:
    get:
      summary: List stores
      operationId: listStores
      tags:
        - stores
      responses:
        "200":
          description: OK
`

func callWalkOperations(t *testing.T, input walkOperationsInput) (*mcp.CallToolResult, walkOperationsOutput) {
	t.Helper()
	result, out, err := handleWalkOperations(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	if out == nil {
		return result, walkOperationsOutput{}
	}
	wo, ok := out.(walkOperationsOutput)
	require.True(t, ok, "expected walkOperationsOutput, got %T", out)
	return result, wo
}

func TestWalkOperations_FilterByMethod(t *testing.T) {
	input := walkOperationsInput{
		Spec:   specInput{Content: walkOperationsTestSpec},
		Method: "get",
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 5, output.Total)
	assert.Equal(t, 3, output.Matched)
	assert.Equal(t, 3, output.Returned)
	require.Len(t, output.Summaries, 3)

	// All results should be GET.
	for _, s := range output.Summaries {
		assert.Equal(t, "GET", s.Method)
	}

	// Verify specific operations are present.
	ids := make([]string, 0, len(output.Summaries))
	for _, s := range output.Summaries {
		ids = append(ids, s.OperationID)
	}
	assert.Contains(t, ids, "listPets")
	assert.Contains(t, ids, "getPet")
	assert.Contains(t, ids, "listStores")
}

func TestWalkOperations_Limit(t *testing.T) {
	input := walkOperationsInput{
		Spec:  specInput{Content: walkOperationsTestSpec},
		Limit: 2,
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 5, output.Total)
	assert.Equal(t, 5, output.Matched)
	assert.Equal(t, 2, output.Returned)
	assert.Len(t, output.Summaries, 2)
}

func TestWalkOperations_FilterByTag(t *testing.T) {
	input := walkOperationsInput{
		Spec: specInput{Content: walkOperationsTestSpec},
		Tag:  "admin",
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "deletePet", output.Summaries[0].OperationID)
	assert.Equal(t, "DELETE", output.Summaries[0].Method)
}

func TestWalkOperations_FilterByPath(t *testing.T) {
	input := walkOperationsInput{
		Spec: specInput{Content: walkOperationsTestSpec},
		Path: "/pets/*",
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 2, output.Matched)
	for _, s := range output.Summaries {
		assert.Equal(t, "/pets/{petId}", s.Path)
	}
}

func TestWalkOperations_FilterByDeprecated(t *testing.T) {
	input := walkOperationsInput{
		Spec:       specInput{Content: walkOperationsTestSpec},
		Deprecated: true,
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "getPet", output.Summaries[0].OperationID)
	assert.True(t, output.Summaries[0].Deprecated)
}

func TestWalkOperations_FilterByOperationID(t *testing.T) {
	input := walkOperationsInput{
		Spec:        specInput{Content: walkOperationsTestSpec},
		OperationID: "createPet",
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "createPet", output.Summaries[0].OperationID)
	assert.Equal(t, "POST", output.Summaries[0].Method)
	assert.Equal(t, "Create a pet", output.Summaries[0].Summary)
}

func TestWalkOperations_DetailMode(t *testing.T) {
	input := walkOperationsInput{
		Spec:        specInput{Content: walkOperationsTestSpec},
		OperationID: "listPets",
		Detail:      true,
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 1, output.Matched)
	assert.Nil(t, output.Summaries)
	require.Len(t, output.Operations, 1)
	assert.Equal(t, "GET", output.Operations[0].Method)
	assert.Equal(t, "/pets", output.Operations[0].Path)
	assert.Equal(t, "listPets", output.Operations[0].Operation.OperationID)
	assert.Equal(t, "List all pets", output.Operations[0].Operation.Summary)
}

func TestWalkOperations_NoMatches(t *testing.T) {
	input := walkOperationsInput{
		Spec:   specInput{Content: walkOperationsTestSpec},
		Method: "patch",
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 5, output.Total)
	assert.Equal(t, 0, output.Matched)
	assert.Equal(t, 0, output.Returned)
	assert.Nil(t, output.Summaries)
}

func TestWalkOperations_FilterByExtension(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Extension Test
  version: "1.0.0"
paths:
  /internal:
    get:
      summary: Internal endpoint
      operationId: getInternal
      x-internal: true
      responses:
        "200":
          description: OK
  /public:
    get:
      summary: Public endpoint
      operationId: getPublic
      responses:
        "200":
          description: OK
`
	input := walkOperationsInput{
		Spec:      specInput{Content: spec},
		Extension: "x-internal=true",
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "getInternal", output.Summaries[0].OperationID)
}

func TestWalkOperations_FilterByExtensionExistence(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Extension Test
  version: "1.0.0"
paths:
  /internal:
    get:
      summary: Internal endpoint
      operationId: getInternal
      x-internal: true
      responses:
        "200":
          description: OK
  /public:
    get:
      summary: Public endpoint
      operationId: getPublic
      responses:
        "200":
          description: OK
`
	input := walkOperationsInput{
		Spec:      specInput{Content: spec},
		Extension: "x-internal",
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "getInternal", output.Summaries[0].OperationID)
}

func TestWalkOperations_FilterByExtensionInvalid(t *testing.T) {
	input := walkOperationsInput{
		Spec:      specInput{Content: walkOperationsTestSpec},
		Extension: "not-an-extension=true",
	}
	result, _ := callWalkOperations(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestWalkOperations_FilterByExactPath(t *testing.T) {
	input := walkOperationsInput{
		Spec: specInput{Content: walkOperationsTestSpec},
		Path: "/pets",
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 2, output.Matched)
	for _, s := range output.Summaries {
		assert.Equal(t, "/pets", s.Path)
	}
}

func TestWalkOperations_InvalidSpec(t *testing.T) {
	input := walkOperationsInput{
		Spec: specInput{Content: "not valid yaml: ["},
	}
	result, _ := callWalkOperations(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestWalkOperations_Offset(t *testing.T) {
	input := walkOperationsInput{
		Spec:   specInput{Content: walkOperationsTestSpec},
		Offset: 2,
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 5, output.Total)
	assert.Equal(t, 5, output.Matched)
	assert.Equal(t, 3, output.Returned)
	assert.Len(t, output.Summaries, 3)
}

func TestWalkOperations_OffsetAndLimit(t *testing.T) {
	input := walkOperationsInput{
		Spec:   specInput{Content: walkOperationsTestSpec},
		Offset: 1,
		Limit:  2,
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 5, output.Total)
	assert.Equal(t, 5, output.Matched)
	assert.Equal(t, 2, output.Returned)
	assert.Len(t, output.Summaries, 2)
}
