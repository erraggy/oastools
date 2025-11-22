package differ

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

// TestExtensionDiffing tests that extensions (x- fields) are properly detected and reported
func TestExtensionDiffing(t *testing.T) {
	tests := []struct {
		name          string
		source        *parser.OAS3Document
		target        *parser.OAS3Document
		expectedCount int
		checkChanges  func(t *testing.T, changes []Change)
	}{
		{
			name: "Document level extension added",
			source: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Extra:   map[string]any{},
			},
			target: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Extra: map[string]any{
					"x-api-id": "test-123",
				},
			},
			expectedCount: 1,
			checkChanges: func(t *testing.T, changes []Change) {
				if changes[0].Category != CategoryExtension {
					t.Errorf("Expected category extension, got %s", changes[0].Category)
				}
				if changes[0].Type != ChangeTypeAdded {
					t.Errorf("Expected type added, got %s", changes[0].Type)
				}
				if changes[0].Path != "document.x-api-id" {
					t.Errorf("Expected path 'document.x-api-id', got %s", changes[0].Path)
				}
			},
		},
		{
			name: "Document level extension removed",
			source: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Extra: map[string]any{
					"x-api-id": "test-123",
				},
			},
			target: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Extra:   map[string]any{},
			},
			expectedCount: 1,
			checkChanges: func(t *testing.T, changes []Change) {
				if changes[0].Type != ChangeTypeRemoved {
					t.Errorf("Expected type removed, got %s", changes[0].Type)
				}
			},
		},
		{
			name: "Document level extension modified",
			source: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Extra: map[string]any{
					"x-api-id": "test-123",
				},
			},
			target: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Extra: map[string]any{
					"x-api-id": "test-456",
				},
			},
			expectedCount: 1,
			checkChanges: func(t *testing.T, changes []Change) {
				if changes[0].Type != ChangeTypeModified {
					t.Errorf("Expected type modified, got %s", changes[0].Type)
				}
				if changes[0].OldValue != "test-123" {
					t.Errorf("Expected old value 'test-123', got %v", changes[0].OldValue)
				}
				if changes[0].NewValue != "test-456" {
					t.Errorf("Expected new value 'test-456', got %v", changes[0].NewValue)
				}
			},
		},
		{
			name: "Multiple extensions changed",
			source: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Extra: map[string]any{
					"x-api-id":   "test-123",
					"x-team":     "platform",
					"x-audience": "internal",
				},
			},
			target: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Extra: map[string]any{
					"x-api-id":      "test-456", // Modified
					"x-team":        "platform", // Unchanged
					"x-environment": "prod",     // Added
					// x-audience removed
				},
			},
			expectedCount: 3, // 1 modified, 1 added, 1 removed
		},
		{
			name: "PathItem level extension",
			source: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Paths: parser.Paths{
					"/pets": &parser.PathItem{
						Get: &parser.Operation{
							Responses: &parser.Responses{
								Codes: map[string]*parser.Response{
									"200": {Description: "OK"},
								},
							},
						},
						Extra: map[string]any{
							"x-rate-limit": 100,
						},
					},
				},
			},
			target: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Paths: parser.Paths{
					"/pets": &parser.PathItem{
						Get: &parser.Operation{
							Responses: &parser.Responses{
								Codes: map[string]*parser.Response{
									"200": {Description: "OK"},
								},
							},
						},
						Extra: map[string]any{
							"x-rate-limit": 200,
						},
					},
				},
			},
			expectedCount: 1,
			checkChanges: func(t *testing.T, changes []Change) {
				if changes[0].Path != "document.paths./pets.x-rate-limit" {
					t.Errorf("Expected path 'document.paths./pets.x-rate-limit', got %s", changes[0].Path)
				}
			},
		},
		{
			name: "Operation level extension",
			source: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Paths: parser.Paths{
					"/pets": &parser.PathItem{
						Get: &parser.Operation{
							Responses: &parser.Responses{
								Codes: map[string]*parser.Response{
									"200": {Description: "OK"},
								},
							},
							Extra: map[string]any{
								"x-code-samples": []string{"sample1"},
							},
						},
					},
				},
			},
			target: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Paths: parser.Paths{
					"/pets": &parser.PathItem{
						Get: &parser.Operation{
							Responses: &parser.Responses{
								Codes: map[string]*parser.Response{
									"200": {Description: "OK"},
								},
							},
							Extra: map[string]any{}, // Extension removed
						},
					},
				},
			},
			expectedCount: 1,
			checkChanges: func(t *testing.T, changes []Change) {
				if changes[0].Path != "document.paths./pets.get.x-code-samples" {
					t.Errorf("Expected path 'document.paths./pets.get.x-code-samples', got %s", changes[0].Path)
				}
			},
		},
		{
			name: "Components level extension",
			source: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"Pet": {Type: "object"},
					},
					Extra: map[string]any{
						"x-schema-registry": "v1",
					},
				},
			},
			target: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"Pet": {Type: "object"},
					},
					Extra: map[string]any{
						"x-schema-registry": "v2",
					},
				},
			},
			expectedCount: 1,
			checkChanges: func(t *testing.T, changes []Change) {
				if changes[0].Path != "document.components.x-schema-registry" {
					t.Errorf("Expected path 'document.components.x-schema-registry', got %s", changes[0].Path)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = ModeSimple

			sourceResult := parser.ParseResult{
				Document:   tt.source,
				Version:    tt.source.OpenAPI,
				OASVersion: parser.OASVersion300,
			}
			targetResult := parser.ParseResult{
				Document:   tt.target,
				Version:    tt.target.OpenAPI,
				OASVersion: parser.OASVersion300,
			}

			result, err := d.DiffParsed(sourceResult, targetResult)
			if err != nil {
				t.Fatalf("DiffParsed failed: %v", err)
			}

			if len(result.Changes) != tt.expectedCount {
				t.Errorf("Expected %d changes, got %d", tt.expectedCount, len(result.Changes))
				for i, change := range result.Changes {
					t.Logf("Change %d: %s", i, change.String())
				}
			}

			if tt.checkChanges != nil && len(result.Changes) > 0 {
				tt.checkChanges(t, result.Changes)
			}
		})
	}
}

