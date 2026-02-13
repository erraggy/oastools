package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
)

func testResponseParseResult() *parser.ParseResult {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "List of pets", Extra: map[string]any{"x-paginated": true}},
							"500": {Description: "Server error"},
						},
					},
				},
				Post: &parser.Operation{
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"201": {Description: "Pet created"},
							"400": {Description: "Bad request"},
						},
					},
				},
			},
			"/pets/{id}": &parser.PathItem{
				Get: &parser.Operation{
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "A pet"},
							"404": {Description: "Not found"},
						},
					},
				},
			},
		},
	}
	return &parser.ParseResult{Document: doc, Version: "3.0.3"}
}

func collectTestResponses(t *testing.T) []*walker.ResponseInfo {
	t.Helper()
	result := testResponseParseResult()
	collector, err := walker.CollectResponses(result)
	if err != nil {
		t.Fatalf("unexpected error collecting responses: %v", err)
	}
	return collector.All
}

func TestHandleWalkResponses_ListAll(t *testing.T) {
	all := collectTestResponses(t)
	if len(all) != 6 {
		t.Fatalf("expected 6 responses, got %d", len(all))
	}

	// Verify all expected status codes are present
	statusCodes := make(map[string]bool)
	for _, info := range all {
		statusCodes[info.StatusCode] = true
	}
	expected := []string{"200", "500", "201", "400", "404"}
	for _, code := range expected {
		if !statusCodes[code] {
			t.Errorf("expected status code %s in results", code)
		}
	}
}

func TestHandleWalkResponses_FilterByStatus200(t *testing.T) {
	all := collectTestResponses(t)
	filtered := filterResponsesByStatus(all, "200")

	if len(filtered) != 2 {
		t.Fatalf("expected 2 responses with status 200, got %d", len(filtered))
	}
	for _, info := range filtered {
		if info.StatusCode != "200" {
			t.Errorf("expected status 200, got %s", info.StatusCode)
		}
	}
}

func TestHandleWalkResponses_FilterByStatus4xx(t *testing.T) {
	all := collectTestResponses(t)
	filtered := filterResponsesByStatus(all, "4xx")

	if len(filtered) != 2 {
		t.Fatalf("expected 2 responses matching 4xx, got %d", len(filtered))
	}
	for _, info := range filtered {
		if info.StatusCode[0] != '4' {
			t.Errorf("expected 4xx status, got %s", info.StatusCode)
		}
	}
}

func TestHandleWalkResponses_FilterByPath(t *testing.T) {
	all := collectTestResponses(t)
	filtered := filterResponsesByPath(all, "/pets/{id}")

	if len(filtered) != 2 {
		t.Fatalf("expected 2 responses for /pets/{id}, got %d", len(filtered))
	}
	for _, info := range filtered {
		if info.PathTemplate != "/pets/{id}" {
			t.Errorf("expected path /pets/{id}, got %s", info.PathTemplate)
		}
	}
}

func TestHandleWalkResponses_FilterByMethod(t *testing.T) {
	all := collectTestResponses(t)
	filtered := filterResponsesByMethod(all, "post")

	if len(filtered) != 2 {
		t.Fatalf("expected 2 responses for POST, got %d", len(filtered))
	}
	for _, info := range filtered {
		if strings.ToLower(info.Method) != "post" {
			t.Errorf("expected method post, got %s", info.Method)
		}
	}
}

func TestHandleWalkResponses_FilterByExtension(t *testing.T) {
	all := collectTestResponses(t)
	extFilter, err := ParseExtensionFilter("x-paginated=true")
	if err != nil {
		t.Fatalf("unexpected error parsing extension filter: %v", err)
	}
	filtered := filterResponsesByExtension(all, extFilter)

	if len(filtered) != 1 {
		t.Fatalf("expected 1 response with x-paginated=true, got %d", len(filtered))
	}
	if filtered[0].StatusCode != "200" {
		t.Errorf("expected status 200, got %s", filtered[0].StatusCode)
	}
	if filtered[0].Response.Description != "List of pets" {
		t.Errorf("expected description 'List of pets', got %s", filtered[0].Response.Description)
	}
}

