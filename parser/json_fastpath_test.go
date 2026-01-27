package parser

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJSONFastPath_OAS2 verifies that OAS 2.0 JSON documents are parsed correctly via the fast path
func TestJSONFastPath_OAS2(t *testing.T) {
	// JSON document that should trigger the fast path
	jsonData := []byte(`{
		"swagger": "2.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0",
			"description": "A test API"
		},
		"host": "api.example.com",
		"basePath": "/v1",
		"schemes": ["https"],
		"paths": {
			"/pets": {
				"get": {
					"operationId": "listPets",
					"summary": "List all pets",
					"responses": {
						"200": {
							"description": "A list of pets"
						}
					}
				}
			}
		}
	}`)

	result, err := ParseWithOptions(WithBytes(jsonData))
	require.NoError(t, err)

	// Verify format detection
	assert.Equal(t, SourceFormatJSON, result.SourceFormat)
	assert.Equal(t, "2.0", result.Version)
	assert.Equal(t, OASVersion20, result.OASVersion)

	// Verify document structure
	doc, ok := result.OAS2Document()
	require.True(t, ok, "expected OAS2Document")

	assert.Equal(t, "2.0", doc.Swagger)
	assert.Equal(t, "Test API", doc.Info.Title)
	assert.Equal(t, "1.0.0", doc.Info.Version)
	assert.Equal(t, "api.example.com", doc.Host)
	assert.Equal(t, "/v1", doc.BasePath)
	assert.Contains(t, doc.Schemes, "https")

	// Verify paths
	require.NotNil(t, doc.Paths)
	pathItem, exists := doc.Paths["/pets"]
	require.True(t, exists)
	require.NotNil(t, pathItem.Get)
	assert.Equal(t, "listPets", pathItem.Get.OperationID)
}

