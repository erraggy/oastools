package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// SanitizeOutputPath validates and cleans an output file path.
// It rejects paths containing ".." after cleaning and paths that
// resolve to symlinks. New files in existing directories are accepted.
// Returns the cleaned absolute path.
func SanitizeOutputPath(path string) (string, error) {
	cleaned := filepath.Clean(path)

	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("pathutil: cannot resolve absolute path: %w", err)
	}

	info, err := os.Lstat(abs)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("pathutil: refusing to write to symlink: %s", abs)
		}
	}

	return abs, nil
}
