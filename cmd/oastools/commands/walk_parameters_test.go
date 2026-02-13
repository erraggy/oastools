package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"
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
	if err := os.WriteFile(tmpFile, []byte(testParameterSpecYAML), 0o644); err != nil {
		t.Fatalf("failed to write test spec: %v", err)
	}
	return tmpFile
}

// captureStdout runs fn while capturing os.Stdout and returns the output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		_ = w.Close()
		os.Stdout = old
	}()

	fn()

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("failed to read from pipe: %v", err)
	}
	return buf.String()
}

func TestHandleWalkParameters_MissingFile(t *testing.T) {
	err := handleWalkParameters([]string{})
	if err == nil {
		t.Fatal("expected error for missing file argument")
	}
	if !strings.Contains(err.Error(), "missing spec file") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleWalkParameters_InvalidFormat(t *testing.T) {
	err := handleWalkParameters([]string{"--format", "xml", "api.yaml"})
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "invalid format") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestHandleWalkParameters_ListAll(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	output := captureStdout(t, func() {
		if err := handleWalkParameters([]string{tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Should list all 4 parameters
	for _, name := range []string{"limit", "offset", "id", "fields"} {
		if !strings.Contains(output, name) {
			t.Errorf("expected output to contain %q", name)
		}
	}
	// Check headers present
	if !strings.Contains(output, "NAME") {
		t.Error("expected output to contain table header 'NAME'")
	}
}

func TestHandleWalkParameters_FilterByIn(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	output := captureStdout(t, func() {
		if err := handleWalkParameters([]string{"--in", "query", tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Should include query params: limit, offset, fields
	for _, name := range []string{"limit", "offset", "fields"} {
		if !strings.Contains(output, name) {
			t.Errorf("expected output to contain %q", name)
		}
	}

	// Should NOT include path param 'id' in data rows.
	for _, line := range strings.Split(output, "\n") {
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
		if err := handleWalkParameters([]string{"--name", "limit", tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	if !strings.Contains(output, "limit") {
		t.Error("expected output to contain 'limit'")
	}
	for _, name := range []string{"offset", "fields"} {
		if strings.Contains(output, name) {
			t.Errorf("expected output to NOT contain %q", name)
		}
	}
}

func TestHandleWalkParameters_FilterByPath(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	output := captureStdout(t, func() {
		if err := handleWalkParameters([]string{"--path", "/pets/{id}", tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Should include parameters from /pets/{id}: id, fields
	for _, name := range []string{"id", "fields"} {
		if !strings.Contains(output, name) {
			t.Errorf("expected output to contain %q", name)
		}
	}
	// Should not contain parameters from /pets
	if strings.Contains(output, "limit") {
		t.Error("expected output to NOT contain 'limit'")
	}
}

func TestHandleWalkParameters_FilterByMethod(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	output := captureStdout(t, func() {
		if err := handleWalkParameters([]string{"--method", "get", tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Should include operation-level params: limit, offset, fields
	for _, name := range []string{"limit", "offset", "fields"} {
		if !strings.Contains(output, name) {
			t.Errorf("expected output to contain %q", name)
		}
	}

	// Path-level param 'id' has empty method, should be excluded.
	for _, line := range strings.Split(output, "\n") {
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
		if err := handleWalkParameters([]string{"--extension", "x-pagination", tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Only 'limit' has x-pagination extension
	if !strings.Contains(output, "limit") {
		t.Error("expected output to contain 'limit'")
	}
	if strings.Contains(output, "offset") {
		t.Error("expected output to NOT contain 'offset'")
	}
}

func TestHandleWalkParameters_NoResults(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	// Capture stderr for the "No parameters matched" message
	oldStderr := os.Stderr
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = wErr

	output := captureStdout(t, func() {
		if err := handleWalkParameters([]string{"--name", "nonexistent", tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	_ = wErr.Close()
	os.Stderr = oldStderr

	var bufErr bytes.Buffer
	_, _ = bufErr.ReadFrom(rErr)
	stderrOutput := bufErr.String()

	// No stdout output
	if output != "" {
		t.Errorf("expected no stdout output, got: %s", output)
	}

	// Should have message on stderr
	if !strings.Contains(stderrOutput, "No parameters matched") {
		t.Errorf("expected stderr message about no results, got: %s", stderrOutput)
	}
}

func TestHandleWalkParameters_QuietMode(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	output := captureStdout(t, func() {
		if err := handleWalkParameters([]string{"-q", tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Headers should NOT be present in quiet mode
	if strings.Contains(output, "NAME") {
		t.Error("expected output to NOT contain table header 'NAME' in quiet mode")
	}
	// Tab-separated in quiet mode
	if !strings.Contains(output, "\t") {
		t.Error("expected tab-separated output in quiet mode")
	}
}

func TestHandleWalkParameters_DetailMode(t *testing.T) {
	tmpFile := writeParameterTestSpec(t)

	output := captureStdout(t, func() {
		if err := handleWalkParameters([]string{"--detail", "--name", "limit", tmpFile}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	// Detail mode outputs YAML by default, should contain parameter fields
	if !strings.Contains(output, "name: limit") {
		t.Errorf("expected detail output to contain 'name: limit', got: %s", output)
	}
	if !strings.Contains(output, "in: query") {
		t.Errorf("expected detail output to contain 'in: query', got: %s", output)
	}
}
