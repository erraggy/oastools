package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
)

func testSecurityParseResult() *parser.ParseResult {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"bearerAuth": {Type: "http", Scheme: "bearer", BearerFormat: "JWT", Extra: map[string]any{"x-scope": "internal"}},
				"apiKey":     {Type: "apiKey", Name: "X-API-Key", In: "header"},
				"oauth":      {Type: "oauth2"},
			},
		},
	}
	return &parser.ParseResult{Document: doc, Version: "3.0.3"}
}

func collectTestSecuritySchemes(t *testing.T) []*walker.SecuritySchemeInfo {
	t.Helper()
	result := testSecurityParseResult()
	collector, err := walker.CollectSecuritySchemes(result)
	if err != nil {
		t.Fatalf("collecting security schemes: %v", err)
	}
	return collector.All
}

func TestFilterSecuritySchemes_All(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	matched, err := filterSecuritySchemes(schemes, "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matched) != 3 {
		t.Errorf("expected 3 schemes, got %d", len(matched))
	}
}

func TestFilterSecuritySchemes_ByName(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	matched, err := filterSecuritySchemes(schemes, "bearerAuth", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matched) != 1 {
		t.Fatalf("expected 1 scheme, got %d", len(matched))
	}
	if matched[0].Name != "bearerAuth" {
		t.Errorf("expected name bearerAuth, got %s", matched[0].Name)
	}
}

func TestFilterSecuritySchemes_ByType(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	matched, err := filterSecuritySchemes(schemes, "", "http", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matched) != 1 {
		t.Fatalf("expected 1 scheme, got %d", len(matched))
	}
	if matched[0].SecurityScheme.Type != "http" {
		t.Errorf("expected type http, got %s", matched[0].SecurityScheme.Type)
	}
}

func TestFilterSecuritySchemes_ByTypeCaseInsensitive(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	matched, err := filterSecuritySchemes(schemes, "", "HTTP", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matched) != 1 {
		t.Fatalf("expected 1 scheme, got %d", len(matched))
	}
	if matched[0].SecurityScheme.Type != "http" {
		t.Errorf("expected type http, got %s", matched[0].SecurityScheme.Type)
	}
}

func TestFilterSecuritySchemes_ByExtension(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	matched, err := filterSecuritySchemes(schemes, "", "", "x-scope=internal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matched) != 1 {
		t.Fatalf("expected 1 scheme, got %d", len(matched))
	}
	if matched[0].Name != "bearerAuth" {
		t.Errorf("expected name bearerAuth, got %s", matched[0].Name)
	}
}

func TestFilterSecuritySchemes_NoMatch(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	matched, err := filterSecuritySchemes(schemes, "nonexistent", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matched) != 0 {
		t.Errorf("expected 0 schemes, got %d", len(matched))
	}
}

func TestFilterSecuritySchemes_InvalidExtension(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	_, err := filterSecuritySchemes(schemes, "", "", "invalid-key")
	if err == nil {
		t.Error("expected error for invalid extension filter")
	}
}

func TestFilterSecuritySchemes_CombinedFilters(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	// Filter by type=http AND extension x-scope=internal
	matched, err := filterSecuritySchemes(schemes, "", "http", "x-scope=internal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matched) != 1 {
		t.Fatalf("expected 1 scheme, got %d", len(matched))
	}
	if matched[0].Name != "bearerAuth" {
		t.Errorf("expected name bearerAuth, got %s", matched[0].Name)
	}
}

func TestFilterSecuritySchemes_CombinedNoMatch(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	// Filter by type=apiKey AND extension x-scope=internal (apiKey has no extensions)
	matched, err := filterSecuritySchemes(schemes, "", "apiKey", "x-scope=internal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matched) != 0 {
		t.Errorf("expected 0 schemes, got %d", len(matched))
	}
}

func TestHandleWalkSecurity_NoArgs(t *testing.T) {
	err := handleWalkSecurity([]string{})
	if err == nil {
		t.Error("expected error when no spec file provided")
	}
	expected := "walk security requires a spec file argument"
	if err.Error() != expected {
		t.Errorf("expected error %q, got %q", expected, err.Error())
	}
}

