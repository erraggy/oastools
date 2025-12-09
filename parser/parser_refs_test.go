package parser

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erraggy/oastools/oaserrors"
)

func TestResolveLocalRefs(t *testing.T) {
	parser := New()
	parser.ResolveRefs = true

	result, err := parser.Parse("../testdata/petstore-3.0.yaml")
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

	result, err := parser.Parse("../testdata/with-external-refs.yaml")
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

func TestPathTraversalSecurity(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()

	// Create a safe directory with an allowed file
	safeDir := filepath.Join(tmpDir, "safe")
	err := os.MkdirAll(safeDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create safe directory: %v", err)
	}

	// Create an allowed file in the safe directory
	allowedFile := filepath.Join(safeDir, "allowed.yaml")
	allowedContent := `
openapi: "3.0.0"
info:
  title: Allowed Component
  version: 1.0.0
paths: {}
`
	err = os.WriteFile(allowedFile, []byte(allowedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write allowed file: %v", err)
	}

	// Create a restricted directory with a forbidden file (outside safe dir)
	restrictedFile := filepath.Join(tmpDir, "forbidden.yaml")
	restrictedContent := `
openapi: "3.0.0"
info:
  title: Forbidden Component
  version: 1.0.0
paths: {}
`
	err = os.WriteFile(restrictedFile, []byte(restrictedContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write restricted file: %v", err)
	}

	tests := []struct {
		name          string
		ref           string
		shouldSucceed bool
	}{
		{
			name:          "Valid reference within baseDir",
			ref:           "./allowed.yaml",
			shouldSucceed: true,
		},
		{
			name:          "Path traversal with ../",
			ref:           "../forbidden.yaml",
			shouldSucceed: false,
		},
		{
			name:          "Path traversal with ../../",
			ref:           "../../forbidden.yaml",
			shouldSucceed: false,
		},
		{
			name:          "Path traversal with ../../../",
			ref:           "../../../etc/passwd",
			shouldSucceed: false,
		},
		{
			name:          "Absolute path outside baseDir",
			ref:           restrictedFile,
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewRefResolver(safeDir)
			result, err := resolver.ResolveExternal(tt.ref)

			if tt.shouldSucceed {
				if err != nil {
					t.Errorf("Expected success but got error: %v", err)
				}
				if result == nil {
					t.Error("Expected non-nil result for successful resolution")
				}
			} else {
				if err == nil {
					t.Error("Expected error but got nil")
					return
				}
				// Use errors.Is for sentinel error check
				if !errors.Is(err, oaserrors.ErrPathTraversal) {
					t.Errorf("Expected ErrPathTraversal, got: %v", err)
				}
				// Use errors.As to verify error type and fields
				var refErr *oaserrors.ReferenceError
				if !errors.As(err, &refErr) {
					t.Errorf("Expected *oaserrors.ReferenceError, got %T", err)
				} else if !refErr.IsPathTraversal {
					t.Errorf("Expected IsPathTraversal=true, got false")
				}
			}
		})
	}
}

func TestPathTraversalWindows(t *testing.T) {
	// Test the Windows edge case mentioned in the code review
	// where "C:\base" and "C:\base2" would pass a simple prefix check

	tmpDir := t.TempDir()

	// Create two directories: "base" and "base2"
	baseDir := filepath.Join(tmpDir, "base")
	base2Dir := filepath.Join(tmpDir, "base2")

	err := os.MkdirAll(baseDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create base directory: %v", err)
	}

	err = os.MkdirAll(base2Dir, 0755)
	if err != nil {
		t.Fatalf("Failed to create base2 directory: %v", err)
	}

	// Create a file in base2
	forbiddenFile := filepath.Join(base2Dir, "forbidden.yaml")
	err = os.WriteFile(forbiddenFile, []byte("openapi: 3.0.0\ninfo:\n  title: Test\n  version: 1.0.0\npaths: {}"), 0644)
	if err != nil {
		t.Fatalf("Failed to write forbidden file: %v", err)
	}

	// Try to access the file in base2 from a resolver with baseDir set to base
	resolver := NewRefResolver(baseDir)

	// Try various ways to escape to base2
	refs := []string{
		"../base2/forbidden.yaml",
		filepath.Join("..", "base2", "forbidden.yaml"),
		forbiddenFile, // absolute path
	}

	for _, ref := range refs {
		t.Run("ref="+ref, func(t *testing.T) {
			result, err := resolver.ResolveExternal(ref)

			// All these should fail with path traversal error
			if err == nil {
				t.Errorf("Expected path traversal error for ref '%s', but got nil error. Result: %v", ref, result)
				return
			}
			// Use errors.Is for sentinel error check
			if !errors.Is(err, oaserrors.ErrPathTraversal) {
				t.Errorf("Expected ErrPathTraversal for ref '%s', got: %v", ref, err)
			}
			// Use errors.As to verify error type
			var refErr *oaserrors.ReferenceError
			if !errors.As(err, &refErr) {
				t.Errorf("Expected *oaserrors.ReferenceError for ref '%s', got %T", ref, err)
			} else if !refErr.IsPathTraversal {
				t.Errorf("Expected IsPathTraversal=true for ref '%s'", ref)
			}
		})
	}
}

func TestMalformedExternalRefs(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	// Create a valid external file
	validExternal := filepath.Join(tmpDir, "valid.yaml")
	validContent := []byte(`
type: object
properties:
  id:
    type: integer
`)
	if err := os.WriteFile(validExternal, validContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a malformed external file
	malformedExternal := filepath.Join(tmpDir, "malformed.yaml")
	malformedContent := []byte(`{{{invalid yaml`)
	if err := os.WriteFile(malformedExternal, malformedContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name      string
		spec      string
		expectErr bool
		errorMsg  string
	}{
		{
			name: "Valid external ref",
			spec: `openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: './valid.yaml'
`,
			expectErr: false,
		},
		{
			name: "Malformed external ref - invalid YAML",
			spec: `openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: './malformed.yaml'
`,
			expectErr: true,
			errorMsg:  "ref resolution warning",
		},
		{
			name: "Non-existent external ref",
			spec: `openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: './nonexistent.yaml'
`,
			expectErr: true,
			errorMsg:  "ref resolution warning",
		},
		{
			name: "HTTP(S) reference not supported",
			spec: `openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: 'https://example.com/schema.yaml'
`,
			expectErr: true,
			errorMsg:  "ref resolution warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write spec to temp file
			specFile := filepath.Join(tmpDir, "spec.yaml")
			if err := os.WriteFile(specFile, []byte(tt.spec), 0644); err != nil {
				t.Fatalf("Failed to create spec file: %v", err)
			}

			parser := New()
			parser.ResolveRefs = true
			result, err := parser.Parse(specFile)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			hasExpectedWarning := false
			for _, w := range result.Warnings {
				if strings.Contains(w, tt.errorMsg) {
					hasExpectedWarning = true
					break
				}
			}

			if tt.expectErr && !hasExpectedWarning {
				t.Errorf("Expected warning containing '%s', but got none. Warnings: %v", tt.errorMsg, result.Warnings)
			}

			if !tt.expectErr && hasExpectedWarning {
				t.Errorf("Did not expect warning, but got one. Warnings: %v", result.Warnings)
			}
		})
	}
}
