package differ

// Concern: the OAS 2.0 tuple form of the schema-or-bool fields (items,
// additionalItems, unevaluatedItems, unevaluatedProperties,
// additionalProperties), where the field holds []*parser.Schema rather than a
// single *parser.Schema or a bool (#502).

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tupleFieldSetter installs a schema-or-bool value on the field under test.
type tupleFieldSetter func(schema *parser.Schema, value any)

func setItems(schema *parser.Schema, value any)       { schema.Items = value }
func setAdditionalItems(s *parser.Schema, value any)  { s.AdditionalItems = value }
func setUnevaluatedItems(s *parser.Schema, value any) { s.UnevaluatedItems = value }
func setUnevaluatedProperties(s *parser.Schema, value any) {
	s.UnevaluatedProperties = value
}
func setAdditionalProperties(s *parser.Schema, value any) {
	s.AdditionalProperties = value
}

// findChange returns the first change reported at path.
func findChange(t *testing.T, result *DiffResult, path string) Change {
	t.Helper()
	for _, change := range result.Changes {
		if change.Path == path {
			return change
		}
	}
	paths := make([]string, 0, len(result.Changes))
	for _, change := range result.Changes {
		paths = append(paths, change.Path)
	}
	t.Fatalf("no change reported at %q, got: %v", path, paths)
	return Change{}
}

