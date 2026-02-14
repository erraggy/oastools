package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testOperationsParseResult() *parser.ParseResult {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					Summary:     "List pets",
					OperationID: "listPets",
					Tags:        []string{"pets"},
					Extra:       map[string]any{"x-internal": true},
				},
				Post: &parser.Operation{
					Summary:     "Create pet",
					OperationID: "createPet",
					Tags:        []string{"pets"},
				},
			},
			"/pets/{id}": &parser.PathItem{
				Get: &parser.Operation{
					Summary:     "Get pet",
					OperationID: "getPet",
					Tags:        []string{"pets"},
				},
				Delete: &parser.Operation{
					Summary:     "Delete pet",
					OperationID: "deletePet",
					Tags:        []string{"pets", "admin"},
					Deprecated:  true,
				},
			},
		},
	}
	return &parser.ParseResult{Document: doc, Version: "3.0.3"}
}

func collectTestOperations(t *testing.T) []*walker.OperationInfo {
	t.Helper()
	result := testOperationsParseResult()
	collector, err := walker.CollectOperations(result)
	require.NoError(t, err)
	return collector.All
}

func TestWalkOperations_ListAll(t *testing.T) {
	ops := collectTestOperations(t)

	matched, err := filterOperations(ops, "", "", "", false, "", "")
	require.NoError(t, err)

	assert.Len(t, matched, 4)
}

func TestWalkOperations_FilterByMethod(t *testing.T) {
	ops := collectTestOperations(t)

	tests := []struct {
		name   string
		method string
		want   int
	}{
		{name: "filter GET", method: "get", want: 2},
		{name: "filter GET uppercase", method: "GET", want: 2},
		{name: "filter POST", method: "post", want: 1},
		{name: "filter DELETE", method: "delete", want: 1},
		{name: "filter PUT (none)", method: "put", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, err := filterOperations(ops, tt.method, "", "", false, "", "")
			require.NoError(t, err)
			assert.Len(t, matched, tt.want)
		})
	}
}

func TestWalkOperations_FilterByPath(t *testing.T) {
	ops := collectTestOperations(t)

	tests := []struct {
		name string
		path string
		want int
	}{
		{name: "exact /pets", path: "/pets", want: 2},
		{name: "exact /pets/{id}", path: "/pets/{id}", want: 2},
		{name: "glob /pets/*", path: "/pets/*", want: 2},
		{name: "nonexistent path", path: "/users", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, err := filterOperations(ops, "", tt.path, "", false, "", "")
			require.NoError(t, err)
			assert.Len(t, matched, tt.want)
		})
	}
}

func TestWalkOperations_FilterByTag(t *testing.T) {
	ops := collectTestOperations(t)

	tests := []struct {
		name string
		tag  string
		want int
	}{
		{name: "tag pets (all)", tag: "pets", want: 4},
		{name: "tag admin", tag: "admin", want: 1},
		{name: "tag nonexistent", tag: "billing", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, err := filterOperations(ops, "", "", tt.tag, false, "", "")
			require.NoError(t, err)
			assert.Len(t, matched, tt.want)
		})
	}
}

func TestWalkOperations_FilterByDeprecated(t *testing.T) {
	ops := collectTestOperations(t)

	matched, err := filterOperations(ops, "", "", "", true, "", "")
	require.NoError(t, err)

	require.Len(t, matched, 1)
	assert.Equal(t, "deletePet", matched[0].Operation.OperationID)
}

func TestWalkOperations_FilterByOperationID(t *testing.T) {
	ops := collectTestOperations(t)

	matched, err := filterOperations(ops, "", "", "", false, "listPets", "")
	require.NoError(t, err)

	require.Len(t, matched, 1)
	assert.Equal(t, "listPets", matched[0].Operation.OperationID)
}

func TestWalkOperations_FilterByExtension(t *testing.T) {
	ops := collectTestOperations(t)

	// Filter for x-internal existence
	matched, err := filterOperations(ops, "", "", "", false, "", "x-internal")
	require.NoError(t, err)

	require.Len(t, matched, 1)
	assert.Equal(t, "listPets", matched[0].Operation.OperationID)
}

func TestWalkOperations_FilterByExtensionValue(t *testing.T) {
	ops := collectTestOperations(t)

	matched, err := filterOperations(ops, "", "", "", false, "", "x-internal=true")
	require.NoError(t, err)

	require.Len(t, matched, 1)
}

func TestWalkOperations_FilterByExtensionInvalid(t *testing.T) {
	ops := collectTestOperations(t)

	_, err := filterOperations(ops, "", "", "", false, "", "invalid-key")
	assert.Error(t, err)
}

func TestWalkOperations_CombinedFilters(t *testing.T) {
	ops := collectTestOperations(t)

	// GET + /pets/{id} should yield only getPet
	matched, err := filterOperations(ops, "get", "/pets/*", "", false, "", "")
	require.NoError(t, err)

	require.Len(t, matched, 1)
	assert.Equal(t, "getPet", matched[0].Operation.OperationID)
}

func TestWalkOperations_SummaryTableOutput(t *testing.T) {
	ops := collectTestOperations(t)

	// Use only GET operations for predictable output
	matched, err := filterOperations(ops, "get", "/pets", "", false, "", "")
	require.NoError(t, err)

	var buf bytes.Buffer
	headers := []string{"METHOD", "PATH", "SUMMARY", "TAGS", "EXTENSIONS"}
	rows := make([][]string, 0, len(matched))
	for _, op := range matched {
		rows = append(rows, []string{
			strings.ToUpper(op.Method),
			op.PathTemplate,
			op.Operation.Summary,
			strings.Join(op.Operation.Tags, ", "),
			FormatExtensions(op.Operation.Extra),
		})
	}

	RenderSummaryTable(&buf, headers, rows, false)
	output := buf.String()

	// Verify table structure
	assert.Contains(t, output, "METHOD")
	assert.Contains(t, output, "GET")
	assert.Contains(t, output, "/pets")
	assert.Contains(t, output, "List pets")
	assert.Contains(t, output, "x-internal")
}

