package validator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// validationErrors returns every error from validating spec, formatted as
// "path: message".
//
// It deliberately does not filter. An earlier version returned only messages
// containing "not declared in parameters" and asserted that set was empty,
// which cannot distinguish "correctly silent" from "wrongly silent" — any other
// error in the same document was invisible to it. That blind spot let a real
// regression pass: a parameter $ref pointing at a wrong-kind component
// suppressed the undeclared error, and the filtered assertion still went green.
// Assert on the whole error set so silence has to be earned.
func validationErrors(t *testing.T, spec string) []string {
	t.Helper()

	p := parser.New()
	p.ValidateStructure = false // focus on validator behavior, not parser structure checks
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err, "test spec should parse")

	v := New()
	v.IncludeWarnings = true
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	messages := make([]string, 0, len(result.Errors))
	for _, e := range result.Errors {
		messages = append(messages, e.Path+": "+e.Message)
	}
	return messages
}

// assertErrorsMatch checks that errs contains exactly one error per entry in
// wantSubstrings, matched by substring, and nothing else. Passing an empty
// wantSubstrings asserts the document is completely clean.
func assertErrorsMatch(t *testing.T, errs []string, wantSubstrings []string) {
	t.Helper()

	assert.Len(t, errs, len(wantSubstrings), "unexpected error count; got: %v", errs)
	for _, want := range wantSubstrings {
		matched := false
		for _, got := range errs {
			if strings.Contains(got, want) {
				matched = true
				break
			}
		}
		assert.True(t, matched, "expected an error containing %q; got: %v", want, errs)
	}
}

