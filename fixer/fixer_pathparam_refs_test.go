package fixer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// fixPathParams runs the missing-path-parameter fix over spec and returns the
// resulting fixes.
func fixPathParams(t *testing.T, spec string) *FixResult {
	t.Helper()

	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := New().FixParsed(*parseResult)
	require.NoError(t, err)
	return result
}

// TestFixMissingPathParameters_ReferencedParametersNotDuplicated covers the
// fixer half of issue #374. A $ref parameter carries no Name or In of its own,
// so a path parameter hoisted into reusable definitions looked undeclared and
// the fixer appended a second, inline copy — leaving two parameters with the
// same name and location, which is invalid.
func TestFixMissingPathParameters_ReferencedParametersNotDuplicated(t *testing.T) {
	tests := []struct {
		name string
		spec string
	}{
		{
			name: "OAS 2.0 path-item level $ref to root parameters",
			spec: `
swagger: "2.0"
info: {title: Test API, version: 1.0.0}
parameters:
  petIdParam: {name: petId, in: path, required: true, type: string}
paths:
  /pets/{petId}:
    parameters:
      - $ref: '#/parameters/petIdParam'
    get:
      operationId: getPet
      responses: {"200": {description: Success}}
`,
		},
		{
			name: "OAS 2.0 operation level $ref to root parameters",
			spec: `
swagger: "2.0"
info: {title: Test API, version: 1.0.0}
parameters:
  petIdParam: {name: petId, in: path, required: true, type: string}
paths:
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - $ref: '#/parameters/petIdParam'
      responses: {"200": {description: Success}}
`,
		},
		{
			name: "OAS 3.0 path-item level $ref to components",
			spec: `
openapi: 3.0.0
info: {title: Test API, version: 1.0.0}
components:
  parameters:
    petIdParam: {name: petId, in: path, required: true, schema: {type: string}}
paths:
  /pets/{petId}:
    parameters:
      - $ref: '#/components/parameters/petIdParam'
    get:
      operationId: getPet
      responses: {"200": {description: Success}}
`,
		},
		{
			name: "OAS 3.1 operation level $ref to components",
			spec: `
openapi: 3.1.0
info: {title: Test API, version: 1.0.0}
components:
  parameters:
    petIdParam: {name: petId, in: path, required: true, schema: {type: string}}
paths:
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - $ref: '#/components/parameters/petIdParam'
      responses: {"200": {description: Success}}
`,
		},
		{
			// The path item's parameters apply to every operation it contains,
			// so none of the three should gain a duplicate.
			name: "path-item $ref covers every operation in the item",
			spec: `
openapi: 3.0.0
info: {title: Test API, version: 1.0.0}
components:
  parameters:
    petIdParam: {name: petId, in: path, required: true, schema: {type: string}}
paths:
  /pets/{petId}:
    parameters:
      - $ref: '#/components/parameters/petIdParam'
    get:
      operationId: getPet
      responses: {"200": {description: Success}}
    put:
      operationId: updatePet
      responses: {"200": {description: Success}}
    delete:
      operationId: deletePet
      responses: {"204": {description: No Content}}
`,
		},
		{
			name: "chained $ref between component parameters",
			spec: `
openapi: 3.0.0
info: {title: Test API, version: 1.0.0}
components:
  parameters:
    aliasParam: {$ref: '#/components/parameters/petIdParam'}
    petIdParam: {name: petId, in: path, required: true, schema: {type: string}}
paths:
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - $ref: '#/components/parameters/aliasParam'
      responses: {"200": {description: Success}}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fixPathParams(t, tt.spec)

			assert.Empty(t, result.Fixes, "referenced path parameter is already declared; nothing to add")
		})
	}
}

// TestFixMissingPathParameters_UnresolvableRefIsLeftAlone checks that the
// fixer declines to guess when a $ref cannot be resolved. Adding a parameter
// the reference may already declare would produce a duplicate name+location,
// which is worse than leaving the document untouched.
func TestFixMissingPathParameters_UnresolvableRefIsLeftAlone(t *testing.T) {
	tests := []struct {
		name string
		spec string
	}{
		{
			name: "external file $ref",
			spec: `
openapi: 3.0.0
info: {title: Test API, version: 1.0.0}
paths:
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - $ref: 'shared.yaml#/components/parameters/petIdParam'
      responses: {"200": {description: Success}}
`,
		},
		{
			// The validator reports this as a wrong-kind reference; the fixer
			// deliberately diverges and stays silent. It cannot know what the
			// author meant, and adding a parameter beside a reference it does
			// not understand risks a duplicate name and location.
			name: "$ref to a component that is not a parameter",
			spec: `
openapi: 3.0.0
info: {title: Test API, version: 1.0.0}
components:
  schemas: {PetId: {type: string}}
paths:
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - $ref: '#/components/schemas/PetId'
      responses: {"200": {description: Success}}
`,
		},
		{
			name: "dangling local $ref",
			spec: `
openapi: 3.0.0
info: {title: Test API, version: 1.0.0}
components:
  parameters: {}
paths:
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - $ref: '#/components/parameters/missingParam'
      responses: {"200": {description: Success}}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fixPathParams(t, tt.spec)

			assert.Empty(t, result.Fixes, "an unresolvable $ref is unknown, not empty")
		})
	}
}

// TestFixMissingPathParameters_StillFixesGenuineGaps guards the fix from
// over-correcting: a path template parameter that really is undeclared must
// still be added, including alongside an unrelated resolvable $ref.
func TestFixMissingPathParameters_StillFixesGenuineGaps(t *testing.T) {
	spec := `
openapi: 3.0.0
info: {title: Test API, version: 1.0.0}
components:
  parameters:
    ownerIdParam: {name: ownerId, in: path, required: true, schema: {type: string}}
paths:
  /owners/{ownerId}/pets/{petId}:
    parameters:
      - $ref: '#/components/parameters/ownerIdParam'
    get:
      operationId: getPet
      responses: {"200": {description: Success}}
`

	result := fixPathParams(t, spec)

	require.Len(t, result.Fixes, 1, "only the undeclared petId should be added")
	assert.Equal(t, FixTypeMissingPathParameter, result.Fixes[0].Type)
	assert.Contains(t, result.Fixes[0].Description, "petId")

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok)
	op := doc.Paths["/owners/{ownerId}/pets/{petId}"].Get
	require.Len(t, op.Parameters, 1)
	assert.Equal(t, "petId", op.Parameters[0].Name)
	assert.Equal(t, parser.ParamInPath, op.Parameters[0].In)
}

// TestFixMissingPathParameters_RefToNonPathParameter checks that resolution
// respects the parameter's location: a $ref that resolves to a query parameter
// does not satisfy a path template.
func TestFixMissingPathParameters_RefToNonPathParameter(t *testing.T) {
	spec := `
openapi: 3.0.0
info: {title: Test API, version: 1.0.0}
components:
  parameters:
    limitParam: {name: limit, in: query, schema: {type: integer}}
paths:
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - $ref: '#/components/parameters/limitParam'
      responses: {"200": {description: Success}}
`

	result := fixPathParams(t, spec)

	require.Len(t, result.Fixes, 1)
	assert.Contains(t, result.Fixes[0].Description, "petId")
}
