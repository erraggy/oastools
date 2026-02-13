package joiner

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)

	oas3Doc, ok := joinResult.Document.(*parser.OAS3Document)
	require.True(t, ok, "Expected OAS3Document")

	// Should have 1 schema (Address is canonical, alphabetically first)
	assert.Len(t, oas3Doc.Components.Schemas, 1, "Expected 1 schema after dedup")

	_, ok = oas3Doc.Components.Schemas["Address"]
	assert.True(t, ok, "Expected Address to be canonical (alphabetically first)")

	_, ok = oas3Doc.Components.Schemas["Location"]
	assert.False(t, ok, "Expected Location to be removed (duplicate of Address)")
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
	require.NoError(t, err)

	oas2Doc, ok := joinResult.Document.(*parser.OAS2Document)
	require.True(t, ok, "Expected OAS2Document")

	// Should have 1 definition (Address is canonical, alphabetically first)
	assert.Len(t, oas2Doc.Definitions, 1, "Expected 1 definition after dedup")

	_, ok = oas2Doc.Definitions["Address"]
	assert.True(t, ok, "Expected Address to be canonical (alphabetically first)")

	_, ok = oas2Doc.Definitions["Location"]
	assert.False(t, ok, "Expected Location to be removed (duplicate of Address)")
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
	require.NoError(t, err)

	oas3Doc, ok := joinResult.Document.(*parser.OAS3Document)
	require.True(t, ok, "Expected OAS3Document")

	// Should have 2 schemas: Address (canonical) and Order
	assert.Len(t, oas3Doc.Components.Schemas, 2, "Expected 2 schemas after dedup")

	// Order's reference to Location should be rewritten to Address
	orderSchema := oas3Doc.Components.Schemas["Order"]
	require.NotNil(t, orderSchema, "Expected Order schema to exist")

	shipToRef := orderSchema.Properties["shipTo"].Ref
	expectedRef := "#/components/schemas/Address"
	assert.Equal(t, expectedRef, shipToRef)
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
	require.NoError(t, err)

	oas2Doc, ok := joinResult.Document.(*parser.OAS2Document)
	require.True(t, ok, "Expected OAS2Document")

	// Should have 2 definitions: Address (canonical) and Order
	assert.Len(t, oas2Doc.Definitions, 2, "Expected 2 definitions after dedup")

	// Order's reference to Location should be rewritten to Address
	orderSchema := oas2Doc.Definitions["Order"]
	require.NotNil(t, orderSchema, "Expected Order definition to exist")

	shipToRef := orderSchema.Properties["shipTo"].Ref
	expectedRef := "#/definitions/Address"
	assert.Equal(t, expectedRef, shipToRef)
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
	require.NoError(t, err)

	oas3Doc, ok := joinResult.Document.(*parser.OAS3Document)
	require.True(t, ok, "Expected OAS3Document")

	// Should have both schemas (no deduplication)
	assert.Len(t, oas3Doc.Components.Schemas, 2, "Expected 2 schemas (no dedup)")
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
	require.NoError(t, err)

	oas3Doc, ok := joinResult.Document.(*parser.OAS3Document)
	require.True(t, ok, "Expected OAS3Document")

	// Should have 3 schemas: Address (canonical), Name (canonical), Age (unique)
	assert.Len(t, oas3Doc.Components.Schemas, 3, "Expected 3 schemas")

	// Verify canonical names
	_, ok = oas3Doc.Components.Schemas["Address"]
	assert.True(t, ok, "Expected Address to be canonical")
	_, ok = oas3Doc.Components.Schemas["Name"]
	assert.True(t, ok, "Expected Name to be canonical")
	_, ok = oas3Doc.Components.Schemas["Age"]
	assert.True(t, ok, "Expected Age to be present")

	// Verify duplicates removed
	_, ok = oas3Doc.Components.Schemas["Location"]
	assert.False(t, ok, "Expected Location to be removed (duplicate of Address)")
	_, ok = oas3Doc.Components.Schemas["Title"]
	assert.False(t, ok, "Expected Title to be removed (duplicate of Name)")
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
	require.NoError(t, err)

	// Check that a warning was generated about deduplication
	found := false
	for _, w := range joinResult.Warnings {
		if strings.Contains(w, "semantic deduplication") && strings.Contains(w, "consolidated") {
			found = true
			break
		}
	}

	assert.True(t, found, "Expected warning about semantic deduplication consolidation")
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
	require.NoError(t, err)

	oas3Doc, ok := joinResult.Document.(*parser.OAS3Document)
	require.True(t, ok, "Expected OAS3Document")

	// Should have 1 schema after deduplication
	assert.Len(t, oas3Doc.Components.Schemas, 1, "Expected 1 schema after dedup")
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
	require.NoError(t, err)

	oas3Doc, ok := joinResult.Document.(*parser.OAS3Document)
	require.True(t, ok, "Expected OAS3Document")

	// Should deduplicate since structural properties are the same
	assert.Len(t, oas3Doc.Components.Schemas, 1, "Expected 1 schema (metadata ignored)")
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
	require.NoError(t, err)

	oas3Doc, ok := joinResult.Document.(*parser.OAS3Document)
	require.True(t, ok, "Expected OAS3Document")

	// Should have 3 schemas:
	// - AnyPayload (empty, preserved)
	// - DynamicData (empty, preserved - NOT deduplicated with AnyPayload)
	// - Person (canonical for Person/User group - alphabetically first)
	// User should be deduplicated into Person
	if !assert.Len(t, oas3Doc.Components.Schemas, 3, "Expected 3 schemas (empty schemas preserved)") {
		names := make([]string, 0, len(oas3Doc.Components.Schemas))
		for name := range oas3Doc.Components.Schemas {
			names = append(names, name)
		}
		t.Logf("Got schemas: %v", names)
	}

	// Verify empty schemas are preserved
	_, ok = oas3Doc.Components.Schemas["AnyPayload"]
	assert.True(t, ok, "Expected AnyPayload empty schema to be preserved")
	_, ok = oas3Doc.Components.Schemas["DynamicData"]
	assert.True(t, ok, "Expected DynamicData empty schema to be preserved")

	// Verify non-empty schema deduplication still works
	// Person is canonical (alphabetically first: "Person" < "User")
	_, ok = oas3Doc.Components.Schemas["Person"]
	assert.True(t, ok, "Expected Person to be canonical (alphabetically first vs User)")
	_, ok = oas3Doc.Components.Schemas["User"]
	assert.False(t, ok, "Expected User to be removed (duplicate of Person)")
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
	require.NoError(t, err)

	oas2Doc, ok := joinResult.Document.(*parser.OAS2Document)
	require.True(t, ok, "Expected OAS2Document")

	// Should have 3 definitions: both empty schemas preserved, User deduplicated into Person
	if !assert.Len(t, oas2Doc.Definitions, 3, "Expected 3 definitions (empty schemas preserved)") {
		names := make([]string, 0, len(oas2Doc.Definitions))
		for name := range oas2Doc.Definitions {
			names = append(names, name)
		}
		t.Logf("Got definitions: %v", names)
	}

	_, ok = oas2Doc.Definitions["AnyPayload"]
	assert.True(t, ok, "Expected AnyPayload empty schema to be preserved")
	_, ok = oas2Doc.Definitions["DynamicData"]
	assert.True(t, ok, "Expected DynamicData empty schema to be preserved")
	// Person is canonical (alphabetically first: "Person" < "User")
	_, ok = oas2Doc.Definitions["Person"]
	assert.True(t, ok, "Expected Person to be canonical")
	_, ok = oas2Doc.Definitions["User"]
	assert.False(t, ok, "Expected User to be removed (duplicate of Person)")
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
	require.NoError(t, err)

	oas3Doc, ok := joinResult.Document.(*parser.OAS3Document)
	require.True(t, ok, "Expected OAS3Document")

	// Both empty schemas should be preserved (not deduplicated)
	_, ok = oas3Doc.Components.Schemas["EmptyPlaceholder"]
	assert.True(t, ok, "Expected EmptyPlaceholder to be preserved")
	_, ok = oas3Doc.Components.Schemas["AnotherEmpty"]
	assert.True(t, ok, "Expected AnotherEmpty to be preserved")

	// Order's reference to AnotherEmpty should remain unchanged
	orderSchema := oas3Doc.Components.Schemas["Order"]
	require.NotNil(t, orderSchema, "Expected Order schema to exist")

	payloadRef := orderSchema.Properties["payload"].Ref
	expectedRef := "#/components/schemas/AnotherEmpty"
	assert.Equal(t, expectedRef, payloadRef, "reference should NOT be rewritten for empty schemas")
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
	require.NoError(t, err)

	oas3Doc, ok := joinResult.Document.(*parser.OAS3Document)
	require.True(t, ok, "Expected OAS3Document")

	// Both schemas should be preserved since they are empty (metadata-only)
	assert.Len(t, oas3Doc.Components.Schemas, 2, "Expected 2 schemas (empty schemas with metadata preserved)")

	_, ok = oas3Doc.Components.Schemas["Wildcard"]
	assert.True(t, ok, "Expected Wildcard empty schema to be preserved")
	_, ok = oas3Doc.Components.Schemas["Placeholder"]
	assert.True(t, ok, "Expected Placeholder empty schema to be preserved")
}
