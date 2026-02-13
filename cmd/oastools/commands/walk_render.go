package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"go.yaml.in/yaml/v4"
)

// RenderSummaryTable renders a table of results.
// In quiet mode, headers are omitted and rows are tab-separated for piping.
// In normal mode, a fixed-width table with headers is rendered.
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
				_, _ = fmt.Fprint(w, "  ")
			}
			_, _ = fmt.Fprintf(w, "%-*s", widths[i], h)
		}
		_, _ = fmt.Fprintln(w)
	}

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if quiet {
				if i > 0 {
					_, _ = fmt.Fprint(w, "\t")
				}
				_, _ = fmt.Fprint(w, cell)
			} else {
				if i > 0 {
					_, _ = fmt.Fprint(w, "  ")
				}
				_, _ = fmt.Fprintf(w, "%-*s", widths[i], cell)
			}
		}
		_, _ = fmt.Fprintln(w)
	}
}

// RenderSummaryStructured renders summary table data as structured output (JSON or YAML).
// It converts header+row pairs into []map[string]string with lowercase keys.
func RenderSummaryStructured(w io.Writer, headers []string, rows [][]string, format string) error {
	records := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		rec := make(map[string]string, len(headers))
		for i, h := range headers {
			val := ""
			if i < len(row) {
				val = row[i]
			}
			rec[strings.ToLower(h)] = val
		}
		records = append(records, rec)
	}
	return RenderDetail(w, records, format)
}

// RenderDetail renders a single node in the specified format (JSON, YAML, or text).
func RenderDetail(w io.Writer, node any, format string) error {
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

	if _, err := fmt.Fprintln(w, strings.TrimRight(string(data), "\n")); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}
	return nil
}
