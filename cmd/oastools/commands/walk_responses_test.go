package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	return collector.All
}

func TestHandleWalkResponses_ListAll(t *testing.T) {
	all := collectTestResponses(t)
	require.Len(t, all, 6)

	// Verify all expected status codes are present
	statusCodes := make(map[string]bool)
	for _, info := range all {
		statusCodes[info.StatusCode] = true
	}
	expected := []string{"200", "500", "201", "400", "404"}
	for _, code := range expected {
		assert.True(t, statusCodes[code], "expected status code %s in results", code)
	}
}

func TestHandleWalkResponses_FilterByStatus200(t *testing.T) {
	all := collectTestResponses(t)
	filtered := filterResponsesByStatus(all, "200")

	require.Len(t, filtered, 2)
	for _, info := range filtered {
		assert.Equal(t, "200", info.StatusCode)
	}
}

func TestHandleWalkResponses_FilterByStatus4xx(t *testing.T) {
	all := collectTestResponses(t)
	filtered := filterResponsesByStatus(all, "4xx")

	require.Len(t, filtered, 2)
	for _, info := range filtered {
		assert.Equal(t, byte('4'), info.StatusCode[0], "expected 4xx status, got %s", info.StatusCode)
	}
}

func TestHandleWalkResponses_FilterByPath(t *testing.T) {
	all := collectTestResponses(t)
	filtered := filterResponsesByPath(all, "/pets/{id}")

	require.Len(t, filtered, 2)
	for _, info := range filtered {
		assert.Equal(t, "/pets/{id}", info.PathTemplate)
	}
}

func TestHandleWalkResponses_FilterByMethod(t *testing.T) {
	all := collectTestResponses(t)
	filtered := filterResponsesByMethod(all, "post")

	require.Len(t, filtered, 2)
	for _, info := range filtered {
		assert.Equal(t, "post", strings.ToLower(info.Method))
	}
}

func TestHandleWalkResponses_FilterByExtension(t *testing.T) {
	all := collectTestResponses(t)
	extFilter, err := ParseExtensionFilter("x-paginated=true")
	require.NoError(t, err)
	filtered := filterResponsesByExtension(all, extFilter)

	require.Len(t, filtered, 1)
	assert.Equal(t, "200", filtered[0].StatusCode)
	assert.Equal(t, "List of pets", filtered[0].Response.Description)
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
	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "DESCRIPTION")
	// Should contain data
	assert.Contains(t, output, "200")
	assert.Contains(t, output, "List of pets")
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
	assert.NotContains(t, output, "STATUS")
	// Data should still be present
	assert.Contains(t, output, "200")
}

func TestHandleWalkResponses_NoResults(t *testing.T) {
	all := collectTestResponses(t)
	filtered := filterResponsesByStatus(all, "999")

	require.Empty(t, filtered)
}

