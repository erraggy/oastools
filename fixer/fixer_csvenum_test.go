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

// TestFix_CSVEnumExpansion_OAS2ParameterOwnEnum reproduces #513: an OAS 2.0
// non-body parameter declares type and enum on the parameter object itself, and
// the same enum reached through a body parameter's schema was expanded while
// the parameter's own was left as it was.
func TestFix_CSVEnumExpansion_OAS2ParameterOwnEnum(t *testing.T) {
	spec := `
swagger: "2.0"
info:
  title: Petstore
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      parameters:
        - name: litterSize
          in: query
          type: integer
          enum: ["1,2,3"]
        - name: body
          in: body
          schema:
            type: object
            properties:
              litterSize:
                type: integer
                enum: ["1,2,3"]
      responses:
        "200":
          description: OK
`
	parseResult, err := parser.ParseWithOptions(parser.WithBytes([]byte(spec)))
	require.NoError(t, err)

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}
	result, err := f.FixParsed(*parseResult)
	require.NoError(t, err)

	fixed := result.Document.(*parser.OAS2Document)
	params := fixed.Paths["/pets"].Get.Parameters
	expanded := []any{int64(1), int64(2), int64(3)}
	assert.Equal(t, expanded, params[0].Enum, "query parameter's own enum")
	assert.Equal(t, expanded, params[1].Schema.Properties["litterSize"].Enum, "body schema property enum")

	require.Len(t, result.Fixes, 2)
	assert.Equal(t, "paths./pets.get.parameters[0]", result.Fixes[0].Path)
	assert.Equal(t, "paths./pets.get.parameters[1].schema.properties.litterSize", result.Fixes[1].Path)
}

