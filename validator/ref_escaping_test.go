package validator

import (
	"testing"
)

// TestEscapedRefsResolve covers issue #379: reference lookups built pointers by
// raw concatenation while the parser's resolver correctly unescaped them per RFC
// 6901, so a component whose name contains "/" or "~" was unreachable to every
// check that resolves references and a valid document was reported as broken.
//
// OAS 2.0 places no charset constraint on the keys of the root-level
// parameters, definitions, and responses objects, so these documents are all
// legitimate and must validate completely clean.
//
// The OAS 3.x side is deliberately not here: those names are illegal per the
// Components Object charset, so the equivalent document is expected to produce
// exactly one error. See TestComponentNameCharset_EscapedRefStillResolves.
func TestEscapedRefsResolve(t *testing.T) {
	tests := []struct {
		name string
		spec string
	}{
		{
			// The parameter case from the issue. It failed as a false positive:
			// paramutil missed the escaped ref so Classify returned
			// ReasonNotAParameter, validRefs missed it identically, and
			// validateRef finally reported it as unresolvable.
			name: "escaped slash in OAS 2.0 parameter name",
			spec: `
swagger: "2.0"
info: {title: T, version: "1.0.0"}
parameters:
  pet/id: {name: petId, in: path, required: true, type: string}
paths:
  /pets/{petId}:
    get:
      summary: Get a pet
      parameters:
        - $ref: '#/parameters/pet~1id'
      responses: {"200": {description: OK}}
`,
		},
		{
			// Same failure for definitions, which is what showed the defect was
			// not parameter-specific.
			name: "escaped slash in OAS 2.0 definition name",
			spec: `
swagger: "2.0"
info: {title: T, version: "1.0.0"}
definitions:
  pet/summary: {type: object, properties: {id: {type: string}}}
paths:
  /pets:
    get:
      summary: List
      responses:
        "200":
          description: OK
          schema: {$ref: '#/definitions/pet~1summary'}
`,
		},
		{
			name: "escaped tilde in OAS 2.0 definition name",
			spec: `
swagger: "2.0"
info: {title: T, version: "1.0.0"}
definitions:
  pet~summary: {type: object, properties: {id: {type: string}}}
paths:
  /pets:
    get:
      summary: List
      responses:
        "200":
          description: OK
          schema: {$ref: '#/definitions/pet~0summary'}
`,
		},
		{
			name: "escaped slash in OAS 2.0 response name",
			spec: `
swagger: "2.0"
info: {title: T, version: "1.0.0"}
responses:
  not/found: {description: Not found}
paths:
  /pets:
    get:
      summary: List
      responses:
        "404": {$ref: '#/responses/not~1found'}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertErrorsMatch(t, validationErrors(t, tt.spec), nil)
		})
	}
}

// TestUnescapedRefDoesNotResolveEscapedName pins the tightening that came with
// the fix. "#/definitions/pet/summary" is a pointer naming definitions → "pet" →
// "summary", not the definition named "pet/summary", so it must not match.
//
// Raw concatenation used to make it match by accident. Asserting the new,
// stricter behavior keeps that from being reintroduced as a leniency.
func TestUnescapedRefDoesNotResolveEscapedName(t *testing.T) {
	spec := `
swagger: "2.0"
info: {title: T, version: "1.0.0"}
definitions:
  pet/summary: {type: object, properties: {id: {type: string}}}
paths:
  /pets:
    get:
      summary: List
      responses:
        "200":
          description: OK
          schema: {$ref: '#/definitions/pet/summary'}
`

	assertErrorsMatch(t, validationErrors(t, spec), []string{
		"$ref '#/definitions/pet/summary' does not resolve to a valid component",
	})
}
