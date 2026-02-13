package commands

import (
	"bytes"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err)
	return collector.All
}

func TestFilterSecuritySchemes_All(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	matched, err := filterSecuritySchemes(schemes, "", "", "")
	require.NoError(t, err)
	assert.Len(t, matched, 3)
}

func TestFilterSecuritySchemes_ByName(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	matched, err := filterSecuritySchemes(schemes, "bearerAuth", "", "")
	require.NoError(t, err)
	require.Len(t, matched, 1)
	assert.Equal(t, "bearerAuth", matched[0].Name)
}

func TestFilterSecuritySchemes_ByType(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	matched, err := filterSecuritySchemes(schemes, "", "http", "")
	require.NoError(t, err)
	require.Len(t, matched, 1)
	assert.Equal(t, "http", matched[0].SecurityScheme.Type)
}

func TestFilterSecuritySchemes_ByTypeCaseInsensitive(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	matched, err := filterSecuritySchemes(schemes, "", "HTTP", "")
	require.NoError(t, err)
	require.Len(t, matched, 1)
	assert.Equal(t, "http", matched[0].SecurityScheme.Type)
}

func TestFilterSecuritySchemes_ByExtension(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	matched, err := filterSecuritySchemes(schemes, "", "", "x-scope=internal")
	require.NoError(t, err)
	require.Len(t, matched, 1)
	assert.Equal(t, "bearerAuth", matched[0].Name)
}

func TestFilterSecuritySchemes_NoMatch(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	matched, err := filterSecuritySchemes(schemes, "nonexistent", "", "")
	require.NoError(t, err)
	assert.Empty(t, matched)
}

func TestFilterSecuritySchemes_InvalidExtension(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	_, err := filterSecuritySchemes(schemes, "", "", "invalid-key")
	assert.Error(t, err)
}

func TestFilterSecuritySchemes_CombinedFilters(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	// Filter by type=http AND extension x-scope=internal
	matched, err := filterSecuritySchemes(schemes, "", "http", "x-scope=internal")
	require.NoError(t, err)
	require.Len(t, matched, 1)
	assert.Equal(t, "bearerAuth", matched[0].Name)
}

func TestFilterSecuritySchemes_CombinedNoMatch(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	// Filter by type=apiKey AND extension x-scope=internal (apiKey has no extensions)
	matched, err := filterSecuritySchemes(schemes, "", "apiKey", "x-scope=internal")
	require.NoError(t, err)
	assert.Empty(t, matched)
}

func TestHandleWalkSecurity_NoArgs(t *testing.T) {
	err := handleWalkSecurity([]string{})
	require.Error(t, err)
	assert.Equal(t, "walk security requires a spec file argument", err.Error())
}

func TestHandleWalkSecurity_InvalidFormat(t *testing.T) {
	err := handleWalkSecurity([]string{"--format", "xml", "spec.yaml"})
	assert.Error(t, err)
}

func TestRenderSecuritySummary(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)
	flags := WalkFlags{Format: FormatText}

	err := renderSecuritySummary(schemes, flags)
	assert.NoError(t, err)
}

func TestRenderSecuritySummary_Quiet(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)
	flags := WalkFlags{Format: FormatText, Quiet: true}

	err := renderSecuritySummary(schemes, flags)
	assert.NoError(t, err)
}

func TestRenderSecurityDetail(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)
	flags := WalkFlags{Format: FormatJSON}

	err := renderSecurityDetail(schemes, flags)
	assert.NoError(t, err)
}

func TestRenderSecurityDetail_YAML(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)
	flags := WalkFlags{Format: FormatYAML}

	err := renderSecurityDetail(schemes, flags)
	assert.NoError(t, err)
}

func TestRenderSecurityDetail_Text(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)
	flags := WalkFlags{Format: FormatText}

	err := renderSecurityDetail(schemes, flags)
	assert.NoError(t, err)
}

func TestRenderSecurityDetail_IncludesName(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	// Filter to bearerAuth
	matched, err := filterSecuritySchemes(schemes, "bearerAuth", "", "")
	require.NoError(t, err)
	require.Len(t, matched, 1)

	view := securityDetailView{
		Name:           matched[0].Name,
		SecurityScheme: matched[0].SecurityScheme,
	}

	var buf bytes.Buffer
	err = RenderDetail(&buf, view, FormatJSON)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"name"`)
	assert.Contains(t, output, "bearerAuth")
	assert.Contains(t, output, `"securityScheme"`)
	assert.Contains(t, output, "bearer")
}

func TestRenderSecurityDetail_IncludesNameYAML(t *testing.T) {
	schemes := collectTestSecuritySchemes(t)

	matched, err := filterSecuritySchemes(schemes, "apiKey", "", "")
	require.NoError(t, err)
	require.Len(t, matched, 1)

	view := securityDetailView{
		Name:           matched[0].Name,
		SecurityScheme: matched[0].SecurityScheme,
	}

	var buf bytes.Buffer
	err = RenderDetail(&buf, view, FormatYAML)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "name:")
	assert.Contains(t, output, "apiKey")
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
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"name"`)
	assert.Contains(t, output, `"type"`)
	assert.Contains(t, output, "bearerAuth")
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
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "name")
	assert.Contains(t, output, "bearerAuth")
}