// TestFix_CSVEnumExpansion_OAS2ItemsEnum tests that an array parameter's enum,
// which OAS 2.0 places on the items object, is expanded.
func TestFix_CSVEnumExpansion_OAS2ItemsEnum(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]*parser.PathItem{
			"/items": {
				Get: &parser.Operation{
					OperationID: "listItems",
					Parameters: []*parser.Parameter{
						{
							Name:  "sizes",
							In:    "query",
							Type:  "array",
							Items: &parser.Items{Type: "number", Enum: []any{"1.5,2.5"}},
						},
						{
							Name: "grid",
							In:   "query",
							Type: "array",
							Items: &parser.Items{
								Type:  "array",
								Items: &parser.Items{Type: "integer", Enum: []any{"1,2"}},
							},
						},
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

	fixed := result.Document.(*parser.OAS2Document)
	params := fixed.Paths["/items"].Get.Parameters
	assert.Equal(t, []any{1.5, 2.5}, params[0].Items.Enum)
	assert.Equal(t, []any{int64(1), int64(2)}, params[1].Items.Items.Enum)

	require.Len(t, result.Fixes, 2)
	assert.Equal(t, "paths./items.get.parameters[0].items", result.Fixes[0].Path)
	assert.Equal(t, "paths./items.get.parameters[1].items.items", result.Fixes[1].Path)
}

// TestFix_CSVEnumExpansion_OAS2ResponseHeader tests that an OAS 2.0 response
// header, which declares type and enum directly as a non-body parameter does,
// is expanded.
func TestFix_CSVEnumExpansion_OAS2ResponseHeader(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]*parser.PathItem{
			"/items": {
				Get: &parser.Operation{
					OperationID: "listItems",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
								Headers: map[string]*parser.Header{
									"X-Rate-Limit": {Type: "integer", Enum: []any{"10,20,30"}},
								},
							},
						},
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

	fixed := result.Document.(*parser.OAS2Document)
	header := fixed.Paths["/items"].Get.Responses.Codes["200"].Headers["X-Rate-Limit"]
	assert.Equal(t, []any{int64(10), int64(20), int64(30)}, header.Enum)
	require.Len(t, result.Fixes, 1)
	assert.Equal(t, "paths./items.get.responses.200.headers.X-Rate-Limit", result.Fixes[0].Path)
}

// TestFix_CSVEnumExpansion_OAS2Definitions tests the reusable parameter and
// response definitions, which an operation reaches only through a $ref.
func TestFix_CSVEnumExpansion_OAS2Definitions(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Parameters: map[string]*parser.Parameter{
			"LitterSize": {Name: "litterSize", In: "query", Type: "integer", Enum: []any{"1,2,3"}},
		},
		Responses: map[string]*parser.Response{
			"Limited": {
				Description: "OK",
				Headers: map[string]*parser.Header{
					"X-Rate-Limit": {Type: "integer", Enum: []any{"10,20"}},
				},
				Schema: &parser.Schema{Type: "integer", Enum: []any{"4,5"}},
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

	fixed := result.Document.(*parser.OAS2Document)
	assert.Equal(t, []any{int64(1), int64(2), int64(3)}, fixed.Parameters["LitterSize"].Enum)
	assert.Equal(t, []any{int64(10), int64(20)}, fixed.Responses["Limited"].Headers["X-Rate-Limit"].Enum)
	assert.Equal(t, []any{int64(4), int64(5)}, fixed.Responses["Limited"].Schema.Enum)

	paths := make([]string, 0, len(result.Fixes))
	for _, fix := range result.Fixes {
		paths = append(paths, fix.Path)
	}
	assert.Equal(t, []string{
		"parameters.LitterSize",
		"responses.Limited.schema",
		"responses.Limited.headers.X-Rate-Limit",
	}, paths)
}

// TestFix_CSVEnumExpansion_PathItemParameters tests the parameters a path item
// declares for every operation it holds, which are visited once rather than
// once per operation.
func TestFix_CSVEnumExpansion_PathItemParameters(t *testing.T) {
	tests := []struct {
		name     string
		document func() (any, parser.OASVersion, string)
		enum     func(any) []any
		fixPath  string
	}{
		{
			name:    "OAS 2.0 parameter-level enum",
			fixPath: "paths./items.parameters[0]",
			document: func() (any, parser.OASVersion, string) {
				doc := &parser.OAS2Document{
					Swagger: "2.0",
					Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
					Paths: map[string]*parser.PathItem{
						"/items": {
							Parameters: []*parser.Parameter{
								{Name: "status", In: "query", Type: "integer", Enum: []any{"1,2,3"}},
							},
							Get:  &parser.Operation{OperationID: "listItems"},
							Post: &parser.Operation{OperationID: "createItem"},
						},
					},
				}
				return doc, parser.OASVersion20, "2.0"
			},
			enum: func(doc any) []any {
				return doc.(*parser.OAS2Document).Paths["/items"].Parameters[0].Enum
			},
		},
		{
			name:    "OAS 3.x schema enum",
			fixPath: "paths./items.parameters[0].schema",
			document: func() (any, parser.OASVersion, string) {
				doc := &parser.OAS3Document{
					OpenAPI:    "3.0.3",
					OASVersion: parser.OASVersion303,
					Info:       &parser.Info{Title: "Test", Version: "1.0.0"},
					Paths: map[string]*parser.PathItem{
						"/items": {
							Parameters: []*parser.Parameter{
								{Name: "status", In: "query", Schema: &parser.Schema{Type: "integer", Enum: []any{"1,2,3"}}},
							},
							Get:  &parser.Operation{OperationID: "listItems"},
							Post: &parser.Operation{OperationID: "createItem"},
						},
					},
				}
				return doc, parser.OASVersion303, "3.0.3"
			},
			enum: func(doc any) []any {
				return doc.(*parser.OAS3Document).Paths["/items"].Parameters[0].Schema.Enum
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, oasVersion, version := tt.document()

			f := New()
			f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}
			result, err := f.FixParsed(parser.ParseResult{
				Document:   doc,
				OASVersion: oasVersion,
				Version:    version,
			})
			require.NoError(t, err)

			assert.Equal(t, []any{int64(1), int64(2), int64(3)}, tt.enum(result.Document))
			require.Len(t, result.Fixes, 1, "a path item's parameters are visited once, not once per operation")
			assert.Equal(t, tt.fixPath, result.Fixes[0].Path)
		})
	}
}

// TestFix_CSVEnumExpansion_OAS3Headers tests OAS 3.x headers, which take the
// same schema-or-content form as a parameter.
func TestFix_CSVEnumExpansion_OAS3Headers(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Headers: map[string]*parser.Header{
				"X-Rate-Limit": {Schema: &parser.Schema{Type: "integer", Enum: []any{"10,20"}}},
			},
		},
		Paths: map[string]*parser.PathItem{
			"/items": {
				Get: &parser.Operation{
					OperationID: "listItems",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
								Headers: map[string]*parser.Header{
									"X-Page-Size": {Schema: &parser.Schema{Type: "integer", Enum: []any{"25,50"}}},
								},
							},
						},
					},
				},
			},
		},
	}

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}
	result, err := f.FixParsed(parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
		Version:    "3.0.3",
	})
	require.NoError(t, err)

	fixed := result.Document.(*parser.OAS3Document)
	assert.Equal(t, []any{int64(10), int64(20)}, fixed.Components.Headers["X-Rate-Limit"].Schema.Enum)
	assert.Equal(t, []any{int64(25), int64(50)},
		fixed.Paths["/items"].Get.Responses.Codes["200"].Headers["X-Page-Size"].Schema.Enum)

	require.Len(t, result.Fixes, 2)
	assert.Equal(t, "components.headers.X-Rate-Limit.schema", result.Fixes[0].Path)
	assert.Equal(t, "paths./items.get.responses.200.headers.X-Page-Size.schema", result.Fixes[1].Path)
}

