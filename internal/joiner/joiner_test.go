package joiner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/erraggy/oastools/internal/parser"
)

func TestJoinOAS3Documents(t *testing.T) {
	testdataDir := filepath.Join("..", "..", "testdata")

	tests := []struct {
		name           string
		files          []string
		config         JoinerConfig
		expectError    bool
		errorContains  string
		validateResult func(*testing.T, *JoinResult)
	}{
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
				PreserveFirstInfo: true,
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
				PreserveFirstInfo: true,
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
				PreserveFirstInfo: true,
			},
			expectError:   true,
			errorContains: "collision in paths: '/users'",
		},
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
			j := New(tt.config)
			result, err := j.Join(tt.files)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error containing '%s', got nil", tt.errorContains)
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
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
		})
	}
}

func TestJoinOAS2Documents(t *testing.T) {
	testdataDir := filepath.Join("..", "..", "testdata")

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
	testdataDir := filepath.Join("..", "..", "testdata")

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := New(DefaultConfig())
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

func TestJoinToFile(t *testing.T) {
	testdataDir := filepath.Join("..", "..", "testdata")
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "joined.yaml")

	j := New(DefaultConfig())
	err := j.JoinToFile([]string{
		filepath.Join(testdataDir, "join-base-3.0.yaml"),
		filepath.Join(testdataDir, "join-extension-3.0.yaml"),
	}, outputPath)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatal("output file was not created")
	}

	// Verify file can be parsed
	p := parser.New()
	result, err := p.Parse(outputPath)
	if err != nil {
		t.Fatalf("failed to parse output file: %v", err)
	}

	if result.Version != "3.0.3" {
		t.Errorf("expected version 3.0.3, got %s", result.Version)
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

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
