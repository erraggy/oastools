package commands

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
)

func testSchemaParseResult() *parser.ParseResult {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"id":   {Type: "integer"},
						"name": {Type: "string"},
					},
					Extra: map[string]any{"x-generated": true},
				},
				"Error": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"code":    {Type: "integer"},
						"message": {Type: "string"},
					},
				},
			},
		},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Type: "array"},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return &parser.ParseResult{Document: doc, Version: "3.0.3", OASVersion: parser.OASVersion303}
}

func TestHandleWalkSchemas_NoArgs(t *testing.T) {
	err := handleWalkSchemas([]string{})
	if err == nil {
		t.Fatal("expected error when no spec file provided")
	}
	if !strings.Contains(err.Error(), "requires a spec file") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestHandleWalkSchemas_InvalidFormat(t *testing.T) {
	err := handleWalkSchemas([]string{"--format", "xml", "test.yaml"})
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestHandleWalkSchemas_ConflictingFlags(t *testing.T) {
	err := handleWalkSchemas([]string{"--component", "--inline", "test.yaml"})
	if err == nil {
		t.Fatal("expected error for conflicting --component and --inline")
	}
	if !strings.Contains(err.Error(), "cannot use both") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestHandleWalkSchemas_ListAll(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	if err != nil {
		t.Fatalf("unexpected error collecting schemas: %v", err)
	}

	// Verify we get both component and inline schemas
	if len(collector.All) == 0 {
		t.Fatal("expected schemas in test document")
	}
	if len(collector.Components) == 0 {
		t.Fatal("expected component schemas")
	}
	if len(collector.Inline) == 0 {
		t.Fatal("expected inline schemas")
	}
}

func TestHandleWalkSchemas_FilterByName(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Filter by name "Pet"
	var filtered []*walker.SchemaInfo
	for _, info := range collector.All {
		if strings.EqualFold(info.Name, "Pet") {
			filtered = append(filtered, info)
		}
	}

	if len(filtered) == 0 {
		t.Fatal("expected to find schema named Pet")
	}
	for _, info := range filtered {
		if !strings.EqualFold(info.Name, "Pet") {
			t.Errorf("expected name Pet, got %s", info.Name)
		}
	}
}

func TestHandleWalkSchemas_FilterByComponent(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All component schemas should be marked as component
	for _, info := range collector.Components {
		if !info.IsComponent {
			t.Errorf("component schema %s not marked as component", info.Name)
		}
	}

	// Component list should not include inline schemas
	for _, info := range collector.Components {
		found := false
		for _, inlineInfo := range collector.Inline {
			if info.JSONPath == inlineInfo.JSONPath {
				found = true
				break
			}
		}
		if found {
			t.Errorf("component schema %s found in inline list", info.Name)
		}
	}
}

func TestHandleWalkSchemas_FilterByInline(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All inline schemas should NOT be marked as component
	for _, info := range collector.Inline {
		if info.IsComponent {
			t.Errorf("inline schema %s marked as component", info.JSONPath)
		}
	}
}

func TestHandleWalkSchemas_FilterByType(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Filter for "array" type
	var filtered []*walker.SchemaInfo
	for _, info := range collector.All {
		if schemaTypeMatches(info.Schema.Type, "array") {
			filtered = append(filtered, info)
		}
	}

	if len(filtered) == 0 {
		t.Fatal("expected at least one array-type schema")
	}
	for _, info := range filtered {
		if !schemaTypeMatches(info.Schema.Type, "array") {
			t.Errorf("filtered schema has type %v, expected array", info.Schema.Type)
		}
	}
}

func TestHandleWalkSchemas_FilterByExtension(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ef, err := ParseExtensionFilter("x-generated=true")
	if err != nil {
		t.Fatalf("unexpected error parsing extension filter: %v", err)
	}

	var filtered []*walker.SchemaInfo
	for _, info := range collector.All {
		if ef.Match(info.Schema.Extra) {
			filtered = append(filtered, info)
		}
	}

	if len(filtered) == 0 {
		t.Fatal("expected at least one schema with x-generated=true")
	}
	for _, info := range filtered {
		if info.Schema.Extra["x-generated"] != true {
			t.Errorf("filtered schema missing x-generated=true extension")
		}
	}
}

func TestSchemaTypeMatches(t *testing.T) {
	tests := []struct {
		name       string
		schemaType any
		filter     string
		want       bool
	}{
		{name: "string match", schemaType: "object", filter: "object", want: true},
		{name: "string case insensitive", schemaType: "Object", filter: "object", want: true},
		{name: "string mismatch", schemaType: "array", filter: "object", want: false},
		{name: "string slice match", schemaType: []string{"string", "null"}, filter: "string", want: true},
		{name: "string slice null match", schemaType: []string{"string", "null"}, filter: "null", want: true},
		{name: "string slice mismatch", schemaType: []string{"string", "null"}, filter: "object", want: false},
		{name: "any slice match", schemaType: []any{"string", "null"}, filter: "string", want: true},
		{name: "any slice null match", schemaType: []any{"string", "null"}, filter: "null", want: true},
		{name: "any slice mismatch", schemaType: []any{"string", "null"}, filter: "object", want: false},
		{name: "any slice non-string element", schemaType: []any{42, "string"}, filter: "string", want: true},
		{name: "any slice non-string only", schemaType: []any{42}, filter: "string", want: false},
		{name: "nil type", schemaType: nil, filter: "object", want: false},
		{name: "unexpected type", schemaType: 42, filter: "object", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := schemaTypeMatches(tt.schemaType, tt.filter)
			if got != tt.want {
				t.Errorf("schemaTypeMatches(%v, %q) = %v, want %v", tt.schemaType, tt.filter, got, tt.want)
			}
		})
	}
}

func TestHandleWalkSchemas_SummaryTableOutput(t *testing.T) {
	// Test that the summary table rendering produces expected columns
	headers := []string{"NAME", "TYPE", "PROPERTIES", "LOCATION", "EXTENSIONS"}
	rows := [][]string{
		{"Pet", "object", "2 props", "component", "x-generated=true"},
		{"Error", "object", "2 props", "component", ""},
	}

	var buf bytes.Buffer
	RenderSummaryTable(&buf, headers, rows, false)
	output := buf.String()

	// Verify headers are present
	for _, h := range headers {
		if !strings.Contains(output, h) {
			t.Errorf("expected header %q in output", h)
		}
	}

	// Verify data rows
	if !strings.Contains(output, "Pet") {
		t.Error("expected Pet in output")
	}
	if !strings.Contains(output, "Error") {
		t.Error("expected Error in output")
	}
	if !strings.Contains(output, "x-generated=true") {
		t.Error("expected extension in output")
	}
}

func TestHandleWalkSchemas_QuietOutput(t *testing.T) {
	headers := []string{"NAME", "TYPE", "PROPERTIES", "LOCATION", "EXTENSIONS"}
	rows := [][]string{
		{"Pet", "object", "2 props", "component", ""},
	}

	var buf bytes.Buffer
	RenderSummaryTable(&buf, headers, rows, true)
	output := buf.String()

	// In quiet mode, headers should not be present
	if strings.Contains(output, "NAME") {
		t.Error("quiet mode should not include headers")
	}
	// Data should use tab separation
	if !strings.Contains(output, "Pet\tobject") {
		t.Errorf("expected tab-separated data in quiet mode, got: %s", output)
	}
}

func TestHandleWalkSchemas_DetailOutput(t *testing.T) {
	schema := &parser.Schema{
		Type:        "object",
		Description: "A pet",
	}

	var buf bytes.Buffer
	err := RenderDetail(&buf, schema, FormatText, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "object") {
		t.Error("detail output should contain schema type")
	}
	if !strings.Contains(output, "A pet") {
		t.Error("detail output should contain schema description")
	}
}

func TestHandleWalkSchemas_SummaryJSON(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	headers := []string{"NAME", "TYPE", "PROPERTIES", "LOCATION", "EXTENSIONS"}
	rows := make([][]string, 0, len(collector.Components))
	for _, info := range collector.Components {
		rows = append(rows, []string{
			info.Name,
			fmt.Sprintf("%v", info.Schema.Type),
			fmt.Sprintf("%d props", len(info.Schema.Properties)),
			"component",
			FormatExtensions(info.Schema.Extra),
		})
	}

	var buf bytes.Buffer
	err = RenderSummaryStructured(&buf, headers, rows, FormatJSON)
	if err != nil {
		t.Fatalf("RenderSummaryStructured failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"name"`) {
		t.Error("expected 'name' key in JSON summary output")
	}
	if !strings.Contains(output, `"type"`) {
		t.Error("expected 'type' key in JSON summary output")
	}
}

func TestHandleWalkSchemas_SummaryYAML(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	headers := []string{"NAME", "TYPE", "PROPERTIES", "LOCATION", "EXTENSIONS"}
	rows := make([][]string, 0, len(collector.Components))
	for _, info := range collector.Components {
		rows = append(rows, []string{
			info.Name,
			fmt.Sprintf("%v", info.Schema.Type),
			fmt.Sprintf("%d props", len(info.Schema.Properties)),
			"component",
			FormatExtensions(info.Schema.Extra),
		})
	}

	var buf bytes.Buffer
	err = RenderSummaryStructured(&buf, headers, rows, FormatYAML)
	if err != nil {
		t.Fatalf("RenderSummaryStructured failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "name") {
		t.Error("expected 'name' key in YAML summary output")
	}
}

func TestHandleWalkSchemas_DetailIncludesContext(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find the Pet component schema
	var pet *walker.SchemaInfo
	for _, info := range collector.Components {
		if info.Name == "Pet" {
			pet = info
			break
		}
	}
	if pet == nil {
		t.Fatal("expected to find Pet schema")
	}

	view := schemaDetailView{
		Name:        pet.Name,
		JSONPath:    pet.JSONPath,
		IsComponent: pet.IsComponent,
		Schema:      pet.Schema,
	}

	var buf bytes.Buffer
	err = RenderDetail(&buf, view, FormatJSON, false)
	if err != nil {
		t.Fatalf("RenderDetail failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"name"`) {
		t.Error("expected 'name' key in detail JSON output")
	}
	if !strings.Contains(output, "Pet") {
		t.Error("expected Pet name in detail output")
	}
	if !strings.Contains(output, `"isComponent"`) {
		t.Error("expected 'isComponent' key in detail output")
	}
	if !strings.Contains(output, `"jsonPath"`) {
		t.Error("expected 'jsonPath' key in detail output")
	}
}

func TestHandleWalkSchemas_DetailIncludesContextYAML(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var pet *walker.SchemaInfo
	for _, info := range collector.Components {
		if info.Name == "Pet" {
			pet = info
			break
		}
	}
	if pet == nil {
		t.Fatal("expected to find Pet schema")
	}

	view := schemaDetailView{
		Name:        pet.Name,
		JSONPath:    pet.JSONPath,
		IsComponent: pet.IsComponent,
		Schema:      pet.Schema,
	}

	var buf bytes.Buffer
	err = RenderDetail(&buf, view, FormatYAML, false)
	if err != nil {
		t.Fatalf("RenderDetail failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "name:") {
		t.Error("expected 'name' key in YAML detail output")
	}
	if !strings.Contains(output, "Pet") {
		t.Error("expected Pet name in YAML detail output")
	}
}

func TestHandleWalkSchemas_NoResults(t *testing.T) {
	// Capture stderr to verify no-results message
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	renderNoResults("schemas", false)

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("failed to read from pipe: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No schemas matched") {
		t.Errorf("expected no-results message, got: %s", output)
	}
}

func TestHandleWalkSchemas_NoResultsQuiet(t *testing.T) {
	// In quiet mode, no message should be output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	renderNoResults("schemas", true)

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("failed to read from pipe: %v", err)
	}

	output := buf.String()
	if output != "" {
		t.Errorf("quiet mode should produce no output, got: %s", output)
	}
}

// testSchemaSpecYAML is a hand-crafted OAS 3.0.3 spec used in schema integration tests.
const testSchemaSpecYAML = `openapi: "3.0.3"
info:
  title: Test
  version: "1.0"
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
      x-generated: true
    Error:
      type: object
      properties:
        code:
          type: integer
        message:
          type: string
paths:
  /pets:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: array
`

func writeSchemaTestSpec(t *testing.T) string {
	t.Helper()
	tmpFile := t.TempDir() + "/test-spec.yaml"
	if err := os.WriteFile(tmpFile, []byte(testSchemaSpecYAML), 0o644); err != nil {
		t.Fatalf("failed to write test spec: %v", err)
	}
	return tmpFile
}

func captureSchemaStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("failed to read from pipe: %v", err)
	}
	return buf.String()
}

func TestHandleWalkSchemas_Integration_ListAll(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	output := captureSchemaStdout(t, func() {
		if err := handleWalkSchemas([]string{tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Should contain component schemas
	if !strings.Contains(output, "Pet") {
		t.Error("expected output to contain 'Pet'")
	}
	if !strings.Contains(output, "Error") {
		t.Error("expected output to contain 'Error'")
	}
	// Should contain table headers
	if !strings.Contains(output, "NAME") {
		t.Error("expected output to contain 'NAME' header")
	}
	if !strings.Contains(output, "TYPE") {
		t.Error("expected output to contain 'TYPE' header")
	}
}

func TestHandleWalkSchemas_Integration_FilterByName(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	output := captureSchemaStdout(t, func() {
		if err := handleWalkSchemas([]string{"--name", "Pet", tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "Pet") {
		t.Error("expected output to contain 'Pet'")
	}
	// Error schema should not appear (different name)
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "Error") {
			t.Error("expected output to NOT contain 'Error' schema")
		}
	}
}

func TestHandleWalkSchemas_Integration_FilterByComponent(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	output := captureSchemaStdout(t, func() {
		if err := handleWalkSchemas([]string{"--component", tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// All rows should show "component" in the LOCATION column
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "NAME") {
			continue
		}
		if strings.Contains(trimmed, "inline") {
			t.Errorf("--component flag should not include inline schemas, got line: %s", line)
		}
	}
}

func TestHandleWalkSchemas_Integration_FilterByInline(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	output := captureSchemaStdout(t, func() {
		if err := handleWalkSchemas([]string{"--inline", tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// All rows should show "inline" in the LOCATION column
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "NAME") {
			continue
		}
		if strings.Contains(trimmed, "component") {
			t.Errorf("--inline flag should not include component schemas, got line: %s", line)
		}
	}
}

func TestHandleWalkSchemas_Integration_FilterByType(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	output := captureSchemaStdout(t, func() {
		if err := handleWalkSchemas([]string{"--type", "array", tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "array") {
		t.Error("expected output to contain 'array' type")
	}
	// Object schemas should not appear in data rows
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "NAME") {
			continue
		}
		if strings.Contains(trimmed, "object") {
			t.Errorf("--type array should not include object schemas, got line: %s", line)
		}
	}
}

func TestHandleWalkSchemas_Integration_FilterByExtension(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	output := captureSchemaStdout(t, func() {
		if err := handleWalkSchemas([]string{"--extension", "x-generated", tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "Pet") {
		t.Error("expected output to contain 'Pet' (has x-generated)")
	}
}

func TestHandleWalkSchemas_Integration_DetailMode(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	output := captureSchemaStdout(t, func() {
		if err := handleWalkSchemas([]string{"--detail", "--name", "Pet", tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Detail mode should output YAML with schema fields
	if !strings.Contains(output, "type: object") {
		t.Errorf("expected detail output to contain 'type: object', got: %s", output)
	}
}

func TestHandleWalkSchemas_Integration_QuietMode(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	output := captureSchemaStdout(t, func() {
		if err := handleWalkSchemas([]string{"-q", tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Quiet mode: no headers
	if strings.Contains(output, "NAME") {
		t.Error("expected quiet mode to NOT contain headers")
	}
	// Should have tab-separated data
	if !strings.Contains(output, "\t") {
		t.Error("expected tab-separated output in quiet mode")
	}
}

func TestHandleWalkSchemas_Integration_NoResults(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	// Capture stderr
	oldStderr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	output := captureSchemaStdout(t, func() {
		if err := handleWalkSchemas([]string{"--name", "Nonexistent", tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	_ = wErr.Close()
	os.Stderr = oldStderr

	var bufErr bytes.Buffer
	_, _ = bufErr.ReadFrom(rErr)
	stderrOutput := bufErr.String()

	if output != "" {
		t.Errorf("expected no stdout output, got: %s", output)
	}
	if !strings.Contains(stderrOutput, "No schemas matched") {
		t.Errorf("expected stderr message about no results, got: %s", stderrOutput)
	}
}

func TestHandleWalkSchemas_Integration_JSONFormat(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	output := captureSchemaStdout(t, func() {
		if err := handleWalkSchemas([]string{"--detail", "--format", "json", "--name", "Pet", tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// JSON output should contain opening brace
	if !strings.Contains(output, "{") {
		t.Errorf("expected JSON output, got: %s", output)
	}
}

func TestHandleWalkSchemas_Integration_InvalidExtensionFilter(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	err := handleWalkSchemas([]string{"--extension", "not-x-prefixed", tmpFile})
	if err == nil {
		t.Fatal("expected error for invalid extension filter")
	}
	if !strings.Contains(err.Error(), "must start with") {
		t.Errorf("unexpected error: %v", err)
	}
}
