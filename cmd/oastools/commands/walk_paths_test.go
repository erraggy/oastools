package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
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
	if err != nil {
		t.Fatalf("collectPaths failed: %v", err)
	}
	return paths
}

func TestWalkPaths_ListAll(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 3 {
		t.Errorf("expected 3 paths, got %d", len(matched))
	}
}

func TestWalkPaths_FilterByPathExact(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "/pets", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 path, got %d", len(matched))
	}
	if matched[0].pathTemplate != "/pets" {
		t.Errorf("expected /pets, got %s", matched[0].pathTemplate)
	}
}

func TestWalkPaths_FilterByPathGlob(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "/pets/*", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 path matching /pets/*, got %d", len(matched))
	}
	if matched[0].pathTemplate != "/pets/{id}" {
		t.Errorf("expected /pets/{id}, got %s", matched[0].pathTemplate)
	}
}

func TestWalkPaths_FilterByExtension(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "", "x-resource")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 path with x-resource, got %d", len(matched))
	}
	if matched[0].pathTemplate != "/pets" {
		t.Errorf("expected /pets, got %s", matched[0].pathTemplate)
	}
}

func TestWalkPaths_FilterByExtensionValue(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "", "x-resource=pets")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 path with x-resource=pets, got %d", len(matched))
	}
}

func TestWalkPaths_FilterByExtensionInvalid(t *testing.T) {
	paths := collectTestPaths(t)

	_, err := filterPaths(paths, "", "invalid-key")
	if err == nil {
		t.Error("expected error for invalid extension key")
	}
}

func TestWalkPaths_FilterByExtensionNoMatch(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "", "x-nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 0 {
		t.Errorf("expected 0 paths, got %d", len(matched))
	}
}

func TestWalkPaths_FilterCombined(t *testing.T) {
	paths := collectTestPaths(t)

	// Path pattern + extension: /pets matches the pattern, and has x-resource
	matched, err := filterPaths(paths, "/pets", "x-resource")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 path, got %d", len(matched))
	}
	if matched[0].pathTemplate != "/pets" {
		t.Errorf("expected /pets, got %s", matched[0].pathTemplate)
	}
}

func TestWalkPaths_FilterCombinedNoMatch(t *testing.T) {
	paths := collectTestPaths(t)

	// /users matches path pattern but has no x-resource extension
	matched, err := filterPaths(paths, "/users", "x-resource")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 0 {
		t.Errorf("expected 0 paths, got %d", len(matched))
	}
}

func TestWalkPaths_FilterNonexistentPath(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "/nonexistent", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 0 {
		t.Errorf("expected 0 paths, got %d", len(matched))
	}
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
			if got != tt.want {
				t.Errorf("pathMethods() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWalkPaths_SummaryTableOutput(t *testing.T) {
	paths := collectTestPaths(t)

	// Filter to just /pets for predictable output
	matched, err := filterPaths(paths, "/pets", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

	if !strings.Contains(output, "PATH") {
		t.Error("expected PATH header in summary output")
	}
	if !strings.Contains(output, "/pets") {
		t.Error("expected /pets in summary output")
	}
	if !strings.Contains(output, "GET, POST") {
		t.Error("expected 'GET, POST' methods in summary output")
	}
	if !strings.Contains(output, "Pet operations") {
		t.Error("expected 'Pet operations' summary in output")
	}
	if !strings.Contains(output, "x-resource") {
		t.Error("expected x-resource extension in output")
	}
}

func TestWalkPaths_SummaryTableQuiet(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "/users", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
	if strings.Contains(output, "PATH") {
		t.Error("quiet mode should not contain headers")
	}
	// Still has data
	if !strings.Contains(output, "/users") {
		t.Error("expected /users in quiet output")
	}
}

func TestWalkPaths_DetailOutput(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "/pets", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 path, got %d", len(matched))
	}

	var buf bytes.Buffer
	err = RenderDetail(&buf, matched[0].pathItem, FormatJSON, false)
	if err != nil {
		t.Fatalf("RenderDetail failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Pet operations") {
		t.Error("expected summary in detail output")
	}
	if !strings.Contains(output, "List pets") {
		t.Error("expected operation summary in detail output")
	}
}

func TestWalkPaths_DetailOutputYAML(t *testing.T) {
	paths := collectTestPaths(t)

	matched, err := filterPaths(paths, "/users", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 path, got %d", len(matched))
	}

	var buf bytes.Buffer
	err = RenderDetail(&buf, matched[0].pathItem, FormatYAML, false)
	if err != nil {
		t.Fatalf("RenderDetail failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "User operations") {
		t.Error("expected summary in YAML detail output")
	}
}

func TestWalkPaths_NoArgsError(t *testing.T) {
	err := handleWalkPaths([]string{})
	if err == nil {
		t.Error("expected error when no spec file provided")
	}
	if !strings.Contains(err.Error(), "requires a spec file") {
		t.Errorf("expected 'requires a spec file' error, got: %v", err)
	}
}

func TestWalkPaths_InvalidFormat(t *testing.T) {
	err := handleWalkPaths([]string{"--format", "xml", "test.yaml"})
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestWalkPaths_CollectPathsNilResult(t *testing.T) {
	_, err := collectPaths(nil)
	if err == nil {
		t.Error("expected error for nil ParseResult")
	}
}
