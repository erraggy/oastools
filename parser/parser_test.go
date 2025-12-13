package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOAS2(t *testing.T) {
	parser := New()
	result, err := parser.Parse("../testdata/petstore-2.0.yaml")
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
	result, err := parser.Parse("../testdata/petstore-3.0.yaml")
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
	result, err := parser.Parse("../testdata/petstore-3.1.yaml")
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
	result, err := parser.Parse("../testdata/petstore-3.2.yaml")
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

func TestParseOAS32WithQueryMethod(t *testing.T) {
	// Test that QUERY method is correctly parsed in OAS 3.2 document
	data := []byte(`openapi: 3.2.0
info:
  title: Test API with QUERY
  version: 1.0.0
paths:
  /users:
    query:
      operationId: queryUsers
      summary: Query users
      responses:
        '200':
          description: Successful query response
`)

	parser := New()
	result, err := parser.ParseBytes(data)
	if err != nil {
		t.Fatalf("Failed to parse OAS 3.2 document with QUERY: %v", err)
	}

	if result.Version != "3.2.0" {
		t.Errorf("Expected version 3.2.0, got %s", result.Version)
	}

	doc, ok := result.Document.(*OAS3Document)
	if !ok {
		t.Fatalf("Expected OAS3Document, got %T", result.Document)
	}

	// Verify QUERY operation is present
	usersPath := doc.Paths["/users"]
	if usersPath == nil {
		t.Fatal("Expected /users path to be present")
	}

	if usersPath.Query == nil {
		t.Fatal("Expected QUERY operation to be present")
	}

	if usersPath.Query.OperationID != "queryUsers" {
		t.Errorf("Expected operationId 'queryUsers', got %s", usersPath.Query.OperationID)
	}

	if usersPath.Query.Summary != "Query users" {
		t.Errorf("Expected summary 'Query users', got %s", usersPath.Query.Summary)
	}

	// Verify GetOperations includes QUERY for OAS 3.2
	ops := GetOperations(usersPath, OASVersion320)
	if ops["query"] == nil {
		t.Error("Expected 'query' to be in operations map for OAS 3.2")
	}

	// Verify GetOperations excludes QUERY for earlier versions
	opsOAS31 := GetOperations(usersPath, OASVersion310)
	if opsOAS31["query"] != nil {
		t.Error("Expected 'query' to NOT be in operations map for OAS 3.1")
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

	tmpDir := t.TempDir()
	tmpfile := filepath.Join(tmpDir, "test.json")
	if err := os.WriteFile(tmpfile, []byte(jsonData), 0600); err != nil {
		t.Fatal(err)
	}

	parser := New()
	result, err := parser.Parse(tmpfile)
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

	testFile := filepath.Join(cwd, "../testdata/petstore-3.0.yaml")
	parser := New()
	result, err := parser.Parse(testFile)
	if err != nil {
		t.Fatalf("Failed to parse with absolute path: %v", err)
	}

	if result.Version != "3.0.3" {
		t.Errorf("Expected version 3.0.3, got %s", result.Version)
	}
}

// TestAllOfficialOASVersions tests that all official OpenAPI Specification versions are properly handled
// This test validates against the complete set of released versions from https://github.com/OAI/OpenAPI-Specification/releases
func TestAllOfficialOASVersions(t *testing.T) {
	// All official OAS versions (excluding release candidates with -rc suffixes)
	// Source: https://github.com/OAI/OpenAPI-Specification/releases
	officialVersions := []struct {
		version       string
		expectedType  string // "OAS2Document" or "OAS3Document"
		shouldSucceed bool
	}{
		// OAS 2.x series
		{"2.0", "OAS2Document", true},

		// OAS 3.0.x series
		{"3.0.0", "OAS3Document", true},
		{"3.0.1", "OAS3Document", true},
		{"3.0.2", "OAS3Document", true},
		{"3.0.3", "OAS3Document", true},
		{"3.0.4", "OAS3Document", true},

		// OAS 3.1.x series
		{"3.1.0", "OAS3Document", true},
		{"3.1.1", "OAS3Document", true},
		{"3.1.2", "OAS3Document", true},

		// OAS 3.2.x series
		{"3.2.0", "OAS3Document", true},
	}

	for _, tt := range officialVersions {
		t.Run("OAS_"+tt.version, func(t *testing.T) {
			parser := New()

			// Build a minimal valid spec for this version
			var data []byte
			if tt.version == "2.0" {
				data = []byte(`
swagger: "` + tt.version + `"
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: Success
`)
			} else {
				data = []byte(`
openapi: "` + tt.version + `"
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: Success
`)
			}

			result, err := parser.ParseBytes(data)
			if err != nil {
				t.Fatalf("Failed to parse OAS %s: %v", tt.version, err)
			}

			// Verify version detection
			if result.Version != tt.version {
				t.Errorf("Version detection failed: expected %s, got %s", tt.version, result.Version)
			}

			// Verify correct document type
			switch tt.expectedType {
			case "OAS2Document":
				if _, ok := result.Document.(*OAS2Document); !ok {
					t.Errorf("Expected *OAS2Document for version %s, got %T", tt.version, result.Document)
				}
			case "OAS3Document":
				if _, ok := result.Document.(*OAS3Document); !ok {
					t.Errorf("Expected *OAS3Document for version %s, got %T", tt.version, result.Document)
				}
			}

			// Should have no validation errors for valid minimal spec
			if len(result.Errors) > 0 {
				t.Errorf("Unexpected validation errors for OAS %s: %v", tt.version, result.Errors)
			}
		})
	}
}

// TestRCVersionsAccepted tests that release candidate versions are handled
// by mapping them to the closest known version without exceeding the base version
func TestRCVersionsAccepted(t *testing.T) {
	tests := []struct {
		rcVersion      string
		expectedOASVer OASVersion
		expectedVerStr string
	}{
		{"3.0.0-rc0", OASVersion300, "3.0.0"},
		{"3.0.0-rc1", OASVersion300, "3.0.0"},
		{"3.0.0-rc2", OASVersion300, "3.0.0"},
		{"3.1.0-rc0", OASVersion310, "3.1.0"},
		{"3.1.0-rc1", OASVersion310, "3.1.0"},
		{"3.0.5-rc0", OASVersion304, "3.0.4"}, // Maps to closest without exceeding
		{"3.1.3-rc0", OASVersion312, "3.1.2"}, // Maps to closest without exceeding
	}

	for _, tt := range tests {
		t.Run("RC_"+tt.rcVersion, func(t *testing.T) {
			parser := New()

			data := []byte(`
openapi: "` + tt.rcVersion + `"
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: Success
`)

			result, err := parser.ParseBytes(data)
			assert.NoError(t, err)
			assert.NotNil(t, result)

			// Verify it mapped to the correct OAS version
			assert.Equal(t, tt.expectedOASVer, result.OASVersion)
			assert.Equal(t, tt.rcVersion, result.Version) // Original version preserved

			// Verify document parsed correctly
			doc, ok := result.Document.(*OAS3Document)
			assert.True(t, ok, "Expected OAS3Document")
			assert.Equal(t, tt.rcVersion, doc.OpenAPI)
		})
	}
}

func TestParseResultSourcePath(t *testing.T) {
	tests := []struct {
		name           string
		parseFunc      func(*Parser) (*ParseResult, error)
		expectedSource string
	}{
		{
			name: "Parse sets actual file path",
			parseFunc: func(p *Parser) (*ParseResult, error) {
				return p.Parse("../testdata/petstore-3.0.yaml")
			},
			expectedSource: "../testdata/petstore-3.0.yaml",
		},
		{
			name: "ParseBytes sets synthetic path",
			parseFunc: func(p *Parser) (*ParseResult, error) {
				return p.ParseBytes([]byte(`
openapi: "3.0.0"
info:
  title: Test ParseBytes
  version: 1.0.0
paths: {}
`))
			},
			expectedSource: "ParseBytes.yaml",
		},
		{
			name: "ParseReader sets synthetic path",
			parseFunc: func(p *Parser) (*ParseResult, error) {
				file, err := os.Open("../testdata/petstore-3.0.yaml")
				if err != nil {
					return nil, err
				}
				defer func() { _ = file.Close() }()
				return p.ParseReader(file)
			},
			expectedSource: "ParseReader.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := New()
			result, err := tt.parseFunc(p)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedSource, result.SourcePath)
		})
	}
}