// TestJSONFastPath_OAS3 verifies that OAS 3.x JSON documents are parsed correctly via the fast path
func TestJSONFastPath_OAS3(t *testing.T) {
	tests := []struct {
		name    string
		version string
	}{
		{"OAS 3.0.0", "3.0.0"},
		{"OAS 3.0.3", "3.0.3"},
		{"OAS 3.1.0", "3.1.0"},
		{"OAS 3.2.0", "3.2.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData := []byte(`{
				"openapi": "` + tt.version + `",
				"info": {
					"title": "Test API",
					"version": "1.0.0"
				},
				"servers": [
					{"url": "https://api.example.com/v1"}
				],
				"paths": {
					"/users": {
						"get": {
							"operationId": "listUsers",
							"responses": {
								"200": {
									"description": "Success"
								}
							}
						}
					}
				}
			}`)

			result, err := ParseWithOptions(WithBytes(jsonData))
			require.NoError(t, err)

			assert.Equal(t, SourceFormatJSON, result.SourceFormat)
			assert.Equal(t, tt.version, result.Version)

			doc, ok := result.OAS3Document()
			require.True(t, ok, "expected OAS3Document")

			assert.Equal(t, tt.version, doc.OpenAPI)
			assert.Equal(t, "Test API", doc.Info.Title)
			require.Len(t, doc.Servers, 1)
			assert.Equal(t, "https://api.example.com/v1", doc.Servers[0].URL)
		})
	}
}

// TestJSONFastPath_WithExtensions verifies that specification extensions (x-*) are preserved
func TestJSONFastPath_WithExtensions(t *testing.T) {
	jsonData := []byte(`{
		"openapi": "3.0.3",
		"info": {
			"title": "Test API",
			"version": "1.0.0",
			"x-custom-field": "custom-value"
		},
		"x-api-id": "test-api-001",
		"paths": {}
	}`)

	result, err := ParseWithOptions(WithBytes(jsonData))
	require.NoError(t, err)

	doc, ok := result.OAS3Document()
	require.True(t, ok)

	// Verify extension in Info
	require.NotNil(t, doc.Info.Extra)
	assert.Equal(t, "custom-value", doc.Info.Extra["x-custom-field"])

	// Verify extension in document root
	require.NotNil(t, doc.Extra)
	assert.Equal(t, "test-api-001", doc.Extra["x-api-id"])
}

// TestJSONFastPath_NotTriggeredForYAML verifies that YAML input does NOT use the fast path
func TestJSONFastPath_NotTriggeredForYAML(t *testing.T) {
	yamlData := []byte(`
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0.0"
paths: {}
`)

	result, err := ParseWithOptions(WithBytes(yamlData))
	require.NoError(t, err)

	// YAML should be detected, not JSON
	assert.Equal(t, SourceFormatYAML, result.SourceFormat)
	assert.Equal(t, "3.0.3", result.Version)
}

// TestJSONFastPath_BypassedWhenSourceMapEnabled verifies that the fast path is bypassed
// when BuildSourceMap is enabled (source maps require YAML node tracking)
func TestJSONFastPath_BypassedWhenSourceMapEnabled(t *testing.T) {
	jsonData := []byte(`{
		"openapi": "3.0.3",
		"info": {"title": "Test", "version": "1.0.0"},
		"paths": {}
	}`)

	result, err := ParseWithOptions(
		WithBytes(jsonData),
		WithSourceMap(true),
	)
	require.NoError(t, err)

	// Document should still parse correctly
	assert.Equal(t, "3.0.3", result.Version)

	// Source map should be built (which means YAML path was used)
	require.NotNil(t, result.SourceMap)
}

// TestJSONFastPath_BypassedWhenPreserveOrderEnabled verifies that the fast path is bypassed
// when PreserveOrder is enabled (order preservation requires YAML node tracking)
func TestJSONFastPath_BypassedWhenPreserveOrderEnabled(t *testing.T) {
	jsonData := []byte(`{
		"openapi": "3.0.3",
		"info": {"title": "Test", "version": "1.0.0"},
		"paths": {}
	}`)

	result, err := ParseWithOptions(
		WithBytes(jsonData),
		WithPreserveOrder(true),
	)
	require.NoError(t, err)

	// Document should still parse correctly
	assert.Equal(t, "3.0.3", result.Version)
}

// TestJSONFastPath_InvalidJSON verifies proper error handling for malformed JSON
func TestJSONFastPath_InvalidJSON(t *testing.T) {
	invalidJSON := []byte(`{
		"openapi": "3.0.3",
		"info": {"title": "Test", "version": "1.0.0"
	}`) // Missing closing braces

	_, err := ParseWithOptions(WithBytes(invalidJSON))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse JSON")
}

// TestJSONFastPath_MissingVersion verifies error handling when version field is missing
func TestJSONFastPath_MissingVersion(t *testing.T) {
	// Valid JSON but missing swagger/openapi field
	jsonData := []byte(`{
		"info": {"title": "Test", "version": "1.0.0"},
		"paths": {}
	}`)

	_, err := ParseWithOptions(WithBytes(jsonData))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unable to detect OpenAPI version")
}

// TestJSONFastPath_WithResolveRefs verifies that reference resolution works in fast path
func TestJSONFastPath_WithResolveRefs(t *testing.T) {
	jsonData := []byte(`{
		"openapi": "3.0.3",
		"info": {"title": "Test", "version": "1.0.0"},
		"paths": {
			"/pets": {
				"get": {
					"responses": {
						"200": {
							"description": "Success",
							"content": {
								"application/json": {
									"schema": {"$ref": "#/components/schemas/Pet"}
								}
							}
						}
					}
				}
			}
		},
		"components": {
			"schemas": {
				"Pet": {
					"type": "object",
					"properties": {
						"name": {"type": "string"}
					}
				}
			}
		}
	}`)

	result, err := ParseWithOptions(
		WithBytes(jsonData),
		WithResolveRefs(true),
	)
	require.NoError(t, err)

	assert.Equal(t, SourceFormatJSON, result.SourceFormat)
	assert.Equal(t, "3.0.3", result.Version)

	// The document should parse without errors
	doc, ok := result.OAS3Document()
	require.True(t, ok)
	require.NotNil(t, doc.Components)
	require.NotNil(t, doc.Components.Schemas)
	_, hasPet := doc.Components.Schemas["Pet"]
	assert.True(t, hasPet)
}

// TestJSONFastPath_EquivalentToYAMLPath verifies that JSON fast path produces
// the same result as parsing the equivalent YAML through the YAML path
func TestJSONFastPath_EquivalentToYAMLPath(t *testing.T) {
	jsonData := []byte(`{
		"openapi": "3.0.3",
		"info": {
			"title": "Equivalence Test API",
			"version": "2.0.0",
			"description": "Testing JSON vs YAML parsing equivalence"
		},
		"servers": [
			{"url": "https://api.example.com", "description": "Production"}
		],
		"paths": {
			"/items": {
				"get": {
					"operationId": "getItems",
					"summary": "Get all items",
					"responses": {
						"200": {
							"description": "Successful response"
						}
					}
				}
			}
		}
	}`)

	// Parse via JSON fast path
	jsonResult, err := ParseWithOptions(WithBytes(jsonData))
	require.NoError(t, err)

	// Force YAML path by enabling source map (which bypasses JSON fast path)
	yamlResult, err := ParseWithOptions(
		WithBytes(jsonData),
		WithSourceMap(true),
	)
	require.NoError(t, err)

	// Compare the parsed documents
	jsonDoc, ok := jsonResult.OAS3Document()
	require.True(t, ok)

	yamlDoc, ok := yamlResult.OAS3Document()
	require.True(t, ok)

	// Verify equivalent parsing
	assert.Equal(t, yamlDoc.OpenAPI, jsonDoc.OpenAPI)
	assert.Equal(t, yamlDoc.Info.Title, jsonDoc.Info.Title)
	assert.Equal(t, yamlDoc.Info.Version, jsonDoc.Info.Version)
	assert.Equal(t, yamlDoc.Info.Description, jsonDoc.Info.Description)
	require.Len(t, jsonDoc.Servers, len(yamlDoc.Servers))
	assert.Equal(t, yamlDoc.Servers[0].URL, jsonDoc.Servers[0].URL)

	// Verify paths are equivalent
	require.Equal(t, len(yamlDoc.Paths), len(jsonDoc.Paths))
	jsonPathItem := jsonDoc.Paths["/items"]
	yamlPathItem := yamlDoc.Paths["/items"]
	require.NotNil(t, jsonPathItem)
	require.NotNil(t, yamlPathItem)
	assert.Equal(t, yamlPathItem.Get.OperationID, jsonPathItem.Get.OperationID)
}

// TestJSONFastPath_File verifies parsing JSON files via the fast path
func TestJSONFastPath_File(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		version     string
		expectOAS2  bool
		expectTitle string
	}{
		{
			name:        "Minimal OAS3 JSON",
			path:        "../testdata/minimal-oas3.json",
			version:     "3.0.0",
			expectOAS2:  false,
			expectTitle: "Minimal API",
		},
		{
			name:        "Minimal OAS2 JSON",
			path:        "../testdata/minimal-oas2.json",
			version:     "2.0",
			expectOAS2:  true,
			expectTitle: "Minimal API",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseWithOptions(WithFilePath(tt.path))
			require.NoError(t, err)

			assert.Equal(t, SourceFormatJSON, result.SourceFormat)
			assert.Equal(t, tt.version, result.Version)

			if tt.expectOAS2 {
				doc, ok := result.OAS2Document()
				require.True(t, ok)
				assert.Equal(t, tt.expectTitle, doc.Info.Title)
			} else {
				doc, ok := result.OAS3Document()
				require.True(t, ok)
				assert.Equal(t, tt.expectTitle, doc.Info.Title)
			}
		})
	}
}

// BenchmarkJSONFastPath compares JSON parsing with and without the fast path
func BenchmarkJSONFastPath(b *testing.B) {
	// Create a moderately sized JSON document for benchmarking
	jsonData := createBenchmarkJSON()

	b.Run("FastPath", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_, err := ParseWithOptions(WithBytes(jsonData))
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}
		}
	})

	b.Run("YAMLPath_SourceMap", func(b *testing.B) {
		// Force YAML path by enabling source map
		b.ReportAllocs()
		for b.Loop() {
			_, err := ParseWithOptions(
				WithBytes(jsonData),
				WithSourceMap(true),
			)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}
		}
	})

	b.Run("YAMLPath_PreserveOrder", func(b *testing.B) {
		// Force YAML path by enabling preserve order
		b.ReportAllocs()
		for b.Loop() {
			_, err := ParseWithOptions(
				WithBytes(jsonData),
				WithPreserveOrder(true),
			)
			if err != nil {
				b.Fatalf("Failed to parse: %v", err)
			}
		}
	})
}

// createBenchmarkJSON creates a JSON document with multiple paths and schemas for benchmarking
func createBenchmarkJSON() []byte {
	doc := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "Benchmark API",
			"version":     "1.0.0",
			"description": "A benchmark API with multiple endpoints",
		},
		"servers": []map[string]any{
			{"url": "https://api.example.com/v1"},
		},
		"paths": map[string]any{},
		"components": map[string]any{
			"schemas": map[string]any{},
		},
	}

	paths := doc["paths"].(map[string]any)
	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)

	// Add 50 paths with operations
	for i := 0; i < 50; i++ {
		pathName := "/resource" + string(rune('A'+i%26)) + "/" + string(rune('0'+i%10))
		paths[pathName] = map[string]any{
			"get": map[string]any{
				"operationId": "get" + string(rune('A'+i%26)),
				"summary":     "Get resource",
				"responses": map[string]any{
					"200": map[string]any{
						"description": "Success",
					},
				},
			},
		}
	}

	// Add 20 schemas
	for i := 0; i < 20; i++ {
		schemaName := "Schema" + string(rune('A'+i%26))
		schemas[schemaName] = map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id":   map[string]any{"type": "string"},
				"name": map[string]any{"type": "string"},
				"data": map[string]any{"type": "object"},
			},
		}
	}

	data, _ := json.Marshal(doc)
	return data
}
