package fixer

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/erraggy/oastools/internal/naming"
	"github.com/erraggy/oastools/internal/pathutil"
	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// slashCase is one spelling of a reference to a generic schema whose type
// parameter contains "/". Every case names the same schema; only the encoding
// of the $ref that reaches it differs.
type slashCase struct {
	name     string
	oas2Spec string
	oas3Spec string
	// schemaName is the definitions/components key the refs point at before the
	// fix. It must be gone from the fixed document: a rename that left it in
	// place would satisfy every other assertion here.
	schemaName string
	// control marks a case that worked before issue #404 was fixed. They are
	// carried so a change to the matching rule cannot quietly break the
	// spellings that already resolved.
	control bool
}

// slashCases enumerates the encodings from issue #404, plus the two controls the
// issue identifies. Specs are written out per case rather than templated: the
// bug was a byte-level mismatch between a $ref and a schema key, so the exact
// literal spelling of each is the thing under test.
var slashCases = []slashCase{
	{
		name:       "no slash, percent-encoded brackets",
		schemaName: "PagedResponse[store.Pet]",
		control:    true,
		oas2Spec: `
swagger: "2.0"
info: {title: T, version: "1.0"}
definitions:
  PagedResponse[store.Pet]:
    type: object
    properties:
      items: {type: array, items: {$ref: '#/definitions/store.Pet'}}
  store.Pet: {type: object, properties: {name: {type: string}}}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          schema: {$ref: '#/definitions/PagedResponse%5Bstore.Pet%5D'}
`,
		oas3Spec: `
openapi: 3.0.3
info: {title: T, version: "1.0"}
components:
  schemas:
    PagedResponse[store.Pet]:
      type: object
      properties:
        items: {type: array, items: {$ref: '#/components/schemas/store.Pet'}}
    store.Pet: {type: object, properties: {name: {type: string}}}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/PagedResponse%5Bstore.Pet%5D'}
`,
	},
	{
		name:       "nested generic, no slash",
		schemaName: "PagedResponse[store.Record[store.Pet]]",
		control:    true,
		oas2Spec: `
swagger: "2.0"
info: {title: T, version: "1.0"}
definitions:
  PagedResponse[store.Record[store.Pet]]:
    type: object
    properties:
      items: {type: array, items: {$ref: '#/definitions/store.Pet'}}
  store.Pet: {type: object, properties: {name: {type: string}}}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          schema: {$ref: '#/definitions/PagedResponse%5Bstore.Record%5Bstore.Pet%5D%5D'}
`,
		oas3Spec: `
openapi: 3.0.3
info: {title: T, version: "1.0"}
components:
  schemas:
    PagedResponse[store.Record[store.Pet]]:
      type: object
      properties:
        items: {type: array, items: {$ref: '#/components/schemas/store.Pet'}}
    store.Pet: {type: object, properties: {name: {type: string}}}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/PagedResponse%5Bstore.Record%5Bstore.Pet%5D%5D'}
`,
	},
	{
		name:       "percent-encoded brackets, raw slashes",
		schemaName: "PagedResponse[example.com/petstore/pkg/store.Pet]",
		oas2Spec: `
swagger: "2.0"
info: {title: T, version: "1.0"}
definitions:
  PagedResponse[example.com/petstore/pkg/store.Pet]:
    type: object
    properties:
      items: {type: array, items: {$ref: '#/definitions/store.Pet'}}
  store.Pet: {type: object, properties: {name: {type: string}}}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          schema: {$ref: '#/definitions/PagedResponse%5Bexample.com/petstore/pkg/store.Pet%5D'}
`,
		oas3Spec: `
openapi: 3.0.3
info: {title: T, version: "1.0"}
components:
  schemas:
    PagedResponse[example.com/petstore/pkg/store.Pet]:
      type: object
      properties:
        items: {type: array, items: {$ref: '#/components/schemas/store.Pet'}}
    store.Pet: {type: object, properties: {name: {type: string}}}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/PagedResponse%5Bexample.com/petstore/pkg/store.Pet%5D'}
`,
	},
	{
		name:       "raw brackets, raw slashes",
		schemaName: "PagedResponse[example.com/petstore/pkg/store.Pet]",
		oas2Spec: `
swagger: "2.0"
info: {title: T, version: "1.0"}
definitions:
  PagedResponse[example.com/petstore/pkg/store.Pet]:
    type: object
    properties:
      items: {type: array, items: {$ref: '#/definitions/store.Pet'}}
  store.Pet: {type: object, properties: {name: {type: string}}}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          schema: {$ref: '#/definitions/PagedResponse[example.com/petstore/pkg/store.Pet]'}
`,
		oas3Spec: `
openapi: 3.0.3
info: {title: T, version: "1.0"}
components:
  schemas:
    PagedResponse[example.com/petstore/pkg/store.Pet]:
      type: object
      properties:
        items: {type: array, items: {$ref: '#/components/schemas/store.Pet'}}
    store.Pet: {type: object, properties: {name: {type: string}}}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/PagedResponse[example.com/petstore/pkg/store.Pet]'}
`,
	},
	{
		name:       "percent-encoded brackets, JSON Pointer slashes",
		schemaName: "PagedResponse[example.com/petstore/pkg/store.Pet]",
		oas2Spec: `
swagger: "2.0"
info: {title: T, version: "1.0"}
definitions:
  PagedResponse[example.com/petstore/pkg/store.Pet]:
    type: object
    properties:
      items: {type: array, items: {$ref: '#/definitions/store.Pet'}}
  store.Pet: {type: object, properties: {name: {type: string}}}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          schema: {$ref: '#/definitions/PagedResponse%5Bexample.com~1petstore~1pkg~1store.Pet%5D'}
`,
		oas3Spec: `
openapi: 3.0.3
info: {title: T, version: "1.0"}
components:
  schemas:
    PagedResponse[example.com/petstore/pkg/store.Pet]:
      type: object
      properties:
        items: {type: array, items: {$ref: '#/components/schemas/store.Pet'}}
    store.Pet: {type: object, properties: {name: {type: string}}}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/PagedResponse%5Bexample.com~1petstore~1pkg~1store.Pet%5D'}
`,
	},
	{
		name:       "fully percent-encoded",
		schemaName: "PagedResponse[example.com/petstore/pkg/store.Pet]",
		oas2Spec: `
swagger: "2.0"
info: {title: T, version: "1.0"}
definitions:
  PagedResponse[example.com/petstore/pkg/store.Pet]:
    type: object
    properties:
      items: {type: array, items: {$ref: '#/definitions/store.Pet'}}
  store.Pet: {type: object, properties: {name: {type: string}}}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          schema: {$ref: '#/definitions/PagedResponse%5Bexample.com%2Fpetstore%2Fpkg%2Fstore.Pet%5D'}
`,
		oas3Spec: `
openapi: 3.0.3
info: {title: T, version: "1.0"}
components:
  schemas:
    PagedResponse[example.com/petstore/pkg/store.Pet]:
      type: object
      properties:
        items: {type: array, items: {$ref: '#/components/schemas/store.Pet'}}
    store.Pet: {type: object, properties: {name: {type: string}}}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/PagedResponse%5Bexample.com%2Fpetstore%2Fpkg%2Fstore.Pet%5D'}
`,
	},
}

