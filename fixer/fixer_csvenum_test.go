package fixer

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// CSV Enum Expansion Tests
// =============================================================================

// TestFix_CSVEnumExpansion_OAS2 tests CSV enum expansion for OAS 2.0 documents
func TestFix_CSVEnumExpansion_OAS2(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Definitions: map[string]*parser.Schema{
			"Status": {
				Type: "integer",
				Enum: []any{"1,2,3,5,10"},
			},
		},
	}

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}

	result, err := f.FixParsed(parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion20,
		Version:    "2.0",
	})

	require.NoError(t, err)
	require.True(t, result.HasFixes())
	assert.Equal(t, 1, result.FixCount)

	fixedDoc := result.Document.(*parser.OAS2Document)
	assert.Equal(t, []any{int64(1), int64(2), int64(3), int64(5), int64(10)}, fixedDoc.Definitions["Status"].Enum)
}

// TestFix_CSVEnumExpansion_OAS3 tests CSV enum expansion for OAS 3.x documents
func TestFix_CSVEnumExpansion_OAS3(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Weight": {
					Type: "number",
					Enum: []any{"0.5,1.0,2.5,5.0"},
				},
			},
		},
	}

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}

	result, err := f.FixParsed(parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion300,
		Version:    "3.0.0",
	})

	require.NoError(t, err)
	require.True(t, result.HasFixes())

	fixedDoc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, []any{0.5, 1.0, 2.5, 5.0}, fixedDoc.Components.Schemas["Weight"].Enum)
}

// TestFix_CSVEnumExpansion_NotEnabledByDefault tests that CSV enum fix is not enabled by default
func TestFix_CSVEnumExpansion_NotEnabledByDefault(t *testing.T) {
	f := New()
	assert.NotContains(t, f.EnabledFixes, FixTypeEnumCSVExpanded)
}

// TestFix_CSVEnumExpansion_NestedSchema tests CSV enum expansion in nested object properties
func TestFix_CSVEnumExpansion_NestedSchema(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Definitions: map[string]*parser.Schema{
			"Pet": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"age": {
						Type: "integer",
						Enum: []any{"1,2,3,5,10,15"},
					},
				},
			},
		},
	}

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}

	result, err := f.FixParsed(parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion20,
		Version:    "2.0",
	})

	require.NoError(t, err)
	require.True(t, result.HasFixes())

	fixedDoc := result.Document.(*parser.OAS2Document)
	assert.Equal(t, []any{int64(1), int64(2), int64(3), int64(5), int64(10), int64(15)}, fixedDoc.Definitions["Pet"].Properties["age"].Enum)
}

// TestFix_CSVEnumExpansion_NoChangesWhenNoCSV tests that no fixes are applied when enums are already proper arrays
func TestFix_CSVEnumExpansion_NoChangesWhenNoCSV(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Status": {
					Type: "integer",
					Enum: []any{int64(1), int64(2), int64(3)}, // Already proper array
				},
			},
		},
	}

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}

	result, err := f.FixParsed(parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion300,
		Version:    "3.0.0",
	})

	require.NoError(t, err)
	assert.False(t, result.HasFixes())
	assert.Equal(t, 0, result.FixCount)
}

// TestFix_CSVEnumExpansion_StringEnumsNotAffected tests that string type enums are not expanded
func TestFix_CSVEnumExpansion_StringEnumsNotAffected(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Status": {
					Type: "string",
					Enum: []any{"active,inactive,pending"}, // CSV in string type - intentional
				},
			},
		},
	}

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}

	result, err := f.FixParsed(parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion300,
		Version:    "3.0.0",
	})

	require.NoError(t, err)
	assert.False(t, result.HasFixes())

	// The enum should remain unchanged
	fixedDoc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, []any{"active,inactive,pending"}, fixedDoc.Components.Schemas["Status"].Enum)
}

