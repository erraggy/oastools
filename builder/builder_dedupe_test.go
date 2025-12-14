package builder

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

// addSchema is a test helper to add a schema directly to the builder's internal map
func (b *Builder) addSchema(name string, schema *parser.Schema) {
	b.schemas[name] = schema
}

func TestBuilder_DeduplicateSchemas_Empty(t *testing.T) {
	b := New(parser.OASVersion320)
	b.DeduplicateSchemas()

	if len(b.schemaAliases) != 0 {
		t.Errorf("Expected 0 aliases, got %d", len(b.schemaAliases))
	}
}

func TestBuilder_DeduplicateSchemas_Single(t *testing.T) {
	b := New(parser.OASVersion320)
	b.addSchema("User", &parser.Schema{Type: "object"})
	b.DeduplicateSchemas()

	if len(b.schemas) != 1 {
		t.Errorf("Expected 1 schema, got %d", len(b.schemas))
	}
	if len(b.schemaAliases) != 0 {
		t.Errorf("Expected 0 aliases, got %d", len(b.schemaAliases))
	}
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
	if len(b.schemas) != 1 {
		t.Errorf("Expected 1 schema after dedup, got %d", len(b.schemas))
	}
	if _, ok := b.schemas["Address"]; !ok {
		t.Error("Expected Address to be canonical (alphabetically first)")
	}

	// Should have 1 alias
	if len(b.schemaAliases) != 1 {
		t.Errorf("Expected 1 alias, got %d", len(b.schemaAliases))
	}
	if b.schemaAliases["Location"] != "Address" {
		t.Errorf("Expected Location -> Address, got %s", b.schemaAliases["Location"])
	}
}

func TestBuilder_DeduplicateSchemas_NoDuplicates(t *testing.T) {
	b := New(parser.OASVersion320)

	b.addSchema("User", &parser.Schema{Type: "object"})
	b.addSchema("Address", &parser.Schema{Type: "string"})
	b.addSchema("Age", &parser.Schema{Type: "integer"})

	b.DeduplicateSchemas()

	if len(b.schemas) != 3 {
		t.Errorf("Expected 3 schemas, got %d", len(b.schemas))
	}
	if len(b.schemaAliases) != 0 {
		t.Errorf("Expected 0 aliases, got %d", len(b.schemaAliases))
	}
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
	if err != nil {
		t.Fatalf("BuildOAS3 failed: %v", err)
	}

	// Should have 2 schemas (Address canonical, Order)
	if len(doc.Components.Schemas) != 2 {
		t.Errorf("Expected 2 schemas after dedup, got %d", len(doc.Components.Schemas))
	}

	// Check that Order's reference was rewritten to Address
	orderSchema := doc.Components.Schemas["Order"]
	shipToRef := orderSchema.Properties["shipTo"].Ref
	expectedRef := "#/components/schemas/Address"
	if shipToRef != expectedRef {
		t.Errorf("Expected shipTo.$ref = %s, got %s", expectedRef, shipToRef)
	}
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
	if err != nil {
		t.Fatalf("BuildOAS2 failed: %v", err)
	}

	// Should have 2 definitions (Address canonical, Order)
	if len(doc.Definitions) != 2 {
		t.Errorf("Expected 2 definitions after dedup, got %d", len(doc.Definitions))
	}

	// Check that Order's reference was rewritten to Address
	orderSchema := doc.Definitions["Order"]
	shipToRef := orderSchema.Properties["shipTo"].Ref
	expectedRef := "#/definitions/Address"
	if shipToRef != expectedRef {
		t.Errorf("Expected shipTo.$ref = %s, got %s", expectedRef, shipToRef)
	}
}

func TestBuilder_WithSemanticDeduplication_Disabled(t *testing.T) {
	// Default behavior: deduplication disabled
	b := New(parser.OASVersion320)

	b.addSchema("Address", &parser.Schema{Type: "object"})
	b.addSchema("Location", &parser.Schema{Type: "object"})

	doc, err := b.BuildOAS3()
	if err != nil {
		t.Fatalf("BuildOAS3 failed: %v", err)
	}

	// Should have both schemas (no dedup)
	if len(doc.Components.Schemas) != 2 {
		t.Errorf("Expected 2 schemas (no dedup), got %d", len(doc.Components.Schemas))
	}
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
	if err != nil {
		t.Fatalf("BuildOAS3 failed: %v", err)
	}

	// Should deduplicate since structural properties are the same
	if len(doc.Components.Schemas) != 1 {
		t.Errorf("Expected 1 schema (metadata ignored), got %d", len(doc.Components.Schemas))
	}
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
	if err != nil {
		t.Fatalf("BuildOAS3 failed: %v", err)
	}

	// Should have 3 schemas: Address (canonical), Age (unique), Name (canonical)
	if len(doc.Components.Schemas) != 3 {
		t.Errorf("Expected 3 schemas, got %d", len(doc.Components.Schemas))
	}

	// Verify canonical names
	if _, ok := doc.Components.Schemas["Address"]; !ok {
		t.Error("Expected Address to be canonical")
	}
	if _, ok := doc.Components.Schemas["Name"]; !ok {
		t.Error("Expected Name to be canonical")
	}
	if _, ok := doc.Components.Schemas["Age"]; !ok {
		t.Error("Expected Age to be present")
	}
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
	if err != nil {
		t.Fatalf("BuildOAS3 failed: %v", err)
	}

	// Verify all Location refs are rewritten to Address
	person := doc.Components.Schemas["Person"]
	if person.Properties["home"].Ref != "#/components/schemas/Address" {
		t.Errorf("Expected home.$ref = Address, got %s", person.Properties["home"].Ref)
	}
	if person.Properties["work"].Ref != "#/components/schemas/Address" {
		t.Errorf("Expected work.$ref = Address, got %s", person.Properties["work"].Ref)
	}

	employee := doc.Components.Schemas["Employee"]
	officeRef := employee.AllOf[1].Properties["office"].Ref
	if officeRef != "#/components/schemas/Address" {
		t.Errorf("Expected office.$ref = Address, got %s", officeRef)
	}
}

func TestBuilder_DeduplicateSchemas_ManualCall(t *testing.T) {
	// Test calling DeduplicateSchemas manually (without option)
	b := New(parser.OASVersion320)

	b.addSchema("Address", &parser.Schema{Type: "object"})
	b.addSchema("Location", &parser.Schema{Type: "object"})

	// Manually call deduplication
	b.DeduplicateSchemas()

	// Verify aliases are set
	if len(b.schemaAliases) != 1 {
		t.Errorf("Expected 1 alias, got %d", len(b.schemaAliases))
	}

	// Build should use the aliases for rewriting
	doc, err := b.BuildOAS3()
	if err != nil {
		t.Fatalf("BuildOAS3 failed: %v", err)
	}

	if len(doc.Components.Schemas) != 1 {
		t.Errorf("Expected 1 schema, got %d", len(doc.Components.Schemas))
	}
}