func TestHandleWalkResponses_SummaryTableOutput(t *testing.T) {
	all := collectTestResponses(t)

	// Build rows the same way as handleWalkResponses
	headers := []string{"STATUS", "DESCRIPTION", "PATH", "METHOD", "EXTENSIONS"}
	rows := make([][]string, 0, len(all))
	for _, info := range all {
		rows = append(rows, []string{
			info.StatusCode,
			info.Response.Description,
			info.PathTemplate,
			strings.ToUpper(info.Method),
			FormatExtensions(info.Response.Extra),
		})
	}

	var buf bytes.Buffer
	RenderSummaryTable(&buf, headers, rows, false)
	output := buf.String()

	// Should contain headers
	if !strings.Contains(output, "STATUS") {
		t.Error("expected STATUS header in output")
	}
	if !strings.Contains(output, "DESCRIPTION") {
		t.Error("expected DESCRIPTION header in output")
	}
	// Should contain data
	if !strings.Contains(output, "200") {
		t.Error("expected status 200 in output")
	}
	if !strings.Contains(output, "List of pets") {
		t.Error("expected 'List of pets' in output")
	}
}

func TestHandleWalkResponses_QuietOutput(t *testing.T) {
	all := collectTestResponses(t)

	headers := []string{"STATUS", "DESCRIPTION", "PATH", "METHOD", "EXTENSIONS"}
	rows := make([][]string, 0, len(all))
	for _, info := range all {
		rows = append(rows, []string{
			info.StatusCode,
			info.Response.Description,
			info.PathTemplate,
			strings.ToUpper(info.Method),
			FormatExtensions(info.Response.Extra),
		})
	}

	var buf bytes.Buffer
	RenderSummaryTable(&buf, headers, rows, true)
	output := buf.String()

	// Quiet mode: no header row
	if strings.Contains(output, "STATUS") {
		t.Error("quiet mode should not include STATUS header")
	}
	// Data should still be present
	if !strings.Contains(output, "200") {
		t.Error("expected status 200 in quiet output")
	}
}

func TestHandleWalkResponses_NoResults(t *testing.T) {
	all := collectTestResponses(t)
	filtered := filterResponsesByStatus(all, "999")

	if len(filtered) != 0 {
		t.Fatalf("expected 0 responses, got %d", len(filtered))
	}
}

func TestHandleWalkResponses_MissingSpec(t *testing.T) {
	err := handleWalkResponses([]string{})
	if err == nil {
		t.Error("expected error when no spec file provided")
	}
	if !strings.Contains(err.Error(), "missing spec file") {
		t.Errorf("expected 'missing spec file' error, got: %v", err)
	}
}

func TestHandleWalkResponses_InvalidFormat(t *testing.T) {
	err := handleWalkResponses([]string{"--format", "invalid", "spec.yaml"})
	if err == nil {
		t.Error("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("expected 'invalid format' error, got: %v", err)
	}
}

func TestHandleWalkResponses_InvalidExtensionFilter(t *testing.T) {
	// Write a minimal spec to a temp file so we get past the parse step
	tmpFile := writeTempSpec(t)

	err := handleWalkResponses([]string{"--extension", "bad-filter", tmpFile})
	if err == nil {
		t.Error("expected error for invalid extension filter")
	}
	if !strings.Contains(err.Error(), "x-") {
		t.Errorf("expected extension key validation error, got: %v", err)
	}
}

func TestHandleWalkResponses_DetailOutput(t *testing.T) {
	tmpFile := writeTempSpec(t)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := handleWalkResponses([]string{"--detail", tmpFile})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "description") {
		t.Error("expected 'description' in detail output")
	}
}

