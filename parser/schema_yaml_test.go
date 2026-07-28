package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// Items, AdditionalProperties, AdditionalItems, UnevaluatedItems and
// UnevaluatedProperties are typed any because JSON Schema lets each hold either
// a schema or a bool. Every decode path therefore has to promote the mapping
// form to *Schema itself, or consumers that type-assert to *Schema skip the
// subtree without reporting anything. The parser has three independent decode
// paths — encoding/json, YAML, and the generated decodeFromMap used when
// ResolveRefs is enabled — and they must agree on the decoded types.

// schemaOrBoolFields lists the any-typed schema-or-bool fields alongside a
// minimal document in each of the two forms, so a test can cover all five
// uniformly.
var schemaOrBoolFields = []struct {
	name       string
	get        func(*Schema) any
	schemaForm string
	boolForm   string
}{
	{
		name: "items",
		get:  func(s *Schema) any { return s.Items },
		schemaForm: `
items:
  type: object
`,
		boolForm: `
items: false
`,
	},
	{
		name: "additionalProperties",
		get:  func(s *Schema) any { return s.AdditionalProperties },
		schemaForm: `
additionalProperties:
  type: object
`,
		boolForm: `
additionalProperties: false
`,
	},
	{
		name: "additionalItems",
		get:  func(s *Schema) any { return s.AdditionalItems },
		schemaForm: `
additionalItems:
  type: object
`,
		boolForm: `
additionalItems: false
`,
	},
	{
		name: "unevaluatedItems",
		get:  func(s *Schema) any { return s.UnevaluatedItems },
		schemaForm: `
unevaluatedItems:
  type: object
`,
		boolForm: `
unevaluatedItems: false
`,
	},
	{
		name: "unevaluatedProperties",
		get:  func(s *Schema) any { return s.UnevaluatedProperties },
		schemaForm: `
unevaluatedProperties:
  type: object
`,
		boolForm: `
unevaluatedProperties: false
`,
	},
}

func TestSchemaUnmarshalYAMLPromotesSchemaOrBoolFields(t *testing.T) {
	for _, field := range schemaOrBoolFields {
		t.Run(field.name, func(t *testing.T) {
			var s Schema
			require.NoError(t, yaml.Unmarshal([]byte(field.schemaForm), &s))

			nested, ok := field.get(&s).(*Schema)
			require.True(t, ok, "expected *Schema, got %T", field.get(&s))
			assert.Equal(t, "object", nested.Type)
		})
	}
}

func TestSchemaUnmarshalYAMLKeepsBoolForm(t *testing.T) {
	for _, field := range schemaOrBoolFields {
		t.Run(field.name, func(t *testing.T) {
			var s Schema
			require.NoError(t, yaml.Unmarshal([]byte(field.boolForm), &s))

			assert.Equal(t, false, field.get(&s), "bool form must survive promotion untouched")
		})
	}
}

func TestSchemaDecodePathsAgreeOnSchemaOrBoolTypes(t *testing.T) {
	const yamlSrc = `
type: array
items:
  type: object
  additionalProperties:
    type: string
additionalProperties:
  type: string
additionalItems:
  type: integer
unevaluatedItems:
  type: number
unevaluatedProperties:
  type: boolean
`
	const jsonSrc = `{
  "type": "array",
  "items": {"type": "object", "additionalProperties": {"type": "string"}},
  "additionalProperties": {"type": "string"},
  "additionalItems": {"type": "integer"},
  "unevaluatedItems": {"type": "number"},
  "unevaluatedProperties": {"type": "boolean"}
}`

	var fromYAML, fromJSON, fromMap Schema
	require.NoError(t, yaml.Unmarshal([]byte(yamlSrc), &fromYAML))
	require.NoError(t, json.Unmarshal([]byte(jsonSrc), &fromJSON))

	var raw map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(yamlSrc), &raw))
	fromMap.decodeFromMap(raw)

	for _, field := range schemaOrBoolFields {
		t.Run(field.name, func(t *testing.T) {
			assert.IsType(t, field.get(&fromJSON), field.get(&fromYAML),
				"YAML and JSON paths must decode %s to the same type", field.name)
			assert.IsType(t, field.get(&fromMap), field.get(&fromYAML),
				"YAML and decodeFromMap paths must decode %s to the same type", field.name)
		})
	}

	// Nesting has to promote too: a consumer walking into items and then into
	// that schema's additionalProperties must find a *Schema at both hops.
	items, ok := fromYAML.Items.(*Schema)
	require.True(t, ok, "expected items to be *Schema, got %T", fromYAML.Items)
	nested, ok := items.AdditionalProperties.(*Schema)
	require.True(t, ok, "expected items.additionalProperties to be *Schema, got %T", items.AdditionalProperties)
	assert.Equal(t, "string", nested.Type)
}

