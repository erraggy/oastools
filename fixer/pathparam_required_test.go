package fixer

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requiredFixPaths returns the reported paths of every
// FixTypePathParameterNotRequired fix, in the order they were recorded, so tests
// can assert both coverage and ordering while ignoring fixes of other types.
func requiredFixPaths(result *FixResult) []string {
	var paths []string
	for _, fix := range result.Fixes {
		if fix.Type == FixTypePathParameterNotRequired {
			paths = append(paths, fix.Path)
		}
	}
	return paths
}

// TestFixPathParamsRequired_OAS3_Components tests that a path parameter in
// components.parameters gets required: true, reported at the definition's path.
func TestFixPathParamsRequired_OAS3_Components(t *testing.T) {
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
        - $ref: '#/components/parameters/UserId'
      responses:
        '200':
          description: Success
components:
  parameters:
    UserId:
      name: userId
      in: path
      schema:
        type: string
`
	parseResult, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := New().FixParsed(*parseResult)
	require.NoError(t, err)

	// The $ref use site is skipped, so the definition is fixed exactly once.
	assert.Equal(t, []string{"components.parameters.UserId"},
		requiredFixPaths(result))

	fix := result.Fixes[0]
	assert.Equal(t, FixTypePathParameterNotRequired, fix.Type)
	assert.Contains(t, fix.Description, "userId")
	assert.Equal(t, false, fix.Before)
	assert.Equal(t, true, fix.After)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3Document")
	require.NotNil(t, doc.Components)
	require.NotNil(t, doc.Components.Parameters["UserId"])
	assert.True(t, doc.Components.Parameters["UserId"].Required)

	// The reference itself is untouched: writing required onto a $ref object
	// would add a sibling the spec does not permit there.
	refParam := doc.Paths["/users/{userId}"].Get.Parameters[0]
	assert.Equal(t, "#/components/parameters/UserId", refParam.Ref)
	assert.False(t, refParam.Required)
}

// TestFixPathParamsRequired_OAS3_PathItemFixedOnce tests that a parameter declared
// on the path item is fixed once, not once per operation in that item.
func TestFixPathParamsRequired_OAS3_PathItemFixedOnce(t *testing.T) {
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
        schema:
          type: string
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
    delete:
      operationId: deleteUser
      responses:
        '204':
          description: No content
`
	parseResult, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := New().FixParsed(*parseResult)
	require.NoError(t, err)

	assert.Equal(t, []string{"paths./users/{userId}.parameters[0]"},
		requiredFixPaths(result))

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3Document")
	assert.True(t, doc.Paths["/users/{userId}"].Parameters[0].Required)
}

// TestFixPathParamsRequired_OAS3_OperationParams tests operation-level parameters,
// including that only the in: path ones are touched and indexes are reported as-is.
func TestFixPathParamsRequired_OAS3_OperationParams(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}/pets/{petId}:
    get:
      operationId: getUserPet
      parameters:
        - name: verbose
          in: query
          schema:
            type: boolean
        - name: userId
          in: path
          schema:
            type: string
        - name: petId
          in: path
          required: true
          schema:
            type: string
        - name: X-Trace
          in: header
          schema:
            type: string
      responses:
        '200':
          description: Success
`
	parseResult, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := New().FixParsed(*parseResult)
	require.NoError(t, err)

	// Only index 1 is defective: 0 and 3 are not path params, 2 is already required.
	assert.Equal(t, []string{"paths./users/{userId}/pets/{petId}.get.parameters[1]"},
		requiredFixPaths(result))

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3Document")
	params := doc.Paths["/users/{userId}/pets/{petId}"].Get.Parameters
	assert.False(t, params[0].Required, "query parameter must not be forced required")
	assert.True(t, params[1].Required)
	assert.True(t, params[2].Required)
	assert.False(t, params[3].Required, "header parameter must not be forced required")
}

// TestFixPathParamsRequired_OAS2 tests all three OAS 2.0 sites: the root-level
// parameters definitions, path item parameters, and operation parameters.
func TestFixPathParamsRequired_OAS2(t *testing.T) {
	spec := `
swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
parameters:
  UserId:
    name: userId
    in: path
    type: string
  Verbose:
    name: verbose
    in: query
    type: boolean
paths:
  /users/{userId}:
    parameters:
      - name: userId
        in: path
        type: string
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - name: petId
          in: path
          type: string
      responses:
        '200':
          description: Success
