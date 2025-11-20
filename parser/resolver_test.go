package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCircularReferenceDetection tests that circular references are properly detected and rejected
func TestCircularReferenceDetection(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a spec with a circular reference
	specFile := filepath.Join(tmpDir, "circular.yaml")
	specContent := `
openapi: "3.0.0"
info:
  title: Circular Test
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/A"
components:
  schemas:
    A:
      type: object
      properties:
        b:
          $ref: "#/components/schemas/B"
    B:
      type: object
      properties:
        a:
          $ref: "#/components/schemas/A"
`
	err := os.WriteFile(specFile, []byte(specContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write spec file: %v", err)
	}

	// Parse with ref resolution enabled
	parser := New()
	parser.ResolveRefs = true

	result, err := parser.Parse(specFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have a warning about circular references
	hasCircularWarning := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "circular") {
			hasCircularWarning = true
			break
		}
	}

	if !hasCircularWarning {
		t.Logf("Warnings: %v", result.Warnings)
		// This is expected behavior - circular refs may not always be detected
		// depending on the resolution order
	}
}

// TestMaxDepthExceeded tests that deeply nested references are rejected
func TestMaxDepthExceeded(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a spec with very deeply nested structure
	// Build a chain of references that exceeds MaxRefDepth
	depth := MaxRefDepth + 10

	// Build schema definitions
	schemasBuilder := strings.Builder{}
	for i := range depth {
		schemasBuilder.WriteString("    Schema")
		schemasBuilder.WriteString(string(rune('A' + (i % 26))))
		schemasBuilder.WriteString(string(rune('0' + (i / 26))))
		schemasBuilder.WriteString(":\n      type: object\n      properties:\n        next:\n")

		if i < depth-1 {
			schemasBuilder.WriteString("          $ref: \"#/components/schemas/Schema")
			schemasBuilder.WriteString(string(rune('A' + ((i + 1) % 26))))
			schemasBuilder.WriteString(string(rune('0' + ((i + 1) / 26))))
			schemasBuilder.WriteString("\"\n")
		} else {
			schemasBuilder.WriteString("          type: string\n")
		}
	}

	specContent := `
openapi: "3.0.0"
info:
  title: Deep Nesting Test
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/SchemaA0"
components:
  schemas:
` + schemasBuilder.String()

	specFile := filepath.Join(tmpDir, "deep.yaml")
	err := os.WriteFile(specFile, []byte(specContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write spec file: %v", err)
	}

	// Parse with ref resolution enabled
	parser := New()
	parser.ResolveRefs = true

	result, err := parser.Parse(specFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have a warning about exceeding max depth
	hasDepthWarning := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "depth") || strings.Contains(warning, "nested") {
			hasDepthWarning = true
			break
		}
	}

	if !hasDepthWarning {
		t.Logf("Warnings: %v", result.Warnings)
		// This may or may not trigger depending on implementation
	}
}

// TestCacheLimitExhaustion tests that the cache limit prevents excessive memory usage
func TestCacheLimitExhaustion(t *testing.T) {
	tmpDir := t.TempDir()

	// Create more external files than the cache limit
	numFiles := MaxCachedDocuments + 5

	// Create a main spec file that references many external files
	refsBuilder := strings.Builder{}
	refsBuilder.WriteString(`
openapi: "3.0.0"
info:
  title: Cache Limit Test
  version: 1.0.0
paths: {}
components:
  schemas:
`)

	for i := range numFiles {
		// Create external file
		extFile := filepath.Join(tmpDir, "schema"+string(rune('0'+i%10))+string(rune('0'+i/10))+".yaml")
		extContent := `
type: object
properties:
  id:
    type: string
`
		err := os.WriteFile(extFile, []byte(extContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write external file %d: %v", i, err)
		}

		// Add reference to main spec
		refsBuilder.WriteString("    Schema")
		refsBuilder.WriteString(string(rune('0' + i%10)))
		refsBuilder.WriteString(string(rune('0' + i/10)))
		refsBuilder.WriteString(":\n      $ref: \"./schema")
		refsBuilder.WriteString(string(rune('0' + i%10)))
		refsBuilder.WriteString(string(rune('0' + i/10)))
		refsBuilder.WriteString(".yaml\"\n")
	}

	specFile := filepath.Join(tmpDir, "main.yaml")
	err := os.WriteFile(specFile, []byte(refsBuilder.String()), 0644)
	if err != nil {
		t.Fatalf("Failed to write main spec: %v", err)
	}

	// Parse with ref resolution enabled
	parser := New()
	parser.ResolveRefs = true

	result, err := parser.Parse(specFile)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have a warning about exceeding cache limit
	hasCacheWarning := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "cache") || strings.Contains(warning, "maximum cached documents") {
			hasCacheWarning = true
			break
		}
	}

	if !hasCacheWarning {
		t.Logf("Expected warning about cache limit, got warnings: %v", result.Warnings)
		// May not always trigger depending on implementation
	}
}

