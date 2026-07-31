package driftguard

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/internal/schemautil"
	"github.com/erraggy/oastools/parser"
)

// The structural hash is what groups schemas before deduplication compares them.
// A field it does not read puts two different schemas in one bucket, and if the
// comparison misses the same field they are then merged. That is how the XML
// Object came to merge schemas serializing to different XML: neither the hash nor
// the comparison looked at it, so each gap hid the other.
//
// This guard sets one field at a time and requires the hash to move.

// hashExclusions lists the fields the hasher deliberately ignores, with the
// reason. [schemautil.SchemaHasher] is documented as a *structural* hash, so
// anything that cannot change how a payload validates or serializes belongs
// here rather than in the hash.
var hashExclusions = map[string]string{
	"Title":        "documentation, cannot change a payload",
	"Description":  "documentation, cannot change a payload",
	"Example":      "documentation, cannot change a payload",
	"Examples":     "documentation, cannot change a payload",
	"ExternalDocs": "documentation, cannot change a payload",
	"Comment":      "$comment is documentation, cannot change a payload",
	"Deprecated":   "advisory, cannot change a payload",
}

func TestHashReadsEveryStructuralSchemaField(t *testing.T) {
	hasher := schemautil.NewSchemaHasher()
	baseline := hasher.Hash(&parser.Schema{})

	for _, f := range fieldsOf[parser.Schema]() {
		t.Run(f.name, func(t *testing.T) {
			schema := &parser.Schema{}
			require.True(t, populate(schema, f),
				"populate cannot produce a value for Schema.%s; extend it rather than "+
					"leaving the field unchecked", f.name)

			// An excluded field is asserted to stay excluded rather than skipped. A
			// skip checks nothing, so hashing Title by accident would go unnoticed;
			// this way the exclusion is a claim the suite keeps honest.
			if reason, excluded := hashExclusions[f.name]; excluded {
				assert.Equal(t, baseline, hasher.Hash(schema),
					"Schema.%s is excluded from the structural hash (%s) but setting it "+
						"changed the hash", f.name, reason)
				return
			}

			assert.NotEqual(t, baseline, hasher.Hash(schema),
				"Schema.%s is set but the structural hash did not change; "+
					"two schemas differing only in this field land in one deduplication bucket",
				f.name)
		})
	}
}

// TestHashFramesAdjacentValues covers the other half of hashing correctly.
//
// SchemaHasher.writeString appends raw bytes, so a value written next to another
// with no framing runs into it and two different schemas produce one hash. A
// delimiter cannot fix this, because any sentinel byte can also occur inside a
// value; only length framing is injective.
//
// Each pair below collided at some point. They are grouped here rather than
// spread across schemautil's own tests because they are one defect, and the next
// field added to the hasher will be susceptible to it too.
func TestHashFramesAdjacentValues(t *testing.T) {
	hasher := schemautil.NewSchemaHasher()

	tests := []struct {
		name        string
		left, right *parser.Schema
	}{
		{
			name:  "required entries run together",
			left:  &parser.Schema{Type: "object", Required: []string{"ab"}},
			right: &parser.Schema{Type: "object", Required: []string{"a", "b"}},
		},
		{
			name:  "enum values run together",
			left:  &parser.Schema{Type: "string", Enum: []any{"ab"}},
			right: &parser.Schema{Type: "string", Enum: []any{"a", "b"}},
		},
		{
			name:  "dependentRequired entries run together",
			left:  &parser.Schema{Type: "object", DependentRequired: map[string][]string{"a": {"bc"}}},
			right: &parser.Schema{Type: "object", DependentRequired: map[string][]string{"a": {"b", "c"}}},
		},
		{
			name:  "xml values run together",
			left:  &parser.Schema{Type: "string", XML: &parser.XML{Name: "anamespace:b"}},
			right: &parser.Schema{Type: "string", XML: &parser.XML{Name: "a", Namespace: "b"}},
		},
		{
			name:  "an xml value containing the framing does not forge a boundary",
			left:  &parser.Schema{Type: "string", XML: &parser.XML{Name: "a1:bnamespace:1:c"}},
			right: &parser.Schema{Type: "string", XML: &parser.XML{Name: "a", Namespace: "b"}},
		},
		{
			name: "a discriminator mapping key runs into its value",
			left: &parser.Schema{Type: "object", Discriminator: &parser.Discriminator{
				PropertyName: "kind", Mapping: map[string]string{"ab": "c"}}},
			right: &parser.Schema{Type: "object", Discriminator: &parser.Discriminator{
				PropertyName: "kind", Mapping: map[string]string{"a": "bc"}}},
		},
		{
			name: "a mapping value forges the defaultMapping label",
			left: &parser.Schema{Type: "object", Discriminator: &parser.Discriminator{
				PropertyName: "kind", Mapping: map[string]string{"k": "xdefaultMapping:y"}}},
			right: &parser.Schema{Type: "object", Discriminator: &parser.Discriminator{
				PropertyName: "kind", Mapping: map[string]string{"k": "x"}, DefaultMapping: "y"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotEqual(t, hasher.Hash(tt.left), hasher.Hash(tt.right),
				"unframed values collided; length-frame them with writeLabeled")
		})
	}
}
