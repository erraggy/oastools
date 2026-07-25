package validator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// undeclaredPathParamErrors returns the "not declared in parameters" errors
// from a validation result. That message is the symptom of issue #374: a path
// parameter hoisted into reusable definitions and referenced by $ref was
// invisible to the consistency check, because a $ref parameter carries no
// Name or In of its own.
func undeclaredPathParamErrors(t *testing.T, spec string) []string {
	t.Helper()

	p := parser.New()
	p.ValidateStructure = false // focus on validator behavior, not parser structure checks
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err, "test spec should parse")

	v := New()
	v.IncludeWarnings = true
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	var messages []string
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "not declared in parameters") {
			messages = append(messages, e.Path+": "+e.Message)
		}
	}
	return messages
}

func TestPathParameterConsistency_ReferencedParameters(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		wantErr bool
	}{
		{
			// Issue #374, reported against OAS 2.0: the parameter lives in the
			// document root's `parameters` and is referenced from the path item.
			name: "OAS 2.0 path-item level $ref to root parameters",
			spec: `
swagger: "2.0"
info: {title: Test, version: "1.0.0"}
parameters:
  petIdParam: {name: petId, in: path, required: true, type: string}
paths:
  /pets/{petId}:
    parameters:
      - $ref: '#/parameters/petIdParam'
    get:
      responses: {"200": {description: OK}}
`,
		},
		{
			name: "OAS 2.0 operation level $ref to root parameters",
			spec: `
swagger: "2.0"
info: {title: Test, version: "1.0.0"}
parameters:
  petIdParam: {name: petId, in: path, required: true, type: string}
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: '#/parameters/petIdParam'
      responses: {"200": {description: OK}}
`,
		},
		{
			name: "OAS 3.0 path-item level $ref to components",
			spec: `
openapi: 3.0.3
info: {title: Test, version: "1.0.0"}
components:
  parameters:
    petIdParam: {name: petId, in: path, required: true, schema: {type: string}}
paths:
  /pets/{petId}:
    parameters:
      - $ref: '#/components/parameters/petIdParam'
    get:
      responses: {"200": {description: OK}}
`,
		},
		{
			name: "OAS 3.1 operation level $ref to components",
			spec: `
openapi: 3.1.0
info: {title: Test, version: "1.0.0"}
components:
  parameters:
    petIdParam: {name: petId, in: path, required: true, schema: {type: string}}
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: '#/components/parameters/petIdParam'
      responses: {"200": {description: OK}}
`,
		},
		{
			// A path item's parameters apply to every operation it contains,
			// which is the "Operations contained within" half of the report.
			name: "path-item $ref covers every operation in the item",
			spec: `
openapi: 3.0.3
info: {title: Test, version: "1.0.0"}
components:
  parameters:
    petIdParam: {name: petId, in: path, required: true, schema: {type: string}}
paths:
  /pets/{petId}:
    parameters:
      - $ref: '#/components/parameters/petIdParam'
    get:
      responses: {"200": {description: OK}}
    put:
      responses: {"200": {description: OK}}
    delete:
      responses: {"204": {description: No Content}}
`,
		},
		{
			name: "multiple template parameters split across levels",
			spec: `
openapi: 3.0.3
info: {title: Test, version: "1.0.0"}
components:
  parameters:
    ownerIdParam: {name: ownerId, in: path, required: true, schema: {type: string}}
    petIdParam: {name: petId, in: path, required: true, schema: {type: string}}
paths:
  /owners/{ownerId}/pets/{petId}:
    parameters:
      - $ref: '#/components/parameters/ownerIdParam'
    get:
      parameters:
        - $ref: '#/components/parameters/petIdParam'
      responses: {"200": {description: OK}}
`,
		},
		{
			// A reusable definition may itself be a $ref; the chain is followed.
			name: "chained $ref between component parameters",
			spec: `
openapi: 3.0.3
info: {title: Test, version: "1.0.0"}
components:
  parameters:
    aliasParam: {$ref: '#/components/parameters/petIdParam'}
    petIdParam: {name: petId, in: path, required: true, schema: {type: string}}
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: '#/components/parameters/aliasParam'
      responses: {"200": {description: OK}}
`,
		},
		{
			// An unresolvable $ref is unknown, not empty: it may well declare
			// the missing name, so no undeclared-parameter error is reported.
			name: "external $ref suppresses the undeclared error",
			spec: `
openapi: 3.0.3
info: {title: Test, version: "1.0.0"}
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: 'shared.yaml#/components/parameters/petIdParam'
      responses: {"200": {description: OK}}
`,
		},
		{
			// Still a real defect, still reported.
			name: "genuinely undeclared path parameter still errors",
			spec: `
openapi: 3.0.3
info: {title: Test, version: "1.0.0"}
paths:
  /pets/{petId}:
    get:
      responses: {"200": {description: OK}}
`,
			wantErr: true,
		},
		{
			name: "OAS 2.0 genuinely undeclared path parameter still errors",
			spec: `
swagger: "2.0"
info: {title: Test, version: "1.0.0"}
paths:
  /pets/{petId}:
    get:
      responses: {"200": {description: OK}}
`,
			wantErr: true,
		},
		{
			// The $ref resolves, but to a query parameter, so the path
			// template parameter really is undeclared.
			name: "$ref to a non-path parameter still errors",
			spec: `
openapi: 3.0.3
info: {title: Test, version: "1.0.0"}
components:
  parameters:
    limitParam: {name: limit, in: query, schema: {type: integer}}
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: '#/components/parameters/limitParam'
      responses: {"200": {description: OK}}
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := undeclaredPathParamErrors(t, tt.spec)

			if tt.wantErr {
				assert.NotEmpty(t, errs, "expected an undeclared path parameter error")
			} else {
				assert.Empty(t, errs, "referenced path parameters should satisfy the path template")
			}
		})
	}
}

// TestPathParameterConsistency_DanglingRef checks that a broken local $ref is
// reported once, as a broken reference, rather than a second time as a
// misleading "parameter not declared" error. Both version paths suppress the
// second report, so both are covered.
func TestPathParameterConsistency_DanglingRef(t *testing.T) {
	tests := []struct {
		name string
		spec string
	}{
		{
			name: "OAS 2.0 dangling ref into root parameters",
			spec: `
swagger: "2.0"
info: {title: Test, version: "1.0.0"}
parameters: {}
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: '#/parameters/missingParam'
      responses: {"200": {description: OK}}
`,
		},
		{
			name: "OAS 3.0 dangling ref into components",
			spec: `
openapi: 3.0.3
info: {title: Test, version: "1.0.0"}
components:
  parameters: {}
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: '#/components/parameters/missingParam'
      responses: {"200": {description: OK}}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := parser.New()
			p.ValidateStructure = false
			parseResult, err := p.ParseBytes([]byte(tt.spec))
			require.NoError(t, err)

			result, err := New().ValidateParsed(*parseResult)
			require.NoError(t, err)

			var refErrors, undeclaredErrors int
			for _, e := range result.Errors {
				switch {
				case strings.Contains(e.Message, "does not resolve to a valid component"):
					refErrors++
				case strings.Contains(e.Message, "not declared in parameters"):
					undeclaredErrors++
				}
			}

			assert.Positive(t, refErrors, "the broken $ref itself should be reported")
			assert.Zero(t, undeclaredErrors, "the same defect should not be reported a second time")
		})
	}
}

