package walker_test

import (
	"fmt"

	"github.com/erraggy/oastools/parser"
	"github.com/erraggy/oastools/walker"
)

func ExampleWalk() {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info: &parser.Info{
			Title:   "Pet Store API",
			Version: "1.0.0",
		},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Summary:     "List all pets",
				},
				Post: &parser.Operation{
					OperationID: "createPet",
					Summary:     "Create a pet",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var operationIDs []string
	_ = walker.Walk(result,
		walker.WithOperationHandler(func(method string, op *parser.Operation, path string) walker.Action {
			operationIDs = append(operationIDs, op.OperationID)
			return walker.Continue
		}),
	)

	for _, id := range operationIDs {
		fmt.Println(id)
	}
	// Output:
	// listPets
	// createPet
}

func ExampleWalk_collectSchemas() {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name": {Type: "string"},
						"age":  {Type: "integer"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	typeCounts := make(map[string]int)
	_ = walker.Walk(result,
		walker.WithSchemaHandler(func(schema *parser.Schema, path string) walker.Action {
			if schemaType, ok := schema.Type.(string); ok && schemaType != "" {
				typeCounts[schemaType]++
			}
			return walker.Continue
		}),
	)

	fmt.Println("object:", typeCounts["object"])
	fmt.Println("string:", typeCounts["string"])
	fmt.Println("integer:", typeCounts["integer"])
	// Output:
	// object: 1
	// string: 1
	// integer: 1
}

func ExampleWalk_mutation() {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
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

	_ = walker.Walk(result,
		walker.WithSchemaHandler(func(schema *parser.Schema, path string) walker.Action {
			if schema.Extra == nil {
				schema.Extra = make(map[string]any)
			}
			schema.Extra["x-visited"] = true
			return walker.Continue
		}),
	)

	fmt.Printf("x-visited: %v\n", doc.Components.Schemas["Pet"].Extra["x-visited"])
	// Output:
	// x-visited: true
}

func ExampleWalk_skipChildren() {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/internal": &parser.PathItem{
				Get: &parser.Operation{OperationID: "internalOp"},
			},
			"/public": &parser.PathItem{
				Get: &parser.Operation{OperationID: "publicOp"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var visitedOps []string
	_ = walker.Walk(result,
		walker.WithPathHandler(func(pathTemplate string, pathItem *parser.PathItem, path string) walker.Action {
			if pathTemplate == "/internal" {
				return walker.SkipChildren
			}
			return walker.Continue
		}),
		walker.WithOperationHandler(func(method string, op *parser.Operation, path string) walker.Action {
			visitedOps = append(visitedOps, op.OperationID)
			return walker.Continue
		}),
	)

	fmt.Println("Visited:", visitedOps)
	// Output:
	// Visited: [publicOp]
}

func ExampleWalk_stop() {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/a": &parser.PathItem{Get: &parser.Operation{OperationID: "opA"}},
			"/b": &parser.PathItem{Get: &parser.Operation{OperationID: "opB"}},
			"/c": &parser.PathItem{Get: &parser.Operation{OperationID: "opC"}},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var firstOp string
	_ = walker.Walk(result,
		walker.WithOperationHandler(func(method string, op *parser.Operation, path string) walker.Action {
			firstOp = op.OperationID
			return walker.Stop
		}),
	)

	fmt.Println("First operation:", firstOp)
	// Output:
	// First operation: opA
}

func ExampleWalk_jsonPaths() {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{OperationID: "getPets"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	_ = walker.Walk(result,
		walker.WithOperationHandler(func(method string, op *parser.Operation, path string) walker.Action {
			fmt.Printf("Operation %s at: %s\n", op.OperationID, path)
			return walker.Continue
		}),
	)
	// Output:
	// Operation getPets at: $.paths['/pets'].get
}

func ExampleWalkWithOptions() {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
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

	var schemaCount int
	_ = walker.WalkWithOptions(
		walker.WithParsed(result),
		walker.OnSchema(func(schema *parser.Schema, path string) walker.Action {
			schemaCount++
			return walker.Continue
		}),
	)

	fmt.Println("Schema count:", schemaCount)
	// Output:
	// Schema count: 1
}

func ExampleWalk_documentTypeSwitch() {
	oas3Doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "OAS 3.x API", Version: "1.0.0"},
	}

	result := &parser.ParseResult{
		Document:   oas3Doc,
		OASVersion: parser.OASVersion303,
	}

	_ = walker.Walk(result,
		walker.WithDocumentHandler(func(doc any, path string) walker.Action {
			switch d := doc.(type) {
			case *parser.OAS2Document:
				fmt.Println("OAS 2.0:", d.Info.Title)
			case *parser.OAS3Document:
				fmt.Println("OAS 3.x:", d.Info.Title)
			}
			return walker.Continue
		}),
	)
	// Output:
	// OAS 3.x: OAS 3.x API
}
