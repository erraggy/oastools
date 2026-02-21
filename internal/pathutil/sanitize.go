package pathutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// SanitizeOutputPath validates and cleans an output file path.
// It resolves ".." components via filepath.Clean + filepath.Abs and
// rejects paths that resolve to symlinks. New files in existing
// directories are accepted. Returns the cleaned absolute path.
func SanitizeOutputPath(path string) (string, error) {
	cleaned := filepath.Clean(path)

	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", fmt.Errorf("pathutil: cannot resolve absolute path: %w", err)
	}

	info, err := os.Lstat(abs)
	switch {
	case err == nil:
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("pathutil: refusing to write to symlink: %s", abs)
		}
	case os.IsNotExist(err):
		// New file — safe to proceed.
	default:
		return "", fmt.Errorf("pathutil: cannot stat path: %w", err)
	}

	return abs, nil
}