func TestHandleWalkSecurity_InvalidFormat(t *testing.T) {
	err := handleWalkSecurity([]string{"--format", "xml", "spec.yaml"})
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestRenderSecuritySummary(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)
	flags := WalkFlags{Format: FormatText}

	// Should not return error
	err := renderSecuritySummary(schemes, flags)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRenderSecuritySummary_Quiet(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)
	flags := WalkFlags{Format: FormatText, Quiet: true}

	err := renderSecuritySummary(schemes, flags)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRenderSecurityDetail(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)
	flags := WalkFlags{Format: FormatJSON}

	err := renderSecurityDetail(schemes, flags)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRenderSecurityDetail_YAML(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)
	flags := WalkFlags{Format: FormatYAML}

	err := renderSecurityDetail(schemes, flags)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRenderSecurityDetail_Text(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)
	flags := WalkFlags{Format: FormatText}

	err := renderSecurityDetail(schemes, flags)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRenderSecurityDetail_IncludesName(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	// Filter to bearerAuth
	matched, err := filterSecuritySchemes(schemes, "bearerAuth", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matched) != 1 {
		t.Fatalf("expected 1 scheme, got %d", len(matched))
	}

	view := securityDetailView{
		Name:           matched[0].Name,
		SecurityScheme: matched[0].SecurityScheme,
	}

	var buf bytes.Buffer
	err = RenderDetail(&buf, view, FormatJSON)
	if err != nil {
		t.Fatalf("RenderDetail failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, `"name"`) {
		t.Error("expected 'name' key in detail JSON output")
	}
	if !strings.Contains(output, "bearerAuth") {
		t.Error("expected bearerAuth name in detail output")
	}
	if !strings.Contains(output, `"securityScheme"`) {
		t.Error("expected 'securityScheme' key in detail output")
	}
	if !strings.Contains(output, "bearer") {
		t.Error("expected bearer scheme in detail output")
	}
}

func TestRenderSecurityDetail_IncludesNameYAML(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	matched, err := filterSecuritySchemes(schemes, "apiKey", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matched) != 1 {
		t.Fatalf("expected 1 scheme, got %d", len(matched))
	}

	view := securityDetailView{
		Name:           matched[0].Name,
		SecurityScheme: matched[0].SecurityScheme,
	}

	var buf bytes.Buffer
	err = RenderDetail(&buf, view, FormatYAML)
	if err != nil {
		t.Fatalf("RenderDetail failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "name:") {
		t.Error("expected 'name' key in YAML detail output")
	}
	if !strings.Contains(output, "apiKey") {
		t.Error("expected apiKey name in YAML detail output")
	}
}

func TestRenderSecuritySummary_JSON(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	headers := []string{"NAME", "TYPE", "SCHEME", "IN", "EXTENSIONS"}
	rows := make([][]string, 0, len(schemes))
	for _, info := range schemes {
		rows = append(rows, []string{
			info.Name,
			info.SecurityScheme.Type,
			info.SecurityScheme.Scheme,
			info.SecurityScheme.In,
			FormatExtensions(info.SecurityScheme.Extra),
		})
	}

	var buf bytes.Buffer
	err := RenderSummaryStructured(&buf, headers, rows, FormatJSON)
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
	if !strings.Contains(output, "bearerAuth") {
		t.Error("expected bearerAuth in JSON summary output")
	}
}

func TestRenderSecuritySummary_YAML(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	headers := []string{"NAME", "TYPE", "SCHEME", "IN", "EXTENSIONS"}
	rows := make([][]string, 0, len(schemes))
	for _, info := range schemes {
		rows = append(rows, []string{
			info.Name,
			info.SecurityScheme.Type,
			info.SecurityScheme.Scheme,
			info.SecurityScheme.In,
			FormatExtensions(info.SecurityScheme.Extra),
		})
	}

	var buf bytes.Buffer
	err := RenderSummaryStructured(&buf, headers, rows, FormatYAML)
	if err != nil {
		t.Fatalf("RenderSummaryStructured failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "name") {
		t.Error("expected 'name' key in YAML summary output")
	}
	if !strings.Contains(output, "bearerAuth") {
		t.Error("expected bearerAuth in YAML summary output")
	}
}