// TestExtensionBreakingDiffing tests extension diffing with breaking change mode
func TestExtensionBreakingDiffing(t *testing.T) {
	source := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Extra: map[string]any{
			"x-api-id": "test-123",
		},
	}

	target := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Extra: map[string]any{
			"x-api-id": "test-456",
		},
	}

	d := New()
	d.Mode = ModeBreaking

	sourceResult := parser.ParseResult{
		Document:   source,
		Version:    source.OpenAPI,
		OASVersion: parser.OASVersion300,
	}
	targetResult := parser.ParseResult{
		Document:   target,
		Version:    target.OpenAPI,
		OASVersion: parser.OASVersion300,
	}

	result, err := d.DiffParsed(sourceResult, targetResult)
	if err != nil {
		t.Fatalf("DiffParsed failed: %v", err)
	}

	if len(result.Changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(result.Changes))
	}

	// Extension changes should be INFO severity (non-breaking)
	if result.Changes[0].Severity != SeverityInfo {
		t.Errorf("Expected severity Info, got %v", result.Changes[0].Severity)
	}

	if result.HasBreakingChanges {
		t.Error("Extension changes should not be classified as breaking")
	}
}

// TestOAS2ExtensionDiffing tests extension diffing with OAS 2.0 documents
func TestOAS2ExtensionDiffing(t *testing.T) {
	source := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Extra: map[string]any{
			"x-api-id": "test-123",
		},
	}

	target := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Extra: map[string]any{
			"x-api-id":   "test-123",
			"x-audience": "public", // Added
		},
	}

	d := New()

	sourceResult := parser.ParseResult{
		Document:   source,
		Version:    source.Swagger,
		OASVersion: parser.OASVersion20,
	}
	targetResult := parser.ParseResult{
		Document:   target,
		Version:    target.Swagger,
		OASVersion: parser.OASVersion20,
	}

	result, err := d.DiffParsed(sourceResult, targetResult)
	if err != nil {
		t.Fatalf("DiffParsed failed: %v", err)
	}

	if len(result.Changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(result.Changes))
	}

	if result.Changes[0].Type != ChangeTypeAdded {
		t.Errorf("Expected type added, got %s", result.Changes[0].Type)
	}

	if result.Changes[0].Category != CategoryExtension {
		t.Errorf("Expected category extension, got %s", result.Changes[0].Category)
	}
}

// TestComplexExtensionValues tests extension diffing with complex values
func TestComplexExtensionValues(t *testing.T) {
	source := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Extra: map[string]any{
			"x-metadata": map[string]any{
				"team": "platform",
				"tier": "gold",
			},
		},
	}

	target := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Extra: map[string]any{
			"x-metadata": map[string]any{
				"team": "platform",
				"tier": "platinum", // Changed nested value
			},
		},
	}

	d := New()

	sourceResult := parser.ParseResult{
		Document:   source,
		Version:    source.OpenAPI,
		OASVersion: parser.OASVersion300,
	}
	targetResult := parser.ParseResult{
		Document:   target,
		Version:    target.OpenAPI,
		OASVersion: parser.OASVersion300,
	}

	result, err := d.DiffParsed(sourceResult, targetResult)
	if err != nil {
		t.Fatalf("DiffParsed failed: %v", err)
	}

	// Should detect the complex value changed
	if len(result.Changes) != 1 {
		t.Errorf("Expected 1 change, got %d", len(result.Changes))
	}

	if result.Changes[0].Type != ChangeTypeModified {
		t.Errorf("Expected type modified, got %s", result.Changes[0].Type)
	}
}
