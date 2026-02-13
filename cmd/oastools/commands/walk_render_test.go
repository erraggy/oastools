package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Contains(t, output, "METHOD")
	// Should contain data
	assert.Contains(t, output, "GET")
	assert.Contains(t, output, "/pets")
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
	assert.NotContains(t, output, "METHOD")
	// Should still contain data
	assert.Contains(t, output, "GET")
}

func TestRenderSummaryTable_Empty(t *testing.T) {
	var buf bytes.Buffer
	RenderSummaryTable(&buf, []string{"A"}, nil, false)
	assert.Equal(t, 0, buf.Len())
}

func TestRenderSummaryStructured_JSON(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"METHOD", "PATH", "SUMMARY"}
	rows := [][]string{
		{"GET", "/pets", "List pets"},
		{"POST", "/pets", "Create pet"},
	}

	err := RenderSummaryStructured(&buf, headers, rows, FormatJSON)
	require.NoError(t, err)
	output := buf.String()

	// Should contain lowercase keys
	assert.Contains(t, output, `"method"`)
	assert.Contains(t, output, `"path"`)
	// Should contain data values
	assert.Contains(t, output, "GET")
	assert.Contains(t, output, "/pets")
	assert.Contains(t, output, "List pets")
}

func TestRenderSummaryStructured_YAML(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"METHOD", "PATH"}
	rows := [][]string{
		{"GET", "/pets"},
	}

	err := RenderSummaryStructured(&buf, headers, rows, FormatYAML)
	require.NoError(t, err)
	output := buf.String()

	assert.Contains(t, output, "method")
	assert.Contains(t, output, "GET")
}

func TestRenderSummaryStructured_EmptyRows(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"A", "B"}
	err := RenderSummaryStructured(&buf, headers, nil, FormatJSON)
	require.NoError(t, err)
	// Empty rows should produce an empty JSON array
	assert.Contains(t, buf.String(), "[]")
}

func TestRenderSummaryStructured_RowShorterThanHeaders(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"A", "B", "C"}
	rows := [][]string{
		{"val1"},
	}

	err := RenderSummaryStructured(&buf, headers, rows, FormatJSON)
	require.NoError(t, err)
	output := buf.String()

	// Missing columns should default to empty string
	assert.Contains(t, output, `"b"`)
	assert.Contains(t, output, `"c"`)
}

func TestRenderDetail_YAML(t *testing.T) {
	var buf bytes.Buffer
	node := map[string]any{
		"summary": "List pets",
		"tags":    []string{"pets"},
	}

	err := RenderDetail(&buf, node, FormatYAML)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "summary")
}

func TestRenderDetail_JSON(t *testing.T) {
	var buf bytes.Buffer
	node := map[string]any{
		"summary": "List pets",
	}

	err := RenderDetail(&buf, node, FormatJSON)
	require.NoError(t, err)
	output := buf.String()
	_ = strings.Contains(output, `"summary"`) // keep strings import used
	assert.Contains(t, output, `"summary"`)
}