// TestFix_CSVEnumExpansion_OAS3ParameterContent tests a parameter that
// describes its values through content rather than schema.
func TestFix_CSVEnumExpansion_OAS3ParameterContent(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Parameters: map[string]*parser.Parameter{
				"Status": {
					Name: "status",
					In:   "query",
					Content: map[string]*parser.MediaType{
						"application/json": {Schema: &parser.Schema{Type: "integer", Enum: []any{"1,2,3"}}},
					},
				},
			},
		},
	}

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}
	result, err := f.FixParsed(parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
		Version:    "3.0.3",
	})
	require.NoError(t, err)

	fixed := result.Document.(*parser.OAS3Document)
	schema := fixed.Components.Parameters["Status"].Content["application/json"].Schema
	assert.Equal(t, []any{int64(1), int64(2), int64(3)}, schema.Enum)
	require.Len(t, result.Fixes, 1)
	assert.Equal(t, "components.parameters.Status.content.application/json.schema", result.Fixes[0].Path)
}

// TestFix_CSVEnumExpansion_OAS3Callbacks tests the path items a callback holds,
// which carry operations as capable of declaring an enum as the document's own
// paths are.
func TestFix_CSVEnumExpansion_OAS3Callbacks(t *testing.T) {
	callback := parser.Callback{
		"{$request.body#/url}": {
			Post: &parser.Operation{
				OperationID: "onEvent",
				Parameters: []*parser.Parameter{
					{Name: "attempt", In: "query", Schema: &parser.Schema{Type: "integer", Enum: []any{"1,2,3"}}},
				},
			},
		},
	}
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]*parser.PathItem{
			"/subscribe": {
				Post: &parser.Operation{
					OperationID: "subscribe",
					Callbacks:   map[string]*parser.Callback{"onEvent": &callback},
				},
			},
		},
	}

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}
	result, err := f.FixParsed(parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
		Version:    "3.0.3",
	})
	require.NoError(t, err)

	fixed := result.Document.(*parser.OAS3Document)
	fixedCallback := fixed.Paths["/subscribe"].Post.Callbacks["onEvent"]
	param := (*fixedCallback)["{$request.body#/url}"].Post.Parameters[0]
	assert.Equal(t, []any{int64(1), int64(2), int64(3)}, param.Schema.Enum)

	require.Len(t, result.Fixes, 1)
	assert.Equal(t,
		"paths./subscribe.post.callbacks.onEvent.{$request.body#/url}.post.parameters[0].schema",
		result.Fixes[0].Path)
}

