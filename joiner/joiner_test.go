package joiner

import (
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

// ========================================
// Tests for package-level convenience functions
// ========================================

// TestJoinConvenience tests the package-level Join convenience function
func TestJoinConvenience(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	tests := []struct {
		name           string
		files          []string
		config         JoinerConfig
		expectError    bool
		errorContains  string
		validateResult func(*testing.T, *JoinResult)
	}{
		{
			name: "join two OAS 3.0 files successfully",
			files: []string{
				filepath.Join(testdataDir, "join-base-3.0.yaml"),
				filepath.Join(testdataDir, "join-extension-3.0.yaml"),
			},
			config:      DefaultConfig(),
			expectError: false,
			validateResult: func(t *testing.T, result *JoinResult) {
				assert.NotNil(t, result)
				assert.Equal(t, "3.0.3", result.Version)
				doc, ok := result.Document.(*parser.OAS3Document)
				assert.True(t, ok)
				assert.GreaterOrEqual(t, len(doc.Paths), 2)
			},
		},
		{
			name: "join three OAS 3.0 files",
			files: []string{
				filepath.Join(testdataDir, "join-base-3.0.yaml"),
				filepath.Join(testdataDir, "join-extension-3.0.yaml"),
				filepath.Join(testdataDir, "join-additional-3.0.yaml"),
			},
			config:      DefaultConfig(),
			expectError: false,
			validateResult: func(t *testing.T, result *JoinResult) {
				assert.NotNil(t, result)
				doc, ok := result.Document.(*parser.OAS3Document)
				assert.True(t, ok)
				assert.GreaterOrEqual(t, len(doc.Paths), 4)
			},
		},
		{
			name: "join two OAS 2.0 files",
			files: []string{
				filepath.Join(testdataDir, "join-base-2.0.yaml"),
				filepath.Join(testdataDir, "join-extension-2.0.yaml"),
			},
			config:      DefaultConfig(),
			expectError: false,
			validateResult: func(t *testing.T, result *JoinResult) {
				assert.NotNil(t, result)
				assert.Equal(t, "2.0", result.Version)
				doc, ok := result.Document.(*parser.OAS2Document)
				assert.True(t, ok)
				assert.GreaterOrEqual(t, len(doc.Paths), 1)
			},
		},
		{
			name: "join with accept-left strategy",
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
				assert.NotNil(t, result)
				assert.Greater(t, result.CollisionCount, 0)
			},
		},
		{
			name: "join with fail on collision - should error",
			files: []string{
				filepath.Join(testdataDir, "join-base-3.0.yaml"),
				filepath.Join(testdataDir, "join-collision-3.0.yaml"),
			},
			config:        DefaultConfig(),
			expectError:   true,
			errorContains: "collision",
		},
		{
			name: "join insufficient files - should error",
			files: []string{
				filepath.Join(testdataDir, "join-base-3.0.yaml"),
			},
			config:        DefaultConfig(),
			expectError:   true,
			errorContains: "at least 2 specification files are required",
		},
		{
			name: "join incompatible versions - should error",
			files: []string{
				filepath.Join(testdataDir, "join-base-2.0.yaml"),
				filepath.Join(testdataDir, "join-base-3.0.yaml"),
			},
			config:        DefaultConfig(),
			expectError:   true,
			errorContains: "incompatible versions",
		},
		{
			name: "join nonexistent file - should error",
			files: []string{
				filepath.Join(testdataDir, "join-base-3.0.yaml"),
				"nonexistent-file.yaml",
			},
			config:      DefaultConfig(),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Join(tt.files, tt.config)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tt.validateResult != nil {
					tt.validateResult(t, result)
				}
			}
		})
	}
}