func TestParseResultCopy(t *testing.T) {
	// Parse a document
	original, err := ParseWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
		WithValidateStructure(true),
	)
	require.NoError(t, err)
	require.NotNil(t, original)

	// Create a copy
	copied := original.Copy()
	require.NotNil(t, copied)

	// Verify all fields are copied
	assert.Equal(t, original.SourcePath, copied.SourcePath)
	assert.Equal(t, original.SourceFormat, copied.SourceFormat)
	assert.Equal(t, original.Version, copied.Version)
	assert.Equal(t, original.OASVersion, copied.OASVersion)
	assert.Equal(t, len(original.Errors), len(copied.Errors))
	assert.Equal(t, len(original.Warnings), len(copied.Warnings))

	// Verify the copy is independent - modifying Data in copy doesn't affect original
	copied.Data["test-key"] = "test-value"
	_, exists := original.Data["test-key"]
	assert.False(t, exists, "Modifying copied Data should not affect original")
}

func TestParseResultCopyNil(t *testing.T) {
	var nilResult *ParseResult
	copied := nilResult.Copy()
	assert.Nil(t, copied)
}

func TestParseResultCopyPreservesMetadata(t *testing.T) {
	original, err := ParseWithOptions(
		WithFilePath("../testdata/petstore-3.0.yaml"),
	)
	require.NoError(t, err)

	// Create a copy
	copied := original.Copy()
	require.NotNil(t, copied)

	// Verify metadata is preserved
	assert.Equal(t, original.LoadTime, copied.LoadTime)
	assert.Equal(t, original.SourceSize, copied.SourceSize)
	assert.Equal(t, original.Stats.PathCount, copied.Stats.PathCount)
	assert.Equal(t, original.Stats.OperationCount, copied.Stats.OperationCount)
	assert.Equal(t, original.Stats.SchemaCount, copied.Stats.SchemaCount)
}

