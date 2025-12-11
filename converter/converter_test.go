package converter

import (
	"net/url"
	"strings"
	"testing"

	"github.com/erraggy/oastools/internal/testutil"
	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConverterNew tests the New() constructor
func TestConverterNew(t *testing.T) {
	c := New()

	if c == nil {
		t.Fatal("Expected non-nil Converter")
	}

	if c.StrictMode {
		t.Error("Expected StrictMode to be false by default")
	}

	if !c.IncludeInfo {
		t.Error("Expected IncludeInfo to be true by default")
	}
}

// TestConverterConvert tests the Converter.Convert method
func TestConverterConvert(t *testing.T) {
	c := New()
	oas2Doc := testutil.NewSimpleOAS2Document()
	tmpFile := testutil.WriteTempYAML(t, oas2Doc)

	result, err := c.Convert(tmpFile, "3.0.3")
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	if result.SourceOASVersion != parser.OASVersion20 {
		t.Errorf("Expected source version OASVersion20")
	}

	if result.TargetOASVersion != parser.OASVersion303 {
		t.Errorf("Expected target version OASVersion303")
	}
}

// TestConverterConvertParsed tests the Converter.ConvertParsed method
func TestConverterConvertParsed(t *testing.T) {
	c := New()
	oas2Doc := testutil.NewSimpleOAS2Document()
	parseResult := parser.ParseResult{
		Document:   oas2Doc,
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Data:       make(map[string]any),
		SourcePath: "test.yaml",
		Errors:     []error{},
		Warnings:   []string{},
	}

	result, err := c.ConvertParsed(parseResult, "3.0.3")
	if err != nil {
		t.Fatalf("ConvertParsed failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful conversion")
	}
}

// TestOAS2ToOAS3Conversion tests OAS 2.0 to OAS 3.x conversion
func TestOAS2ToOAS3Conversion(t *testing.T) {
	c := New()
	oas2Doc := testutil.NewDetailedOAS2Document()
	parseResult := parser.ParseResult{
		Document:   oas2Doc,
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Data:       make(map[string]any),
		SourcePath: "test.yaml",
	}

	result, err := c.ConvertParsed(parseResult, "3.0.3")
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	doc, ok := result.Document.(*parser.OAS3Document)
	if !ok {
		t.Fatal("Expected OAS3Document")
	}

	// Verify servers were created from host/basePath
	if len(doc.Servers) == 0 {
		t.Error("Expected servers to be created")
	}

	// Verify components were created
	if doc.Components == nil {
		t.Fatal("Expected components to be created")
	}

	// Verify definitions were converted to schemas
	if len(doc.Components.Schemas) != len(oas2Doc.Definitions) {
		t.Errorf("Expected %d schemas, got %d", len(oas2Doc.Definitions), len(doc.Components.Schemas))
	}
}

// TestOAS3ToOAS2Conversion tests OAS 3.x to OAS 2.0 conversion
func TestOAS3ToOAS2Conversion(t *testing.T) {
	c := New()
	oas3Doc := testutil.NewDetailedOAS3Document()

	// get server URL and host to verify path parameters are handled
	serverURL := oas3Doc.Servers[0].URL
	u, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("Failed to parse server URL: %v", err)
	}
	host := u.Host
	oas3Doc.Servers[0].URL = serverURL + "/{foo}/bar"
	parseResult := parser.ParseResult{
		Document:   oas3Doc,
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Data:       make(map[string]any),
		SourcePath: "test.yaml",
	}

	result, err := c.ConvertParsed(parseResult, "2.0")
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	doc, ok := result.Document.(*parser.OAS2Document)
	if !ok {
		t.Fatal("Expected OAS2Document")
	}

	// Verify host/basePath were created from servers
	if doc.Host != host {
		t.Errorf("Expected host to be set to %q, got %q", host, doc.Host)
	}

	// Verify definitions were created from schemas
	if oas3Doc.Components != nil && len(oas3Doc.Components.Schemas) > 0 {
		if len(doc.Definitions) != len(oas3Doc.Components.Schemas) {
			t.Errorf("Expected %d definitions, got %d", len(oas3Doc.Components.Schemas), len(doc.Definitions))
		}
	}
}