// versionedSpec pairs an OAS version label with the spec written for it.
type versionedSpec struct {
	name string
	spec string
}

// versions returns the case's two specs in a fixed order. A map would let Go's
// randomized iteration reorder the subtests on every run.
func (c slashCase) versions() []versionedSpec {
	return []versionedSpec{{"oas2", c.oas2Spec}, {"oas3", c.oas3Spec}}
}

// slashCaseNamed looks a case up by name so a test that needs one specific
// encoding does not break when the table is reordered or extended.
func slashCaseNamed(t *testing.T, name string) slashCase {
	t.Helper()

	for _, tc := range slashCases {
		if tc.name == name {
			return tc
		}
	}
	t.Fatalf("no slashCase named %q", name)
	return slashCase{}
}

// allNamingStrategies is every strategy --generic-naming accepts. Issue #404
// reproduced under all of them, so the regression is pinned across all of them.
var allNamingStrategies = []GenericNamingStrategy{
	GenericNamingUnderscore,
	GenericNamingOf,
	GenericNamingFor,
	GenericNamingFlattened,
	GenericNamingDot,
}

// fixSchemaNames renames invalid schema names in spec and returns the fix
// result, whose Document field carries the fixed document.
func fixSchemaNames(t *testing.T, spec string, strategy GenericNamingStrategy) *FixResult {
	t.Helper()

	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	return fixSchemaNamesParsed(t, parseResult, strategy)
}

// fixSchemaNamesParsed is [fixSchemaNames] for a document a test has already
// parsed, typically to assert its validity before the fix. parseResult is left
// untouched: FixParsed copies unless MutableInput is set.
func fixSchemaNamesParsed(t *testing.T, parseResult *parser.ParseResult, strategy GenericNamingStrategy) *FixResult {
	t.Helper()

	f := New()
	f.EnabledFixes = []FixType{FixTypeRenamedGenericSchema}
	f.GenericNamingConfig.Strategy = strategy
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	return result
}

// responseSchemaRef returns the $ref of the 200 response schema for GET /pets,
// for either OAS version.
func responseSchemaRef(t *testing.T, doc any) string {
	t.Helper()

	switch d := doc.(type) {
	case *parser.OAS2Document:
		resp := d.Paths["/pets"].Get.Responses.Codes["200"]
		require.NotNil(t, resp.Schema)
		return resp.Schema.Ref
	case *parser.OAS3Document:
		resp := d.Paths["/pets"].Get.Responses.Codes["200"]
		mt, ok := resp.Content["application/json"]
		require.True(t, ok, "expected an application/json response body")
		require.NotNil(t, mt.Schema)
		return mt.Schema.Ref
	default:
		t.Fatalf("unexpected document type %T", doc)
		return ""
	}
}