func TestOAS2Document(t *testing.T) {
	t.Run("returns document for OAS 2.0 spec", func(t *testing.T) {
		result, err := ParseWithOptions(
			WithFilePath("../testdata/petstore-2.0.yaml"),
		)
		require.NoError(t, err)

		doc, ok := result.OAS2Document()
		assert.True(t, ok, "OAS2Document should return true for 2.0 spec")
		assert.NotNil(t, doc, "Document should not be nil")
		assert.Equal(t, "Petstore API", doc.Info.Title)
	})

	t.Run("returns false for OAS 3.0 spec", func(t *testing.T) {
		result, err := ParseWithOptions(
			WithFilePath("../testdata/petstore-3.0.yaml"),
		)
		require.NoError(t, err)

		doc, ok := result.OAS2Document()
		assert.False(t, ok, "OAS2Document should return false for 3.0 spec")
		assert.Nil(t, doc, "Document should be nil for 3.0 spec")
	})
}

func TestOAS3Document(t *testing.T) {
	t.Run("returns document for OAS 3.x spec", func(t *testing.T) {
		result, err := ParseWithOptions(
			WithFilePath("../testdata/petstore-3.0.yaml"),
		)
		require.NoError(t, err)

		doc, ok := result.OAS3Document()
		assert.True(t, ok, "OAS3Document should return true for 3.x spec")
		assert.NotNil(t, doc, "Document should not be nil")
		assert.Equal(t, "Petstore API", doc.Info.Title)
	})

	t.Run("returns false for OAS 2.0 spec", func(t *testing.T) {
		result, err := ParseWithOptions(
			WithFilePath("../testdata/petstore-2.0.yaml"),
		)
		require.NoError(t, err)

		doc, ok := result.OAS3Document()
		assert.False(t, ok, "OAS3Document should return false for 2.0 spec")
		assert.Nil(t, doc, "Document should be nil for 2.0 spec")
	})

	t.Run("works with OAS 3.1 spec", func(t *testing.T) {
		result, err := ParseWithOptions(
			WithFilePath("../testdata/petstore-3.1.yaml"),
		)
		require.NoError(t, err)

		doc, ok := result.OAS3Document()
		assert.True(t, ok, "OAS3Document should return true for 3.1 spec")
		assert.NotNil(t, doc)
	})

	t.Run("works with OAS 3.2 spec", func(t *testing.T) {
		result, err := ParseWithOptions(
			WithFilePath("../testdata/petstore-3.2.yaml"),
		)
		require.NoError(t, err)

		doc, ok := result.OAS3Document()
		assert.True(t, ok, "OAS3Document should return true for 3.2 spec")
		assert.NotNil(t, doc)
	})
}

