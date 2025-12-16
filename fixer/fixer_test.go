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
	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3Document")
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
	doc, ok := result.Document.(*parser.OAS3Document)
	require.True(t, ok, "expected OAS3Document")
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
	doc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok, "expected OAS2Document")
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

// Note: BenchmarkCorpus_Fix has been moved to corpus_bench_test.go
// Run with: go test -tags=corpus -bench=BenchmarkCorpus ./fixer/...

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
// Generic Schema Name Fixing Tests
// =============================================================================

// TestFixInvalidSchemaNamesOAS3 tests renaming schemas with invalid characters in OAS 3.x
func TestFixInvalidSchemaNamesOAS3(t *testing.T) {
	tests := []struct {
		name           string
		spec           string
		strategy       GenericNamingStrategy
		expectedSchema string
		expectedRef    string
		expectFix      bool
	}{
		{
			name: "brackets renamed with underscore strategy",
			spec: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Response[User]"
components:
  schemas:
    Response[User]:
      type: object
      properties:
        data:
          $ref: "#/components/schemas/User"
    User:
      type: object
      properties:
        id:
          type: integer
`,
			strategy:       GenericNamingUnderscore,
			expectedSchema: "Response_User_",
			expectedRef:    "#/components/schemas/Response_User_",
			expectFix:      true,
		},
		{
			name: "brackets renamed with of strategy",
			spec: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Response[User]"
components:
  schemas:
    Response[User]:
      type: object
      properties:
        data:
          $ref: "#/components/schemas/User"
    User:
      type: object
      properties:
        id:
          type: integer
`,
			strategy:       GenericNamingOf,
			expectedSchema: "ResponseOfUser",
			expectedRef:    "#/components/schemas/ResponseOfUser",
			expectFix:      true,
		},
		{
			name: "brackets renamed with for strategy",
			spec: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/List[Item]"
components:
  schemas:
    List[Item]:
      type: array
      items:
        $ref: "#/components/schemas/Item"
    Item:
      type: object
`,
			strategy:       GenericNamingFor,
			expectedSchema: "ListForItem",
			expectedRef:    "#/components/schemas/ListForItem",
			expectFix:      true,
		},
		{
			name: "brackets renamed with flattened strategy",
			spec: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /data:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Container[Value]"
components:
  schemas:
    Container[Value]:
      type: object
    Value:
      type: string
`,
			strategy:       GenericNamingFlattened,
			expectedSchema: "ContainerValue",
			expectedRef:    "#/components/schemas/ContainerValue",
			expectFix:      true,
		},
		{
			name: "brackets renamed with dot strategy",
			spec: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /data:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Wrapper[Data]"
components:
  schemas:
    Wrapper[Data]:
      type: object
    Data:
      type: string
`,
			strategy:       GenericNamingDot,
			expectedSchema: "Wrapper.Data",
			expectedRef:    "#/components/schemas/Wrapper.Data",
			expectFix:      true,
		},
		{
			name: "valid schema names not modified",
			spec: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/UserResponse"
components:
  schemas:
    UserResponse:
      type: object
      properties:
        data:
          $ref: "#/components/schemas/User"
    User:
      type: object
`,
			strategy:       GenericNamingOf,
			expectedSchema: "UserResponse",
			expectedRef:    "#/components/schemas/UserResponse",
			expectFix:      false,
		},
		{
			name: "angle brackets renamed",
			spec: `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /data:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/List<Item>"
components:
  schemas:
    List<Item>:
      type: array
    Item:
      type: object
`,
			strategy:       GenericNamingOf,
			expectedSchema: "ListOfItem",
			expectedRef:    "#/components/schemas/ListOfItem",
			expectFix:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse
			parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(tt.spec)))
			require.NoError(t, err)

			// Fix with specific strategy
			f := New()
			f.EnabledFixes = []FixType{FixTypeRenamedGenericSchema}
			f.GenericNamingConfig.Strategy = tt.strategy
			result, err := f.FixParsed(*parseResult)
			require.NoError(t, err)

			// Assert
			doc := result.Document.(*parser.OAS3Document)

			if tt.expectFix {
				assert.True(t, result.HasFixes(), "expected fixes to be applied")
				assert.Contains(t, doc.Components.Schemas, tt.expectedSchema,
					"expected schema %s to exist", tt.expectedSchema)

				// Verify the ref was rewritten in paths
				pathItem := doc.Paths["/users"]
				if pathItem == nil {
					pathItem = doc.Paths["/data"]
				}
				require.NotNil(t, pathItem)
				require.NotNil(t, pathItem.Get)
				respContent := pathItem.Get.Responses.Codes["200"].Content["application/json"]
				assert.Equal(t, tt.expectedRef, respContent.Schema.Ref)
			} else {
				assert.False(t, result.HasFixes(), "expected no fixes to be applied")
				assert.Contains(t, doc.Components.Schemas, tt.expectedSchema)
			}
		})
	}
}