func TestWalkOperations_SummaryTableQuiet(t *testing.T) {
	ops := collectTestOperations(t)

	matched, err := filterOperations(ops, "post", "", "", false, "", "")
	require.NoError(t, err)

	var buf bytes.Buffer
	headers := []string{"METHOD", "PATH", "SUMMARY", "TAGS", "EXTENSIONS"}
	rows := make([][]string, 0, len(matched))
	for _, op := range matched {
		rows = append(rows, []string{
			strings.ToUpper(op.Method),
			op.PathTemplate,
			op.Operation.Summary,
			strings.Join(op.Operation.Tags, ", "),
			FormatExtensions(op.Operation.Extra),
		})
	}

	RenderSummaryTable(&buf, headers, rows, true)
	output := buf.String()

	// Quiet: no headers
	assert.NotContains(t, output, "METHOD")
	// Still has data
	assert.Contains(t, output, "POST")
}

func TestWalkOperations_SummaryJSON(t *testing.T) {
	ops := collectTestOperations(t)

	matched, err := filterOperations(ops, "get", "/pets", "", false, "", "")
	require.NoError(t, err)

	headers := []string{"METHOD", "PATH", "SUMMARY", "TAGS", "EXTENSIONS"}
	rows := make([][]string, 0, len(matched))
	for _, op := range matched {
		rows = append(rows, []string{
			strings.ToUpper(op.Method),
			op.PathTemplate,
			op.Operation.Summary,
			strings.Join(op.Operation.Tags, ", "),
			FormatExtensions(op.Operation.Extra),
		})
	}

	var buf bytes.Buffer
	err = RenderSummaryStructured(&buf, headers, rows, FormatJSON)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"method"`)
	assert.Contains(t, output, `"path"`)
	assert.Contains(t, output, "GET")
	assert.Contains(t, output, "/pets")
}

func TestWalkOperations_SummaryYAML(t *testing.T) {
	ops := collectTestOperations(t)

	matched, err := filterOperations(ops, "post", "", "", false, "", "")
	require.NoError(t, err)

	headers := []string{"METHOD", "PATH", "SUMMARY", "TAGS", "EXTENSIONS"}
	rows := make([][]string, 0, len(matched))
	for _, op := range matched {
		rows = append(rows, []string{
			strings.ToUpper(op.Method),
			op.PathTemplate,
			op.Operation.Summary,
			strings.Join(op.Operation.Tags, ", "),
			FormatExtensions(op.Operation.Extra),
		})
	}

	var buf bytes.Buffer
	err = RenderSummaryStructured(&buf, headers, rows, FormatYAML)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "method")
	assert.Contains(t, output, "POST")
}

func TestWalkOperations_DetailIncludesPathAndMethod(t *testing.T) {
	ops := collectTestOperations(t)

	matched, err := filterOperations(ops, "", "", "", false, "listPets", "")
	require.NoError(t, err)

	require.Len(t, matched, 1)

	view := operationDetailView{
		Method:    strings.ToUpper(matched[0].Method),
		Path:      matched[0].PathTemplate,
		Operation: matched[0].Operation,
	}

	var buf bytes.Buffer
	err = RenderDetail(&buf, view, FormatJSON)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"method"`)
	assert.Contains(t, output, `"path"`)
	assert.Contains(t, output, "GET")
	assert.Contains(t, output, "/pets")
	assert.Contains(t, output, "listPets")
}

func TestWalkOperations_DetailIncludesPathAndMethodYAML(t *testing.T) {
	ops := collectTestOperations(t)

	matched, err := filterOperations(ops, "", "", "", false, "createPet", "")
	require.NoError(t, err)

	require.Len(t, matched, 1)

	view := operationDetailView{
		Method:    strings.ToUpper(matched[0].Method),
		Path:      matched[0].PathTemplate,
		Operation: matched[0].Operation,
	}

	var buf bytes.Buffer
	err = RenderDetail(&buf, view, FormatYAML)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "method:")
	assert.Contains(t, output, "path:")
	assert.Contains(t, output, "createPet")
}

func TestWalkOperations_NoArgsError(t *testing.T) {
	err := handleWalkOperations([]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires a spec file")
}

func TestWalkOperations_InvalidFormat(t *testing.T) {
	err := handleWalkOperations([]string{"--format", "xml", "test.yaml"})
	assert.Error(t, err)
}

func TestWalkOperations_MatchOperationMethod(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		filter   string
		expected bool
	}{
		{name: "empty filter matches", method: "get", filter: "", expected: true},
		{name: "exact match lowercase", method: "get", filter: "get", expected: true},
		{name: "case insensitive", method: "get", filter: "GET", expected: true},
		{name: "mismatch", method: "get", filter: "post", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchOperationMethod(tt.method, tt.filter)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestWalkOperations_MatchOperationTag(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		filter   string
		expected bool
	}{
		{name: "empty filter matches", tags: []string{"a"}, filter: "", expected: true},
		{name: "tag present", tags: []string{"a", "b"}, filter: "a", expected: true},
		{name: "tag absent", tags: []string{"a"}, filter: "b", expected: false},
		{name: "empty tags with filter", tags: nil, filter: "a", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchOperationTag(tt.tags, tt.filter)
			assert.Equal(t, tt.expected, got)
		})
	}
}
