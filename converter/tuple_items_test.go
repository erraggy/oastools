// tuple_items_test.go covers conversion of the OAS 2.0 tuple form of `items`.
package converter

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const tupleItemsOAS2Spec = `
swagger: "2.0"
info:
  title: petstore
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: successful operation
          schema:
            $ref: "#/definitions/PetTuple"
definitions:
  PetTuple:
    type: array
    items:
      - type: string
      - $ref: "#/definitions/PetDetails"
  PetDetails:
    type: object
    properties:
      name:
        type: string
`

// TestTupleItemsRefIsRewrittenOnUpconversion covers #502 from the converter's
// side: OAS 2.0 spells the pool "#/definitions", OAS 3.x spells it
// "#/components/schemas", and a $ref the rewrite never visits keeps the old
// prefix and points at nothing in the converted document.
//
// The target is 3.1, which is where a tuple survives conversion and so where a
// $ref inside one still has to resolve. OAS 3.0 forbids tuples outright and
// drops the positions, leaving no reference to check: see
// TestTupleConversionByTargetVersion.
func TestTupleItemsRefIsRewrittenOnUpconversion(t *testing.T) {
	parseResult, err := parser.New().ParseBytes([]byte(tupleItemsOAS2Spec))
	require.NoError(t, err)

	result, err := ConvertWithOptions(
		WithParsed(*parseResult),
		WithTargetVersion("3.1.0"),
	)
	require.NoError(t, err)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected an OAS 3 document, got %T", result.Document)

	tuple := doc.Components.Schemas["PetTuple"]
	require.NotNil(t, tuple)

	refs := collectRefsUnderItems(t, tuple)
	assert.Contains(t, refs, "#/components/schemas/PetDetails",
		"a $ref reachable through items kept the OAS 2.0 prefix and now points at nothing")
	assert.NotContains(t, refs, "#/definitions/PetDetails")
}

// collectRefsUnderItems gathers the $ref values reachable through a schema's
// items, whichever shape the field holds, so it keeps checking the reference
// once the tuple's own conversion is settled.
func collectRefsUnderItems(t *testing.T, schema *parser.Schema) []string {
	t.Helper()

	var refs []string
	for _, s := range schemautil.SchemaOrBoolSchemas(schema.Items) {
		refs = append(refs, s.Ref)
	}
	for _, s := range schema.PrefixItems {
		refs = append(refs, s.Ref)
	}
	if len(refs) == 0 {
		t.Fatalf("neither items nor prefixItems held a schema to check, items was %T", schema.Items)
	}

	return refs
}

// TestSchemaOrBoolFieldsAreConvertedForOAS31 covers the fields the OAS 3.1
// exclusiveMinimum/exclusiveMaximum conversion gained alongside the tuple form.
// OAS 2.0 spells those keywords as booleans paired with minimum/maximum, 3.1 as
// numbers, so a field the conversion skips keeps the old spelling and is invalid
// in the converted document.
func TestSchemaOrBoolFieldsAreConvertedForOAS31(t *testing.T) {
	const spec = `
swagger: "2.0"
info:
  title: t
  version: "1.0.0"
paths: {}
definitions:
  Holder:
    type: array
    items:
      - type: number
        minimum: 5
        exclusiveMinimum: true
    additionalItems:
      type: number
      minimum: 5
      exclusiveMinimum: true
    additionalProperties:
      type: number
      minimum: 5
      exclusiveMinimum: true
    unevaluatedItems:
      type: number
      minimum: 5
      exclusiveMinimum: true
    unevaluatedProperties:
      type: number
      minimum: 5
      exclusiveMinimum: true
`
	parsed, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)
	parseResult := *parsed

	result, err := ConvertWithOptions(WithParsed(parseResult), WithTargetVersion("3.1.0"))
	require.NoError(t, err)

	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok)
	converted := doc.Components.Schemas["Holder"]

	check := func(name string, field any) {
		t.Helper()
		var s *parser.Schema
		switch v := field.(type) {
		case *parser.Schema:
			s = v
		case []*parser.Schema:
			require.NotEmpty(t, v, name)
			s = v[0]
		default:
			t.Fatalf("%s held no schema, got %T", name, field)
		}
		assert.Equal(t, 5.0, s.ExclusiveMinimum,
			"%s kept the OAS 3.0 boolean spelling instead of the 3.1 numeric one", name)
		assert.Nil(t, s.Minimum, "%s should have moved minimum into exclusiveMinimum", name)
	}

	// The two array fields are checked under their OAS 3.1 names: the tuple is
	// spelled prefixItems there, and items takes over the trailing-element role
	// draft 4 gave additionalItems.
	check("prefixItems", converted.PrefixItems)
	check("items", converted.Items)
	check("additionalProperties", converted.AdditionalProperties)
	check("unevaluatedItems", converted.UnevaluatedItems)
	check("unevaluatedProperties", converted.UnevaluatedProperties)

	assert.Nil(t, converted.AdditionalItems,
		"additionalItems is the draft 4 spelling and has no role in an OAS 3.1 document")
}