`
	parseResult, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := New().FixParsed(*parseResult)
	require.NoError(t, err)

	// Definitions first, then paths in sorted order — /pets before /users.
	assert.Equal(t, []string{
		"parameters.UserId",
		"paths./pets/{petId}.get.parameters[0]",
		"paths./users/{userId}.parameters[0]",
	}, requiredFixPaths(result))

	doc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok, "expected OAS2Document")
	assert.True(t, doc.Parameters["UserId"].Required)
	assert.False(t, doc.Parameters["Verbose"].Required, "query definition must not be forced required")
	assert.True(t, doc.Paths["/users/{userId}"].Parameters[0].Required)
	assert.True(t, doc.Paths["/pets/{petId}"].Get.Parameters[0].Required)
}

// TestFixPathParamsRequired_DryRun tests that dry-run reports the fix without
// modifying the document.
func TestFixPathParamsRequired_DryRun(t *testing.T) {
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
          schema:
            type: string
      responses:
        '200':
          description: Success
`
	parseResult, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)

	f := New()
	f.DryRun = true
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	assert.Equal(t, []string{"paths./users/{userId}.get.parameters[0]"},
		requiredFixPaths(result))

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3Document")
	assert.False(t, doc.Paths["/users/{userId}"].Get.Parameters[0].Required,
		"dry run must not modify the document")
}

// TestFixPathParamsRequired_Disabled tests that the fix is skipped when the caller
// enables a different fix type.
func TestFixPathParamsRequired_Disabled(t *testing.T) {
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
          schema:
            type: string
      responses:
        '200':
          description: Success
`
	parseResult, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeMissingPathParameter),
	)
	require.NoError(t, err)

	assert.Empty(t, requiredFixPaths(result))
}

// TestFixPathParamsRequired_NoDefect tests that a already-valid document records
// no fixes.
func TestFixPathParamsRequired_NoDefect(t *testing.T) {
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
	parseResult, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := New().FixParsed(*parseResult)
	require.NoError(t, err)

	assert.False(t, result.HasFixes())
}

// TestFixPathParamsRequired_ResolvesValidatorErrors is the parity check the fix
// exists for: every "Path parameters must have required: true" the validator
// reports must be gone after fixing, at all three sites and in both versions.
func TestFixPathParamsRequired_ResolvesValidatorErrors(t *testing.T) {
	const requiredMsg = "Path parameters must have required: true"

	tests := []struct {
		name string
		spec string
	}{
		{
			name: "OAS 3.x",
			spec: `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    parameters:
      - name: userId
        in: path
        schema:
          type: string
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - $ref: '#/components/parameters/PetId'
      responses:
        '200':
          description: Success
components:
  parameters:
    PetId:
      name: petId
      in: path
      schema:
        type: string
`,
		},
		{
			name: "OAS 2.0",
			spec: `
swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
parameters:
  PetId:
    name: petId
    in: path
    type: string
paths:
  /users/{userId}:
    parameters:
      - name: userId
        in: path
        type: string
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - $ref: '#/parameters/PetId'
      responses:
        '200':
          description: Success
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseResult, err := parser.New().ParseBytes([]byte(tt.spec))
			require.NoError(t, err)

			v := validator.New()
			before, err := v.ValidateParsed(*parseResult)
			require.NoError(t, err)
			require.NotZero(t, countErrorsContaining(before, requiredMsg),
				"spec must exhibit the defect before fixing")

			result, err := New().FixParsed(*parseResult)
			require.NoError(t, err)
			require.NotEmpty(t, requiredFixPaths(result))

			after, err := v.ValidateParsed(*result.ToParseResult())
			require.NoError(t, err)
			assert.Zero(t, countErrorsContaining(after, requiredMsg),
				"fixer must resolve every required: true error the validator reports")
		})
	}
}

// countErrorsContaining counts validation errors whose message contains substr.
func countErrorsContaining(result *validator.ValidationResult, substr string) int {
	count := 0
	for _, e := range result.Errors {
		if strings.Contains(e.Message, substr) {
			count++
		}
	}
	return count
}

// TestFixPathParamsRequired_NilTolerance tests that nil path items and nil
// parameter entries are skipped rather than panicking. Parsers can leave either
// behind for a document with an empty YAML mapping in the list.
func TestFixPathParamsRequired_NilTolerance(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Paths: parser.Paths{
			"/nil": nil,
			"/users/{userId}": {
				Parameters: []*parser.Parameter{nil},
				Get: &parser.Operation{
					Parameters: []*parser.Parameter{
						nil,
						{Name: "userId", In: parser.ParamInPath},
					},
				},
			},
		},
		Components: &parser.Components{
			Parameters: map[string]*parser.Parameter{"Nil": nil},
		},
	}

	f := New()
	result := &FixResult{}
	f.fixPathParamsRequiredOAS3(doc, result)

	assert.Equal(t, []string{"paths./users/{userId}.get.parameters[1]"},
		requiredFixPaths(result))
	assert.True(t, doc.Paths["/users/{userId}"].Get.Parameters[1].Required)
}

// TestBuildPathParamRequiredDescription tests that an unnamed parameter — invalid
// for a reason the validator reports separately — does not produce a description
// with an empty quoted name.
func TestBuildPathParamRequiredDescription(t *testing.T) {
	assert.Equal(t, "Set required: true on path parameter 'userId'",
		buildPathParamRequiredDescription("userId"))
	assert.Equal(t, "Set required: true on path parameter",
		buildPathParamRequiredDescription(""))
}
