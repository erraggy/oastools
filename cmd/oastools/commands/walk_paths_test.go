package commands

import (
	"bytes"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testPathParseResult() *parser.ParseResult {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Summary: "Pet operations",
				Get:     &parser.Operation{Summary: "List pets"},
				Post:    &parser.Operation{Summary: "Create pet"},
				Extra:   map[string]any{"x-resource": "pets"},
			},
			"/pets/{id}": &parser.PathItem{
				Summary: "Single pet",
				Get:     &parser.Operation{Summary: "Get pet"},
				Delete:  &parser.Operation{Summary: "Delete pet"},
			},
			"/users": &parser.PathItem{
				Summary: "User operations",
				Get:     &parser.Operation{Summary: "List users"},
			},
		},
	}
	return &parser.ParseResult{Document: doc, Version: "3.0.3"}
}

func collectTestPaths(t *testing.T) []pathInfo {
	t.Helper()
	result := testPathParseResult()
	paths, err := collectPaths(result)
	require.NoError(t, err)
	return paths
}

func TestWalkPaths_ListAll(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "", "")
	require.NoError(t, err)

	assert.Len(t, matched, 3)
}

func TestWalkPaths_FilterByPathExact(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "/pets", "")
	require.NoError(t, err)

	require.Len(t, matched, 1)
	assert.Equal(t, "/pets", matched[0].pathTemplate)
}

func TestWalkPaths_FilterByPathGlob(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "/pets/*", "")
	require.NoError(t, err)

	require.Len(t, matched, 1)
	assert.Equal(t, "/pets/{id}", matched[0].pathTemplate)
}

func TestWalkPaths_FilterByExtension(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "", "x-resource")
	require.NoError(t, err)

	require.Len(t, matched, 1)
	assert.Equal(t, "/pets", matched[0].pathTemplate)
}

func TestWalkPaths_FilterByExtensionValue(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "", "x-resource=pets")
	require.NoError(t, err)

	require.Len(t, matched, 1)
}

func TestWalkPaths_FilterByExtensionInvalid(t *testing.T) {
	paths := collectTestPaths(t)

	_, err := filterPaths(paths, "", "invalid-key")
	assert.Error(t, err)
}

func TestWalkPaths_FilterByExtensionNoMatch(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "", "x-nonexistent")
	require.NoError(t, err)

	assert.Empty(t, matched)
}

func TestWalkPaths_FilterCombined(t *testing.T) {
	paths := collectTestPaths(t)

	// Path pattern + extension: /pets matches the pattern, and has x-resource
	matched, err := filterPaths(paths, "/pets", "x-resource")
	require.NoError(t, err)

	require.Len(t, matched, 1)
	assert.Equal(t, "/pets", matched[0].pathTemplate)
}

func TestWalkPaths_FilterCombinedNoMatch(t *testing.T) {
	paths := collectTestPaths(t)

	// /users matches path pattern but has no x-resource extension
	matched, err := filterPaths(paths, "/users", "x-resource")
	require.NoError(t, err)

	assert.Empty(t, matched)
}

func TestWalkPaths_FilterNonexistentPath(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "/nonexistent", "")
	require.NoError(t, err)

	assert.Empty(t, matched)
}

