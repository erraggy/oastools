package joiner

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// oas3JoinTestCase represents a test case for joining OAS3 documents
type oas3JoinTestCase struct {
	name           string
	files          []string
	config         JoinerConfig
	expectError    bool
	errorContains  string
	validateResult func(*testing.T, *JoinResult)
}

func TestJoinOAS3_SuccessfulJoins(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	tests := []oas3JoinTestCase{
		{
			name: "successful join with no collisions",
			files: []string{
				filepath.Join(testdataDir, "join-base-3.0.yaml"),
				filepath.Join(testdataDir, "join-extension-3.0.yaml"),
			},
			config:      DefaultConfig(),
			expectError: false,
			validateResult: func(t *testing.T, result *JoinResult) {
				doc, ok := result.Document.(*parser.OAS3Document)
				if !ok {
					t.Fatalf("expected *parser.OAS3Document, got %T", result.Document)
				}

				// Check that we have paths from both documents
				if len(doc.Paths) != 3 {
					t.Errorf("expected 3 paths, got %d", len(doc.Paths))
				}

				// Check specific paths exist
				if doc.Paths["/users"] == nil {
					t.Error("expected /users path from base document")
				}
				if doc.Paths["/users/{userId}"] == nil {
					t.Error("expected /users/{userId} path from base document")
				}
				if doc.Paths["/products"] == nil {
					t.Error("expected /products path from extension document")
				}

				// Check that schemas from both documents are present
				if doc.Components == nil {
					t.Fatal("expected components to be present")
				}
				if doc.Components.Schemas["User"] == nil {
					t.Error("expected User schema from base document")
				}
				if doc.Components.Schemas["Product"] == nil {
					t.Error("expected Product schema from extension document")
				}

				// Check that security schemes from both documents are present
				if doc.Components.SecuritySchemes["bearerAuth"] == nil {
					t.Error("expected bearerAuth security scheme from base document")
				}
				if doc.Components.SecuritySchemes["apiKey"] == nil {
					t.Error("expected apiKey security scheme from extension document")
				}

				// Check that servers are merged
				if len(doc.Servers) != 2 {
					t.Errorf("expected 2 servers, got %d", len(doc.Servers))
				}
			},
		},
		{
			name: "successful join with 3 documents",
			files: []string{
				filepath.Join(testdataDir, "join-base-3.0.yaml"),
				filepath.Join(testdataDir, "join-extension-3.0.yaml"),
				filepath.Join(testdataDir, "join-additional-3.0.yaml"),
			},
			config:      DefaultConfig(),
			expectError: false,
			validateResult: func(t *testing.T, result *JoinResult) {
				doc, ok := result.Document.(*parser.OAS3Document)
				if !ok {
					t.Fatalf("expected *parser.OAS3Document, got %T", result.Document)
				}

				// Check that we have paths from all three documents
				if len(doc.Paths) != 5 {
					t.Errorf("expected 5 paths (2 from base, 1 from extension, 2 from additional), got %d", len(doc.Paths))
				}

				// Check specific paths from all documents
				expectedPaths := []string{"/users", "/users/{userId}", "/products", "/orders", "/orders/{orderId}"}
				for _, path := range expectedPaths {
					if doc.Paths[path] == nil {
						t.Errorf("expected path '%s' to be present", path)
					}
				}

				// Check that schemas from all documents are present
				if doc.Components == nil {
					t.Fatal("expected components to be present")
				}
				expectedSchemas := []string{"User", "UserList", "Product", "Order", "OrderList"}
				for _, schema := range expectedSchemas {
					if doc.Components.Schemas[schema] == nil {
						t.Errorf("expected schema '%s' to be present", schema)
					}
				}

				// Check that security schemes from all documents are present
				if doc.Components.SecuritySchemes["bearerAuth"] == nil {
					t.Error("expected bearerAuth security scheme")
				}
				if doc.Components.SecuritySchemes["apiKey"] == nil {
					t.Error("expected apiKey security scheme")
				}
				if doc.Components.SecuritySchemes["oauth2"] == nil {
					t.Error("expected oauth2 security scheme")
				}

				// Check that servers are merged (3 servers total)
				if len(doc.Servers) != 3 {
					t.Errorf("expected 3 servers (one from each document), got %d", len(doc.Servers))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runJoinOAS3Test(t, tt)
		})
	}
}

func TestJoinOAS3_CollisionStrategies(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	tests := []oas3JoinTestCase{
		{
			name: "fail on path collision",
			files: []string{
				filepath.Join(testdataDir, "join-base-3.0.yaml"),
				filepath.Join(testdataDir, "join-collision-3.0.yaml"),
			},
			config:        DefaultConfig(),
			expectError:   true,
			errorContains: "collision in paths: '/users'",
		},
		{
			name: "accept left on path collision",
			files: []string{
				filepath.Join(testdataDir, "join-base-3.0.yaml"),
				filepath.Join(testdataDir, "join-collision-3.0.yaml"),
			},
			config: JoinerConfig{
				PathStrategy:      StrategyAcceptLeft,
				SchemaStrategy:    StrategyAcceptLeft,
				ComponentStrategy: StrategyAcceptLeft,
				DeduplicateTags:   true,
				MergeArrays:       true,
			},
			expectError: false,
			validateResult: func(t *testing.T, result *JoinResult) {
				doc, ok := result.Document.(*parser.OAS3Document)
				if !ok {
					t.Fatalf("expected *parser.OAS3Document, got %T", result.Document)
				}

				// Check that we kept the first /users path (GET operation, not POST)
				if doc.Paths["/users"] == nil {
					t.Fatal("expected /users path")
				}
				if doc.Paths["/users"].Get == nil {
					t.Error("expected GET operation from first document")
				}
				if doc.Paths["/users"].Post != nil {
					t.Error("should not have POST operation (should be from first document)")
				}

				// Check collision count
				if result.CollisionCount != 2 {
					t.Errorf("expected 2 collisions (path + schema), got %d", result.CollisionCount)
				}
			},
		},
		{
			name: "accept right on schema collision",
			files: []string{
				filepath.Join(testdataDir, "join-base-3.0.yaml"),
				filepath.Join(testdataDir, "join-collision-3.0.yaml"),
			},
			config: JoinerConfig{
				PathStrategy:      StrategyAcceptLeft,
				SchemaStrategy:    StrategyAcceptRight,
				ComponentStrategy: StrategyAcceptLeft,
				DeduplicateTags:   true,
				MergeArrays:       true,
			},
			expectError: false,
			validateResult: func(t *testing.T, result *JoinResult) {
				doc, ok := result.Document.(*parser.OAS3Document)
				if !ok {
					t.Fatalf("expected *parser.OAS3Document, got %T", result.Document)
				}

				// Check that we overwrote the User schema with the second document's version
				if doc.Components.Schemas["User"] == nil {
					t.Fatal("expected User schema")
				}

				// The second document's User schema has 'username' instead of 'name'
				userSchema := doc.Components.Schemas["User"]
				if userSchema.Properties == nil {
					t.Fatal("expected User schema to have properties")
				}
				if userSchema.Properties["username"] == nil {
					t.Error("expected 'username' property from second document's User schema")
				}
			},
		},
		{
			name: "fail on schema collision with fail-on-paths strategy",
			files: []string{
				filepath.Join(testdataDir, "join-base-3.0.yaml"),
				filepath.Join(testdataDir, "join-collision-3.0.yaml"),
			},
			config: JoinerConfig{
				PathStrategy:      StrategyFailOnPaths,
				SchemaStrategy:    StrategyFailOnPaths,
				ComponentStrategy: StrategyFailOnPaths,
				DeduplicateTags:   true,
				MergeArrays:       true,
			},
			expectError:   true,
			errorContains: "collision in paths: '/users'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runJoinOAS3Test(t, tt)
		})
	}
}