func TestHandleWalkResponses_DetailIncludesContext(t *testing.T) {
	all := collectTestResponses(t)
	filtered := filterResponsesByStatus(all, "200")
	filtered = filterResponsesByPath(filtered, "/pets")

	if len(filtered) != 1 {
		t.Fatalf("expected 1 response, got %d", len(filtered))
	}

	view := responseDetailView{
		StatusCode: filtered[0].StatusCode,
		Path:       filtered[0].PathTemplate,
		Method:     strings.ToUpper(filtered[0].Method),
		Response:   filtered[0].Response,
	}

	var buf bytes.Buffer
	err := RenderDetail(&buf, view, FormatJSON)
	if err != nil {
		t.Fatalf("RenderDetail failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"statusCode"`) {
		t.Error("expected 'statusCode' key in detail JSON output")
	}
	if !strings.Contains(output, `"path"`) {
		t.Error("expected 'path' key in detail JSON output")
	}
	if !strings.Contains(output, `"method"`) {
		t.Error("expected 'method' key in detail JSON output")
	}
	if !strings.Contains(output, "200") {
		t.Error("expected status 200 in detail output")
	}
	if !strings.Contains(output, "/pets") {
		t.Error("expected /pets in detail output")
	}
	if !strings.Contains(output, "GET") {
		t.Error("expected GET in detail output")
	}
}

func TestHandleWalkResponses_DetailIncludesContextYAML(t *testing.T) {
	all := collectTestResponses(t)
	filtered := filterResponsesByStatus(all, "200")
	filtered = filterResponsesByPath(filtered, "/pets")

	if len(filtered) != 1 {
		t.Fatalf("expected 1 response, got %d", len(filtered))
	}

	view := responseDetailView{
		StatusCode: filtered[0].StatusCode,
		Path:       filtered[0].PathTemplate,
		Method:     strings.ToUpper(filtered[0].Method),
		Response:   filtered[0].Response,
	}

	var buf bytes.Buffer
	err := RenderDetail(&buf, view, FormatYAML)
	if err != nil {
		t.Fatalf("RenderDetail failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "statusCode:") {
		t.Error("expected 'statusCode' key in YAML detail output")
	}
	if !strings.Contains(output, "path:") {
		t.Error("expected 'path' key in YAML detail output")
	}
	if !strings.Contains(output, "method:") {
		t.Error("expected 'method' key in YAML detail output")
	}
	if !strings.Contains(output, "200") {
		t.Error("expected status 200 value in YAML detail output")
	}
	if !strings.Contains(output, "/pets") {
		t.Error("expected /pets path value in YAML detail output")
	}
	if !strings.Contains(output, "GET") {
		t.Error("expected GET method value in YAML detail output")
	}
}

func TestHandleWalkResponses_SummaryJSON(t *testing.T) {
	all := collectTestResponses(t)

	headers := []string{"STATUS", "DESCRIPTION", "PATH", "METHOD", "EXTENSIONS"}
	rows := make([][]string, 0, len(all))
	for _, info := range all {
		rows = append(rows, []string{
			info.StatusCode,
			info.Response.Description,
			info.PathTemplate,
			strings.ToUpper(info.Method),
			FormatExtensions(info.Response.Extra),
		})
	}

	var buf bytes.Buffer
	err := RenderSummaryStructured(&buf, headers, rows, FormatJSON)
	if err != nil {
		t.Fatalf("RenderSummaryStructured failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"status"`) {
		t.Error("expected 'status' key in JSON summary output")
	}
	if !strings.Contains(output, `"path"`) {
		t.Error("expected 'path' key in JSON summary output")
	}
	if !strings.Contains(output, "200") {
		t.Error("expected 200 in JSON summary output")
	}
}

func TestHandleWalkResponses_SummaryYAML(t *testing.T) {
	all := collectTestResponses(t)

	headers := []string{"STATUS", "DESCRIPTION", "PATH", "METHOD", "EXTENSIONS"}
	rows := make([][]string, 0, len(all))
	for _, info := range all {
		rows = append(rows, []string{
			info.StatusCode,
			info.Response.Description,
			info.PathTemplate,
			strings.ToUpper(info.Method),
			FormatExtensions(info.Response.Extra),
		})
	}

	var buf bytes.Buffer
	err := RenderSummaryStructured(&buf, headers, rows, FormatYAML)
	if err != nil {
		t.Fatalf("RenderSummaryStructured failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "status") {
		t.Error("expected 'status' key in YAML summary output")
	}
	if !strings.Contains(output, "200") {
		t.Error("expected 200 in YAML summary output")
	}
}

func TestHandleWalkResponses_CombinedFilters(t *testing.T) {
	all := collectTestResponses(t)

	// Filter by status 200 AND path /pets
	filtered := filterResponsesByStatus(all, "200")
	filtered = filterResponsesByPath(filtered, "/pets")

	if len(filtered) != 1 {
		t.Fatalf("expected 1 response for status 200 on /pets, got %d", len(filtered))
	}
	if filtered[0].Response.Description != "List of pets" {
		t.Errorf("expected 'List of pets', got %s", filtered[0].Response.Description)
	}
}

func TestHandleWalkResponses_MethodCaseInsensitive(t *testing.T) {
	all := collectTestResponses(t)

	// Filter by method "POST" (uppercase) should match "post" (lowercase)
	filtered := filterResponsesByMethod(all, "POST")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 responses for POST (case insensitive), got %d", len(filtered))
	}

	// Filter by method "Get" (mixed case) should match "get"
	filtered = filterResponsesByMethod(all, "Get")
	if len(filtered) != 4 {
		t.Fatalf("expected 4 responses for Get (case insensitive), got %d", len(filtered))
	}
}

func TestHandleWalkResponses_PathGlob(t *testing.T) {
	all := collectTestResponses(t)

	// Glob /pets/* should match /pets/{id}
	filtered := filterResponsesByPath(all, "/pets/*")
	if len(filtered) != 2 {
		t.Fatalf("expected 2 responses for /pets/*, got %d", len(filtered))
	}
	for _, info := range filtered {
		if info.PathTemplate != "/pets/{id}" {
			t.Errorf("expected path /pets/{id}, got %s", info.PathTemplate)
		}
	}
}

func TestHandleWalkResponses_Integration_SummaryTable(t *testing.T) {
	tmpFile := writeTempSpec(t)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := handleWalkResponses([]string{tmpFile})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "STATUS") {
		t.Error("expected STATUS header in summary table output")
	}
	if !strings.Contains(output, "200") {
		t.Error("expected status 200 in summary output")
	}
	if !strings.Contains(output, "OK") {
		t.Error("expected description 'OK' in summary output")
	}
}

func TestHandleWalkResponses_Integration_StatusFilter(t *testing.T) {
	tmpFile := writeTempSpec(t)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := handleWalkResponses([]string{"--status", "404", tmpFile})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "404") {
		t.Error("expected status 404 in filtered output")
	}
	if !strings.Contains(output, "Not found") {
		t.Error("expected description 'Not found' in filtered output")
	}
	// Should not contain the 200 response
	if strings.Contains(output, "\n200") || strings.HasPrefix(output, "200") {
		t.Error("expected 200 to be filtered out")
	}
}

func TestHandleWalkResponses_Integration_NoResults(t *testing.T) {
	tmpFile := writeTempSpec(t)

	err := handleWalkResponses([]string{"--status", "999", tmpFile})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No error -- just prints "No responses matched" to stderr
}

func TestHandleWalkResponses_Integration_MethodFilter(t *testing.T) {
	tmpFile := writeTempSpec(t)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := handleWalkResponses([]string{"--method", "get", tmpFile})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "GET") {
		t.Error("expected GET in method-filtered output")
	}
}

