# Walk Command Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add an `oastools walk` CLI command for structured, version-agnostic querying of OAS documents.

**Architecture:** Subcommand-per-node-type (operations, schemas, parameters, responses, security, paths) using the existing walker package's collector pattern. Shared infrastructure in `walk.go` handles extension filtering, table rendering, and detail output. All code lives in `cmd/oastools/commands/`.

**Tech Stack:** Go 1.24+, existing `walker` and `parser` packages, `flag` stdlib for CLI parsing.

---

## Task Dependencies

Tasks are organized into 3 phases. Tasks within each phase can run in parallel.

- **Phase 1** (Tasks 1-3): Foundation — can all run in parallel
- **Phase 2** (Tasks 4-5): Walker collectors + CLI router — depend on Phase 1
- **Phase 3** (Tasks 6-11): Subcommands — each depends on Tasks 1-2, can run in parallel with each other

```
Phase 1 (parallel):  [Task 1: Extension Filter] [Task 2: Renderers] [Task 3: Walker Collectors]
                            ↓                        ↓                      ↓
Phase 2 (sequential): [Task 4: walk.go CLI Router (needs 1+2)]  [Task 5: main.go integration]
                            ↓
Phase 3 (parallel):   [Task 6: operations] [Task 7: schemas] [Task 8: parameters]
                      [Task 9: responses]  [Task 10: security] [Task 11: paths]
```

---

### Task 1: Extension Filter Parser & Matcher

Standalone logic with no dependencies. Can be developed and tested in isolation.

**Files:**
- Create: `cmd/oastools/commands/walk_extension_filter.go`
- Test: `cmd/oastools/commands/walk_extension_filter_test.go`

**Step 1: Write the failing tests**

