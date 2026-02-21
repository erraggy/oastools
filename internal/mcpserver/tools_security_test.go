package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalOAS30Spec is a minimal valid OAS 3.0 spec for security tests.
const minimalOAS30Spec = `openapi: "3.0.0"
info:
  title: Test
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
      responses:
        "200":
          description: OK
`

func TestFixTool_OutputPathSymlinkRejected(t *testing.T) {
	tmpDir := t.TempDir()
	realFile := filepath.Join(tmpDir, "real.yaml")
	linkFile := filepath.Join(tmpDir, "link.yaml")

	require.NoError(t, os.WriteFile(realFile, []byte("placeholder"), 0o600))
	require.NoError(t, os.Symlink(realFile, linkFile))

	input := fixInput{
		Spec:            specInput{Content: minimalOAS30Spec},
		Output:          linkFile,
		IncludeDocument: true,
	}
	result, _, err := handleFix(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError, "expected error for symlink output path")
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "path")
}

func TestFixTool_OutputPathNormalized(t *testing.T) {
	// Verify that the output path is cleaned and normalized to an absolute path.
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub")
	require.NoError(t, os.Mkdir(subDir, 0o755))

	// Use a path with redundant components that cleans to a valid location.
	messyPath := filepath.Join(subDir, ".", "output.yaml")

	input := fixInput{
		Spec:            specInput{Content: minimalOAS30Spec},
		Output:          messyPath,
		IncludeDocument: true,
	}
	_, output, err := handleFix(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	// The written path should be the cleaned absolute path.
	expected := filepath.Join(subDir, "output.yaml")
	assert.Equal(t, expected, output.WrittenTo)

	// Verify the file was actually written.
	data, err := os.ReadFile(expected)
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestConvertTool_OutputPathSymlinkRejected(t *testing.T) {
	tmpDir := t.TempDir()
	realFile := filepath.Join(tmpDir, "real.yaml")
	linkFile := filepath.Join(tmpDir, "link.yaml")

	require.NoError(t, os.WriteFile(realFile, []byte("placeholder"), 0o600))
	require.NoError(t, os.Symlink(realFile, linkFile))

	input := convertInput{
		Spec:   specInput{Content: minimalOAS30Spec},
		Target: "3.1",
		Output: linkFile,
	}
	result, _, err := handleConvert(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError, "expected error for symlink output path")
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "path")
}

func TestJoinTool_OutputPathSymlinkRejected(t *testing.T) {
	tmpDir := t.TempDir()
	realFile := filepath.Join(tmpDir, "real.yaml")
	linkFile := filepath.Join(tmpDir, "link.yaml")

	require.NoError(t, os.WriteFile(realFile, []byte("placeholder"), 0o600))
	require.NoError(t, os.Symlink(realFile, linkFile))

	input := joinInput{
		Specs: []specInput{
			{Content: joinSpecA},
			{Content: joinSpecB},
		},
		Output: linkFile,
	}
	result, _, err := handleJoin(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError, "expected error for symlink output path")
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "path")
}

func TestOverlayApplyTool_OutputPathSymlinkRejected(t *testing.T) {
	tmpDir := t.TempDir()
	realFile := filepath.Join(tmpDir, "real.yaml")
	linkFile := filepath.Join(tmpDir, "link.yaml")

	require.NoError(t, os.WriteFile(realFile, []byte("placeholder"), 0o600))
	require.NoError(t, os.Symlink(realFile, linkFile))

	input := overlayApplyInput{
		Spec:    specInput{Content: overlayTestSpec},
		Overlay: specInput{Content: overlayTestOverlay},
		Output:  linkFile,
	}
	result, _, err := handleOverlayApply(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError, "expected error for symlink output path")
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "path")
}

func TestGenerateTool_OutputDirSymlinkRejected(t *testing.T) {
	tmpDir := t.TempDir()
	realDir := filepath.Join(tmpDir, "realdir")
	linkDir := filepath.Join(tmpDir, "linkdir")

	require.NoError(t, os.Mkdir(realDir, 0o755))
	require.NoError(t, os.Symlink(realDir, linkDir))

	input := generateInput{
		Spec:      specInput{Content: minimalSpecWithSchemaAndOp},
		Types:     true,
		OutputDir: linkDir,
	}
	result, _, err := handleGenerate(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.IsError, "expected error for symlink output_dir")
	text := result.Content[0].(*mcp.TextContent).Text
	assert.Contains(t, text, "path")
}

func TestOutputFilePermissions(t *testing.T) {
	// Verify that files are written with 0o600 (owner read/write only).
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "fixed.yaml")

	input := fixInput{
		Spec:   specInput{Content: minimalOAS30Spec},
		Output: outPath,
	}
	_, output, err := handleFix(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	require.NotEmpty(t, output.WrittenTo)

	info, err := os.Stat(outPath)
	require.NoError(t, err)
	// On Unix, the file mode should be 0o600 (no group/other permissions).
	perm := info.Mode().Perm()
	assert.Equal(t, os.FileMode(0o600), perm, "expected 0600 permissions, got %o", perm)
}
