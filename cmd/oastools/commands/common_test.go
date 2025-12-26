package commands

import (
	"bytes"
	"testing"

	"github.com/erraggy/oastools/parser"
)

func TestValidateOutputFormat(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantErr bool
	}{
		{"valid text", FormatText, false},
		{"valid json", FormatJSON, false},
		{"valid yaml", FormatYAML, false},
		{"invalid format", "xml", true},
		{"empty format", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOutputFormat(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOutputFormat(%q) error = %v, wantErr %v", tt.format, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCollisionStrategy(t *testing.T) {
	tests := []struct {
		name         string
		strategyName string
		value        string
		wantErr      bool
	}{
		{"empty value", "path-strategy", "", false},
		{"valid accept-left", "path-strategy", "accept-left", false},
		{"valid accept-right", "schema-strategy", "accept-right", false},
		{"valid fail", "component-strategy", "fail", false},
		{"invalid strategy", "path-strategy", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCollisionStrategy(tt.strategyName, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCollisionStrategy(%q, %q) error = %v, wantErr %v", tt.strategyName, tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestValidateEquivalenceMode(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty value", "", false},
		{"valid none", "none", false},
		{"valid shallow", "shallow", false},
		{"valid deep", "deep", false},
		{"invalid mode", "invalid", true},
		{"case sensitive DEEP", "DEEP", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEquivalenceMode(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEquivalenceMode(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestMarshalDocument(t *testing.T) {
	doc := map[string]string{"key": "value"}

	t.Run("json format", func(t *testing.T) {
		data, err := MarshalDocument(doc, parser.SourceFormatJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) == 0 {
			t.Error("expected non-empty output")
		}
	})

	t.Run("yaml format", func(t *testing.T) {
		data, err := MarshalDocument(doc, parser.SourceFormatYAML)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) == 0 {
			t.Error("expected non-empty output")
		}
	})
}

func TestOutputStructured(t *testing.T) {
	data := map[string]string{"test": "value"}

	t.Run("invalid format", func(t *testing.T) {
		err := OutputStructured(data, "invalid")
		if err == nil {
			t.Error("expected error for invalid format")
		}
	})
}

func TestFormatSpecPath(t *testing.T) {
	tests := []struct {
		name     string
		specPath string
		want     string
	}{
		{"stdin path", StdinFilePath, "<stdin>"},
		{"normal file path", "/path/to/openapi.yaml", "/path/to/openapi.yaml"},
		{"relative path", "api/spec.json", "api/spec.json"},
		{"empty path", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSpecPath(tt.specPath)
			if got != tt.want {
				t.Errorf("FormatSpecPath(%q) = %q, want %q", tt.specPath, got, tt.want)
			}
		})
	}
}

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

func (e errorWriter) Write(_ []byte) (n int, err error) {
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
