package fixer

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// Acceptance Test - THE MOST IMPORTANT TEST
// =============================================================================

// TestStubMissingRef_FixesValidationError proves the core value proposition:
// A document with a missing $ref fails validation, but passes after fixing.
func TestStubMissingRef_FixesValidationError(t *testing.T) {
	// 1. Document with missing ref - this is INVALID
	input := `{
		"swagger": "2.0",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/test": {
				"get": {
					"operationId": "getTest",
					"responses": {
						"200": {
							"description": "ok",
							"schema": {"$ref": "#/definitions/foo.Bar"}
						}
					}
				}
			}
		},
		"definitions": {}
	}`

	// 2. Parse and validate - should FAIL
	doc, err := parser.ParseWithOptions(parser.WithBytes([]byte(input)))
	require.NoError(t, err)

	v := validator.New()
	result, err := v.ValidateParsed(*doc)
	require.NoError(t, err)
	require.False(t, result.Valid, "should fail validation before fix")

	// Check for specific error about unresolved ref
	require.NotEmpty(t, result.Errors, "should have validation errors")
	hasRefError := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "foo.Bar") || strings.Contains(e.Message, "does not resolve") {
			hasRefError = true
			break
		}
	}
	require.True(t, hasRefError, "should have error about missing ref")

	// 3. Fix with stub-missing-refs enabled
	fixed, err := FixWithOptions(
		WithParsed(*doc),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)
	require.Equal(t, 1, fixed.FixCount, "should have exactly 1 fix")
	require.Equal(t, FixTypeStubMissingRef, fixed.Fixes[0].Type)

	// 4. Validate again - should PASS
	result2, err := v.ValidateParsed(*fixed.ToParseResult())
	require.NoError(t, err)
	require.True(t, result2.Valid, "should pass validation after stubbing")
}

// TestStubMissingRef_FixesValidationError_OAS3 proves the same for OAS 3.x
func TestStubMissingRef_FixesValidationError_OAS3(t *testing.T) {
	// Document with missing ref in OAS 3.0
	input := `{
		"openapi": "3.0.3",
		"info": {"title": "Test", "version": "1.0"},
		"paths": {
			"/test": {
				"get": {
					"operationId": "getTest",
					"responses": {
						"200": {
							"description": "ok",
							"content": {
								"application/json": {
									"schema": {"$ref": "#/components/schemas/MissingSchema"}
								}
							}
						}
					}
				}
			}
		}
	}`

	// Parse and validate - should FAIL
	doc, err := parser.ParseWithOptions(parser.WithBytes([]byte(input)))
	require.NoError(t, err)

	v := validator.New()
	result, err := v.ValidateParsed(*doc)
	require.NoError(t, err)
	require.False(t, result.Valid, "should fail validation before fix")

	// Fix with stub-missing-refs enabled
	fixed, err := FixWithOptions(
		WithParsed(*doc),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)
	require.Equal(t, 1, fixed.FixCount, "should have exactly 1 fix")

	// Validate again - should PASS
	result2, err := v.ValidateParsed(*fixed.ToParseResult())
	require.NoError(t, err)
	require.True(t, result2.Valid, "should pass validation after stubbing")
}

// =============================================================================
// OAS 2.0 Tests
// =============================================================================

// TestStubMissingSchema_OAS2 tests stubbing a missing schema definition
func TestStubMissingSchema_OAS2(t *testing.T) {
	spec := `
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
            $ref: "#/definitions/Foo"
definitions: {}
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)

	// Verify fix was applied
	require.Equal(t, 1, result.FixCount)
	require.Equal(t, FixTypeStubMissingRef, result.Fixes[0].Type)
	assert.Contains(t, result.Fixes[0].Description, "Foo")
	assert.Equal(t, "definitions.Foo", result.Fixes[0].Path)

	// Verify stub was created
	doc := result.Document.(*parser.OAS2Document)
	require.NotNil(t, doc.Definitions)
	require.Contains(t, doc.Definitions, "Foo")
	assert.NotNil(t, doc.Definitions["Foo"])
}

// TestStubMissingResponse_OAS2 tests stubbing a missing response reference
func TestStubMissingResponse_OAS2(t *testing.T) {
	spec := `
swagger: "2.0"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "404":
          $ref: "#/responses/NotFound"
responses: {}
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)

	// Verify fix was applied
	require.Equal(t, 1, result.FixCount)
	require.Equal(t, FixTypeStubMissingRef, result.Fixes[0].Type)
	assert.Contains(t, result.Fixes[0].Description, "NotFound")
	assert.Equal(t, "responses.NotFound", result.Fixes[0].Path)

	// Verify stub was created with description
	doc := result.Document.(*parser.OAS2Document)
	require.NotNil(t, doc.Responses)
	require.Contains(t, doc.Responses, "NotFound")
	assert.NotNil(t, doc.Responses["NotFound"])
	assert.NotEmpty(t, doc.Responses["NotFound"].Description)
}