// schemaNames returns the definitions/components.schemas keys of either version.
func schemaNames(t *testing.T, doc any) map[string]*parser.Schema {
	t.Helper()

	switch d := doc.(type) {
	case *parser.OAS2Document:
		return d.Definitions
	case *parser.OAS3Document:
		require.NotNil(t, d.Components)
		return d.Components.Schemas
	default:
		t.Fatalf("unexpected document type %T", doc)
		return nil
	}
}

// TestFixSchemaNamesRoundTripsToValid is the regression for issue #404.
//
// The fixer reported a rename as applied while leaving every $ref to the
// renamed schema untouched, so its output failed validation with exactly the
// unresolved-$ref error the input had. Under OAS 3.x it also emitted a name
// containing "/", which the component-name charset rejects outright.
//
// The invariant asserted is the one the issue asks for: a document the fixer
// reports as fixed must validate. Checking validity rather than a literal name
// keeps the assertion meaningful across all five naming strategies, whose
// outputs differ.
func TestFixSchemaNamesRoundTripsToValid(t *testing.T) {
	for _, tc := range slashCases {
		t.Run(tc.name, func(t *testing.T) {
			for _, v := range tc.versions() {
				version, spec := v.name, v.spec
				t.Run(version, func(t *testing.T) {
					for _, strategy := range allNamingStrategies {
						t.Run(strategy.String(), func(t *testing.T) {
							// The input is invalid: the encoded $ref resolves to nothing.
							parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
							require.NoError(t, err)
							before, err := validator.New().ValidateParsed(*parseResult)
							require.NoError(t, err)
							require.False(t, before.Valid,
								"the unfixed document should fail validation, or this case proves nothing")

							result := fixSchemaNamesParsed(t, parseResult, strategy)
							require.True(t, result.HasFixes(), "the invalid schema name should be renamed")

							after, err := validator.New().ValidateParsed(*result.ToParseResult())
							require.NoError(t, err)
							assert.True(t, after.Valid,
								"a document reported as fixed must validate; errors: %v", after.Errors)

							// Validity alone would also hold if the fixer had reverted the
							// rename, so pin the actual outcome: the ref reaches a schema
							// that exists, under a name that no longer needs encoding.
							ref := responseSchemaRef(t, result.Document)
							names := schemaNames(t, result.Document)
							renamed := strings.TrimPrefix(strings.TrimPrefix(ref,
								pathutil.RefPrefixSchemas), pathutil.RefPrefixDefinitions)
							assert.Contains(t, names, renamed,
								"the rewritten ref must name an existing schema")
							assert.NotContains(t, names, tc.schemaName,
								"the old key must be gone, not merely joined by a new one")
							assert.NotContains(t, ref, "%",
								"the rewritten ref should need no percent-encoding")
							assert.NotContains(t, renamed, "~",
								"the rewritten ref should need no JSON Pointer escaping")
						})
					}
				})
			}
		})
	}
}

// TestFixSchemaNamesSlashProducesLegalComponentName covers Bug B of issue #404
// on its own terms.
//
// The OAS 3.x component-name charset is what rejected the old output, but a "/"
// is equally unwanted in an OAS 2.0 definitions key: it is legal there, yet
// forces every $ref reaching it to be escaped, which is the encoding trouble
// --fix-schema-names exists to remove. Both versions are therefore asserted.
func TestFixSchemaNamesSlashProducesLegalComponentName(t *testing.T) {
	for _, tc := range slashCases {
		if tc.control {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			for _, v := range tc.versions() {
				version, spec := v.name, v.spec
				t.Run(version, func(t *testing.T) {
					for _, strategy := range allNamingStrategies {
						t.Run(strategy.String(), func(t *testing.T) {
							result := fixSchemaNames(t, spec, strategy)

							for name := range schemaNames(t, result.Document) {
								assert.NotContains(t, name, "/",
									"a renamed schema must not carry a path separator")
							}
							for _, fix := range result.Fixes {
								after, ok := fix.After.(string)
								require.True(t, ok, "a rename fix should record the new name")
								assert.NotContains(t, after, "/",
									"the reported new name must not carry a path separator")
							}
						})
					}
				})
			}
		})
	}
}

// TestFixSchemaNamesSlashExactName pins the names from the issue's own
// reproduction, so the flattening is not merely legal but the intended shape.
func TestFixSchemaNamesSlashExactName(t *testing.T) {
	tests := []struct {
		strategy GenericNamingStrategy
		want     string
	}{
		{GenericNamingOf, "PagedResponseOfexample.com.petstore.pkg.store.Pet"},
		{GenericNamingFor, "PagedResponseForexample.com.petstore.pkg.store.Pet"},
		{GenericNamingFlattened, "PagedResponseexample.com.petstore.pkg.store.Pet"},
		{GenericNamingDot, "PagedResponse.example.com.petstore.pkg.store.Pet"},
		{GenericNamingUnderscore, "PagedResponse_example.com.petstore.pkg.store.Pet_"},
	}

	spec := slashCaseNamed(t, "percent-encoded brackets, raw slashes").oas3Spec

	for _, tt := range tests {
		t.Run(tt.strategy.String(), func(t *testing.T) {
			result := fixSchemaNames(t, spec, tt.strategy)

			assert.Contains(t, schemaNames(t, result.Document), tt.want)
			assert.Equal(t, pathutil.RefPrefixSchemas+tt.want,
				responseSchemaRef(t, result.Document))
		})
	}
}