```go
// walk_extension_filter_test.go
package commands

import (
	"testing"
)

func TestParseExtensionFilter(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ExtensionFilter
		wantErr bool
	}{
		{
			name:  "simple existence",
			input: "x-foo",
			want: ExtensionFilter{
				Groups: [][]ExtensionExpr{
					{{Key: "x-foo"}},
				},
			},
		},
		{
			name:  "key=value",
			input: "x-foo=bar",
			want: ExtensionFilter{
				Groups: [][]ExtensionExpr{
					{{Key: "x-foo", Value: strPtr("bar")}},
				},
			},
		},
		{
			name:  "negated existence",
			input: "!x-foo",
			want: ExtensionFilter{
				Groups: [][]ExtensionExpr{
					{{Key: "x-foo", Negated: true}},
				},
			},
		},
		{
			name:  "negated value",
			input: "x-foo!=bar",
			want: ExtensionFilter{
				Groups: [][]ExtensionExpr{
					{{Key: "x-foo", Value: strPtr("bar"), Negated: true}},
				},
			},
		},
		{
			name:  "AND operator",
			input: "x-foo+x-bar=1",
			want: ExtensionFilter{
				Groups: [][]ExtensionExpr{
					{
						{Key: "x-foo"},
						{Key: "x-bar", Value: strPtr("1")},
					},
				},
			},
		},
		{
			name:  "OR operator",
			input: "x-foo,x-bar=1",
			want: ExtensionFilter{
				Groups: [][]ExtensionExpr{
					{{Key: "x-foo"}},
					{{Key: "x-bar", Value: strPtr("1")}},
				},
			},
		},
		{
			name:  "mixed AND+OR",
			input: "x-a+x-b=1,!x-c",
			want: ExtensionFilter{
				Groups: [][]ExtensionExpr{
					{
						{Key: "x-a"},
						{Key: "x-b", Value: strPtr("1")},
					},
					{{Key: "x-c", Negated: true}},
				},
			},
		},
		{
			name:    "missing x- prefix",
			input:   "foo",
			wantErr: true,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "bare equals",
			input:   "=value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseExtensionFilter(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Compare groups length and content
			if len(got.Groups) != len(tt.want.Groups) {
				t.Fatalf("groups: got %d, want %d", len(got.Groups), len(tt.want.Groups))
			}
			for i, group := range got.Groups {
				if len(group) != len(tt.want.Groups[i]) {
					t.Fatalf("group[%d]: got %d exprs, want %d", i, len(group), len(tt.want.Groups[i]))
				}
				for j, expr := range group {
					wantExpr := tt.want.Groups[i][j]
					if expr.Key != wantExpr.Key {
						t.Errorf("group[%d][%d].Key: got %q, want %q", i, j, expr.Key, wantExpr.Key)
					}
					if expr.Negated != wantExpr.Negated {
						t.Errorf("group[%d][%d].Negated: got %v, want %v", i, j, expr.Negated, wantExpr.Negated)
					}
					gotVal := "<nil>"
					if expr.Value != nil {
						gotVal = *expr.Value
					}
					wantVal := "<nil>"
					if wantExpr.Value != nil {
						wantVal = *wantExpr.Value
					}
					if gotVal != wantVal {
						t.Errorf("group[%d][%d].Value: got %q, want %q", i, j, gotVal, wantVal)
					}
				}
			}
		})
	}
}

func TestExtensionFilter_Match(t *testing.T) {
	tests := []struct {
		name       string
		filter     string
		extensions map[string]any
		want       bool
	}{
		{
			name:       "existence match",
			filter:     "x-foo",
			extensions: map[string]any{"x-foo": "bar"},
			want:       true,
		},
		{
			name:       "existence no match",
			filter:     "x-foo",
			extensions: map[string]any{"x-bar": "baz"},
			want:       false,
		},
		{
			name:       "existence nil extensions",
			filter:     "x-foo",
			extensions: nil,
			want:       false,
		},
		{
			name:       "value match string",
			filter:     "x-foo=bar",
			extensions: map[string]any{"x-foo": "bar"},
			want:       true,
		},
		{
			name:       "value match bool as string",
			filter:     "x-internal=true",
			extensions: map[string]any{"x-internal": true},
			want:       true,
		},
		{
			name:       "value no match",
			filter:     "x-foo=bar",
			extensions: map[string]any{"x-foo": "baz"},
			want:       false,
		},
		{
			name:       "negated existence - key absent",
			filter:     "!x-foo",
			extensions: map[string]any{"x-bar": "baz"},
			want:       true,
		},
		{
			name:       "negated existence - key present",
			filter:     "!x-foo",
			extensions: map[string]any{"x-foo": "bar"},
			want:       false,
		},
		{
			name:       "negated value",
			filter:     "x-foo!=bar",
			extensions: map[string]any{"x-foo": "baz"},
			want:       true,
		},
		{
			name:       "AND - both match",
			filter:     "x-foo+x-bar",
			extensions: map[string]any{"x-foo": "1", "x-bar": "2"},
			want:       true,
		},
		{
			name:       "AND - one missing",
			filter:     "x-foo+x-bar",
			extensions: map[string]any{"x-foo": "1"},
			want:       false,
		},
		{
			name:       "OR - first matches",
			filter:     "x-foo,x-bar",
			extensions: map[string]any{"x-foo": "1"},
			want:       true,
		},
		{
			name:       "OR - second matches",
			filter:     "x-foo,x-bar",
			extensions: map[string]any{"x-bar": "1"},
			want:       true,
		},
		{
			name:       "OR - none match",
			filter:     "x-foo,x-bar",
			extensions: map[string]any{"x-baz": "1"},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := ParseExtensionFilter(tt.filter)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			got := f.Match(tt.extensions)
			if got != tt.want {
				t.Errorf("Match(%v) = %v, want %v", tt.extensions, got, tt.want)
			}
		})
	}
}

func strPtr(s string) *string { return &s }
```

**Step 2: Run tests to verify they fail**

Run: `go test ./cmd/oastools/commands/ -run 'TestParseExtensionFilter|TestExtensionFilter_Match' -v`
Expected: FAIL — types and functions not defined

**Step 3: Write the implementation**

