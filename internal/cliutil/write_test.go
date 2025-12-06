package cliutil

import (
	"bytes"
	"testing"
)

func TestWritef(t *testing.T) {
	var buf bytes.Buffer
	Writef(&buf, "Hello, %s!", "World")
	if got := buf.String(); got != "Hello, World!" {
		t.Errorf("Writef() = %q, want %q", got, "Hello, World!")
	}
}

func TestWritef_NoArgs(t *testing.T) {
	var buf bytes.Buffer
	Writef(&buf, "Simple message")
	if got := buf.String(); got != "Simple message" {
		t.Errorf("Writef() = %q, want %q", got, "Simple message")
	}
}

func TestWritef_MultipleArgs(t *testing.T) {
	var buf bytes.Buffer
	Writef(&buf, "%s: %d items, %v active", "Status", 42, true)
	want := "Status: 42 items, true active"
	if got := buf.String(); got != want {
		t.Errorf("Writef() = %q, want %q", got, want)
	}
}

// errorWriter is a writer that always returns an error
type errorWriter struct{}

func (e errorWriter) Write(p []byte) (n int, err error) {
	return 0, &writeError{}
}

type writeError struct{}

func (e *writeError) Error() string {
	return "simulated write error"
}

func TestWritef_WriteError(t *testing.T) {
	// This test verifies that Writef handles write errors gracefully
	// by logging to stderr rather than panicking
	var ew errorWriter
	// Should not panic
	Writef(ew, "This will fail")
}