// TestCanonicalSchemaRefKey covers the matching rule that replaced the fixed
// candidate-encoding list, including the mixed spelling no candidate covered.
func TestCanonicalSchemaRefKey(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want string
	}{
		{
			name: "plain name is unchanged",
			ref:  "#/components/schemas/Pet",
			want: "#/components/schemas/Pet",
		},
		{
			name: "percent-encoded brackets",
			ref:  "#/components/schemas/Response%5BUser%5D",
			want: "#/components/schemas/Response[User]",
		},
		{
			name: "JSON Pointer slash",
			ref:  "#/definitions/pet~1summary",
			want: "#/definitions/pet/summary",
		},
		{
			name: "JSON Pointer tilde",
			ref:  "#/definitions/pet~0summary",
			want: "#/definitions/pet~summary",
		},
		{
			name: "mixed: percent-encoded brackets with raw slashes",
			ref:  "#/definitions/Paged%5Bexample.com/pkg.Pet%5D",
			want: "#/definitions/Paged[example.com/pkg.Pet]",
		},
		{
			name: "mixed: percent-encoded brackets with pointer slashes",
			ref:  "#/definitions/Paged%5Bexample.com~1pkg.Pet%5D",
			want: "#/definitions/Paged[example.com/pkg.Pet]",
		},
		{
			name: "fully percent-encoded",
			ref:  "#/definitions/Paged%5Bexample.com%2Fpkg.Pet%5D",
			want: "#/definitions/Paged[example.com/pkg.Pet]",
		},
		{
			name: "invalid percent sequence is left alone rather than failing",
			ref:  "#/definitions/Discount%OFF",
			want: "#/definitions/Discount%OFF",
		},
		{
			name: "percent-encoded pointer separators reduce to the raw prefix",
			ref:  "#%2Fcomponents%2Fschemas%2FPaged%5BPet%5D",
			want: "#/components/schemas/Paged[Pet]",
		},
		{
			name: "an encoded pointer whose name is doubly encoded decodes only once",
			ref:  "#%2Fdefinitions%2FPaged%255BPet%255D",
			want: "#/definitions/Paged%5BPet%5D",
		},
		{
			name: "a non-schema ref is not a key and passes through",
			ref:  "#/components/parameters/Limit%5Bx%5D",
			want: "#/components/parameters/Limit%5Bx%5D",
		},
		{
			name: "an external ref passes through",
			ref:  "other.yaml#/components/schemas/Pet",
			want: "other.yaml#/components/schemas/Pet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, canonicalSchemaRefKey(tt.ref))
		})
	}
}

// TestFlattenPathQualifier covers the "/" replacement behind Bug B, including
// the degenerate separators that must not leave a stray dot behind.
func TestFlattenPathQualifier(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no slash is unchanged", "store.Pet", "store.Pet"},
		{"empty is unchanged", "", ""},
		{"package path", "example.com/petstore/pkg/store.Pet", "example.com.petstore.pkg.store.Pet"},
		{"single slash", "pkg/Pet", "pkg.Pet"},
		{"doubled slash drops the empty segment", "pkg//Pet", "pkg.Pet"},
		{"leading slash does not produce a leading dot", "/pkg.Pet", "pkg.Pet"},
		{"trailing slash does not produce a trailing dot", "pkg.Pet/", "pkg.Pet"},
		{"only slashes", "//", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, flattenPathQualifier(tt.input))
		})
	}
}

// TestTransformSchemaNameCleansBase covers the base half of the name, which
// carries whatever preceded the bracket.
//
// Only the type parameters were being cleaned, so any invalid character in the
// base survived into the new name and the fixer reported a fix that left the
// document as illegal as it found it.
func TestTransformSchemaNameCleansBase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"path-qualified base", "pkg/v1.Response[Pet]", "pkg.v1.ResponseOfPet"},
		{"space in base", "Res ponse[User]", "Res_ponseOfUser"},
		{"comma in base", "Res,ponse[User]", "Res_ponseOfUser"},
		{"braces in base", "Res{p}onse[User]", "Res_p_onseOfUser"},
		{"pipe in base", "a|b[User]", "a_bOfUser"},
		{"clean base is untouched", "Response[User]", "ResponseOfUser"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := transformSchemaName(tt.input, GenericNamingConfig{Strategy: GenericNamingOf}, charsetComponentName)
			assert.Equal(t, tt.want, got)
			assert.Regexp(t, naming.ComponentNamePattern, got,
				"a generated name must satisfy the OAS 3.x component-name charset")
		})
	}
}