```go
// walk_extension_filter.go
package commands

import (
	"fmt"
	"strings"
)

// ExtensionFilter represents a parsed --extension filter.
// Groups are OR'd together; expressions within a group are AND'd.
type ExtensionFilter struct {
	Groups [][]ExtensionExpr
}

// ExtensionExpr is a single extension filter expression.
type ExtensionExpr struct {
	Key     string  // e.g., "x-audited-by"
	Value   *string // nil = existence check only
	Negated bool    // ! prefix or != operator
}

// ParseExtensionFilter parses the --extension flag value.
// Grammar: FILTER = EXPR ( ("," | "+") EXPR )*
// , = OR (separates groups), + = AND (within a group)
func ParseExtensionFilter(input string) (ExtensionFilter, error) {
	if input == "" {
		return ExtensionFilter{}, fmt.Errorf("empty extension filter")
	}

	var filter ExtensionFilter

	// Split by , for OR groups
	orParts := strings.Split(input, ",")
	for _, orPart := range orParts {
		if orPart == "" {
			return ExtensionFilter{}, fmt.Errorf("empty expression in extension filter")
		}

		// Split by + for AND expressions within a group
		andParts := strings.Split(orPart, "+")
		var group []ExtensionExpr

		for _, part := range andParts {
			expr, err := parseExtensionExpr(part)
			if err != nil {
				return ExtensionFilter{}, err
			}
			group = append(group, expr)
		}

		filter.Groups = append(filter.Groups, group)
	}

	return filter, nil
}

// parseExtensionExpr parses a single expression like "x-foo", "x-foo=bar", "!x-foo", "x-foo!=bar".
func parseExtensionExpr(s string) (ExtensionExpr, error) {
	if s == "" {
		return ExtensionExpr{}, fmt.Errorf("empty expression in extension filter")
	}

	var expr ExtensionExpr

	// Check for ! prefix (negation)
	if s[0] == '!' {
		expr.Negated = true
		s = s[1:]
	}

	// Check for != operator
	if idx := strings.Index(s, "!="); idx > 0 {
		expr.Key = s[:idx]
		val := s[idx+2:]
		expr.Value = &val
		expr.Negated = true
	} else if idx := strings.Index(s, "="); idx > 0 {
		// Check for = operator
		expr.Key = s[:idx]
		val := s[idx+1:]
		expr.Value = &val
	} else {
		// Existence check only
		expr.Key = s
	}

	// Validate key starts with x-
	if !strings.HasPrefix(expr.Key, "x-") {
		return ExtensionExpr{}, fmt.Errorf("invalid extension key %q: must start with \"x-\"", expr.Key)
	}

	return expr, nil
}

// Match evaluates the filter against a node's extensions.
// Returns true if the filter matches.
func (f ExtensionFilter) Match(extensions map[string]any) bool {
	// OR across groups: any group matching is sufficient
	for _, group := range f.Groups {
		if matchGroup(group, extensions) {
			return true
		}
	}
	return false
}

// matchGroup evaluates AND logic: all expressions must match.
func matchGroup(group []ExtensionExpr, extensions map[string]any) bool {
	for _, expr := range group {
		if !matchExpr(expr, extensions) {
			return false
		}
	}
	return true
}

// matchExpr evaluates a single expression against extensions.
func matchExpr(expr ExtensionExpr, extensions map[string]any) bool {
	val, exists := extensions[expr.Key]

	if expr.Value == nil {
		// Existence check
		if expr.Negated {
			return !exists
		}
		return exists
	}

	// Value comparison
	valStr := fmt.Sprintf("%v", val)
	matches := exists && valStr == *expr.Value

	if expr.Negated {
		return !matches
	}
	return matches
}

// FormatExtensions formats a map of extensions as a comma-separated string for summary output.
func FormatExtensions(extra map[string]any) string {
	if len(extra) == 0 {
		return ""
	}
	var parts []string
	for k, v := range extra {
		if strings.HasPrefix(k, "x-") {
			parts = append(parts, fmt.Sprintf("%s=%v", k, v))
		}
	}
	return strings.Join(parts, ", ")
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./cmd/oastools/commands/ -run 'TestParseExtensionFilter|TestExtensionFilter_Match' -v`
Expected: PASS

**Step 5: Run go_diagnostics**

Run: `go_diagnostics` on `cmd/oastools/commands/walk_extension_filter.go`

**Step 6: Commit**

```bash
git add cmd/oastools/commands/walk_extension_filter.go cmd/oastools/commands/walk_extension_filter_test.go
git commit -m "feat(walk): add extension filter parser and matcher"
```

---

### Task 2: Summary Table & Detail Renderers

Shared rendering infrastructure. No dependencies on other tasks.

**Files:**
- Create: `cmd/oastools/commands/walk_render.go`
- Test: `cmd/oastools/commands/walk_render_test.go`

**Step 1: Write failing tests**

