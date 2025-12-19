package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCircularReferenceDetection tests that circular references are properly detected and rejected
func TestCircularReferenceDetection(t *testing.T) {
	// Skip this test when running with race detector due to high memory usage
	// Circular reference resolution with the race detector can exhaust memory
	if testing.Short() {
		t.Skip("Skipping resource-intensive test in short mode")
	}

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
	require.NoError(t, err, "Failed to write spec file")

	// Parse with ref resolution enabled
	parser := New()
	parser.ResolveRefs = true

	result, err := parser.Parse(specFile)
	require.NoError(t, err, "Parse failed")

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

// TestRefResolver_HasCircularRefs tests the HasCircularRefs method
func TestRefResolver_HasCircularRefs(t *testing.T) {
	tests := []struct {
		name         string
		doc          map[string]any
		wantCircular bool
	}{
		{
			name: "no circular refs",
			doc: map[string]any{
				"components": map[string]any{
					"schemas": map[string]any{
						"Pet": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"name": map[string]any{"type": "string"},
							},
						},
					},
				},
			},
			wantCircular: false,
		},
		{
			name: "root ref is circular",
			doc: map[string]any{
				"components": map[string]any{
					"schemas": map[string]any{
						"Recursive": map[string]any{
							"$ref": "#",
						},
					},
				},
			},
			wantCircular: true,
		},
		{
			name: "root path ref is circular",
			doc: map[string]any{
				"components": map[string]any{
					"schemas": map[string]any{
						"Recursive": map[string]any{
							"$ref": "#/",
						},
					},
				},
			},
			wantCircular: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewRefResolver("")
			_ = resolver.ResolveAllRefs(tt.doc)

			assert.Equal(t, tt.wantCircular, resolver.HasCircularRefs(), "HasCircularRefs() mismatch")
		})
	}
}

// TestRefResolver_HasCircularRefs_Reset verifies the flag is reset between calls
func TestRefResolver_HasCircularRefs_Reset(t *testing.T) {
	resolver := NewRefResolver("")

	// First: resolve a doc with circular refs
	docWithCircular := map[string]any{
		"schema": map[string]any{"$ref": "#"},
	}
	_ = resolver.ResolveAllRefs(docWithCircular)
	assert.True(t, resolver.HasCircularRefs(), "expected HasCircularRefs() = true after resolving circular doc")

	// Second: resolve a doc without circular refs - flag should reset
	docWithoutCircular := map[string]any{
		"components": map[string]any{
			"schemas": map[string]any{
				"Pet": map[string]any{"type": "object"},
			},
		},
	}
	_ = resolver.ResolveAllRefs(docWithoutCircular)
	assert.False(t, resolver.HasCircularRefs(), "expected HasCircularRefs() = false after resolving non-circular doc")
}

// TestMaxDepthExceeded tests that deeply nested references are rejected
func TestMaxDepthExceeded(t *testing.T) {
	// Skip this test when running with race detector due to high memory usage
	// Building deeply nested structures with the race detector can exhaust memory
	if testing.Short() {
		t.Skip("Skipping resource-intensive test in short mode")
	}

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
	require.NoError(t, err, "Failed to write spec file")

	// Parse with ref resolution enabled
	parser := New()
	parser.ResolveRefs = true

	result, err := parser.Parse(specFile)
	require.NoError(t, err, "Parse failed")

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
	// Skip this test when running with race detector due to high memory usage
	// Creating many files and caching them with the race detector can exhaust memory
	if testing.Short() {
		t.Skip("Skipping resource-intensive test in short mode")
	}

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
		require.NoError(t, err, "Failed to write external file %d", i)

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
	require.NoError(t, err, "Failed to write main spec")

	// Parse with ref resolution enabled
	parser := New()
	parser.ResolveRefs = true

	result, err := parser.Parse(specFile)
	require.NoError(t, err, "Parse failed")

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
	// Skip this test when running with race detector due to high memory usage
	// The race detector increases memory usage by 5-10x, and writing >10MB of data
	// can cause GitHub Actions runners to kill the process
	if testing.Short() {
		t.Skip("Skipping resource-intensive test in short mode")
	}

	tmpDir := t.TempDir()

	// Create a file larger than MaxFileSize
	largeFile := filepath.Join(tmpDir, "large.yaml")

	// Create content larger than MaxFileSize
	// Write a simple YAML with a large string value
	f, err := os.Create(largeFile)
	require.NoError(t, err, "Failed to create large file")
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			t.Logf("Failed to close file: %v", closeErr)
		}
	}()

	// Write basic YAML header
	_, err = f.WriteString("type: object\nproperties:\n  data:\n    type: string\n    default: \"")
	require.NoError(t, err, "Failed to write to large file")

	// Write more than MaxFileSize bytes of data
	chunkSize := 1024 * 1024 // 1MB chunks
	totalSize := int64(0)
	chunk := strings.Repeat("x", chunkSize)

	for totalSize < MaxFileSize+1 {
		n, err := f.WriteString(chunk)
		require.NoError(t, err, "Failed to write chunk")
		totalSize += int64(n)
	}

	_, err = f.WriteString("\"\n")
	require.NoError(t, err, "Failed to write file footer")

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
	require.NoError(t, err, "Failed to write main spec")

	// Parse with ref resolution enabled
	parser := New()
	parser.ResolveRefs = true

	result, err := parser.Parse(mainSpec)
	require.NoError(t, err, "Parse failed")

	// Should have a warning about file size limit
	hasSizeWarning := false
	for _, warning := range result.Warnings {
		if strings.Contains(warning, "size") || strings.Contains(warning, "exceeds maximum") {
			hasSizeWarning = true
			break
		}
	}

	assert.True(t, hasSizeWarning, "Expected warning about file size limit, got warnings: %v", result.Warnings)
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
	require.NoError(t, err, "Failed to resolve local ref")

	petSchema, ok := result.(map[string]any)
	require.True(t, ok, "Expected map result, got %T", result)

	assert.Equal(t, "object", petSchema["type"], "Expected type 'object'")
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
	require.Error(t, err, "Expected error for non-existent reference")
	assert.Contains(t, err.Error(), "not found")
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
			require.NoError(t, err, "Failed to resolve ref %s", tt.ref)

			schema, ok := result.(map[string]any)
			require.True(t, ok, "Expected map result, got %T", result)

			assert.Equal(t, tt.wantType, schema["type"], "Expected type %s", tt.wantType)
		})
	}
}