func TestIsOAS2(t *testing.T) {
	t.Run("returns true for OAS 2.0 spec", func(t *testing.T) {
		result, err := ParseWithOptions(
			WithFilePath("../testdata/petstore-2.0.yaml"),
		)
		require.NoError(t, err)

		assert.True(t, result.IsOAS2(), "IsOAS2 should return true for 2.0 spec")
	})

	t.Run("returns false for OAS 3.0 spec", func(t *testing.T) {
		result, err := ParseWithOptions(
			WithFilePath("../testdata/petstore-3.0.yaml"),
		)
		require.NoError(t, err)

		assert.False(t, result.IsOAS2(), "IsOAS2 should return false for 3.0 spec")
	})

	t.Run("returns false for OAS 3.1 spec", func(t *testing.T) {
		result, err := ParseWithOptions(
			WithFilePath("../testdata/petstore-3.1.yaml"),
		)
		require.NoError(t, err)

		assert.False(t, result.IsOAS2(), "IsOAS2 should return false for 3.1 spec")
	})

	t.Run("returns false for OAS 3.2 spec", func(t *testing.T) {
		result, err := ParseWithOptions(
			WithFilePath("../testdata/petstore-3.2.yaml"),
		)
		require.NoError(t, err)

		assert.False(t, result.IsOAS2(), "IsOAS2 should return false for 3.2 spec")
	})
}

func TestIsOAS3(t *testing.T) {
	t.Run("returns false for OAS 2.0 spec", func(t *testing.T) {
		result, err := ParseWithOptions(
			WithFilePath("../testdata/petstore-2.0.yaml"),
		)
		require.NoError(t, err)

		assert.False(t, result.IsOAS3(), "IsOAS3 should return false for 2.0 spec")
	})

	t.Run("returns true for OAS 3.0 spec", func(t *testing.T) {
		result, err := ParseWithOptions(
			WithFilePath("../testdata/petstore-3.0.yaml"),
		)
		require.NoError(t, err)

		assert.True(t, result.IsOAS3(), "IsOAS3 should return true for 3.0 spec")
	})

	t.Run("returns true for OAS 3.1 spec", func(t *testing.T) {
		result, err := ParseWithOptions(
			WithFilePath("../testdata/petstore-3.1.yaml"),
		)
		require.NoError(t, err)

		assert.True(t, result.IsOAS3(), "IsOAS3 should return true for 3.1 spec")
	})

	t.Run("returns true for OAS 3.2 spec", func(t *testing.T) {
		result, err := ParseWithOptions(
			WithFilePath("../testdata/petstore-3.2.yaml"),
		)
		require.NoError(t, err)

		assert.True(t, result.IsOAS3(), "IsOAS3 should return true for 3.2 spec")
	})

	t.Run("returns false for unknown version", func(t *testing.T) {
		// Create a ParseResult with an unknown OASVersion
		result := &ParseResult{
			OASVersion: Unknown,
		}
		assert.False(t, result.IsOAS3(), "IsOAS3 should return false for unknown version")
	})
}

func TestIsOAS3AllVersions(t *testing.T) {
	// Test all known OAS 3.x versions return true for IsOAS3
	oas3Versions := []struct {
		version    OASVersion
		versionStr string
	}{
		{OASVersion300, "3.0.0"},
		{OASVersion301, "3.0.1"},
		{OASVersion302, "3.0.2"},
		{OASVersion303, "3.0.3"},
		{OASVersion304, "3.0.4"},
		{OASVersion310, "3.1.0"},
		{OASVersion311, "3.1.1"},
		{OASVersion312, "3.1.2"},
		{OASVersion320, "3.2.0"},
	}

	for _, tt := range oas3Versions {
		t.Run("OAS_"+tt.versionStr, func(t *testing.T) {
			result := &ParseResult{
				OASVersion: tt.version,
			}
			assert.True(t, result.IsOAS3(), "IsOAS3 should return true for %s", tt.versionStr)
			assert.False(t, result.IsOAS2(), "IsOAS2 should return false for %s", tt.versionStr)
		})
	}
}