// TestSchemaOrBoolFieldsAreWalkedOnDowngrade covers the fields the feature walk
// gained alongside the tuple form. The walk reports OAS 3.x features that cannot
// be expressed in OAS 2.0, so a field it skips downgrades silently.
func TestSchemaOrBoolFieldsAreWalkedOnDowngrade(t *testing.T) {
	const spec = `
openapi: 3.1.0
info:
  title: t
  version: "1.0.0"
paths: {}
components:
  schemas:
    Holder:
      type: array
      additionalItems:
        type: object
        deprecated: true
      unevaluatedItems:
        type: object
        deprecated: true
      unevaluatedProperties:
        type: object
        deprecated: true
`
	parsed, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion("2.0"))
	require.NoError(t, err)

	paths := make([]string, 0, len(result.Issues))
	for _, issue := range result.Issues {
		paths = append(paths, issue.Path)
	}
	joined := strings.Join(paths, "\n")

	for _, field := range []string{"additionalItems", "unevaluatedItems", "unevaluatedProperties"} {
		assert.Contains(t, joined, "."+field,
			"the deprecated schema under %s was not reached; issue paths:\n%s", field, joined)
	}
}

// TestCheckSchemaNullableReachesTupleAdditionalProperties covers the tuple form
// in the nullable check, which reports the OAS 3.1 deprecation of 'nullable'.
func TestCheckSchemaNullableReachesTupleAdditionalProperties(t *testing.T) {
	const spec = `
openapi: 3.0.3
info:
  title: t
  version: "1.0.0"
paths: {}
components:
  schemas:
    Holder:
      type: object
      additionalProperties:
        - type: string
          nullable: true
`
	parsed, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion("3.1.0"))
	require.NoError(t, err)

	paths := make([]string, 0, len(result.Issues))
	for _, issue := range result.Issues {
		paths = append(paths, issue.Path)
	}
	assert.Contains(t, strings.Join(paths, "\n"), "additionalProperties[0]",
		"the nullable schema in a tuple additionalProperties element was not reached")
}