// TestDiffSchemaTupleForm covers every schema-or-bool field in its tuple form,
// in both diff modes.
func TestDiffSchemaTupleForm(t *testing.T) {
	tests := []struct {
		name            string
		set             tupleFieldSetter
		source          any
		target          any
		wantPath        string
		wantType        ChangeType
		wantSeverity    Severity
		wantChangeCount int
		wantNoChanges   bool
		wantMessagePart string
	}{
		{
			name:          "items tuple unchanged",
			set:           setItems,
			source:        []*parser.Schema{{Type: "string"}, {Type: "integer"}},
			target:        []*parser.Schema{{Type: "string"}, {Type: "integer"}},
			wantNoChanges: true,
		},
		{
			name:            "items tuple element type changed",
			set:             setItems,
			source:          []*parser.Schema{{Type: "string"}, {Type: "integer"}},
			target:          []*parser.Schema{{Type: "string"}, {Type: "boolean"}},
			wantPath:        "test.schema.items[1].type",
			wantType:        ChangeTypeModified,
			wantSeverity:    SeverityError,
			wantChangeCount: 1,
		},
		{
			name:            "items tuple grew",
			set:             setItems,
			source:          []*parser.Schema{{Type: "string"}},
			target:          []*parser.Schema{{Type: "string"}, {Type: "integer"}},
			wantPath:        "test.schema.items[1]",
			wantType:        ChangeTypeAdded,
			wantSeverity:    SeverityInfo,
			wantChangeCount: 1,
		},
		{
			name:            "items tuple shrank",
			set:             setItems,
			source:          []*parser.Schema{{Type: "string"}, {Type: "integer"}},
			target:          []*parser.Schema{{Type: "string"}},
			wantPath:        "test.schema.items[1]",
			wantType:        ChangeTypeRemoved,
			wantSeverity:    SeverityWarning,
			wantChangeCount: 1,
		},
		{
			name:            "items tuple to single schema",
			set:             setItems,
			source:          []*parser.Schema{{Type: "string"}},
			target:          &parser.Schema{Type: "string"},
			wantPath:        "test.schema.items",
			wantType:        ChangeTypeModified,
			wantSeverity:    SeverityError,
			wantChangeCount: 1,
			wantMessagePart: "items changed from tuple of schemas to schema",
		},
		{
			name:            "items single schema to tuple",
			set:             setItems,
			source:          &parser.Schema{Type: "string"},
			target:          []*parser.Schema{{Type: "string"}},
			wantPath:        "test.schema.items",
			wantType:        ChangeTypeModified,
			wantSeverity:    SeverityError,
			wantChangeCount: 1,
			wantMessagePart: "items changed from schema to tuple of schemas",
		},
		{
			name:            "items tuple to bool",
			set:             setItems,
			source:          []*parser.Schema{{Type: "string"}},
			target:          true,
			wantPath:        "test.schema.items",
			wantType:        ChangeTypeModified,
			wantSeverity:    SeverityError,
			wantChangeCount: 1,
			wantMessagePart: "items changed from tuple of schemas to boolean",
		},
		{
			name:            "items nil to tuple",
			set:             setItems,
			source:          nil,
			target:          []*parser.Schema{{Type: "string"}},
			wantPath:        "test.schema.items",
			wantType:        ChangeTypeAdded,
			wantSeverity:    SeverityWarning,
			wantChangeCount: 1,
		},
		{
			name:            "items tuple to nil",
			set:             setItems,
			source:          []*parser.Schema{{Type: "string"}},
			target:          nil,
			wantPath:        "test.schema.items",
			wantType:        ChangeTypeRemoved,
			wantSeverity:    SeverityError,
			wantChangeCount: 1,
		},
		{
			name:            "additionalItems tuple element type changed",
			set:             setAdditionalItems,
			source:          []*parser.Schema{{Type: "string"}},
			target:          []*parser.Schema{{Type: "integer"}},
			wantPath:        "test.schema.additionalItems[0].type",
			wantType:        ChangeTypeModified,
			wantSeverity:    SeverityError,
			wantChangeCount: 1,
		},
		{
			name:            "additionalItems single schema compared",
			set:             setAdditionalItems,
			source:          &parser.Schema{Type: "string"},
			target:          &parser.Schema{Type: "integer"},
			wantPath:        "test.schema.additionalItems.type",
			wantType:        ChangeTypeModified,
			wantSeverity:    SeverityError,
			wantChangeCount: 1,
		},
		{
			name:            "unevaluatedItems tuple element type changed",
			set:             setUnevaluatedItems,
			source:          []*parser.Schema{{Type: "string"}},
			target:          []*parser.Schema{{Type: "integer"}},
			wantPath:        "test.schema.unevaluatedItems[0].type",
			wantType:        ChangeTypeModified,
			wantSeverity:    SeverityError,
			wantChangeCount: 1,
		},
		{
			name:            "unevaluatedProperties tuple element type changed",
			set:             setUnevaluatedProperties,
			source:          []*parser.Schema{{Type: "string"}},
			target:          []*parser.Schema{{Type: "integer"}},
			wantPath:        "test.schema.unevaluatedProperties[0].type",
			wantType:        ChangeTypeModified,
			wantSeverity:    SeverityError,
			wantChangeCount: 1,
		},
		{
			name:            "additionalProperties tuple element type changed",
			set:             setAdditionalProperties,
			source:          []*parser.Schema{{Type: "string"}},
			target:          []*parser.Schema{{Type: "integer"}},
			wantPath:        "test.schema.additionalProperties[0].type",
			wantType:        ChangeTypeModified,
			wantSeverity:    SeverityError,
			wantChangeCount: 1,
		},
		{
			name:            "unevaluatedItems tuple to bool",
			set:             setUnevaluatedItems,
			source:          []*parser.Schema{{Type: "string"}},
			target:          false,
			wantPath:        "test.schema.unevaluatedItems",
			wantType:        ChangeTypeModified,
			wantSeverity:    SeverityWarning,
			wantChangeCount: 1,
			wantMessagePart: "unevaluatedItems changed from tuple of schemas to boolean",
		},
		// A YAML `items: [null]` decodes to a nil element, so these shapes
		// reach the differ from a parsed document as well as from one built
		// in code. The JSON path drops the element instead (#510).
		{
			name:            "items tuple element became nil",
			set:             setItems,
			source:          []*parser.Schema{{Type: "string"}, {Type: "integer"}},
			target:          []*parser.Schema{{Type: "string"}, nil},
			wantPath:        "test.schema.items[1]",
			wantType:        ChangeTypeRemoved,
			wantSeverity:    SeverityError,
			wantChangeCount: 1,
		},
		{
			name:            "items tuple element was nil",
			set:             setItems,
			source:          []*parser.Schema{{Type: "string"}, nil},
			target:          []*parser.Schema{{Type: "string"}, {Type: "integer"}},
			wantPath:        "test.schema.items[1]",
			wantType:        ChangeTypeAdded,
			wantSeverity:    SeverityInfo,
			wantChangeCount: 1,
		},
	}

	modes := []struct {
		name string
		mode DiffMode
	}{
		{name: "ModeSimple", mode: ModeSimple},
		{name: "ModeBreaking", mode: ModeBreaking},
	}

	for _, tt := range tests {
		for _, mode := range modes {
			t.Run(tt.name+"/"+mode.name, func(t *testing.T) {
				source := &parser.Schema{Type: "array"}
				target := &parser.Schema{Type: "array"}
				tt.set(source, tt.source)
				tt.set(target, tt.target)

				d := New()
				d.Mode = mode.mode
				result := &DiffResult{}
				d.diffSchemaRecursiveUnified(source, target, "test.schema", newSchemaVisited(), result)

				if tt.wantNoChanges {
					assert.Empty(t, result.Changes, "expected no changes")
					return
				}

				assert.Len(t, result.Changes, tt.wantChangeCount)
				change := findChange(t, result, tt.wantPath)
				assert.Equal(t, tt.wantType, change.Type)

				wantSeverity := tt.wantSeverity
				if mode.mode == ModeSimple {
					// ModeSimple leaves Severity unset. That zero value is
					// SeverityError, so this asserts the field was not filled
					// in, not that the change is an error.
					wantSeverity = 0
				}
				assert.Equal(t, wantSeverity, change.Severity)

				if tt.wantMessagePart != "" {
					assert.Contains(t, change.Message, tt.wantMessagePart)
				}
			})
		}
	}
}

