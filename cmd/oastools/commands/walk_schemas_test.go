package commands

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires a spec file")
}

func TestHandleWalkSchemas_InvalidFormat(t *testing.T) {
	err := handleWalkSchemas([]string{"--format", "xml", "test.yaml"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid format")
}

func TestHandleWalkSchemas_ConflictingFlags(t *testing.T) {
	err := handleWalkSchemas([]string{"--component", "--inline", "test.yaml"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot use both")
}

func TestHandleWalkSchemas_ListAll(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	require.NoError(t, err)

	// Verify we get both component and inline schemas
	require.NotEmpty(t, collector.All)
	require.NotEmpty(t, collector.Components)
	require.NotEmpty(t, collector.Inline)
}

func TestHandleWalkSchemas_FilterByName(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	require.NoError(t, err)

	// Filter by name "Pet"
	var filtered []*walker.SchemaInfo
	for _, info := range collector.All {
		if strings.EqualFold(info.Name, "Pet") {
			filtered = append(filtered, info)
		}
	}

	require.NotEmpty(t, filtered)
	for _, info := range filtered {
		assert.True(t, strings.EqualFold(info.Name, "Pet"), "expected name Pet, got %s", info.Name)
	}
}

func TestHandleWalkSchemas_FilterByComponent(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	require.NoError(t, err)

	// All component schemas should be marked as component
	for _, info := range collector.Components {
		assert.True(t, info.IsComponent, "component schema %s not marked as component", info.Name)
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
		assert.False(t, found, "component schema %s found in inline list", info.Name)
	}
}

func TestHandleWalkSchemas_FilterByInline(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	require.NoError(t, err)

	// All inline schemas should NOT be marked as component
	for _, info := range collector.Inline {
		assert.False(t, info.IsComponent, "inline schema %s marked as component", info.JSONPath)
	}
}

func TestHandleWalkSchemas_FilterByType(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	require.NoError(t, err)

	// Filter for "array" type
	var filtered []*walker.SchemaInfo
	for _, info := range collector.All {
		if schemaTypeMatches(info.Schema.Type, "array") {
			filtered = append(filtered, info)
		}
	}

	require.NotEmpty(t, filtered)
	for _, info := range filtered {
		assert.True(t, schemaTypeMatches(info.Schema.Type, "array"), "filtered schema has type %v, expected array", info.Schema.Type)
	}
}

func TestHandleWalkSchemas_FilterByExtension(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	require.NoError(t, err)

	ef, err := ParseExtensionFilter("x-generated=true")
	require.NoError(t, err)

	var filtered []*walker.SchemaInfo
	for _, info := range collector.All {
		if ef.Match(info.Schema.Extra) {
			filtered = append(filtered, info)
		}
	}

	require.NotEmpty(t, filtered)
	for _, info := range filtered {
		assert.Equal(t, true, info.Schema.Extra["x-generated"])
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
			assert.Equal(t, tt.want, got, "schemaTypeMatches(%v, %q)", tt.schemaType, tt.filter)
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
		assert.Contains(t, output, h, "expected header %q in output", h)
	}

	// Verify data rows
	assert.Contains(t, output, "Pet")
	assert.Contains(t, output, "Error")
	assert.Contains(t, output, "x-generated=true")
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
	assert.NotContains(t, output, "NAME", "quiet mode should not include headers")
	// Data should use tab separation
	assert.Contains(t, output, "Pet\tobject", "expected tab-separated data in quiet mode")
}

func TestHandleWalkSchemas_DetailOutput(t *testing.T) {
	schema := &parser.Schema{
		Type:        "object",
		Description: "A pet",
	}

	var buf bytes.Buffer
	err := RenderDetail(&buf, schema, FormatText)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "object", "detail output should contain schema type")
	assert.Contains(t, output, "A pet", "detail output should contain schema description")
}

func TestHandleWalkSchemas_SummaryJSON(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	require.NoError(t, err)

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
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"name"`)
	assert.Contains(t, output, `"type"`)
}

func TestHandleWalkSchemas_SummaryYAML(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	require.NoError(t, err)

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
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "name")
	assert.Contains(t, output, "type")
}

func TestHandleWalkSchemas_DetailIncludesContext(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	require.NoError(t, err)

	// Find the Pet component schema
	var pet *walker.SchemaInfo
	for _, info := range collector.Components {
		if info.Name == "Pet" {
			pet = info
			break
		}
	}
	require.NotNil(t, pet, "expected to find Pet schema")

	view := schemaDetailView{
		Name:        pet.Name,
		JSONPath:    pet.JSONPath,
		IsComponent: pet.IsComponent,
		Schema:      pet.Schema,
	}

	var buf bytes.Buffer
	err = RenderDetail(&buf, view, FormatJSON)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"name"`)
	assert.Contains(t, output, "Pet")
	assert.Contains(t, output, `"isComponent"`)
	assert.Contains(t, output, `"jsonPath"`)
}

func TestHandleWalkSchemas_DetailIncludesContextYAML(t *testing.T) {
	result := testSchemaParseResult()
	collector, err := walker.CollectSchemas(result)
	require.NoError(t, err)

	var pet *walker.SchemaInfo
	for _, info := range collector.Components {
		if info.Name == "Pet" {
			pet = info
			break
		}
	}
	require.NotNil(t, pet, "expected to find Pet schema")

	view := schemaDetailView{
		Name:        pet.Name,
		JSONPath:    pet.JSONPath,
		IsComponent: pet.IsComponent,
		Schema:      pet.Schema,
	}

	var buf bytes.Buffer
	err = RenderDetail(&buf, view, FormatYAML)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "name:")
	assert.Contains(t, output, "Pet")
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
	_, err := buf.ReadFrom(r)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "No schemas matched")
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
	_, err := buf.ReadFrom(r)
	require.NoError(t, err)

	output := buf.String()
	assert.Empty(t, output, "quiet mode should produce no output")
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
	err := os.WriteFile(tmpFile, []byte(testSchemaSpecYAML), 0o644)
	require.NoError(t, err)
	return tmpFile
}

func captureSchemaStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	return buf.String()
}

func TestHandleWalkSchemas_Integration_ListAll(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	output := captureSchemaStdout(t, func() {
		err := handleWalkSchemas([]string{tmpFile})
		require.NoError(t, err)
	})

	// Should contain component schemas
	assert.Contains(t, output, "Pet")
	assert.Contains(t, output, "Error")
	// Should contain table headers
	assert.Contains(t, output, "NAME")
	assert.Contains(t, output, "TYPE")
}

func TestHandleWalkSchemas_Integration_FilterByName(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	output := captureSchemaStdout(t, func() {
		err := handleWalkSchemas([]string{"--name", "Pet", tmpFile})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Pet")
	// Error schema should not appear (different name)
	for line := range strings.SplitSeq(output, "\n") {
		trimmed := strings.TrimSpace(line)
		assert.False(t, strings.HasPrefix(trimmed, "Error"), "expected output to NOT contain 'Error' schema")
	}
}

func TestHandleWalkSchemas_Integration_FilterByComponent(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	output := captureSchemaStdout(t, func() {
		err := handleWalkSchemas([]string{"--component", tmpFile})
		require.NoError(t, err)
	})

	// All rows should show "component" in the LOCATION column
	for line := range strings.SplitSeq(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "NAME") {
			continue
		}
		assert.NotContains(t, trimmed, "inline", "--component flag should not include inline schemas, got line: %s", line)
	}
}

func TestHandleWalkSchemas_Integration_FilterByInline(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	output := captureSchemaStdout(t, func() {
		err := handleWalkSchemas([]string{"--inline", tmpFile})
		require.NoError(t, err)
	})

	// All rows should show "inline" in the LOCATION column
	for line := range strings.SplitSeq(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "NAME") {
			continue
		}
		assert.NotContains(t, trimmed, "component", "--inline flag should not include component schemas, got line: %s", line)
	}
}

func TestHandleWalkSchemas_Integration_FilterByType(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	output := captureSchemaStdout(t, func() {
		err := handleWalkSchemas([]string{"--type", "array", tmpFile})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "array")
	// Object schemas should not appear in data rows
	for line := range strings.SplitSeq(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "NAME") {
			continue
		}
		assert.NotContains(t, trimmed, "object", "--type array should not include object schemas, got line: %s", line)
	}
}

func TestHandleWalkSchemas_Integration_FilterByExtension(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	output := captureSchemaStdout(t, func() {
		err := handleWalkSchemas([]string{"--extension", "x-generated", tmpFile})
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Pet")
}

func TestHandleWalkSchemas_Integration_DetailMode(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	output := captureSchemaStdout(t, func() {
		err := handleWalkSchemas([]string{"--detail", "--name", "Pet", tmpFile})
		require.NoError(t, err)
	})

	// Detail mode should output YAML with schema fields
	assert.Contains(t, output, "type: object")
}

func TestHandleWalkSchemas_Integration_QuietMode(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	output := captureSchemaStdout(t, func() {
		err := handleWalkSchemas([]string{"-q", tmpFile})
		require.NoError(t, err)
	})

	// Quiet mode: no headers
	assert.NotContains(t, output, "NAME")
	// Should have tab-separated data
	assert.Contains(t, output, "\t")
}

func TestHandleWalkSchemas_Integration_NoResults(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	// Capture stderr
	oldStderr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	output := captureSchemaStdout(t, func() {
		err := handleWalkSchemas([]string{"--name", "Nonexistent", tmpFile})
		require.NoError(t, err)
	})

	_ = wErr.Close()
	os.Stderr = oldStderr

	var bufErr bytes.Buffer
	_, _ = bufErr.ReadFrom(rErr)
	stderrOutput := bufErr.String()

	assert.Empty(t, output, "expected no stdout output")
	assert.Contains(t, stderrOutput, "No schemas matched")
}

func TestHandleWalkSchemas_Integration_JSONFormat(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	output := captureSchemaStdout(t, func() {
		err := handleWalkSchemas([]string{"--detail", "--format", "json", "--name", "Pet", tmpFile})
		require.NoError(t, err)
	})

	// JSON output should contain opening brace
	assert.Contains(t, output, "{")
}

func TestHandleWalkSchemas_Integration_InvalidExtensionFilter(t *testing.T) {
	tmpFile := writeSchemaTestSpec(t)

	err := handleWalkSchemas([]string{"--extension", "not-x-prefixed", tmpFile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must start with")
}