// TestTupleConversionByTargetVersion pins what happens to an OAS 2.0 tuple at
// each target, which differs because the three versions disagree about tuples.
// OAS 2.0 takes items from JSON Schema draft 4, where an array of schemas
// constrains each position. OAS 3.0 forbids that outright: "Value MUST be an
// object and not an array". OAS 3.1 and later inherit 2020-12, which moves the
// positions to prefixItems. So the tuple converts at 3.1+, and at 3.0 it either
// collapses or is dropped and reported (#508).
func TestTupleConversionByTargetVersion(t *testing.T) {
	const heterogeneous = `swagger: "2.0"
info:
  title: t
  version: "1.0.0"
paths: {}
definitions:
  Row:
    type: array
    items:
      - type: string
      - type: integer
`
	const uniformBounded = `swagger: "2.0"
info:
  title: t
  version: "1.0.0"
paths: {}
definitions:
  Row:
    type: array
    items:
      - type: number
      - type: number
    maxItems: 2
`
	const uniformOpen = `swagger: "2.0"
info:
  title: t
  version: "1.0.0"
paths: {}
definitions:
  Row:
    type: array
    items:
      - type: number
      - type: number
`
	const uniformBools = `swagger: "2.0"
info:
  title: t
  version: "1.0.0"
paths: {}
definitions:
  Row:
    type: array
    items:
      - true
      - true
    maxItems: 2
`

	tests := []struct {
		name        string
		spec        string
		target      string
		wantWarning bool
		assert      func(t *testing.T, s *parser.Schema)
	}{
		{
			name:   "3.1 keeps the positions as prefixItems",
			spec:   heterogeneous,
			target: "3.1.0",
			assert: func(t *testing.T, s *parser.Schema) {
				require.Len(t, s.PrefixItems, 2)
				assert.Equal(t, "string", s.PrefixItems[0].Type)
				assert.Equal(t, "integer", s.PrefixItems[1].Type)
				assert.Nil(t, s.Items, "nothing constrains what follows, as in the source")
			},
		},
		{
			name:   "3.2 keeps the positions as prefixItems",
			spec:   heterogeneous,
			target: "3.2.0",
			assert: func(t *testing.T, s *parser.Schema) {
				require.Len(t, s.PrefixItems, 2)
				assert.Equal(t, "string", s.PrefixItems[0].Type)
			},
		},
		{
			name:        "3.0 drops positions it cannot express, and says so",
			spec:        heterogeneous,
			target:      "3.0.3",
			wantWarning: true,
			assert: func(t *testing.T, s *parser.Schema) {
				empty, ok := s.Items.(*parser.Schema)
				require.True(t, ok, "OAS 3.0 requires items when type is array, got %T", s.Items)
				assert.Equal(t, &parser.Schema{}, empty, "the empty schema accepts everything the tuple accepted")
				assert.Empty(t, s.PrefixItems, "prefixItems is not valid in OAS 3.0 either")
			},
		},
		{
			name:   "3.0 collapses a uniform tuple that cannot grow",
			spec:   uniformBounded,
			target: "3.0.3",
			assert: func(t *testing.T, s *parser.Schema) {
				single, ok := s.Items.(*parser.Schema)
				require.True(t, ok, "expected a single schema, got %T", s.Items)
				assert.Equal(t, "number", single.Type)
			},
		},
		{
			name:        "3.0 will not collapse a uniform tuple that can grow",
			spec:        uniformOpen,
			target:      "3.0.3",
			wantWarning: true,
			assert: func(t *testing.T, s *parser.Schema) {
				// draft 4 lets anything follow a tuple when additionalItems is
				// absent, so `items: {type: number}` would forbid arrays the
				// source allows.
				empty, ok := s.Items.(*parser.Schema)
				require.True(t, ok)
				assert.Equal(t, &parser.Schema{}, empty)
			},
		},
		{
			name:        "3.0 will not collapse to a boolean schema",
			spec:        uniformBools,
			target:      "3.0.3",
			wantWarning: true,
			assert: func(t *testing.T, s *parser.Schema) {
				// OAS 3.0 has no bare-boolean schema form, so collapsing here
				// would trade an invalid tuple for an invalid items.
				empty, ok := s.Items.(*parser.Schema)
				require.True(t, ok)
				assert.Equal(t, &parser.Schema{}, empty)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.New().ParseBytes([]byte(tt.spec))
			require.NoError(t, err)

			result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion(tt.target))
			require.NoError(t, err)

			doc, ok := result.Document.(*parser.OAS3Document)
			require.True(t, ok)
			tt.assert(t, doc.Components.Schemas["Row"])

			var tupleWarnings int
			for _, issue := range result.Issues {
				if issue.Severity == SeverityWarning && strings.Contains(issue.Message, "tuple") {
					tupleWarnings++
				}
			}
			if tt.wantWarning {
				assert.Equal(t, 1, tupleWarnings, "expected exactly one warning naming the tuple")
			} else {
				assert.Zero(t, tupleWarnings, "a tuple that came across is not a conversion issue")
			}
		})
	}
}

// TestTupleSurvivesRoundTrip is the counterpart to the per-target assertions.
// Each direction can look right on its own while losing something the other
// direction would reveal, and 2.0 and 3.1 can both express a tuple, so a round
// trip through 3.1 has to come back unchanged.
func TestTupleSurvivesRoundTrip(t *testing.T) {
	const spec = `swagger: "2.0"
info:
  title: t
  version: "1.0.0"
paths: {}
definitions:
  Row:
    type: array
    items:
      - type: string
      - type: integer
    additionalItems: false
`
	parsed, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)
	original := parsed.Document.(*parser.OAS2Document).Definitions["Row"].DeepCopy()

	up, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion("3.1.0"))
	require.NoError(t, err)

	upDoc := up.Document.(*parser.OAS3Document)
	require.Len(t, upDoc.Components.Schemas["Row"].PrefixItems, 2)
	assert.Equal(t, false, upDoc.Components.Schemas["Row"].Items,
		"additionalItems: false becomes items: false in 2020-12")

	upParsed := parser.ParseResult{Document: up.Document, OASVersion: parser.OASVersion310, Version: "3.1.0"}
	down, err := ConvertWithOptions(WithParsed(upParsed), WithTargetVersion("2.0"))
	require.NoError(t, err)

	returned := down.Document.(*parser.OAS2Document).Definitions["Row"]
	assert.True(t, original.Equals(returned),
		"a tuple both versions can express must survive 2.0 to 3.1 and back")
}

