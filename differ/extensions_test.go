package differ

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				assert.Equal(t, CategoryExtension, changes[0].Category)
				assert.Equal(t, ChangeTypeAdded, changes[0].Type)
				assert.Equal(t, "document.x-api-id", changes[0].Path)
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
				assert.Equal(t, ChangeTypeRemoved, changes[0].Type)
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
				assert.Equal(t, ChangeTypeModified, changes[0].Type)
				assert.Equal(t, "test-123", changes[0].OldValue)
				assert.Equal(t, "test-456", changes[0].NewValue)
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
				assert.Equal(t, "document.paths./pets.x-rate-limit", changes[0].Path)
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
				assert.Equal(t, "document.paths./pets.get.x-code-samples", changes[0].Path)
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
				assert.Equal(t, "document.components.x-schema-registry", changes[0].Path)
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
			require.NoError(t, err)

			assert.Equal(t, tt.expectedCount, len(result.Changes))
			if len(result.Changes) != tt.expectedCount {
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
	require.NoError(t, err)

	require.Len(t, result.Changes, 1)

	// Extension changes should be INFO severity (non-breaking)
	assert.Equal(t, SeverityInfo, result.Changes[0].Severity)

	assert.False(t, result.HasBreakingChanges, "Extension changes should not be classified as breaking")
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
	require.NoError(t, err)

	require.Len(t, result.Changes, 1)

	assert.Equal(t, ChangeTypeAdded, result.Changes[0].Type)

	assert.Equal(t, CategoryExtension, result.Changes[0].Category)
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
	require.NoError(t, err)

	// Should detect the complex value changed
	require.Len(t, result.Changes, 1)

	assert.Equal(t, ChangeTypeModified, result.Changes[0].Type)
}

// TestNewExtensionLocations tests extension diffing at newly added locations
func TestNewExtensionLocations(t *testing.T) {
	tests := []struct {
		name          string
		source        *parser.OAS3Document
		target        *parser.OAS3Document
		expectedCount int
		expectedPath  string
	}{
		{
			name: "Info level extension",
			source: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info: &parser.Info{
					Title:   "Test API",
					Version: "1.0.0",
					Extra: map[string]any{
						"x-logo": "logo.png",
					},
				},
			},
			target: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info: &parser.Info{
					Title:   "Test API",
					Version: "1.0.0",
					Extra: map[string]any{
						"x-logo": "new-logo.png",
					},
				},
			},
			expectedCount: 1,
			expectedPath:  "document.info.x-logo",
		},
		{
			name: "Server level extension",
			source: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Servers: []*parser.Server{
					{
						URL: "https://api.example.com",
						Extra: map[string]any{
							"x-region": "us-east-1",
						},
					},
				},
			},
			target: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Servers: []*parser.Server{
					{
						URL: "https://api.example.com",
						Extra: map[string]any{
							"x-region": "us-west-2",
						},
					},
				},
			},
			expectedCount: 1,
			expectedPath:  "document.servers[https://api.example.com].x-region",
		},
		{
			name: "Parameter level extension",
			source: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Paths: parser.Paths{
					"/pets": &parser.PathItem{
						Get: &parser.Operation{
							Parameters: []*parser.Parameter{
								{
									Name: "limit",
									In:   "query",
									Extra: map[string]any{
										"x-example-values": []int{10, 20, 50},
									},
								},
							},
							Responses: &parser.Responses{
								Codes: map[string]*parser.Response{
									"200": {Description: "OK"},
								},
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
							Parameters: []*parser.Parameter{
								{
									Name: "limit",
									In:   "query",
									Extra: map[string]any{
										"x-example-values": []int{10, 20, 50, 100},
									},
								},
							},
							Responses: &parser.Responses{
								Codes: map[string]*parser.Response{
									"200": {Description: "OK"},
								},
							},
						},
					},
				},
			},
			expectedCount: 1,
			expectedPath:  "document.paths./pets.get.parameters[limit:query].x-example-values",
		},
		{
			name: "RequestBody level extension",
			source: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Paths: parser.Paths{
					"/pets": &parser.PathItem{
						Post: &parser.Operation{
							RequestBody: &parser.RequestBody{
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Type: "object"},
									},
								},
								Extra: map[string]any{
									"x-body-examples": "see-docs",
								},
							},
							Responses: &parser.Responses{
								Codes: map[string]*parser.Response{
									"201": {Description: "Created"},
								},
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
						Post: &parser.Operation{
							RequestBody: &parser.RequestBody{
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Type: "object"},
									},
								},
								Extra: map[string]any{}, // Removed
							},
							Responses: &parser.Responses{
								Codes: map[string]*parser.Response{
									"201": {Description: "Created"},
								},
							},
						},
					},
				},
			},
			expectedCount: 1,
			expectedPath:  "document.paths./pets.post.requestBody.x-body-examples",
		},
		{
			name: "Response level extension",
			source: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Paths: parser.Paths{
					"/pets": &parser.PathItem{
						Get: &parser.Operation{
							Responses: &parser.Responses{
								Codes: map[string]*parser.Response{
									"200": {
										Description: "OK",
										Extra: map[string]any{
											"x-cache-ttl": 3600,
										},
									},
								},
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
									"200": {
										Description: "OK",
										Extra: map[string]any{
											"x-cache-ttl": 7200,
										},
									},
								},
							},
						},
					},
				},
			},
			expectedCount: 1,
			expectedPath:  "document.paths./pets.get.responses[200].x-cache-ttl",
		},
		{
			name: "Schema level extension",
			source: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"Pet": {
							Type: "object",
							Extra: map[string]any{
								"x-table-name": "pets",
							},
						},
					},
				},
			},
			target: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"Pet": {
							Type: "object",
							Extra: map[string]any{
								"x-table-name": "pet_records",
							},
						},
					},
				},
			},
			expectedCount: 1,
			expectedPath:  "document.components.schemas.Pet.x-table-name",
		},
		{
			name: "SecurityScheme level extension",
			source: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					SecuritySchemes: map[string]*parser.SecurityScheme{
						"api_key": {
							Type: "apiKey",
							Name: "X-API-Key",
							In:   "header",
							Extra: map[string]any{
								"x-key-source": "env-var",
							},
						},
					},
				},
			},
			target: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					SecuritySchemes: map[string]*parser.SecurityScheme{
						"api_key": {
							Type: "apiKey",
							Name: "X-API-Key",
							In:   "header",
							Extra: map[string]any{
								"x-key-source": "vault",
							},
						},
					},
				},
			},
			expectedCount: 1,
			expectedPath:  "document.components.securitySchemes.api_key.x-key-source",
		},
		{
			name: "Tag level extension",
			source: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Tags: []*parser.Tag{
					{
						Name: "pets",
						Extra: map[string]any{
							"x-display-order": 1,
						},
					},
				},
			},
			target: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Tags: []*parser.Tag{
					{
						Name: "pets",
						Extra: map[string]any{
							"x-display-order": 2,
						},
					},
				},
			},
			expectedCount: 1,
			expectedPath:  "document.tags[pets].x-display-order",
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
			require.NoError(t, err)

			assert.Equal(t, tt.expectedCount, len(result.Changes))
			if len(result.Changes) != tt.expectedCount {
				for i, change := range result.Changes {
					t.Logf("Change %d: %s at %s", i, change.Type, change.Path)
				}
			}

			if len(result.Changes) > 0 {
				assert.Equal(t, tt.expectedPath, result.Changes[0].Path)
				assert.Equal(t, CategoryExtension, result.Changes[0].Category)
			}
		})
	}
}

