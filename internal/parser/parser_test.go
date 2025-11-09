package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseOAS2(t *testing.T) {
	parser := New()
	result, err := parser.Parse("../../testdata/petstore-2.0.yaml")
	if err != nil {
		t.Fatalf("Failed to parse OAS 2.0 file: %v", err)
	}

	if result.Version != "2.0" {
		t.Errorf("Expected version 2.0, got %s", result.Version)
	}

	doc, ok := result.Document.(*OAS2Document)
	if !ok {
		t.Fatalf("Expected OAS2Document, got %T", result.Document)
	}

	if doc.Info == nil {
		t.Fatal("Info should not be nil")
	}

	if doc.Info.Title != "Petstore API" {
		t.Errorf("Expected title 'Petstore API', got '%s'", doc.Info.Title)
	}

	if doc.Info.Version != "1.0.0" {
		t.Errorf("Expected info version '1.0.0', got '%s'", doc.Info.Version)
	}

	if len(result.Errors) > 0 {
		t.Errorf("Unexpected validation errors: %v", result.Errors)
	}
}

func TestParseOAS30(t *testing.T) {
	parser := New()
	result, err := parser.Parse("../../testdata/petstore-3.0.yaml")
	if err != nil {
		t.Fatalf("Failed to parse OAS 3.0 file: %v", err)
	}

	if result.Version != "3.0.3" {
		t.Errorf("Expected version 3.0.3, got %s", result.Version)
	}

	doc, ok := result.Document.(*OAS3Document)
	if !ok {
		t.Fatalf("Expected OAS3Document, got %T", result.Document)
	}

	if doc.Info == nil {
		t.Fatal("Info should not be nil")
	}

	if doc.Info.Title != "Petstore API" {
		t.Errorf("Expected title 'Petstore API', got '%s'", doc.Info.Title)
	}

	if len(result.Errors) > 0 {
		t.Errorf("Unexpected validation errors: %v", result.Errors)
	}
}

func TestParseOAS31(t *testing.T) {
	parser := New()
	result, err := parser.Parse("../../testdata/petstore-3.1.yaml")
	if err != nil {
		t.Fatalf("Failed to parse OAS 3.1 file: %v", err)
	}

	if result.Version != "3.1.0" {
		t.Errorf("Expected version 3.1.0, got %s", result.Version)
	}

	doc, ok := result.Document.(*OAS3Document)
	if !ok {
		t.Fatalf("Expected OAS3Document, got %T", result.Document)
	}

	if doc.Info == nil {
		t.Fatal("Info should not be nil")
	}

	if doc.Info.Summary != "A modern pet store API" {
		t.Errorf("Expected summary 'A modern pet store API', got '%s'", doc.Info.Summary)
	}

	if doc.JSONSchemaDialect == "" {
		t.Error("Expected JSONSchemaDialect to be set")
	}

	if len(result.Errors) > 0 {
		t.Errorf("Unexpected validation errors: %v", result.Errors)
	}
}

func TestParseOAS32(t *testing.T) {
	parser := New()
	result, err := parser.Parse("../../testdata/petstore-3.2.yaml")
	if err != nil {
		t.Fatalf("Failed to parse OAS 3.2 file: %v", err)
	}

	if result.Version != "3.2.0" {
		t.Errorf("Expected version 3.2.0, got %s", result.Version)
	}

	_, ok := result.Document.(*OAS3Document)
	if !ok {
		t.Fatalf("Expected OAS3Document, got %T", result.Document)
	}

	if len(result.Errors) > 0 {
		t.Errorf("Unexpected validation errors: %v", result.Errors)
	}
}

