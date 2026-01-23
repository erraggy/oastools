package joiner

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
)

func TestJoiner_SemanticDeduplication_OAS3(t *testing.T) {
	// Create two documents with identical schemas under different names
	doc1 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 1", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Address": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"street": {Type: "string"},
						"city":   {Type: "string"},
					},
					Required: []string{"street", "city"},
				},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	doc2 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 2", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				// Identical schema with different name
				"Location": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"street": {Type: "string"},
						"city":   {Type: "string"},
					},
					Required: []string{"street", "city"},
				},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	// Create parse results
	results := []parser.ParseResult{
		{
			Document:     doc1,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api1.yaml",
			SourceFormat: "yaml",
		},
		{
			Document:     doc2,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api2.yaml",
			SourceFormat: "yaml",
		},
	}

	// Join with semantic deduplication enabled
	config := DefaultConfig()
	config.SemanticDeduplication = true
	j := New(config)

	joinResult, err := j.JoinParsed(results)
	if err != nil {
		t.Fatalf("JoinParsed failed: %v", err)
	}

	oas3Doc, ok := joinResult.Document.(*parser.OAS3Document)
	if !ok {
		t.Fatal("Expected OAS3Document")
	}

	// Should have 1 schema (Address is canonical, alphabetically first)
	if len(oas3Doc.Components.Schemas) != 1 {
		t.Errorf("Expected 1 schema after dedup, got %d", len(oas3Doc.Components.Schemas))
	}

	if _, ok := oas3Doc.Components.Schemas["Address"]; !ok {
		t.Error("Expected Address to be canonical (alphabetically first)")
	}

	if _, ok := oas3Doc.Components.Schemas["Location"]; ok {
		t.Error("Expected Location to be removed (duplicate of Address)")
	}
}

func TestJoiner_SemanticDeduplication_OAS2(t *testing.T) {
	// Create two documents with identical schemas under different names
	doc1 := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "API 1", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Definitions: map[string]*parser.Schema{
			"Address": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"street": {Type: "string"},
					"city":   {Type: "string"},
				},
				Required: []string{"street", "city"},
			},
		},
		OASVersion: parser.OASVersion20,
	}

	doc2 := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "API 2", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Definitions: map[string]*parser.Schema{
			// Identical schema with different name
			"Location": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"street": {Type: "string"},
					"city":   {Type: "string"},
				},
				Required: []string{"street", "city"},
			},
		},
		OASVersion: parser.OASVersion20,
	}

	// Create parse results
	results := []parser.ParseResult{
		{
			Document:     doc1,
			Version:      "2.0",
			OASVersion:   parser.OASVersion20,
			SourcePath:   "api1.yaml",
			SourceFormat: "yaml",
		},
		{
			Document:     doc2,
			Version:      "2.0",
			OASVersion:   parser.OASVersion20,
			SourcePath:   "api2.yaml",
			SourceFormat: "yaml",
		},
	}

	// Join with semantic deduplication enabled
	config := DefaultConfig()
	config.SemanticDeduplication = true
	j := New(config)

	joinResult, err := j.JoinParsed(results)
	if err != nil {
		t.Fatalf("JoinParsed failed: %v", err)
	}

	oas2Doc, ok := joinResult.Document.(*parser.OAS2Document)
	if !ok {
		t.Fatal("Expected OAS2Document")
	}

	// Should have 1 definition (Address is canonical, alphabetically first)
	if len(oas2Doc.Definitions) != 1 {
		t.Errorf("Expected 1 definition after dedup, got %d", len(oas2Doc.Definitions))
	}

	if _, ok := oas2Doc.Definitions["Address"]; !ok {
		t.Error("Expected Address to be canonical (alphabetically first)")
	}

	if _, ok := oas2Doc.Definitions["Location"]; ok {
		t.Error("Expected Location to be removed (duplicate of Address)")
	}
}

