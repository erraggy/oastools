package commands

import (
	"os"
	"path/filepath"
	"testing"
)

// TestHandleValidate_ErrorPaths tests error handling for the validate command.
func TestHandleValidate_ErrorPaths(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		err := HandleValidate([]string{"/nonexistent/path/to/file.yaml"})
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("malformed YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		malformedFile := filepath.Join(tmpDir, "malformed.yaml")
		if err := os.WriteFile(malformedFile, []byte("not: valid: yaml: [unclosed"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		err := HandleValidate([]string{malformedFile})
		if err == nil {
			t.Error("expected error for malformed YAML")
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		malformedFile := filepath.Join(tmpDir, "malformed.json")
		if err := os.WriteFile(malformedFile, []byte(`{"unclosed": `), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		err := HandleValidate([]string{malformedFile})
		if err == nil {
			t.Error("expected error for malformed JSON")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		emptyFile := filepath.Join(tmpDir, "empty.yaml")
		if err := os.WriteFile(emptyFile, []byte(""), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		err := HandleValidate([]string{emptyFile})
		if err == nil {
			t.Error("expected error for empty file")
		}
	})

	t.Run("non-OpenAPI content", func(t *testing.T) {
		tmpDir := t.TempDir()
		nonOASFile := filepath.Join(tmpDir, "not-oas.yaml")
		content := `name: just a random yaml file
items:
  - one
  - two
`
		if err := os.WriteFile(nonOASFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		err := HandleValidate([]string{nonOASFile})
		if err == nil {
			t.Error("expected error for non-OpenAPI content")
		}
	})
}

// TestHandleParse_ErrorPaths tests error handling for the parse command.
func TestHandleParse_ErrorPaths(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		err := HandleParse([]string{"/nonexistent/path/to/file.yaml"})
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("malformed YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		malformedFile := filepath.Join(tmpDir, "malformed.yaml")
		if err := os.WriteFile(malformedFile, []byte("not: valid: yaml: [unclosed"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		err := HandleParse([]string{malformedFile})
		if err == nil {
			t.Error("expected error for malformed YAML")
		}
	})

	t.Run("empty file", func(t *testing.T) {
		tmpDir := t.TempDir()
		emptyFile := filepath.Join(tmpDir, "empty.yaml")
		if err := os.WriteFile(emptyFile, []byte(""), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		err := HandleParse([]string{emptyFile})
		if err == nil {
			t.Error("expected error for empty file")
		}
	})
}

// TestHandleFix_ErrorPaths tests error handling for the fix command.
func TestHandleFix_ErrorPaths(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		err := HandleFix([]string{"/nonexistent/path/to/file.yaml"})
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})

	t.Run("malformed YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		malformedFile := filepath.Join(tmpDir, "malformed.yaml")
		if err := os.WriteFile(malformedFile, []byte("not: valid: yaml: [unclosed"), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		err := HandleFix([]string{malformedFile})
		if err == nil {
			t.Error("expected error for malformed YAML")
		}
	})
}

// TestHandleConvert_ErrorPaths tests error handling for the convert command.
func TestHandleConvert_ErrorPaths(t *testing.T) {
	t.Run("non-existent file", func(t *testing.T) {
		err := HandleConvert([]string{"--to", "3.0", "/nonexistent/path/to/file.yaml"})
		if err == nil {
			t.Error("expected error for non-existent file")
		}
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
		if err := os.WriteFile(validFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		err := HandleConvert([]string{"--to", "invalid", validFile})
		if err == nil {
			t.Error("expected error for invalid target version")
		}
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
		if err := os.WriteFile(validFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		err := HandleConvert([]string{validFile})
		if err == nil {
			t.Error("expected error when target version not specified")
		}
	})
}

// TestHandleJoin_ErrorPaths tests error handling for the join command.
func TestHandleJoin_ErrorPaths(t *testing.T) {
	t.Run("no files provided", func(t *testing.T) {
		err := HandleJoin([]string{})
		if err == nil {
			t.Error("expected error when no files provided")
		}
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
		if err := os.WriteFile(validFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		err := HandleJoin([]string{validFile})
		if err == nil {
			t.Error("expected error when only one file provided")
		}
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
		if err := os.WriteFile(validFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		err := HandleJoin([]string{validFile, "/nonexistent/path.yaml"})
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})
}

// TestHandleDiff_ErrorPaths tests error handling for the diff command.
func TestHandleDiff_ErrorPaths(t *testing.T) {
	t.Run("no files provided", func(t *testing.T) {
		err := HandleDiff([]string{})
		if err == nil {
			t.Error("expected error when no files provided")
		}
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
		if err := os.WriteFile(validFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		err := HandleDiff([]string{validFile})
		if err == nil {
			t.Error("expected error when only one file provided")
		}
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
		if err := os.WriteFile(validFile, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		err := HandleDiff([]string{validFile, "/nonexistent/path.yaml"})
		if err == nil {
			t.Error("expected error for non-existent file")
		}
	})
}
