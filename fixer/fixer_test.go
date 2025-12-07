package fixer

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/internal/corpusutil"
	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew tests the New constructor
func TestNew(t *testing.T) {
	f := New()
	require.NotNil(t, f)
	assert.False(t, f.InferTypes)
	assert.Nil(t, f.EnabledFixes)
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

// TestExtractPathParameters tests the extractPathParameters helper
func TestExtractPathParameters(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected map[string]bool
	}{
		{
			name:     "no parameters",
			path:     "/users",
			expected: map[string]bool{},
		},
		{
			name:     "single parameter",
			path:     "/users/{userId}",
			expected: map[string]bool{"userId": true},
		},
		{
			name:     "multiple parameters",
			path:     "/users/{userId}/posts/{postId}",
			expected: map[string]bool{"userId": true, "postId": true},
		},
		{
			name:     "parameter at start",
			path:     "/{version}/users",
			expected: map[string]bool{"version": true},
		},
		{
			name:     "parameter with hyphen",
			path:     "/users/{user-id}",
			expected: map[string]bool{"user-id": true},
		},
		{
			name:     "parameter with underscore",
			path:     "/users/{user_id}",
			expected: map[string]bool{"user_id": true},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractPathParameters(tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestInferParameterType tests type inference from parameter names
func TestInferParameterType(t *testing.T) {
	tests := []struct {
		name           string
		paramName      string
		expectedType   string
		expectedFormat string
	}{
		{
			name:           "lowercase id suffix",
			paramName:      "userid",
			expectedType:   "integer",
			expectedFormat: "",
		},
		{
			name:           "camelCase Id suffix",
			paramName:      "userId",
			expectedType:   "integer",
			expectedFormat: "",
		},
		{
			name:           "uppercase ID suffix",
			paramName:      "userID",
			expectedType:   "integer",
			expectedFormat: "",
		},
		{
			name:           "uuid in name",
			paramName:      "userUuid",
			expectedType:   "string",
			expectedFormat: "uuid",
		},
		{
			name:           "guid in name",
			paramName:      "sessionGuid",
			expectedType:   "string",
			expectedFormat: "uuid",
		},
		{
			name:           "plain string name",
			paramName:      "name",
			expectedType:   "string",
			expectedFormat: "",
		},
		{
			name:           "slug parameter",
			paramName:      "slug",
			expectedType:   "string",
			expectedFormat: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotType, gotFormat := inferParameterType(tc.paramName)
			assert.Equal(t, tc.expectedType, gotType, "type mismatch")
			assert.Equal(t, tc.expectedFormat, gotFormat, "format mismatch")
		})
	}
}

// TestFixMissingPathParametersOAS3 tests fixing missing path parameters in OAS 3.x
func TestFixMissingPathParametersOAS3(t *testing.T) {
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

	f := New()
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	assert.True(t, result.HasFixes())
	assert.Equal(t, 1, result.FixCount)
	assert.Equal(t, FixTypeMissingPathParameter, result.Fixes[0].Type)
	assert.Contains(t, result.Fixes[0].Description, "userId")

	// Verify the parameter was added
	doc := result.Document.(*parser.OAS3Document)
	pathItem := doc.Paths["/users/{userId}"]
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Get)
	require.Len(t, pathItem.Get.Parameters, 1)

	param := pathItem.Get.Parameters[0]
	assert.Equal(t, "userId", param.Name)
	assert.Equal(t, "path", param.In)
	assert.True(t, param.Required)
	assert.NotNil(t, param.Schema)
	assert.Equal(t, "string", param.Schema.Type)
}

// TestFixMissingPathParametersOAS3_WithInfer tests type inference
func TestFixMissingPathParametersOAS3_WithInfer(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}/documents/{documentUuid}:
    get:
      operationId: getDocument
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	f := New()
	f.InferTypes = true
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	assert.Equal(t, 2, result.FixCount)

	// Find the parameters
	doc := result.Document.(*parser.OAS3Document)
	params := doc.Paths["/users/{userId}/documents/{documentUuid}"].Get.Parameters
	require.Len(t, params, 2)

	// Check types - they may be in any order
	paramsByName := make(map[string]*parser.Parameter)
	for _, p := range params {
		paramsByName[p.Name] = p
	}

	assert.Equal(t, "integer", paramsByName["userId"].Schema.Type)
	assert.Equal(t, "string", paramsByName["documentUuid"].Schema.Type)
	assert.Equal(t, "uuid", paramsByName["documentUuid"].Schema.Format)
}

// TestFixMissingPathParametersOAS2 tests fixing missing path parameters in OAS 2.0
func TestFixMissingPathParametersOAS2(t *testing.T) {
	spec := `
swagger: "2.0"
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

	f := New()
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	assert.True(t, result.HasFixes())
	assert.Equal(t, 1, result.FixCount)

	// Verify the parameter was added with OAS 2.0 style (type directly on param)
	doc := result.Document.(*parser.OAS2Document)
	pathItem := doc.Paths["/users/{userId}"]
	require.NotNil(t, pathItem)
	require.NotNil(t, pathItem.Get)
	require.Len(t, pathItem.Get.Parameters, 1)

	param := pathItem.Get.Parameters[0]
	assert.Equal(t, "userId", param.Name)
	assert.Equal(t, "path", param.In)
	assert.True(t, param.Required)
	assert.Equal(t, "string", param.Type) // OAS 2.0 uses Type directly
}

// TestFixNoChangesNeeded tests that no fixes are applied when spec is valid
func TestFixNoChangesNeeded(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    get:
      operationId: getUser
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	f := New()
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	assert.False(t, result.HasFixes())
	assert.Equal(t, 0, result.FixCount)
}

// TestFixPathItemLevelParameters tests that PathItem-level params are considered
func TestFixPathItemLevelParameters(t *testing.T) {
	spec := `
openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
paths:
  /users/{userId}:
    parameters:
      - name: userId
        in: path
        required: true
        schema:
          type: string
    get:
      operationId: getUser
      responses:
        '200':
          description: Success
    put:
      operationId: updateUser
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	require.NoError(t, err)

	f := New()
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// No fixes needed - userId is declared at PathItem level
	assert.False(t, result.HasFixes())
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

	// By default, all fixes are enabled
	assert.True(t, f.isFixEnabled(FixTypeMissingPathParameter))

	// When specific fixes are set, only those are enabled
	f.EnabledFixes = []FixType{FixTypeMissingPathParameter}
	assert.True(t, f.isFixEnabled(FixTypeMissingPathParameter))

	// Other fix types would be disabled (if they existed)
	f.EnabledFixes = []FixType{}
	assert.True(t, f.isFixEnabled(FixTypeMissingPathParameter)) // empty = all enabled
}

// TestFixResult_HasFixes tests the HasFixes helper method
func TestFixResult_HasFixes(t *testing.T) {
	result := &FixResult{FixCount: 0}
	assert.False(t, result.HasFixes())

	result.FixCount = 1
	assert.True(t, result.HasFixes())
}

// =============================================================================
// Corpus Integration Tests
// =============================================================================

// TestCorpus_FixerReducesErrors tests that the fixer reduces validation errors
// for corpus specs that have missing path parameter issues.
func TestCorpus_FixerReducesErrors(t *testing.T) {
	// Skip if corpus isn't downloaded
	spec := corpusutil.GetByName("DigitalOcean")
	if spec == nil {
		t.Skip("DigitalOcean spec not found in corpus")
	}
	corpusutil.SkipIfNotCached(t, *spec)
	corpusutil.SkipLargeInShortMode(t, *spec)

	// Parse the spec
	p := parser.New()
	parseResult, err := p.Parse(spec.GetLocalPath())
	require.NoError(t, err, "Failed to parse %s", spec.Name)

	// Validate before fixing
	v := validator.New()
	v.StrictMode = true
	beforeResult, err := v.ValidateParsed(*parseResult)
	require.NoError(t, err, "Failed to validate before fixing")

	beforeErrors := beforeResult.ErrorCount

	// Apply fixes
	f := New()
	fixResult, err := f.FixParsed(*parseResult)
	require.NoError(t, err, "Failed to fix %s", spec.Name)

	t.Logf("%s: Applied %d fixes", spec.Name, fixResult.FixCount)

	// Validate after fixing - need to create a new ParseResult with fixed doc
	fixedParseResult := &parser.ParseResult{
		Document:     fixResult.Document,
		OASVersion:   fixResult.SourceOASVersion,
		Version:      fixResult.SourceVersion,
		SourceFormat: fixResult.SourceFormat,
	}

	afterResult, err := v.ValidateParsed(*fixedParseResult)
	require.NoError(t, err, "Failed to validate after fixing")

	afterErrors := afterResult.ErrorCount

	// The fixer should reduce errors (or at least not increase them)
	t.Logf("%s: Errors before: %d, after: %d, reduced by: %d",
		spec.Name, beforeErrors, afterErrors, beforeErrors-afterErrors)

	assert.LessOrEqual(t, afterErrors, beforeErrors,
		"Fixer should not increase errors for %s", spec.Name)
}

// TestCorpus_FixerAllInvalidSpecs tests the fixer on all invalid corpus specs
func TestCorpus_FixerAllInvalidSpecs(t *testing.T) {
	invalidSpecs := corpusutil.GetInvalidSpecs(false) // exclude large

	for _, spec := range invalidSpecs {
		t.Run(spec.Name, func(t *testing.T) {
			corpusutil.SkipIfNotCached(t, spec)
			corpusutil.SkipIfHasParsingIssues(t, spec)

			// Parse
			p := parser.New()
			parseResult, err := p.Parse(spec.GetLocalPath())
			require.NoError(t, err, "Failed to parse")

			// Validate before
			v := validator.New()
			v.StrictMode = true
			beforeResult, err := v.ValidateParsed(*parseResult)
			require.NoError(t, err, "Failed to validate before")

			// Fix
			f := New()
			fixResult, err := f.FixParsed(*parseResult)
			require.NoError(t, err, "Failed to fix")

			// Validate after
			fixedParseResult := &parser.ParseResult{
				Document:     fixResult.Document,
				OASVersion:   fixResult.SourceOASVersion,
				Version:      fixResult.SourceVersion,
				SourceFormat: fixResult.SourceFormat,
			}

			afterResult, err := v.ValidateParsed(*fixedParseResult)
			require.NoError(t, err, "Failed to validate after")

			t.Logf("Fixes: %d, Errors before: %d, after: %d, reduced: %d",
				fixResult.FixCount,
				beforeResult.ErrorCount,
				afterResult.ErrorCount,
				beforeResult.ErrorCount-afterResult.ErrorCount)

			// Fixer should not increase errors
			assert.LessOrEqual(t, afterResult.ErrorCount, beforeResult.ErrorCount)
		})
	}
}

// TestCorpus_FixerValidSpecs tests that fixer doesn't break valid specs
func TestCorpus_FixerValidSpecs(t *testing.T) {
	validSpecs := corpusutil.GetValidSpecs(false) // exclude large

	for _, spec := range validSpecs {
		t.Run(spec.Name, func(t *testing.T) {
			corpusutil.SkipIfNotCached(t, spec)

			// Parse
			p := parser.New()
			parseResult, err := p.Parse(spec.GetLocalPath())
			require.NoError(t, err, "Failed to parse")

			// Fix (should have no changes)
			f := New()
			fixResult, err := f.FixParsed(*parseResult)
			require.NoError(t, err, "Failed to fix")

			t.Logf("Fixes applied: %d", fixResult.FixCount)

			// Validate after - should still be valid
			v := validator.New()
			v.StrictMode = true

			fixedParseResult := &parser.ParseResult{
				Document:     fixResult.Document,
				OASVersion:   fixResult.SourceOASVersion,
				Version:      fixResult.SourceVersion,
				SourceFormat: fixResult.SourceFormat,
			}

			afterResult, err := v.ValidateParsed(*fixedParseResult)
			require.NoError(t, err, "Failed to validate after")

			assert.True(t, afterResult.Valid,
				"Valid spec should remain valid after fixing. Errors: %d",
				afterResult.ErrorCount)
		})
	}
}

// TestCorpus_FixerWithInferTypes tests type inference on real specs
func TestCorpus_FixerWithInferTypes(t *testing.T) {
	spec := corpusutil.GetByName("Asana")
	if spec == nil {
		t.Skip("Asana spec not found in corpus")
	}
	corpusutil.SkipIfNotCached(t, *spec)

	// Parse
	p := parser.New()
	parseResult, err := p.Parse(spec.GetLocalPath())
	require.NoError(t, err)

	// Fix with type inference
	f := New()
	f.InferTypes = true
	fixResult, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	t.Logf("Applied %d fixes with type inference", fixResult.FixCount)

	// Check that some parameters were inferred as integers
	integerCount := 0
	stringCount := 0
	for _, fix := range fixResult.Fixes {
		if strings.Contains(fix.Description, "type: integer") {
			integerCount++
		} else if strings.Contains(fix.Description, "type: string") {
			stringCount++
		}
	}

	t.Logf("Integer params: %d, String params: %d", integerCount, stringCount)

	// With inference, we expect some integer types for ID parameters
	if fixResult.FixCount > 0 {
		assert.True(t, integerCount > 0 || stringCount > 0,
			"With --infer, should see typed parameters")
	}
}

// BenchmarkFix benchmarks fixing a spec with missing path parameters
func BenchmarkFix(b *testing.B) {
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
  /projects/{projectId}/tasks/{taskId}:
    get:
      operationId: getTask
      responses:
        '200':
          description: Success
    put:
      operationId: updateTask
      responses:
        '200':
          description: Success
`
	p := parser.New()
	parseResult, err := p.ParseBytes([]byte(spec))
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		f := New()
		_, err := f.FixParsed(*parseResult)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCorpus_Fix benchmarks fixing a real corpus spec
func BenchmarkCorpus_Fix(b *testing.B) {
	spec := corpusutil.GetByName("Asana")
	if spec == nil {
		b.Skip("Asana spec not found")
	}
	if !spec.IsAvailable() {
		b.Skipf("Corpus file %s not cached", spec.Filename)
	}

	p := parser.New()
	parseResult, err := p.Parse(spec.GetLocalPath())
	if err != nil {
		b.Fatal(err)
	}

	for b.Loop() {
		f := New()
		_, err := f.FixParsed(*parseResult)
		if err != nil {
			b.Fatal(err)
		}
	}
}
