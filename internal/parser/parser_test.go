package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

// TestParseOAS310EdgeCase tests parsing OAS 3.10.0 which would fail with string comparison
func TestParseOAS310EdgeCase(t *testing.T) {
	parser := New()
	data := []byte(`
openapi: "3.10.0"
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
	if err != nil {
		t.Fatalf("Failed to parse OAS 3.10.0: %v", err)
	}

	if result.Version != "3.10.0" {
		t.Errorf("Expected version 3.10.0, got %s", result.Version)
	}

	// Should be parsed as OAS3Document since 3.10.0 is >= 3.0.0 and < 4.0.0
	_, ok := result.Document.(*OAS3Document)
	if !ok {
		t.Fatalf("Expected OAS3Document for version 3.10.0, got %T", result.Document)
	}

	// Should have no validation errors
	if len(result.Errors) > 0 {
		t.Errorf("Unexpected validation errors for OAS 3.10.0: %v", result.Errors)
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
			name:            "Webhooks in OAS 3.0.9 should error",
			version:         "3.0.9",
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
			name:            "Webhooks in OAS 3.10.0 should be valid - edge case",
			version:         "3.10.0",
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
			name:            "OAS 3.0.9 requires paths",
			version:         "3.0.9",
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
		{
			name:            "OAS 3.10.0 with only webhooks is valid - edge case",
			version:         "3.10.0",
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

// TestRCVersionsExcluded tests that release candidate versions are handled
// These are not official releases but may appear in the wild
func TestRCVersionsExcluded(t *testing.T) {
	rcVersions := []string{
		"3.0.0-rc0",
		"3.0.0-rc1",
		"3.0.0-rc2",
		"3.1.0-rc0",
		"3.1.0-rc1",
	}

	for _, rcVersion := range rcVersions {
		t.Run("RC_"+rcVersion, func(t *testing.T) {
			parser := New()

			data := []byte(`
openapi: "` + rcVersion + `"
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
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}

			// RC versions should be detected as-is
			if result.Version != rcVersion {
				t.Errorf("Version detection failed: expected %s, got %s", rcVersion, result.Version)
			}

			// RC versions may or may not parse successfully depending on implementation
			// At minimum, they should be detected and not cause a crash
			t.Logf("RC version %s parsed with %d errors", rcVersion, len(result.Errors))
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
			errorContains: "unsupported OpenAPI version",
		},
		{
			name:          "Version 2.5.0 should be rejected",
			version:       "2.5.0",
			expectError:   true,
			errorContains: "unsupported OpenAPI version",
		},
		{
			name:          "Version 5.0.0 should be rejected",
			version:       "5.0.0",
			expectError:   true,
			errorContains: "unsupported OpenAPI version",
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
		})
	}
}
