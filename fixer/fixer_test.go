package fixer

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew tests the New constructor
func TestNew(t *testing.T) {
	f := New()
	require.NotNil(t, f)
	assert.False(t, f.InferTypes)
	assert.Equal(t, []FixType{FixTypeMissingPathParameter}, f.EnabledFixes)
}

// TestFixWithOptions_NoInput tests that FixWithOptions fails with no input
func TestFixWithOptions_NoInput(t *testing.T) {
	_, err := FixWithOptions()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no input source specified")
}

// TestFixWithOptions_MultipleInputs tests that FixWithOptions fails with multiple inputs
func TestFixWithOptions_MultipleInputs(t *testing.T) {
	_, err := FixWithOptions(
		WithFilePath("test.yaml"),
		WithParsed(parser.ParseResult{}),
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "multiple input sources")
}

// TestFixWithOptions_EmptyPath tests that FixWithOptions fails with empty path
func TestFixWithOptions_EmptyPath(t *testing.T) {
	_, err := FixWithOptions(
		WithFilePath(""),
	)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file path cannot be empty")
}

// TestDeepCopyOAS3Document tests that deep copy preserves OASVersion
func TestDeepCopyOAS3Document(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info: &parser.Info{
			Title:   "Test",
			Version: "1.0.0",
		},
	}

	copied, err := deepCopyOAS3Document(doc)
	require.NoError(t, err)

	assert.Equal(t, doc.OASVersion, copied.OASVersion)
	assert.Equal(t, doc.OpenAPI, copied.OpenAPI)
	assert.Equal(t, doc.Info.Title, copied.Info.Title)

	// Ensure it's a true copy (mutating one doesn't affect the other)
	copied.Info.Title = "Modified"
	assert.NotEqual(t, doc.Info.Title, copied.Info.Title)
}

// TestDeepCopyOAS2Document tests that deep copy preserves OASVersion
func TestDeepCopyOAS2Document(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger:    "2.0",
		OASVersion: parser.OASVersion20,
		Info: &parser.Info{
			Title:   "Test",
			Version: "1.0.0",
		},
	}

	copied, err := deepCopyOAS2Document(doc)
	require.NoError(t, err)

	assert.Equal(t, doc.OASVersion, copied.OASVersion)
	assert.Equal(t, doc.Swagger, copied.Swagger)
	assert.Equal(t, doc.Info.Title, copied.Info.Title)

	// Ensure it's a true copy
	copied.Info.Title = "Modified"
	assert.NotEqual(t, doc.Info.Title, copied.Info.Title)
}

// TestIsFixEnabled tests the fix type filtering
func TestIsFixEnabled(t *testing.T) {
	f := New()

	// By default, only missing params fix is enabled
	assert.True(t, f.isFixEnabled(FixTypeMissingPathParameter))
	assert.False(t, f.isFixEnabled(FixTypePrunedUnusedSchema))

	// When specific fixes are set, only those are enabled
	f.EnabledFixes = []FixType{FixTypeMissingPathParameter}
	assert.True(t, f.isFixEnabled(FixTypeMissingPathParameter))

	// Empty slice enables all fixes (backwards compatibility)
	f.EnabledFixes = []FixType{}
	assert.True(t, f.isFixEnabled(FixTypeMissingPathParameter)) // empty = all enabled
	assert.True(t, f.isFixEnabled(FixTypePrunedUnusedSchema))   // empty = all enabled
}

// TestFixResult_HasFixes tests the HasFixes helper method
func TestFixResult_HasFixes(t *testing.T) {
	result := &FixResult{FixCount: 0}
	assert.False(t, result.HasFixes())

	result.FixCount = 1
	assert.True(t, result.HasFixes())
}