// TestBoolTuplePositionDownconversion covers a bool at a tuple POSITION going to
// OAS 2.0. Draft 4 has no boolean schema form, so a bool cannot stay where it
// is: `true` and the empty schema both accept anything, so that one converts,
// while `false` accepts nothing and draft 4 cannot say so without `not`, which
// OAS 2.0 does not have.
//
// A bool in `items` is a different matter and stays a bool, because it lands in
// `additionalItems`, where draft 4 does accept one.
func TestBoolTuplePositionDownconversion(t *testing.T) {
	const spec = `openapi: 3.1.0
info:
  title: t
  version: "1.0.0"
paths: {}
components:
  schemas:
    Permissive:
      type: array
      prefixItems:
        - true
        - type: string
      items: false
    Restrictive:
      type: array
      prefixItems:
        - false
        - type: string
`
	parsed, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion("2.0"))
	require.NoError(t, err)

	doc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok)

	tupleOf := func(name string) []*parser.Schema {
		t.Helper()
		tuple, ok := doc.Definitions[name].Items.([]*parser.Schema)
		require.True(t, ok, "%s: expected the tuple form, got %T", name, doc.Definitions[name].Items)
		return tuple
	}

	permissive := tupleOf("Permissive")
	require.Len(t, permissive, 2)
	assert.Equal(t, &parser.Schema{}, permissive[0], "`true` says what the empty schema says")
	_, isBool := permissive[0].IsBool()
	assert.False(t, isBool, "a boolean schema is not valid in an OAS 2.0 document")
	assert.Equal(t, false, doc.Definitions["Permissive"].AdditionalItems,
		"a bool is legal in draft 4 additionalItems, so items keeps its value there")

	restrictive := tupleOf("Restrictive")
	require.Len(t, restrictive, 2)
	assert.Equal(t, &parser.Schema{}, restrictive[0])

	var reported int
	for _, issue := range result.Issues {
		if strings.Contains(issue.Message, "boolean schema 'false' at a tuple position") {
			reported++
			assert.Equal(t, SeverityWarning, issue.Severity)
		}
	}
	assert.Equal(t, 1, reported,
		"only `false` loses meaning, so only `false` is reported")
}

// TestAdditionalItemsDoesNotSurviveConversion covers the branch the tuple cases
// do not reach: a schema whose items is a single schema rather than a tuple.
// draft 4 ignores additionalItems unless items is an array, so the field says
// nothing there, and no OAS 3.x version has the keyword to carry it: it appears
// nowhere in the 3.0.4 or 3.1.1 specifications.
func TestAdditionalItemsDoesNotSurviveConversion(t *testing.T) {
	const spec = `swagger: "2.0"
info:
  title: t
  version: "1.0.0"
paths: {}
definitions:
  ObjectItems:
    type: array
    items:
      type: string
    additionalItems: false
  NoItems:
    type: object
    additionalItems:
      type: string
`
	for _, target := range []string{"3.0.3", "3.1.0", "3.2.0"} {
		t.Run(target, func(t *testing.T) {
			parsed, err := parser.New().ParseBytes([]byte(spec))
			require.NoError(t, err)

			result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion(target))
			require.NoError(t, err)

			doc, ok := result.Document.(*parser.OAS3Document)
			require.True(t, ok)

			objectItems := doc.Components.Schemas["ObjectItems"]
			assert.Nil(t, objectItems.AdditionalItems,
				"additionalItems has no meaning beside a single-schema items and no spelling in OAS 3.x")
			single, ok := objectItems.Items.(*parser.Schema)
			require.True(t, ok, "the single-schema items is untouched, got %T", objectItems.Items)
			assert.Equal(t, "string", single.Type)

			assert.Nil(t, doc.Components.Schemas["NoItems"].AdditionalItems,
				"a schema with no items at all cannot carry additionalItems either")
		})
	}
}

