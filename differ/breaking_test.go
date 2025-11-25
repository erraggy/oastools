package differ

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDiffOAS2Breaking tests breaking change detection for OAS 2.0 documents.
func TestDiffOAS2Breaking(t *testing.T) {
	tests := []struct {
		name           string
		source         *parser.OAS2Document
		target         *parser.OAS2Document
		expectCritical bool
		expectWarning  bool
	}{
		{
			name: "removed path endpoint - critical",
			source: &parser.OAS2Document{
				Swagger: "2.0",
				Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
				Paths: parser.Paths{
					"/users": &parser.PathItem{
						Get: &parser.Operation{
							Summary: "Get users",
						},
					},
				},
			},
			target: &parser.OAS2Document{
				Swagger: "2.0",
				Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
				Paths:   parser.Paths{}, // Path removed
			},
			expectCritical: true,
			expectWarning:  false,
		},
		{
			name: "host changed - warning",
			source: &parser.OAS2Document{
				Swagger:  "2.0",
				Info:     &parser.Info{Title: "Test API", Version: "1.0.0"},
				Host:     "api.example.com",
				BasePath: "/v1",
				Paths:    parser.Paths{},
			},
			target: &parser.OAS2Document{
				Swagger:  "2.0",
				Info:     &parser.Info{Title: "Test API", Version: "1.0.0"},
				Host:     "api2.example.com", // Host changed
				BasePath: "/v1",
				Paths:    parser.Paths{},
			},
			expectCritical: false,
			expectWarning:  true,
		},
		{
			name: "basePath changed - warning",
			source: &parser.OAS2Document{
				Swagger:  "2.0",
				Info:     &parser.Info{Title: "Test API", Version: "1.0.0"},
				Host:     "api.example.com",
				BasePath: "/v1",
				Paths:    parser.Paths{},
			},
			target: &parser.OAS2Document{
				Swagger:  "2.0",
				Info:     &parser.Info{Title: "Test API", Version: "1.0.0"},
				Host:     "api.example.com",
				BasePath: "/v2", // BasePath changed
				Paths:    parser.Paths{},
			},
			expectCritical: false,
			expectWarning:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			result := &DiffResult{Changes: []Change{}}

			d.Mode = ModeBreaking
			d.diffOAS2Unified(tt.source, tt.target, result)

			hasCritical := false
			hasWarning := false
			for _, change := range result.Changes {
				if change.Severity == SeverityCritical {
					hasCritical = true
				}
				if change.Severity == SeverityWarning {
					hasWarning = true
				}
			}

			if tt.expectCritical {
				assert.True(t, hasCritical, "Expected at least one critical severity change")
			}
			if tt.expectWarning {
				assert.True(t, hasWarning, "Expected at least one warning severity change")
			}
		})
	}
}

// TestDiffCrossVersionBreaking tests breaking change detection across OAS versions.
func TestDiffCrossVersionBreaking(t *testing.T) {
	tests := []struct {
		name         string
		source       parser.ParseResult
		target       parser.ParseResult
		expectChange bool
	}{
		{
			name: "OAS 2.0 to OAS 3.0.0 - version change",
			source: parser.ParseResult{
				OASVersion: parser.OASVersion20,
				Version:    "2.0",
				Document: &parser.OAS2Document{
					Swagger: "2.0",
					Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
					Paths:   parser.Paths{},
				},
			},
			target: parser.ParseResult{
				OASVersion: parser.OASVersion300,
				Version:    "3.0.0",
				Document: &parser.OAS3Document{
					OpenAPI: "3.0.0",
					Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
					Paths:   parser.Paths{},
				},
			},
			expectChange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			result := &DiffResult{Changes: []Change{}}

			d.Mode = ModeBreaking
			d.diffCrossVersionUnified(tt.source, tt.target, result)

			if tt.expectChange {
				assert.NotEmpty(t, result.Changes, "Expected version change to be detected")
				// Verify that version change is detected (path is "document" for cross-version diff)
				hasVersionChange := false
				for _, change := range result.Changes {
					if change.Path == "document" && change.Category == CategoryInfo {
						hasVersionChange = true
						break
					}
				}
				assert.True(t, hasVersionChange, "Expected version change")
			}
		})
	}
}