// TestFixInvalidSchemaNamesOAS2 tests renaming schemas with brackets in OAS 2.0
func TestFixInvalidSchemaNamesOAS2(t *testing.T) {
	tests := []struct {
		name           string
		spec           string
		strategy       GenericNamingStrategy
		expectedSchema string
		expectedRef    string
		expectFix      bool
	}{
		{
			name: "brackets renamed with underscore strategy",
			spec: `
swagger: "2.0"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getUsers
      produces:
        - application/json
      responses:
        "200":
          description: Success
          schema:
            $ref: "#/definitions/Response[User]"
definitions:
  Response[User]:
    type: object
    properties:
      data:
        $ref: "#/definitions/User"
  User:
    type: object
    properties:
      id:
        type: integer
`,
			strategy:       GenericNamingUnderscore,
			expectedSchema: "Response_User_",
			expectedRef:    "#/definitions/Response_User_",
			expectFix:      true,
		},
		{
			name: "brackets renamed with of strategy",
			spec: `
swagger: "2.0"
info:
  title: Test API
  version: "1.0"
paths:
  /items:
    get:
      operationId: getItems
      produces:
        - application/json
      responses:
        "200":
          description: Success
          schema:
            $ref: "#/definitions/List[Item]"
definitions:
  List[Item]:
    type: array
    items:
      $ref: "#/definitions/Item"
  Item:
    type: object
`,
			strategy:       GenericNamingOf,
			expectedSchema: "ListOfItem",
			expectedRef:    "#/definitions/ListOfItem",
			expectFix:      true,
		},
		{
			name: "valid schema names not modified",
			spec: `
swagger: "2.0"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "200":
          description: Success
          schema:
            $ref: "#/definitions/UserList"
definitions:
  UserList:
    type: array
    items:
      $ref: "#/definitions/User"
  User:
    type: object
`,
			strategy:       GenericNamingOf,
			expectedSchema: "UserList",
			expectedRef:    "#/definitions/UserList",
			expectFix:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse
			parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(tt.spec)))
			require.NoError(t, err)

			// Fix with specific strategy
			f := New()
			f.EnabledFixes = []FixType{FixTypeRenamedGenericSchema}
			f.GenericNamingConfig.Strategy = tt.strategy
			result, err := f.FixParsed(*parseResult)
			require.NoError(t, err)

			// Assert
			doc := result.Document.(*parser.OAS2Document)

			if tt.expectFix {
				assert.True(t, result.HasFixes(), "expected fixes to be applied")
				assert.Contains(t, doc.Definitions, tt.expectedSchema,
					"expected definition %s to exist", tt.expectedSchema)

				// Verify the ref was rewritten in responses
				pathItem := doc.Paths["/users"]
				if pathItem == nil {
					pathItem = doc.Paths["/items"]
				}
				require.NotNil(t, pathItem)
				require.NotNil(t, pathItem.Get)
				assert.Equal(t, tt.expectedRef, pathItem.Get.Responses.Codes["200"].Schema.Ref)
			} else {
				assert.False(t, result.HasFixes(), "expected no fixes to be applied")
				assert.Contains(t, doc.Definitions, tt.expectedSchema)
			}
		})
	}
}