// TestArrayValuedSchemaOrBoolIsNotCarriedAcross covers a source that is
// malformed but parseable: an array in a field that takes a schema or a bool.
// draft 4 takes neither an array in `additionalItems`, nor does 2020-12 take
// one in `items`, so neither can be moved into the other's slot. Carrying one
// across would put an array back into exactly the field this conversion exists
// to clear.
func TestArrayValuedSchemaOrBoolIsNotCarriedAcross(t *testing.T) {
	t.Run("2.0 additionalItems array does not become 3.1 items", func(t *testing.T) {
		const spec = `swagger: "2.0"
info:
  title: t
  version: "1.0.0"
paths: {}
definitions:
  Weird:
    type: array
    items:
      - type: string
    additionalItems:
      - type: integer
      - type: boolean
`
		parsed, err := parser.New().ParseBytes([]byte(spec))
		require.NoError(t, err)

		result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion("3.1.0"))
		require.NoError(t, err)

		schema := result.Document.(*parser.OAS3Document).Components.Schemas["Weird"]
		require.Len(t, schema.PrefixItems, 1, "the legal tuple still converts")
		assert.Nil(t, schema.Items, "an array cannot be 2020-12 items")
		assert.Nil(t, schema.AdditionalItems, "and the keyword does not survive into OAS 3.x")

		assertReports(t, result, "additionalItems")
	})

	t.Run("3.1 items array does not become 2.0 additionalItems", func(t *testing.T) {
		const spec = `openapi: 3.1.0
info:
  title: t
  version: "1.0.0"
paths: {}
components:
  schemas:
    Weird:
      type: array
      prefixItems:
        - type: string
      items:
        - type: integer
        - type: boolean
`
		parsed, err := parser.New().ParseBytes([]byte(spec))
		require.NoError(t, err)

		result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion("2.0"))
		require.NoError(t, err)

		schema := result.Document.(*parser.OAS2Document).Definitions["Weird"]
		tuple, ok := schema.Items.([]*parser.Schema)
		require.True(t, ok, "the legal tuple still converts, got %T", schema.Items)
		require.Len(t, tuple, 1)
		assert.Nil(t, schema.AdditionalItems, "an array cannot be draft 4 additionalItems")

		assertReports(t, result, "items")
	})
}

// assertReports fails unless exactly one warning names the field, so a silent
// drop and a doubled report are both caught.
func assertReports(t *testing.T, result *ConversionResult, field string) {
	t.Helper()

	var matched int
	for _, issue := range result.Issues {
		if issue.Severity == SeverityWarning && strings.Contains(issue.Message, "which no OAS version accepts there") &&
			strings.Contains(issue.Message, "'"+field+"'") {
			matched++
		}
	}
	assert.Equal(t, 1, matched, "expected exactly one warning naming %q; issues: %+v", field, result.Issues)
}

// TestEarlyReturnShapesAreDeliberate pins the two shapes that reach a
// conversion's early return, because both look like oversights and neither is.
// The rule they follow is the one the reporting cases follow: put each thing
// where the target can hold it, and report only what has nowhere to go.
func TestEarlyReturnShapesAreDeliberate(t *testing.T) {
	t.Run("an inert additionalItems is dropped without a report", func(t *testing.T) {
		// draft 4 ignores additionalItems unless items is an array, so beside a
		// single-schema items it constrains nothing. Dropping it loses nothing,
		// even when it holds an array, so there is no loss to announce. Contrast
		// TestArrayValuedSchemaOrBoolIsNotCarriedAcross, where the same value is
		// reported because a tuple beside it makes the field live.
		const spec = `swagger: "2.0"
info:
  title: t
  version: "1.0.0"
paths: {}
definitions:
  A:
    type: array
    items:
      type: string
    additionalItems:
      - type: integer
      - type: boolean
`
		parsed, err := parser.New().ParseBytes([]byte(spec))
		require.NoError(t, err)

		result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion("3.1.0"))
		require.NoError(t, err)

		schema := result.Document.(*parser.OAS3Document).Components.Schemas["A"]
		assert.Nil(t, schema.AdditionalItems)
		single, ok := schema.Items.(*parser.Schema)
		require.True(t, ok, "the single-schema items is untouched, got %T", schema.Items)
		assert.Equal(t, "string", single.Type)

		for _, issue := range result.Issues {
			assert.NotContains(t, issue.Message, "additionalItems",
				"dropping a field that constrained nothing is not a conversion issue")
		}
	})

	t.Run("an array items with no prefixItems converts rather than dropping", func(t *testing.T) {
		// OAS 2.0 spells a tuple exactly this way, so the array has somewhere to
		// go and the conversion keeps it. Dropping it would discard the only
		// thing the source said.
		const spec = `openapi: 3.1.0
info:
  title: t
  version: "1.0.0"
paths: {}
components:
  schemas:
    B:
      type: array
      items:
        - type: integer
        - type: boolean
`
		parsed, err := parser.New().ParseBytes([]byte(spec))
		require.NoError(t, err)

		result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion("2.0"))
		require.NoError(t, err)

		schema := result.Document.(*parser.OAS2Document).Definitions["B"]
		tuple, ok := schema.Items.([]*parser.Schema)
		require.True(t, ok, "expected the tuple to survive, got %T", schema.Items)
		require.Len(t, tuple, 2)
		assert.Equal(t, "integer", tuple[0].Type)
		assert.Equal(t, "boolean", tuple[1].Type)
		assert.Nil(t, schema.AdditionalItems)
	})
}

