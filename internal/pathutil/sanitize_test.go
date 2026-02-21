package pathutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeOutputPath(t *testing.T) {
	t.Run("clean path accepted", func(t *testing.T) {
		tmpDir := t.TempDir()
		target := filepath.Join(tmpDir, "output.yaml")

		// Create the file so it exists
		require.NoError(t, os.WriteFile(target, []byte("test"), 0o600))

		got, err := SanitizeOutputPath(target)
		require.NoError(t, err)
		assert.Equal(t, target, got)
	})

	t.Run("returns absolute path from relative", func(t *testing.T) {
		// Use a relative path that will be resolved
		got, err := SanitizeOutputPath("output.yaml")
		require.NoError(t, err)
		assert.True(t, filepath.IsAbs(got), "expected absolute path, got %s", got)
	})

	t.Run("dot-dot path resolved by filepath.Abs", func(t *testing.T) {
		// filepath.Clean + filepath.Abs resolves ".." traversal on all platforms.
		// The function relies on this rather than string matching (which would
		// false-positive on legitimate paths like /data/my..config/).
		got, err := SanitizeOutputPath("/tmp/../etc/passwd")
		require.NoError(t, err)
		assert.NotContains(t, got, "..", "filepath.Abs should resolve all '..' components")
	})

	t.Run("symlink target rejected", func(t *testing.T) {
		tmpDir := t.TempDir()
		realFile := filepath.Join(tmpDir, "real.yaml")
		linkFile := filepath.Join(tmpDir, "link.yaml")

		require.NoError(t, os.WriteFile(realFile, []byte("test"), 0o600))
		require.NoError(t, os.Symlink(realFile, linkFile))

		_, err := SanitizeOutputPath(linkFile)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "symlink")
	})

	t.Run("new file in existing directory accepted", func(t *testing.T) {
		tmpDir := t.TempDir()
		newFile := filepath.Join(tmpDir, "newfile.yaml")

		got, err := SanitizeOutputPath(newFile)
		require.NoError(t, err)
		assert.Equal(t, newFile, got)
	})

	t.Run("regular file accepted", func(t *testing.T) {
		tmpDir := t.TempDir()
		target := filepath.Join(tmpDir, "existing.json")
		require.NoError(t, os.WriteFile(target, []byte("{}"), 0o600))

		got, err := SanitizeOutputPath(target)
		require.NoError(t, err)
		assert.Equal(t, target, got)
	})

	t.Run("directory accepted", func(t *testing.T) {
		tmpDir := t.TempDir()

		got, err := SanitizeOutputPath(tmpDir)
		require.NoError(t, err)
		assert.Equal(t, tmpDir, got)
	})

	t.Run("symlink directory rejected", func(t *testing.T) {
		tmpDir := t.TempDir()
		realDir := filepath.Join(tmpDir, "realdir")
		linkDir := filepath.Join(tmpDir, "linkdir")

		require.NoError(t, os.Mkdir(realDir, 0o755))
		require.NoError(t, os.Symlink(realDir, linkDir))

		_, err := SanitizeOutputPath(linkDir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "symlink")
	})

	t.Run("lstat permission error fails closed", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("test requires non-root user")
		}
		tmpDir := t.TempDir()
		noAccessDir := filepath.Join(tmpDir, "noaccess")
		require.NoError(t, os.Mkdir(noAccessDir, 0o000))
		t.Cleanup(func() { _ = os.Chmod(noAccessDir, 0o755) })

		target := filepath.Join(noAccessDir, "file.yaml")
		_, err := SanitizeOutputPath(target)
		require.Error(t, err, "should fail closed when Lstat returns a permission error")
		assert.Contains(t, err.Error(), "cannot stat path")
	})
}
