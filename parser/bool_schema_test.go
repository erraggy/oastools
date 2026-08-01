package parser

import (
	"encoding/json"
	"testing"

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
			if err != nil {
				t.Fatalf("ParseBytes: %v", err)
			}
			doc, ok := result.OAS3Document()
			if !ok {
				t.Fatal("not an OAS3 document")
			}

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
				if c.schema == nil {
					t.Errorf("%s: schema was dropped", c.name)
					continue
				}
				gotValue, gotBool := c.schema.IsBool()
				if gotBool != c.wantBool || gotValue != c.wantValue {
					t.Errorf("%s: IsBool() = (%v, %v), want (%v, %v)",
						c.name, gotValue, gotBool, c.wantValue, c.wantBool)
				}
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
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	doc, ok := result.OAS3Document()
	if !ok {
		t.Fatal("not an OAS3 document")
	}

	if got := len(doc.Components.Schemas); got != 4 {
		t.Errorf("got %d component schemas, want 4 — a boolean schema was dropped", got)
	}
	for name, want := range map[string]bool{"anything": true, "nothing": false} {
		s := doc.Components.Schemas[name]
		if s == nil {
			t.Errorf("%s: dropped by the ResolveRefs decode path", name)
			continue
		}
		if v, ok := s.IsBool(); !ok || v != want {
			t.Errorf("%s: IsBool() = (%v, %v), want (%v, true)", name, v, ok, want)
		}
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
			if err != nil {
				t.Fatalf("yaml.Marshal: %v", err)
			}
			if string(gotYAML) != tt.wantYAML {
				t.Errorf("yaml = %q, want %q", gotYAML, tt.wantYAML)
			}

			gotJSON, err := json.Marshal(tt.schema)
			if err != nil {
				t.Fatalf("json.Marshal: %v", err)
			}
			if string(gotJSON) != tt.wantJSON {
				t.Errorf("json = %q, want %q", gotJSON, tt.wantJSON)
			}
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
			if got := tt.a.Equals(tt.b); got != tt.equal {
				t.Errorf("Equals() = %v, want %v", got, tt.equal)
			}
		})
	}
}

// TestBoolSchemaDeepCopyDoesNotAlias guards the pointer. BoolForm is a *bool,
// so a struct copy without an explicit deep copy would share the pointee
// between the original and the copy.
func TestBoolSchemaDeepCopyDoesNotAlias(t *testing.T) {
	original := NewBoolSchema(true)
	clone := original.DeepCopy()

	if v, ok := clone.IsBool(); !ok || !v {
		t.Fatalf("clone.IsBool() = (%v, %v), want (true, true)", v, ok)
	}
	if original.BoolForm == clone.BoolForm {
		t.Error("DeepCopy shares the BoolForm pointer with the original")
	}

	*clone.BoolForm = false
	if v, _ := original.IsBool(); !v {
		t.Error("mutating the clone changed the original")
	}
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
	if _, err := New().ParseBytes([]byte(spec)); err == nil {
		t.Error("want an error for a string where a schema is expected, got nil")
	}
}