// TestFixNestedGenericTypesOAS3 tests renaming nested generic types
func TestFixNestedGenericTypesOAS3(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /data:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Response[List[User]]"
components:
  schemas:
    Response[List[User]]:
      type: object
      properties:
        data:
          $ref: "#/components/schemas/List[User]"
    List[User]:
      type: array
      items:
        $ref: "#/components/schemas/User"
    User:
      type: object
      properties:
        id:
          type: integer
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix with "of" strategy
	f := New()
	f.EnabledFixes = []FixType{FixTypeRenamedGenericSchema}
	f.GenericNamingConfig.Strategy = GenericNamingOf
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert - nested generics should be transformed recursively
	doc := result.Document.(*parser.OAS3Document)

	// Should have 2 fixes (Response[List[User]] and List[User])
	assert.Equal(t, 2, result.FixCount)

	// Check the transformed names exist
	assert.Contains(t, doc.Components.Schemas, "ResponseOfListOfUser")
	assert.Contains(t, doc.Components.Schemas, "ListOfUser")
	assert.Contains(t, doc.Components.Schemas, "User") // unchanged

	// Verify refs were rewritten
	responseSchema := doc.Components.Schemas["ResponseOfListOfUser"]
	assert.Equal(t, "#/components/schemas/ListOfUser", responseSchema.Properties["data"].Ref)
}

// TestFixGenericSchemaNameCollision tests that name collisions are handled
func TestFixGenericSchemaNameCollision(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /data:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Response[User]"
components:
  schemas:
    Response[User]:
      type: object
      properties:
        data:
          type: string
    ResponseOfUser:
      type: object
      properties:
        existing:
          type: boolean
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix - should avoid collision with existing ResponseOfUser
	f := New()
	f.EnabledFixes = []FixType{FixTypeRenamedGenericSchema}
	f.GenericNamingConfig.Strategy = GenericNamingOf
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert
	doc := result.Document.(*parser.OAS3Document)
	assert.True(t, result.HasFixes())

	// Original ResponseOfUser should still exist
	assert.Contains(t, doc.Components.Schemas, "ResponseOfUser")

	// Renamed schema should have numeric suffix to avoid collision
	assert.Contains(t, doc.Components.Schemas, "ResponseOfUser2")

	// Response[User] should be gone
	assert.NotContains(t, doc.Components.Schemas, "Response[User]")
}

// =============================================================================
// Pruning Tests
// =============================================================================

// TestPruneUnusedSchemasOAS3 tests removing orphaned schemas in OAS 3.x
func TestPruneUnusedSchemasOAS3(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
    OrphanedSchema:
      type: object
      properties:
        unused:
          type: string
    AnotherOrphan:
      type: string
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix with pruning enabled
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert
	doc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, 2, result.FixCount) // 2 orphaned schemas removed

	// User should remain (referenced)
	assert.Contains(t, doc.Components.Schemas, "User")

	// Orphaned schemas should be removed
	assert.NotContains(t, doc.Components.Schemas, "OrphanedSchema")
	assert.NotContains(t, doc.Components.Schemas, "AnotherOrphan")
}