func TestJoiner_SemanticDeduplication_ReferenceRewriting_OAS3(t *testing.T) {
	// Create documents where one references a schema that will be deduplicated
	doc1 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 1", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Address": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"street": {Type: "string"},
					},
				},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	doc2 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 2", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				// Identical to Address
				"Location": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"street": {Type: "string"},
					},
				},
				// References Location (which will be deduplicated to Address)
				"Order": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"shipTo": {Ref: "#/components/schemas/Location"},
					},
				},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	results := []parser.ParseResult{
		{
			Document:     doc1,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api1.yaml",
			SourceFormat: "yaml",
		},
		{
			Document:     doc2,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api2.yaml",
			SourceFormat: "yaml",
		},
	}

	config := DefaultConfig()
	config.SemanticDeduplication = true
	j := New(config)

	joinResult, err := j.JoinParsed(results)
	if err != nil {
		t.Fatalf("JoinParsed failed: %v", err)
	}

	oas3Doc, ok := joinResult.Document.(*parser.OAS3Document)
	if !ok {
		t.Fatal("Expected OAS3Document")
	}

	// Should have 2 schemas: Address (canonical) and Order
	if len(oas3Doc.Components.Schemas) != 2 {
		t.Errorf("Expected 2 schemas after dedup, got %d", len(oas3Doc.Components.Schemas))
	}

	// Order's reference to Location should be rewritten to Address
	orderSchema := oas3Doc.Components.Schemas["Order"]
	if orderSchema == nil {
		t.Fatal("Expected Order schema to exist")
	}

	shipToRef := orderSchema.Properties["shipTo"].Ref
	expectedRef := "#/components/schemas/Address"
	if shipToRef != expectedRef {
		t.Errorf("Expected shipTo.$ref = %s, got %s", expectedRef, shipToRef)
	}
}

func TestJoiner_SemanticDeduplication_ReferenceRewriting_OAS2(t *testing.T) {
	// Create documents where one references a schema that will be deduplicated
	doc1 := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "API 1", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Definitions: map[string]*parser.Schema{
			"Address": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"street": {Type: "string"},
				},
			},
		},
		OASVersion: parser.OASVersion20,
	}

	doc2 := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "API 2", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Definitions: map[string]*parser.Schema{
			// Identical to Address
			"Location": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"street": {Type: "string"},
				},
			},
			// References Location (which will be deduplicated to Address)
			"Order": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"shipTo": {Ref: "#/definitions/Location"},
				},
			},
		},
		OASVersion: parser.OASVersion20,
	}

	results := []parser.ParseResult{
		{
			Document:     doc1,
			Version:      "2.0",
			OASVersion:   parser.OASVersion20,
			SourcePath:   "api1.yaml",
			SourceFormat: "yaml",
		},
		{
			Document:     doc2,
			Version:      "2.0",
			OASVersion:   parser.OASVersion20,
			SourcePath:   "api2.yaml",
			SourceFormat: "yaml",
		},
	}

	config := DefaultConfig()
	config.SemanticDeduplication = true
	j := New(config)

	joinResult, err := j.JoinParsed(results)
	if err != nil {
		t.Fatalf("JoinParsed failed: %v", err)
	}

	oas2Doc, ok := joinResult.Document.(*parser.OAS2Document)
	if !ok {
		t.Fatal("Expected OAS2Document")
	}

	// Should have 2 definitions: Address (canonical) and Order
	if len(oas2Doc.Definitions) != 2 {
		t.Errorf("Expected 2 definitions after dedup, got %d", len(oas2Doc.Definitions))
	}

	// Order's reference to Location should be rewritten to Address
	orderSchema := oas2Doc.Definitions["Order"]
	if orderSchema == nil {
		t.Fatal("Expected Order definition to exist")
	}

	shipToRef := orderSchema.Properties["shipTo"].Ref
	expectedRef := "#/definitions/Address"
	if shipToRef != expectedRef {
		t.Errorf("Expected shipTo.$ref = %s, got %s", expectedRef, shipToRef)
	}
}