```go
// walk_render_test.go
package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestRenderSummaryTable(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"METHOD", "PATH", "SUMMARY"}
	rows := [][]string{
		{"GET", "/pets", "List pets"},
		{"POST", "/pets", "Create pet"},
	}

	RenderSummaryTable(&buf, headers, rows, false)
	output := buf.String()

	// Should contain headers
	if !strings.Contains(output, "METHOD") {
		t.Error("expected headers in output")
	}
	// Should contain data
	if !strings.Contains(output, "GET") {
		t.Error("expected GET in output")
	}
	if !strings.Contains(output, "/pets") {
		t.Error("expected /pets in output")
	}
}

func TestRenderSummaryTable_Quiet(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"METHOD", "PATH"}
	rows := [][]string{
		{"GET", "/pets"},
	}

	RenderSummaryTable(&buf, headers, rows, true)
	output := buf.String()

	// Quiet mode: no header row
	if strings.Contains(output, "METHOD") {
		t.Error("quiet mode should not include headers")
	}
	// Should still contain data
	if !strings.Contains(output, "GET") {
		t.Error("expected GET in output")
	}
}

func TestRenderSummaryTable_Empty(t *testing.T) {
	var buf bytes.Buffer
	RenderSummaryTable(&buf, []string{"A"}, nil, false)
	if buf.Len() != 0 {
		t.Errorf("expected empty output for no rows, got %q", buf.String())
	}
}

func TestRenderDetail_YAML(t *testing.T) {
	var buf bytes.Buffer
	node := map[string]any{
		"summary": "List pets",
		"tags":    []string{"pets"},
	}

	err := RenderDetail(&buf, node, FormatYAML, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "summary") {
		t.Error("expected summary in YAML output")
	}
}

func TestRenderDetail_JSON(t *testing.T) {
	var buf bytes.Buffer
	node := map[string]any{
		"summary": "List pets",
	}

	err := RenderDetail(&buf, node, FormatJSON, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, `"summary"`) {
		t.Error("expected summary in JSON output")
	}
}
```

**Step 2: Run tests to verify fail**

Run: `go test ./cmd/oastools/commands/ -run 'TestRenderSummaryTable|TestRenderDetail' -v`

**Step 3: Write implementation**

```go
// walk_render.go
package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"go.yaml.in/yaml/v4"
)

// RenderSummaryTable renders a table of results.
// In quiet mode, headers are omitted and rows are tab-separated.
func RenderSummaryTable(w io.Writer, headers []string, rows [][]string, quiet bool) {
	if len(rows) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	if !quiet {
		// Print header
		for i, h := range headers {
			if i > 0 {
				fmt.Fprint(w, "  ")
			}
			fmt.Fprintf(w, "%-*s", widths[i], h)
		}
		fmt.Fprintln(w)
	}

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if quiet {
				if i > 0 {
					fmt.Fprint(w, "\t")
				}
				fmt.Fprint(w, cell)
			} else {
				if i > 0 {
					fmt.Fprint(w, "  ")
				}
				fmt.Fprintf(w, "%-*s", widths[i], cell)
			}
		}
		fmt.Fprintln(w)
	}
}

// RenderDetail renders one or more nodes in the specified format.
func RenderDetail(w io.Writer, node any, format string, quiet bool) error {
	var data []byte
	var err error

	switch format {
	case FormatJSON:
		data, err = json.MarshalIndent(node, "", "  ")
	case FormatYAML, FormatText:
		data, err = yaml.Marshal(node)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return fmt.Errorf("marshaling output: %w", err)
	}

	fmt.Fprintln(w, strings.TrimRight(string(data), "\n"))
	return nil
}
```

**Step 4: Run tests to verify pass**

Run: `go test ./cmd/oastools/commands/ -run 'TestRenderSummaryTable|TestRenderDetail' -v`

**Step 5: Run go_diagnostics**

**Step 6: Commit**

```bash
git add cmd/oastools/commands/walk_render.go cmd/oastools/commands/walk_render_test.go
git commit -m "feat(walk): add summary table and detail renderers"
```

---

### Task 3: Walker Collectors for Parameters, Responses, Security

Add new collectors to the walker package (benefits library users too).

**Files:**
- Modify: `walker/collectors.go` (add new collectors after existing ones)
- Modify: `walker/collectors_test.go` (add tests)

**Context needed:** Read `walker/collectors.go:1-155` for the existing `CollectSchemas` and `CollectOperations` patterns, and `walker/context.go:1-50` for WalkContext fields.

**Step 1: Write failing tests**

Add to `walker/collectors_test.go`:

```go
func TestCollectParameters(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					Parameters: []*parser.Parameter{
						{Name: "limit", In: "query"},
						{Name: "offset", In: "query"},
					},
				},
			},
			"/pets/{id}": &parser.PathItem{
				Parameters: []*parser.Parameter{
					{Name: "id", In: "path", Required: true},
				},
			},
		},
	}

	result := &parser.ParseResult{Document: doc, Version: "3.0.3"}
	collector, err := CollectParameters(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(collector.All) != 3 {
		t.Errorf("expected 3 parameters, got %d", len(collector.All))
	}
}

func TestCollectResponses(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "OK"},
							"404": {Description: "Not Found"},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{Document: doc, Version: "3.0.3"}
	collector, err := CollectResponses(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(collector.All) != 2 {
		t.Errorf("expected 2 responses, got %d", len(collector.All))
	}
}

func TestCollectSecuritySchemes(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"bearerAuth": {Type: "http", Scheme: "bearer"},
				"apiKey":     {Type: "apiKey", Name: "X-API-Key", In: "header"},
			},
		},
	}

	result := &parser.ParseResult{Document: doc, Version: "3.0.3"}
	collector, err := CollectSecuritySchemes(result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(collector.All) != 2 {
		t.Errorf("expected 2 security schemes, got %d", len(collector.All))
	}
}
```