// TestOAS3ToOAS3Conversion tests OAS 3.x to OAS 3.y version update
func TestOAS3ToOAS3Conversion(t *testing.T) {
	c := New()
	oas3Doc := testutil.NewDetailedOAS3Document()
	oas3Doc.OpenAPI = "3.0.3"
	oas3Doc.OASVersion = parser.OASVersion303

	parseResult := parser.ParseResult{
		Document:   oas3Doc,
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Data:       make(map[string]any),
		SourcePath: "test.yaml",
	}

	result, err := c.ConvertParsed(parseResult, "3.1.0")
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	doc, ok := result.Document.(*parser.OAS3Document)
	if !ok {
		t.Fatal("Expected OAS3Document")
	}

	if doc.OpenAPI != "3.1.0" {
		t.Errorf("Expected OpenAPI version 3.1.0, got %s", doc.OpenAPI)
	}

	// Should have an info message about version update
	hasInfoMessage := false
	for _, issue := range result.Issues {
		if issue.Severity == SeverityInfo {
			hasInfoMessage = true
			break
		}
	}

	if !hasInfoMessage {
		t.Error("Expected info message about version update")
	}
}

// TestSameVersionConversion tests conversion when source and target are the same
func TestSameVersionConversion(t *testing.T) {
	c := New()
	oas2Doc := testutil.NewSimpleOAS2Document()
	parseResult := parser.ParseResult{
		Document:   oas2Doc,
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Data:       make(map[string]any),
		SourcePath: "test.yaml",
	}

	result, err := c.ConvertParsed(parseResult, "2.0")
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	if !result.Success {
		t.Error("Expected successful conversion")
	}

	// Should have an info message about no conversion needed
	found := false
	for _, issue := range result.Issues {
		if issue.Severity == SeverityInfo && issue.Path == "document" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected info message about no conversion needed")
	}
}

// TestInvalidTargetVersion tests error handling for invalid target version
func TestInvalidTargetVersion(t *testing.T) {
	c := New()
	oas2Doc := testutil.NewSimpleOAS2Document()
	parseResult := parser.ParseResult{
		Document:   oas2Doc,
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Data:       make(map[string]any),
		SourcePath: "test.yaml",
	}

	_, err := c.ConvertParsed(parseResult, "invalid.version")
	if err == nil {
		t.Fatal("Expected error for invalid target version")
	}
	assert.Contains(t, err.Error(), "converter: invalid target version")
}