func TestJoiner_SemanticDeduplication_Disabled(t *testing.T) {
	// Create two documents with identical schemas
	doc1 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 1", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Address": {Type: "object"},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	doc2 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 2", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Location": {Type: "object"},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	results := []parser.ParseResult{
		{
			Document:     doc1,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api1.yaml",
			SourceFormat: "yaml",
		},
		{
			Document:     doc2,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api2.yaml",
			SourceFormat: "yaml",
		},
	}

	// Default config - deduplication disabled
	config := DefaultConfig()
	j := New(config)

	joinResult, err := j.JoinParsed(results)
	if err != nil {
		t.Fatalf("JoinParsed failed: %v", err)
	}

	oas3Doc, ok := joinResult.Document.(*parser.OAS3Document)
	if !ok {
		t.Fatal("Expected OAS3Document")
	}

	// Should have both schemas (no deduplication)
	if len(oas3Doc.Components.Schemas) != 2 {
		t.Errorf("Expected 2 schemas (no dedup), got %d", len(oas3Doc.Components.Schemas))
	}
}

func TestJoiner_SemanticDeduplication_MultipleGroups(t *testing.T) {
	// Create documents with multiple equivalence groups
	doc1 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 1", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				// Group 1: objects with name property
				"Address": {
					Type:       "object",
					Properties: map[string]*parser.Schema{"name": {Type: "string"}},
				},
				// Group 2: simple strings
				"Name": {Type: "string"},
				// Unique
				"Age": {Type: "integer"},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	doc2 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 2", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				// Group 1: identical to Address
				"Location": {
					Type:       "object",
					Properties: map[string]*parser.Schema{"name": {Type: "string"}},
				},
				// Group 2: identical to Name
				"Title": {Type: "string"},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	results := []parser.ParseResult{
		{
			Document:     doc1,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api1.yaml",
			SourceFormat: "yaml",
		},
		{
			Document:     doc2,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api2.yaml",
			SourceFormat: "yaml",
		},
	}

	config := DefaultConfig()
	config.SemanticDeduplication = true
	j := New(config)

	joinResult, err := j.JoinParsed(results)
	if err != nil {
		t.Fatalf("JoinParsed failed: %v", err)
	}

	oas3Doc, ok := joinResult.Document.(*parser.OAS3Document)
	if !ok {
		t.Fatal("Expected OAS3Document")
	}

	// Should have 3 schemas: Address (canonical), Name (canonical), Age (unique)
	if len(oas3Doc.Components.Schemas) != 3 {
		t.Errorf("Expected 3 schemas, got %d", len(oas3Doc.Components.Schemas))
	}

	// Verify canonical names
	if _, ok := oas3Doc.Components.Schemas["Address"]; !ok {
		t.Error("Expected Address to be canonical")
	}
	if _, ok := oas3Doc.Components.Schemas["Name"]; !ok {
		t.Error("Expected Name to be canonical")
	}
	if _, ok := oas3Doc.Components.Schemas["Age"]; !ok {
		t.Error("Expected Age to be present")
	}

	// Verify duplicates removed
	if _, ok := oas3Doc.Components.Schemas["Location"]; ok {
		t.Error("Expected Location to be removed (duplicate of Address)")
	}
	if _, ok := oas3Doc.Components.Schemas["Title"]; ok {
		t.Error("Expected Title to be removed (duplicate of Name)")
	}
}