// TestFix_CSVEnumExpansion_OAS3EncodingHeaders tests the headers an encoding
// declares, including the encodings OAS 3.2 lets one nest.
func TestFix_CSVEnumExpansion_OAS3EncodingHeaders(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI:    "3.2.0",
		OASVersion: parser.OASVersion320,
		Info:       &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]*parser.PathItem{
			"/upload": {
				Post: &parser.Operation{
					OperationID: "upload",
					RequestBody: &parser.RequestBody{
						Content: map[string]*parser.MediaType{
							"multipart/form-data": {
								Encoding: map[string]*parser.Encoding{
									"profile": {
										Headers: map[string]*parser.Header{
											"X-Part-Size": {Schema: &parser.Schema{Type: "integer", Enum: []any{"1,2"}}},
										},
										ItemEncoding: &parser.Encoding{
											Headers: map[string]*parser.Header{
												"X-Item-Size": {Schema: &parser.Schema{Type: "integer", Enum: []any{"3,4"}}},
											},
										},
										PrefixEncoding: []*parser.Encoding{{
											Headers: map[string]*parser.Header{
												"X-Lead-Size": {Schema: &parser.Schema{Type: "integer", Enum: []any{"5,6"}}},
											},
										}},
									},
								},
								ItemSchema: &parser.Schema{Type: "integer", Enum: []any{"7,8"}},
							},
						},
					},
				},
			},
		},
	}

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}
	result, err := f.FixParsed(parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion320,
		Version:    "3.2.0",
	})
	require.NoError(t, err)

	fixed := result.Document.(*parser.OAS3Document)
	encoding := fixed.Paths["/upload"].Post.RequestBody.Content["multipart/form-data"].Encoding["profile"]
	assert.Equal(t, []any{int64(1), int64(2)}, encoding.Headers["X-Part-Size"].Schema.Enum)
	assert.Equal(t, []any{int64(3), int64(4)}, encoding.ItemEncoding.Headers["X-Item-Size"].Schema.Enum)
	assert.Equal(t, []any{int64(5), int64(6)}, encoding.PrefixEncoding[0].Headers["X-Lead-Size"].Schema.Enum)

	mediaType := fixed.Paths["/upload"].Post.RequestBody.Content["multipart/form-data"]
	assert.Equal(t, []any{int64(7), int64(8)}, mediaType.ItemSchema.Enum)

	paths := make([]string, 0, len(result.Fixes))
	for _, fix := range result.Fixes {
		paths = append(paths, fix.Path)
	}
	const content = "paths./upload.post.requestBody.content.multipart/form-data"
	assert.Equal(t, []string{
		content + ".itemSchema",
		content + ".encoding.profile.headers.X-Part-Size.schema",
		content + ".encoding.profile.itemEncoding.headers.X-Item-Size.schema",
		content + ".encoding.profile.prefixEncoding[0].headers.X-Lead-Size.schema",
	}, paths)
}

