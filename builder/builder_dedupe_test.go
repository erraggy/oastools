package builder

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addSchema is a test helper to add a schema directly to the builder's internal map
func (b *Builder) addSchema(name string, schema *parser.Schema) {
	b.schemas[name] = schema
}

func TestBuilder_DeduplicateSchemas_Empty(t *testing.T) {
	b := New(parser.OASVersion320)
	b.DeduplicateSchemas()

	assert.Len(t, b.schemaAliases, 0)
}

func TestBuilder_DeduplicateSchemas_Single(t *testing.T) {
	b := New(parser.OASVersion320)
	b.addSchema("User", &parser.Schema{Type: "object"})
	b.DeduplicateSchemas()

	assert.Len(t, b.schemas, 1)
	assert.Len(t, b.schemaAliases, 0)
}

func TestBuilder_DeduplicateSchemas_Duplicates(t *testing.T) {
	b := New(parser.OASVersion320)

	// Add identical schemas with different names
	b.addSchema("Address", &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"street": {Type: "string"},
			"city":   {Type: "string"},
		},
		Required: []string{"street", "city"},
	})
	b.addSchema("Location", &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"street": {Type: "string"},
			"city":   {Type: "string"},
		},
		Required: []string{"street", "city"},
	})

	b.DeduplicateSchemas()

	// Should have 1 canonical schema (Address, alphabetically first)
	assert.Len(t, b.schemas, 1)
	assert.Contains(t, b.schemas, "Address", "Expected Address to be canonical (alphabetically first)")

	// Should have 1 alias
	assert.Len(t, b.schemaAliases, 1)
	assert.Equal(t, "Address", b.schemaAliases["Location"])
}

func TestBuilder_DeduplicateSchemas_NoDuplicates(t *testing.T) {
	b := New(parser.OASVersion320)

	b.addSchema("User", &parser.Schema{Type: "object"})
	b.addSchema("Address", &parser.Schema{Type: "string"})
	b.addSchema("Age", &parser.Schema{Type: "integer"})

	b.DeduplicateSchemas()

	assert.Len(t, b.schemas, 3)
	assert.Len(t, b.schemaAliases, 0)
}

func TestBuilder_WithSemanticDeduplication_OAS3(t *testing.T) {
	b := New(parser.OASVersion320, WithSemanticDeduplication(true))

	// Add identical schemas
	b.addSchema("Address", &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
		},
	})
	b.addSchema("Location", &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
		},
	})

	// Add a reference to Location
	b.addSchema("Order", &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"shipTo": {Ref: "#/components/schemas/Location"},
		},
	})

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	// Should have 2 schemas (Address canonical, Order)
	assert.Len(t, doc.Components.Schemas, 2)

	// Check that Order's reference was rewritten to Address
	orderSchema := doc.Components.Schemas["Order"]
	shipToRef := orderSchema.Properties["shipTo"].Ref
	expectedRef := "#/components/schemas/Address"
	assert.Equal(t, expectedRef, shipToRef)
}

func TestBuilder_WithSemanticDeduplication_OAS2(t *testing.T) {
	b := New(parser.OASVersion20, WithSemanticDeduplication(true))

	// Add identical schemas
	b.addSchema("Address", &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
		},
	})
	b.addSchema("Location", &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
		},
	})

	// Add a reference to Location
	b.addSchema("Order", &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"shipTo": {Ref: "#/definitions/Location"},
		},
	})

	doc, err := b.BuildOAS2()
	require.NoError(t, err)

	// Should have 2 definitions (Address canonical, Order)
	assert.Len(t, doc.Definitions, 2)

	// Check that Order's reference was rewritten to Address
	orderSchema := doc.Definitions["Order"]
	shipToRef := orderSchema.Properties["shipTo"].Ref
	expectedRef := "#/definitions/Address"
	assert.Equal(t, expectedRef, shipToRef)
}

func TestBuilder_WithSemanticDeduplication_Disabled(t *testing.T) {
	// Default behavior: deduplication disabled
	b := New(parser.OASVersion320)

	b.addSchema("Address", &parser.Schema{Type: "object"})
	b.addSchema("Location", &parser.Schema{Type: "object"})

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	// Should have both schemas (no dedup)
	assert.Len(t, doc.Components.Schemas, 2)
}