// TestNewExtensionLocationsBreaking tests extension diffing in breaking mode for new locations
func TestNewExtensionLocationsBreaking(t *testing.T) {
	// All these should be SeverityInfo (non-breaking)
	tests := []struct {
		name   string
		source *parser.OAS3Document
		target *parser.OAS3Document
	}{
		{
			name: "Info extension change",
			source: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info: &parser.Info{
					Title:   "Test",
					Version: "1.0.0",
					Extra:   map[string]any{"x-logo": "old.png"},
				},
			},
			target: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info: &parser.Info{
					Title:   "Test",
					Version: "1.0.0",
					Extra:   map[string]any{"x-logo": "new.png"},
				},
			},
		},
		{
			name: "Server extension change",
			source: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Servers: []*parser.Server{
					{URL: "https://api.example.com", Extra: map[string]any{"x-region": "us-east"}},
				},
			},
			target: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Servers: []*parser.Server{
					{URL: "https://api.example.com", Extra: map[string]any{"x-region": "us-west"}},
				},
			},
		},
		{
			name: "Schema extension change",
			source: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"Pet": {Type: "object", Extra: map[string]any{"x-db-table": "pets"}},
					},
				},
			},
			target: &parser.OAS3Document{
				OpenAPI: "3.0.3",
				Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"Pet": {Type: "object", Extra: map[string]any{"x-db-table": "pet_records"}},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.Mode = ModeBreaking

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
			require.NoError(t, err)

			require.NotEmpty(t, result.Changes, "Expected at least one change")

			// All extension changes should be SeverityInfo
			for _, change := range result.Changes {
				if change.Category == CategoryExtension {
					assert.Equal(t, SeverityInfo, change.Severity, "Extension change should be SeverityInfo")
				}
			}

			// Extension changes should not be breaking
			assert.False(t, result.HasBreakingChanges, "Extension changes should not be classified as breaking")
		})
	}
}