// TestJoinParsedConvenience tests the package-level JoinParsed convenience function
func TestJoinParsedConvenience(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	tests := []struct {
		name           string
		setupDocs      func(*testing.T) []parser.ParseResult
		config         JoinerConfig
		expectError    bool
		errorContains  string
		validateResult func(*testing.T, *JoinResult)
	}{
		{
			name: "join two parsed OAS 3.0 documents",
			setupDocs: func(t *testing.T) []parser.ParseResult {
				doc1, err := parser.ParseWithOptions(
					parser.WithFilePath(filepath.Join(testdataDir, "join-base-3.0.yaml")),
					parser.WithValidateStructure(true),
				)
				require.NoError(t, err)
				doc2, err := parser.ParseWithOptions(
					parser.WithFilePath(filepath.Join(testdataDir, "join-extension-3.0.yaml")),
					parser.WithValidateStructure(true),
				)
				require.NoError(t, err)
				return []parser.ParseResult{*doc1, *doc2}
			},
			config:      DefaultConfig(),
			expectError: false,
			validateResult: func(t *testing.T, result *JoinResult) {
				assert.NotNil(t, result)
				assert.Equal(t, "3.0.3", result.Version)
				doc, ok := result.Document.(*parser.OAS3Document)
				assert.True(t, ok)
				assert.GreaterOrEqual(t, len(doc.Paths), 2)
			},
		},
		{
			name: "join three parsed documents",
			setupDocs: func(t *testing.T) []parser.ParseResult {
				doc1, err := parser.ParseWithOptions(
					parser.WithFilePath(filepath.Join(testdataDir, "join-base-3.0.yaml")),
					parser.WithValidateStructure(true),
				)
				require.NoError(t, err)
				doc2, err := parser.ParseWithOptions(
					parser.WithFilePath(filepath.Join(testdataDir, "join-extension-3.0.yaml")),
					parser.WithValidateStructure(true),
				)
				require.NoError(t, err)
				doc3, err := parser.ParseWithOptions(
					parser.WithFilePath(filepath.Join(testdataDir, "join-additional-3.0.yaml")),
					parser.WithValidateStructure(true),
				)
				require.NoError(t, err)
				return []parser.ParseResult{*doc1, *doc2, *doc3}
			},
			config:      DefaultConfig(),
			expectError: false,
			validateResult: func(t *testing.T, result *JoinResult) {
				assert.NotNil(t, result)
				doc, ok := result.Document.(*parser.OAS3Document)
				assert.True(t, ok)
				assert.GreaterOrEqual(t, len(doc.Paths), 4)
			},
		},
		{
			name: "join parsed OAS 2.0 documents",
			setupDocs: func(t *testing.T) []parser.ParseResult {
				doc1, err := parser.ParseWithOptions(
					parser.WithFilePath(filepath.Join(testdataDir, "join-base-2.0.yaml")),
					parser.WithValidateStructure(true),
				)
				require.NoError(t, err)
				doc2, err := parser.ParseWithOptions(
					parser.WithFilePath(filepath.Join(testdataDir, "join-extension-2.0.yaml")),
					parser.WithValidateStructure(true),
				)
				require.NoError(t, err)
				return []parser.ParseResult{*doc1, *doc2}
			},
			config:      DefaultConfig(),
			expectError: false,
			validateResult: func(t *testing.T, result *JoinResult) {
				assert.NotNil(t, result)
				assert.Equal(t, "2.0", result.Version)
				doc, ok := result.Document.(*parser.OAS2Document)
				assert.True(t, ok)
				assert.GreaterOrEqual(t, len(doc.Paths), 1)
			},
		},
		{
			name: "join with collision resolution",
			setupDocs: func(t *testing.T) []parser.ParseResult {
				doc1, err := parser.ParseWithOptions(
					parser.WithFilePath(filepath.Join(testdataDir, "join-base-3.0.yaml")),
					parser.WithValidateStructure(true),
				)
				require.NoError(t, err)
				doc2, err := parser.ParseWithOptions(
					parser.WithFilePath(filepath.Join(testdataDir, "join-collision-3.0.yaml")),
					parser.WithValidateStructure(true),
				)
				require.NoError(t, err)
				return []parser.ParseResult{*doc1, *doc2}
			},
			config: JoinerConfig{
				PathStrategy:      StrategyAcceptRight,
				SchemaStrategy:    StrategyAcceptRight,
				ComponentStrategy: StrategyAcceptRight,
				DeduplicateTags:   true,
				MergeArrays:       true,
			},
			expectError: false,
			validateResult: func(t *testing.T, result *JoinResult) {
				assert.NotNil(t, result)
				assert.Greater(t, result.CollisionCount, 0)
			},
		},
		{
			name: "join documents from ParseBytes",
			setupDocs: func(t *testing.T) []parser.ParseResult {
				data1 := []byte(`openapi: "3.0.0"
info:
  title: API 1
  version: 1.0.0
paths:
  /api1:
    get:
      responses:
        '200':
          description: Success
`)
				data2 := []byte(`openapi: "3.0.0"
info:
  title: API 2
  version: 1.0.0
paths:
  /api2:
    get:
      responses:
        '200':
          description: Success
`)
				doc1, err := parser.ParseWithOptions(
					parser.WithBytes(data1),
					parser.WithValidateStructure(true),
				)
				require.NoError(t, err)
				doc2, err := parser.ParseWithOptions(
					parser.WithBytes(data2),
					parser.WithValidateStructure(true),
				)
				require.NoError(t, err)
				return []parser.ParseResult{*doc1, *doc2}
			},
			config:      DefaultConfig(),
			expectError: false,
			validateResult: func(t *testing.T, result *JoinResult) {
				assert.NotNil(t, result)
				doc, ok := result.Document.(*parser.OAS3Document)
				assert.True(t, ok)
				assert.Equal(t, 2, len(doc.Paths))
			},
		},
		{
			name: "join insufficient documents - should error",
			setupDocs: func(t *testing.T) []parser.ParseResult {
				doc1, err := parser.ParseWithOptions(
					parser.WithFilePath(filepath.Join(testdataDir, "join-base-3.0.yaml")),
					parser.WithValidateStructure(true),
				)
				require.NoError(t, err)
				return []parser.ParseResult{*doc1}
			},
			config:        DefaultConfig(),
			expectError:   true,
			errorContains: "at least 2 specification documents are required",
		},
		{
			name: "join documents with parse errors - should error",
			setupDocs: func(t *testing.T) []parser.ParseResult {
				doc1, err := parser.ParseWithOptions(
					parser.WithFilePath(filepath.Join(testdataDir, "join-base-3.0.yaml")),
					parser.WithValidateStructure(true),
				)
				require.NoError(t, err)
				// Create a document with errors
				doc2, err := parser.ParseWithOptions(
					parser.WithFilePath(filepath.Join(testdataDir, "invalid-oas3.yaml")),
					parser.WithValidateStructure(true),
				)
				require.NoError(t, err)
				return []parser.ParseResult{*doc1, *doc2}
			},
			config:        DefaultConfig(),
			expectError:   true,
			errorContains: "Errors is not empty",
		},
		{
			name: "join incompatible versions - should error",
			setupDocs: func(t *testing.T) []parser.ParseResult {
				doc1, err := parser.ParseWithOptions(
					parser.WithFilePath(filepath.Join(testdataDir, "join-base-2.0.yaml")),
					parser.WithValidateStructure(true),
				)
				require.NoError(t, err)
				doc2, err := parser.ParseWithOptions(
					parser.WithFilePath(filepath.Join(testdataDir, "join-base-3.0.yaml")),
					parser.WithValidateStructure(true),
				)
				require.NoError(t, err)
				return []parser.ParseResult{*doc1, *doc2}
			},
			config:        DefaultConfig(),
			expectError:   true,
			errorContains: "incompatible versions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docs := tt.setupDocs(t)
			result, err := JoinParsed(docs, tt.config)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tt.validateResult != nil {
					tt.validateResult(t, result)
				}
			}
		})
	}
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

// TestJoinWithOptions_BackwardCompatibility tests that new API produces same results as old API
func TestJoinWithOptions_BackwardCompatibility(t *testing.T) {
	paths := []string{"../testdata/join-base-3.0.yaml", "../testdata/join-extension-3.0.yaml"}

	// Old API
	oldConfig := DefaultConfig()
	oldConfig.PathStrategy = StrategyAcceptLeft
	oldResult, err := Join(paths, oldConfig)
	require.NoError(t, err)

	// New API
	newResult, err := JoinWithOptions(
		WithFilePaths(paths...),
		WithPathStrategy(StrategyAcceptLeft),
	)
	require.NoError(t, err)

	// Compare results
	assert.Equal(t, oldResult.Version, newResult.Version)
	assert.Equal(t, oldResult.SourceFormat, newResult.SourceFormat)
	assert.Equal(t, oldResult.CollisionCount, newResult.CollisionCount)
}