func TestJoinOAS3_ErrorCases(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	tests := []oas3JoinTestCase{
		{
			name: "insufficient files",
			files: []string{
				filepath.Join(testdataDir, "join-base-3.0.yaml"),
			},
			config:        DefaultConfig(),
			expectError:   true,
			errorContains: "at least 2 specification files are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runJoinOAS3Test(t, tt)
		})
	}
}

// runJoinOAS3Test executes a single OAS3 join test case
func runJoinOAS3Test(t *testing.T, tt oas3JoinTestCase) {
	j := New(tt.config)
	result, err := j.Join(tt.files)

	if tt.expectError {
		if err == nil {
			t.Fatalf("expected error containing '%s', got nil", tt.errorContains)
		}
		if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
			t.Errorf("expected error containing '%s', got '%s'", tt.errorContains, err.Error())
		}
		return
	}

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if tt.validateResult != nil {
		tt.validateResult(t, result)
	}
}

func TestJoinOAS2Documents(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	tests := []struct {
		name           string
		files          []string
		config         JoinerConfig
		expectError    bool
		validateResult func(*testing.T, *JoinResult)
	}{
		{
			name: "successful join OAS 2.0",
			files: []string{
				filepath.Join(testdataDir, "join-base-2.0.yaml"),
				filepath.Join(testdataDir, "join-extension-2.0.yaml"),
			},
			config:      DefaultConfig(),
			expectError: false,
			validateResult: func(t *testing.T, result *JoinResult) {
				doc, ok := result.Document.(*parser.OAS2Document)
				if !ok {
					t.Fatalf("expected *parser.OAS2Document, got %T", result.Document)
				}

				// Check that we have paths from both documents
				if len(doc.Paths) != 2 {
					t.Errorf("expected 2 paths, got %d", len(doc.Paths))
				}

				// Check that definitions from both documents are present
				if doc.Definitions["User"] == nil {
					t.Error("expected User definition from base document")
				}
				if doc.Definitions["Product"] == nil {
					t.Error("expected Product definition from extension document")
				}

				// Check that schemes are merged
				if len(doc.Schemes) != 2 {
					t.Errorf("expected 2 schemes (https, http), got %d", len(doc.Schemes))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := New(tt.config)
			result, err := j.Join(tt.files)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestVersionCompatibility(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	tests := []struct {
		name        string
		files       []string
		expectError bool
	}{
		{
			name: "incompatible versions (2.0 and 3.0)",
			files: []string{
				filepath.Join(testdataDir, "join-base-2.0.yaml"),
				filepath.Join(testdataDir, "join-base-3.0.yaml"),
			},
			expectError: true,
		},
		{
			name: "compatible 3.x versions (3.0 and 3.1)",
			files: []string{
				filepath.Join(testdataDir, "petstore-3.0.yaml"),
				filepath.Join(testdataDir, "petstore-3.1.yaml"),
			},
			expectError: false, // Should succeed - all 3.x versions are compatible
		},
		{
			name: "compatible 3.x versions (3.0 and 3.2)",
			files: []string{
				filepath.Join(testdataDir, "petstore-3.0.yaml"),
				filepath.Join(testdataDir, "petstore-3.2.yaml"),
			},
			expectError: false, // Should succeed - all 3.x versions are compatible
		},
		{
			name: "compatible 3.x versions (3.1 and 3.2)",
			files: []string{
				filepath.Join(testdataDir, "petstore-3.1.yaml"),
				filepath.Join(testdataDir, "petstore-3.2.yaml"),
			},
			expectError: false, // Should succeed - all 3.x versions are compatible
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use accept-left strategy to allow path collisions when testing version compatibility
			// We're testing whether versions CAN be joined, not whether they have colliding content
			config := JoinerConfig{
				PathStrategy:      StrategyAcceptLeft,
				SchemaStrategy:    StrategyAcceptLeft,
				ComponentStrategy: StrategyAcceptLeft,
				DeduplicateTags:   true,
				MergeArrays:       true,
			}
			j := New(config)
			_, err := j.Join(tt.files)

			if tt.expectError && err == nil {
				t.Fatal("expected error for incompatible versions, got nil")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestWriteResult(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "joined.yaml")

	j := New(DefaultConfig())
	result, err := j.Join([]string{
		filepath.Join(testdataDir, "join-base-3.0.yaml"),
		filepath.Join(testdataDir, "join-extension-3.0.yaml"),
	})
	if err != nil {
		t.Fatalf("unexpected error from Join: %v", err)
	}

	err = j.WriteResult(result, outputPath)
	if err != nil {
		t.Fatalf("unexpected error from WriteResult: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("output file was not created")
	}

	// Verify file can be parsed
	p := parser.New()
	parseResult, err := p.Parse(outputPath)
	if err != nil {
		t.Fatalf("failed to parse output file: %v", err)
	}

	if parseResult.Version != "3.0.3" {
		t.Errorf("expected version 3.0.3, got %s", parseResult.Version)
	}
}

func TestCollisionStrategies(t *testing.T) {
	tests := []struct {
		name         string
		strategy     CollisionStrategy
		section      string
		shouldError  bool
		shouldAccept bool
	}{
		{
			name:        "fail on collision - paths",
			strategy:    StrategyFailOnCollision,
			section:     "paths",
			shouldError: true,
		},
		{
			name:        "fail on collision - schemas",
			strategy:    StrategyFailOnCollision,
			section:     "schemas",
			shouldError: true,
		},
		{
			name:        "fail on paths - paths",
			strategy:    StrategyFailOnPaths,
			section:     "paths",
			shouldError: true,
		},
		{
			name:        "fail on paths - schemas (should not error)",
			strategy:    StrategyFailOnPaths,
			section:     "schemas",
			shouldError: false,
		},
		{
			name:         "accept left",
			strategy:     StrategyAcceptLeft,
			section:      "paths",
			shouldError:  false,
			shouldAccept: true,
		},
		{
			name:         "accept right",
			strategy:     StrategyAcceptRight,
			section:      "paths",
			shouldError:  false,
			shouldAccept: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := New(DefaultConfig())
			err := j.handleCollision("test", tt.section, tt.strategy, "file1.yaml", "file2.yaml")

			if tt.shouldError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestJoinParsed(t *testing.T) {
	p := parser.New()
	p.ValidateStructure = true

	doc1, err := p.Parse("../testdata/join-base-3.0.yaml")
	require.NoError(t, err)
	require.NotNil(t, doc1)
	doc2, err := p.Parse("../testdata/join-extension-3.0.yaml")
	require.NoError(t, err)
	require.NotNil(t, doc2)

	j := New(DefaultConfig())
	result, err := j.JoinParsed([]parser.ParseResult{*doc1, *doc2})
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestJoinParsed_GenericSourceNameWarning(t *testing.T) {
	p := parser.New()
	p.ValidateStructure = true

	t.Run("warns for generic ParseBytes source names", func(t *testing.T) {
		doc1, err := p.Parse("../testdata/join-base-3.0.yaml")
		require.NoError(t, err)
		doc2, err := p.Parse("../testdata/join-extension-3.0.yaml")
		require.NoError(t, err)

		// Simulate documents parsed from bytes (generic source names)
		doc1.SourcePath = "ParseBytes.yaml"
		doc2.SourcePath = "ParseBytes.yaml"

		j := New(DefaultConfig())
		result, err := j.JoinParsed([]parser.ParseResult{*doc1, *doc2})
		require.NoError(t, err)

		// Should have warnings for both documents
		genericWarnings := result.StructuredWarnings.ByCategory(WarnGenericSourceName)
		assert.Len(t, genericWarnings, 2, "expected 2 generic source name warnings")

		// Verify warning content
		if len(genericWarnings) >= 2 {
			assert.Contains(t, genericWarnings[0].Message, "document 0")
			assert.Contains(t, genericWarnings[1].Message, "document 1")
			assert.Contains(t, genericWarnings[0].Message, "ParseResult.SourcePath")
		}
	})

	t.Run("warns for empty source names", func(t *testing.T) {
		doc1, err := p.Parse("../testdata/join-base-3.0.yaml")
		require.NoError(t, err)
		doc2, err := p.Parse("../testdata/join-extension-3.0.yaml")
		require.NoError(t, err)

		// Simulate documents with empty source names
		doc1.SourcePath = ""
		doc2.SourcePath = ""

		j := New(DefaultConfig())
		result, err := j.JoinParsed([]parser.ParseResult{*doc1, *doc2})
		require.NoError(t, err)

		genericWarnings := result.StructuredWarnings.ByCategory(WarnGenericSourceName)
		assert.Len(t, genericWarnings, 2)
		if len(genericWarnings) >= 1 {
			assert.Contains(t, genericWarnings[0].Message, "empty source name")
		}
	})

	t.Run("no warning when SourcePath is set to meaningful name", func(t *testing.T) {
		doc1, err := p.Parse("../testdata/join-base-3.0.yaml")
		require.NoError(t, err)
		doc2, err := p.Parse("../testdata/join-extension-3.0.yaml")
		require.NoError(t, err)

		// Set meaningful source names (as recommended for JoinParsed)
		doc1.SourcePath = "users-api"
		doc2.SourcePath = "billing-api"

		j := New(DefaultConfig())
		result, err := j.JoinParsed([]parser.ParseResult{*doc1, *doc2})
		require.NoError(t, err)

		// Should have no generic source name warnings
		genericWarnings := result.StructuredWarnings.ByCategory(WarnGenericSourceName)
		assert.Empty(t, genericWarnings, "no warnings expected when SourcePath is meaningful")
	})

	t.Run("warning appears in both StructuredWarnings and Warnings string slice", func(t *testing.T) {
		doc1, err := p.Parse("../testdata/join-base-3.0.yaml")
		require.NoError(t, err)
		doc2, err := p.Parse("../testdata/join-extension-3.0.yaml")
		require.NoError(t, err)

		// Set generic source name to trigger warning
		doc1.SourcePath = "ParseBytes.yaml"
		doc2.SourcePath = "users-api" // This one is fine

		j := New(DefaultConfig())
		result, err := j.JoinParsed([]parser.ParseResult{*doc1, *doc2})
		require.NoError(t, err)

		// Verify structured warning exists
		genericWarnings := result.StructuredWarnings.ByCategory(WarnGenericSourceName)
		require.Len(t, genericWarnings, 1, "expected 1 generic source name warning")

		// Verify warning message is also in the string slice for backward compatibility
		found := false
		for _, w := range result.Warnings {
			if strings.Contains(w, "document 0") && strings.Contains(w, "ParseBytes.yaml") {
				found = true
				break
			}
		}
		assert.True(t, found, "expected warning message in result.Warnings string slice")
	})
}

// TestJSONFormatPreservation tests that JSON input produces JSON output
func TestJSONFormatPreservation(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	// Test with JSON files - first need to create JSON versions
	j := New(DefaultConfig())
	result, err := j.Join([]string{
		filepath.Join(testdataDir, "minimal-oas2.json"),
		filepath.Join(testdataDir, "minimal-oas2.json"), // joining same file twice for simplicity
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify source format was detected as JSON from first file
	assert.Equal(t, parser.SourceFormatJSON, result.SourceFormat)
	t.Logf("Successfully verified JSON format detection for joined documents")
}

// TestYAMLFormatPreservation tests that YAML input preserves YAML format
func TestYAMLFormatPreservation(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	// Test with YAML files
	j := New(DefaultConfig())
	result, err := j.Join([]string{
		filepath.Join(testdataDir, "minimal-oas2.yaml"),
		filepath.Join(testdataDir, "minimal-oas2.yaml"), // joining same file twice for simplicity
	})

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify source format was detected as YAML from first file
	assert.Equal(t, parser.SourceFormatYAML, result.SourceFormat)
	t.Logf("Successfully verified YAML format detection for joined documents")
}

// TestMixedFormatJoining tests that joining JSON + YAML uses format from first file
func TestMixedFormatJoining(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	tests := []struct {
		name           string
		files          []string
		expectedFormat parser.SourceFormat
	}{
		{
			name: "JSON first, then YAML - should output JSON",
			files: []string{
				filepath.Join(testdataDir, "minimal-oas2.json"),
				filepath.Join(testdataDir, "minimal-oas2.yaml"),
			},
			expectedFormat: parser.SourceFormatJSON,
		},
		{
			name: "YAML first, then JSON - should output YAML",
			files: []string{
				filepath.Join(testdataDir, "minimal-oas2.yaml"),
				filepath.Join(testdataDir, "minimal-oas2.json"),
			},
			expectedFormat: parser.SourceFormatYAML,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := New(DefaultConfig())
			result, err := j.Join(tt.files)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.expectedFormat, result.SourceFormat,
				"Expected format from first file to be preserved")
			t.Logf("Successfully verified format preservation: %s", tt.expectedFormat)
		})
	}
}

// TestMergeUniqueStrings_OverflowSafety tests that mergeUniqueStrings handles overflow gracefully.
// While triggering actual overflow is impractical in tests (would require enormous slices),
// this test documents the expected behavior: the function should work correctly even when
// capacity calculation defaults to 0 (falling back to dynamic growth via append).
func TestMergeUniqueStrings_OverflowSafety(t *testing.T) {
	j := New(DefaultConfig())

	tests := []struct {
		name     string
		a        []string
		b        []string
		expected []string
	}{
		{
			name:     "basic merge with duplicates",
			a:        []string{"a", "b", "c"},
			b:        []string{"b", "c", "d"},
			expected: []string{"a", "b", "c", "d"},
		},
		{
			name:     "empty first slice",
			a:        []string{},
			b:        []string{"x", "y"},
			expected: []string{"x", "y"},
		},
		{
			name:     "empty second slice",
			a:        []string{"x", "y"},
			b:        []string{},
			expected: []string{"x", "y"},
		},
		{
			name:     "both empty",
			a:        []string{},
			b:        []string{},
			expected: []string{},
		},
		{
			name:     "all duplicates",
			a:        []string{"a", "b"},
			b:        []string{"a", "b"},
			expected: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := j.mergeUniqueStrings(tt.a, tt.b)

			// Verify result contains all expected elements
			assert.Equal(t, len(tt.expected), len(result),
				"Result should have %d elements, got %d", len(tt.expected), len(result))

			// Verify each expected element is present (order may vary due to map iteration)
			resultMap := make(map[string]bool)
			for _, s := range result {
				resultMap[s] = true
			}

			for _, expected := range tt.expected {
				assert.True(t, resultMap[expected],
					"Expected %q to be in result %v", expected, result)
			}

			// Verify no duplicates in result
			assert.Equal(t, len(result), len(resultMap),
				"Result should not contain duplicates")
		})
	}
}

// TestJoinWithOptions_FilePaths tests the functional options API with file paths
func TestJoinWithOptions_FilePaths(t *testing.T) {
	result, err := JoinWithOptions(
		WithFilePaths("../testdata/join-base-3.0.yaml", "../testdata/join-extension-3.0.yaml"),
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Document)
}

// TestJoinWithOptions_Parsed tests the functional options API with parsed documents
func TestJoinWithOptions_Parsed(t *testing.T) {
	doc1, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/join-base-3.0.yaml"),
		parser.WithValidateStructure(true),
	)
	require.NoError(t, err)

	doc2, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/join-extension-3.0.yaml"),
		parser.WithValidateStructure(true),
	)
	require.NoError(t, err)

	result, err := JoinWithOptions(
		WithParsed(*doc1, *doc2),
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestJoinWithOptions_MixedSources tests mixing file paths and parsed docs
func TestJoinWithOptions_MixedSources(t *testing.T) {
	doc1, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/join-base-3.0.yaml"),
		parser.WithValidateStructure(true),
	)
	require.NoError(t, err)

	result, err := JoinWithOptions(
		WithParsed(*doc1),
		WithFilePaths("../testdata/join-extension-3.0.yaml"),
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestJoinWithOptions_WithConfig tests using WithConfig option
func TestJoinWithOptions_WithConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.PathStrategy = StrategyAcceptLeft
	cfg.DeduplicateTags = false

	result, err := JoinWithOptions(
		WithFilePaths("../testdata/join-base-3.0.yaml", "../testdata/join-extension-3.0.yaml"),
		WithConfig(cfg),
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestJoinWithOptions_IndividualStrategies tests setting individual strategies
func TestJoinWithOptions_IndividualStrategies(t *testing.T) {
	result, err := JoinWithOptions(
		WithFilePaths("../testdata/join-base-3.0.yaml", "../testdata/join-extension-3.0.yaml"),
		WithPathStrategy(StrategyAcceptLeft),
		WithSchemaStrategy(StrategyAcceptRight),
		WithComponentStrategy(StrategyFailOnCollision),
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestJoinWithOptions_NotEnoughDocuments tests error when < 2 documents
func TestJoinWithOptions_NotEnoughDocuments(t *testing.T) {
	_, err := JoinWithOptions(
		WithFilePaths("../testdata/join-base-3.0.yaml"),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 2 documents are required")
}

// TestJoinWithOptions_MixedSources_ParseError tests error when a file path fails to parse in mixed mode
func TestJoinWithOptions_MixedSources_ParseError(t *testing.T) {
	doc1, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/join-base-3.0.yaml"),
		parser.WithValidateStructure(true),
	)
	require.NoError(t, err)

	// Mix parsed doc with a non-existent file
	_, err = JoinWithOptions(
		WithParsed(*doc1),
		WithFilePaths("nonexistent-file.yaml"),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "joiner: failed to parse")
}

// TestJoinOAS3_RenameStrategies tests the rename-left and rename-right strategies
func TestJoinOAS3_RenameStrategies(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	t.Run("rename-left strategy", func(t *testing.T) {
		config := DefaultConfig()
		config.SchemaStrategy = StrategyRenameLeft

		j := New(config)
		result, err := j.Join([]string{
			filepath.Join(testdataDir, "join-collision-rename-base-3.0.yaml"),
			filepath.Join(testdataDir, "join-collision-rename-ext-3.0.yaml"),
		})

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// Should have both User schemas: original "User" and renamed one
		assert.NotNil(t, doc.Components.Schemas["User"])
		// Find the renamed schema (should be User_join-collision-rename-base-3.0)
		foundRenamed := false
		for name := range doc.Components.Schemas {
			if strings.HasPrefix(name, "User_") && name != "User" {
				foundRenamed = true
				break
			}
		}
		assert.True(t, foundRenamed, "renamed schema should exist")

		// Should have 2 paths
		assert.Equal(t, 2, len(doc.Paths))
	})

	t.Run("rename-right strategy", func(t *testing.T) {
		config := DefaultConfig()
		config.SchemaStrategy = StrategyRenameRight

		j := New(config)
		result, err := j.Join([]string{
			filepath.Join(testdataDir, "join-collision-rename-base-3.0.yaml"),
			filepath.Join(testdataDir, "join-collision-rename-ext-3.0.yaml"),
		})

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// Should have both User schemas: original "User" and renamed one
		assert.NotNil(t, doc.Components.Schemas["User"])
		// Find the renamed schema
		foundRenamed := false
		for name := range doc.Components.Schemas {
			if strings.HasPrefix(name, "User_") && name != "User" {
				foundRenamed = true
				break
			}
		}
		assert.True(t, foundRenamed, "renamed schema should exist")

		// Should have 2 paths
		assert.Equal(t, 2, len(doc.Paths))
	})
}

// TestJoinOAS3_DeduplicateStrategy tests the deduplicate strategy with equivalence detection
func TestJoinOAS3_DeduplicateStrategy(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	t.Run("deduplicate equivalent schemas", func(t *testing.T) {
		config := DefaultConfig()
		config.SchemaStrategy = StrategyDeduplicateEquivalent
		config.EquivalenceMode = "deep"

		j := New(config)
		result, err := j.Join([]string{
			filepath.Join(testdataDir, "join-equivalent-schemas-base-3.0.yaml"),
			filepath.Join(testdataDir, "join-equivalent-schemas-ext-3.0.yaml"),
		})

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// Should have only one Product schema (deduplicated)
		assert.NotNil(t, doc.Components.Schemas["Product"])
		assert.Equal(t, 1, len(doc.Components.Schemas))

		// Should have both paths
		assert.Equal(t, 2, len(doc.Paths))
	})

	t.Run("deduplicate fails on non-equivalent schemas", func(t *testing.T) {
		config := DefaultConfig()
		config.SchemaStrategy = StrategyDeduplicateEquivalent
		config.EquivalenceMode = "deep"

		j := New(config)
		_, err := j.Join([]string{
			filepath.Join(testdataDir, "join-collision-rename-base-3.0.yaml"),
			filepath.Join(testdataDir, "join-collision-rename-ext-3.0.yaml"),
		})

		// Should fail because User schemas are different
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not equivalent")
	})
}

// TestJoinOAS3_DiscriminatorRewriting tests discriminator reference rewriting
func TestJoinOAS3_DiscriminatorRewriting(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	t.Run("discriminator with rename-right", func(t *testing.T) {
		config := DefaultConfig()
		config.SchemaStrategy = StrategyRenameRight

		j := New(config)
		result, err := j.Join([]string{
			filepath.Join(testdataDir, "join-discriminator-base-3.0.yaml"),
			filepath.Join(testdataDir, "join-discriminator-ext-3.0.yaml"),
		})

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// Should have Pet schema from base
		assert.NotNil(t, doc.Components.Schemas["Pet"])

		// Should have Dog schema from base and renamed Dog from ext
		assert.NotNil(t, doc.Components.Schemas["Dog"])

		// Find renamed Dog schema
		foundRenamed := false
		for name := range doc.Components.Schemas {
			if strings.HasPrefix(name, "Dog_") {
				foundRenamed = true
				break
			}
		}
		assert.True(t, foundRenamed, "renamed Dog schema should exist")

		// Check discriminator mapping was updated
		pet := doc.Components.Schemas["Pet"]
		if pet.Discriminator != nil && pet.Discriminator.Mapping != nil {
			// The discriminator mapping should still reference Dog (from base)
			assert.Contains(t, pet.Discriminator.Mapping, "dog")
		}
	})
}

// TestJoinOAS3_CircularReferences tests circular reference handling
func TestJoinOAS3_CircularReferences(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	t.Run("circular refs with rename", func(t *testing.T) {
		config := DefaultConfig()
		config.SchemaStrategy = StrategyRenameLeft

		j := New(config)
		result, err := j.Join([]string{
			filepath.Join(testdataDir, "join-circular-refs-base-3.0.yaml"),
			filepath.Join(testdataDir, "join-circular-refs-ext-3.0.yaml"),
		})

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// Should have two Node schemas
		assert.NotNil(t, doc.Components.Schemas["Node"])
		foundRenamed := false
		for name := range doc.Components.Schemas {
			if strings.HasPrefix(name, "Node_") {
				foundRenamed = true
				break
			}
		}
		assert.True(t, foundRenamed, "renamed Node schema should exist")

		// Should have 2 paths
		assert.Equal(t, 2, len(doc.Paths))
	})
}

// TestJoinWithOptions_NewStrategies tests the functional options API with new strategies
func TestJoinWithOptions_NewStrategies(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	t.Run("with rename template option", func(t *testing.T) {
		result, err := JoinWithOptions(
			WithFilePaths(
				filepath.Join(testdataDir, "join-collision-rename-base-3.0.yaml"),
				filepath.Join(testdataDir, "join-collision-rename-ext-3.0.yaml"),
			),
			WithSchemaStrategy(StrategyRenameRight),
			WithRenameTemplate("{{.Name}}_{{.Source}}"),
		)

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// Should have both User schemas
		assert.NotNil(t, doc.Components.Schemas["User"])
		foundRenamed := false
		for name := range doc.Components.Schemas {
			if strings.HasPrefix(name, "User_") {
				foundRenamed = true
				break
			}
		}
		assert.True(t, foundRenamed)
	})

	t.Run("with equivalence mode option", func(t *testing.T) {
		result, err := JoinWithOptions(
			WithFilePaths(
				filepath.Join(testdataDir, "join-equivalent-schemas-base-3.0.yaml"),
				filepath.Join(testdataDir, "join-equivalent-schemas-ext-3.0.yaml"),
			),
			WithSchemaStrategy(StrategyDeduplicateEquivalent),
			WithEquivalenceMode("deep"),
		)

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// Should have only one Product schema
		assert.Equal(t, 1, len(doc.Components.Schemas))
	})

	t.Run("with custom rename template patterns", func(t *testing.T) {
		tests := []struct {
			name           string
			template       string
			expectedPrefix string
		}{
			{"name and source", "{{.Name}}_{{.Source}}", "User_join_collision_rename"},
			{"name and index", "{{.Name}}_v{{.Index}}", "User_v"},
			{"source only prefix", "{{.Source}}_{{.Name}}", "join_collision_rename"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := JoinWithOptions(
					WithFilePaths(
						filepath.Join(testdataDir, "join-collision-rename-base-3.0.yaml"),
						filepath.Join(testdataDir, "join-collision-rename-ext-3.0.yaml"),
					),
					WithSchemaStrategy(StrategyRenameRight),
					WithRenameTemplate(tt.template),
				)

				require.NoError(t, err)
				require.NotNil(t, result)

				doc, ok := result.Document.(*parser.OAS3Document)
				require.True(t, ok)

				// Check that renamed schema follows the expected pattern
				foundExpected := false
				for name := range doc.Components.Schemas {
					if strings.HasPrefix(name, tt.expectedPrefix) {
						foundExpected = true
						break
					}
				}
				assert.True(t, foundExpected, "expected schema name with prefix %s", tt.expectedPrefix)
			})
		}
	})

	t.Run("with invalid template falls back to default", func(t *testing.T) {
		// Suppress expected WARN output from template execution fallback
		original := joinerLogger
		joinerLogger = slog.New(slog.NewTextHandler(io.Discard, nil))
		t.Cleanup(func() { joinerLogger = original })

		result, err := JoinWithOptions(
			WithFilePaths(
				filepath.Join(testdataDir, "join-collision-rename-base-3.0.yaml"),
				filepath.Join(testdataDir, "join-collision-rename-ext-3.0.yaml"),
			),
			WithSchemaStrategy(StrategyRenameRight),
			WithRenameTemplate("{{.InvalidField}}"), // Invalid template field
		)

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// Should still have both schemas (fallback worked)
		assert.NotNil(t, doc.Components.Schemas["User"])
		foundRenamed := false
		for name := range doc.Components.Schemas {
			if strings.HasPrefix(name, "User_") {
				foundRenamed = true
				break
			}
		}
		assert.True(t, foundRenamed, "should have renamed schema with fallback pattern")
	})
}

// TestJoinOAS3_NamespacePrefix tests the namespace prefix functionality
func TestJoinOAS3_NamespacePrefix(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")
	basePath := filepath.Join(testdataDir, "join-collision-rename-base-3.0.yaml")
	extPath := filepath.Join(testdataDir, "join-collision-rename-ext-3.0.yaml")

	t.Run("namespace prefix on collision with rename-right", func(t *testing.T) {
		config := DefaultConfig()
		config.SchemaStrategy = StrategyRenameRight
		config.NamespacePrefix = map[string]string{
			extPath: "Ext",
		}

		j := New(config)
		result, err := j.Join([]string{basePath, extPath})

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// Should have original User and prefixed Ext_User
		assert.NotNil(t, doc.Components.Schemas["User"], "original User schema should exist")
		assert.NotNil(t, doc.Components.Schemas["Ext_User"], "prefixed Ext_User schema should exist")
	})

	t.Run("always apply prefix", func(t *testing.T) {
		config := DefaultConfig()
		config.SchemaStrategy = StrategyAcceptLeft
		config.NamespacePrefix = map[string]string{
			extPath: "Ext",
		}
		config.AlwaysApplyPrefix = true

		j := New(config)
		result, err := j.Join([]string{basePath, extPath})

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// With AlwaysApplyPrefix, all schemas from ext should be prefixed
		// Original User from base should exist
		assert.NotNil(t, doc.Components.Schemas["User"], "original User schema should exist")
		// Extension User should be prefixed even though accept-left keeps original
		assert.NotNil(t, doc.Components.Schemas["Ext_User"], "prefixed Ext_User schema should exist")
	})

	t.Run("functional options WithNamespacePrefix", func(t *testing.T) {
		result, err := JoinWithOptions(
			WithFilePaths(basePath, extPath),
			WithSchemaStrategy(StrategyRenameRight),
			WithNamespacePrefix(extPath, "Api2"),
		)

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// Should have original User and prefixed Api2_User
		assert.NotNil(t, doc.Components.Schemas["User"], "original User schema should exist")
		assert.NotNil(t, doc.Components.Schemas["Api2_User"], "prefixed Api2_User schema should exist")
	})

	t.Run("functional options WithAlwaysApplyPrefix", func(t *testing.T) {
		result, err := JoinWithOptions(
			WithFilePaths(basePath, extPath),
			WithSchemaStrategy(StrategyAcceptLeft),
			WithNamespacePrefix(extPath, "V2"),
			WithAlwaysApplyPrefix(true),
		)

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// With AlwaysApplyPrefix, User from ext gets prefixed regardless of collision handling
		assert.NotNil(t, doc.Components.Schemas["User"], "original User from base should exist")
		assert.NotNil(t, doc.Components.Schemas["V2_User"], "prefixed V2_User should exist")
	})

	t.Run("namespace prefix on collision with rename-left", func(t *testing.T) {
		config := DefaultConfig()
		config.SchemaStrategy = StrategyRenameLeft
		config.NamespacePrefix = map[string]string{
			basePath: "Base",
		}

		j := New(config)
		result, err := j.Join([]string{basePath, extPath})

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// With rename-left, the left (base) schema gets renamed using prefix
		// Original name goes to the right (ext) schema
		assert.NotNil(t, doc.Components.Schemas["User"], "User schema (from ext) should exist")
		assert.NotNil(t, doc.Components.Schemas["Base_User"], "prefixed Base_User (from base) should exist")
	})
}

// TestGeneratePrefixedSchemaName tests the helper function for prefixed schema names
func TestGeneratePrefixedSchemaName(t *testing.T) {
	j := New(DefaultConfig())

	tests := []struct {
		name         string
		originalName string
		prefix       string
		expected     string
	}{
		{
			name:         "basic prefix",
			originalName: "User",
			prefix:       "Api",
			expected:     "Api_User",
		},
		{
			name:         "empty prefix returns original",
			originalName: "Schema",
			prefix:       "",
			expected:     "Schema",
		},
		{
			name:         "complex schema name",
			originalName: "UserProfile",
			prefix:       "Users",
			expected:     "Users_UserProfile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := j.generatePrefixedSchemaName(tt.originalName, tt.prefix)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetNamespacePrefix tests the namespace prefix lookup function
func TestGetNamespacePrefix(t *testing.T) {
	config := DefaultConfig()
	config.NamespacePrefix = map[string]string{
		"users.yaml":   "Users",
		"billing.yaml": "Billing",
	}

	j := New(config)

	t.Run("returns prefix for configured source", func(t *testing.T) {
		assert.Equal(t, "Users", j.getNamespacePrefix("users.yaml"))
		assert.Equal(t, "Billing", j.getNamespacePrefix("billing.yaml"))
	})

	t.Run("returns empty for unconfigured source", func(t *testing.T) {
		assert.Equal(t, "", j.getNamespacePrefix("other.yaml"))
	})

	t.Run("handles nil map", func(t *testing.T) {
		emptyConfig := DefaultConfig()
		emptyConfig.NamespacePrefix = nil
		j2 := New(emptyConfig)
		assert.Equal(t, "", j2.getNamespacePrefix("any.yaml"))
	})
}

// TestValidStrategies tests the ValidStrategies helper function
func TestValidStrategies(t *testing.T) {
	strategies := ValidStrategies()

	// Should return all valid strategy strings
	assert.Contains(t, strategies, string(StrategyAcceptLeft))
	assert.Contains(t, strategies, string(StrategyAcceptRight))
	assert.Contains(t, strategies, string(StrategyFailOnCollision))
	assert.Contains(t, strategies, string(StrategyFailOnPaths))
	assert.Contains(t, strategies, string(StrategyRenameLeft))
	assert.Contains(t, strategies, string(StrategyRenameRight))
	assert.Contains(t, strategies, string(StrategyDeduplicateEquivalent))

	// Should have exactly 7 strategies
	assert.Equal(t, 7, len(strategies))
}

// TestIsValidStrategy tests the IsValidStrategy helper function
func TestIsValidStrategy(t *testing.T) {
	tests := []struct {
		strategy string
		expected bool
	}{
		{"accept-left", true},
		{"accept-right", true},
		{"fail", true},
		{"fail-on-paths", true},
		{"rename-left", true},
		{"rename-right", true},
		{"deduplicate", true},
		{"invalid", false},
		{"", false},
		{"AcceptLeft", false}, // Case sensitive
		{"ACCEPT-LEFT", false},
	}

	for _, tt := range tests {
		t.Run(tt.strategy, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsValidStrategy(tt.strategy))
		})
	}
}

// TestJoinWithOptions_FunctionalOptions tests additional functional options
func TestJoinWithOptions_FunctionalOptions(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")
	basePath := filepath.Join(testdataDir, "join-base-3.0.yaml")
	extPath := filepath.Join(testdataDir, "join-extension-3.0.yaml")

	t.Run("WithDefaultStrategy", func(t *testing.T) {
		result, err := JoinWithOptions(
			WithFilePaths(basePath, extPath),
			WithDefaultStrategy(StrategyAcceptLeft),
		)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("WithDeduplicateTags false", func(t *testing.T) {
		result, err := JoinWithOptions(
			WithFilePaths(basePath, extPath),
			WithDeduplicateTags(false),
		)
		require.NoError(t, err)
		assert.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// Without deduplication, tags may have duplicates
		// The result document is valid even if Tags is nil or empty
		assert.NotNil(t, doc)
	})

	t.Run("WithDeduplicateTags true", func(t *testing.T) {
		result, err := JoinWithOptions(
			WithFilePaths(basePath, extPath),
			WithDeduplicateTags(true),
		)
		require.NoError(t, err)
		assert.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)
		// With deduplication enabled, the result document is valid
		assert.NotNil(t, doc)
	})

	t.Run("WithMergeArrays false", func(t *testing.T) {
		result, err := JoinWithOptions(
			WithFilePaths(basePath, extPath),
			WithMergeArrays(false),
		)
		require.NoError(t, err)
		assert.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// Without array merging, should only have servers from first document
		assert.Equal(t, 1, len(doc.Servers))
	})

	t.Run("WithMergeArrays true", func(t *testing.T) {
		result, err := JoinWithOptions(
			WithFilePaths(basePath, extPath),
			WithMergeArrays(true),
		)
		require.NoError(t, err)
		assert.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS3Document)
		require.True(t, ok)

		// With array merging, should have servers from both documents
		assert.Equal(t, 2, len(doc.Servers))
	})

	t.Run("WithCollisionReport enabled", func(t *testing.T) {
		collisionBasePath := filepath.Join(testdataDir, "join-collision-rename-base-3.0.yaml")
		collisionExtPath := filepath.Join(testdataDir, "join-collision-rename-ext-3.0.yaml")

		result, err := JoinWithOptions(
			WithFilePaths(collisionBasePath, collisionExtPath),
			WithSchemaStrategy(StrategyAcceptLeft),
			WithCollisionReport(true),
		)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Collision report should be populated
		assert.NotNil(t, result.CollisionDetails)
		assert.Greater(t, len(result.CollisionDetails.Events), 0)
	})

	t.Run("WithCollisionReport disabled", func(t *testing.T) {
		result, err := JoinWithOptions(
			WithFilePaths(basePath, extPath),
			WithCollisionReport(false),
		)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Collision report should be nil when disabled
		assert.Nil(t, result.CollisionDetails)
	})
}

// TestJoinOAS2_NamespacePrefix tests namespace prefix functionality for OAS 2.0 documents
func TestJoinOAS2_NamespacePrefix(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")
	basePath := filepath.Join(testdataDir, "join-collision-rename-base-2.0.yaml")
	extPath := filepath.Join(testdataDir, "join-collision-rename-ext-2.0.yaml")

	t.Run("namespace prefix on collision with rename-right", func(t *testing.T) {
		config := DefaultConfig()
		config.SchemaStrategy = StrategyRenameRight
		config.NamespacePrefix = map[string]string{
			extPath: "Ext",
		}

		j := New(config)
		result, err := j.Join([]string{basePath, extPath})

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS2Document)
		require.True(t, ok)

		// Should have original User and prefixed Ext_User
		assert.NotNil(t, doc.Definitions["User"], "original User definition should exist")
		assert.NotNil(t, doc.Definitions["Ext_User"], "prefixed Ext_User definition should exist")
	})

	t.Run("always apply prefix", func(t *testing.T) {
		config := DefaultConfig()
		config.SchemaStrategy = StrategyAcceptLeft
		config.NamespacePrefix = map[string]string{
			extPath: "Ext",
		}
		config.AlwaysApplyPrefix = true

		j := New(config)
		result, err := j.Join([]string{basePath, extPath})

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS2Document)
		require.True(t, ok)

		// With AlwaysApplyPrefix, all definitions from ext should be prefixed
		assert.NotNil(t, doc.Definitions["User"], "original User definition should exist")
		assert.NotNil(t, doc.Definitions["Ext_User"], "prefixed Ext_User definition should exist")
		assert.NotNil(t, doc.Definitions["Ext_UserList"], "prefixed Ext_UserList definition should exist")
	})

	t.Run("namespace prefix on collision with rename-left", func(t *testing.T) {
		config := DefaultConfig()
		config.SchemaStrategy = StrategyRenameLeft
		config.NamespacePrefix = map[string]string{
			basePath: "Base",
		}

		j := New(config)
		result, err := j.Join([]string{basePath, extPath})

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS2Document)
		require.True(t, ok)

		// With rename-left, the left (base) definition gets renamed using prefix
		assert.NotNil(t, doc.Definitions["User"], "User definition (from ext) should exist")
		assert.NotNil(t, doc.Definitions["Base_User"], "prefixed Base_User (from base) should exist")
	})

	t.Run("functional options for OAS2", func(t *testing.T) {
		result, err := JoinWithOptions(
			WithFilePaths(basePath, extPath),
			WithSchemaStrategy(StrategyRenameRight),
			WithNamespacePrefix(extPath, "Api2"),
		)

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS2Document)
		require.True(t, ok)

		// Should have original User and prefixed Api2_User
		assert.NotNil(t, doc.Definitions["User"], "original User definition should exist")
		assert.NotNil(t, doc.Definitions["Api2_User"], "prefixed Api2_User definition should exist")
	})
}

// TestJoinOAS2_RenameStrategies tests rename strategies for OAS 2.0 documents
func TestJoinOAS2_RenameStrategies(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")
	basePath := filepath.Join(testdataDir, "join-collision-rename-base-2.0.yaml")
	extPath := filepath.Join(testdataDir, "join-collision-rename-ext-2.0.yaml")

	t.Run("rename-left strategy", func(t *testing.T) {
		config := DefaultConfig()
		config.SchemaStrategy = StrategyRenameLeft

		j := New(config)
		result, err := j.Join([]string{basePath, extPath})

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS2Document)
		require.True(t, ok)

		// Should have both User definitions
		assert.NotNil(t, doc.Definitions["User"])
		// Find the renamed definition
		foundRenamed := false
		for name := range doc.Definitions {
			if strings.HasPrefix(name, "User_") && name != "User" {
				foundRenamed = true
				break
			}
		}
		assert.True(t, foundRenamed, "renamed definition should exist")

		// Should have 2 paths
		assert.Equal(t, 2, len(doc.Paths))
	})

	t.Run("rename-right strategy", func(t *testing.T) {
		config := DefaultConfig()
		config.SchemaStrategy = StrategyRenameRight

		j := New(config)
		result, err := j.Join([]string{basePath, extPath})

		require.NoError(t, err)
		require.NotNil(t, result)

		doc, ok := result.Document.(*parser.OAS2Document)
		require.True(t, ok)

		// Should have both User definitions
		assert.NotNil(t, doc.Definitions["User"])
		// Find the renamed definition
		foundRenamed := false
		for name := range doc.Definitions {
			if strings.HasPrefix(name, "User_") && name != "User" {
				foundRenamed = true
				break
			}
		}
		assert.True(t, foundRenamed, "renamed definition should exist")

		// Should have 2 paths
		assert.Equal(t, 2, len(doc.Paths))
	})

	t.Run("deduplicate strategy fails on non-equivalent", func(t *testing.T) {
		config := DefaultConfig()
		config.SchemaStrategy = StrategyDeduplicateEquivalent
		config.EquivalenceMode = "deep"

		j := New(config)
		_, err := j.Join([]string{basePath, extPath})

		// Should fail because User definitions are different
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not equivalent")
	})
}

func TestWithSourceMaps(t *testing.T) {
	// Create mock SourceMaps
	sm1 := parser.NewSourceMap()
	sm2 := parser.NewSourceMap()

	sourceMaps := map[string]*parser.SourceMap{
		"api1.yaml": sm1,
		"api2.yaml": sm2,
	}

	// Test WithSourceMaps option
	cfg := &joinConfig{
		filePaths:  make([]string, 0),
		parsedDocs: make([]parser.ParseResult, 0),
	}

	opt := WithSourceMaps(sourceMaps)
	err := opt(cfg)

	require.NoError(t, err)
	assert.Equal(t, sourceMaps, cfg.sourceMaps)
	assert.Same(t, sm1, cfg.sourceMaps["api1.yaml"])
	assert.Same(t, sm2, cfg.sourceMaps["api2.yaml"])
}

func TestWithSourceMaps_NilMap(t *testing.T) {
	cfg := &joinConfig{
		filePaths:  make([]string, 0),
		parsedDocs: make([]parser.ParseResult, 0),
	}

	opt := WithSourceMaps(nil)
	err := opt(cfg)

	require.NoError(t, err)
	assert.Nil(t, cfg.sourceMaps)
}

func TestJoiner_getLocation(t *testing.T) {
	// Create a SourceMap with test data
	sm := parser.NewSourceMap()

	// Set up test locations using the Copy method workaround
	// Since set() is unexported, we'll test via integration

	tests := []struct {
		name         string
		sourceMaps   map[string]*parser.SourceMap
		filePath     string
		jsonPath     string
		expectedLine int
		expectedCol  int
	}{
		{
			name:         "nil SourceMaps returns zeros",
			sourceMaps:   nil,
			filePath:     "test.yaml",
			jsonPath:     "$.components.schemas.User",
			expectedLine: 0,
			expectedCol:  0,
		},
		{
			name:         "missing file returns zeros",
			sourceMaps:   map[string]*parser.SourceMap{"other.yaml": sm},
			filePath:     "test.yaml",
			jsonPath:     "$.components.schemas.User",
			expectedLine: 0,
			expectedCol:  0,
		},
		{
			name:         "nil SourceMap for file returns zeros",
			sourceMaps:   map[string]*parser.SourceMap{"test.yaml": nil},
			filePath:     "test.yaml",
			jsonPath:     "$.components.schemas.User",
			expectedLine: 0,
			expectedCol:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &Joiner{
				config:     DefaultConfig(),
				SourceMaps: tt.sourceMaps,
			}

			line, col := j.getLocation(tt.filePath, tt.jsonPath)
			assert.Equal(t, tt.expectedLine, line)
			assert.Equal(t, tt.expectedCol, col)
		})
	}
}

func TestCollisionError_Error_WithLineNumbers(t *testing.T) {
	tests := []struct {
		name           string
		err            *CollisionError
		shouldContain  []string
		shouldNotMatch []string
	}{
		{
			name: "with line numbers",
			err: &CollisionError{
				Section:      "components.schemas",
				Key:          "User",
				FirstFile:    "api1.yaml",
				FirstPath:    "components.schemas.User",
				FirstLine:    42,
				FirstColumn:  5,
				SecondFile:   "api2.yaml",
				SecondPath:   "components.schemas.User",
				SecondLine:   108,
				SecondColumn: 3,
				Strategy:     StrategyFailOnCollision,
			},
			shouldContain: []string{
				"api1.yaml (line 42)",
				"api2.yaml (line 108)",
				"components.schemas",
				"User",
			},
		},
		{
			name: "without line numbers (zeros)",
			err: &CollisionError{
				Section:    "paths",
				Key:        "/users",
				FirstFile:  "base.yaml",
				FirstPath:  "paths./users",
				FirstLine:  0,
				SecondFile: "ext.yaml",
				SecondPath: "paths./users",
				SecondLine: 0,
				Strategy:   StrategyFailOnCollision,
			},
			shouldContain:  []string{"base.yaml at paths./users", "ext.yaml at paths./users"},
			shouldNotMatch: []string{"(line 0)"},
		},
		{
			name: "only first has line number",
			err: &CollisionError{
				Section:    "definitions",
				Key:        "Pet",
				FirstFile:  "pets.yaml",
				FirstPath:  "definitions.Pet",
				FirstLine:  25,
				SecondFile: "animals.yaml",
				SecondPath: "definitions.Pet",
				SecondLine: 0,
				Strategy:   StrategyFailOnCollision,
			},
			shouldContain:  []string{"pets.yaml (line 25)", "animals.yaml at definitions.Pet"},
			shouldNotMatch: []string{"animals.yaml (line"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()

			for _, contain := range tt.shouldContain {
				assert.Contains(t, errStr, contain)
			}

			for _, notMatch := range tt.shouldNotMatch {
				assert.NotContains(t, errStr, notMatch)
			}
		})
	}
}

func TestCollisionError_StructFields(t *testing.T) {
	err := &CollisionError{
		Section:      "components.schemas",
		Key:          "Address",
		FirstFile:    "service-a.yaml",
		FirstPath:    "components.schemas.Address",
		FirstLine:    150,
		FirstColumn:  4,
		SecondFile:   "service-b.yaml",
		SecondPath:   "components.schemas.Address",
		SecondLine:   200,
		SecondColumn: 6,
		Strategy:     StrategyFailOnPaths,
	}

	assert.Equal(t, "components.schemas", err.Section)
	assert.Equal(t, "Address", err.Key)
	assert.Equal(t, "service-a.yaml", err.FirstFile)
	assert.Equal(t, "components.schemas.Address", err.FirstPath)
	assert.Equal(t, 150, err.FirstLine)
	assert.Equal(t, 4, err.FirstColumn)
	assert.Equal(t, "service-b.yaml", err.SecondFile)
	assert.Equal(t, "components.schemas.Address", err.SecondPath)
	assert.Equal(t, 200, err.SecondLine)
	assert.Equal(t, 6, err.SecondColumn)
	assert.Equal(t, StrategyFailOnPaths, err.Strategy)
}

func TestWithCollisionHandler_NilReturnsError(t *testing.T) {
	_, err := JoinWithOptions(
		WithFilePaths("../testdata/join-base-3.0.yaml", "../testdata/join-extension-3.0.yaml"),
		WithCollisionHandler(nil),
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "collision handler cannot be nil")
}

func TestWithCollisionHandlerFor_EmptyTypesReturnsError(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return ContinueWithStrategy(), nil
	}

	_, err := JoinWithOptions(
		WithFilePaths("../testdata/join-base-3.0.yaml", "../testdata/join-extension-3.0.yaml"),
		WithCollisionHandlerFor(handler),
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one collision type must be specified")
}

func TestWithCollisionHandler_SetsHandler(t *testing.T) {
	handlerCalled := false
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		handlerCalled = true
		return ContinueWithStrategy(), nil
	}

	cfg := &joinConfig{
		filePaths:  make([]string, 0),
		parsedDocs: make([]parser.ParseResult, 0),
	}

	opt := WithCollisionHandler(handler)
	err := opt(cfg)

	require.NoError(t, err)
	assert.NotNil(t, cfg.collisionHandler)
	assert.Nil(t, cfg.collisionHandlerTypes) // nil means all types

	// Verify handler is callable
	_, _ = cfg.collisionHandler(CollisionContext{})
	assert.True(t, handlerCalled)
}

func TestWithCollisionHandlerFor_SetsHandlerAndTypes(t *testing.T) {
	handler := func(collision CollisionContext) (CollisionResolution, error) {
		return ContinueWithStrategy(), nil
	}

	cfg := &joinConfig{
		filePaths:  make([]string, 0),
		parsedDocs: make([]parser.ParseResult, 0),
	}

	opt := WithCollisionHandlerFor(handler, CollisionTypeSchema, CollisionTypePath)
	err := opt(cfg)

	require.NoError(t, err)
	assert.NotNil(t, cfg.collisionHandler)
	assert.NotNil(t, cfg.collisionHandlerTypes)
	assert.True(t, cfg.collisionHandlerTypes[CollisionTypeSchema])
	assert.True(t, cfg.collisionHandlerTypes[CollisionTypePath])
	assert.False(t, cfg.collisionHandlerTypes[CollisionTypeWebhook])
}

func TestWithCollisionHandlerFor_NilReturnsError(t *testing.T) {
	cfg := &joinConfig{
		filePaths:  make([]string, 0),
		parsedDocs: make([]parser.ParseResult, 0),
	}

	opt := WithCollisionHandlerFor(nil, CollisionTypeSchema)
	err := opt(cfg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "collision handler cannot be nil")
}

func TestJoinResult_ToParseResult(t *testing.T) {
	t.Run("OAS3 result converts correctly", func(t *testing.T) {
		// Create a JoinResult with OAS3 document
		joinResult := &JoinResult{
			Document:      &parser.OAS3Document{OpenAPI: "3.0.3", Info: &parser.Info{Title: "Test API", Version: "1.0"}},
			Version:       "3.0.3",
			OASVersion:    parser.OASVersion303,
			SourceFormat:  parser.SourceFormatYAML,
			Warnings:      []string{"warning1", "warning2"},
			Stats:         parser.DocumentStats{PathCount: 5, OperationCount: 10},
			firstFilePath: "/path/to/first.yaml",
		}

		parseResult := joinResult.ToParseResult()

		assert.Equal(t, "/path/to/first.yaml", parseResult.SourcePath)
		assert.Equal(t, parser.SourceFormatYAML, parseResult.SourceFormat)
		assert.Equal(t, "3.0.3", parseResult.Version)
		assert.Equal(t, parser.OASVersion303, parseResult.OASVersion)
		assert.NotNil(t, parseResult.Document)
		assert.Empty(t, parseResult.Errors)
		assert.Equal(t, []string{"warning1", "warning2"}, parseResult.Warnings)
		assert.Equal(t, 5, parseResult.Stats.PathCount)
		assert.Equal(t, 10, parseResult.Stats.OperationCount)

		// Verify Document type assertion works
		doc, ok := parseResult.Document.(*parser.OAS3Document)
		assert.True(t, ok)
		assert.Equal(t, "Test API", doc.Info.Title)
	})

	t.Run("OAS2 result converts correctly", func(t *testing.T) {
		joinResult := &JoinResult{
			Document:      &parser.OAS2Document{Swagger: "2.0", Info: &parser.Info{Title: "Swagger API", Version: "1.0"}},
			Version:       "2.0",
			OASVersion:    parser.OASVersion20,
			SourceFormat:  parser.SourceFormatJSON,
			Stats:         parser.DocumentStats{PathCount: 3},
			firstFilePath: "/api/swagger.json",
		}

		parseResult := joinResult.ToParseResult()

		assert.Equal(t, "/api/swagger.json", parseResult.SourcePath)
		assert.Equal(t, parser.SourceFormatJSON, parseResult.SourceFormat)
		assert.Equal(t, "2.0", parseResult.Version)
		assert.Equal(t, parser.OASVersion20, parseResult.OASVersion)

		doc, ok := parseResult.Document.(*parser.OAS2Document)
		assert.True(t, ok)
		assert.Equal(t, "Swagger API", doc.Info.Title)
	})

	t.Run("empty firstFilePath uses default", func(t *testing.T) {
		joinResult := &JoinResult{
			Document:      &parser.OAS3Document{OpenAPI: "3.1.0"},
			Version:       "3.1.0",
			OASVersion:    parser.OASVersion310,
			SourceFormat:  parser.SourceFormatYAML,
			firstFilePath: "", // Empty
		}

		parseResult := joinResult.ToParseResult()

		assert.Equal(t, "joiner", parseResult.SourcePath)
	})

	t.Run("structured warnings are converted", func(t *testing.T) {
		joinResult := &JoinResult{
			Document:     &parser.OAS3Document{OpenAPI: "3.0.0"},
			Version:      "3.0.0",
			OASVersion:   parser.OASVersion300,
			SourceFormat: parser.SourceFormatYAML,
			StructuredWarnings: JoinWarnings{
				{Category: WarnSchemaCollision, Message: "collision warning"},
				{Category: WarnGenericSourceName, Message: "source name warning"},
			},
		}

		parseResult := joinResult.ToParseResult()

		// StructuredWarnings should be converted via WarningStrings()
		assert.Len(t, parseResult.Warnings, 2)
	})

	t.Run("legacy Warnings slice is used when StructuredWarnings is empty", func(t *testing.T) {
		joinResult := &JoinResult{
			Document:           &parser.OAS3Document{OpenAPI: "3.0.0"},
			Version:            "3.0.0",
			OASVersion:         parser.OASVersion300,
			SourceFormat:       parser.SourceFormatYAML,
			Warnings:           []string{"legacy warning 1", "legacy warning 2"},
			StructuredWarnings: nil, // Empty - should fall back to Warnings
		}

		parseResult := joinResult.ToParseResult()

		require.Len(t, parseResult.Warnings, 2)
		assert.Equal(t, "legacy warning 1", parseResult.Warnings[0])
		assert.Equal(t, "legacy warning 2", parseResult.Warnings[1])
	})

	t.Run("Data field is nil and LoadTime/SourceSize are zero", func(t *testing.T) {
		// JoinResult aggregates multiple sources, so individual LoadTime/SourceSize
		// are not meaningful - they should be zero. Data is also not populated.
		joinResult := &JoinResult{
			Document:     &parser.OAS3Document{OpenAPI: "3.0.0"},
			Version:      "3.0.0",
			OASVersion:   parser.OASVersion300,
			SourceFormat: parser.SourceFormatYAML,
		}

		parseResult := joinResult.ToParseResult()

		assert.Nil(t, parseResult.Data)
		assert.Zero(t, parseResult.LoadTime)
		assert.Zero(t, parseResult.SourceSize)
	})
}