// TestPathParameterConsistency_OAS2HasNoRequiredCheck documents why the
// required-reported-once tests below are OAS 3 only: OAS 2.0 has no
// "required: true" check for path parameters at all, so there is no per-path-item
// error for the OAS 2.0 path to report more than once.
//
// OAS 2.0 does mandate required: true on path parameters, so this is a real gap
// — but a pre-existing one, unrelated to $ref resolution. It is tracked in
// issue #378. This test pins the current behavior so that adding the check is a
// deliberate, visible change rather than a silent one: it fails once the check
// lands, at which point it should be replaced with the positive assertions.
func TestPathParameterConsistency_OAS2HasNoRequiredCheck(t *testing.T) {
	spec := `
swagger: "2.0"
info: {title: Test, version: "1.0.0"}
paths:
  /pets/{petId}:
    parameters:
      - name: petId
        in: path
        type: string
    get:
      responses: {"200": {description: OK}}
    put:
      responses: {"200": {description: OK}}
    delete:
      responses: {"204": {description: No Content}}
`

	p := parser.New()
	p.ValidateStructure = false
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := New().ValidateParsed(*parseResult)
	require.NoError(t, err)

	for _, e := range result.Errors {
		assert.NotContains(t, e.Message, "Path parameters must have required: true",
			"OAS 2.0 does not implement this check; update this test if that changes")
	}
}

// TestPathParameterConsistency_RequiredReportedOncePerPathItem guards against
// path-item level parameters being re-validated for every operation in the
// item, which produced one identical error per operation.
//
// OAS 3 only, per TestPathParameterConsistency_OAS2HasNoRequiredCheck.
func TestPathParameterConsistency_RequiredReportedOncePerPathItem(t *testing.T) {
	spec := `
openapi: 3.0.3
info: {title: Test, version: "1.0.0"}
paths:
  /pets/{petId}:
    parameters:
      - name: petId
        in: path
        schema: {type: string}
    get:
      responses: {"200": {description: OK}}
    put:
      responses: {"200": {description: OK}}
    delete:
      responses: {"204": {description: No Content}}
`

	p := parser.New()
	p.ValidateStructure = false
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := New().ValidateParsed(*parseResult)
	require.NoError(t, err)

	var requiredErrors []string
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "Path parameters must have required: true") {
			requiredErrors = append(requiredErrors, e.Path)
		}
	}

	assert.Equal(t, []string{"paths./pets/{petId}.parameters[0]"}, requiredErrors,
		"a path-item parameter defect should be reported once, not once per operation")
}

// TestPathParameterConsistency_RequiredSkippedForRefs checks that a referenced
// parameter missing required: true is reported at its definition rather than
// at every place it is used.
//
// OAS 3 only, per TestPathParameterConsistency_OAS2HasNoRequiredCheck.
func TestPathParameterConsistency_RequiredSkippedForRefs(t *testing.T) {
	spec := `
openapi: 3.0.3
info: {title: Test, version: "1.0.0"}
components:
  parameters:
    petIdParam: {name: petId, in: path, schema: {type: string}}
paths:
  /pets/{petId}:
    parameters:
      - $ref: '#/components/parameters/petIdParam'
    get:
      responses: {"200": {description: OK}}
    put:
      responses: {"200": {description: OK}}
`

	p := parser.New()
	p.ValidateStructure = false
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := New().ValidateParsed(*parseResult)
	require.NoError(t, err)

	var requiredErrors []string
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "Path parameters must have required: true") {
			requiredErrors = append(requiredErrors, e.Path)
		}
	}

	assert.Equal(t, []string{"components.parameters.petIdParam"}, requiredErrors,
		"the defect belongs to the definition, not to each use site")
}