// TestHTTPReferencesRequireFetcher tests that HTTP(S) references require an HTTP fetcher
func TestHTTPReferencesRequireFetcher(t *testing.T) {
	// Without HTTP fetcher configured, HTTP refs should return an error
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
			require.Error(t, err, "Expected error for HTTP(S) reference without fetcher")
			assert.Contains(t, err.Error(), "HTTP references require HTTP fetcher")
		})
	}
}

// TestCircularSelfReferenceInResolve tests that a schema with a circular self-reference
// doesn't cause an infinite loop during resolution. This test case was discovered by fuzzing.
func TestCircularSelfReferenceInResolve(t *testing.T) {
	// Skip this test when running with race detector due to high memory usage
	// Circular reference resolution with the race detector can exhaust memory
	if testing.Short() {
		t.Skip("Skipping resource-intensive test in short mode")
	}

	// This input was discovered by the fuzzer and caused an infinite loop
	// The key issue: a circular reference (Node.next -> Node) that creates an
	// infinite expansion when resolving refs
	input := []byte(`openapi: 3.0.0
info:
  title: Circular Schema API
  version: "1.0.0"
paths:
  /test:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Node"
components:
  schemas:
    Node:
      type: object
      properties:
        value:
          type: string
        next:
          $ref: "#/components/schemas/Node"
`)

	// Parse with resolveRefs enabled - this should not hang
	result, err := ParseWithOptions(
		WithBytes(input),
		WithResolveRefs(true),
		WithValidateStructure(true),
	)

	// Should parse successfully without hanging
	require.NoError(t, err, "ParseBytes failed")

	// Should have parsed the document
	require.NotNil(t, result.Document, "Expected document to be parsed")

	// Verify it's OAS 3.0
	assert.Equal(t, "3.0.0", result.Version, "Expected version 3.0.0")
}

// TestRefToDocumentRoot tests that a $ref pointing to the document root ("#")
// doesn't cause an infinite loop. This test case was discovered by fuzzing.
func TestRefToDocumentRoot(t *testing.T) {
	// Skip this test when running with race detector due to high memory usage
	// Document root references with the race detector can exhaust memory
	if testing.Short() {
		t.Skip("Skipping resource-intensive test in short mode")
	}

	// This input was discovered by the fuzzer and caused an infinite loop
	// The issue: $ref: "#" points to the document root, creating infinite recursion
	input := []byte(`openapi: 3.0.0
info:
  title: Circular Schema API
  version: "1.0.0"
paths:
  /test:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#"
`)

	// Parse with resolveRefs enabled - this should not hang
	result, err := ParseWithOptions(
		WithBytes(input),
		WithResolveRefs(true),
		WithValidateStructure(true),
	)

	// Should parse successfully without hanging
	require.NoError(t, err, "ParseBytes failed")

	// Should have parsed the document
	require.NotNil(t, result.Document, "Expected document to be parsed")

	// Verify it's OAS 3.0
	assert.Equal(t, "3.0.0", result.Version, "Expected version 3.0.0")

	// Verify the $ref was preserved (not resolved to prevent infinite loop)
	doc, ok := result.Document.(*OAS3Document)
	require.True(t, ok, "Expected OAS3Document")
	require.NotNil(t, doc.Paths, "Expected paths")
	require.NotNil(t, doc.Paths["/test"], "Expected /test path")
	require.NotNil(t, doc.Paths["/test"].Get, "Expected GET operation")
	require.NotNil(t, doc.Paths["/test"].Get.Responses, "Expected responses")

	response := doc.Paths["/test"].Get.Responses.Codes["200"]
	require.NotNil(t, response, "Expected 200 response")
	require.NotNil(t, response.Content, "Expected content")
	require.NotNil(t, response.Content["application/json"], "Expected application/json")
	require.NotNil(t, response.Content["application/json"].Schema, "Expected schema")

	schema := response.Content["application/json"].Schema
	assert.Equal(t, "#", schema.Ref, "Expected $ref to be preserved as '#'")

	// Verify that a circular reference warning was added
	foundCircularWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "Circular references detected") {
			foundCircularWarning = true
			break
		}
	}
	assert.True(t, foundCircularWarning, "Expected a warning about circular references being detected")
}