// TestFixSchemaNamesRewritesDiscriminatorMapping covers the discriminator
// branch of the ref rewriting, for both the full-ref and bare-name spellings a
// mapping may use.
func TestFixSchemaNamesRewritesDiscriminatorMapping(t *testing.T) {
	spec := `
openapi: 3.0.3
info: {title: T, version: "1.0"}
components:
  schemas:
    Pet:
      type: object
      discriminator:
        propertyName: kind
        mapping:
          encoded: '#/components/schemas/Dog%5Bexample.com/pkg.Breed%5D'
          bare: 'Cat[example.com/pkg.Breed]'
          bareEncoded: 'Fish%5Bexample.com%2Fpkg.Breed%5D'
      properties:
        kind: {type: string}
    "Dog[example.com/pkg.Breed]": {type: object}
    "Cat[example.com/pkg.Breed]": {type: object}
    "Fish[example.com/pkg.Breed]": {type: object}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Pet'}
`
	result := fixSchemaNames(t, spec, GenericNamingOf)
	doc := result.Document.(*parser.OAS3Document)

	mapping := doc.Components.Schemas["Pet"].Discriminator.Mapping
	assert.Equal(t, pathutil.RefPrefixSchemas+"DogOfexample.com.pkg.Breed", mapping["encoded"],
		"a percent-encoded full ref must be rewritten")
	assert.Equal(t, "CatOfexample.com.pkg.Breed", mapping["bare"],
		"a bare name must be rewritten and stay bare")
	assert.Equal(t, "FishOfexample.com.pkg.Breed", mapping["bareEncoded"],
		"a bare name carrying an encoding must be rewritten too")
}

// TestFixSchemaNamesRewritesDiscriminatorDefaultMapping covers issue #397's
// reference-bearing field: discriminator.defaultMapping (OAS 3.2+) names a schema
// in exactly the two spellings a mapping value may use, so a rename that rewrote
// mapping but not defaultMapping would leave the fallback target dangling.
func TestFixSchemaNamesRewritesDiscriminatorDefaultMapping(t *testing.T) {
	spec := `
openapi: 3.2.0
info: {title: T, version: "1.0"}
components:
  schemas:
    Pet:
      type: object
      required: [kind]
      discriminator:
        propertyName: kind
        defaultMapping: '#/components/schemas/Other[example.com/pkg.Breed]'
        mapping:
          dog: '#/components/schemas/Dog[example.com/pkg.Breed]'
      properties:
        kind: {type: string}
    "Dog[example.com/pkg.Breed]": {type: object}
    "Other[example.com/pkg.Breed]": {type: object}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Pet'}
`
	result := fixSchemaNames(t, spec, GenericNamingOf)
	doc := result.Document.(*parser.OAS3Document)

	d := doc.Components.Schemas["Pet"].Discriminator
	assert.Equal(t, pathutil.RefPrefixSchemas+"DogOfexample.com.pkg.Breed", d.Mapping["dog"],
		"the mapping value must be rewritten")
	assert.Equal(t, pathutil.RefPrefixSchemas+"OtherOfexample.com.pkg.Breed", d.DefaultMapping,
		"defaultMapping must be rewritten alongside mapping")
}

// TestFixSchemaNamesRewritesBareDefaultMapping is the bare-name spelling of the
// same field: a defaultMapping may name its target without a pointer prefix, and
// must stay bare after the rewrite.
func TestFixSchemaNamesRewritesBareDefaultMapping(t *testing.T) {
	spec := `
openapi: 3.2.0
info: {title: T, version: "1.0"}
components:
  schemas:
    Pet:
      type: object
      required: [kind]
      discriminator:
        propertyName: kind
        defaultMapping: 'Other[example.com/pkg.Breed]'
      properties:
        kind: {type: string}
    "Other[example.com/pkg.Breed]": {type: object}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Pet'}
`
	result := fixSchemaNames(t, spec, GenericNamingOf)
	doc := result.Document.(*parser.OAS3Document)

	assert.Equal(t, "OtherOfexample.com.pkg.Breed",
		doc.Components.Schemas["Pet"].Discriminator.DefaultMapping,
		"a bare defaultMapping must be rewritten and stay bare")
}

