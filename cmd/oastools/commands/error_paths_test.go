package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHandleValidate_ErrorPaths tests error handling for the validate command.
func TestHandleValidate_ErrorPaths(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		err := HandleValidate([]string{"/nonexistent/path/to/file.yaml"})
		assert.Error(t, err)
	})

	t.Run("malformed YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		malformedFile := filepath.Join(tmpDir, "malformed.yaml")
		require.NoError(t, os.WriteFile(malformedFile, []byte("not: valid: yaml: [unclosed"), 0644))
		err := HandleValidate([]string{malformedFile})
		assert.Error(t, err)
	})

	t.Run("malformed JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		malformedFile := filepath.Join(tmpDir, "malformed.json")
		require.NoError(t, os.WriteFile(malformedFile, []byte(`{"unclosed": `), 0644))
		err := HandleValidate([]string{malformedFile})
		assert.Error(t, err)
	})

	t.Run("empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		emptyFile := filepath.Join(tmpDir, "empty.yaml")
		require.NoError(t, os.WriteFile(emptyFile, []byte(""), 0644))
		err := HandleValidate([]string{emptyFile})
		assert.Error(t, err)
	})

	t.Run("non-OpenAPI content", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonOASFile := filepath.Join(tmpDir, "not-oas.yaml")
		content := `name: just a random yaml file
items:
  - one
  - two
`
		require.NoError(t, os.WriteFile(nonOASFile, []byte(content), 0644))
		err := HandleValidate([]string{nonOASFile})
		assert.Error(t, err)
	})
}

// TestHandleParse_ErrorPaths tests error handling for the parse command.
func TestHandleParse_ErrorPaths(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		err := HandleParse([]string{"/nonexistent/path/to/file.yaml"})
		assert.Error(t, err)
	})

	t.Run("malformed YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		malformedFile := filepath.Join(tmpDir, "malformed.yaml")
		require.NoError(t, os.WriteFile(malformedFile, []byte("not: valid: yaml: [unclosed"), 0644))
		err := HandleParse([]string{malformedFile})
		assert.Error(t, err)
	})

	t.Run("empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		emptyFile := filepath.Join(tmpDir, "empty.yaml")
		require.NoError(t, os.WriteFile(emptyFile, []byte(""), 0644))
		err := HandleParse([]string{emptyFile})
		assert.Error(t, err)
	})
}

// TestHandleFix_ErrorPaths tests error handling for the fix command.
func TestHandleFix_ErrorPaths(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		err := HandleFix([]string{"/nonexistent/path/to/file.yaml"})
		assert.Error(t, err)
	})

	t.Run("malformed YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		malformedFile := filepath.Join(tmpDir, "malformed.yaml")
		require.NoError(t, os.WriteFile(malformedFile, []byte("not: valid: yaml: [unclosed"), 0644))
		err := HandleFix([]string{malformedFile})
		assert.Error(t, err)
	})
}

// TestHandleConvert_ErrorPaths tests error handling for the convert command.
func TestHandleConvert_ErrorPaths(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		err := HandleConvert([]string{"--to", "3.0", "/nonexistent/path/to/file.yaml"})
		assert.Error(t, err)
	})

	t.Run("invalid target version", func(t *testing.T) {
		tmpDir := t.TempDir()
		validFile := filepath.Join(tmpDir, "valid.yaml")
		content := `openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`
		require.NoError(t, os.WriteFile(validFile, []byte(content), 0644))
		err := HandleConvert([]string{"--to", "invalid", validFile})
		assert.Error(t, err)
	})

	t.Run("missing target version", func(t *testing.T) {
		tmpDir := t.TempDir()
		validFile := filepath.Join(tmpDir, "valid.yaml")
		content := `openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`
		require.NoError(t, os.WriteFile(validFile, []byte(content), 0644))
		err := HandleConvert([]string{validFile})
		assert.Error(t, err)
	})
}

// TestHandleJoin_ErrorPaths tests error handling for the join command.
func TestHandleJoin_ErrorPaths(t *testing.T) {
	t.Run("no files provided", func(t *testing.T) {
		err := HandleJoin([]string{})
		assert.Error(t, err)
	})

	t.Run("single file provided", func(t *testing.T) {
		tmpDir := t.TempDir()
		validFile := filepath.Join(tmpDir, "valid.yaml")
		content := `openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`
		require.NoError(t, os.WriteFile(validFile, []byte(content), 0644))
		err := HandleJoin([]string{validFile})
		assert.Error(t, err)
	})

	t.Run("non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()
		validFile := filepath.Join(tmpDir, "valid.yaml")
		content := `openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`
		require.NoError(t, os.WriteFile(validFile, []byte(content), 0644))
		err := HandleJoin([]string{validFile, "/nonexistent/path.yaml"})
		assert.Error(t, err)
	})
}

// TestHandleDiff_ErrorPaths tests error handling for the diff command.
func TestHandleDiff_ErrorPaths(t *testing.T) {
	t.Run("no files provided", func(t *testing.T) {
		err := HandleDiff([]string{})
		assert.Error(t, err)
	})

	t.Run("single file provided", func(t *testing.T) {
		tmpDir := t.TempDir()
		validFile := filepath.Join(tmpDir, "valid.yaml")
		content := `openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`
		require.NoError(t, os.WriteFile(validFile, []byte(content), 0644))
		err := HandleDiff([]string{validFile})
		assert.Error(t, err)
	})

	t.Run("non-existent file", func(t *testing.T) {
		tmpDir := t.TempDir()
		validFile := filepath.Join(tmpDir, "valid.yaml")
		content := `openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths: {}
`
		require.NoError(t, os.WriteFile(validFile, []byte(content), 0644))
		err := HandleDiff([]string{validFile, "/nonexistent/path.yaml"})
		assert.Error(t, err)
	})
}