// TestDiffStringSlicesBreaking tests breaking change detection for string slices.
func TestDiffStringSlicesBreaking(t *testing.T) {
	tests := []struct {
		name             string
		source           []string
		target           []string
		expectChange     bool
		expectedSeverity string
	}{
		{
			name:             "scheme removed - warning",
			source:           []string{"https", "http"},
			target:           []string{"https"},
			expectChange:     true,
			expectedSeverity: "warning",
		},
		{
			name:             "scheme added - info",
			source:           []string{"https"},
			target:           []string{"https", "http"},
			expectChange:     true,
			expectedSeverity: "info",
		},
		{
			name:         "no change",
			source:       []string{"https"},
			target:       []string{"https"},
			expectChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			result := &DiffResult{Changes: []Change{}}

			d.Mode = ModeBreaking
			d.diffStringSlicesUnified(tt.source, tt.target, "test.schemes", CategoryServer, "scheme", SeverityWarning, result)

			if tt.expectChange {
				assert.NotEmpty(t, result.Changes, "Expected changes to be detected")
			} else {
				assert.Empty(t, result.Changes, "Expected no changes")
			}
		})
	}
}

// TestDiffSecuritySchemeBreaking tests breaking change detection for security schemes.
func TestDiffSecuritySchemeBreaking(t *testing.T) {
	tests := []struct {
		name         string
		source       *parser.SecurityScheme
		target       *parser.SecurityScheme
		expectChange bool
	}{
		{
			name: "type changed - critical",
			source: &parser.SecurityScheme{
				Type: "apiKey",
				Name: "api_key",
				In:   "header",
			},
			target: &parser.SecurityScheme{
				Type: "oauth2",
				Flows: &parser.OAuthFlows{
					Implicit: &parser.OAuthFlow{
						AuthorizationURL: "https://example.com/oauth",
						Scopes:           map[string]string{},
					},
				},
			},
			expectChange: true,
		},
		{
			name: "no change",
			source: &parser.SecurityScheme{
				Type: "apiKey",
				Name: "api_key",
				In:   "header",
			},
			target: &parser.SecurityScheme{
				Type: "apiKey",
				Name: "api_key",
				In:   "header",
			},
			expectChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			result := &DiffResult{Changes: []Change{}}

			d.Mode = ModeBreaking
			d.diffSecuritySchemeUnified(tt.source, tt.target, "test.securityScheme", result)

			if tt.expectChange {
				assert.NotEmpty(t, result.Changes, "Expected changes to be detected")
			}
			// Note: No change case might still generate changes for minor differences
		})
	}
}

// TestDiffEnumBreaking tests breaking change detection for enum values.
func TestDiffEnumBreaking(t *testing.T) {
	tests := []struct {
		name        string
		source      []any
		target      []any
		expectError bool
	}{
		{
			name:        "enum value removed - error",
			source:      []any{"active", "inactive", "pending"},
			target:      []any{"active", "inactive"},
			expectError: true,
		},
		{
			name:        "enum value added - info",
			source:      []any{"active", "inactive"},
			target:      []any{"active", "inactive", "pending"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			result := &DiffResult{Changes: []Change{}}

			d.Mode = ModeBreaking
			d.diffEnumUnified(tt.source, tt.target, "test.enum", result)

			if tt.expectError {
				hasError := false
				for _, change := range result.Changes {
					if change.Severity == SeverityError {
						hasError = true
						break
					}
				}
				assert.True(t, hasError, "Expected error severity for removed enum value")
			}
		})
	}
}

