package generator

import (
	"strings"
	"testing"
)

// TestExecuteTemplate tests the template execution with formatting.
func TestExecuteTemplate(t *testing.T) {
	// Test with a simple template that produces valid Go code
	data := &TypesFileData{
		Header: HeaderData{
			PackageName: "testpkg",
		},
		Types: []TypeDefinition{
			{
				Kind: "alias",
				Alias: &AliasData{
					TypeName:   "TestType",
					TargetType: "string",
					Comment:    "TestType is a test type",
					IsDefined:  true,
				},
			},
		},
	}

	content, err := executeTemplate("types.go.tmpl", data)
	if err != nil {
		t.Fatalf("executeTemplate failed: %v", err)
	}

	// Verify the output contains expected content
	if !strings.Contains(string(content), "package testpkg") {
		t.Error("expected output to contain 'package testpkg'")
	}
	if !strings.Contains(string(content), "TestType") {
		t.Error("expected output to contain 'TestType'")
	}
}

// TestGetTemplates tests the lazy template loading functionality.
func TestGetTemplates(t *testing.T) {
	// First call should initialize templates
	tmpl, err := getTemplates()
	if err != nil {
		t.Fatalf("getTemplates() returned error: %v", err)
	}
	if tmpl == nil {
		t.Fatal("getTemplates() returned nil template")
	}

	// Second call should return cached templates (sync.Once)
	tmpl2, err := getTemplates()
	if err != nil {
		t.Fatalf("second getTemplates() call returned error: %v", err)
	}
	if tmpl2 != tmpl {
		t.Error("expected same template instance from sync.Once")
	}

	// Verify some expected templates exist
	expectedTemplates := []string{"client.go.tmpl", "types.go.tmpl", "router.go.tmpl"}
	for _, name := range expectedTemplates {
		if tmpl.Lookup(name) == nil {
			t.Errorf("expected template %q to exist", name)
		}
	}
}