func TestBuilder_DeduplicateSchemas_MetadataIgnored(t *testing.T) {
	b := New(parser.OASVersion320, WithSemanticDeduplication(true))

	// Schemas differ only in metadata - should still be deduplicated
	b.addSchema("Address", &parser.Schema{
		Type:        "object",
		Title:       "An Address",
		Description: "Represents a physical address",
		Properties: map[string]*parser.Schema{
			"street": {Type: "string"},
		},
	})
	b.addSchema("Location", &parser.Schema{
		Type:        "object",
		Title:       "A Location",
		Description: "A place on earth",
		Properties: map[string]*parser.Schema{
			"street": {Type: "string"},
		},
	})

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	// Should deduplicate since structural properties are the same
	assert.Len(t, doc.Components.Schemas, 1)
}

func TestBuilder_DeduplicateSchemas_MultipleGroups(t *testing.T) {
	b := New(parser.OASVersion320, WithSemanticDeduplication(true))

	// Group 1: objects with name property
	b.addSchema("Address", &parser.Schema{
		Type:       "object",
		Properties: map[string]*parser.Schema{"name": {Type: "string"}},
	})
	b.addSchema("Location", &parser.Schema{
		Type:       "object",
		Properties: map[string]*parser.Schema{"name": {Type: "string"}},
	})

	// Group 2: simple strings
	b.addSchema("Name", &parser.Schema{Type: "string"})
	b.addSchema("Title", &parser.Schema{Type: "string"})

	// Unique schema
	b.addSchema("Age", &parser.Schema{Type: "integer"})

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	// Should have 3 schemas: Address (canonical), Age (unique), Name (canonical)
	assert.Len(t, doc.Components.Schemas, 3)

	// Verify canonical names
	assert.Contains(t, doc.Components.Schemas, "Address", "Expected Address to be canonical")
	assert.Contains(t, doc.Components.Schemas, "Name", "Expected Name to be canonical")
	assert.Contains(t, doc.Components.Schemas, "Age", "Expected Age to be present")
}

func TestBuilder_DeduplicateSchemas_NestedReferences(t *testing.T) {
	b := New(parser.OASVersion320, WithSemanticDeduplication(true))

	// Add identical schemas
	b.addSchema("Address", &parser.Schema{
		Type:       "object",
		Properties: map[string]*parser.Schema{"street": {Type: "string"}},
	})
	b.addSchema("Location", &parser.Schema{
		Type:       "object",
		Properties: map[string]*parser.Schema{"street": {Type: "string"}},
	})

	// Add schema with nested reference to Location
	b.addSchema("Person", &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"home": {Ref: "#/components/schemas/Location"},
			"work": {Ref: "#/components/schemas/Location"},
		},
	})

	// Add allOf reference
	b.addSchema("Employee", &parser.Schema{
		AllOf: []*parser.Schema{
			{Ref: "#/components/schemas/Person"},
			{
				Type: "object",
				Properties: map[string]*parser.Schema{
					"office": {Ref: "#/components/schemas/Location"},
				},
			},
		},
	})

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	// Verify all Location refs are rewritten to Address
	person := doc.Components.Schemas["Person"]
	assert.Equal(t, "#/components/schemas/Address", person.Properties["home"].Ref)
	assert.Equal(t, "#/components/schemas/Address", person.Properties["work"].Ref)

	employee := doc.Components.Schemas["Employee"]
	officeRef := employee.AllOf[1].Properties["office"].Ref
	assert.Equal(t, "#/components/schemas/Address", officeRef)
}

func TestBuilder_DeduplicateSchemas_ManualCall(t *testing.T) {
	// Test calling DeduplicateSchemas manually (without option)
	b := New(parser.OASVersion320)

	b.addSchema("Address", &parser.Schema{Type: "object"})
	b.addSchema("Location", &parser.Schema{Type: "object"})

	// Manually call deduplication
	b.DeduplicateSchemas()

	// Verify aliases are set
	assert.Len(t, b.schemaAliases, 1)

	// Build should use the aliases for rewriting
	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	assert.Len(t, doc.Components.Schemas, 1)
}