// TestFix_CSVEnumExpansion_OAS3Webhooks tests the path items OAS 3.1 places in
// webhooks, a sibling of paths rather than an entry under it.
func TestFix_CSVEnumExpansion_OAS3Webhooks(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Info:       &parser.Info{Title: "Test", Version: "1.0.0"},
		Webhooks: map[string]*parser.PathItem{
			"petCreated": {
				Post: &parser.Operation{
					OperationID: "petCreated",
					Parameters: []*parser.Parameter{
						{Name: "attempt", In: "query", Schema: &parser.Schema{Type: "integer", Enum: []any{"1,2,3"}}},
					},
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

	fixed := result.Document.(*parser.OAS3Document)
	param := fixed.Webhooks["petCreated"].Post.Parameters[0]
	assert.Equal(t, []any{int64(1), int64(2), int64(3)}, param.Schema.Enum)

	require.Len(t, result.Fixes, 1)
	assert.Equal(t, "webhooks.petCreated.post.parameters[0].schema", result.Fixes[0].Path)
}

// TestFix_CSVEnumExpansion_OAS3ComponentsPathItems tests the path items OAS 3.1
// lets components hold, which no path reaches except through a $ref.
func TestFix_CSVEnumExpansion_OAS3ComponentsPathItems(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Info:       &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			PathItems: map[string]*parser.PathItem{
				"Shared": {
					Get: &parser.Operation{
						OperationID: "shared",
						Parameters: []*parser.Parameter{
							{Name: "status", In: "query", Schema: &parser.Schema{Type: "integer", Enum: []any{"1,2"}}},
						},
					},
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

	fixed := result.Document.(*parser.OAS3Document)
	param := fixed.Components.PathItems["Shared"].Get.Parameters[0]
	assert.Equal(t, []any{int64(1), int64(2)}, param.Schema.Enum)

	require.Len(t, result.Fixes, 1)
	assert.Equal(t, "components.pathItems.Shared.get.parameters[0].schema", result.Fixes[0].Path)
}

// TestFix_CSVEnumExpansion_SortedProperties tests that the fixes a schema's
// properties produce are recorded in a stable order, which ranging over the
// map would leave to chance.
func TestFix_CSVEnumExpansion_SortedProperties(t *testing.T) {
	names := []string{"alpha", "bravo", "charlie", "delta", "echo"}

	for range 8 {
		properties := make(map[string]*parser.Schema, len(names))
		for _, name := range names {
			properties[name] = &parser.Schema{Type: "integer", Enum: []any{"1,2"}}
		}
		doc := &parser.OAS2Document{
			Swagger:     "2.0",
			Info:        &parser.Info{Title: "Test", Version: "1.0.0"},
			Definitions: map[string]*parser.Schema{"Pet": {Type: "object", Properties: properties}},
		}

		f := New()
		f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}
		result, err := f.FixParsed(parser.ParseResult{
			Document:   doc,
			OASVersion: parser.OASVersion20,
			Version:    "2.0",
		})
		require.NoError(t, err)

		paths := make([]string, 0, len(result.Fixes))
		for _, fix := range result.Fixes {
			paths = append(paths, fix.Path)
		}
		assert.Equal(t, []string{
			"definitions.Pet.properties.alpha",
			"definitions.Pet.properties.bravo",
			"definitions.Pet.properties.charlie",
			"definitions.Pet.properties.delta",
			"definitions.Pet.properties.echo",
		}, paths)
	}
}

// TestFix_CSVEnumExpansion_OAS2StringTypeUnchanged tests that the restriction
// to numeric types holds at the declarations a parameter and its items make
// directly. A comma inside a string enum value is legitimate, so it is left as
// the document wrote it.
func TestFix_CSVEnumExpansion_OAS2StringTypeUnchanged(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]*parser.PathItem{
			"/items": {
				Get: &parser.Operation{
					OperationID: "listItems",
					Parameters: []*parser.Parameter{
						{Name: "city", In: "query", Type: "string", Enum: []any{"Austin,TX"}},
						{
							Name:  "cities",
							In:    "query",
							Type:  "array",
							Items: &parser.Items{Type: "string", Enum: []any{"Austin,TX"}},
						},
					},
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
								Headers: map[string]*parser.Header{
									"X-Region": {Type: "string", Enum: []any{"us,east"}},
								},
							},
						},
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

	fixed := result.Document.(*parser.OAS2Document)
	op := fixed.Paths["/items"].Get
	assert.Equal(t, []any{"Austin,TX"}, op.Parameters[0].Enum)
	assert.Equal(t, []any{"Austin,TX"}, op.Parameters[1].Items.Enum)
	assert.Equal(t, []any{"us,east"}, op.Responses.Codes["200"].Headers["X-Region"].Enum)
	assert.Empty(t, result.Fixes)
}

// TestFix_CSVEnumExpansion_OAS32AdditionalOperations tests that a custom OAS
// 3.2 method is reported under additionalOperations, the segment the document
// spells it in, and that the version comes from the parse result when the
// document does not carry one.
func TestFix_CSVEnumExpansion_OAS32AdditionalOperations(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]*parser.PathItem{
			"/items": {
				AdditionalOperations: map[string]*parser.Operation{
					"PURGE": {
						OperationID: "purgeItems",
						Parameters: []*parser.Parameter{
							{Name: "status", In: "query", Schema: &parser.Schema{Type: "integer", Enum: []any{"1,2"}}},
						},
					},
				},
			},
		},
	}

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}
	result, err := f.FixParsed(parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion320,
		Version:    "3.2.0",
	})
	require.NoError(t, err)

	fixed := result.Document.(*parser.OAS3Document)
	param := fixed.Paths["/items"].AdditionalOperations["PURGE"].Parameters[0]
	assert.Equal(t, []any{int64(1), int64(2)}, param.Schema.Enum)

	require.Len(t, result.Fixes, 1)
	assert.Equal(t, "paths./items.additionalOperations.PURGE.parameters[0].schema", result.Fixes[0].Path)
}