// TestFix_CSVEnumExpansion_WithOptions tests CSV enum expansion using functional options
func TestFix_CSVEnumExpansion_WithOptions(t *testing.T) {
	spec := `
openapi: "3.0.0"
info:
  title: Test API
  version: "1.0"
paths: {}
components:
  schemas:
    Priority:
      type: integer
      enum:
        - "1,2,3,4,5"
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	result, err := FixWithOptions(
		WithParsed(*parseResult),
		WithEnabledFixes(FixTypeEnumCSVExpanded),
	)

	require.NoError(t, err)
	require.True(t, result.HasFixes())
	assert.Equal(t, 1, result.FixCount)

	fixedDoc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, []any{int64(1), int64(2), int64(3), int64(4), int64(5)}, fixedDoc.Components.Schemas["Priority"].Enum)
}

// TestFix_CSVEnumExpansion_OAS31TypeArray tests CSV enum expansion with OAS 3.1 type arrays
func TestFix_CSVEnumExpansion_OAS31TypeArray(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"NullableStatus": {
					Type: []any{"integer", "null"}, // OAS 3.1 type array
					Enum: []any{"1,2,3"},
				},
			},
		},
	}

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}

	result, err := f.FixParsed(parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
		Version:    "3.1.0",
	})

	require.NoError(t, err)
	require.True(t, result.HasFixes())

	fixedDoc := result.Document.(*parser.OAS3Document)
	assert.Equal(t, []any{int64(1), int64(2), int64(3)}, fixedDoc.Components.Schemas["NullableStatus"].Enum)
}

// TestFix_CSVEnumExpansion_FixDescriptionContainsCount tests that fix description contains the value count
func TestFix_CSVEnumExpansion_FixDescriptionContainsCount(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Definitions: map[string]*parser.Schema{
			"Status": {
				Type: "integer",
				Enum: []any{"1,2,3,4,5"},
			},
		},
	}

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}

	result, err := f.FixParsed(parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion20,
		Version:    "2.0",
	})

	require.NoError(t, err)
	require.Len(t, result.Fixes, 1)
	assert.Contains(t, result.Fixes[0].Description, "5 individual values")
}

// TestFix_CSVEnumExpansion_OAS2PathParameter tests CSV enum expansion in OAS 2.0 path parameters
func TestFix_CSVEnumExpansion_OAS2PathParameter(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]*parser.PathItem{
			"/items/{status}": {
				Get: &parser.Operation{
					OperationID: "getItems",
					Parameters: []*parser.Parameter{
						{
							Name: "status",
							In:   "path",
							Schema: &parser.Schema{
								Type: "integer",
								Enum: []any{"1,2,3"},
							},
						},
					},
				},
			},
		},
	}
	parseResult := &parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	fixed := result.Document.(*parser.OAS2Document)
	param := fixed.Paths["/items/{status}"].Get.Parameters[0]
	assert.Equal(t, []any{int64(1), int64(2), int64(3)}, param.Schema.Enum)
	assert.Len(t, result.Fixes, 1)
}

// TestFix_CSVEnumExpansion_OAS3PathParameter tests CSV enum expansion in OAS 3.x path parameters
func TestFix_CSVEnumExpansion_OAS3PathParameter(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]*parser.PathItem{
			"/items/{status}": {
				Get: &parser.Operation{
					OperationID: "getItems",
					Parameters: []*parser.Parameter{
						{
							Name: "status",
							In:   "path",
							Schema: &parser.Schema{
								Type: "integer",
								Enum: []any{"1,2,3"},
							},
						},
					},
				},
			},
		},
	}
	parseResult := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	fixed := result.Document.(*parser.OAS3Document)
	param := fixed.Paths["/items/{status}"].Get.Parameters[0]
	assert.Equal(t, []any{int64(1), int64(2), int64(3)}, param.Schema.Enum)
	assert.Len(t, result.Fixes, 1)
}

// TestFix_CSVEnumExpansion_OAS3RequestBody tests CSV enum expansion in OAS 3.x request bodies
func TestFix_CSVEnumExpansion_OAS3RequestBody(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]*parser.PathItem{
			"/items": {
				Post: &parser.Operation{
					OperationID: "createItem",
					RequestBody: &parser.RequestBody{
						Content: map[string]*parser.MediaType{
							"application/json": {
								Schema: &parser.Schema{
									Type: "object",
									Properties: map[string]*parser.Schema{
										"priority": {
											Type: "integer",
											Enum: []any{"1,2,3,4,5"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	parseResult := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	fixed := result.Document.(*parser.OAS3Document)
	schema := fixed.Paths["/items"].Post.RequestBody.Content["application/json"].Schema
	assert.Equal(t, []any{int64(1), int64(2), int64(3), int64(4), int64(5)}, schema.Properties["priority"].Enum)
	assert.Len(t, result.Fixes, 1)
}

// TestFix_CSVEnumExpansion_OAS3Response tests CSV enum expansion in OAS 3.x responses
func TestFix_CSVEnumExpansion_OAS3Response(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]*parser.PathItem{
			"/items": {
				Get: &parser.Operation{
					OperationID: "getItems",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "Success",
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{
											Type: "object",
											Properties: map[string]*parser.Schema{
												"status": {
													Type: "integer",
													Enum: []any{"0,1,2"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	parseResult := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	fixed := result.Document.(*parser.OAS3Document)
	schema := fixed.Paths["/items"].Get.Responses.Codes["200"].Content["application/json"].Schema
	assert.Equal(t, []any{int64(0), int64(1), int64(2)}, schema.Properties["status"].Enum)
	assert.Len(t, result.Fixes, 1)
}

// TestFix_CSVEnumExpansion_AllInvalidPartsNoFix tests that when all CSV parts are invalid,
// no fix is applied (the empty expansion guard)
func TestFix_CSVEnumExpansion_AllInvalidPartsNoFix(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"BadEnum": {
					Type: "integer",
					Enum: []any{"abc,def,ghi"}, // All invalid
				},
			},
		},
	}
	parseResult := &parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	// Should NOT apply a fix since all parts are invalid
	assert.Empty(t, result.Fixes, "should not apply fix when all CSV parts are invalid")

	// Enum should remain unchanged
	fixed := result.Document.(*parser.OAS3Document)
	assert.Equal(t, []any{"abc,def,ghi"}, fixed.Components.Schemas["BadEnum"].Enum)
}