// =============================================================================
// OAS 3.x Tests
// =============================================================================

// TestStubMissingSchema_OAS3 tests stubbing a missing schema in components
func TestStubMissingSchema_OAS3(t *testing.T) {
	spec := `
openapi: "3.0.3"
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
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Foo"
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)

	// Verify fix was applied
	require.Equal(t, 1, result.FixCount)
	require.Equal(t, FixTypeStubMissingRef, result.Fixes[0].Type)
	assert.Contains(t, result.Fixes[0].Description, "Foo")
	assert.Equal(t, "components.schemas.Foo", result.Fixes[0].Path)

	// Verify stub was created
	doc := result.Document.(*parser.OAS3Document)
	require.NotNil(t, doc.Components)
	require.NotNil(t, doc.Components.Schemas)
	require.Contains(t, doc.Components.Schemas, "Foo")
	assert.NotNil(t, doc.Components.Schemas["Foo"])
}

// TestStubMissingResponse_OAS3 tests stubbing a missing response reference
func TestStubMissingResponse_OAS3(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "404":
          $ref: "#/components/responses/NotFound"
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)

	// Verify fix was applied
	require.Equal(t, 1, result.FixCount)
	require.Equal(t, FixTypeStubMissingRef, result.Fixes[0].Type)
	assert.Contains(t, result.Fixes[0].Description, "NotFound")
	assert.Equal(t, "components.responses.NotFound", result.Fixes[0].Path)

	// Verify stub was created with description
	doc := result.Document.(*parser.OAS3Document)
	require.NotNil(t, doc.Components)
	require.NotNil(t, doc.Components.Responses)
	require.Contains(t, doc.Components.Responses, "NotFound")
	assert.NotNil(t, doc.Components.Responses["NotFound"])
	assert.NotEmpty(t, doc.Components.Responses["NotFound"].Description)
}

// =============================================================================
// Multiple Refs and Edge Cases
// =============================================================================

// TestStubMissing_MultipleRefs tests that multiple missing refs all get stubbed
func TestStubMissing_MultipleRefs(t *testing.T) {
	spec := `
openapi: "3.0.3"
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
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
        "404":
          $ref: "#/components/responses/NotFound"
  /items:
    get:
      operationId: getItems
      responses:
        "200":
          description: Success
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Item"
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)

	// Verify all fixes were applied
	require.Equal(t, 3, result.FixCount, "should have 3 fixes: User, Item, NotFound")

	// Verify all stubs were created
	doc := result.Document.(*parser.OAS3Document)
	require.NotNil(t, doc.Components)
	require.Contains(t, doc.Components.Schemas, "User")
	require.Contains(t, doc.Components.Schemas, "Item")
	require.Contains(t, doc.Components.Responses, "NotFound")
}

// TestStubMissing_ExistingNotTouched tests that existing definitions are not modified
func TestStubMissing_ExistingNotTouched(t *testing.T) {
	spec := `
openapi: "3.0.3"
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
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ExistingSchema"
        "201":
          description: Created
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/MissingSchema"
components:
  schemas:
    ExistingSchema:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)

	// Only 1 fix should be applied (for MissingSchema)
	require.Equal(t, 1, result.FixCount)

	// Verify existing schema is untouched
	doc := result.Document.(*parser.OAS3Document)
	existing := doc.Components.Schemas["ExistingSchema"]
	require.NotNil(t, existing)
	assert.Equal(t, "object", existing.Type)
	require.NotNil(t, existing.Properties)
	assert.Contains(t, existing.Properties, "id")
	assert.Contains(t, existing.Properties, "name")

	// Verify missing schema was stubbed
	require.Contains(t, doc.Components.Schemas, "MissingSchema")
}

// TestStubMissing_ExternalRefIgnored tests that external refs are not stubbed
func TestStubMissing_ExternalRefIgnored(t *testing.T) {
	spec := `
openapi: "3.0.3"
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
          content:
            application/json:
              schema:
                $ref: "./other.yaml#/components/schemas/ExternalSchema"
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)

	// No fixes should be applied for external refs
	assert.Equal(t, 0, result.FixCount, "should not stub external refs")
}