func TestParseInvalidFile(t *testing.T) {
	parser := New()
	_, err := parser.Parse("nonexistent.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestParseInvalidYAML(t *testing.T) {
	parser := New()
	_, err := parser.ParseBytes([]byte("invalid: yaml: content: ["))
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestParseMissingVersion(t *testing.T) {
	parser := New()
	data := []byte(`
info:
  title: Test API
  version: 1.0.0
paths: {}
`)
	_, err := parser.ParseBytes(data)
	if err == nil {
		t.Error("Expected error for missing version field")
	}
}

func TestParseValidationErrors(t *testing.T) {
	parser := New()
	data := []byte(`
swagger: "2.0"
paths: {}
`)
	result, err := parser.ParseBytes(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(result.Errors) == 0 {
		t.Error("Expected validation errors for missing required fields")
	}

	// Should have errors for missing info
	hasInfoError := false
	for _, err := range result.Errors {
		// Check if error message mentions missing info field
		errMsg := err.Error()
		if strings.Contains(errMsg, "info") && strings.Contains(errMsg, "missing") {
			hasInfoError = true
			break
		}
	}
	if !hasInfoError {
		t.Errorf("Expected error for missing info field, got: %v", result.Errors)
	}
}

func TestParseWithValidationDisabled(t *testing.T) {
	parser := New()
	parser.ValidateStructure = false

	data := []byte(`
swagger: "2.0"
paths: {}
`)
	result, err := parser.ParseBytes(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(result.Errors) > 0 {
		t.Error("Should not have validation errors when validation is disabled")
	}
}

func TestResolveLocalRefs(t *testing.T) {
	parser := New()
	parser.ResolveRefs = true

	result, err := parser.Parse("../../testdata/petstore-3.0.yaml")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(result.Warnings) > 0 {
		t.Logf("Warnings during ref resolution: %v", result.Warnings)
	}

	// The file should parse successfully with refs resolved
	if result.Version != "3.0.3" {
		t.Errorf("Expected version 3.0.3, got %s", result.Version)
	}
}

func TestResolveExternalRefs(t *testing.T) {
	parser := New()
	parser.ResolveRefs = true

	result, err := parser.Parse("../../testdata/with-external-refs.yaml")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(result.Warnings) > 0 {
		t.Logf("Warnings during ref resolution: %v", result.Warnings)
	}

	// The file should parse successfully with external refs resolved
	if result.Version != "3.0.3" {
		t.Errorf("Expected version 3.0.3, got %s", result.Version)
	}
}

func TestDetectVersion(t *testing.T) {
	parser := New()

	tests := []struct {
		name     string
		data     map[string]interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "OAS 2.0",
			data:     map[string]interface{}{"swagger": "2.0"},
			expected: "2.0",
			wantErr:  false,
		},
		{
			name:     "OAS 3.0.0",
			data:     map[string]interface{}{"openapi": "3.0.0"},
			expected: "3.0.0",
			wantErr:  false,
		},
		{
			name:     "OAS 3.1.0",
			data:     map[string]interface{}{"openapi": "3.1.0"},
			expected: "3.1.0",
			wantErr:  false,
		},
		{
			name:     "Missing version",
			data:     map[string]interface{}{"info": "test"},
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := parser.detectVersion(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("detectVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if version != tt.expected {
				t.Errorf("detectVersion() = %v, want %v", version, tt.expected)
			}
		})
	}
}

func TestParseJSON(t *testing.T) {
	// Create a temporary JSON file
	jsonData := `{
		"swagger": "2.0",
		"info": {
			"title": "Test API",
			"version": "1.0.0"
		},
		"paths": {}
	}`

	tmpfile, err := os.CreateTemp("", "test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.Write([]byte(jsonData)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	parser := New()
	result, err := parser.Parse(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to parse JSON file: %v", err)
	}

	if result.Version != "2.0" {
		t.Errorf("Expected version 2.0, got %s", result.Version)
	}

	doc, ok := result.Document.(*OAS2Document)
	if !ok {
		t.Fatalf("Expected OAS2Document, got %T", result.Document)
	}

	if doc.Info.Title != "Test API" {
		t.Errorf("Expected title 'Test API', got '%s'", doc.Info.Title)
	}
}

func TestParseRelativePaths(t *testing.T) {
	// Test that parsing works with relative paths
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(cwd, "../../testdata/petstore-3.0.yaml")
	parser := New()
	result, err := parser.Parse(testFile)
	if err != nil {
		t.Fatalf("Failed to parse with absolute path: %v", err)
	}

	if result.Version != "3.0.3" {
		t.Errorf("Expected version 3.0.3, got %s", result.Version)
	}
}