func TestSchemaUnmarshalYAMLItemsTupleForm(t *testing.T) {
	// OAS 2.0 tuple validation: items may be a sequence of schemas.
	const src = `
type: array
items:
  - type: string
  - type: integer
`
	var s Schema
	require.NoError(t, yaml.Unmarshal([]byte(src), &s))

	schemas, ok := s.Items.([]*Schema)
	require.True(t, ok, "expected []*Schema, got %T", s.Items)
	require.Len(t, schemas, 2)
	assert.Equal(t, "string", schemas[0].Type)
	assert.Equal(t, "integer", schemas[1].Type)
}

func TestSchemaUnmarshalYAMLPromotesThroughAnchor(t *testing.T) {
	const src = `
shared: &shared
  type: object
schema:
  type: array
  items: *shared
`
	var doc struct {
		Schema Schema `yaml:"schema"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(src), &doc))

	items, ok := doc.Schema.Items.(*Schema)
	require.True(t, ok, "expected *Schema, got %T", doc.Schema.Items)
	assert.Equal(t, "object", items.Type)
}

func TestSchemaUnmarshalYAMLPromotesThroughMergeKey(t *testing.T) {
	// A merged key is resolved by the decoder but never appears under the
	// merging node, so promotion falls back to the decoded map. The field must
	// still end up a *Schema.
	const src = `
base: &base
  items:
    type: object
    x-inner: inner
    unknown-inner: 2
schema:
  <<: *base
  type: array
`
	var doc struct {
		Schema Schema `yaml:"schema"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(src), &doc))

	items, ok := doc.Schema.Items.(*Schema)
	require.True(t, ok, "expected *Schema, got %T", doc.Schema.Items)
	assert.Equal(t, "object", items.Type)

	// Known limitation of the fallback, pinned rather than fixed. The map-based
	// decoder collects only x-* keys, so an unrecognized key inside a merged
	// subtree is dropped where the node-based path would have kept it. Matching
	// the node path would mean carrying a YAML-only quirk further: the JSON
	// path's ExtractExtensions is x-* only too, so what the fallback produces
	// here is what an equivalent JSON document produces everywhere.
	assert.Equal(t, "inner", items.Extra["x-inner"])
	assert.NotContains(t, items.Extra, "unknown-inner")
}

func TestSchemaUnmarshalYAMLKeepsExtensions(t *testing.T) {
	const src = `
type: array
x-outer: outer
unknown-outer: 1
items:
  type: object
  x-inner: inner
  unknown-inner: 2
`
	var s Schema
	require.NoError(t, yaml.Unmarshal([]byte(src), &s))

	assert.Equal(t, "outer", s.Extra["x-outer"], "the inline Extra map must survive the alias decode")

	items, ok := s.Items.(*Schema)
	require.True(t, ok, "expected *Schema, got %T", s.Items)
	assert.Equal(t, "inner", items.Extra["x-inner"], "a promoted schema must capture its own extensions")

	// The inline Extra map keeps every unrecognized key, not just x-*. Promoting
	// through the field's node preserves that; promoting the decoded map would
	// narrow it to x-* and lose these on a round trip.
	assert.Equal(t, 1, s.Extra["unknown-outer"])
	assert.Equal(t, 2, items.Extra["unknown-inner"], "a promoted schema must decode like a top-level one")
}

func TestSchemaUnmarshalYAMLReportsNestedDecodeErrors(t *testing.T) {
	// Promoting through the field's own node means a nested schema gets the
	// same decoding — and the same errors — as a top-level one. A sequence is
	// neither discriminator form, so Discriminator.UnmarshalYAML rejects it.
	const src = `
type: array
items:
  discriminator:
    - petType
`
	var s Schema
	err := yaml.Unmarshal([]byte(src), &s)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "discriminator")
}

func TestSchemaUnmarshalYAMLRoundTrip(t *testing.T) {
	const src = `
type: array
items:
  type: object
  additionalProperties: false
`
	var s Schema
	require.NoError(t, yaml.Unmarshal([]byte(src), &s))

	out, err := yaml.Marshal(&s)
	require.NoError(t, err)

	var reparsed Schema
	require.NoError(t, yaml.Unmarshal(out, &reparsed))

	items, ok := reparsed.Items.(*Schema)
	require.True(t, ok, "expected *Schema, got %T", reparsed.Items)
	assert.Equal(t, "object", items.Type)
	assert.Equal(t, false, items.AdditionalProperties)
}