**Step 2: Run tests to verify fail**

Run: `go test ./walker/ -run 'TestCollectParameters|TestCollectResponses|TestCollectSecuritySchemes' -v`

**Step 3: Write implementation**

Append to `walker/collectors.go`:

```go
// ParameterInfo contains information about a collected parameter.
type ParameterInfo struct {
	Parameter    *parser.Parameter
	Name         string // Parameter name
	In           string // Location: query, header, path, cookie
	JSONPath     string
	PathTemplate string // Owning path
	Method       string // Owning operation method (empty if path-level)
	IsComponent  bool
}

// ParameterCollector holds parameters collected during a walk.
type ParameterCollector struct {
	All        []*ParameterInfo
	ByName     map[string][]*ParameterInfo
	ByLocation map[string][]*ParameterInfo // "query", "header", "path", "cookie"
	ByPath     map[string][]*ParameterInfo
}

// CollectParameters walks the document and collects all parameters.
func CollectParameters(result *parser.ParseResult) (*ParameterCollector, error) {
	collector := &ParameterCollector{
		All:        make([]*ParameterInfo, 0),
		ByName:     make(map[string][]*ParameterInfo),
		ByLocation: make(map[string][]*ParameterInfo),
		ByPath:     make(map[string][]*ParameterInfo),
	}

	err := Walk(result,
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			info := &ParameterInfo{
				Parameter:    param,
				Name:         param.Name,
				In:           param.In,
				JSONPath:     wc.JSONPath,
				PathTemplate: wc.PathTemplate,
				Method:       wc.Method,
				IsComponent:  wc.IsComponent,
			}

			collector.All = append(collector.All, info)
			collector.ByName[param.Name] = append(collector.ByName[param.Name], info)
			collector.ByLocation[param.In] = append(collector.ByLocation[param.In], info)
			if wc.PathTemplate != "" {
				collector.ByPath[wc.PathTemplate] = append(collector.ByPath[wc.PathTemplate], info)
			}

			return Continue
		}),
	)

	if err != nil {
		return nil, err
	}

	return collector, nil
}

// ResponseInfo contains information about a collected response.
type ResponseInfo struct {
	Response     *parser.Response
	StatusCode   string // "200", "404", "default", etc.
	JSONPath     string
	PathTemplate string
	Method       string
	IsComponent  bool
}

// ResponseCollector holds responses collected during a walk.
type ResponseCollector struct {
	All          []*ResponseInfo
	ByStatusCode map[string][]*ResponseInfo
	ByPath       map[string][]*ResponseInfo
}

// CollectResponses walks the document and collects all responses.
func CollectResponses(result *parser.ParseResult) (*ResponseCollector, error) {
	collector := &ResponseCollector{
		All:          make([]*ResponseInfo, 0),
		ByStatusCode: make(map[string][]*ResponseInfo),
		ByPath:       make(map[string][]*ResponseInfo),
	}

	err := Walk(result,
		WithResponseHandler(func(wc *WalkContext, resp *parser.Response) Action {
			info := &ResponseInfo{
				Response:     resp,
				StatusCode:   wc.StatusCode,
				JSONPath:     wc.JSONPath,
				PathTemplate: wc.PathTemplate,
				Method:       wc.Method,
				IsComponent:  wc.IsComponent,
			}

			collector.All = append(collector.All, info)
			if wc.StatusCode != "" {
				collector.ByStatusCode[wc.StatusCode] = append(collector.ByStatusCode[wc.StatusCode], info)
			}
			if wc.PathTemplate != "" {
				collector.ByPath[wc.PathTemplate] = append(collector.ByPath[wc.PathTemplate], info)
			}

			return Continue
		}),
	)

	if err != nil {
		return nil, err
	}

	return collector, nil
}

// SecuritySchemeInfo contains information about a collected security scheme.
type SecuritySchemeInfo struct {
	SecurityScheme *parser.SecurityScheme
	Name           string
	JSONPath       string
}

// SecuritySchemeCollector holds security schemes collected during a walk.
type SecuritySchemeCollector struct {
	All    []*SecuritySchemeInfo
	ByName map[string]*SecuritySchemeInfo
}

// CollectSecuritySchemes walks the document and collects all security schemes.
func CollectSecuritySchemes(result *parser.ParseResult) (*SecuritySchemeCollector, error) {
	collector := &SecuritySchemeCollector{
		All:    make([]*SecuritySchemeInfo, 0),
		ByName: make(map[string]*SecuritySchemeInfo),
	}

	err := Walk(result,
		WithSecuritySchemeHandler(func(wc *WalkContext, scheme *parser.SecurityScheme) Action {
			info := &SecuritySchemeInfo{
				SecurityScheme: scheme,
				Name:           wc.Name,
				JSONPath:       wc.JSONPath,
			}

			collector.All = append(collector.All, info)
			if wc.Name != "" {
				collector.ByName[wc.Name] = info
			}

			return Continue
		}),
	)

	if err != nil {
		return nil, err
	}

	return collector, nil
}
```