func TestFixResult_ToParseResult(t *testing.T) {
	t.Run("OAS3 result converts correctly", func(t *testing.T) {
		fixResult := &FixResult{
			Document:         &parser.OAS3Document{OpenAPI: "3.0.3", Info: &parser.Info{Title: "Test API", Version: "1.0"}},
			SourceVersion:    "3.0.3",
			SourceOASVersion: parser.OASVersion303,
			SourceFormat:     parser.SourceFormatYAML,
			SourcePath:       "/path/to/api.yaml",
			Fixes: []Fix{
				{Type: FixTypeMissingPathParameter, Path: "paths./users.get", Description: "Added userId"},
			},
			FixCount: 1,
			Success:  true,
			Stats:    parser.DocumentStats{PathCount: 5, OperationCount: 10},
		}

		parseResult := fixResult.ToParseResult()

		assert.Equal(t, "/path/to/api.yaml", parseResult.SourcePath)
		assert.Equal(t, parser.SourceFormatYAML, parseResult.SourceFormat)
		assert.Equal(t, "3.0.3", parseResult.Version)
		assert.Equal(t, parser.OASVersion303, parseResult.OASVersion)
		assert.NotNil(t, parseResult.Document)
		assert.Empty(t, parseResult.Errors)
		assert.Empty(t, parseResult.Warnings) // Fixes are not errors/warnings
		assert.Equal(t, 5, parseResult.Stats.PathCount)
		assert.Equal(t, 10, parseResult.Stats.OperationCount)

		// Verify Document type assertion works
		doc, ok := parseResult.Document.(*parser.OAS3Document)
		assert.True(t, ok)
		assert.Equal(t, "Test API", doc.Info.Title)
	})

	t.Run("OAS2 result converts correctly", func(t *testing.T) {
		fixResult := &FixResult{
			Document:         &parser.OAS2Document{Swagger: "2.0", Info: &parser.Info{Title: "Swagger API", Version: "1.0"}},
			SourceVersion:    "2.0",
			SourceOASVersion: parser.OASVersion20,
			SourceFormat:     parser.SourceFormatJSON,
			SourcePath:       "/api/swagger.json",
			Stats:            parser.DocumentStats{PathCount: 3},
		}

		parseResult := fixResult.ToParseResult()

		assert.Equal(t, "/api/swagger.json", parseResult.SourcePath)
		assert.Equal(t, parser.SourceFormatJSON, parseResult.SourceFormat)
		assert.Equal(t, "2.0", parseResult.Version)
		assert.Equal(t, parser.OASVersion20, parseResult.OASVersion)

		doc, ok := parseResult.Document.(*parser.OAS2Document)
		assert.True(t, ok)
		assert.Equal(t, "Swagger API", doc.Info.Title)
	})

	t.Run("empty SourcePath uses default", func(t *testing.T) {
		fixResult := &FixResult{
			Document:         &parser.OAS3Document{OpenAPI: "3.1.0"},
			SourceVersion:    "3.1.0",
			SourceOASVersion: parser.OASVersion310,
			SourceFormat:     parser.SourceFormatYAML,
			SourcePath:       "", // Empty
		}

		parseResult := fixResult.ToParseResult()

		assert.Equal(t, "fixer", parseResult.SourcePath)
	})

	t.Run("Errors and Warnings are empty slices", func(t *testing.T) {
		// Fixes are informational, not errors/warnings
		fixResult := &FixResult{
			Document:         &parser.OAS3Document{OpenAPI: "3.0.0"},
			SourceVersion:    "3.0.0",
			SourceOASVersion: parser.OASVersion300,
			SourceFormat:     parser.SourceFormatYAML,
			Fixes: []Fix{
				{Type: FixTypeMissingPathParameter},
				{Type: FixTypeRenamedGenericSchema},
			},
			FixCount: 2,
		}

		parseResult := fixResult.ToParseResult()

		assert.NotNil(t, parseResult.Errors)
		assert.Empty(t, parseResult.Errors)
		assert.NotNil(t, parseResult.Warnings)
		assert.Empty(t, parseResult.Warnings)
	})

	t.Run("Data field is nil and LoadTime/SourceSize are zero", func(t *testing.T) {
		// FixResult doesn't track LoadTime/SourceSize, so they should be zero
		fixResult := &FixResult{
			Document:         &parser.OAS3Document{OpenAPI: "3.0.0"},
			SourceVersion:    "3.0.0",
			SourceOASVersion: parser.OASVersion300,
			SourceFormat:     parser.SourceFormatYAML,
		}

		parseResult := fixResult.ToParseResult()

		assert.Nil(t, parseResult.Data)
		assert.Zero(t, parseResult.LoadTime)
		assert.Zero(t, parseResult.SourceSize)
	})

	t.Run("nil Document produces warning", func(t *testing.T) {
		fixResult := &FixResult{
			Document:         nil, // Nil document should produce warning
			SourceVersion:    "3.0.0",
			SourceOASVersion: parser.OASVersion300,
			SourceFormat:     parser.SourceFormatYAML,
			SourcePath:       "/path/to/api.yaml",
		}

		parseResult := fixResult.ToParseResult()

		require.Len(t, parseResult.Warnings, 1, "Should have one warning for nil document")
		assert.Contains(t, parseResult.Warnings[0], "Document is nil", "Warning should mention nil document")
		assert.Contains(t, parseResult.Warnings[0], "downstream operations may fail", "Warning should mention downstream impact")
	})
}