// TestStubMissing_NilMapsInitialized_OAS2 tests that nil maps are initialized
func TestStubMissing_NilMapsInitialized_OAS2(t *testing.T) {
	spec := `
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
            $ref: "#/definitions/User"
`
	// Note: No definitions key at all - the map will be nil
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)

	// Verify fix was applied
	require.Equal(t, 1, result.FixCount)

	// Verify map was initialized and stub added
	doc := result.Document.(*parser.OAS2Document)
	require.NotNil(t, doc.Definitions, "Definitions map should be initialized")
	require.Contains(t, doc.Definitions, "User")
}

// TestStubMissing_NilMapsInitialized_OAS3 tests that nil Components and maps are initialized
func TestStubMissing_NilMapsInitialized_OAS3(t *testing.T) {
	spec := `
openapi: "3.0.3"
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
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
`
	// Note: No components key at all - Components will be nil
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)

	// Verify fix was applied
	require.Equal(t, 1, result.FixCount)

	// Verify Components and Schemas were initialized
	doc := result.Document.(*parser.OAS3Document)
	require.NotNil(t, doc.Components, "Components should be initialized")
	require.NotNil(t, doc.Components.Schemas, "Schemas map should be initialized")
	require.Contains(t, doc.Components.Schemas, "User")
}

// =============================================================================
// Configuration Tests
// =============================================================================

// TestStubMissing_CustomResponseDesc tests custom description for response stubs
func TestStubMissing_CustomResponseDesc(t *testing.T) {
	spec := `
swagger: "2.0"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "404":
          $ref: "#/responses/NotFound"
responses: {}
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	customDesc := "Custom stub description for testing"
	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
		WithStubResponseDescription(customDesc),
	)
	require.NoError(t, err)

	// Verify stub has custom description
	doc := result.Document.(*parser.OAS2Document)
	require.Contains(t, doc.Responses, "NotFound")
	assert.Equal(t, customDesc, doc.Responses["NotFound"].Description)
}

// TestStubMissing_CustomResponseDesc_OAS3 tests custom description for OAS 3.x
func TestStubMissing_CustomResponseDesc_OAS3(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "404":
          $ref: "#/components/responses/NotFound"
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	customDesc := "Custom OAS3 stub description"
	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
		WithStubResponseDescription(customDesc),
	)
	require.NoError(t, err)

	// Verify stub has custom description
	doc := result.Document.(*parser.OAS3Document)
	require.NotNil(t, doc.Components)
	require.Contains(t, doc.Components.Responses, "NotFound")
	assert.Equal(t, customDesc, doc.Components.Responses["NotFound"].Description)
}

// TestStubMissing_DisabledByDefault tests that stub-missing-refs doesn't run by default
func TestStubMissing_DisabledByDefault(t *testing.T) {
	spec := `
openapi: "3.0.3"
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
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/MissingSchema"
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Use default fixer (no explicit enabled fixes for stub)
	result, err := FixWithOptions(
		WithParsed(*parseResult),
		// Default EnabledFixes is only FixTypeMissingPathParameter
	)
	require.NoError(t, err)

	// Should have no stub fixes
	stubFixes := 0
	for _, fix := range result.Fixes {
		if fix.Type == FixTypeStubMissingRef {
			stubFixes++
		}
	}
	assert.Equal(t, 0, stubFixes, "stub-missing-refs should not run by default")

	// Verify schema was NOT stubbed
	doc := result.Document.(*parser.OAS3Document)
	if doc.Components != nil && doc.Components.Schemas != nil {
		assert.NotContains(t, doc.Components.Schemas, "MissingSchema",
			"missing schema should not be stubbed when fix is disabled")
	}
}

