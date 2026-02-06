package fixer

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFixMissingPathParametersOAS3 tests fixing missing path parameters in OAS 3.x
func TestFixMissingPathParametersOAS3(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	f := New()
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	assert.True(t, result.HasFixes())
	assert.Equal(t, 1, result.FixCount)
	assert.Equal(t, FixTypeMissingPathParameter, result.Fixes[0].Type)
	assert.Contains(t, result.Fixes[0].Description, "userId")

	// Verify the parameter was added
	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3Document")
	pathItem := doc.Paths["/users/{userId}"]
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Get)
	require.Len(t, pathItem.Get.Parameters, 1)

	param := pathItem.Get.Parameters[0]
	assert.Equal(t, "userId", param.Name)
	assert.Equal(t, "path", param.In)
	assert.True(t, param.Required)
	assert.NotNil(t, param.Schema)
	assert.Equal(t, "string", param.Schema.Type)
}

// TestFixMissingPathParametersOAS3_WithInfer tests type inference
func TestFixMissingPathParametersOAS3_WithInfer(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}/documents/{documentUuid}:
    get:
      operationId: getDocument
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	f := New()
	f.InferTypes = true
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	assert.Equal(t, 2, result.FixCount)

	// Find the parameters
	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3Document")
	params := doc.Paths["/users/{userId}/documents/{documentUuid}"].Get.Parameters
	require.Len(t, params, 2)

	// Check types - they may be in any order
	paramsByName := make(map[string]*parser.Parameter)
	for _, p := range params {
		paramsByName[p.Name] = p
	}

	assert.Equal(t, "integer", paramsByName["userId"].Schema.Type)
	assert.Equal(t, "string", paramsByName["documentUuid"].Schema.Type)
	assert.Equal(t, "uuid", paramsByName["documentUuid"].Schema.Format)
}

// TestFixMissingPathParametersOAS2 tests fixing missing path parameters in OAS 2.0
func TestFixMissingPathParametersOAS2(t *testing.T) {
	spec := `
swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	f := New()
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	assert.True(t, result.HasFixes())
	assert.Equal(t, 1, result.FixCount)

	// Verify the parameter was added with OAS 2.0 style (type directly on param)
	doc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok, "expected OAS2Document")
	pathItem := doc.Paths["/users/{userId}"]
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Get)
	require.Len(t, pathItem.Get.Parameters, 1)

	param := pathItem.Get.Parameters[0]
	assert.Equal(t, "userId", param.Name)
	assert.Equal(t, "path", param.In)
	assert.True(t, param.Required)
	assert.Equal(t, "string", param.Type) // OAS 2.0 uses Type directly
}

// TestFixNoChangesNeeded tests that no fixes are applied when spec is valid
func TestFixNoChangesNeeded(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      operationId: getUser
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	f := New()
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	assert.False(t, result.HasFixes())
	assert.Equal(t, 0, result.FixCount)
}

// TestFixPathItemLevelParameters tests that PathItem-level params are considered
func TestFixPathItemLevelParameters(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    parameters:
      - name: userId
        in: path
        required: true
        schema:
          type: string
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
    put:
      operationId: updateUser
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	f := New()
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// No fixes needed - userId is declared at PathItem level
	assert.False(t, result.HasFixes())
}