// TestFix_CSVEnumExpansion_OAS31TypeArrayOfStrings tests a type spelled as
// []string rather than []any, which the shared accessor understands and the
// local helper it replaced did not.
func TestFix_CSVEnumExpansion_OAS31TypeArrayOfStrings(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Info:       &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Status": {Type: []string{"null", "integer"}, Enum: []any{"1,2,3"}},
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

	fixed := result.Document.(*parser.OAS3Document)
	assert.Equal(t, []any{int64(1), int64(2), int64(3)}, fixed.Components.Schemas["Status"].Enum)
	require.Len(t, result.Fixes, 1)
	assert.Equal(t, "components.schemas.Status", result.Fixes[0].Path)
}

// TestFix_CSVEnumExpansion_DryRunDoesNotMutate tests that DryRun suppresses the
// write while still reporting the fix. Skipping the pass would make the preview
// under-report, which is worse than mutating, because nothing signals it.
func TestFix_CSVEnumExpansion_DryRunDoesNotMutate(t *testing.T) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: map[string]*parser.PathItem{
			"/items": {
				Get: &parser.Operation{
					OperationID: "listItems",
					Parameters: []*parser.Parameter{
						{Name: "status", In: "query", Type: "integer", Enum: []any{"1,2,3"}},
					},
				},
			},
		},
	}

	f := New()
	f.EnabledFixes = []FixType{FixTypeEnumCSVExpanded}
	f.DryRun = true
	f.MutableInput = true // so the source document is the one the pass would write to

	result, err := f.FixParsed(parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion20,
		Version:    "2.0",
	})
	require.NoError(t, err)

	require.Len(t, result.Fixes, 1, "the fix is still reported")
	assert.Equal(t, "paths./items.get.parameters[0]", result.Fixes[0].Path)
	assert.Equal(t, []any{int64(1), int64(2), int64(3)}, result.Fixes[0].After)

	assert.Equal(t, []any{"1,2,3"}, doc.Paths["/items"].Get.Parameters[0].Enum,
		"the document is left as it was")
}

// TestFix_CSVEnumExpansion_SchemaKeywords tests the JSON Schema keywords a
// schema may nest another schema under. Reaching only properties, items,
// additionalProperties and the composition keywords left an enum unexpanded in
// eleven other places.
func TestFix_CSVEnumExpansion_SchemaKeywords(t *testing.T) {
	csv := func() *parser.Schema {
		return &parser.Schema{Type: "integer", Enum: []any{"1,2"}}
	}
	doc := &parser.OAS3Document{
		OpenAPI:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Info:       &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Wide": {
					Type:                  "object",
					PrefixItems:           []*parser.Schema{csv()},
					Contains:              csv(),
					PropertyNames:         csv(),
					PatternProperties:     map[string]*parser.Schema{"^x-": csv()},
					DependentSchemas:      map[string]*parser.Schema{"foo": csv()},
					If:                    csv(),
					Then:                  csv(),
					Else:                  csv(),
					AdditionalItems:       csv(),
					UnevaluatedItems:      csv(),
					UnevaluatedProperties: csv(),
					ContentSchema:         csv(),
					Defs:                  map[string]*parser.Schema{"Inner": csv()},
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

	paths := make([]string, 0, len(result.Fixes))
	for _, fix := range result.Fixes {
		paths = append(paths, fix.Path)
	}
	const base = "components.schemas.Wide"
	assert.ElementsMatch(t, []string{
		base + ".prefixItems[0]",
		base + ".contains",
		base + ".propertyNames",
		base + ".patternProperties.^x-",
		base + ".dependentSchemas.foo",
		base + ".if",
		base + ".then",
		base + ".else",
		base + ".additionalItems",
		base + ".unevaluatedItems",
		base + ".unevaluatedProperties",
		base + ".contentSchema",
		base + ".$defs.Inner",
	}, paths)
}