**Step 4: Run tests to verify pass**

Run: `go test ./walker/ -run 'TestCollectParameters|TestCollectResponses|TestCollectSecuritySchemes' -v`

**Step 5: Run go_diagnostics**

**Step 6: Commit**

```bash
git add walker/collectors.go walker/collectors_test.go
git commit -m "feat(walker): add collectors for parameters, responses, and security schemes"
```

---

### Task 4: Walk Command Router (`walk.go`)

The CLI entry point that parses the subcommand and routes to the right handler. Depends on Tasks 1 and 2.

**Files:**
- Create: `cmd/oastools/commands/walk.go`
- Test: `cmd/oastools/commands/walk_test.go`

**Context needed:** Read `cmd/oastools/commands/validate.go` for the `SetupFlags`/`Handle` pattern and `cmd/oastools/commands/common.go` for shared utilities.

**Step 1: Write failing tests**

```go
// walk_test.go
package commands

import (
	"testing"
)

func TestHandleWalk_NoArgs(t *testing.T) {
	err := HandleWalk([]string{})
	if err == nil {
		t.Error("expected error when no subcommand provided")
	}
}

func TestHandleWalk_InvalidSubcommand(t *testing.T) {
	err := HandleWalk([]string{"invalid"})
	if err == nil {
		t.Error("expected error for invalid subcommand")
	}
}

func TestHandleWalk_Help(t *testing.T) {
	err := HandleWalk([]string{"--help"})
	if err != nil {
		t.Errorf("unexpected error for --help: %v", err)
	}
}
```

**Step 2: Run to verify fail**

**Step 3: Write implementation**

```go
// walk.go
package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/erraggy/oastools/parser"
)

// walkSubcommands lists valid walk subcommands.
var walkSubcommands = []string{
	"operations", "schemas", "parameters", "responses", "security", "paths",
}

// HandleWalk routes the walk command to the appropriate subcommand handler.
func HandleWalk(args []string) error {
	if len(args) == 0 {
		printWalkUsage()
		return fmt.Errorf("walk command requires a subcommand")
	}

	subcommand := args[0]

	// Handle --help at walk level
	if subcommand == "--help" || subcommand == "-h" || subcommand == "help" {
		printWalkUsage()
		return nil
	}

	subArgs := args[1:]

	switch subcommand {
	case "operations":
		return handleWalkOperations(subArgs)
	case "schemas":
		return handleWalkSchemas(subArgs)
	case "parameters":
		return handleWalkParameters(subArgs)
	case "responses":
		return handleWalkResponses(subArgs)
	case "security":
		return handleWalkSecurity(subArgs)
	case "paths":
		return handleWalkPaths(subArgs)
	default:
		printWalkUsage()
		return fmt.Errorf("unknown walk subcommand: %s", subcommand)
	}
}

// WalkFlags contains common flags shared by all walk subcommands.
type WalkFlags struct {
	Format      string
	Quiet       bool
	Detail      bool
	Extension   string
	ResolveRefs bool
}

// parseSpec parses the spec file, handling stdin and resolve-refs.
func parseSpec(specPath string, resolveRefs bool) (*parser.ParseResult, error) {
	p := parser.New()
	p.ResolveRefs = resolveRefs

	if specPath == StdinFilePath {
		return p.ParseReader(os.Stdin)
	}
	return p.Parse(specPath)
}

// renderNoResults prints an informative message when no results match the filters.
func renderNoResults(nodeType string, quiet bool) {
	if !quiet {
		Writef(os.Stderr, "No %s matched the given filters.\n", nodeType)
	}
}

// matchPath checks if a path template matches a pattern (supports simple glob with *).
func matchPath(pathTemplate, pattern string) bool {
	if pattern == "" {
		return true
	}
	// Simple glob: * matches any path segment
	if strings.Contains(pattern, "*") {
		patternParts := strings.Split(pattern, "/")
		pathParts := strings.Split(pathTemplate, "/")
		if len(patternParts) != len(pathParts) {
			return false
		}
		for i, pp := range patternParts {
			if pp == "*" {
				continue
			}
			if pp != pathParts[i] {
				return false
			}
		}
		return true
	}
	return pathTemplate == pattern
}

// matchStatusCode checks if a status code matches a pattern (supports wildcards like 2xx, 4xx).
func matchStatusCode(code, pattern string) bool {
	if pattern == "" {
		return true
	}
	pattern = strings.ToLower(pattern)
	code = strings.ToLower(code)
	if len(pattern) == 3 && strings.HasSuffix(pattern, "xx") {
		return len(code) >= 1 && code[0] == pattern[0]
	}
	return code == pattern
}

func printWalkUsage() {
	Writef(os.Stderr, `Usage: oastools walk <subcommand> [flags] <file|url|->