func TestPathParameterConsistency_ReferencedParameters(t *testing.T) {
	tests := []struct {
		name string
		spec string
		// wantErrors lists a substring per expected error. Empty means the
		// document must validate completely clean — not merely free of
		// undeclared-parameter errors.
		wantErrors []string
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
			// A reusable definition may itself be a $ref; the chain is followed,
			// and the alias itself is clean. validateOAS3Components skips $ref
			// entries rather than telling a pure alias it needs a schema or
			// content — a reference carries neither by design.
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
			wantErrors: nil,
		},
		{
			// An unresolvable $ref is unknown, not empty: it may well declare
			// the missing name, so no undeclared-parameter error is reported.
			// External refs are the ONE cause of unresolvability that stays
			// silent by design — every other cause is reported by some check.
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
			wantErrors: []string{"Path template references parameter '{petId}' but it is not declared in parameters"},
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
			wantErrors: []string{"Path template references parameter '{petId}' but it is not declared in parameters"},
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
			wantErrors: []string{"Path template references parameter '{petId}' but it is not declared in parameters"},
		},
		{
			// REGRESSION GUARD. A $ref naming a component that exists but is not
			// a parameter is accepted by validateRef — validRefs is one flat set
			// across all component kinds — and is unresolvable to the parameter
			// resolver, which suppresses the undeclared-parameter check. Before
			// validateParameterRefKinds existed, this document validated clean
			// while erroring on main: a false positive traded for a false negative.
			name: "OAS 3.x $ref to a schema in a parameter slot is reported",
			spec: `
openapi: 3.0.3
info: {title: Test, version: "1.0.0"}
components:
  schemas: {PetId: {type: string}}
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: '#/components/schemas/PetId'
      responses: {"200": {description: OK}}
`,
			wantErrors: []string{"$ref '#/components/schemas/PetId' resolves to a component that is not a parameter definition"},
		},
		{
			name: "OAS 2.0 $ref to a definition in a parameter slot is reported",
			spec: `
swagger: "2.0"
info: {title: Test, version: "1.0.0"}
definitions: {PetId: {type: string}}
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: '#/definitions/PetId'
      responses: {"200": {description: OK}}
`,
			wantErrors: []string{"$ref '#/definitions/PetId' resolves to a component that is not a parameter definition"},
		},
		{
			// Same defect at path-item level, which applies to every operation.
			name: "wrong-kind $ref at path-item level is reported once",
			spec: `
openapi: 3.0.3
info: {title: Test, version: "1.0.0"}
components:
  schemas: {PetId: {type: string}}
paths:
  /pets/{petId}:
    parameters:
      - $ref: '#/components/schemas/PetId'
    get:
      responses: {"200": {description: OK}}
    put:
      responses: {"200": {description: OK}}
`,
			wantErrors: []string{"paths./pets/{petId}.parameters[0]: $ref '#/components/schemas/PetId' resolves to a component that is not a parameter definition"},
		},
		{
			// REGRESSION GUARD. The wrong-kind ref sits one hop DOWN, on the
			// definition rather than at the use site. The use site defers to the
			// definition because the ref does name a parameter; reference
			// validation accepts the schema because it exists; and the failed
			// resolution suppresses the undeclared check. Without a check over
			// the definitions themselves, the document validates clean.
			//
			// The alias carries a schema so the pre-existing "schema or content"
			// false positive cannot mask the result.
			name: "definition aliasing a non-parameter is reported at the definition",
			spec: `
openapi: 3.0.3
info: {title: Test, version: "1.0.0"}
components:
  schemas: {PetId: {type: string}}
  parameters:
    aliasParam: {$ref: '#/components/schemas/PetId', schema: {type: string}}
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: '#/components/parameters/aliasParam'
      responses: {"200": {description: OK}}
`,
			wantErrors: []string{"components.parameters.aliasParam: $ref '#/components/schemas/PetId' resolves to a component that is not a parameter definition"},
		},
		{
			name: "OAS 2.0 definition aliasing a non-parameter is reported at the definition",
			spec: `
swagger: "2.0"
info: {title: Test, version: "1.0.0"}
definitions: {PetId: {type: string}}
parameters:
  aliasParam: {$ref: '#/definitions/PetId', type: string}
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: '#/parameters/aliasParam'
      responses: {"200": {description: OK}}
`,
			wantErrors: []string{"parameters.aliasParam: $ref '#/definitions/PetId' resolves to a component that is not a parameter definition"},
		},
		{
			// The defect belongs to the definition, so referencing it from
			// several operations must not multiply the error.
			name: "definition-site wrong-kind is reported once regardless of use count",
			spec: `
openapi: 3.0.3
info: {title: Test, version: "1.0.0"}
components:
  schemas: {PetId: {type: string}}
  parameters:
    aliasParam: {$ref: '#/components/schemas/PetId', schema: {type: string}}
paths:
  /pets/{petId}:
    parameters:
      - $ref: '#/components/parameters/aliasParam'
    get:
      responses: {"200": {description: OK}}
    put:
      responses: {"200": {description: OK}}
    delete:
      responses: {"204": {description: No Content}}
`,
			wantErrors: []string{"components.parameters.aliasParam: $ref '#/components/schemas/PetId' resolves to a component that is not a parameter definition"},
		},
		{
			// REGRESSION GUARD. A cycle is built entirely from references that
			// individually exist, so reference validation has nothing to object
			// to, and every member IS a parameter, so the wrong-kind check skips
			// it. Giving the members a schema also silences the incidental
			// "schema or content" error that used to mask this. Without a cycle
			// check the whole document validates clean.
			name: "reference cycle between parameter definitions is reported",
			spec: `
openapi: 3.0.3
info: {title: Test, version: "1.0.0"}
components:
  parameters:
    loopA: {$ref: '#/components/parameters/loopB', schema: {type: string}}
    loopB: {$ref: '#/components/parameters/loopA', schema: {type: string}}
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: '#/components/parameters/loopA'
      responses: {"200": {description: OK}}
`,
			wantErrors: []string{"$ref '#/components/parameters/loopA' leads to a reference cycle between parameter definitions"},
		},
		{
			// Distinct from a cycle: nothing repeats, the chain is just longer
			// than the resolver follows. Reported rather than silently suppressed,
			// since suppression would hide whatever the chain actually declares.
			name: "too-long reference chain is reported, and not called a cycle",
			spec: `
openapi: 3.0.3
info: {title: Test, version: "1.0.0"}
components:
  parameters:
    a1: {$ref: '#/components/parameters/a2', schema: {type: string}}
    a2: {$ref: '#/components/parameters/a3', schema: {type: string}}
    a3: {$ref: '#/components/parameters/a4', schema: {type: string}}
    a4: {$ref: '#/components/parameters/a5', schema: {type: string}}
    a5: {$ref: '#/components/parameters/a6', schema: {type: string}}
    a6: {$ref: '#/components/parameters/a7', schema: {type: string}}
    a7: {$ref: '#/components/parameters/a8', schema: {type: string}}
    a8: {$ref: '#/components/parameters/a9', schema: {type: string}}
    a9: {$ref: '#/components/parameters/a10', schema: {type: string}}
    a10: {$ref: '#/components/parameters/a11', schema: {type: string}}
    a11: {$ref: '#/components/parameters/a12', schema: {type: string}}
    a12: {name: petId, in: path, required: true, schema: {type: string}}
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: '#/components/parameters/a1'
      responses: {"200": {description: OK}}
`,
			wantErrors: []string{"$ref '#/components/parameters/a1' leads to a parameter reference chain too long to follow"},
		},
		{
			// A ref that DOES name a parameter but whose chain breaks further
			// down is a different defect, and must not be mislabeled wrong-kind.
			// Defines, not Resolve, is what draws that line.
			name: "$ref to a parameter whose own chain is broken is not called wrong-kind",
			spec: `
openapi: 3.0.3
info: {title: Test, version: "1.0.0"}
components:
  parameters:
    aliasParam: {$ref: '#/components/parameters/goneParam'}
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: '#/components/parameters/aliasParam'
      responses: {"200": {description: OK}}
`,
			// Only the dangling ref is reported. The alias is not additionally
			// told it needs a schema or content: that check now skips $ref
			// entries, so a broken chain is named once rather than twice.
			wantErrors: []string{
				"components.parameters.aliasParam: $ref '#/components/parameters/goneParam' does not resolve to a valid component",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErrorsMatch(t, validationErrors(t, tt.spec), tt.wantErrors)
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

// TestPathParameterConsistency_OAS2RequiredReportedOncePerPathItem is the OAS
// 2.0 counterpart of TestPathParameterConsistency_RequiredReportedOncePerPathItem.
//
// Swagger 2.0 mandates required: true on path parameters just as OAS 3.x does,
// and both versions run the same check, so both must dedup a path-item
// parameter to a single error rather than repeating it per operation. Three
// operations are declared so a regression to the per-operation placement shows
// up as three errors.
func TestPathParameterConsistency_OAS2RequiredReportedOncePerPathItem(t *testing.T) {
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

	assert.Equal(t, []string{"paths./pets/{petId}.parameters[0]"}, requiredErrorPaths(t, spec),
		"a path-item parameter defect should be reported once, not once per operation")
}

// TestPathParameterConsistency_OAS2RequiredSkippedForRefs is the OAS 2.0
// counterpart of TestPathParameterConsistency_RequiredSkippedForRefs: a
// referenced parameter is reported at its definition in the root-level
// parameters, not at every use site.
func TestPathParameterConsistency_OAS2RequiredSkippedForRefs(t *testing.T) {
	spec := `
swagger: "2.0"
info: {title: Test, version: "1.0.0"}
parameters:
  petIdParam: {name: petId, in: path, type: string}
paths:
  /pets/{petId}:
    parameters:
      - $ref: '#/parameters/petIdParam'
    get:
      responses: {"200": {description: OK}}
    put:
      responses: {"200": {description: OK}}
`

	assert.Equal(t, []string{"parameters.petIdParam"}, requiredErrorPaths(t, spec),
		"the defect belongs to the definition, not to each use site")
}

// TestPathParameterConsistency_OAS2AliasNotRequiredChecked pins the interaction
// between the two halves of issue #378: the definition-site required check must
// not fire on a pure $ref alias. An alias carries no In, so it cannot match
// in: path — this asserts that rather than assuming it.
func TestPathParameterConsistency_OAS2AliasNotRequiredChecked(t *testing.T) {
	spec := `
swagger: "2.0"
info: {title: Test, version: "1.0.0"}
parameters:
  aliasParam: {$ref: '#/parameters/petIdParam'}
  petIdParam: {name: petId, in: path, required: true, type: string}
paths:
  /pets/{petId}:
    get:
      parameters:
        - $ref: '#/parameters/aliasParam'
      responses: {"200": {description: OK}}
`

	assert.Empty(t, requiredErrorPaths(t, spec),
		"a pure $ref alias has no In and must not be treated as a path parameter")
}

// requiredErrorPaths collects the paths of every "required: true" error the
// validator reports for spec, so a test can assert both how many were reported
// and where.
func requiredErrorPaths(t *testing.T, spec string) []string {
	t.Helper()

	p := parser.New()
	p.ValidateStructure = false
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := New().ValidateParsed(*parseResult)
	require.NoError(t, err)

	var paths []string
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "Path parameters must have required: true") {
			paths = append(paths, e.Path)
		}
	}
	return paths
}

// TestPathParameterConsistency_RequiredReportedOncePerPathItem guards against
// path-item level parameters being re-validated for every operation in the
// item, which produced one identical error per operation.
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

	assert.Equal(t, []string{"paths./pets/{petId}.parameters[0]"}, requiredErrorPaths(t, spec),
		"a path-item parameter defect should be reported once, not once per operation")
}

