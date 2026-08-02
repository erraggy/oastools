package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

// boolSchemaSpecs is the same document in both source formats. parser keeps
// separate YAML and JSON decode paths, so anything asserted about decoding has
// to be asserted twice or it covers half the surface.
var boolSchemaSpecs = map[string]string{
	"yaml": `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  schemas:
    anything: true
    nothing: false
    object: {}
    nested:
      type: object
      properties:
        p: true
        q: false
`,
	"json": `{
  "openapi": "3.2.0",
  "info": {"title": "API", "version": "1.0.0"},
  "components": {
    "schemas": {
      "anything": true,
      "nothing": false,
      "object": {},
      "nested": {
        "type": "object",
        "properties": {"p": true, "q": false}
      }
    }
  }
}`,
}

// TestBoolSchemaDecodes covers the bare-boolean schema form that JSON Schema
// 2020-12 allows and OAS 3.1+ adopts. `true` accepts anything, `false` accepts
// nothing, and both are legal wherever a Schema Object is expected.
func TestBoolSchemaDecodes(t *testing.T) {
	for _, format := range []string{"yaml", "json"} {
		t.Run(format, func(t *testing.T) {
			result, err := New().ParseBytes([]byte(boolSchemaSpecs[format]))
			require.NoError(t, err)
			doc, ok := result.OAS3Document()
			require.True(t, ok, "expected an OAS3 document")

			checks := []struct {
				name      string
				schema    *Schema
				wantValue bool
				wantBool  bool
			}{
				{"anything", doc.Components.Schemas["anything"], true, true},
				{"nothing", doc.Components.Schemas["nothing"], false, true},
				// An empty object is a schema that constrains nothing. It is not
				// the boolean `true`, and must not be reported as one.
				{"object", doc.Components.Schemas["object"], false, false},
				{"nested.properties.p", doc.Components.Schemas["nested"].Properties["p"], true, true},
				{"nested.properties.q", doc.Components.Schemas["nested"].Properties["q"], false, true},
			}
			for _, c := range checks {
				require.NotNil(t, c.schema, "%s: schema was dropped", c.name)
				gotValue, gotBool := c.schema.IsBool()
				assert.Equal(t, c.wantBool, gotBool, "%s: IsBool ok", c.name)
				assert.Equal(t, c.wantValue, gotValue, "%s: IsBool value", c.name)
			}
		})
	}
}

// TestBoolSchemaSurvivesResolveRefs covers the third decode path. decodeFromMap
// is map-driven, so before decodeSchemaValue existed it dropped a boolean value
// silently — the schema simply was not there, with no error to say so.
func TestBoolSchemaSurvivesResolveRefs(t *testing.T) {
	p := New()
	p.ResolveRefs = true

	result, err := p.ParseBytes([]byte(boolSchemaSpecs["yaml"]))
	require.NoError(t, err)
	doc, ok := result.OAS3Document()
	require.True(t, ok, "expected an OAS3 document")

	assert.Len(t, doc.Components.Schemas, 4, "a boolean schema was dropped")
	for name, want := range map[string]bool{"anything": true, "nothing": false} {
		schema := doc.Components.Schemas[name]
		require.NotNil(t, schema, "%s: dropped by the ResolveRefs decode path", name)
		got, isBool := schema.IsBool()
		assert.True(t, isBool, "%s: not reported as a boolean schema", name)
		assert.Equal(t, want, got, "%s: wrong boolean value", name)
	}
}

// TestBoolSchemaRoundTrips checks that a boolean schema serializes back as the
// bare scalar. Emitting `{}` instead would silently rewrite it into a different
// schema — one that constrains nothing, rather than `false`, which permits
// nothing.
func TestBoolSchemaRoundTrips(t *testing.T) {
	tests := []struct {
		name     string
		schema   *Schema
		wantYAML string
		wantJSON string
	}{
		{"true", NewBoolSchema(true), "true\n", "true"},
		{"false", NewBoolSchema(false), "false\n", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotYAML, err := yaml.Marshal(tt.schema)
			require.NoError(t, err)
			assert.Equal(t, tt.wantYAML, string(gotYAML))

			gotJSON, err := json.Marshal(tt.schema)
			require.NoError(t, err)
			assert.Equal(t, tt.wantJSON, string(gotJSON))
		})
	}
}

// TestBoolSchemaEquality guards the pair that matters most: `true` and `false`
// are opposite schemas. Treating BoolForm as spelling — the way
// Discriminator.StringForm is treated — would make them compare equal and let
// semantic deduplication merge them.
func TestBoolSchemaEquality(t *testing.T) {
	tests := []struct {
		name  string
		a, b  *Schema
		equal bool
	}{
		{"true equals true", NewBoolSchema(true), NewBoolSchema(true), true},
		{"false equals false", NewBoolSchema(false), NewBoolSchema(false), true},
		{"true does not equal false", NewBoolSchema(true), NewBoolSchema(false), false},
		{"true does not equal empty object", NewBoolSchema(true), &Schema{}, false},
		{"false does not equal empty object", NewBoolSchema(false), &Schema{}, false},
		{"empty object does not equal true", &Schema{}, NewBoolSchema(true), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.equal, tt.a.Equals(tt.b))
		})
	}
}

// TestIsBool covers the accessor's own contract, including the nil receiver it
// documents. The two returns are independent: (false, false) is an ordinary
// object schema, while (false, true) is the boolean schema `false`.
func TestIsBool(t *testing.T) {
	tests := []struct {
		name      string
		schema    *Schema
		wantValue bool
		wantOK    bool
	}{
		{"nil receiver", nil, false, false},
		{"empty object schema", &Schema{}, false, false},
		{"populated object schema", &Schema{Type: "string"}, false, false},
		{"boolean true", NewBoolSchema(true), true, true},
		{"boolean false", NewBoolSchema(false), false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotOK := tt.schema.IsBool()
			assert.Equal(t, tt.wantOK, gotOK)
			assert.Equal(t, tt.wantValue, gotValue)
		})
	}
}

// TestBoolSchemaDeepCopyDoesNotAlias guards the pointer. BoolForm is a *bool,
// so a struct copy without an explicit deep copy would share the pointee
// between the original and the copy.
func TestBoolSchemaDeepCopyDoesNotAlias(t *testing.T) {
	original := NewBoolSchema(true)
	clone := original.DeepCopy()

	value, ok := clone.IsBool()
	require.True(t, ok, "clone is not a boolean schema")
	require.True(t, value, "clone has the wrong value")
	require.NotSame(t, original.BoolForm, clone.BoolForm,
		"DeepCopy shares the BoolForm pointer with the original")

	*clone.BoolForm = false
	originalValue, _ := original.IsBool()
	assert.True(t, originalValue, "mutating the clone changed the original")
}

// TestQuotedTrueIsNotABoolSchema guards the tag check. In YAML a quoted "true"
// is a string scalar, which is not a schema at all — so it must not be silently
// accepted as the boolean form.
func TestQuotedTrueIsNotABoolSchema(t *testing.T) {
	spec := `
openapi: 3.2.0
info:
  title: API
  version: 1.0.0
components:
  schemas:
    quoted: "true"
`
	_, err := New().ParseBytes([]byte(spec))
	assert.Error(t, err, "a string where a schema is expected should not parse")
}