func TestJoiner_SemanticDeduplication_WarningsGenerated(t *testing.T) {
	// Create two documents with identical schemas
	doc1 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 1", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Address": {Type: "object"},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	doc2 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 2", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Location": {Type: "object"},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	results := []parser.ParseResult{
		{
			Document:     doc1,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api1.yaml",
			SourceFormat: "yaml",
		},
		{
			Document:     doc2,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api2.yaml",
			SourceFormat: "yaml",
		},
	}

	config := DefaultConfig()
	config.SemanticDeduplication = true
	j := New(config)

	joinResult, err := j.JoinParsed(results)
	if err != nil {
		t.Fatalf("JoinParsed failed: %v", err)
	}

	// Check that a warning was generated about deduplication
	found := false
	for _, w := range joinResult.Warnings {
		if strings.Contains(w, "semantic deduplication") && strings.Contains(w, "consolidated") {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected warning about semantic deduplication consolidation")
	}
}

func TestJoiner_WithSemanticDeduplication_Option(t *testing.T) {
	// Create two documents with identical schemas
	doc1 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 1", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Address": {Type: "object"},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	doc2 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 2", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Location": {Type: "object"},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	results := []parser.ParseResult{
		{
			Document:     doc1,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api1.yaml",
			SourceFormat: "yaml",
		},
		{
			Document:     doc2,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api2.yaml",
			SourceFormat: "yaml",
		},
	}

	// Use functional option
	joinResult, err := JoinWithOptions(
		WithParsed(results...),
		WithSemanticDeduplication(true),
	)
	if err != nil {
		t.Fatalf("JoinWithOptions failed: %v", err)
	}

	oas3Doc, ok := joinResult.Document.(*parser.OAS3Document)
	if !ok {
		t.Fatal("Expected OAS3Document")
	}

	// Should have 1 schema after deduplication
	if len(oas3Doc.Components.Schemas) != 1 {
		t.Errorf("Expected 1 schema after dedup, got %d", len(oas3Doc.Components.Schemas))
	}
}

func TestJoiner_SemanticDeduplication_MetadataIgnored(t *testing.T) {
	// Create two documents with schemas that differ only in metadata
	doc1 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 1", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Address": {
					Type:        "object",
					Title:       "An Address",
					Description: "Represents a physical address",
					Properties: map[string]*parser.Schema{
						"street": {Type: "string"},
					},
				},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	doc2 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 2", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Location": {
					Type:        "object",
					Title:       "A Location",
					Description: "A place on earth",
					Properties: map[string]*parser.Schema{
						"street": {Type: "string"},
					},
				},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	results := []parser.ParseResult{
		{
			Document:     doc1,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api1.yaml",
			SourceFormat: "yaml",
		},
		{
			Document:     doc2,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api2.yaml",
			SourceFormat: "yaml",
		},
	}

	config := DefaultConfig()
	config.SemanticDeduplication = true
	j := New(config)

	joinResult, err := j.JoinParsed(results)
	if err != nil {
		t.Fatalf("JoinParsed failed: %v", err)
	}

	oas3Doc, ok := joinResult.Document.(*parser.OAS3Document)
	if !ok {
		t.Fatal("Expected OAS3Document")
	}

	// Should deduplicate since structural properties are the same
	if len(oas3Doc.Components.Schemas) != 1 {
		t.Errorf("Expected 1 schema (metadata ignored), got %d", len(oas3Doc.Components.Schemas))
	}
}

func TestJoiner_SemanticDeduplication_EmptySchemasPreserved(t *testing.T) {
	// Create two documents with empty schemas (no constraints).
	// Empty schemas serve different semantic purposes and should NOT be consolidated.
	doc1 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 1", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				// Empty schema used as a placeholder
				"AnyPayload": {},
				// Non-empty schema with a real type
				"User": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name": {Type: "string"},
					},
				},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	doc2 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 2", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				// Another empty schema used as a wildcard type
				"DynamicData": {},
				// Schema identical to User - should be deduplicated
				"Person": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name": {Type: "string"},
					},
				},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	results := []parser.ParseResult{
		{
			Document:     doc1,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api1.yaml",
			SourceFormat: "yaml",
		},
		{
			Document:     doc2,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api2.yaml",
			SourceFormat: "yaml",
		},
	}

	config := DefaultConfig()
	config.SemanticDeduplication = true
	j := New(config)

	joinResult, err := j.JoinParsed(results)
	if err != nil {
		t.Fatalf("JoinParsed failed: %v", err)
	}

	oas3Doc, ok := joinResult.Document.(*parser.OAS3Document)
	if !ok {
		t.Fatal("Expected OAS3Document")
	}

	// Should have 3 schemas:
	// - AnyPayload (empty, preserved)
	// - DynamicData (empty, preserved - NOT deduplicated with AnyPayload)
	// - Person (canonical for Person/User group - alphabetically first)
	// User should be deduplicated into Person
	if len(oas3Doc.Components.Schemas) != 3 {
		names := make([]string, 0, len(oas3Doc.Components.Schemas))
		for name := range oas3Doc.Components.Schemas {
			names = append(names, name)
		}
		t.Errorf("Expected 3 schemas (empty schemas preserved), got %d: %v", len(oas3Doc.Components.Schemas), names)
	}

	// Verify empty schemas are preserved
	if _, ok := oas3Doc.Components.Schemas["AnyPayload"]; !ok {
		t.Error("Expected AnyPayload empty schema to be preserved")
	}
	if _, ok := oas3Doc.Components.Schemas["DynamicData"]; !ok {
		t.Error("Expected DynamicData empty schema to be preserved")
	}

	// Verify non-empty schema deduplication still works
	// Person is canonical (alphabetically first: "Person" < "User")
	if _, ok := oas3Doc.Components.Schemas["Person"]; !ok {
		t.Error("Expected Person to be canonical (alphabetically first vs User)")
	}
	if _, ok := oas3Doc.Components.Schemas["User"]; ok {
		t.Error("Expected User to be removed (duplicate of Person)")
	}
}

