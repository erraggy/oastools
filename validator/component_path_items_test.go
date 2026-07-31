package validator

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

// validationWarnings mirrors validationErrors for the warning severity, which is
// collected in a separate slice on the result. It exists for the negative
// assertion these tests turn on: the path-template checks must stay out of
// components.pathItems, and those report warnings rather than errors.
func validationWarnings(t *testing.T, spec string) []string {
	t.Helper()
	return warningsWithStrict(t, spec, false)
}

// warningsWithStrict is validationWarnings with StrictMode selectable, because
// the response-status-code arm of validateResponseStatusCodes is only reachable
// as a warning: the parser's own decoder rejects a malformed status code as a
// construct error before any document carrying one reaches the validator, so the
// strict-mode non-standard-code and missing-2xx warnings are what prove the
// responses check runs at all for a given call site.
func warningsWithStrict(t *testing.T, spec string, strict bool) []string {
	t.Helper()

	p := parser.New()
	p.ValidateStructure = false
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err, "test spec should parse")

	v := New()
	v.IncludeWarnings = true
	v.StrictMode = strict
	result, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err)

	messages := make([]string, 0, len(result.Warnings))
	for _, w := range result.Warnings {
		messages = append(messages, w.Path+": "+w.Message)
	}
	return messages
}

// TestComponentPathItemsOperationsAreValidated covers the first half of issue
// #392: operations under components.pathItems never reached
// validateOAS3Operation, so they skipped every check operations under paths and
// webhooks get. A reusable path item is reached by $ref and describes real
// operations, so the same defects must be reported there.
func TestComponentPathItemsOperationsAreValidated(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		wantErrors []string
	}{
		{
			name: "request body with no content",
			spec: `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  pathItems:
    Shared:
      post:
        operationId: sharedPost
        requestBody:
          description: no content object at all
        responses:
          "201": {description: Created}
`,
			wantErrors: []string{
				"components.pathItems.Shared.post.requestBody: RequestBody must have a content object with at least one media type",
			},
		},
		{
			name: "schema defect inside a request body media type",
			spec: `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  pathItems:
    Shared:
      post:
        operationId: sharedPost
        requestBody:
          content:
            application/json:
              schema: {type: string, enum: [1]}
        responses:
          "201": {description: Created}
`,
			wantErrors: []string{
				"components.pathItems.Shared.post.requestBody.content.application/json.schema.enum[0]: Enum value must be a string",
			},
		},
		{
			name: "invalid media type in a request body",
			spec: `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  pathItems:
    Shared:
      post:
        operationId: sharedPost
        requestBody:
          content:
            "not a media type": {schema: {type: object}}
        responses:
          "201": {description: Created}
`,
			wantErrors: []string{
				`components.pathItems.Shared.post.requestBody.content.not a media type: Invalid media type: not a media type`,
			},
		},
		{
			name: "path parameter missing required: true",
			spec: `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  pathItems:
    Shared:
      parameters:
        - {name: id, in: path, schema: {type: string}}
      get:
        operationId: sharedGet
        responses:
          "200": {description: OK}
`,
			wantErrors: []string{
				"components.pathItems.Shared.parameters[0]: Path parameters must have required: true",
			},
		},
		{
			name: "a well-formed reusable path item is clean",
			spec: `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  pathItems:
    Shared:
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      post:
        operationId: sharedPost
        requestBody:
          content:
            application/json: {schema: {type: object}}
        responses:
          "201": {description: Created}
`,
			wantErrors: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErrorsMatch(t, validationErrors(t, tt.spec), tt.wantErrors)
		})
	}
}

// TestComponentPathItemsResponsesAreValidated covers the responses half of #392
// separately, because validateResponseStatusCodes only ever reports warnings for
// a document that parses: a malformed code is rejected by the parser's decoder
// first. Strict mode's non-standard-code and missing-2xx warnings are therefore
// the observable proof that the check reaches these operations.
func TestComponentPathItemsResponsesAreValidated(t *testing.T) {
	spec := `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  pathItems:
    Shared:
      get:
        operationId: sharedGet
        summary: reusable
        responses:
          "299": {description: valid format, not a standard code}
      delete:
        operationId: sharedDelete
        summary: no success response
        responses:
          "404": {description: Not Found}
`
	warnings := strings.Join(warningsWithStrict(t, spec, true), "\n")

	assert.Contains(t, warnings,
		"components.pathItems.Shared.get.responses.299: Non-standard HTTP status code: 299")
	assert.Contains(t, warnings,
		"components.pathItems.Shared.delete.responses: Operation should define at least one successful response")
}

// TestComponentPathItemsSkipPathTemplateChecks pins the exclusion #392 states
// explicitly. A reusable path item has no path field, so the Paths-Object-scoped
// rule that a path parameter's name match a template expression cannot apply;
// running warnUnusedPathParams here would warn about every well-formed path
// parameter instead of finding a defect. Webhooks are excluded for the same
// reason, and are asserted alongside so the two stay consistent.
func TestComponentPathItemsSkipPathTemplateChecks(t *testing.T) {
	spec := `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths: {}
webhooks:
  onEvent:
    parameters:
      - {name: hookId, in: path, required: true, schema: {type: string}}
    post:
      operationId: onEventPost
      requestBody:
        content:
          application/json: {schema: {type: object}}
      responses:
        "200": {description: OK}
components:
  pathItems:
    Shared:
      parameters:
        - {name: id, in: path, required: true, schema: {type: string}}
      get:
        operationId: sharedGet
        summary: reusable
        responses:
          "200": {description: OK}
`
	for _, warning := range validationWarnings(t, spec) {
		assert.NotContains(t, warning, "not used in path template",
			"a reusable path item and a webhook have no path template, so a well-formed "+
				"path parameter in either must not be warned about")
	}
	assertErrorsMatch(t, validationErrors(t, spec), nil)
}

