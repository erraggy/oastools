package mcpserver

import (
	"context"
	"fmt"
	"strings"
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
	assert.True(t, matchWalkPath("/users/{id}", "/**/users/*"))
}

func TestMatchWalkPath_DoubleStarNoMatch(t *testing.T) {
	assert.False(t, matchWalkPath("/pets/{petId}", "/users/**"))
	assert.False(t, matchWalkPath("/stores", "/users/**/stores"))
}

func TestMatchWalkPath_SingleStarUnchanged(t *testing.T) {
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
	assert.True(t, matchWalkPath("/sites/{id}/termStore/groups/{gid}/sets/{sid}/children/{tid}", "/sites/*/termStore/**"))
	assert.False(t, matchWalkPath("/sites/{id}/other/groups", "/sites/*/termStore/**"))
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
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "users.listPosts", output.Summaries[0].OperationID)
}

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

func TestWalkOperations_DetailDefaultLimit(t *testing.T) {
	// Generate a spec with >25 operations to verify detail limit kicks in.
	paths := make([]string, 0, 30)
	for i := range 30 {
		paths = append(paths, fmt.Sprintf(`  /resource%d:
    get:
      operationId: getResource%d
      summary: Get resource %d
      responses:
        "200":
          description: OK`, i, i, i))
	}
	spec := fmt.Sprintf(`openapi: "3.0.0"
info:
  title: Detail Limit Test
  version: "1.0.0"
paths:
%s
`, strings.Join(paths, "\n"))

	input := walkOperationsInput{
		Spec:   specInput{Content: spec},
		Detail: true,
	}
	_, output := callWalkOperations(t, input)

	assert.Equal(t, 30, output.Total)
	assert.Equal(t, 30, output.Matched)
	assert.Equal(t, 25, output.Returned, "detail mode should default to 25 items")
	assert.Nil(t, output.Summaries, "detail mode should not populate summaries")
	assert.Len(t, output.Operations, 25)
}
