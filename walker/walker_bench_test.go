package walker

import (
	"testing"

	"github.com/erraggy/oastools/parser"
)

func BenchmarkWalkSmallDocument(b *testing.B) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get:  &parser.Operation{OperationID: "listPets"},
				Post: &parser.Operation{OperationID: "createPet"},
			},
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet":   {Type: "object"},
				"Error": {Type: "object"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	for b.Loop() {
		_ = Walk(result,
			WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
				return Continue
			}),
			WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
				return Continue
			}),
		)
	}
}

func BenchmarkWalkMediumDocument(b *testing.B) {
	// Build a medium-sized document with 50 paths and schemas
	paths := make(parser.Paths)
	schemas := make(map[string]*parser.Schema)

	for i := range 50 {
		pathName := "/resource" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		paths[pathName] = &parser.PathItem{
			Get: &parser.Operation{
				OperationID: "get" + pathName,
				Responses: &parser.Responses{
					Codes: map[string]*parser.Response{
						"200": {Description: "OK"},
					},
				},
			},
		}

		schemaName := "Schema" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		schemas[schemaName] = &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"id":   {Type: "integer"},
				"name": {Type: "string"},
			},
		}
	}

	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		Info:       &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths:      paths,
		Components: &parser.Components{Schemas: schemas},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	for b.Loop() {
		_ = Walk(result,
			WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
				return Continue
			}),
			WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
				return Continue
			}),
		)
	}
}

func BenchmarkWalkNoHandlers(b *testing.B) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{OperationID: "listPets"},
			},
		},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {Type: "object"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	for b.Loop() {
		_ = Walk(result)
	}
}

func BenchmarkWalkAllHandlers(b *testing.B) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Servers: []*parser.Server{{URL: "https://api.example.com"}},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Parameters:  []*parser.Parameter{{Name: "limit", In: "query"}},
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "OK"},
						},
					},
				},
			},
		},
		Components: &parser.Components{
			Schemas:         map[string]*parser.Schema{"Pet": {Type: "object"}},
			SecuritySchemes: map[string]*parser.SecurityScheme{"api_key": {Type: "apiKey"}},
		},
		Tags: []*parser.Tag{{Name: "pets"}},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	for b.Loop() {
		_ = Walk(result,
			WithDocumentHandler(func(wc *WalkContext, doc any) Action { return Continue }),
			WithInfoHandler(func(wc *WalkContext, info *parser.Info) Action { return Continue }),
			WithServerHandler(func(wc *WalkContext, server *parser.Server) Action { return Continue }),
			WithTagHandler(func(wc *WalkContext, tag *parser.Tag) Action { return Continue }),
			WithPathHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action { return Continue }),
			WithPathItemHandler(func(wc *WalkContext, pathItem *parser.PathItem) Action { return Continue }),
			WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action { return Continue }),
			WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action { return Continue }),
			WithResponseHandler(func(wc *WalkContext, resp *parser.Response) Action { return Continue }),
			WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action { return Continue }),
			WithSecuritySchemeHandler(func(wc *WalkContext, scheme *parser.SecurityScheme) Action { return Continue }),
		)
	}
}

func BenchmarkWalkSchemaOnly(b *testing.B) {
	// Deep schema nesting
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Root": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"level1": {
							Type: "object",
							Properties: map[string]*parser.Schema{
								"level2": {
									Type: "object",
									Properties: map[string]*parser.Schema{
										"level3": {Type: "string"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	for b.Loop() {
		_ = Walk(result,
			WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
				return Continue
			}),
		)
	}
}

func BenchmarkWalkWithStop(b *testing.B) {
	// Large document but stop early
	paths := make(parser.Paths)
	for i := range 100 {
		pathName := "/resource" + string(rune('0'+i/10)) + string(rune('0'+i%10))
		paths[pathName] = &parser.PathItem{
			Get: &parser.Operation{OperationID: "get" + pathName},
		}
	}

	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths:   paths,
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	for b.Loop() {
		_ = Walk(result,
			WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
				return Stop // Stop immediately
			}),
		)
	}
}

func BenchmarkWalkOAS2(b *testing.B) {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "OK"},
						},
					},
				},
			},
		},
		Definitions: map[string]*parser.Schema{
			"Pet":   {Type: "object"},
			"Error": {Type: "object"},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion20,
	}

	for b.Loop() {
		_ = Walk(result,
			WithOperationHandler(func(wc *WalkContext, op *parser.Operation) Action {
				return Continue
			}),
			WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
				return Continue
			}),
		)
	}
}
