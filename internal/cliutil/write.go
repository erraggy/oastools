// Package cliutil provides utilities for CLI operations.
package cliutil

import (
	"fmt"
	"io"
	"os"
)

// Writef writes formatted output to the writer.
// If the write fails, it logs to stderr (useful for debugging).
func Writef(w io.Writer, format string, args ...any) {
	if _, err := fmt.Fprintf(w, format, args...); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "write error: %v\n", err)
	}
}