func TestHandleWalkResponses_MissingSpec(t *testing.T) {
	err := handleWalkResponses([]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing spec file")
}

func TestHandleWalkResponses_InvalidFormat(t *testing.T) {
	err := handleWalkResponses([]string{"--format", "invalid", "spec.yaml"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid format")
}

func TestHandleWalkResponses_InvalidExtensionFilter(t *testing.T) {
	// Write a minimal spec to a temp file so we get past the parse step
	tmpFile := writeTempSpec(t)

	err := handleWalkResponses([]string{"--extension", "bad-filter", tmpFile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "x-")
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

	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "description")
}

func TestHandleWalkResponses_DetailIncludesContext(t *testing.T) {
	all := collectTestResponses(t)
	filtered := filterResponsesByStatus(all, "200")
	filtered = filterResponsesByPath(filtered, "/pets")

	require.Len(t, filtered, 1)

	view := responseDetailView{
		StatusCode: filtered[0].StatusCode,
		Path:       filtered[0].PathTemplate,
		Method:     strings.ToUpper(filtered[0].Method),
		Response:   filtered[0].Response,
	}

	var buf bytes.Buffer
	err := RenderDetail(&buf, view, FormatJSON)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"statusCode"`)
	assert.Contains(t, output, `"path"`)
	assert.Contains(t, output, `"method"`)
	assert.Contains(t, output, "200")
	assert.Contains(t, output, "/pets")
	assert.Contains(t, output, "GET")
}

func TestHandleWalkResponses_DetailIncludesContextYAML(t *testing.T) {
	all := collectTestResponses(t)
	filtered := filterResponsesByStatus(all, "200")
	filtered = filterResponsesByPath(filtered, "/pets")

	require.Len(t, filtered, 1)

	view := responseDetailView{
		StatusCode: filtered[0].StatusCode,
		Path:       filtered[0].PathTemplate,
		Method:     strings.ToUpper(filtered[0].Method),
		Response:   filtered[0].Response,
	}

	var buf bytes.Buffer
	err := RenderDetail(&buf, view, FormatYAML)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "statusCode:")
	assert.Contains(t, output, "path:")
	assert.Contains(t, output, "method:")
	assert.Contains(t, output, "200")
	assert.Contains(t, output, "/pets")
	assert.Contains(t, output, "GET")
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
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"status"`)
	assert.Contains(t, output, `"path"`)
	assert.Contains(t, output, "200")
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
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "status")
	assert.Contains(t, output, "200")
}

func TestHandleWalkResponses_CombinedFilters(t *testing.T) {
	all := collectTestResponses(t)

	// Filter by status 200 AND path /pets
	filtered := filterResponsesByStatus(all, "200")
	filtered = filterResponsesByPath(filtered, "/pets")

	require.Len(t, filtered, 1)
	assert.Equal(t, "List of pets", filtered[0].Response.Description)
}

func TestHandleWalkResponses_MethodCaseInsensitive(t *testing.T) {
	all := collectTestResponses(t)

	// Filter by method "POST" (uppercase) should match "post" (lowercase)
	filtered := filterResponsesByMethod(all, "POST")
	require.Len(t, filtered, 2)

	// Filter by method "Get" (mixed case) should match "get"
	filtered = filterResponsesByMethod(all, "Get")
	require.Len(t, filtered, 4)
}

func TestHandleWalkResponses_PathGlob(t *testing.T) {
	all := collectTestResponses(t)

	// Glob /pets/* should match /pets/{id}
	filtered := filterResponsesByPath(all, "/pets/*")
	require.Len(t, filtered, 2)
	for _, info := range filtered {
		assert.Equal(t, "/pets/{id}", info.PathTemplate)
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

	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "STATUS")
	assert.Contains(t, output, "200")
	assert.Contains(t, output, "OK")
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

	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "404")
	assert.Contains(t, output, "Not found")
	// Should not contain the 200 response
	assert.False(t, strings.Contains(output, "\n200") || strings.HasPrefix(output, "200"), "expected 200 to be filtered out")
}

func TestHandleWalkResponses_Integration_NoResults(t *testing.T) {
	tmpFile := writeTempSpec(t)

	err := handleWalkResponses([]string{"--status", "999", tmpFile})
	require.NoError(t, err)
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

	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "GET")
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

	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "/pets")
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
	require.NoError(t, err)
	_, writeErr := tmpFile.WriteString(content)
	require.NoError(t, writeErr)
	_ = tmpFile.Close()

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = handleWalkResponses([]string{"--extension", "x-cached=true", tmpFile.Name()})

	_ = w.Close()
	os.Stdout = old

	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := buf.String()

	assert.Contains(t, output, "200")
	assert.NotContains(t, output, "500")
}

func TestHandleWalkResponses_ParseError(t *testing.T) {
	err := handleWalkResponses([]string{"/nonexistent/path/spec.yaml"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "walk responses")
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
	require.NoError(t, err)
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	_ = tmpFile.Close()
	return tmpFile.Name()
}