// TestPruneKeepsDefaultMappingTarget covers the pruning side: a schema reachable
// only through discriminator.defaultMapping is referenced, and deleting it would
// dangle the fallback.
func TestPruneKeepsDefaultMappingTarget(t *testing.T) {
	spec := `
openapi: 3.2.0
info: {title: T, version: "1.0"}
components:
  schemas:
    Pet:
      type: object
      required: [kind]
      discriminator:
        propertyName: kind
        defaultMapping: '#/components/schemas/OtherPet'
      properties:
        kind: {type: string}
    OtherPet: {type: object}
    Orphan: {type: object}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Pet'}
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	doc := result.Document.(*parser.OAS3Document)
	assert.Contains(t, doc.Components.Schemas, "OtherPet",
		"a schema reachable only through defaultMapping must survive pruning")
	assert.NotContains(t, doc.Components.Schemas, "Orphan",
		"the genuinely unreferenced schema should still be pruned")
}

// TestFixSchemaNamesKeepsLiteralPercentName covers a name that genuinely
// contains a percent sequence, pinning that the rename map's keys and the
// lookup agree about decoding.
//
// Decoding is lossy — "Foo%20Bar[Pet]" decodes to "Foo Bar[Pet]" — so it is
// only safe when both sides do it. Keying on the undecoded name while looking
// up a decoded one (or the reverse) makes this ref match nothing, which is the
// unrewritten-$ref half of issue #404 reintroduced for these documents.
//
// Which schema an ambiguous spelling resolves to is pinned separately, by
// TestFixSchemaNamesDisambiguatesEncodedTwin.
func TestFixSchemaNamesKeepsLiteralPercentName(t *testing.T) {
	spec := `
openapi: 3.0.3
info: {title: T, version: "1.0"}
components:
  schemas:
    "Foo%20Bar[Pet]": {type: object}
    Pet: {type: object}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Foo%20Bar[Pet]'}
`
	result := fixSchemaNames(t, spec, GenericNamingOf)

	assert.Equal(t, pathutil.RefPrefixSchemas+"Foo_20BarOfPet",
		responseSchemaRef(t, result.Document),
		"the ref must be rewritten, not left dangling")

	after, err := validator.New().ValidateParsed(*result.ToParseResult())
	require.NoError(t, err)
	assert.True(t, after.Valid, "errors: %v", after.Errors)
}

// TestFixSchemaNamesDistinctNamesStayDistinct covers collision handling for
// names that only differ in a character the transform removes.
//
// "a/b.Pet" and "a.b.Pet" both reduce to "a.b.Pet", so without a claimed-name
// check the second rename is handed a name the first already took and one
// schema is silently overwritten — while both renames are still reported as
// applied.
func TestFixSchemaNamesDistinctNamesStayDistinct(t *testing.T) {
	spec := `
openapi: 3.0.3
info: {title: T, version: "1.0"}
components:
  schemas:
    "Resp[a/b.Pet]": {type: object, properties: {x: {type: string}}}
    "Resp[a.b.Pet]": {type: object, properties: {y: {type: integer}}}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Resp%5Ba/b.Pet%5D'}
`
	result := fixSchemaNames(t, spec, GenericNamingOf)
	names := schemaNames(t, result.Document)

	assert.Len(t, names, 2, "neither schema may be overwritten by the other")
	assert.Len(t, result.Fixes, 2, "both renames should be reported")

	after, err := validator.New().ValidateParsed(*result.ToParseResult())
	require.NoError(t, err)
	assert.True(t, after.Valid, "errors: %v", after.Errors)
}

// TestFixSchemaNamesCollisionIsDeterministic pins which schema keeps the
// unsuffixed name when two reduce to the same one. Building the rename map by
// ranging over the schemas map would hand the suffix to a different schema on
// each run.
func TestFixSchemaNamesCollisionIsDeterministic(t *testing.T) {
	spec := `
openapi: 3.0.3
info: {title: T, version: "1.0"}
components:
  schemas:
    "Resp[a/b.Pet]": {type: object}
    "Resp[a.b.Pet]": {type: object}
    "Resp[a//b.Pet]": {type: object}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200": {description: OK}
`
	// Compare the old->new pairs, not the set of new names. All three reduce to
	// the same base, so the resulting names are always {X, X2, X3} whichever
	// schema gets which — only the pairing reveals an ordering flip.
	var first []string
	for range 12 {
		result := fixSchemaNames(t, spec, GenericNamingOf)

		got := make([]string, 0, len(result.Fixes))
		for _, fix := range result.Fixes {
			got = append(got, fmt.Sprintf("%v->%v", fix.Before, fix.After))
		}
		slices.Sort(got)

		if first == nil {
			first = got
			continue
		}
		require.Equal(t, first, got, "collision suffixes must not depend on map iteration order")
	}

	// The unsuffixed name goes to the first candidate in byte order ("." is 0x2E,
	// "/" is 0x2F), and the suffixes follow from there.
	assert.Equal(t, []string{
		"Resp[a.b.Pet]->RespOfa.b.Pet",
		"Resp[a//b.Pet]->RespOfa.b.Pet2",
		"Resp[a/b.Pet]->RespOfa.b.Pet3",
	}, first, "all three schemas must survive, each with a stable name")
}

// TestPruneKeepsMixedEncodingRefSchema covers the pruning half of the same
// encoding problem: a referenced schema whose $ref mixes the two conventions
// was recovered under a name no key matched, so pruning deleted it.
func TestPruneKeepsMixedEncodingRefSchema(t *testing.T) {
	spec := `
swagger: "2.0"
info: {title: T, version: "1.0"}
definitions:
  "Paged[example.com/pkg.Pet]": {type: object}
paths:
  /p:
    get:
      operationId: g
      responses:
        "200":
          description: OK
          schema: {$ref: '#/definitions/Paged%5Bexample.com/pkg.Pet%5D'}
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	doc := result.Document.(*parser.OAS2Document)
	assert.Contains(t, doc.Definitions, "Paged[example.com/pkg.Pet]",
		"a referenced schema must survive pruning whatever encoding its ref uses")
	assert.Zero(t, result.FixCount, "nothing should have been pruned")
}