// TestFix_HasLocation tests the HasLocation helper method
func TestFix_HasLocation(t *testing.T) {
	tests := []struct {
		name     string
		fix      Fix
		expected bool
	}{
		{
			name:     "no location",
			fix:      Fix{Path: "paths./users.get"},
			expected: false,
		},
		{
			name:     "with line",
			fix:      Fix{Path: "paths./users.get", Line: 10},
			expected: true,
		},
		{
			name:     "with line and column",
			fix:      Fix{Path: "paths./users.get", Line: 10, Column: 5},
			expected: true,
		},
		{
			name:     "with file and line",
			fix:      Fix{Path: "paths./users.get", File: "api.yaml", Line: 10, Column: 5},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.fix.HasLocation())
		})
	}
}

// TestFix_Location tests the Location helper method
func TestFix_Location(t *testing.T) {
	tests := []struct {
		name     string
		fix      Fix
		expected string
	}{
		{
			name:     "no location returns path",
			fix:      Fix{Path: "paths./users.get"},
			expected: "paths./users.get",
		},
		{
			name:     "line and column only",
			fix:      Fix{Path: "paths./users.get", Line: 10, Column: 5},
			expected: "10:5",
		},
		{
			name:     "file, line and column",
			fix:      Fix{Path: "paths./users.get", File: "api.yaml", Line: 10, Column: 5},
			expected: "api.yaml:10:5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.fix.Location())
		})
	}
}

// TestWithSourceMap_Fixer tests the WithSourceMap option function
func TestWithSourceMap_Fixer(t *testing.T) {
	sm := parser.NewSourceMap()
	cfg := &fixConfig{}
	opt := WithSourceMap(sm)
	err := opt(cfg)

	require.NoError(t, err)
	assert.Equal(t, sm, cfg.sourceMap)
}

// =============================================================================
// Option Function Tests
// =============================================================================

// TestWithGenericNaming tests the WithGenericNaming option
func TestWithGenericNaming(t *testing.T) {
	cfg := &fixConfig{}
	opt := WithGenericNaming(GenericNamingOf)
	err := opt(cfg)

	require.NoError(t, err)
	assert.Equal(t, GenericNamingOf, cfg.genericNamingConfig.Strategy)
}

// TestWithGenericNamingConfig tests the WithGenericNamingConfig option
func TestWithGenericNamingConfig(t *testing.T) {
	customConfig := GenericNamingConfig{
		Strategy:       GenericNamingFor,
		Separator:      "-",
		ParamSeparator: "-",
		PreserveCasing: true,
	}

	cfg := &fixConfig{}
	opt := WithGenericNamingConfig(customConfig)
	err := opt(cfg)

	require.NoError(t, err)
	assert.Equal(t, customConfig, cfg.genericNamingConfig)
}

// TestWithDryRun tests the WithDryRun option
func TestWithDryRun(t *testing.T) {
	cfg := &fixConfig{}
	opt := WithDryRun(true)
	err := opt(cfg)

	require.NoError(t, err)
	assert.True(t, cfg.dryRun)
}

// TestWithEnabledFixes tests the WithEnabledFixes option
func TestWithEnabledFixes(t *testing.T) {
	cfg := &fixConfig{}
	opt := WithEnabledFixes(FixTypePrunedUnusedSchema, FixTypeRenamedGenericSchema)
	err := opt(cfg)

	require.NoError(t, err)
	assert.Equal(t, []FixType{FixTypePrunedUnusedSchema, FixTypeRenamedGenericSchema}, cfg.enabledFixes)
}

// TestIsFixEnabled_MultipleTypes tests isFixEnabled with multiple fix types
func TestIsFixEnabled_MultipleTypes(t *testing.T) {
	f := New()

	// By default, only missing path parameter fix is enabled
	assert.True(t, f.isFixEnabled(FixTypeMissingPathParameter))
	assert.False(t, f.isFixEnabled(FixTypePrunedUnusedSchema))
	assert.False(t, f.isFixEnabled(FixTypeRenamedGenericSchema))
	assert.False(t, f.isFixEnabled(FixTypePrunedEmptyPath))

	// Restrict to specific fixes
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema, FixTypeRenamedGenericSchema}
	assert.False(t, f.isFixEnabled(FixTypeMissingPathParameter))
	assert.True(t, f.isFixEnabled(FixTypePrunedUnusedSchema))
	assert.True(t, f.isFixEnabled(FixTypeRenamedGenericSchema))
	assert.False(t, f.isFixEnabled(FixTypePrunedEmptyPath))
}

// TestFixer_SourceMapPassedThrough tests that source map is passed to the Fixer
func TestFixer_SourceMapPassedThrough(t *testing.T) {
	sm := parser.NewSourceMap()

	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithSourceMap(sm),
	)
	require.NoError(t, err)
	assert.NotNil(t, result)
	// Verify fix was applied (missing userId parameter)
	assert.True(t, result.HasFixes())
}