// TestUniformCollapseKeepsTheLengthBound covers what a collapse must carry
// besides the element schema. `additionalItems: false` caps a draft 4 tuple at
// its own length, and collapsing to a single `items` keeps only what each
// element must look like, so without the cap the output accepts any number of
// them.
func TestUniformCollapseKeepsTheLengthBound(t *testing.T) {
	const spec = `swagger: "2.0"
info:
  title: t
  version: "1.0.0"
paths: {}
definitions:
  Capped:
    type: array
    items:
      - type: string
      - type: string
    additionalItems: false
  AlreadyBounded:
    type: array
    maxItems: 2
    items:
      - type: string
      - type: string
  Unbounded:
    type: array
    items:
      - type: string
      - type: string
    additionalItems:
      type: string
`
	parsed, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion("3.0.3"))
	require.NoError(t, err)
	schemas := result.Document.(*parser.OAS3Document).Components.Schemas

	capped := schemas["Capped"]
	require.NotNil(t, capped.MaxItems, "additionalItems: false capped the array; the cap has to survive")
	assert.Equal(t, 2, *capped.MaxItems)

	// The other two justifications need no cap: one is already bounded, and an
	// additionalItems equal to the positions never bounded anything.
	require.NotNil(t, schemas["AlreadyBounded"].MaxItems)
	assert.Equal(t, 2, *schemas["AlreadyBounded"].MaxItems, "an existing bound is not rewritten")
	assert.Nil(t, schemas["Unbounded"].MaxItems, "a repeated element schema is not a length bound")

	for _, name := range []string{"Capped", "AlreadyBounded", "Unbounded"} {
		single, ok := schemas[name].Items.(*parser.Schema)
		require.True(t, ok, "%s should have collapsed, got %T", name, schemas[name].Items)
		assert.Equal(t, "string", single.Type)
	}
}

// TestMalformedItemsDoesNotTakeAdditionalItemsWithIt covers the field beside the
// one being reported. Converting to OAS 2.0 spells the positions as a draft 4
// tuple in `items`, which makes `additionalItems` the live trailing constraint,
// so it has to survive the removal of an unrelated malformed value.
func TestMalformedItemsDoesNotTakeAdditionalItemsWithIt(t *testing.T) {
	const spec = `openapi: 3.1.0
info:
  title: t
  version: "1.0.0"
paths: {}
components:
  schemas:
    Row:
      type: array
      prefixItems:
        - type: string
      items:
        - type: number
      additionalItems:
        type: integer
`
	parsed, err := parser.New().ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := ConvertWithOptions(WithParsed(*parsed), WithTargetVersion("2.0"))
	require.NoError(t, err)

	row := result.Document.(*parser.OAS2Document).Definitions["Row"]
	tuple, ok := row.Items.([]*parser.Schema)
	require.True(t, ok, "the prefixItems positions become the tuple, got %T", row.Items)
	require.Len(t, tuple, 1)

	rest, ok := row.AdditionalItems.(*parser.Schema)
	require.True(t, ok, "additionalItems is live beside a draft 4 tuple and must survive, got %T", row.AdditionalItems)
	assert.Equal(t, "integer", rest.Type)
}