// TestFixSchemaNamesRewritesEncodedPointerRef is the regression for issue #408.
//
// A generator that runs a URL escaper over a whole pointer rather than over each
// token emits "#%2Fcomponents%2Fschemas%2F…". Pruning already recognized that
// shape and kept the schema, but the rewrite did not, so a rename left the ref
// pointing at a name no longer present — a dangling ref in output the fixer
// reported as successfully fixed.
func TestFixSchemaNamesRewritesEncodedPointerRef(t *testing.T) {
	for _, tt := range []struct {
		version string
		spec    string
	}{
		{
			version: "oas3",
			spec: `
openapi: 3.0.3
info: {title: T, version: "1.0"}
components:
  schemas:
    "Paged[Pet]": {type: object}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#%2Fcomponents%2Fschemas%2FPaged%5BPet%5D'}
`,
		},
		{
			version: "oas2",
			spec: `
swagger: "2.0"
info: {title: T, version: "1.0"}
definitions:
  "Paged[Pet]": {type: object}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          schema: {$ref: '#%2Fdefinitions%2FPaged%5BPet%5D'}
`,
		},
	} {
		t.Run(tt.version, func(t *testing.T) {
			result := fixSchemaNames(t, tt.spec, GenericNamingOf)

			names := schemaNames(t, result.Document)
			require.Contains(t, names, "PagedOfPet", "the schema should have been renamed")

			assert.NotContains(t, responseSchemaRef(t, result.Document), "Paged%5BPet%5D",
				"the ref must not still name the schema by its old name")
			assert.Contains(t, responseSchemaRef(t, result.Document), "PagedOfPet",
				"a ref whose pointer separators are encoded must be rewritten too")
		})
	}
}

// TestPruneKeepsEncodedPointerRefSchema pins the pruning half of #408. It passed
// before the fix — appendSchemaNames already decoded the prefix — and must keep
// passing now that it shares pathutil.CutRefPrefix with the rewrite, since a
// prune that stops recognizing the ref would delete a referenced schema.
func TestPruneKeepsEncodedPointerRefSchema(t *testing.T) {
	spec := `
swagger: "2.0"
info: {title: T, version: "1.0"}
definitions:
  "Paged[Pet]": {type: object}
paths:
  /p:
    get:
      operationId: g
      responses:
        "200":
          description: OK
          schema: {$ref: '#%2Fdefinitions%2FPaged%5BPet%5D'}
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	doc := result.Document.(*parser.OAS2Document)
	assert.Contains(t, doc.Definitions, "Paged[Pet]")
	assert.Zero(t, result.FixCount, "nothing should have been pruned")
}

// TestPruneStillRemovesUnreferencedMixedEncodingSchema is the control: matching
// more spellings must not make every schema look referenced.
func TestPruneStillRemovesUnreferencedMixedEncodingSchema(t *testing.T) {
	spec := `
swagger: "2.0"
info: {title: T, version: "1.0"}
definitions:
  "Paged[example.com/pkg.Pet]": {type: object}
  "Paged[example.com/pkg.Orphan]": {type: object}
paths:
  /p:
    get:
      operationId: g
      responses:
        "200":
          description: OK
          schema: {$ref: '#/definitions/Paged%5Bexample.com/pkg.Pet%5D'}
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	doc := result.Document.(*parser.OAS2Document)
	assert.Contains(t, doc.Definitions, "Paged[example.com/pkg.Pet]")
	assert.NotContains(t, doc.Definitions, "Paged[example.com/pkg.Orphan]",
		"the unreferenced schema should still be pruned")
}

// TestFixSchemaNamesDisambiguatesEncodedTwin covers two schemas whose names
// differ only by percent-encoding, which is what the exact-spelling key in
// buildRefRenameMap exists for.
//
// "Foo%20Bar[Pet]" and "Foo Bar[Pet]" share a decoded form, so registering only
// the decoded key would collapse them and send one schema's refs to the other's
// new name. Registering the exact spelling first keeps each ref pointing at the
// schema it actually named.
func TestFixSchemaNamesDisambiguatesEncodedTwin(t *testing.T) {
	spec := `
openapi: 3.0.3
info: {title: T, version: "1.0"}
components:
  schemas:
    "Foo%20Bar[Pet]": {type: object}
    "Foo Bar[Pet]": {type: object}
    Pet: {type: object}
paths:
  /encoded:
    get:
      operationId: getEncoded
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Foo%20Bar[Pet]'}
  /literal:
    get:
      operationId: getLiteral
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Foo Bar[Pet]'}
`
	result := fixSchemaNames(t, spec, GenericNamingOf)
	doc := result.Document.(*parser.OAS3Document)

	refFor := func(path string) string {
		return doc.Paths[path].Get.Responses.Codes["200"].Content["application/json"].Schema.Ref
	}

	assert.Equal(t, pathutil.RefPrefixSchemas+"Foo_20BarOfPet", refFor("/encoded"),
		"the percent-spelled ref must follow the schema it named")
	assert.Equal(t, pathutil.RefPrefixSchemas+"Foo_BarOfPet", refFor("/literal"),
		"the literal-space ref must follow the schema it named")

	after, err := validator.New().ValidateParsed(*result.ToParseResult())
	require.NoError(t, err)
	assert.True(t, after.Valid, "errors: %v", after.Errors)
}

