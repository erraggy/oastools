// Package testutil provides test utilities and fixtures for unit tests.
package testutil

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v4"

	"github.com/erraggy/oastools/parser"
)

// NewSimpleOAS2Document creates a minimal OAS 2.0 document for testing.
// Contains only required fields: swagger, info, host, basePath, schemes, paths.
func NewSimpleOAS2Document() *parser.OAS2Document {
	return &parser.OAS2Document{
		Swagger:    "2.0",
		OASVersion: parser.OASVersion20,
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Host:     "api.example.com",
		BasePath: "/v1",
		Schemes:  []string{"https"},
		Paths:    make(map[string]*parser.PathItem),
	}
}

// NewDetailedOAS2Document creates a complete OAS 2.0 document with common features for testing.
// Includes paths, operations, schemas, and definitions.
func NewDetailedOAS2Document() *parser.OAS2Document {
	doc := NewSimpleOAS2Document()
	doc.Definitions = map[string]*parser.Schema{
		"Pet": {
			Type: "object",
			Properties: map[string]*parser.Schema{
				"id":   {Type: "integer"},
				"name": {Type: "string"},
			},
		},
	}
	doc.Paths = map[string]*parser.PathItem{
		"/pets": {
			Get: &parser.Operation{
				Summary:     "List pets",
				OperationID: "listPets",
			},
		},
	}
	return doc
}

// NewSimpleOAS3Document creates a minimal OAS 3.x document for testing.
// Contains only required fields: openapi, info, paths, servers.
func NewSimpleOAS3Document() *parser.OAS3Document {
	return &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Servers: []*parser.Server{
			{
				URL:         "https://api.example.com/v1",
				Description: "Production server",
			},
		},
		Paths: make(map[string]*parser.PathItem),
	}
}

// NewDetailedOAS3Document creates a complete OAS 3.x document with common features for testing.
// Includes paths, operations, schemas, and components.
func NewDetailedOAS3Document() *parser.OAS3Document {
	return &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Servers: []*parser.Server{
			{
				URL:         "https://api.example.com/v1",
				Description: "Production server",
			},
		},
		Paths: map[string]*parser.PathItem{
			"/pets": {
				Get: &parser.Operation{
					Summary:     "List pets",
					OperationID: "listPets",
				},
			},
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"id":   {Type: "integer"},
						"name": {Type: "string"},
					},
				},
			},
		},
	}
}

// WriteTempYAML marshals a document to YAML and writes it to a temporary file.
// Returns the path to the temporary file.
// The file is automatically cleaned up when the test completes (via t.TempDir).
func WriteTempYAML(t *testing.T, doc any) string {
	t.Helper()

	data, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal document to YAML: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "test.yaml")
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		t.Fatalf("Failed to write temporary YAML file: %v", err)
	}

	return tmpFile
}

// WriteTempJSON marshals a document to JSON and writes it to a temporary file.
// Returns the path to the temporary file.
// The file is automatically cleaned up when the test completes (via t.TempDir).
func WriteTempJSON(t *testing.T, doc any) string {
	t.Helper()

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal document to JSON: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "test.json")
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		t.Fatalf("Failed to write temporary JSON file: %v", err)
	}

	return tmpFile
}