// TestConvert_ParseError tests error when file cannot be parsed
func TestConvert_ParseError(t *testing.T) {
	c := New()
	_, err := c.Convert("nonexistent-file.yaml", "3.0.3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "converter: failed to parse specification")
}

// TestStrictMode tests strict mode behavior
func TestStrictMode(t *testing.T) {
	c := New()
	c.StrictMode = true

	// Create an OAS 3.x document with webhooks that will cause critical issues when converting to 2.0
	oas3Doc := testutil.NewDetailedOAS3Document()
	oas3Doc.Webhooks = map[string]*parser.PathItem{
		"newPet": {},
	}

	parseResult := parser.ParseResult{
		Document:   oas3Doc,
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Data:       make(map[string]any),
		SourcePath: "test.yaml",
	}

	// Should fail in strict mode due to critical issues
	_, err := c.ConvertParsed(parseResult, "2.0")
	if err == nil {
		t.Error("Expected error in strict mode with critical issues")
	}
}

// TestIncludeInfo tests IncludeInfo flag behavior
func TestIncludeInfo(t *testing.T) {
	c := New()
	c.IncludeInfo = false

	oas2Doc := testutil.NewSimpleOAS2Document()
	parseResult := parser.ParseResult{
		Document:   oas2Doc,
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Data:       make(map[string]any),
		SourcePath: "test.yaml",
	}

	result, err := c.ConvertParsed(parseResult, "3.0.3")
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	// Check that info messages are filtered out
	for _, issue := range result.Issues {
		if issue.Severity == SeverityInfo {
			t.Error("Expected no info messages when IncludeInfo is false")
		}
	}

	if result.InfoCount != 0 {
		t.Errorf("Expected InfoCount to be 0, got %d", result.InfoCount)
	}
}

// TestConversionResultHelpers tests ConversionResult helper methods
func TestConversionResultHelpers(t *testing.T) {
	result := &ConversionResult{
		Issues: []ConversionIssue{
			{Severity: SeverityInfo},
			{Severity: SeverityWarning},
			{Severity: SeverityCritical},
		},
		InfoCount:     1,
		WarningCount:  1,
		CriticalCount: 1,
	}

	if !result.HasCriticalIssues() {
		t.Error("Expected HasCriticalIssues to return true")
	}

	if !result.HasWarnings() {
		t.Error("Expected HasWarnings to return true")
	}

	result.CriticalCount = 0
	if result.HasCriticalIssues() {
		t.Error("Expected HasCriticalIssues to return false")
	}

	result.WarningCount = 0
	if result.HasWarnings() {
		t.Error("Expected HasWarnings to return false")
	}
}

// Add these tests to converter_test.go:

func TestSeverityString(t *testing.T) {
	tests := []struct {
		severity Severity
		expected string
	}{
		{SeverityInfo, "info"},
		{SeverityWarning, "warning"},
		{SeverityCritical, "critical"},
		{Severity(999), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.severity.String(); got != tt.expected {
			t.Errorf("Severity.String() = %v, want %v", got, tt.expected)
		}
	}
}

func TestConversionIssueString(t *testing.T) {
	tests := []struct {
		name  string
		issue ConversionIssue
		want  string
	}{
		{
			name: "critical issue with context",
			issue: ConversionIssue{
				Path:     "paths./pets",
				Message:  "TRACE method not supported",
				Severity: SeverityCritical,
				Context:  "OAS 2.0 does not support TRACE",
			},
			// Test that it contains expected strings
		},
		{
			name: "warning without context",
			issue: ConversionIssue{
				Path:     "servers[0]",
				Message:  "Multiple servers found",
				Severity: SeverityWarning,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.issue.String()
			if !strings.Contains(got, tt.issue.Path) {
				t.Errorf("String() should contain path")
			}
			if !strings.Contains(got, tt.issue.Message) {
				t.Errorf("String() should contain message")
			}
		})
	}
}

// TestRefRewritingOAS2ToOAS3 tests that $ref paths are properly rewritten when converting from OAS 2.0 to OAS 3.x
func TestRefRewritingOAS2ToOAS3(t *testing.T) {
	// Create OAS 2.0 document with refs
	oas2Doc := &parser.OAS2Document{
		Swagger:    "2.0",
		OASVersion: parser.OASVersion20,
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(parser.Paths),
		Definitions: map[string]*parser.Schema{
			"Pet": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"name": {Type: "string"},
				},
			},
			"Owner": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"pet": {
						Ref: "#/definitions/Pet", // This should be rewritten
					},
				},
			},
		},
	}

	// Add path with ref
	oas2Doc.Paths["/pets"] = &parser.PathItem{
		Get: &parser.Operation{
			Responses: &parser.Responses{
				Codes: map[string]*parser.Response{
					"200": {
						Description: "Success",
						Schema: &parser.Schema{
							Ref: "#/definitions/Pet", // This should be rewritten
						},
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Document:   oas2Doc,
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Data:       make(map[string]any),
		SourcePath: "test.yaml",
	}

	c := New()
	result, err := c.ConvertParsed(parseResult, "3.0.3")
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected successful conversion")
	}

	// Verify document is OAS 3.x
	oas3Doc, ok := result.Document.(*parser.OAS3Document)
	if !ok {
		t.Fatal("Expected OAS3Document")
	}

	// Verify refs were rewritten in components/schemas
	ownerSchema := oas3Doc.Components.Schemas["Owner"]
	if ownerSchema == nil {
		t.Fatal("Owner schema not found")
	}

	petProp := ownerSchema.Properties["pet"]
	if petProp == nil {
		t.Fatal("Pet property not found")
	}

	expectedRef := "#/components/schemas/Pet"
	if petProp.Ref != expectedRef {
		t.Errorf("Expected ref '%s', got '%s'", expectedRef, petProp.Ref)
	}

	// Verify refs were rewritten in paths
	pathItem := oas3Doc.Paths["/pets"]
	if pathItem == nil {
		t.Fatal("Path /pets not found")
	}

	responseSchema := pathItem.Get.Responses.Codes["200"].Content["application/json"].Schema
	if responseSchema == nil {
		t.Fatal("Response schema not found")
	}

	if responseSchema.Ref != expectedRef {
		t.Errorf("Expected response schema ref '%s', got '%s'", expectedRef, responseSchema.Ref)
	}

	t.Logf("Successfully converted and verified $ref rewriting from OAS 2.0 to OAS 3.x")
}

// TestRefRewritingOAS3ToOAS2 tests that $ref paths are properly rewritten when converting from OAS 3.x to OAS 2.0
func TestRefRewritingOAS3ToOAS2(t *testing.T) {
	// Create OAS 3.x document with refs
	oas3Doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: make(parser.Paths),
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name": {Type: "string"},
					},
				},
				"Owner": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"pet": {
							Ref: "#/components/schemas/Pet", // This should be rewritten
						},
					},
				},
			},
		},
	}

	// Add path with ref
	oas3Doc.Paths["/pets"] = &parser.PathItem{
		Get: &parser.Operation{
			Responses: &parser.Responses{
				Codes: map[string]*parser.Response{
					"200": {
						Description: "Success",
						Content: map[string]*parser.MediaType{
							"application/json": {
								Schema: &parser.Schema{
									Ref: "#/components/schemas/Pet", // This should be rewritten
								},
							},
						},
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Document:   oas3Doc,
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Data:       make(map[string]any),
		SourcePath: "test.yaml",
	}

	c := New()
	result, err := c.ConvertParsed(parseResult, "2.0")
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected successful conversion")
	}

	// Verify document is OAS 2.0
	oas2Doc, ok := result.Document.(*parser.OAS2Document)
	if !ok {
		t.Fatal("Expected OAS2Document")
	}

	// Verify refs were rewritten in definitions
	ownerSchema := oas2Doc.Definitions["Owner"]
	if ownerSchema == nil {
		t.Fatal("Owner schema not found")
	}

	petProp := ownerSchema.Properties["pet"]
	if petProp == nil {
		t.Fatal("Pet property not found")
	}

	expectedRef := "#/definitions/Pet"
	if petProp.Ref != expectedRef {
		t.Errorf("Expected ref '%s', got '%s'", expectedRef, petProp.Ref)
	}

	// Verify refs were rewritten in paths
	pathItem := oas2Doc.Paths["/pets"]
	if pathItem == nil {
		t.Fatal("Path /pets not found")
	}

	responseSchema := pathItem.Get.Responses.Codes["200"].Schema
	if responseSchema == nil {
		t.Fatal("Response schema not found")
	}

	if responseSchema.Ref != expectedRef {
		t.Errorf("Expected response schema ref '%s', got '%s'", expectedRef, responseSchema.Ref)
	}

	t.Logf("Successfully converted and verified $ref rewriting from OAS 3.x to OAS 2.0")
}

// TestRefRewritingNestedSchemas tests that nested schema refs are properly rewritten
func TestRefRewritingNestedSchemas(t *testing.T) {
	// Create OAS 2.0 document with nested refs (avoiding Items field due to deep copy limitations)
	oas2Doc := &parser.OAS2Document{
		Swagger:    "2.0",
		OASVersion: parser.OASVersion20,
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: parser.Paths{},
		Definitions: map[string]*parser.Schema{
			"Pet": {
				Type: "object",
			},
			"ComplexObject": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"favorite": {
						Ref: "#/definitions/Pet", // Should be rewritten
					},
					"owner": {
						Type: "object",
						Properties: map[string]*parser.Schema{
							"pet": {
								Ref: "#/definitions/Pet", // Nested ref, should be rewritten
							},
						},
					},
				},
				AllOf: []*parser.Schema{
					{Ref: "#/definitions/Pet"}, // Should be rewritten
				},
				AnyOf: []*parser.Schema{
					{Ref: "#/definitions/Pet"}, // Should be rewritten
				},
				OneOf: []*parser.Schema{
					{Ref: "#/definitions/Pet"}, // Should be rewritten
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Document:   oas2Doc,
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Data:       make(map[string]any),
		SourcePath: "test.yaml",
	}

	c := New()
	result, err := c.ConvertParsed(parseResult, "3.0.3")
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	oas3Doc := result.Document.(*parser.OAS3Document)
	expectedRef := "#/components/schemas/Pet"

	// Check ComplexObject nested refs
	complexSchema := oas3Doc.Components.Schemas["ComplexObject"]

	// Check direct property ref
	if complexSchema.Properties["favorite"].Ref != expectedRef {
		t.Errorf("ComplexObject.favorite ref not rewritten: expected '%s', got '%s'", expectedRef, complexSchema.Properties["favorite"].Ref)
	}

	// Check nested property ref
	ownerPetRef := complexSchema.Properties["owner"].Properties["pet"].Ref
	if ownerPetRef != expectedRef {
		t.Errorf("ComplexObject.owner.pet ref not rewritten: expected '%s', got '%s'", expectedRef, ownerPetRef)
	}

	// Check allOf ref
	if complexSchema.AllOf[0].Ref != expectedRef {
		t.Errorf("ComplexObject.allOf[0] ref not rewritten: expected '%s', got '%s'", expectedRef, complexSchema.AllOf[0].Ref)
	}

	// Check anyOf ref
	if complexSchema.AnyOf[0].Ref != expectedRef {
		t.Errorf("ComplexObject.anyOf[0] ref not rewritten: expected '%s', got '%s'", expectedRef, complexSchema.AnyOf[0].Ref)
	}

	// Check oneOf ref
	if complexSchema.OneOf[0].Ref != expectedRef {
		t.Errorf("ComplexObject.oneOf[0] ref not rewritten: expected '%s', got '%s'", expectedRef, complexSchema.OneOf[0].Ref)
	}

	t.Logf("Successfully verified nested schema $ref rewriting")
}

// TestRefRewritingParameters tests that parameter refs are properly rewritten
func TestRefRewritingParameters(t *testing.T) {
	// Create OAS 2.0 document with parameter refs
	oas2Doc := &parser.OAS2Document{
		Swagger:    "2.0",
		OASVersion: parser.OASVersion20,
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Paths: parser.Paths{
			"/pets/{petId}": &parser.PathItem{
				Get: &parser.Operation{
					Parameters: []*parser.Parameter{
						{Ref: "#/parameters/PetId"}, // Should be rewritten
					},
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "Success"},
						},
					},
				},
			},
		},
		Parameters: map[string]*parser.Parameter{
			"PetId": {
				Name:     "petId",
				In:       "path",
				Required: true,
				Type:     "string",
			},
		},
	}

	parseResult := parser.ParseResult{
		Document:   oas2Doc,
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Data:       make(map[string]any),
		SourcePath: "test.yaml",
	}

	c := New()
	result, err := c.ConvertParsed(parseResult, "3.0.3")
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	oas3Doc := result.Document.(*parser.OAS3Document)
	expectedRef := "#/components/parameters/PetId"

	// Check parameter ref was rewritten
	pathItem := oas3Doc.Paths["/pets/{petId}"]
	if pathItem.Get.Parameters[0].Ref != expectedRef {
		t.Errorf("Parameter ref not rewritten: expected '%s', got '%s'", expectedRef, pathItem.Get.Parameters[0].Ref)
	}

	t.Logf("Successfully verified parameter $ref rewriting")
}

