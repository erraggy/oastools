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

func TestRenderSummaryStructured_JSON(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"METHOD", "PATH", "SUMMARY"}
	rows := [][]string{
		{"GET", "/pets", "List pets"},
		{"POST", "/pets", "Create pet"},
	}

	err := RenderSummaryStructured(&buf, headers, rows, FormatJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()

	// Should contain lowercase keys
	if !strings.Contains(output, `"method"`) {
		t.Error("expected lowercase 'method' key in JSON output")
	}
	if !strings.Contains(output, `"path"`) {
		t.Error("expected lowercase 'path' key in JSON output")
	}
	// Should contain data values
	if !strings.Contains(output, "GET") {
		t.Error("expected GET in JSON output")
	}
	if !strings.Contains(output, "/pets") {
		t.Error("expected /pets in JSON output")
	}
	if !strings.Contains(output, "List pets") {
		t.Error("expected 'List pets' in JSON output")
	}
}

func TestRenderSummaryStructured_YAML(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"METHOD", "PATH"}
	rows := [][]string{
		{"GET", "/pets"},
	}

	err := RenderSummaryStructured(&buf, headers, rows, FormatYAML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()

	if !strings.Contains(output, "method") {
		t.Error("expected 'method' key in YAML output")
	}
	if !strings.Contains(output, "GET") {
		t.Error("expected GET in YAML output")
	}
}

func TestRenderSummaryStructured_EmptyRows(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"A", "B"}
	err := RenderSummaryStructured(&buf, headers, nil, FormatJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Empty rows should produce an empty JSON array
	if !strings.Contains(buf.String(), "[]") {
		t.Errorf("expected empty array, got %q", buf.String())
	}
}

func TestRenderSummaryStructured_RowShorterThanHeaders(t *testing.T) {
	var buf bytes.Buffer
	headers := []string{"A", "B", "C"}
	rows := [][]string{
		{"val1"},
	}

	err := RenderSummaryStructured(&buf, headers, rows, FormatJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()

	// Missing columns should default to empty string
	if !strings.Contains(output, `"b"`) {
		t.Error("expected key 'b' even when row is shorter than headers")
	}
	if !strings.Contains(output, `"c"`) {
		t.Error("expected key 'c' even when row is shorter than headers")
	}
}

func TestRenderDetail_YAML(t *testing.T) {
	var buf bytes.Buffer
	node := map[string]any{
		"summary": "List pets",
		"tags":    []string{"pets"},
	}

	err := RenderDetail(&buf, node, FormatYAML)
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

	err := RenderDetail(&buf, node, FormatJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, `"summary"`) {
		t.Error("expected summary in JSON output")
	}
}
