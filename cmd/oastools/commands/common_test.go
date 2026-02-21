package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
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
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
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
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMarshalDocument(t *testing.T) {
	doc := map[string]string{"key": "value"}

	t.Run("json format", func(t *testing.T) {
		data, err := MarshalDocument(doc, parser.SourceFormatJSON)
		require.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("yaml format", func(t *testing.T) {
		data, err := MarshalDocument(doc, parser.SourceFormatYAML)
		require.NoError(t, err)
		assert.NotEmpty(t, data)
	})
}

func TestOutputStructured(t *testing.T) {
	data := map[string]string{"test": "value"}

	t.Run("invalid format", func(t *testing.T) {
		err := OutputStructured(data, "invalid")
		assert.Error(t, err)
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
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWritef(t *testing.T) {
	var buf bytes.Buffer
	Writef(&buf, "Hello, %s!", "World")
	assert.Equal(t, "Hello, World!", buf.String())
}

func TestWritef_NoArgs(t *testing.T) {
	var buf bytes.Buffer
	Writef(&buf, "Simple message")
	assert.Equal(t, "Simple message", buf.String())
}

func TestWritef_MultipleArgs(t *testing.T) {
	var buf bytes.Buffer
	Writef(&buf, "%s: %d items, %v active", "Status", 42, true)
	want := "Status: 42 items, true active"
	assert.Equal(t, want, buf.String())
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

func TestRejectSymlinkOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require elevated privileges on Windows")
	}

	t.Run("non-existent path is allowed", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonExistent := filepath.Join(tmpDir, "does-not-exist.yaml")
		err := RejectSymlinkOutput(nonExistent)
		assert.NoError(t, err)
	})

	t.Run("regular file is allowed", func(t *testing.T) {
		tmpDir := t.TempDir()
		regularFile := filepath.Join(tmpDir, "regular.yaml")
		require.NoError(t, os.WriteFile(regularFile, []byte("test"), 0600))

		err := RejectSymlinkOutput(regularFile)
		assert.NoError(t, err)
	})

	t.Run("symlink is rejected", func(t *testing.T) {
		tmpDir := t.TempDir()
		target := filepath.Join(tmpDir, "target.yaml")
		require.NoError(t, os.WriteFile(target, []byte("test"), 0600))

		symlinkPath := filepath.Join(tmpDir, "symlink.yaml")
		require.NoError(t, os.Symlink(target, symlinkPath))

		err := RejectSymlinkOutput(symlinkPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "refusing to write to symlink")
		assert.Contains(t, err.Error(), symlinkPath)
	})

	t.Run("symlink to non-existent target is rejected", func(t *testing.T) {
		tmpDir := t.TempDir()
		symlinkPath := filepath.Join(tmpDir, "dangling-symlink.yaml")
		require.NoError(t, os.Symlink("/nonexistent/target", symlinkPath))

		err := RejectSymlinkOutput(symlinkPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "refusing to write to symlink")
	})
}