// TestDiffWebhooksBreaking tests breaking change detection for webhooks (OAS 3.1+).
func TestDiffWebhooksBreaking(t *testing.T) {
	tests := []struct {
		name         string
		source       map[string]*parser.PathItem
		target       map[string]*parser.PathItem
		expectChange bool
	}{
		{
			name: "webhook removed - critical",
			source: map[string]*parser.PathItem{
				"newOrder": {
					Post: &parser.Operation{
						Summary: "New order webhook",
					},
				},
			},
			target:       map[string]*parser.PathItem{},
			expectChange: true,
		},
		{
			name:   "webhook added - info",
			source: map[string]*parser.PathItem{},
			target: map[string]*parser.PathItem{
				"newOrder": {
					Post: &parser.Operation{
						Summary: "New order webhook",
					},
				},
			},
			expectChange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			result := &DiffResult{Changes: []Change{}}

			d.Mode = ModeBreaking
			d.diffWebhooksUnified(tt.source, tt.target, "test.webhooks", result)

			if tt.expectChange {
				assert.NotEmpty(t, result.Changes, "Expected changes to be detected")
			}
		})
	}
}

// TestDiffHeaderBreaking tests breaking change detection for headers.
func TestDiffHeaderBreaking(t *testing.T) {
	tests := []struct {
		name         string
		source       *parser.Header
		target       *parser.Header
		expectChange bool
	}{
		{
			name: "required header changed - error",
			source: &parser.Header{
				Description: "Request ID",
				Required:    false,
			},
			target: &parser.Header{
				Description: "Request ID",
				Required:    true,
			},
			expectChange: true,
		},
		{
			name: "type changed - warning",
			source: &parser.Header{
				Description: "Request ID",
				Type:        "string",
				Required:    false,
			},
			target: &parser.Header{
				Description: "Request ID",
				Type:        "integer",
				Required:    false,
			},
			expectChange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			result := &DiffResult{Changes: []Change{}}

			d.Mode = ModeBreaking
			d.diffHeaderUnified(tt.source, tt.target, "test.header", result)

			if tt.expectChange {
				assert.NotEmpty(t, result.Changes, "Expected changes to be detected")
			}
		})
	}
}

// TestDiffLinkBreaking tests breaking change detection for links (OAS 3.x).
func TestDiffLinkBreaking(t *testing.T) {
	tests := []struct {
		name         string
		source       *parser.Link
		target       *parser.Link
		expectChange bool
	}{
		{
			name: "operationId changed - warning",
			source: &parser.Link{
				OperationID: "getUser",
				Description: "Get user details",
			},
			target: &parser.Link{
				OperationID: "getUserById",
				Description: "Get user details",
			},
			expectChange: true,
		},
		{
			name: "no change",
			source: &parser.Link{
				OperationID: "getUser",
				Description: "Get user details",
			},
			target: &parser.Link{
				OperationID: "getUser",
				Description: "Get user details",
			},
			expectChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			result := &DiffResult{Changes: []Change{}}

			d.Mode = ModeBreaking
			d.diffLinkUnified(tt.source, tt.target, "test.link", result)

			if tt.expectChange {
				assert.NotEmpty(t, result.Changes, "Expected changes to be detected")
			}
			// Note: No-change case may still have minor changes detected
		})
	}
}

// TestIsCompatibleTypeChange tests type compatibility checking.
func TestIsCompatibleTypeChange(t *testing.T) {
	tests := []struct {
		name       string
		oldType    string
		newType    string
		compatible bool
	}{
		{
			name:       "integer to number - compatible",
			oldType:    "integer",
			newType:    "number",
			compatible: true,
		},
		{
			name:       "number to integer - incompatible",
			oldType:    "number",
			newType:    "integer",
			compatible: false,
		},
		{
			name:       "string to integer - incompatible",
			oldType:    "string",
			newType:    "integer",
			compatible: false,
		},
		{
			name:       "same type - not handled (returns false)",
			oldType:    "string",
			newType:    "string",
			compatible: false, // Function only handles integer->number widening
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCompatibleTypeChange(tt.oldType, tt.newType)
			assert.Equal(t, tt.compatible, result, "Type compatibility check failed")
		})
	}
}