// TestPathParameterConsistency_RequiredCheckedWithoutOperations pins a
// behavior WIDENING introduced alongside the dedup fix.
//
// The path-item required check previously lived inside the per-operation loop,
// so a path item with parameters but no operations never ran it — such an item
// is legal (it may carry only summary, servers, or a $ref) and its parameters
// are still bound by the rule. Hoisting the check out of that loop means it now
// runs regardless, which can turn a previously-passing document invalid.
//
// The new behavior is correct; this test exists so the widening is deliberate
// and visible rather than an unnoticed side effect of the restructure.
func TestPathParameterConsistency_RequiredCheckedWithoutOperations(t *testing.T) {
	spec := `
openapi: 3.0.3
info: {title: Test, version: "1.0.0"}
paths:
  /pets/{petId}:
    description: legal path item carrying parameters but no operations
    parameters:
      - name: petId
        in: path
        schema: {type: string}
`

	assertErrorsMatch(t, validationErrors(t, spec), []string{
		"paths./pets/{petId}.parameters[0]: Path parameters must have required: true",
	})
}

// TestPathParameterConsistency_RequiredSkippedForRefs checks that a referenced
// parameter missing required: true is reported at its definition rather than
// at every place it is used.
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

	assert.Equal(t, []string{"components.parameters.petIdParam"}, requiredErrorPaths(t, spec),
		"the defect belongs to the definition, not to each use site")
}
