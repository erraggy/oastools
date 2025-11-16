package converter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"gopkg.in/yaml.v3"
)

// TestConvertConvenience tests the package-level Convert convenience function
func TestConvertConvenience(t *testing.T) {
	// Create a simple OAS 2.0 document
	oas2Doc := createSimpleOAS2Document()
	tmpFile := writeTemporaryYAML(t, oas2Doc)
	defer func() { _ = os.Remove(tmpFile) }()

	// Test conversion using convenience function
	result, err := Convert(tmpFile, "3.0.3")
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if result.TargetVersion != "3.0.3" {
		t.Errorf("Expected target version 3.0.3, got %s", result.TargetVersion)
	}

	if !result.Success {
		t.Errorf("Expected successful conversion, got Success=false")
	}

	// Verify document was converted
	doc, ok := result.Document.(*parser.OAS3Document)
	if !ok {
		t.Fatal("Expected OAS3Document")
	}

	if doc.OpenAPI != "3.0.3" {
		t.Errorf("Expected OpenAPI version 3.0.3, got %s", doc.OpenAPI)
	}
}

// TestConvertParsedConvenience tests the ConvertParsed convenience function
func TestConvertParsedConvenience(t *testing.T) {
	oas2Doc := createSimpleOAS2Document()
	parseResult := parser.ParseResult{
		Document:   oas2Doc,
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Data:       make(map[string]interface{}),
		SourcePath: "test.yaml",
		Errors:     []error{},
		Warnings:   []string{},
	}

	result, err := ConvertParsed(parseResult, "3.0.3")
	if err != nil {
		t.Fatalf("ConvertParsed failed: %v", err)
	}

	if result.TargetVersion != "3.0.3" {
		t.Errorf("Expected target version 3.0.3, got %s", result.TargetVersion)
	}

	if !result.Success {
		t.Errorf("Expected successful conversion")
	}
}

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
	oas2Doc := createSimpleOAS2Document()
	tmpFile := writeTemporaryYAML(t, oas2Doc)
	defer func() { _ = os.Remove(tmpFile) }()

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
	oas2Doc := createSimpleOAS2Document()
	parseResult := parser.ParseResult{
		Document:   oas2Doc,
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Data:       make(map[string]interface{}),
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
	oas2Doc := createDetailedOAS2Document()
	parseResult := parser.ParseResult{
		Document:   oas2Doc,
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Data:       make(map[string]interface{}),
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
	oas3Doc := createDetailedOAS3Document()
	parseResult := parser.ParseResult{
		Document:   oas3Doc,
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Data:       make(map[string]interface{}),
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
	if doc.Host == "" {
		t.Error("Expected host to be set")
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
	oas3Doc := createDetailedOAS3Document()
	oas3Doc.OpenAPI = "3.0.3"
	oas3Doc.OASVersion = parser.OASVersion303

	parseResult := parser.ParseResult{
		Document:   oas3Doc,
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Data:       make(map[string]interface{}),
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
	oas2Doc := createSimpleOAS2Document()
	parseResult := parser.ParseResult{
		Document:   oas2Doc,
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Data:       make(map[string]interface{}),
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
	oas2Doc := createSimpleOAS2Document()
	parseResult := parser.ParseResult{
		Document:   oas2Doc,
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Data:       make(map[string]interface{}),
		SourcePath: "test.yaml",
	}

	_, err := c.ConvertParsed(parseResult, "invalid.version")
	if err == nil {
		t.Fatal("Expected error for invalid target version")
	}
}

// TestStrictMode tests strict mode behavior
func TestStrictMode(t *testing.T) {
	c := New()
	c.StrictMode = true

	// Create an OAS 3.x document with webhooks that will cause critical issues when converting to 2.0
	oas3Doc := createDetailedOAS3Document()
	oas3Doc.Webhooks = map[string]*parser.PathItem{
		"newPet": {},
	}

	parseResult := parser.ParseResult{
		Document:   oas3Doc,
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Data:       make(map[string]interface{}),
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

	oas2Doc := createSimpleOAS2Document()
	parseResult := parser.ParseResult{
		Document:   oas2Doc,
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Data:       make(map[string]interface{}),
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

// Helper functions

func createSimpleOAS2Document() *parser.OAS2Document {
	return &parser.OAS2Document{
		Swagger:    "2.0",
		OASVersion: parser.OASVersion20,
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Host:     "api.example.com",
		BasePath: "/v1",
		Schemes:  []string{"https"},
		Paths:    make(map[string]*parser.PathItem),
	}
}

func createDetailedOAS2Document() *parser.OAS2Document {
	doc := createSimpleOAS2Document()
	doc.Definitions = map[string]*parser.Schema{
		"Pet": {
			Type: "object",
			Properties: map[string]*parser.Schema{
				"id":   {Type: "integer"},
				"name": {Type: "string"},
			},
		},
	}
	doc.Paths = map[string]*parser.PathItem{
		"/pets": {
			Get: &parser.Operation{
				Summary:     "List pets",
				OperationID: "listPets",
			},
		},
	}
	return doc
}

func createDetailedOAS3Document() *parser.OAS3Document {
	return &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info: &parser.Info{
			Title:   "Test API",
			Version: "1.0.0",
		},
		Servers: []*parser.Server{
			{
				URL:         "https://api.example.com/v1",
				Description: "Production server",
			},
		},
		Paths: map[string]*parser.PathItem{
			"/pets": {
				Get: &parser.Operation{
					Summary:     "List pets",
					OperationID: "listPets",
				},
			},
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"id":   {Type: "integer"},
						"name": {Type: "string"},
					},
				},
			},
		},
	}
}

func writeTemporaryYAML(t *testing.T, doc interface{}) string {
	t.Helper()

	data, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal document: %v", err)
	}

	tmpFile := filepath.Join(t.TempDir(), "test.yaml")
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		t.Fatalf("Failed to write temporary file: %v", err)
	}

	return tmpFile
}
