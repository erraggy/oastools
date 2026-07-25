package validator

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// oas3WithComponentSection builds a minimal OAS 3.1 document declaring one
// component of the given kind under the given name. 3.1 is used so pathItems is
// legal; every other section exists in 3.0 as well.
func oas3WithComponentSection(t *testing.T, field, name, body string) string {
	t.Helper()

	return fmt.Sprintf(`openapi: 3.1.0
info: {title: T, version: "1.0.0"}
components:
  %s:
    %q: %s
paths: {}
`, field, name, body)
}

// TestComponentNameCharset covers issue #380: OAS 3.x requires every fixed field
// of the Components Object to use keys matching ^[a-zA-Z0-9\.\-_]+$, and the
// validator enforced this nowhere.
//
// Every section is exercised rather than schemas alone, because the previous gap
// was precisely that only schemas had any name validation at all.
func TestComponentNameCharset(t *testing.T) {
	// bodies are the minimal valid value for each section, so the only error a
	// case can produce is the name one.
	bodies := map[string]string{
		"schemas":         `{type: object}`,
		"responses":       `{description: OK}`,
		"parameters":      `{name: q, in: query, schema: {type: string}}`,
		"examples":        `{value: 1}`,
		"requestBodies":   `{content: {application/json: {schema: {type: object}}}}`,
		"headers":         `{schema: {type: string}}`,
		"securitySchemes": `{type: apiKey, name: k, in: header}`,
		"links":           `{operationId: getPet}`,
		"callbacks":       `{}`,
		"pathItems":       `{}`,
		"mediaTypes":      `{schema: {type: object}}`,
	}

	for field, body := range bodies {
		t.Run(field+"/rejects slash", func(t *testing.T) {
			spec := oas3WithComponentSection(t, field, "pet/summary", body)
			assertErrorsMatch(t, validationErrors(t, spec), []string{
				fmt.Sprintf(`components.%s.pet/summary: Component name "pet/summary" must match`, field),
			})
		})

		t.Run(field+"/accepts legal name", func(t *testing.T) {
			spec := oas3WithComponentSection(t, field, "Pet.Summary-v2_1", body)
			assertErrorsMatch(t, validationErrors(t, spec), nil)
		})
	}
}

// TestComponentNameCharsetRejectedCharacters pins the boundary of the pattern
// rather than a single representative violation.
func TestComponentNameCharsetRejectedCharacters(t *testing.T) {
	rejected := []string{
		"pet/summary",
		"pet~summary",
		"pet summary",
		"pet:summary",
		"pet+summary",
		"pet#summary",
		"pet@summary",
		"pet%summary",
		"pet[0]",
		"pét",
	}

	for _, name := range rejected {
		t.Run(name, func(t *testing.T) {
			spec := oas3WithComponentSection(t, "schemas", name, `{type: object}`)
			errs := validationErrors(t, spec)
			assert.Len(t, errs, 1, "expected exactly the name error; got: %v", errs)
			assert.Contains(t, strings.Join(errs, "\n"), "must match",
				"name %q should be rejected by the charset rule", name)
		})
	}
}

// TestComponentNameCharsetAcceptedCharacters is the control: the pattern permits
// letters, digits, dot, hyphen, and underscore, and real specs lean on all of
// them. msgraph alone contributes thousands of dotted names like
// microsoft.graph.user.
func TestComponentNameCharsetAcceptedCharacters(t *testing.T) {
	accepted := []string{
		"Pet",
		"pet",
		"Pet2",
		"microsoft.graph.user",
		"Pet-Summary",
		"Pet_Summary",
		"_leading",
		"-leading",
		".leading",
		"9",
	}

	for _, name := range accepted {
		t.Run(name, func(t *testing.T) {
			spec := oas3WithComponentSection(t, "schemas", name, `{type: object}`)
			assertErrorsMatch(t, validationErrors(t, spec), nil)
		})
	}
}

// TestComponentNameCharsetNotAppliedToOAS2 guards the constraint that made this
// check version-specific.
//
// OAS 2.0 places no charset constraint on the keys of its root-level
// parameters, definitions, and responses objects. Names containing "/" or "~"
// are legitimate there and are referenced with RFC 6901 escaping, so applying
// the OAS 3.x pattern to a 2.0 document would reject valid specs — the exact
// false positive #379 fixed.
func TestComponentNameCharsetNotAppliedToOAS2(t *testing.T) {
	spec := `
swagger: "2.0"
info: {title: T, version: "1.0.0"}
definitions:
  pet/summary: {type: object, properties: {id: {type: string}}}
parameters:
  pet/id: {name: petId, in: path, required: true, type: string}
responses:
  not/found: {description: Not found}
paths:
  /pets/{petId}:
    get:
      summary: Get a pet
      parameters:
        - $ref: '#/parameters/pet~1id'
      responses:
        "200":
          description: OK
          schema: {$ref: '#/definitions/pet~1summary'}
        "404": {$ref: '#/responses/not~1found'}
`

	assertErrorsMatch(t, validationErrors(t, spec), nil)
}

// TestComponentNameCharset_EscapedRefStillResolves records the interaction
// between this check and #379.
//
// Before #379 this document failed with a misleading "does not resolve to a
// valid component" error, because reference lookups did not escape. After it,
// the reference resolved and the document validated clean — accepting a name the
// specification forbids. Both halves are now right: exactly one error, and it
// names the real defect.
func TestComponentNameCharset_EscapedRefStillResolves(t *testing.T) {
	spec := `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
components:
  schemas:
    pet/summary: {type: object, properties: {id: {type: string}}}
paths:
  /pets:
    get:
      summary: List
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/pet~1summary'}
`

	assertErrorsMatch(t, validationErrors(t, spec), []string{
		`components.schemas.pet/summary: Component name "pet/summary" must match`,
	})
}

// TestComponentNameCharsetSkipsBlankNames keeps this check from double-reporting
// a defect validateSchemaName already owns. An empty or whitespace-only name is
// a missing name rather than a malformed one, and is reported as such.
func TestComponentNameCharsetSkipsBlankNames(t *testing.T) {
	spec := `
openapi: 3.0.3
info: {title: T, version: "1.0.0"}
components:
  schemas:
    "": {type: object}
paths: {}
`

	errs := validationErrors(t, spec)
	assert.Len(t, errs, 1, "the blank name should be reported once, not also as a charset violation; got: %v", errs)
	assert.Contains(t, strings.Join(errs, "\n"), "schema name cannot be empty")
}
