package generator

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/erraggy/oastools/internal/fileutil"
)

// WriteFiles writes all generated files to the specified output directory.
// The directory is created if it doesn't exist.
func (r *GenerateResult) WriteFiles(outputDir string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	for _, file := range r.Files {
		safeName := filepath.Base(file.Name)
		if safeName != file.Name {
			return fmt.Errorf("invalid file name %q: must not contain path separators", file.Name)
		}
		filePath := filepath.Join(outputDir, safeName)
		if err := os.WriteFile(filePath, file.Content, fileutil.ReadableByAll); err != nil {
			return fmt.Errorf("failed to write file %s: %w", file.Name, err)
		}
	}

	return nil
}

// WriteFile writes a single generated file to the specified path.
func (f *GeneratedFile) WriteFile(path string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, f.Content, fileutil.ReadableByAll); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