// TestDiffSchemaTupleMessagesNameOASShapes checks that no reported message
// leaks a Go type name for a legal OAS 2.0 document.
func TestDiffSchemaTupleMessagesNameOASShapes(t *testing.T) {
	shapes := []any{
		[]*parser.Schema{{Type: "string"}},
		&parser.Schema{Type: "string"},
		true,
		nil,
	}

	d := New()
	d.Mode = ModeBreaking

	for _, sourceShape := range shapes {
		for _, targetShape := range shapes {
			source := &parser.Schema{
				Type:                  "array",
				Items:                 sourceShape,
				AdditionalItems:       sourceShape,
				AdditionalProperties:  sourceShape,
				UnevaluatedItems:      sourceShape,
				UnevaluatedProperties: sourceShape,
			}
			target := &parser.Schema{
				Type:                  "array",
				Items:                 targetShape,
				AdditionalItems:       targetShape,
				AdditionalProperties:  targetShape,
				UnevaluatedItems:      targetShape,
				UnevaluatedProperties: targetShape,
			}

			result := &DiffResult{}
			d.diffSchemaRecursiveUnified(source, target, "test.schema", newSchemaVisited(), result)

			for _, change := range result.Changes {
				assert.NotContains(t, change.Message, "parser.Schema",
					"message leaks a Go type name: %q", change.Message)
			}
		}
	}
}

// TestDiffSchemaTupleItemsEndToEnd proves the decode path yields
// []*parser.Schema for the OAS 2.0 tuple form, so the tuple arm is reached for
// a real document rather than only for a hand-built schema.
func TestDiffSchemaTupleItemsEndToEnd(t *testing.T) {
	sourceSpec := `swagger: "2.0"
info:
  title: Tuple API
  version: 1.0.0
paths: {}
definitions:
  A:
    type: array
    items:
      - type: string
      - type: integer
`

	targetSpec := `swagger: "2.0"
info:
  title: Tuple API
  version: 1.0.0
paths: {}
definitions:
  A:
    type: array
    items:
      - type: boolean
      - type: integer
`

	source, err := parser.ParseWithOptions(parser.WithReader(strings.NewReader(sourceSpec)))
	require.NoError(t, err)
	target, err := parser.ParseWithOptions(parser.WithReader(strings.NewReader(targetSpec)))
	require.NoError(t, err)

	sourceDoc, ok := source.Document.(*parser.OAS2Document)
	require.True(t, ok, "expected an OAS 2.0 document")
	_, isTuple := sourceDoc.Definitions["A"].Items.([]*parser.Schema)
	require.True(t, isTuple, "expected the tuple form to decode to []*parser.Schema, got %T",
		sourceDoc.Definitions["A"].Items)

	result, err := DiffWithOptions(
		WithSourceParsed(*source),
		WithTargetParsed(*target),
		WithMode(ModeBreaking),
	)
	require.NoError(t, err)

	change := findChange(t, result, "document.definitions.A.items[0].type")
	assert.Equal(t, ChangeTypeModified, change.Type)
	assert.Equal(t, SeverityError, change.Severity)
}

// TestDiffSchemaTupleNilElementEndToEnd proves a nil tuple element reaches the
// differ from a parsed document: YAML decodes `- null` to a nil *Schema. The
// JSON path drops the element instead, which is #510.
func TestDiffSchemaTupleNilElementEndToEnd(t *testing.T) {
	sourceSpec := `swagger: "2.0"
info:
  title: Tuple API
  version: 1.0.0
paths: {}
definitions:
  A:
    type: array
    items:
      - type: string
      - type: integer
`

	targetSpec := `swagger: "2.0"
info:
  title: Tuple API
  version: 1.0.0
paths: {}
definitions:
  A:
    type: array
    items:
      - type: string
      - null
`

	source, err := parser.ParseWithOptions(parser.WithReader(strings.NewReader(sourceSpec)))
	require.NoError(t, err)
	target, err := parser.ParseWithOptions(parser.WithReader(strings.NewReader(targetSpec)))
	require.NoError(t, err)

	targetDoc, ok := target.Document.(*parser.OAS2Document)
	require.True(t, ok, "expected an OAS 2.0 document")
	elems, isTuple := targetDoc.Definitions["A"].Items.([]*parser.Schema)
	require.True(t, isTuple, "expected the tuple form to decode to []*parser.Schema, got %T",
		targetDoc.Definitions["A"].Items)
	require.Len(t, elems, 2, "expected the null element to be kept, so indices stay aligned")
	require.Nil(t, elems[1], "expected the null element to decode to a nil schema")

	result, err := DiffWithOptions(
		WithSourceParsed(*source),
		WithTargetParsed(*target),
		WithMode(ModeBreaking),
	)
	require.NoError(t, err)

	change := findChange(t, result, "document.definitions.A.items[1]")
	assert.Equal(t, ChangeTypeRemoved, change.Type)
	assert.Equal(t, SeverityError, change.Severity)
}