func TestPathMethods(t *testing.T) {
	tests := []struct {
		name string
		pi   *parser.PathItem
		want string
	}{
		{
			name: "GET and POST",
			pi: &parser.PathItem{
				Get:  &parser.Operation{},
				Post: &parser.Operation{},
			},
			want: "GET, POST",
		},
		{
			name: "all methods",
			pi: &parser.PathItem{
				Get:     &parser.Operation{},
				Put:     &parser.Operation{},
				Post:    &parser.Operation{},
				Delete:  &parser.Operation{},
				Options: &parser.Operation{},
				Head:    &parser.Operation{},
				Patch:   &parser.Operation{},
				Trace:   &parser.Operation{},
			},
			want: "GET, PUT, POST, DELETE, OPTIONS, HEAD, PATCH, TRACE",
		},
		{
			name: "no methods",
			pi:   &parser.PathItem{},
			want: "",
		},
		{
			name: "single method",
			pi:   &parser.PathItem{Delete: &parser.Operation{}},
			want: "DELETE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pathMethods(tt.pi)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWalkPaths_SummaryTableOutput(t *testing.T) {
	paths := collectTestPaths(t)

	// Filter to just /pets for predictable output
	matched, err := filterPaths(paths, "/pets", "")
	require.NoError(t, err)

	var buf bytes.Buffer
	headers := []string{"PATH", "METHODS", "SUMMARY", "EXTENSIONS"}
	rows := make([][]string, 0, len(matched))
	for _, p := range matched {
		rows = append(rows, []string{
			p.pathTemplate,
			pathMethods(p.pathItem),
			p.pathItem.Summary,
			FormatExtensions(p.pathItem.Extra),
		})
	}

	RenderSummaryTable(&buf, headers, rows, false)
	output := buf.String()

	assert.Contains(t, output, "PATH")
	assert.Contains(t, output, "/pets")
	assert.Contains(t, output, "GET, POST")
	assert.Contains(t, output, "Pet operations")
	assert.Contains(t, output, "x-resource")
}

func TestWalkPaths_SummaryTableQuiet(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "/users", "")
	require.NoError(t, err)

	var buf bytes.Buffer
	headers := []string{"PATH", "METHODS", "SUMMARY", "EXTENSIONS"}
	rows := make([][]string, 0, len(matched))
	for _, p := range matched {
		rows = append(rows, []string{
			p.pathTemplate,
			pathMethods(p.pathItem),
			p.pathItem.Summary,
			FormatExtensions(p.pathItem.Extra),
		})
	}

	RenderSummaryTable(&buf, headers, rows, true)
	output := buf.String()

	// Quiet: no headers
	assert.NotContains(t, output, "PATH")
	// Still has data
	assert.Contains(t, output, "/users")
}

func TestWalkPaths_SummaryJSON(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "/pets", "")
	require.NoError(t, err)

	headers := []string{"PATH", "METHODS", "SUMMARY", "EXTENSIONS"}
	rows := make([][]string, 0, len(matched))
	for _, p := range matched {
		rows = append(rows, []string{
			p.pathTemplate,
			pathMethods(p.pathItem),
			p.pathItem.Summary,
			FormatExtensions(p.pathItem.Extra),
		})
	}

	var buf bytes.Buffer
	err = RenderSummaryStructured(&buf, headers, rows, FormatJSON)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"path"`)
	assert.Contains(t, output, "/pets")
}

func TestWalkPaths_SummaryYAML(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "/users", "")
	require.NoError(t, err)

	headers := []string{"PATH", "METHODS", "SUMMARY", "EXTENSIONS"}
	rows := make([][]string, 0, len(matched))
	for _, p := range matched {
		rows = append(rows, []string{
			p.pathTemplate,
			pathMethods(p.pathItem),
			p.pathItem.Summary,
			FormatExtensions(p.pathItem.Extra),
		})
	}

	var buf bytes.Buffer
	err = RenderSummaryStructured(&buf, headers, rows, FormatYAML)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "path")
	assert.Contains(t, output, "/users")
}

func TestWalkPaths_DetailIncludesPath(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "/pets", "")
	require.NoError(t, err)

	require.Len(t, matched, 1)

	view := pathDetailView{
		Path:     matched[0].pathTemplate,
		PathItem: matched[0].pathItem,
	}

	var buf bytes.Buffer
	err = RenderDetail(&buf, view, FormatJSON)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"path"`)
	assert.Contains(t, output, "/pets")
	assert.Contains(t, output, "Pet operations")
	assert.Contains(t, output, "List pets")
}

func TestWalkPaths_DetailIncludesPathYAML(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "/users", "")
	require.NoError(t, err)

	require.Len(t, matched, 1)

	view := pathDetailView{
		Path:     matched[0].pathTemplate,
		PathItem: matched[0].pathItem,
	}

	var buf bytes.Buffer
	err = RenderDetail(&buf, view, FormatYAML)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "path:")
	assert.Contains(t, output, "/users")
	assert.Contains(t, output, "User operations")
	assert.Contains(t, output, "List users")
}

func TestWalkPaths_NoArgsError(t *testing.T) {
	err := handleWalkPaths([]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires a spec file")
}

func TestWalkPaths_InvalidFormat(t *testing.T) {
	err := handleWalkPaths([]string{"--format", "xml", "test.yaml"})
	assert.Error(t, err)
}

func TestWalkPaths_CollectPathsNilResult(t *testing.T) {
	_, err := collectPaths(nil)
	assert.Error(t, err)
}