// TestBuildRefRenameMapDecodedKeyIsDeterministic covers two names that share a
// decoded form without either being it: "A%2FB[Pet]" and "A~1B[Pet]" both reduce
// to "A/B[Pet]", and only one can claim that key.
//
// Ranging over the rename map to register decoded keys handed it to a different
// name on each run, so a $ref spelled "#/components/schemas/A/B[Pet]" resolved
// to a different schema run to run.
func TestBuildRefRenameMapDecodedKeyIsDeterministic(t *testing.T) {
	spec := `
openapi: 3.0.3
info: {title: T, version: "1.0"}
components:
  schemas:
    "A%2FB[Pet]": {type: object}
    "A~1B[Pet]": {type: object}
    Pet: {type: object}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/A/B[Pet]'}
`
	seen := make(map[string]bool)
	for range 40 {
		result := fixSchemaNames(t, spec, GenericNamingOf)
		seen[responseSchemaRef(t, result.Document)] = true
	}

	assert.Equal(t, map[string]bool{pathutil.RefPrefixSchemas + "A_2FBOfPet": true}, seen,
		"an ambiguous ref must resolve to the same schema on every run")
}

// TestPruneKeepsBareDiscriminatorTargets covers the bare spelling a discriminator
// may use for its target.
//
// The spec recommends treating a mapping or defaultMapping value that is a valid
// schema name as a schema name, so "Dog" and "#/components/schemas/Dog" name the
// same schema. Pruning recognized only the pointer form, so a schema referenced
// only by a bare mapping value was deleted from a legal document.
func TestPruneKeepsBareDiscriminatorTargets(t *testing.T) {
	spec := `
openapi: 3.2.0
info: {title: T, version: "1.0"}
components:
  schemas:
    Pet:
      type: object
      required: [kind]
      properties: {kind: {type: string}}
      discriminator:
        propertyName: kind
        defaultMapping: OtherPet
        mapping: {dog: Dog}
    Dog: {type: object}
    OtherPet: {type: object}
    Orphan: {type: object}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Pet'}
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	doc := result.Document.(*parser.OAS3Document)
	assert.Contains(t, doc.Components.Schemas, "Dog",
		"a bare mapping value names a schema and must keep it alive")
	assert.Contains(t, doc.Components.Schemas, "OtherPet",
		"a bare defaultMapping value must keep its schema alive too")
	assert.NotContains(t, doc.Components.Schemas, "Orphan",
		"matching bare names must not make every schema look referenced")
}

// TestPruneKeepsBareDiscriminatorTargetsWithSlash covers the shape this repo's own
// generic-name fixer produces: a schema name containing "/".
//
// Treating "/" as evidence that a value was a pointer rather than a name deleted
// both targets here. Only "#" marks a pointer; OAS 2.0 places no charset
// constraint on definition keys, and "Dog[example.com/pkg.Breed]" is a name the
// fixer emits and the discriminator tests already use as a bare mapping value.
func TestPruneKeepsBareDiscriminatorTargetsWithSlash(t *testing.T) {
	spec := `
openapi: 3.2.0
info: {title: T, version: "1.0"}
components:
  schemas:
    Pet:
      type: object
      required: [kind]
      properties: {kind: {type: string}}
      discriminator:
        propertyName: kind
        defaultMapping: 'Other[example.com/pkg.Breed]'
        mapping:
          dog: 'Dog[example.com/pkg.Breed]'
    "Dog[example.com/pkg.Breed]": {type: object}
    "Other[example.com/pkg.Breed]": {type: object}
    "Orphan[example.com/pkg.Breed]": {type: object}
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Pet'}
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	doc := result.Document.(*parser.OAS3Document)
	assert.Contains(t, doc.Components.Schemas, "Dog[example.com/pkg.Breed]",
		"a bare mapping value containing \"/\" must keep its schema alive")
	assert.Contains(t, doc.Components.Schemas, "Other[example.com/pkg.Breed]",
		"a bare defaultMapping value containing \"/\" must too")
	assert.NotContains(t, doc.Components.Schemas, "Orphan[example.com/pkg.Breed]",
		"an unreferenced schema with the same shape is still pruned")
}