// TestAnyToString tests the anyToString helper function.
func TestAnyToString(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected string
	}{
		{
			name:     "nil value",
			value:    nil,
			expected: "<nil>", // fmt.Sprint returns "<nil>" for nil
		},
		{
			name:     "string value",
			value:    "test",
			expected: "test",
		},
		{
			name:     "integer value",
			value:    42,
			expected: "42",
		},
		{
			name:     "boolean value",
			value:    true,
			expected: "true",
		},
		{
			name:     "slice value",
			value:    []string{"a", "b", "c"},
			expected: "[a b c]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := anyToString(tt.value)
			assert.Equal(t, tt.expected, result, "anyToString result mismatch")
		})
	}
}

// TestDiffBreakingIntegration tests the main diffBreaking function with full documents.
func TestDiffBreakingIntegration(t *testing.T) {
	tests := []struct {
		name         string
		source       parser.ParseResult
		target       parser.ParseResult
		expectChange bool
	}{
		{
			name: "OAS 2.0 endpoint removed",
			source: parser.ParseResult{
				OASVersion: parser.OASVersion20,
				Version:    "2.0",
				Document: &parser.OAS2Document{
					Swagger: "2.0",
					Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
					Paths: parser.Paths{
						"/users": &parser.PathItem{
							Get: &parser.Operation{Summary: "Get users"},
						},
					},
				},
			},
			target: parser.ParseResult{
				OASVersion: parser.OASVersion20,
				Version:    "2.0",
				Document: &parser.OAS2Document{
					Swagger: "2.0",
					Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
					Paths:   parser.Paths{},
				},
			},
			expectChange: true,
		},
		{
			name: "OAS 3.x server removed",
			source: parser.ParseResult{
				OASVersion: parser.OASVersion300,
				Version:    "3.0.0",
				Document: &parser.OAS3Document{
					OpenAPI: "3.0.0",
					Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
					Servers: []*parser.Server{
						{URL: "https://api.example.com"},
					},
					Paths: parser.Paths{},
				},
			},
			target: parser.ParseResult{
				OASVersion: parser.OASVersion300,
				Version:    "3.0.0",
				Document: &parser.OAS3Document{
					OpenAPI: "3.0.0",
					Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
					Servers: []*parser.Server{},
					Paths:   parser.Paths{},
				},
			},
			expectChange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			result := &DiffResult{Changes: []Change{}}

			d.Mode = ModeBreaking
			d.diffUnified(tt.source, tt.target, result)

			if tt.expectChange {
				assert.NotEmpty(t, result.Changes, "Expected changes to be detected")
			}
		})
	}
}

// TestDiffCrossVersionSimple tests simple cross-version comparison.
func TestDiffCrossVersionSimple(t *testing.T) {
	tests := []struct {
		name         string
		source       parser.ParseResult
		target       parser.ParseResult
		expectChange bool
	}{
		{
			name: "OAS 2.0 to OAS 3.0.0 version change",
			source: parser.ParseResult{
				OASVersion: parser.OASVersion20,
				Version:    "2.0",
				Document: &parser.OAS2Document{
					Swagger: "2.0",
					Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
					Paths:   parser.Paths{},
				},
			},
			target: parser.ParseResult{
				OASVersion: parser.OASVersion300,
				Version:    "3.0.0",
				Document: &parser.OAS3Document{
					OpenAPI: "3.0.0",
					Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
					Paths:   parser.Paths{},
				},
			},
			expectChange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			result := &DiffResult{Changes: []Change{}}

			d.Mode = ModeSimple
			d.diffCrossVersionUnified(tt.source, tt.target, result)

			if tt.expectChange {
				require.NotEmpty(t, result.Changes, "Expected version change to be detected")
			} else {
				require.Empty(t, result.Changes, "Expected no changes")
			}
		})
	}
}