// TestJSONFormatPreservation tests that JSON input produces JSON output
func TestJSONFormatPreservation(t *testing.T) {
	// Test with JSON file
	c := New()
	result, err := c.Convert("../testdata/minimal-oas2.json", "3.0.3")
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	// Verify source format was detected as JSON
	if result.SourceFormat != parser.SourceFormatJSON {
		t.Errorf("Expected source format to be JSON, got %s", result.SourceFormat)
	}

	t.Logf("Successfully verified JSON format detection")
}

// TestYAMLFormatPreservation tests that YAML input preserves YAML format
func TestYAMLFormatPreservation(t *testing.T) {
	// Test with YAML file
	c := New()
	result, err := c.Convert("../testdata/minimal-oas2.yaml", "3.0.3")
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	// Verify source format was detected as YAML
	if result.SourceFormat != parser.SourceFormatYAML {
		t.Errorf("Expected source format to be YAML, got %s", result.SourceFormat)
	}

	t.Logf("Successfully verified YAML format detection")
}

// ========================================
// Tests for metric propagation
// ========================================

// TestConvertParsedPropagatesMetrics tests that LoadTime and SourceSize are propagated from ParseResult to ConversionResult
func TestConvertParsedPropagatesMetrics(t *testing.T) {
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/minimal-oas2.yaml"),
		parser.WithValidateStructure(true),
	)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	c := New()
	result, err := c.ConvertParsed(*parseResult, "3.0.3")
	if err != nil {
		t.Fatalf("ConvertParsed() error = %v", err)
	}

	// Verify metrics are propagated
	if result.LoadTime != parseResult.LoadTime {
		t.Errorf("LoadTime not propagated: got %v, want %v", result.LoadTime, parseResult.LoadTime)
	}
	if result.SourceSize != parseResult.SourceSize {
		t.Errorf("SourceSize not propagated: got %d, want %d", result.SourceSize, parseResult.SourceSize)
	}

	// Verify metrics are non-zero (they should have been captured during parsing)
	if result.LoadTime == 0 {
		t.Error("Expected LoadTime to be > 0 after propagation")
	}
	if result.SourceSize == 0 {
		t.Error("Expected SourceSize to be > 0 after propagation")
	}
}

