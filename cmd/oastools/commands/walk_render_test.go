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