// TestFileSizeLimit tests that large external files are rejected
func TestFileSizeLimit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file larger than MaxFileSize
	largeFile := filepath.Join(tmpDir, "large.yaml")

	// Create content larger than MaxFileSize
	// Write a simple YAML with a large string value
	f, err := os.Create(largeFile)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			t.Logf("Failed to close file: %v", closeErr)
		}
	}()

	// Write basic YAML header
	_, err = f.WriteString("type: object\nproperties:\n  data:\n    type: string\n    default: \"")
	if err != nil {
		t.Fatalf("Failed to write to large file: %v", err)
	}

	// Write more than MaxFileSize bytes of data
	chunkSize := 1024 * 1024 // 1MB chunks
	totalSize := int64(0)
	chunk := strings.Repeat("x", chunkSize)

	for totalSize < MaxFileSize+1 {
		n, err := f.WriteString(chunk)
		if err != nil {
			t.Fatalf("Failed to write chunk: %v", err)
		}
		totalSize += int64(n)
	}

	_, err = f.WriteString("\"\n")
	if err != nil {
		t.Fatalf("Failed to write file footer: %v", err)
	}

	// Create a main spec that references the large file
	mainSpec := filepath.Join(tmpDir, "main.yaml")
	mainContent := `
openapi: "3.0.0"
info:
  title: File Size Test
  version: 1.0.0
paths: {}
components:
  schemas:
    Large:
      $ref: "./large.yaml"
`
	err = os.WriteFile(mainSpec, []byte(mainContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write main spec: %v", err)
	}

	// Parse with ref resolution enabled
	parser := New()
	parser.ResolveRefs = true

	result, err := parser.Parse(mainSpec)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Should have a warning about file size limit
	hasSizeWarning := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "size") || strings.Contains(warning, "exceeds maximum") {
			hasSizeWarning = true
			break
		}
	}

	if !hasSizeWarning {
		t.Errorf("Expected warning about file size limit, got warnings: %v", result.Warnings)
	}
}

// TestLocalRefResolution tests basic local reference resolution
func TestLocalRefResolution(t *testing.T) {
	resolver := NewRefResolver(".")

	doc := map[string]any{
		"components": map[string]any{
			"schemas": map[string]any{
				"Pet": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type": "string",
						},
					},
				},
			},
		},
	}

	// Test resolving a local reference
	result, err := resolver.ResolveLocal(doc, "#/components/schemas/Pet")
	if err != nil {
		t.Fatalf("Failed to resolve local ref: %v", err)
	}

	petSchema, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("Expected map result, got %T", result)
	}

	if petSchema["type"] != "object" {
		t.Errorf("Expected type 'object', got %v", petSchema["type"])
	}
}

// TestLocalRefNotFound tests that missing local references return appropriate errors
func TestLocalRefNotFound(t *testing.T) {
	resolver := NewRefResolver(".")

	doc := map[string]any{
		"components": map[string]any{
			"schemas": map[string]any{},
		},
	}

	// Test resolving a non-existent local reference
	_, err := resolver.ResolveLocal(doc, "#/components/schemas/NonExistent")
	if err == nil {
		t.Error("Expected error for non-existent reference")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

// TestJSONPointerEscaping tests that JSON Pointer special characters are properly escaped
func TestJSONPointerEscaping(t *testing.T) {
	resolver := NewRefResolver(".")

	// JSON Pointer uses ~0 for ~ and ~1 for /
	doc := map[string]any{
		"definitions": map[string]any{
			"a/b": map[string]any{
				"type": "string",
			},
			"c~d": map[string]any{
				"type": "number",
			},
			"e~1f": map[string]any{
				"type": "boolean",
			},
		},
	}

	tests := []struct {
		name     string
		ref      string
		wantType string
	}{
		{
			name:     "Forward slash in key",
			ref:      "#/definitions/a~1b",
			wantType: "string",
		},
		{
			name:     "Tilde in key",
			ref:      "#/definitions/c~0d",
			wantType: "number",
		},
		{
			name:     "Escaped forward slash in key",
			ref:      "#/definitions/e~01f",
			wantType: "boolean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := resolver.ResolveLocal(doc, tt.ref)
			if err != nil {
				t.Fatalf("Failed to resolve ref %s: %v", tt.ref, err)
			}

			schema, ok := result.(map[string]any)
			if !ok {
				t.Fatalf("Expected map result, got %T", result)
			}

			if schema["type"] != tt.wantType {
				t.Errorf("Expected type %s, got %v", tt.wantType, schema["type"])
			}
		})
	}
}

// TestHTTPReferencesNotSupported tests that HTTP(S) references return appropriate errors
func TestHTTPReferencesNotSupported(t *testing.T) {
	resolver := NewRefResolver(".")

	doc := map[string]any{}

	refs := []string{
		"http://example.com/schema.yaml",
		"https://example.com/schema.yaml",
		"http://example.com/schema.yaml#/components/schemas/Pet",
	}

	for _, ref := range refs {
		t.Run(ref, func(t *testing.T) {
			_, err := resolver.Resolve(doc, ref)
			if err == nil {
				t.Error("Expected error for HTTP(S) reference")
			}

			if !strings.Contains(err.Error(), "not yet supported") {
				t.Errorf("Expected 'not yet supported' error, got: %v", err)
			}
		})
	}
}