// TestConvertWithOptions_FilePath tests the functional options API with file path
func TestConvertWithOptions_FilePath(t *testing.T) {
	result, err := ConvertWithOptions(
		WithFilePath("../testdata/minimal-oas2.yaml"),
		WithTargetVersion("3.0.3"),
		WithIncludeInfo(true),
	)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "2.0", result.SourceVersion)
	assert.Equal(t, "3.0.3", result.TargetVersion)
}

// TestConvertWithOptions_Parsed tests the functional options API with parsed result
func TestConvertWithOptions_Parsed(t *testing.T) {
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/minimal-oas2.yaml"),
		parser.WithValidateStructure(true),
	)
	require.NoError(t, err)

	result, err := ConvertWithOptions(
		WithParsed(*parseResult),
		WithTargetVersion("3.0.3"),
	)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "2.0", result.SourceVersion)
}

// TestConvertWithOptions_StrictMode tests that strict mode is applied
func TestConvertWithOptions_StrictMode(t *testing.T) {
	result, err := ConvertWithOptions(
		WithFilePath("../testdata/minimal-oas2.yaml"),
		WithTargetVersion("3.0.3"),
		WithStrictMode(true),
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
	// Strict mode doesn't fail conversion, just affects issue reporting
}

// TestConvertWithOptions_DisableInfo tests that info messages can be disabled
func TestConvertWithOptions_DisableInfo(t *testing.T) {
	result, err := ConvertWithOptions(
		WithFilePath("../testdata/minimal-oas2.yaml"),
		WithTargetVersion("3.0.3"),
		WithIncludeInfo(false),
	)
	require.NoError(t, err)
	assert.True(t, result.Success)
	// Info messages should be filtered out
	assert.Equal(t, 0, result.InfoCount)
}

// TestConvertWithOptions_DefaultValues tests that default values are applied correctly
func TestConvertWithOptions_DefaultValues(t *testing.T) {
	result, err := ConvertWithOptions(
		WithFilePath("../testdata/minimal-oas2.yaml"),
		WithTargetVersion("3.0.3"),
		// Not specifying WithStrictMode or WithIncludeInfo to test defaults
	)
	require.NoError(t, err)
	assert.True(t, result.Success)
	// Default: IncludeInfo = true, so info messages may be present
}

// TestConvertWithOptions_NoInputSource tests error when no input source is specified
func TestConvertWithOptions_NoInputSource(t *testing.T) {
	_, err := ConvertWithOptions(
		WithTargetVersion("3.0.3"),
		WithStrictMode(false),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify an input source")
}

// TestConvertWithOptions_NoTargetVersion tests error when no target version is specified
func TestConvertWithOptions_NoTargetVersion(t *testing.T) {
	_, err := ConvertWithOptions(
		WithFilePath("../testdata/minimal-oas2.yaml"),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify a target version")
}

// TestConvertWithOptions_EmptyTargetVersion tests error when empty target version is provided
func TestConvertWithOptions_EmptyTargetVersion(t *testing.T) {
	_, err := ConvertWithOptions(
		WithFilePath("../testdata/minimal-oas2.yaml"),
		WithTargetVersion(""),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target version cannot be empty")
}

// TestConvertWithOptions_MultipleInputSources tests error when multiple input sources are specified
func TestConvertWithOptions_MultipleInputSources(t *testing.T) {
	parseResult, err := parser.ParseWithOptions(
		parser.WithFilePath("../testdata/minimal-oas2.yaml"),
		parser.WithValidateStructure(true),
	)
	require.NoError(t, err)

	_, err = ConvertWithOptions(
		WithFilePath("../testdata/minimal-oas2.yaml"),
		WithParsed(*parseResult),
		WithTargetVersion("3.0.3"),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must specify exactly one input source")
}

// TestConvertWithOptions_AllOptions tests using all options together
func TestConvertWithOptions_AllOptions(t *testing.T) {
	result, err := ConvertWithOptions(
		WithFilePath("../testdata/minimal-oas2.yaml"),
		WithTargetVersion("3.0.3"),
		WithStrictMode(true),
		WithIncludeInfo(false),
		WithUserAgent("test-converter/1.0"),
	)
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, 0, result.InfoCount)
}

// TestWithFilePath_Converter tests the WithFilePath option function
func TestWithFilePath_Converter(t *testing.T) {
	cfg := &convertConfig{}
	opt := WithFilePath("test.yaml")
	err := opt(cfg)

	require.NoError(t, err)
	require.NotNil(t, cfg.filePath)
	assert.Equal(t, "test.yaml", *cfg.filePath)
}

// TestWithParsed_Converter tests the WithParsed option function
func TestWithParsed_Converter(t *testing.T) {
	parseResult := parser.ParseResult{Version: "2.0"}
	cfg := &convertConfig{}
	opt := WithParsed(parseResult)
	err := opt(cfg)

	require.NoError(t, err)
	require.NotNil(t, cfg.parsed)
	assert.Equal(t, "2.0", cfg.parsed.Version)
}

// TestWithTargetVersion tests the WithTargetVersion option function
func TestWithTargetVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{"valid_3.0.3", "3.0.3", false},
		{"valid_2.0", "2.0", false},
		{"valid_3.1.0", "3.1.0", false},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &convertConfig{}
			opt := WithTargetVersion(tt.version)
			err := opt(cfg)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.version, cfg.targetVersion)
			}
		})
	}
}