// TestWithMutableInput tests the WithMutableInput option
func TestWithMutableInput(t *testing.T) {
	cfg := &fixConfig{}
	opt := WithMutableInput(true)
	err := opt(cfg)

	require.NoError(t, err)
	assert.True(t, cfg.mutableInput)

	// Test false value
	cfg2 := &fixConfig{}
	opt2 := WithMutableInput(false)
	err2 := opt2(cfg2)

	require.NoError(t, err2)
	assert.False(t, cfg2.mutableInput)
}

// TestMutableInput_OAS3_MutatesOriginal verifies that WithMutableInput(true)
// mutates the original document instead of copying
func TestMutableInput_OAS3_MutatesOriginal(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	// Get the original document
	originalDoc, ok := parseResult.OAS3Document()
	require.True(t, ok)

	// Verify no parameters initially
	pathItem := originalDoc.Paths["/users/{userId}"]
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Get)
	originalParamCount := len(pathItem.Get.Parameters)

	// Fix with mutable input
	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithMutableInput(true),
	)
	require.NoError(t, err)
	assert.True(t, result.HasFixes())

	// Verify the original document was mutated
	assert.Greater(t, len(pathItem.Get.Parameters), originalParamCount,
		"Original document should be mutated when MutableInput is true")
}

// TestMutableInput_OAS3_PreservesOriginal verifies that WithMutableInput(false)
// (the default) does not mutate the original document
func TestMutableInput_OAS3_PreservesOriginal(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	// Get the original document
	originalDoc, ok := parseResult.OAS3Document()
	require.True(t, ok)

	// Verify no parameters initially
	pathItem := originalDoc.Paths["/users/{userId}"]
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Get)
	originalParamCount := len(pathItem.Get.Parameters)

	// Fix WITHOUT mutable input (default behavior)
	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithMutableInput(false),
	)
	require.NoError(t, err)
	assert.True(t, result.HasFixes())

	// Verify the original document was NOT mutated
	assert.Equal(t, originalParamCount, len(pathItem.Get.Parameters),
		"Original document should NOT be mutated when MutableInput is false")
}

// TestMutableInput_OAS2_MutatesOriginal verifies that WithMutableInput(true)
// mutates the original OAS 2.0 document
func TestMutableInput_OAS2_MutatesOriginal(t *testing.T) {
	spec := `
swagger: "2.0"
info:
  title: Test API
  version: 1.0.0
paths:
  /pets/{petId}:
    get:
      operationId: getPet
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	// Get the original document
	originalDoc, ok := parseResult.OAS2Document()
	require.True(t, ok)

	// Verify no parameters initially
	pathItem := originalDoc.Paths["/pets/{petId}"]
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Get)
	originalParamCount := len(pathItem.Get.Parameters)

	// Fix with mutable input
	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithMutableInput(true),
	)
	require.NoError(t, err)
	assert.True(t, result.HasFixes())

	// Verify the original document was mutated
	assert.Greater(t, len(pathItem.Get.Parameters), originalParamCount,
		"Original OAS2 document should be mutated when MutableInput is true")
}

// TestMutableInput_DefaultIsFalse verifies that the default behavior
// is to NOT mutate the input (defensive copy)
func TestMutableInput_DefaultIsFalse(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	// Get the original document
	originalDoc, ok := parseResult.OAS3Document()
	require.True(t, ok)

	// Verify no parameters initially
	pathItem := originalDoc.Paths["/users/{userId}"]
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Get)
	originalParamCount := len(pathItem.Get.Parameters)

	// Fix WITHOUT specifying mutable input (should default to false)
	result, err := FixWithOptions(
		WithParsed(*parseResult),
	)
	require.NoError(t, err)
	assert.True(t, result.HasFixes())

	// Verify the original document was NOT mutated (default behavior)
	assert.Equal(t, originalParamCount, len(pathItem.Get.Parameters),
		"Original document should NOT be mutated by default")
}

// TestFixer_MutableInput_DirectAPI tests using the MutableInput field
// directly on the Fixer struct
func TestFixer_MutableInput_DirectAPI(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	// Get the original document
	originalDoc, ok := parseResult.OAS3Document()
	require.True(t, ok)

	pathItem := originalDoc.Paths["/users/{userId}"]
	require.NotNil(t, pathItem)
	originalParamCount := len(pathItem.Get.Parameters)

	// Use direct Fixer API with MutableInput
	f := New()
	f.MutableInput = true

	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)
	assert.True(t, result.HasFixes())

	// Verify the original document was mutated
	assert.Greater(t, len(pathItem.Get.Parameters), originalParamCount,
		"Original document should be mutated when Fixer.MutableInput is true")
}
