package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testParameterSpecYAML is a hand-crafted OAS 3.0.3 spec used in parameter tests.
// Extensions (x-pagination) are included directly so they survive parsing.
const testParameterSpecYAML = `openapi: "3.0.3"
info:
  title: Test
  version: "1.0"
paths:
  /pets:
    get:
      parameters:
        - name: limit
          in: query
          x-pagination: true
        - name: offset
          in: query
      responses:
        '200':
          description: OK
  /pets/{id}:
    parameters:
      - name: id
        in: path
        required: true
    get:
      parameters:
        - name: fields
          in: query
      responses:
        '200':
          description: OK
`

// writeParameterTestSpec writes the test YAML spec to a temp file.
func writeParameterTestSpec(t *testing.T) string {
	t.Helper()
	tmpFile := t.TempDir() + "/test-spec.yaml"
	require.NoError(t, os.WriteFile(tmpFile, []byte(testParameterSpecYAML), 0o644))
	return tmpFile
}

// captureStdout runs fn while capturing os.Stdout and returns the output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() {
		_ = w.Close()
		os.Stdout = old
	}()

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	return buf.String()
}

func TestHandleWalkParameters_MissingFile(t *testing.T) {
	err := handleWalkParameters([]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing spec file")
}

func TestHandleWalkParameters_InvalidFormat(t *testing.T) {
	err := handleWalkParameters([]string{"--format", "xml", "api.yaml"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid format")
}

func TestHandleWalkParameters_ListAll(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	output := captureStdout(t, func() {
		require.NoError(t, handleWalkParameters([]string{tmpFile}))
	})

	// Should list all 4 parameters
	for _, name := range []string{"limit", "offset", "id", "fields"} {
		assert.Contains(t, output, name)
	}
	// Check headers present
	assert.Contains(t, output, "NAME")
}

func TestHandleWalkParameters_FilterByIn(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	output := captureStdout(t, func() {
		require.NoError(t, handleWalkParameters([]string{"--in", "query", tmpFile}))
	})

	// Should include query params: limit, offset, fields
	for _, name := range []string{"limit", "offset", "fields"} {
		assert.Contains(t, output, name)
	}

	// Should NOT include path param 'id' in data rows.
	for line := range strings.SplitSeq(output, "\n") {
		if strings.Contains(line, "NAME") || strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "id ") || strings.TrimSpace(line) == "id" {
			t.Error("expected output to NOT contain 'id' path parameter")
		}
	}
}

func TestHandleWalkParameters_FilterByName(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	output := captureStdout(t, func() {
		require.NoError(t, handleWalkParameters([]string{"--name", "limit", tmpFile}))
	})

	assert.Contains(t, output, "limit")
	for _, name := range []string{"offset", "fields"} {
		assert.NotContains(t, output, name)
	}
}

func TestHandleWalkParameters_FilterByPath(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	output := captureStdout(t, func() {
		require.NoError(t, handleWalkParameters([]string{"--path", "/pets/{id}", tmpFile}))
	})

	// Should include parameters from /pets/{id}: id, fields
	for _, name := range []string{"id", "fields"} {
		assert.Contains(t, output, name)
	}
	// Should not contain parameters from /pets
	assert.NotContains(t, output, "limit")
}

func TestHandleWalkParameters_FilterByMethod(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	output := captureStdout(t, func() {
		require.NoError(t, handleWalkParameters([]string{"--method", "get", tmpFile}))
	})

	// Should include operation-level params: limit, offset, fields
	for _, name := range []string{"limit", "offset", "fields"} {
		assert.Contains(t, output, name)
	}

	// Path-level param 'id' has empty method, should be excluded.
	for line := range strings.SplitSeq(output, "\n") {
		if strings.Contains(line, "NAME") || strings.TrimSpace(line) == "" {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "id ") || trimmed == "id" {
			t.Error("expected output to NOT contain path-level 'id' parameter when filtering by method")
		}
	}
}

func TestHandleWalkParameters_FilterByExtension(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	output := captureStdout(t, func() {
		require.NoError(t, handleWalkParameters([]string{"--extension", "x-pagination", tmpFile}))
	})

	// Only 'limit' has x-pagination extension
	assert.Contains(t, output, "limit")
	assert.NotContains(t, output, "offset")
}

func TestHandleWalkParameters_NoResults(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	// Capture stderr for the "No parameters matched" message
	oldStderr := os.Stderr
	rErr, wErr, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = wErr

	output := captureStdout(t, func() {
		require.NoError(t, handleWalkParameters([]string{"--name", "nonexistent", tmpFile}))
	})

	_ = wErr.Close()
	os.Stderr = oldStderr

	var bufErr bytes.Buffer
	_, _ = bufErr.ReadFrom(rErr)
	stderrOutput := bufErr.String()

	// No stdout output
	assert.Equal(t, "", output)

	// Should have message on stderr
	assert.Contains(t, stderrOutput, "No parameters matched")
}

func TestHandleWalkParameters_QuietMode(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	output := captureStdout(t, func() {
		require.NoError(t, handleWalkParameters([]string{"-q", tmpFile}))
	})

	// Headers should NOT be present in quiet mode
	assert.NotContains(t, output, "NAME")
	// Tab-separated in quiet mode
	assert.Contains(t, output, "\t")
}

func TestHandleWalkParameters_DetailMode(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	output := captureStdout(t, func() {
		require.NoError(t, handleWalkParameters([]string{"--detail", "--name", "limit", tmpFile}))
	})

	// Detail mode outputs YAML by default, should contain parameter fields
	assert.Contains(t, output, "limit")
	assert.Contains(t, output, "name:")
	assert.Contains(t, output, "in:")
}

func TestHandleWalkParameters_DetailIncludesContext(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	output := captureStdout(t, func() {
		require.NoError(t, handleWalkParameters([]string{"--detail", "--format", "json", "--name", "limit", tmpFile}))
	})

	assert.Contains(t, output, `"path"`)
	assert.Contains(t, output, `"method"`)
	assert.Contains(t, output, "/pets")
}

func TestHandleWalkParameters_SummaryJSON(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	output := captureStdout(t, func() {
		require.NoError(t, handleWalkParameters([]string{"--format", "json", tmpFile}))
	})

	assert.Contains(t, output, `"name"`)
	assert.Contains(t, output, `"in"`)
	assert.Contains(t, output, "limit")
}

func TestHandleWalkParameters_SummaryYAML(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	output := captureStdout(t, func() {
		require.NoError(t, handleWalkParameters([]string{"--format", "yaml", tmpFile}))
	})

	assert.Contains(t, output, "name")
	assert.Contains(t, output, "in")
	assert.Contains(t, output, "limit")
}
