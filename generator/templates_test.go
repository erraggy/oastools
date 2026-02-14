package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err, "executeTemplate failed")

	// Verify the output contains expected content
	assert.Contains(t, string(content), "package testpkg", "expected output to contain 'package testpkg'")
	assert.Contains(t, string(content), "TestType", "expected output to contain 'TestType'")
}

// TestGetTemplates tests the lazy template loading functionality.
func TestGetTemplates(t *testing.T) {
	// First call should initialize templates
	tmpl, err := getTemplates()
	require.NoError(t, err, "getTemplates() returned error")
	require.NotNil(t, tmpl, "getTemplates() returned nil template")

	// Second call should return cached templates (sync.Once)
	tmpl2, err := getTemplates()
	require.NoError(t, err, "second getTemplates() call returned error")
	assert.Equal(t, tmpl, tmpl2, "expected same template instance from sync.Once")

	// Verify some expected templates exist
	expectedTemplates := []string{"client.go.tmpl", "types.go.tmpl", "router.go.tmpl"}
	for _, name := range expectedTemplates {
		assert.NotNil(t, tmpl.Lookup(name), "expected template %q to exist", name)
	}
}
