package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOAS2(t *testing.T) {
	parser := New()
	result, err := parser.Parse("../../testdata/petstore-2.0.yaml")
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
	result, err := parser.Parse("../../testdata/petstore-3.0.yaml")
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
	result, err := parser.Parse("../../testdata/petstore-3.1.yaml")
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
	result, err := parser.Parse("../../testdata/petstore-3.2.yaml")
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

func TestParseValidationErrors(t *testing.T) {
	parser := New()
	data := []byte(`
swagger: "2.0"
paths: {}
`)
	result, err := parser.ParseBytes(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(result.Errors) == 0 {
		t.Error("Expected validation errors for missing required fields")
	}

	// Should have errors for missing info
	hasInfoError := false
	for _, err := range result.Errors {
		// Check if error message mentions missing info field
		errMsg := err.Error()
		if strings.Contains(errMsg, "info") && strings.Contains(errMsg, "missing") {
			hasInfoError = true
			break
		}
	}
	if !hasInfoError {
		t.Errorf("Expected error for missing info field, got: %v", result.Errors)
	}
}

func TestParseWithValidationDisabled(t *testing.T) {
	parser := New()
	parser.ValidateStructure = false

	data := []byte(`
swagger: "2.0"
paths: {}
`)
	result, err := parser.ParseBytes(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(result.Errors) > 0 {
		t.Error("Should not have validation errors when validation is disabled")
	}
}

func TestResolveLocalRefs(t *testing.T) {
	parser := New()
	parser.ResolveRefs = true

	result, err := parser.Parse("../../testdata/petstore-3.0.yaml")
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

	result, err := parser.Parse("../../testdata/with-external-refs.yaml")
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

func TestDetectVersion(t *testing.T) {
	parser := New()

	tests := []struct {
		name     string
		data     map[string]interface{}
		expected string
		wantErr  bool
	}{
		{
			name:     "OAS 2.0",
			data:     map[string]interface{}{"swagger": "2.0"},
			expected: "2.0",
			wantErr:  false,
		},
		{
			name:     "OAS 3.0.0",
			data:     map[string]interface{}{"openapi": "3.0.0"},
			expected: "3.0.0",
			wantErr:  false,
		},
		{
			name:     "OAS 3.1.0",
			data:     map[string]interface{}{"openapi": "3.1.0"},
			expected: "3.1.0",
			wantErr:  false,
		},
		{
			name:     "Missing version",
			data:     map[string]interface{}{"info": "test"},
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := parser.detectVersion(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("detectVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if version != tt.expected {
				t.Errorf("detectVersion() = %v, want %v", version, tt.expected)
			}
		})
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

	tmpfile, err := os.CreateTemp("", "test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	if _, err := tmpfile.Write([]byte(jsonData)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	parser := New()
	result, err := parser.Parse(tmpfile.Name())
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

	testFile := filepath.Join(cwd, "../../testdata/petstore-3.0.yaml")
	parser := New()
	result, err := parser.Parse(testFile)
	if err != nil {
		t.Fatalf("Failed to parse with absolute path: %v", err)
	}

	if result.Version != "3.0.3" {
		t.Errorf("Expected version 3.0.3, got %s", result.Version)
	}
}

// TestVersionInRange tests the semantic version range checking
// This test would have caught the bug where string comparison was used
func TestVersionInRange(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		minVersion string
		maxVersion string
		expected   bool
	}{
		// Exclusive upper bound tests [min, max)
		{
			name:       "3.0.0 in range [3.0.0, 4.0.0) exclusive",
			version:    "3.0.0",
			minVersion: "3.0.0",
			maxVersion: "4.0.0",
			expected:   true,
		},
		{
			name:       "3.1.0 in range [3.0.0, 4.0.0) exclusive",
			version:    "3.1.0",
			minVersion: "3.0.0",
			maxVersion: "4.0.0",
			expected:   true,
		},
		{
			name:       "3.10.0 in range [3.0.0, 4.0.0) exclusive - would fail with string comparison",
			version:    "3.10.0",
			minVersion: "3.0.0",
			maxVersion: "4.0.0",
			expected:   true,
		},
		{
			name:       "3.2.0 in range [3.0.0, 4.0.0) exclusive",
			version:    "3.2.0",
			minVersion: "3.0.0",
			maxVersion: "4.0.0",
			expected:   true,
		},
		{
			name:       "3.99.99 in range [3.0.0, 4.0.0) exclusive",
			version:    "3.99.99",
			minVersion: "3.0.0",
			maxVersion: "4.0.0",
			expected:   true,
		},
		{
			name:       "4.0.0 not in range [3.0.0, 4.0.0) - exclusive upper bound",
			version:    "4.0.0",
			minVersion: "3.0.0",
			maxVersion: "4.0.0",
			expected:   false,
		},
		{
			name:       "2.0 not in range [3.0.0, 4.0.0) exclusive",
			version:    "2.0",
			minVersion: "3.0.0",
			maxVersion: "4.0.0",
			expected:   false,
		},
		{
			name:       "3.0.0 in range [3.0.0, 3.1.0) exclusive",
			version:    "3.0.0",
			minVersion: "3.0.0",
			maxVersion: "3.1.0",
			expected:   true,
		},
		{
			name:       "3.0.9 in range [3.0.0, 3.1.0) exclusive",
			version:    "3.0.9",
			minVersion: "3.0.0",
			maxVersion: "3.1.0",
			expected:   true,
		},
		{
			name:       "3.1.0 not in range [3.0.0, 3.1.0) - exclusive upper bound",
			version:    "3.1.0",
			minVersion: "3.0.0",
			maxVersion: "3.1.0",
			expected:   false,
		},

		// No upper bound tests (empty maxVersion) - equivalent to v >= minVersion
		{
			name:       "3.1.0 >= 3.1.0 (no upper bound)",
			version:    "3.1.0",
			minVersion: "3.1.0",
			maxVersion: "",
			expected:   true,
		},
		{
			name:       "3.2.0 >= 3.1.0 (no upper bound)",
			version:    "3.2.0",
			minVersion: "3.1.0",
			maxVersion: "",
			expected:   true,
		},
		{
			name:       "3.10.0 >= 3.1.0 (no upper bound) - would fail with string comparison",
			version:    "3.10.0",
			minVersion: "3.1.0",
			maxVersion: "",
			expected:   true,
		},
		{
			name:       "3.0.9 not >= 3.1.0 (no upper bound)",
			version:    "3.0.9",
			minVersion: "3.1.0",
			maxVersion: "",
			expected:   false,
		},

		// Less than tests (min="0.0.0", exclusive max) - equivalent to v < maxVersion
		{
			name:       "3.0.0 < 3.1.0 (lower bound 0.0.0)",
			version:    "3.0.0",
			minVersion: "0.0.0",
			maxVersion: "3.1.0",
			expected:   true,
		},
		{
			name:       "3.1.0 not < 3.1.0 (lower bound 0.0.0)",
			version:    "3.1.0",
			minVersion: "0.0.0",
			maxVersion: "3.1.0",
			expected:   false,
		},
		{
			name:       "3.2.0 < 3.10.0 (lower bound 0.0.0) - would be wrong with string comparison",
			version:    "3.2.0",
			minVersion: "0.0.0",
			maxVersion: "3.10.0",
			expected:   true,
		},
		{
			name:       "3.10.0 not < 3.2.0 (lower bound 0.0.0) - would be wrong with string comparison",
			version:    "3.10.0",
			minVersion: "0.0.0",
			maxVersion: "3.2.0",
			expected:   false,
		},

		// Invalid version string
		{
			name:       "invalid version string",
			version:    "invalid",
			minVersion: "3.0.0",
			maxVersion: "4.0.0",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := versionInRangeExclusive(tt.version, tt.minVersion, tt.maxVersion)
			if result != tt.expected {
				t.Errorf("versionInRangeExclusive(%s, %s, %s) = %v, want %v",
					tt.version, tt.minVersion, tt.maxVersion, result, tt.expected)
			}
		})
	}
}

// TestWebhooksVersionValidation tests that webhooks are properly validated based on version
func TestWebhooksVersionValidation(t *testing.T) {
	tests := []struct {
		name            string
		version         string
		includeWebhooks bool
		expectError     bool
		errorContains   string
	}{
		{
			name:            "Webhooks in OAS 3.0.0 should error",
			version:         "3.0.0",
			includeWebhooks: true,
			expectError:     true,
			errorContains:   "webhooks",
		},
		{
			name:            "Webhooks in OAS 3.0.1 should error",
			version:         "3.0.1",
			includeWebhooks: true,
			expectError:     true,
			errorContains:   "webhooks",
		},
		{
			name:            "Webhooks in OAS 3.1.0 should be valid",
			version:         "3.1.0",
			includeWebhooks: true,
			expectError:     false,
		},
		{
			name:            "Webhooks in OAS 3.2.0 should be valid",
			version:         "3.2.0",
			includeWebhooks: true,
			expectError:     false,
		},
		{
			name:            "No webhooks in OAS 3.0.0 should be valid",
			version:         "3.0.0",
			includeWebhooks: false,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New()

			webhooksSection := ""
			if tt.includeWebhooks {
				webhooksSection = `
webhooks:
  newPet:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
      responses:
        '200':
          description: Success
`
			}

			data := []byte(`openapi: "` + tt.version + `"
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '200':
          description: Success
` + webhooksSection)

			result, err := parser.ParseBytes(data)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			hasWebhookError := false
			for _, e := range result.Errors {
				if strings.Contains(e.Error(), tt.errorContains) {
					hasWebhookError = true
					break
				}
			}

			if tt.expectError && !hasWebhookError {
				t.Errorf("Expected error containing '%s' for version %s with webhooks, but got errors: %v",
					tt.errorContains, tt.version, result.Errors)
			}

			if !tt.expectError && hasWebhookError {
				t.Errorf("Did not expect webhook error for version %s, but got: %v",
					tt.version, result.Errors)
			}
		})
	}
}

// TestPathsRequirementVersionValidation tests that paths requirement is properly validated based on version
func TestPathsRequirementVersionValidation(t *testing.T) {
	tests := []struct {
		name            string
		version         string
		includePaths    bool
		includeWebhooks bool
		expectError     bool
		errorContains   string
	}{
		{
			name:            "OAS 3.0.0 requires paths",
			version:         "3.0.0",
			includePaths:    false,
			includeWebhooks: false,
			expectError:     true,
			errorContains:   "paths",
		},
		{
			name:            "OAS 3.0.2 requires paths",
			version:         "3.0.2",
			includePaths:    false,
			includeWebhooks: false,
			expectError:     true,
			errorContains:   "paths",
		},
		{
			name:            "OAS 3.1.0 requires paths or webhooks",
			version:         "3.1.0",
			includePaths:    false,
			includeWebhooks: false,
			expectError:     true,
			errorContains:   "paths",
		},
		{
			name:            "OAS 3.1.0 with webhooks is valid",
			version:         "3.1.0",
			includePaths:    false,
			includeWebhooks: true,
			expectError:     false,
		},
		{
			name:            "OAS 3.2.0 with webhooks is valid",
			version:         "3.2.0",
			includePaths:    false,
			includeWebhooks: true,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New()

			pathsSection := ""
			if tt.includePaths {
				pathsSection = `paths:
  /test:
    get:
      responses:
        '200':
          description: Success
`
			}

			webhooksSection := ""
			if tt.includeWebhooks {
				webhooksSection = `webhooks:
  newPet:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
      responses:
        '200':
          description: Success
`
			}

			data := []byte(`openapi: "` + tt.version + `"
info:
  title: Test API
  version: 1.0.0
` + pathsSection + webhooksSection)

			result, err := parser.ParseBytes(data)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			hasExpectedError := false
			for _, e := range result.Errors {
				if strings.Contains(e.Error(), tt.errorContains) {
					hasExpectedError = true
					break
				}
			}

			if tt.expectError && !hasExpectedError {
				t.Errorf("Expected error containing '%s' for version %s, but got errors: %v",
					tt.errorContains, tt.version, result.Errors)
			}

			if !tt.expectError && len(result.Errors) > 0 {
				t.Errorf("Did not expect errors for version %s, but got: %v",
					tt.version, result.Errors)
			}
		})
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

// TestInvalidVersionValidation tests that invalid version strings are properly rejected
func TestInvalidVersionValidation(t *testing.T) {
	tests := []struct {
		name          string
		version       string
		expectError   bool
		errorContains string
	}{
		{
			name:          "Version 4.0.0 should be rejected",
			version:       "4.0.0",
			expectError:   true,
			errorContains: "invalid OAS version",
		},
		{
			name:          "Version 2.5.0 should be rejected",
			version:       "2.5.0",
			expectError:   true,
			errorContains: "invalid OAS version",
		},
		{
			name:          "Version 5.0.0 should be rejected",
			version:       "5.0.0",
			expectError:   true,
			errorContains: "invalid OAS version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New()

			data := []byte(`openapi: "` + tt.version + `"
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
			if tt.expectError {
				assert.Nil(t, result)
				assert.ErrorContains(t, err, tt.errorContains)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, result)
			}
		})
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
		errorContains string
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
			errorContains: "path traversal detected",
		},
		{
			name:          "Path traversal with ../../",
			ref:           "../../forbidden.yaml",
			shouldSucceed: false,
			errorContains: "path traversal detected",
		},
		{
			name:          "Path traversal with ../../../",
			ref:           "../../../etc/passwd",
			shouldSucceed: false,
			errorContains: "path traversal detected",
		},
		{
			name:          "Absolute path outside baseDir",
			ref:           restrictedFile,
			shouldSucceed: false,
			errorContains: "path traversal detected",
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
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
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
			} else if !strings.Contains(err.Error(), "path traversal detected") {
				t.Errorf("Expected 'path traversal detected' error for ref '%s', got: %v", ref, err)
			}
		})
	}
}

func TestInvalidStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode string
		oasVersion string
		expectErr  bool
	}{
		{"Valid 200", "200", "2.0", false},
		{"Valid 404", "404", "3.0.0", false},
		{"Valid 2XX wildcard", "2XX", "3.0.0", false},
		{"Valid 5XX wildcard", "5XX", "2.0", false},
		{"Valid default", "default", "3.0.0", false},
		{"Valid extension field x-custom", "x-custom", "3.0.0", false},
		{"Valid extension field x-rate-limit", "x-rate-limit", "2.0", false},
		{"Valid extension field x-", "x-", "3.0.0", false},
		{"Invalid 99 - too low", "99", "3.0.0", true},
		{"Invalid 600 - too high", "600", "2.0", true},
		{"Invalid 6XX - out of range wildcard", "6XX", "3.0.0", true},
		{"Invalid XXX - all wildcards", "XXX", "3.0.0", true},
		{"Invalid 2X3 - mixed wildcard", "2X3", "2.0", true},
		{"Invalid empty string", "", "3.0.0", true},
		{"Invalid two chars", "20", "3.0.0", true},
		{"Invalid four chars", "2000", "2.0", true},
		{"Invalid non-numeric", "abc", "3.0.0", true},
		{"Invalid x without dash", "x", "3.0.0", true},
		{"Invalid xCustom without dash", "xCustom", "2.0", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var spec string
			if tt.oasVersion == "2.0" {
				spec = `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '` + tt.statusCode + `':
          description: Test response
`
			} else {
				spec = `openapi: "` + tt.oasVersion + `"
info:
  title: Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        '` + tt.statusCode + `':
          description: Test response
`
			}

			parser := New()
			result, err := parser.ParseBytes([]byte(spec))

			// Check for invalid status code error in either parse error or validation errors
			// Parse error check (fail-fast during unmarshaling)
			hasStatusCodeError := err != nil && strings.Contains(err.Error(), "invalid status code")

			// Check validation errors (caught during validation phase)
			if !hasStatusCodeError && result != nil {
				for _, e := range result.Errors {
					if strings.Contains(e.Error(), "invalid status code") {
						hasStatusCodeError = true
						break
					}
				}
			}

			if tt.expectErr && !hasStatusCodeError {
				t.Errorf("Expected invalid status code error for '%s', but got no such error. Parse error: %v, Validation errors: %v",
					tt.statusCode, err, result.Errors)
			}

			if !tt.expectErr && hasStatusCodeError {
				t.Errorf("Did not expect invalid status code error for '%s', but got one. Parse error: %v, Validation errors: %v",
					tt.statusCode, err, result.Errors)
			}

			// For valid status codes, ensure parsing succeeded
			if !tt.expectErr && err != nil {
				t.Errorf("Expected successful parse for valid status code '%s', but got parse error: %v",
					tt.statusCode, err)
			}
		})
	}
}

func TestDuplicateOperationIds(t *testing.T) {
	tests := []struct {
		name      string
		spec      string
		expectErr bool
		errorMsg  string
	}{
		{
			name: "OAS 2.0 - Duplicate operationId",
			spec: `swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
  /accounts:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
`,
			expectErr: true,
			errorMsg:  "duplicate operationId",
		},
		{
			name: "OAS 3.0 - Duplicate operationId",
			spec: `openapi: "3.0.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
  /accounts:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
`,
			expectErr: true,
			errorMsg:  "duplicate operationId",
		},
		{
			name: "OAS 3.1 - Unique operationIds",
			spec: `openapi: "3.1.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /users:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
  /accounts:
    get:
      operationId: getAccount
      responses:
        '200':
          description: Success
`,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := New()
			result, err := parser.ParseBytes([]byte(tt.spec))
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			hasDuplicateError := false
			for _, e := range result.Errors {
				if strings.Contains(e.Error(), tt.errorMsg) {
					hasDuplicateError = true
					break
				}
			}

			if tt.expectErr && !hasDuplicateError {
				t.Errorf("Expected duplicate operationId error, but got none. Errors: %v", result.Errors)
			}

			if !tt.expectErr && hasDuplicateError {
				t.Errorf("Did not expect duplicate operationId error, but got one. Errors: %v", result.Errors)
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