// TestStubMissing_DryRun tests that dry run does not create stubs
func TestStubMissing_DryRun(t *testing.T) {
	spec := `
openapi: "3.0.3"
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
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/MissingSchema"
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
		WithDryRun(true),
	)
	require.NoError(t, err)

	// In dry run mode, stub-missing-refs is skipped entirely (per pipeline.go)
	// So no fixes should be reported and no stubs created
	assert.Equal(t, 0, result.FixCount, "dry run should not report stub fixes")

	// Verify schema was NOT stubbed
	doc := result.Document.(*parser.OAS3Document)
	if doc.Components != nil && doc.Components.Schemas != nil {
		assert.NotContains(t, doc.Components.Schemas, "MissingSchema",
			"dry run should not create stubs")
	}
}

// =============================================================================
// Helper Function Tests
// =============================================================================

// TestIsLocalRef tests the isLocalRef helper function
func TestIsLocalRef(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected bool
	}{
		{
			name:     "local ref with definitions",
			ref:      "#/definitions/User",
			expected: true,
		},
		{
			name:     "local ref with components",
			ref:      "#/components/schemas/User",
			expected: true,
		},
		{
			name:     "external file ref",
			ref:      "./other.yaml#/definitions/User",
			expected: false,
		},
		{
			name:     "URL ref",
			ref:      "https://example.com/spec.yaml#/components/schemas/User",
			expected: false,
		},
		{
			name:     "relative file ref",
			ref:      "common/schemas.yaml#/User",
			expected: false,
		},
		{
			name:     "empty ref",
			ref:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLocalRef(tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestExtractResponseNameFromRef tests the extractResponseNameFromRef helper
func TestExtractResponseNameFromRef(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		version  parser.OASVersion
		expected string
	}{
		{
			name:     "OAS 2.0 response ref",
			ref:      "#/responses/NotFound",
			version:  parser.OASVersion20,
			expected: "NotFound",
		},
		{
			name:     "OAS 3.0 response ref",
			ref:      "#/components/responses/NotFound",
			version:  parser.OASVersion303,
			expected: "NotFound",
		},
		{
			name:     "OAS 3.1 response ref",
			ref:      "#/components/responses/ServerError",
			version:  parser.OASVersion310,
			expected: "ServerError",
		},
		{
			name:     "non-response ref OAS 2.0",
			ref:      "#/definitions/User",
			version:  parser.OASVersion20,
			expected: "",
		},
		{
			name:     "non-response ref OAS 3.0",
			ref:      "#/components/schemas/User",
			version:  parser.OASVersion303,
			expected: "",
		},
		{
			name:     "external ref",
			ref:      "./other.yaml#/responses/NotFound",
			version:  parser.OASVersion20,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractResponseNameFromRef(tt.ref, tt.version)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Nil Safety Tests
// =============================================================================

// TestStubMissingRefsOAS2_NilDoc tests that nil document doesn't panic
func TestStubMissingRefsOAS2_NilDoc(t *testing.T) {
	f := New()
	result := &FixResult{Fixes: make([]Fix, 0)}

	// Should not panic
	f.stubMissingRefsOAS2(nil, result)
	assert.Equal(t, 0, result.FixCount)
}

// TestStubMissingRefsOAS3_NilDoc tests that nil document doesn't panic
func TestStubMissingRefsOAS3_NilDoc(t *testing.T) {
	f := New()
	result := &FixResult{Fixes: make([]Fix, 0)}

	// Should not panic
	f.stubMissingRefsOAS3(nil, result)
	assert.Equal(t, 0, result.FixCount)
}

// =============================================================================
// StubConfig Tests
// =============================================================================

// TestDefaultStubConfig tests that DefaultStubConfig returns sensible defaults
func TestDefaultStubConfig(t *testing.T) {
	config := DefaultStubConfig()
	assert.NotEmpty(t, config.ResponseDescription, "should have default response description")
	assert.Contains(t, config.ResponseDescription, "stub", "description should mention stub")
}

// TestWithStubConfig tests the WithStubConfig option
func TestWithStubConfig(t *testing.T) {
	spec := `
swagger: "2.0"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "404":
          $ref: "#/responses/NotFound"
responses: {}
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	customConfig := StubConfig{
		ResponseDescription: "Custom response via config",
	}
	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
		WithStubConfig(customConfig),
	)
	require.NoError(t, err)

	doc := result.Document.(*parser.OAS2Document)
	require.Contains(t, doc.Responses, "NotFound")
	assert.Equal(t, customConfig.ResponseDescription, doc.Responses["NotFound"].Description)
}

// =============================================================================
// Complex Reference Scenarios
// =============================================================================

// TestStubMissing_NestedRefs tests refs inside nested schemas
func TestStubMissing_NestedRefs(t *testing.T) {
	spec := `
openapi: "3.0.3"
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
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/User"
components:
  schemas:
    User:
      type: object
      properties:
        address:
          $ref: "#/components/schemas/Address"
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)

	// Address should be stubbed
	require.Equal(t, 1, result.FixCount)

	doc := result.Document.(*parser.OAS3Document)
	require.Contains(t, doc.Components.Schemas, "Address")
}

// TestStubMissing_AllOfRefs tests refs inside allOf composition
func TestStubMissing_AllOfRefs(t *testing.T) {
	spec := `
openapi: "3.0.3"
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
          content:
            application/json:
              schema:
                allOf:
                  - $ref: "#/components/schemas/BaseModel"
                  - type: object
                    properties:
                      id:
                        type: integer
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)

	// BaseModel should be stubbed
	require.Equal(t, 1, result.FixCount)

	doc := result.Document.(*parser.OAS3Document)
	require.NotNil(t, doc.Components)
	require.Contains(t, doc.Components.Schemas, "BaseModel")
}