// TestComponentPathItemsDuplicateOperationIds covers the operationId decision
// #392 asks to settle: each components.pathItems entry counts exactly once,
// however many places $ref it and whether or not anything does.
func TestComponentPathItemsDuplicateOperationIds(t *testing.T) {
	tests := []struct {
		name       string
		spec       string
		wantErrors []string
	}{
		{
			// The sweep runs after paths, so the collision is reported at the
			// components site — where a rename belongs, since the path is the
			// live operation.
			name: "collides with an operation under paths",
			spec: `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    get:
      operationId: shared
      responses:
        "200": {description: OK}
components:
  pathItems:
    Shared:
      get:
        operationId: shared
        responses:
          "200": {description: OK}
`,
			wantErrors: []string{
				"components.pathItems.Shared.get: Duplicate operationId 'shared' (first seen at paths./pets.get)",
			},
		},
		{
			// Names are swept in sorted order, so "B" is always the one reported
			// as the duplicate rather than whichever map iteration reached first.
			name: "two reusable path items collide, reported in sorted name order",
			spec: `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  pathItems:
    B:
      get:
        operationId: shared
        responses:
          "200": {description: OK}
    A:
      get:
        operationId: shared
        responses:
          "200": {description: OK}
`,
			wantErrors: []string{
				"components.pathItems.B.get: Duplicate operationId 'shared' (first seen at components.pathItems.A.get)",
			},
		},
		{
			// The self-collision hazard the issue raises. Path item $refs are
			// preserved verbatim rather than resolved in place, so the use sites
			// carry no operations and the single declaration is counted once.
			name: "one reusable path item referenced from several places does not collide with itself",
			spec: `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths:
  /a:
    $ref: '#/components/pathItems/Shared'
  /b:
    $ref: '#/components/pathItems/Shared'
webhooks:
  onEvent:
    $ref: '#/components/pathItems/Shared'
components:
  pathItems:
    Shared:
      get:
        operationId: sharedGet
        responses:
          "200": {description: OK}
`,
			wantErrors: nil,
		},
		{
			// An entry nothing references arguably describes nothing, but an
			// operationId duplicating a live one is a latent defect one $ref from
			// mattering, so it is reported.
			name: "an unreferenced reusable path item is still counted",
			spec: `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    get:
      operationId: shared
      responses:
        "200": {description: OK}
components:
  pathItems:
    NeverReferenced:
      get:
        operationId: shared
        responses:
          "200": {description: OK}
`,
			wantErrors: []string{
				"components.pathItems.NeverReferenced.get: Duplicate operationId 'shared' (first seen at paths./pets.get)",
			},
		},
		{
			name: "distinct operationIds across paths and reusable path items are clean",
			spec: `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200": {description: OK}
components:
  pathItems:
    Shared:
      get:
        operationId: sharedGet
        responses:
          "200": {description: OK}
`,
			wantErrors: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErrorsMatch(t, validationErrors(t, tt.spec), tt.wantErrors)
		})
	}
}

// TestComponentPathItemsSortedOrderIsStable runs the sorted-order case enough
// times to fail if the sweep ever ranges the map directly. A single run passes
// by chance roughly half the time, which is what makes the ordering guarantee
// worth pinning separately from the collision itself.
func TestComponentPathItemsSortedOrderIsStable(t *testing.T) {
	spec := `
openapi: 3.1.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  pathItems:
    B:
      get:
        operationId: shared
        responses:
          "200": {description: OK}
    A:
      get:
        operationId: shared
        responses:
          "200": {description: OK}
    C:
      get:
        operationId: shared
        responses:
          "200": {description: OK}
`
	for range 50 {
		errs := validationErrors(t, spec)
		require.Len(t, errs, 2, "A wins the id; B and C are the duplicates")
		joined := strings.Join(errs, "\n")
		assert.Contains(t, joined, "components.pathItems.B.get: Duplicate operationId 'shared' (first seen at components.pathItems.A.get)")
		assert.Contains(t, joined, "components.pathItems.C.get: Duplicate operationId 'shared' (first seen at components.pathItems.A.get)")
	}
}

// TestComponentPathItemsAdditionalOperationsAreValidated covers the OAS 3.2
// methods, which GetOperations only surfaces at 3.2+. Validating the fixed
// methods while skipping query and additionalOperations would leave the same
// blind spot for exactly the operations most likely to be hand-written.
func TestComponentPathItemsAdditionalOperationsAreValidated(t *testing.T) {
	spec := `
openapi: 3.2.0
info: {title: T, version: "1.0.0"}
paths: {}
components:
  pathItems:
    Shared:
      query:
        operationId: sharedQuery
        requestBody:
          description: no content
        responses:
          "200": {description: OK}
      additionalOperations:
        PURGE:
          operationId: sharedPurge
          requestBody:
            content:
              application/json:
                schema: {type: string, enum: [1]}
          responses:
            "200": {description: OK}
`
	assertErrorsMatch(t, validationErrors(t, spec), []string{
		"components.pathItems.Shared.query.requestBody: RequestBody must have a content object with at least one media type",
		"components.pathItems.Shared.PURGE.requestBody.content.application/json.schema.enum[0]: Enum value must be a string",
	})
}