// TestPruneUnusedSchemasOAS2 tests removing orphaned schemas in OAS 2.0
func TestPruneUnusedSchemasOAS2(t *testing.T) {
	spec := `
swagger: "2.0"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getUsers
      produces:
        - application/json
      responses:
        "200":
          description: Success
          schema:
            $ref: "#/definitions/User"
definitions:
  User:
    type: object
    properties:
      id:
        type: integer
  UnusedDefinition:
    type: object
    properties:
      orphan:
        type: string
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix with pruning enabled
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert
	doc := result.Document.(*parser.OAS2Document)
	assert.Equal(t, 1, result.FixCount)

	// User should remain (referenced)
	assert.Contains(t, doc.Definitions, "User")

	// Orphaned definition should be removed
	assert.NotContains(t, doc.Definitions, "UnusedDefinition")
}

// TestPruneTransitiveReferencesPreserved tests that transitive refs are preserved
func TestPruneTransitiveReferencesPreserved(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/UserResponse"
components:
  schemas:
    UserResponse:
      type: object
      properties:
        user:
          $ref: "#/components/schemas/User"
    User:
      type: object
      properties:
        address:
          $ref: "#/components/schemas/Address"
    Address:
      type: object
      properties:
        city:
          type: string
    Orphan:
      type: object
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix with pruning enabled
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert
	doc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, 1, result.FixCount) // Only Orphan removed

	// All transitively referenced schemas should remain
	assert.Contains(t, doc.Components.Schemas, "UserResponse")
	assert.Contains(t, doc.Components.Schemas, "User")
	assert.Contains(t, doc.Components.Schemas, "Address")

	// Orphan should be removed
	assert.NotContains(t, doc.Components.Schemas, "Orphan")
}

// TestPruneCircularReferencesHandled tests that circular refs don't cause infinite loops
func TestPruneCircularReferencesHandled(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /nodes:
    get:
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Node"
components:
  schemas:
    Node:
      type: object
      properties:
        children:
          type: array
          items:
            $ref: "#/components/schemas/Node"
        parent:
          $ref: "#/components/schemas/Node"
    Orphan:
      type: object
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix with pruning enabled - should not hang on circular refs
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert
	doc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, 1, result.FixCount) // Only Orphan removed

	// Node (with circular refs) should remain
	assert.Contains(t, doc.Components.Schemas, "Node")

	// Orphan should be removed
	assert.NotContains(t, doc.Components.Schemas, "Orphan")
}

// TestPruneEmptyPaths tests removing paths with no operations
func TestPruneEmptyPaths(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      responses:
        "200":
          description: Success
  /empty:
    parameters:
      - name: id
        in: query
        schema:
          type: string
  /also-empty: {}
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix with path pruning enabled
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedEmptyPath}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert
	doc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, 2, result.FixCount) // Two empty paths removed

	// /users should remain (has operations)
	assert.Contains(t, doc.Paths, "/users")

	// Empty paths should be removed
	assert.NotContains(t, doc.Paths, "/empty")
	assert.NotContains(t, doc.Paths, "/also-empty")
}

// TestPruneEmptyPathsOAS2 tests removing empty paths in OAS 2.0
func TestPruneEmptyPathsOAS2(t *testing.T) {
	spec := `
swagger: "2.0"
info:
  title: Test API
  version: "1.0"
paths:
  /items:
    get:
      operationId: getItems
      responses:
        "200":
          description: Success
  /empty-path:
    parameters:
      - name: filter
        in: query
        type: string
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix with path pruning enabled
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedEmptyPath}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert
	doc := result.Document.(*parser.OAS2Document)
	assert.Equal(t, 1, result.FixCount)

	// /items should remain
	assert.Contains(t, doc.Paths, "/items")

	// Empty path should be removed
	assert.NotContains(t, doc.Paths, "/empty-path")
}

// TestPruneAllSchemasWhenNoneReferenced tests that schemas map becomes nil when all pruned
func TestPruneAllSchemasWhenNoneReferenced(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /health:
    get:
      responses:
        "200":
          description: OK
components:
  schemas:
    UnusedSchema:
      type: object
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert
	doc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, 1, result.FixCount)

	// Schemas should be nil when all are pruned
	assert.Nil(t, doc.Components.Schemas)
}

// TestPruneWithNilComponents tests pruning when components is nil
func TestPruneWithNilComponents(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /health:
    get:
      responses:
        "200":
          description: OK
`
	// Parse
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Fix - should not panic with nil components
	f := New()
	f.EnabledFixes = []FixType{FixTypePrunedUnusedSchema}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Assert - no fixes since no schemas
	assert.Equal(t, 0, result.FixCount)
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