// TestStubMissing_DotInSchemaName tests schema names with dots (common in Java/C# APIs)
func TestStubMissing_DotInSchemaName(t *testing.T) {
	spec := `
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
            $ref: "#/definitions/com.example.User"
definitions: {}
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)

	// Schema with dots should be stubbed
	require.Equal(t, 1, result.FixCount)

	doc := result.Document.(*parser.OAS2Document)
	require.Contains(t, doc.Definitions, "com.example.User")
}

// TestStubMissing_NilResponsesMap_OAS2 tests stubbing response with nil Responses map
func TestStubMissing_NilResponsesMap_OAS2(t *testing.T) {
	spec := `
swagger: "2.0"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "404":
          $ref: "#/responses/NotFound"
`
	// Note: No "responses:" key at all
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)

	require.Equal(t, 1, result.FixCount)

	doc := result.Document.(*parser.OAS2Document)
	require.NotNil(t, doc.Responses, "Responses map should be initialized")
	require.Contains(t, doc.Responses, "NotFound")
}

// TestStubMissing_NilResponsesMap_OAS3 tests stubbing response with nil Responses map
func TestStubMissing_NilResponsesMap_OAS3(t *testing.T) {
	spec := `
openapi: "3.0.3"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "404":
          $ref: "#/components/responses/NotFound"
components:
  schemas: {}
`
	// Note: components exists but no responses map
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)

	require.Equal(t, 1, result.FixCount)

	doc := result.Document.(*parser.OAS3Document)
	require.NotNil(t, doc.Components.Responses, "Responses map should be initialized")
	require.Contains(t, doc.Components.Responses, "NotFound")
}

// TestStubMissing_EmptyStubConfigDescription tests that empty description falls back to default
func TestStubMissing_EmptyStubConfigDescription(t *testing.T) {
	spec := `
swagger: "2.0"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "404":
          $ref: "#/responses/NotFound"
responses: {}
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	// Use empty description in config - should fall back to default
	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
		WithStubConfig(StubConfig{ResponseDescription: ""}),
	)
	require.NoError(t, err)

	doc := result.Document.(*parser.OAS2Document)
	require.Contains(t, doc.Responses, "NotFound")
	// Should use default description, not empty
	assert.Equal(t, DefaultStubConfig().ResponseDescription, doc.Responses["NotFound"].Description)
}

// TestStubMissing_SchemaAlreadyExists tests that existing schema is not re-stubbed
func TestStubMissing_SchemaAlreadyExists(t *testing.T) {
	spec := `
openapi: "3.0.3"
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
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ExistingUser"
components:
  schemas:
    ExistingUser:
      type: object
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)

	// No fixes needed - schema already exists
	assert.Equal(t, 0, result.FixCount)
}

// TestStubMissing_ResponseAlreadyExists tests that existing response is not re-stubbed
func TestStubMissing_ResponseAlreadyExists(t *testing.T) {
	spec := `
swagger: "2.0"
info:
  title: Test API
  version: "1.0"
paths:
  /users:
    get:
      operationId: getUsers
      responses:
        "404":
          $ref: "#/responses/NotFound"
responses:
  NotFound:
    description: Resource not found
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)

	// No fixes needed - response already exists
	assert.Equal(t, 0, result.FixCount)

	// Verify original description is preserved
	doc := result.Document.(*parser.OAS2Document)
	assert.Equal(t, "Resource not found", doc.Responses["NotFound"].Description)
}

// TestStubMissing_MixedLocalAndExternalRefs tests document with both local and external refs
func TestStubMissing_MixedLocalAndExternalRefs(t *testing.T) {
	spec := `
openapi: "3.0.3"
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
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/LocalSchema"
        "400":
          description: Bad request
          content:
            application/json:
              schema:
                $ref: "./external.yaml#/components/schemas/ExternalError"
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeStubMissingRef),
	)
	require.NoError(t, err)

	// Only local ref should be stubbed
	require.Equal(t, 1, result.FixCount)

	doc := result.Document.(*parser.OAS3Document)
	require.Contains(t, doc.Components.Schemas, "LocalSchema")
	// External schema should NOT be in components
	assert.NotContains(t, doc.Components.Schemas, "ExternalError")
}