// TestWithStrictMode_Converter tests the WithStrictMode option function
func TestWithStrictMode_Converter(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled", true},
		{"disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &convertConfig{}
			opt := WithStrictMode(tt.enabled)
			err := opt(cfg)

			require.NoError(t, err)
			assert.Equal(t, tt.enabled, cfg.strictMode)
		})
	}
}

// TestWithIncludeInfo tests the WithIncludeInfo option function
func TestWithIncludeInfo(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
	}{
		{"enabled", true},
		{"disabled", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &convertConfig{}
			opt := WithIncludeInfo(tt.enabled)
			err := opt(cfg)

			require.NoError(t, err)
			assert.Equal(t, tt.enabled, cfg.includeInfo)
		})
	}
}

// TestWithUserAgent_Converter tests the WithUserAgent option function
func TestWithUserAgent_Converter(t *testing.T) {
	cfg := &convertConfig{}
	opt := WithUserAgent("custom-agent/2.0")
	err := opt(cfg)

	require.NoError(t, err)
	assert.Equal(t, "custom-agent/2.0", cfg.userAgent)
}

// TestApplyOptions_Defaults_Converter tests that default values are set correctly
func TestApplyOptions_Defaults_Converter(t *testing.T) {
	cfg, err := applyOptions(
		WithFilePath("test.yaml"),
		WithTargetVersion("3.0.3"),
	)

	require.NoError(t, err)
	assert.False(t, cfg.strictMode, "default strictMode should be false")
	assert.True(t, cfg.includeInfo, "default includeInfo should be true")
	assert.Equal(t, "", cfg.userAgent, "default userAgent should be empty")
}