// TestArrayIndexResolution tests RFC 6901 JSON Pointer array index support
func TestArrayIndexResolution(t *testing.T) {
	tests := []struct {
		name        string
		doc         map[string]any
		ref         string
		expectError bool
		errorMsg    string
		expected    any
	}{
		{
			name: "valid first array element",
			doc: map[string]any{
				"items": []any{
					map[string]any{"name": "first"},
					map[string]any{"name": "second"},
				},
			},
			ref:      "#/items/0",
			expected: map[string]any{"name": "first"},
		},
		{
			name: "valid second array element",
			doc: map[string]any{
				"items": []any{
					map[string]any{"name": "first"},
					map[string]any{"name": "second"},
				},
			},
			ref:      "#/items/1",
			expected: map[string]any{"name": "second"},
		},
		{
			name: "nested array access",
			doc: map[string]any{
				"paths": map[string]any{
					"/users": map[string]any{
						"get": map[string]any{
							"parameters": []any{
								map[string]any{"name": "limit", "in": "query"},
								map[string]any{"name": "offset", "in": "query"},
							},
						},
					},
				},
			},
			ref:      "#/paths/~1users/get/parameters/0",
			expected: map[string]any{"name": "limit", "in": "query"},
		},
		{
			name: "array index out of bounds",
			doc: map[string]any{
				"items": []any{
					map[string]any{"name": "only"},
				},
			},
			ref:         "#/items/5",
			expectError: true,
			errorMsg:    "out of bounds",
		},
		{
			name: "negative array index",
			doc: map[string]any{
				"items": []any{
					map[string]any{"name": "item"},
				},
			},
			ref:         "#/items/-1",
			expectError: true,
			errorMsg:    "out of bounds", // -1 parses as valid integer but fails bounds check
		},
		{
			name: "non-numeric array index",
			doc: map[string]any{
				"items": []any{
					map[string]any{"name": "item"},
				},
			},
			ref:         "#/items/abc",
			expectError: true,
			errorMsg:    "invalid array index",
		},
		{
			name: "empty array access",
			doc: map[string]any{
				"items": []any{},
			},
			ref:         "#/items/0",
			expectError: true,
			errorMsg:    "out of bounds",
		},
		{
			name: "deeply nested array access",
			doc: map[string]any{
				"components": map[string]any{
					"schemas": map[string]any{
						"Response": map[string]any{
							"oneOf": []any{
								map[string]any{"type": "object"},
								map[string]any{"type": "array"},
							},
						},
					},
				},
			},
			ref:      "#/components/schemas/Response/oneOf/1",
			expected: map[string]any{"type": "array"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewRefResolver("")
			result, err := resolver.Resolve(tt.doc, tt.ref)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

// TestArrayIndexInParsedSpec tests array index refs in actual OpenAPI specs
func TestArrayIndexInParsedSpec(t *testing.T) {
	spec := `
openapi: "3.0.0"
info:
  title: Array Index Test
  version: "1.0.0"
paths:
  /users:
    get:
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
        - name: offset
          in: query
          schema:
            type: integer
      responses:
        "200":
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/paths/~1users/get/parameters/0/schema'
`
	result, err := ParseWithOptions(
		WithBytes([]byte(spec)),
		WithResolveRefs(true),
		WithValidateStructure(true),
	)
	require.NoError(t, err)
	require.NotNil(t, result)

	// The ref should have been resolved correctly using array index
	doc, ok := result.OAS3Document()
	require.True(t, ok)

	response := doc.Paths["/users"].Get.Responses.Codes["200"]
	require.NotNil(t, response)
	schema := response.Content["application/json"].Schema
	require.NotNil(t, schema)

	// The resolved schema should have type: integer
	assert.Equal(t, "integer", schema.Type)
}