func TestJoiner_SemanticDeduplication_EmptySchemasPreserved_OAS2(t *testing.T) {
	// Same test for OAS 2.0 documents
	doc1 := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "API 1", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Definitions: map[string]*parser.Schema{
			"AnyPayload": {},
			"User": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"name": {Type: "string"},
				},
			},
		},
		OASVersion: parser.OASVersion20,
	}

	doc2 := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "API 2", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Definitions: map[string]*parser.Schema{
			"DynamicData": {},
			"Person": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"name": {Type: "string"},
				},
			},
		},
		OASVersion: parser.OASVersion20,
	}

	results := []parser.ParseResult{
		{
			Document:     doc1,
			Version:      "2.0",
			OASVersion:   parser.OASVersion20,
			SourcePath:   "api1.yaml",
			SourceFormat: "yaml",
		},
		{
			Document:     doc2,
			Version:      "2.0",
			OASVersion:   parser.OASVersion20,
			SourcePath:   "api2.yaml",
			SourceFormat: "yaml",
		},
	}

	config := DefaultConfig()
	config.SemanticDeduplication = true
	j := New(config)

	joinResult, err := j.JoinParsed(results)
	if err != nil {
		t.Fatalf("JoinParsed failed: %v", err)
	}

	oas2Doc, ok := joinResult.Document.(*parser.OAS2Document)
	if !ok {
		t.Fatal("Expected OAS2Document")
	}

	// Should have 3 definitions: both empty schemas preserved, User deduplicated into Person
	if len(oas2Doc.Definitions) != 3 {
		names := make([]string, 0, len(oas2Doc.Definitions))
		for name := range oas2Doc.Definitions {
			names = append(names, name)
		}
		t.Errorf("Expected 3 definitions (empty schemas preserved), got %d: %v", len(oas2Doc.Definitions), names)
	}

	if _, ok := oas2Doc.Definitions["AnyPayload"]; !ok {
		t.Error("Expected AnyPayload empty schema to be preserved")
	}
	if _, ok := oas2Doc.Definitions["DynamicData"]; !ok {
		t.Error("Expected DynamicData empty schema to be preserved")
	}
	// Person is canonical (alphabetically first: "Person" < "User")
	if _, ok := oas2Doc.Definitions["Person"]; !ok {
		t.Error("Expected Person to be canonical")
	}
	if _, ok := oas2Doc.Definitions["User"]; ok {
		t.Error("Expected User to be removed (duplicate of Person)")
	}
}