Query and explore OpenAPI specification documents.

Subcommands:
  operations    List or inspect operations
  schemas       List or inspect schemas
  parameters    List or inspect parameters
  responses     List or inspect responses
  security      List or inspect security schemes
  paths         List or inspect path items

Common Flags:
  --format      Output format: text (default), json, yaml
  -q, --quiet   Suppress headers and decoration for piping
  --detail      Show full node instead of summary table
  --extension   Filter by extension (e.g., x-internal=true)
  --resolve-refs  Resolve $ref pointers in detail output

Examples:
  oastools walk operations api.yaml
  oastools walk operations --method get --path /pets --detail api.yaml
  oastools walk schemas --name Pet --detail api.yaml
  oastools walk operations --extension x-audited-by api.yaml
  oastools walk responses --status '4xx' -q --format json api.yaml | jq

Run 'oastools walk <subcommand> --help' for subcommand-specific flags.
`)
}
```

**Step 4: Run tests, verify pass**

**Step 5: Run go_diagnostics** — the subcommand handler functions don't exist yet, so there will be compile errors. That's expected — they'll be created in Phase 3 tasks. For now, add stub functions so it compiles:

Create temporary stubs in `walk.go` (these will be replaced by Phase 3 tasks):

```go
// Stubs — replaced by subcommand implementations
func handleWalkOperations(args []string) error { return fmt.Errorf("not yet implemented") }
func handleWalkSchemas(args []string) error    { return fmt.Errorf("not yet implemented") }
func handleWalkParameters(args []string) error { return fmt.Errorf("not yet implemented") }
func handleWalkResponses(args []string) error  { return fmt.Errorf("not yet implemented") }
func handleWalkSecurity(args []string) error   { return fmt.Errorf("not yet implemented") }
func handleWalkPaths(args []string) error      { return fmt.Errorf("not yet implemented") }
```

**Step 6: Commit**

```bash
git add cmd/oastools/commands/walk.go cmd/oastools/commands/walk_test.go
git commit -m "feat(walk): add walk command router with common flags and utilities"
```

---

### Task 5: Integrate Walk Command into main.go

Wire up the walk command in the CLI entry point.

**Files:**
- Modify: `cmd/oastools/main.go`

**Step 1: Add "walk" to validCommands and the switch statement**

In `cmd/oastools/main.go`, add `"walk"` to the `validCommands` slice (line 13) and add a case in the switch (after the overlay case, around line 123):

```go
// Add to validCommands slice:
var validCommands = []string{
	"validate", "fix", "convert", "diff", "generate", "join", "overlay", "parse", "walk", "version", "help",
}

// Add case in switch statement:
case "walk":
	if err := commands.HandleWalk(os.Args[2:]); err != nil {
		commands.Writef(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
```

Also add `walk` to the `printUsage()` function:

```
  walk        Query and explore OpenAPI specification documents
```

**Step 2: Run go_diagnostics**

**Step 3: Commit**

```bash
git add cmd/oastools/main.go
git commit -m "feat(walk): integrate walk command into CLI entry point"
```

---

### Task 6: Operations Subcommand

**Files:**
- Create: `cmd/oastools/commands/walk_operations.go`
- Test: `cmd/oastools/commands/walk_operations_test.go`

**Context needed:** Read `walker/collectors.go:87-155` for `OperationInfo`/`OperationCollector`, and `parser/paths.go:37-56` for `Operation` struct fields including `Extra`.