// TestApplyOptions_OverrideDefaults_Converter tests that options override defaults
func TestApplyOptions_OverrideDefaults_Converter(t *testing.T) {
	cfg, err := applyOptions(
		WithFilePath("test.yaml"),
		WithTargetVersion("3.0.3"),
		WithStrictMode(true),
		WithIncludeInfo(false),
		WithUserAgent("custom/1.0"),
	)

	require.NoError(t, err)
	assert.True(t, cfg.strictMode)
	assert.False(t, cfg.includeInfo)
	assert.Equal(t, "custom/1.0", cfg.userAgent)
}

// TestNullableDeprecationWarning tests that nullable: true generates warnings
// when converting from OAS 3.0.x to OAS 3.1.x
func TestNullableDeprecationWarning(t *testing.T) {
	c := New()

	// Create an OAS 3.0.3 document with nullable schemas
	oas3Doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"NullableString": {
					Type:     "string",
					Nullable: true,
				},
				"NullableObject": {
					Type:     "object",
					Nullable: true,
					Properties: map[string]*parser.Schema{
						"nestedNullable": {
							Type:     "integer",
							Nullable: true,
						},
					},
				},
			},
		},
		Paths: make(map[string]*parser.PathItem),
	}

	parseResult := parser.ParseResult{
		Document:   oas3Doc,
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Data:       make(map[string]any),
		SourcePath: "test.yaml",
	}

	result, err := c.ConvertParsed(parseResult, "3.1.0")
	require.NoError(t, err)

	// Count nullable warnings
	nullableWarnings := 0
	for _, issue := range result.Issues {
		if issue.Severity == SeverityWarning && strings.Contains(issue.Message, "nullable") {
			nullableWarnings++
		}
	}

	// Should have 3 warnings: NullableString, NullableObject, and nestedNullable
	assert.Equal(t, 3, nullableWarnings, "Expected 3 nullable deprecation warnings")
}

// TestNullableDeprecationNotTriggeredFor30To30 tests that nullable warnings
// are not generated when converting within 3.0.x versions
func TestNullableDeprecationNotTriggeredFor30To30(t *testing.T) {
	c := New()

	oas3Doc := &parser.OAS3Document{
		OpenAPI:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"NullableString": {
					Type:     "string",
					Nullable: true,
				},
			},
		},
		Paths: make(map[string]*parser.PathItem),
	}

	parseResult := parser.ParseResult{
		Document:   oas3Doc,
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Data:       make(map[string]any),
		SourcePath: "test.yaml",
	}

	result, err := c.ConvertParsed(parseResult, "3.0.3")
	require.NoError(t, err)

	// Should NOT have nullable warnings
	for _, issue := range result.Issues {
		if strings.Contains(issue.Message, "nullable") {
			t.Errorf("Unexpected nullable warning when converting 3.0.x to 3.0.x: %s", issue.Message)
		}
	}
}
