package validator

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Component sections were not all introduced at once, so each case declares the
// earliest version in which its section is a legal Components field. Using one
// blanket version would test some sections against a document that may not
// legally carry them.
const (
	oas31 = "3.1.0" // pathItems
	oas32 = "3.2.0" // mediaTypes
)

// oas3WithComponentSection builds a minimal OAS 3.x document declaring one
// component of the given kind under the given name.
func oas3WithComponentSection(t *testing.T, version, field, name, body string) string {
	t.Helper()

	return fmt.Sprintf(`openapi: %s
info: {title: T, version: "1.0.0"}
components:
  %s:
    %q: %s
paths: {}
`, version, field, name, body)
}

// TestComponentNameCharset covers issue #380: OAS 3.x requires every fixed field
// of the Components Object to use keys matching ^[a-zA-Z0-9\.\-_]+$, and the
// validator enforced this nowhere.
//
// Every section is exercised rather than schemas alone, because the previous gap
// was precisely that only schemas had any name validation at all.
func TestComponentNameCharset(t *testing.T) {
	for field, section := range componentSections() {
		t.Run(field+"/rejects slash", func(t *testing.T) {
			spec := oas3WithComponentSection(t, section.version, field, "pet/summary", section.body)
			assertErrorsMatch(t, validationErrors(t, spec), []string{
				fmt.Sprintf(`components.%s.pet/summary: Component name "pet/summary" must match`, field),
			})
		})

		t.Run(field+"/accepts legal name", func(t *testing.T) {
			spec := oas3WithComponentSection(t, section.version, field, "Pet.Summary-v2_1", section.body)
			assertErrorsMatch(t, validationErrors(t, spec), nil)
		})
	}
}

// componentSections pairs each Components field with the earliest OAS version
// that accepts it and a minimal valid value, so the only error a case can
// produce is the name one.
func componentSections() map[string]struct{ version, body string } {
	return map[string]struct{ version, body string }{
		"schemas":         {oas31, `{type: object}`},
		"responses":       {oas31, `{description: OK}`},
		"parameters":      {oas31, `{name: q, in: query, schema: {type: string}}`},
		"examples":        {oas31, `{value: 1}`},
		"requestBodies":   {oas31, `{content: {application/json: {schema: {type: object}}}}`},
		"headers":         {oas31, `{schema: {type: string}}`},
		"securitySchemes": {oas31, `{type: apiKey, name: k, in: header}`},
		"links":           {oas31, `{operationId: getPet}`},
		"callbacks":       {oas31, `{}`},
		"pathItems":       {oas31, `{}`},                       // OAS 3.1+
		"mediaTypes":      {oas32, `{schema: {type: object}}`}, // OAS 3.2+
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
			spec := oas3WithComponentSection(t, oas31, "schemas", name, `{type: object}`)
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
			spec := oas3WithComponentSection(t, oas31, "schemas", name, `{type: object}`)
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
//
// Both blank forms are covered because they are separate branches: a
// whitespace-only name also fails the charset pattern, so suppressing it takes
// its own guard, whereas an empty name would be caught by either check.
func TestComponentNameCharsetSkipsBlankNames(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "", want: "schema name cannot be empty"},
		{name: "   ", want: "schema name cannot be whitespace-only"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%q", tt.name), func(t *testing.T) {
			spec := oas3WithComponentSection(t, oas31, "schemas", tt.name, `{type: object}`)

			errs := validationErrors(t, spec)
			assert.Len(t, errs, 1,
				"the blank name should be reported once, not also as a charset violation; got: %v", errs)
			assert.Contains(t, strings.Join(errs, "\n"), tt.want)
		})
	}
}

// TestComponentNameCharsetMixedSection covers a section holding a legal name
// alongside an illegal one.
//
// checkComponentNames scans for a defect before building any reporting state, so
// the reporting loop only ever runs for a section that has at least one, and it
// must then still skip the legal names it walks past. Every other case here uses
// either all legal names or a single illegal one, so neither exercises a legal
// name reached during reporting.
func TestComponentNameCharsetMixedSection(t *testing.T) {
	spec := `openapi: ` + oas31 + `
info: {title: T, version: "1.0.0"}
components:
  schemas:
    "Legal.Name-v2_1": {type: object}
    "pet/summary": {type: object}
paths: {}
`

	errs := validationErrors(t, spec)
	assert.Len(t, errs, 1,
		"only the illegal name should be reported; got: %v", errs)
	assert.Contains(t, strings.Join(errs, "\n"), `components.schemas.pet/summary`)
	assert.NotContains(t, strings.Join(errs, "\n"), "Legal.Name-v2_1",
		"the legal name must not be reported")
}

// TestComponentBlankNames covers the sections validateSchemaName does not reach.
//
// Only schemas had any name validation before this check, so a blank key
// anywhere else went unreported entirely. It is reported as a missing name
// rather than as a charset violation, because that describes the defect and
// says what to do about it.
func TestComponentBlankNames(t *testing.T) {
	for field, section := range componentSections() {
		if field == componentSchemasName {
			continue // validateSchemaName owns these; see the test above
		}

		t.Run(field+"/empty", func(t *testing.T) {
			spec := oas3WithComponentSection(t, section.version, field, "", section.body)
			assertErrorsMatch(t, validationErrors(t, spec), []string{
				fmt.Sprintf("components.%s: Component name cannot be empty", field),
			})
		})

		t.Run(field+"/whitespace only", func(t *testing.T) {
			spec := oas3WithComponentSection(t, section.version, field, "   ", section.body)
			assertErrorsMatch(t, validationErrors(t, spec), []string{
				fmt.Sprintf("components.%s.   : Component name cannot be whitespace-only", field),
			})
		})
	}
}
