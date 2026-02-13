package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
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
	if err != nil {
		t.Fatalf("CollectOperations failed: %v", err)
	}
	return collector.All
}

func TestWalkOperations_ListAll(t *testing.T) {
	ops := collectTestOperations(t)

	matched, err := filterOperations(ops, "", "", "", false, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 4 {
		t.Errorf("expected 4 operations, got %d", len(matched))
	}
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
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(matched) != tt.want {
				t.Errorf("filterOperations(method=%q) returned %d, want %d", tt.method, len(matched), tt.want)
			}
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
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(matched) != tt.want {
				t.Errorf("filterOperations(path=%q) returned %d, want %d", tt.path, len(matched), tt.want)
			}
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
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(matched) != tt.want {
				t.Errorf("filterOperations(tag=%q) returned %d, want %d", tt.tag, len(matched), tt.want)
			}
		})
	}
}

func TestWalkOperations_FilterByDeprecated(t *testing.T) {
	ops := collectTestOperations(t)

	matched, err := filterOperations(ops, "", "", "", true, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 deprecated operation, got %d", len(matched))
	}
	if matched[0].Operation.OperationID != "deletePet" {
		t.Errorf("expected deprecated operation deletePet, got %s", matched[0].Operation.OperationID)
	}
}

func TestWalkOperations_FilterByOperationID(t *testing.T) {
	ops := collectTestOperations(t)

	matched, err := filterOperations(ops, "", "", "", false, "listPets", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(matched))
	}
	if matched[0].Operation.OperationID != "listPets" {
		t.Errorf("expected listPets, got %s", matched[0].Operation.OperationID)
	}
}

func TestWalkOperations_FilterByExtension(t *testing.T) {
	ops := collectTestOperations(t)

	// Filter for x-internal existence
	matched, err := filterOperations(ops, "", "", "", false, "", "x-internal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 operation with x-internal, got %d", len(matched))
	}
	if matched[0].Operation.OperationID != "listPets" {
		t.Errorf("expected listPets, got %s", matched[0].Operation.OperationID)
	}
}

func TestWalkOperations_FilterByExtensionValue(t *testing.T) {
	ops := collectTestOperations(t)

	matched, err := filterOperations(ops, "", "", "", false, "", "x-internal=true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 operation with x-internal=true, got %d", len(matched))
	}
}

func TestWalkOperations_FilterByExtensionInvalid(t *testing.T) {
	ops := collectTestOperations(t)

	_, err := filterOperations(ops, "", "", "", false, "", "invalid-key")
	if err == nil {
		t.Error("expected error for invalid extension key")
	}
}

func TestWalkOperations_CombinedFilters(t *testing.T) {
	ops := collectTestOperations(t)

	// GET + /pets/{id} should yield only getPet
	matched, err := filterOperations(ops, "get", "/pets/*", "", false, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(matched))
	}
	if matched[0].Operation.OperationID != "getPet" {
		t.Errorf("expected getPet, got %s", matched[0].Operation.OperationID)
	}
}

func TestWalkOperations_SummaryTableOutput(t *testing.T) {
	ops := collectTestOperations(t)

	// Use only GET operations for predictable output
	matched, err := filterOperations(ops, "get", "/pets", "", false, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
	if !strings.Contains(output, "METHOD") {
		t.Error("expected METHOD header in summary output")
	}
	if !strings.Contains(output, "GET") {
		t.Error("expected GET in summary output")
	}
	if !strings.Contains(output, "/pets") {
		t.Error("expected /pets in summary output")
	}
	if !strings.Contains(output, "List pets") {
		t.Error("expected 'List pets' summary in output")
	}
	if !strings.Contains(output, "x-internal") {
		t.Error("expected x-internal extension in output")
	}
}

func TestWalkOperations_SummaryTableQuiet(t *testing.T) {
	ops := collectTestOperations(t)

	matched, err := filterOperations(ops, "post", "", "", false, "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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
	if strings.Contains(output, "METHOD") {
		t.Error("quiet mode should not contain headers")
	}
	// Still has data
	if !strings.Contains(output, "POST") {
		t.Error("expected POST in quiet output")
	}
}

func TestWalkOperations_DetailOutput(t *testing.T) {
	ops := collectTestOperations(t)

	matched, err := filterOperations(ops, "", "", "", false, "listPets", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(matched))
	}

	var buf bytes.Buffer
	err = RenderDetail(&buf, matched[0].Operation, FormatJSON, false)
	if err != nil {
		t.Fatalf("RenderDetail failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "listPets") {
		t.Error("expected operationId in detail output")
	}
	if !strings.Contains(output, "List pets") {
		t.Error("expected summary in detail output")
	}
}

func TestWalkOperations_DetailOutputYAML(t *testing.T) {
	ops := collectTestOperations(t)

	matched, err := filterOperations(ops, "", "", "", false, "createPet", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(matched) != 1 {
		t.Fatalf("expected 1 operation, got %d", len(matched))
	}

	var buf bytes.Buffer
	err = RenderDetail(&buf, matched[0].Operation, FormatYAML, false)
	if err != nil {
		t.Fatalf("RenderDetail failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "createPet") {
		t.Error("expected operationId in YAML detail output")
	}
}

func TestWalkOperations_NoArgsError(t *testing.T) {
	err := handleWalkOperations([]string{})
	if err == nil {
		t.Error("expected error when no spec file provided")
	}
	if !strings.Contains(err.Error(), "requires a spec file") {
		t.Errorf("expected 'requires a spec file' error, got: %v", err)
	}
}

func TestWalkOperations_InvalidFormat(t *testing.T) {
	err := handleWalkOperations([]string{"--format", "xml", "test.yaml"})
	if err == nil {
		t.Error("expected error for invalid format")
	}
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
			if got != tt.expected {
				t.Errorf("matchOperationMethod(%q, %q) = %v, want %v", tt.method, tt.filter, got, tt.expected)
			}
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
			if got != tt.expected {
				t.Errorf("matchOperationTag(%v, %q) = %v, want %v", tt.tags, tt.filter, got, tt.expected)
			}
		})
	}
}