func TestJoiner_SemanticDeduplication_EmptySchemaReferenceRewriting(t *testing.T) {
	// Verify that references to empty schemas are NOT rewritten during deduplication.
	// Empty schemas are preserved as-is, so references should remain unchanged.
	doc1 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 1", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				// Empty placeholder schema
				"EmptyPlaceholder": {},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	doc2 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 2", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				// Another empty schema
				"AnotherEmpty": {},
				// Order schema that references AnotherEmpty
				"Order": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"payload": {Ref: "#/components/schemas/AnotherEmpty"},
					},
				},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	results := []parser.ParseResult{
		{
			Document:     doc1,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api1.yaml",
			SourceFormat: "yaml",
		},
		{
			Document:     doc2,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api2.yaml",
			SourceFormat: "yaml",
		},
	}

	config := DefaultConfig()
	config.SemanticDeduplication = true
	j := New(config)

	joinResult, err := j.JoinParsed(results)
	if err != nil {
		t.Fatalf("JoinParsed failed: %v", err)
	}

	oas3Doc, ok := joinResult.Document.(*parser.OAS3Document)
	if !ok {
		t.Fatal("Expected OAS3Document")
	}

	// Both empty schemas should be preserved (not deduplicated)
	if _, ok := oas3Doc.Components.Schemas["EmptyPlaceholder"]; !ok {
		t.Error("Expected EmptyPlaceholder to be preserved")
	}
	if _, ok := oas3Doc.Components.Schemas["AnotherEmpty"]; !ok {
		t.Error("Expected AnotherEmpty to be preserved")
	}

	// Order's reference to AnotherEmpty should remain unchanged
	orderSchema := oas3Doc.Components.Schemas["Order"]
	if orderSchema == nil {
		t.Fatal("Expected Order schema to exist")
	}

	payloadRef := orderSchema.Properties["payload"].Ref
	expectedRef := "#/components/schemas/AnotherEmpty"
	if payloadRef != expectedRef {
		t.Errorf("Expected payload.$ref = %s, got %s (reference should NOT be rewritten for empty schemas)", expectedRef, payloadRef)
	}
}

func TestJoiner_SemanticDeduplication_EmptyWithMetadataPreserved(t *testing.T) {
	// Empty schemas with metadata (title, description) should also be preserved
	doc1 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 1", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Wildcard": {
					Title:       "Any Type",
					Description: "Accepts any value",
				},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	doc2 := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "API 2", Version: "1.0.0"},
		Paths:   make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Placeholder": {
					Title:       "Placeholder",
					Description: "To be defined later",
				},
			},
		},
		OASVersion: parser.OASVersion303,
	}

	results := []parser.ParseResult{
		{
			Document:     doc1,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api1.yaml",
			SourceFormat: "yaml",
		},
		{
			Document:     doc2,
			Version:      "3.0.3",
			OASVersion:   parser.OASVersion303,
			SourcePath:   "api2.yaml",
			SourceFormat: "yaml",
		},
	}

	config := DefaultConfig()
	config.SemanticDeduplication = true
	j := New(config)

	joinResult, err := j.JoinParsed(results)
	if err != nil {
		t.Fatalf("JoinParsed failed: %v", err)
	}

	oas3Doc, ok := joinResult.Document.(*parser.OAS3Document)
	if !ok {
		t.Fatal("Expected OAS3Document")
	}

	// Both schemas should be preserved since they are empty (metadata-only)
	if len(oas3Doc.Components.Schemas) != 2 {
		t.Errorf("Expected 2 schemas (empty schemas with metadata preserved), got %d", len(oas3Doc.Components.Schemas))
	}

	if _, ok := oas3Doc.Components.Schemas["Wildcard"]; !ok {
		t.Error("Expected Wildcard empty schema to be preserved")
	}
	if _, ok := oas3Doc.Components.Schemas["Placeholder"]; !ok {
		t.Error("Expected Placeholder empty schema to be preserved")
	}
}