func TestHandleWalkResponses_Integration_PathFilter(t *testing.T) {
	tmpFile := writeTempSpec(t)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := handleWalkResponses([]string{"--path", "/pets", tmpFile})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "/pets") {
		t.Error("expected /pets in path-filtered output")
	}
}

func TestHandleWalkResponses_Integration_ExtensionFilter(t *testing.T) {
	// Write a spec with extensions
	content := `openapi: "3.0.3"
info:
  title: Test
  version: "1.0"
paths:
  /pets:
    get:
      responses:
        "200":
          description: OK
          x-cached: true
        "500":
          description: Error
`
	tmpFile, err := os.CreateTemp(t.TempDir(), "spec-ext-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, writeErr := tmpFile.WriteString(content); writeErr != nil {
		t.Fatalf("failed to write temp file: %v", writeErr)
	}
	_ = tmpFile.Close()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = handleWalkResponses([]string{"--extension", "x-cached=true", tmpFile.Name()})

	_ = w.Close()
	os.Stdout = old

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "200") {
		t.Error("expected status 200 in extension-filtered output")
	}
	if strings.Contains(output, "500") {
		t.Error("expected 500 to be filtered out")
	}
}

func TestHandleWalkResponses_ParseError(t *testing.T) {
	err := handleWalkResponses([]string{"/nonexistent/path/spec.yaml"})
	if err == nil {
		t.Error("expected error for nonexistent spec file")
	}
	if !strings.Contains(err.Error(), "walk responses") {
		t.Errorf("expected 'walk responses' prefix in error, got: %v", err)
	}
}

// writeTempSpec writes a minimal OAS 3.0 spec to a temp file and returns the path.
func writeTempSpec(t *testing.T) string {
	t.Helper()
	content := `openapi: "3.0.3"
info:
  title: Test
  version: "1.0"
paths:
  /pets:
    get:
      responses:
        "200":
          description: OK
        "404":
          description: Not found
`
	tmpFile, err := os.CreateTemp(t.TempDir(), "spec-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	_ = tmpFile.Close()
	return tmpFile.Name()
}