**Step 1: Write failing tests**

Test flag parsing, filtering by method/path/tag/extension, summary and detail output.

**Step 2: Implement `handleWalkOperations`**

- Setup flags: `--method`, `--path`, `--tag`, `--deprecated`, `--operationId`, plus common flags
- Parse spec, use `walker.CollectOperations(result)`
- Filter collected operations
- Render as summary table (columns: METHOD, PATH, SUMMARY, TAGS, EXTENSIONS) or detail

**Step 3: Remove stub from walk.go**

Delete the `handleWalkOperations` stub function.

**Step 4: Run tests, run go_diagnostics**

**Step 5: Commit**

```bash
git commit -m "feat(walk): add operations subcommand"
```

---

### Task 7: Schemas Subcommand

**Files:**
- Create: `cmd/oastools/commands/walk_schemas.go`
- Test: `cmd/oastools/commands/walk_schemas_test.go`

**Context needed:** Read `walker/collectors.go:1-85` for `SchemaInfo`/`SchemaCollector`, and `parser/schema.go` for `Schema` struct.

**Step 1: Write failing tests**

Test filtering by name, component/inline, type, extension.

**Step 2: Implement `handleWalkSchemas`**

- Flags: `--name`, `--component`, `--inline`, `--type`, plus common flags
- Use `walker.CollectSchemas(result)`
- Summary columns: NAME, TYPE, PROPERTIES, COMPONENT/INLINE, EXTENSIONS

**Step 3: Remove stub, test, commit**

---

### Task 8: Parameters Subcommand

**Files:**
- Create: `cmd/oastools/commands/walk_parameters.go`
- Test: `cmd/oastools/commands/walk_parameters_test.go`

**Context needed:** Read the `ParameterCollector` from Task 3, and `parser/parameters.go:1-43` for `Parameter` struct.

**Step 1-5:** Same pattern. Flags: `--in`, `--name`, `--path`, `--method`. Summary columns: NAME, IN, REQUIRED, TYPE, PATH, METHOD, EXTENSIONS.

---

### Task 9: Responses Subcommand

**Files:**
- Create: `cmd/oastools/commands/walk_responses.go`
- Test: `cmd/oastools/commands/walk_responses_test.go`

**Context needed:** Read `ResponseCollector` from Task 3, and `parser/paths.go:110-121` for `Response` struct.

**Step 1-5:** Same pattern. Flags: `--status`, `--path`, `--method`. Summary columns: STATUS, DESCRIPTION, PATH, METHOD, EXTENSIONS.

---

### Task 10: Security Subcommand

**Files:**
- Create: `cmd/oastools/commands/walk_security.go`
- Test: `cmd/oastools/commands/walk_security_test.go`

**Context needed:** Read `SecuritySchemeCollector` from Task 3, and `parser/security.go:8-35` for `SecurityScheme` struct.

**Step 1-5:** Same pattern. Flags: `--name`, `--type` (apiKey, http, oauth2, openIdConnect). Summary columns: NAME, TYPE, SCHEME, IN, EXTENSIONS.

---

### Task 11: Paths Subcommand

**Files:**
- Create: `cmd/oastools/commands/walk_paths.go`
- Test: `cmd/oastools/commands/walk_paths_test.go`

**Context needed:** Read `parser/paths.go:14-34` for `PathItem` struct, and use `walker.Walk` with `WithPathHandler`.

**Step 1-5:** Same pattern. Flags: `--path` (glob filter). Summary columns: PATH, METHODS, SUMMARY, EXTENSIONS. Methods column is computed from non-nil operation fields on PathItem (Get, Put, Post, Delete, etc.).

---

### Task 12: Final Integration Test & Cleanup

After all subcommands are implemented.

**Files:**
- Remove stubs from `walk.go` if any remain
- Run `make check` to verify everything passes

**Step 1: Run full test suite**

```bash
make check
```

**Step 2: Test against real petstore files**

```bash
go run ./cmd/oastools walk operations testdata/petstore-3.0.yaml
go run ./cmd/oastools walk operations testdata/petstore-2.0.yaml
go run ./cmd/oastools walk schemas --component testdata/petstore-3.0.yaml
go run ./cmd/oastools walk schemas --component testdata/petstore-2.0.yaml
```

Verify same output shape for equivalent specs.

**Step 3: Test piping**

```bash
go run ./cmd/oastools walk operations -q --format json testdata/petstore-3.0.yaml | jq '.'
```

**Step 4: Commit any final fixes**

```bash
git commit -m "test(walk): add integration tests and cleanup"
```
