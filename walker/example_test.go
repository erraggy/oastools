package walker_test

import (
	"fmt"
	"strings"

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
		walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
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
		walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
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
		walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
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
		walker.WithPathHandler(func(wc *walker.WalkContext, pathItem *parser.PathItem) walker.Action {
			if wc.PathTemplate == "/internal" {
				return walker.SkipChildren
			}
			return walker.Continue
		}),
		walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
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
		walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
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
		walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
			fmt.Printf("Operation %s at: %s\n", op.OperationID, wc.JSONPath)
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
		walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
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
		walker.WithDocumentHandler(func(wc *walker.WalkContext, doc any) walker.Action {
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

func ExampleCollectSchemas() {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Pet Store", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type:        "object",
					Description: "A pet in the store",
					Properties: map[string]*parser.Schema{
						"name": {Type: "string"},
					},
				},
				"Error": {
					Type:        "object",
					Description: "An error response",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	// Collect all schemas
	schemas, _ := walker.CollectSchemas(result)

	// Print component schema count
	fmt.Printf("Total schemas: %d\n", len(schemas.All))
	fmt.Printf("Component schemas: %d\n", len(schemas.Components))

	// Look up by name
	if pet, ok := schemas.ByName["Pet"]; ok {
		fmt.Printf("Found Pet: %s\n", pet.Schema.Description)
	}
	// Output:
	// Total schemas: 3
	// Component schemas: 3
	// Found Pet: A pet in the store
}

func ExampleCollectOperations() {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Pet Store", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Tags:        []string{"pets"},
				},
				Post: &parser.Operation{
					OperationID: "createPet",
					Tags:        []string{"pets"},
				},
			},
			"/pets/{petId}": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPet",
					Tags:        []string{"pets"},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	// Collect all operations
	ops, _ := walker.CollectOperations(result)

	// Print operation counts
	fmt.Printf("Total operations: %d\n", len(ops.All))

	// Group by tag
	for tag, tagOps := range ops.ByTag {
		fmt.Printf("Tag '%s' has %d operations\n", tag, len(tagOps))
	}
	// Output:
	// Total operations: 3
	// Tag 'pets' has 3 operations
}

func ExampleCollectParameters() {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Pet Store", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Parameters: []*parser.Parameter{
						{Name: "limit", In: "query"},
						{Name: "offset", In: "query"},
					},
				},
			},
			"/pets/{petId}": &parser.PathItem{
				Parameters: []*parser.Parameter{
					{Name: "petId", In: "path", Required: true},
				},
				Get: &parser.Operation{
					OperationID: "getPet",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	params, _ := walker.CollectParameters(result)

	fmt.Printf("Total parameters: %d\n", len(params.All))
	fmt.Printf("Query parameters: %d\n", len(params.ByLocation["query"]))
	fmt.Printf("Path parameters: %d\n", len(params.ByLocation["path"]))
	// Output:
	// Total parameters: 3
	// Query parameters: 2
	// Path parameters: 1
}

func ExampleCollectResponses() {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Pet Store", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {Description: "A list of pets"},
							"500": {Description: "Server error"},
						},
					},
				},
				Post: &parser.Operation{
					OperationID: "createPet",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"201": {Description: "Pet created"},
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

	responses, _ := walker.CollectResponses(result)

	fmt.Printf("Total responses: %d\n", len(responses.All))
	fmt.Printf("Success (200): %d\n", len(responses.ByStatusCode["200"]))
	fmt.Printf("Server errors (500): %d\n", len(responses.ByStatusCode["500"]))
	// Output:
	// Total responses: 3
	// Success (200): 1
	// Server errors (500): 1
}

func ExampleCollectSecuritySchemes() {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Pet Store", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"bearerAuth": {
					Type:   "http",
					Scheme: "bearer",
				},
				"apiKey": {
					Type: "apiKey",
					Name: "X-API-Key",
					In:   "header",
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	schemes, _ := walker.CollectSecuritySchemes(result)

	fmt.Printf("Total schemes: %d\n", len(schemes.All))

	if bearer, ok := schemes.ByName["bearerAuth"]; ok {
		fmt.Printf("bearerAuth: type=%s, scheme=%s\n", bearer.SecurityScheme.Type, bearer.SecurityScheme.Scheme)
	}
	// Output:
	// Total schemes: 2
	// bearerAuth: type=http, scheme=bearer
}

func ExampleWithParentTracking() {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Pet Store", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "Success",
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Type: "array"},
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

	_ = walker.Walk(result,
		walker.WithParentTracking(),
		walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
			// Find which operation this schema belongs to
			if op, ok := wc.ParentOperation(); ok {
				fmt.Printf("Schema in operation: %s\n", op.OperationID)
			}
			// Check ancestor depth
			fmt.Printf("Ancestor depth: %d\n", wc.Depth())
			return walker.Continue
		}),
	)
	// Output:
	// Schema in operation: listPets
	// Ancestor depth: 4
}

func ExampleWithSchemaPostHandler() {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Pet Store", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name":   {Type: "string"},
						"age":    {Type: "integer"},
						"status": {Type: "string"},
					},
				},
				"Error": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"code":    {Type: "integer"},
						"message": {Type: "string"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	// Count properties in each top-level component schema after children are processed
	// Use strings.HasPrefix to identify top-level component schemas by their JSON path
	_ = walker.Walk(result,
		walker.WithSchemaPostHandler(func(wc *walker.WalkContext, schema *parser.Schema) {
			// Only count top-level component schemas (not nested properties)
			if wc.IsComponent && wc.Name != "" && !strings.Contains(wc.JSONPath, ".properties") {
				fmt.Printf("%s has %d properties\n", wc.Name, len(schema.Properties))
			}
		}),
	)
	// Output:
	// Error has 2 properties
	// Pet has 3 properties
}

func ExampleWithRefHandler() {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Pet": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"owner": {Ref: "#/components/schemas/User"},
					},
				},
				"User": {Type: "object"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	var refs []string
	_ = walker.Walk(result,
		walker.WithRefHandler(func(wc *walker.WalkContext, ref *walker.RefInfo) walker.Action {
			refs = append(refs, ref.Ref)
			return walker.Continue
		}),
	)

	fmt.Printf("Found %d reference(s)\n", len(refs))
	for _, ref := range refs {
		fmt.Println(ref)
	}
	// Output:
	// Found 1 reference(s)
	// #/components/schemas/User
}

func ExampleWithMapRefTracking() {
	// Some polymorphic schema fields (Items, AdditionalItems, AdditionalProperties,
	// UnevaluatedItems, UnevaluatedProperties) can contain map[string]any instead of
	// *parser.Schema in certain parsing scenarios. WithMapRefTracking enables
	// detection of $ref values stored in these map structures.
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0.0"},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"Container": {
					Type: "array",
					// Simulating Items as map[string]any with a $ref (can occur in some parsing scenarios)
					Items: map[string]any{
						"$ref": "#/components/schemas/Item",
					},
				},
				"Item": {Type: "string"},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion310,
	}

	var refs []string
	_ = walker.Walk(result,
		walker.WithMapRefTracking(), // Enable detection of refs in map structures
		walker.WithRefHandler(func(wc *walker.WalkContext, ref *walker.RefInfo) walker.Action {
			refs = append(refs, ref.Ref)
			return walker.Continue
		}),
	)

	fmt.Printf("Found %d reference(s)\n", len(refs))
	for _, ref := range refs {
		fmt.Println(ref)
	}
	// Output:
	// Found 1 reference(s)
	// #/components/schemas/Item
}

func ExampleWithOAS3DocumentPostHandler() {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.3",
		Info:    &parser.Info{Title: "Pet Store", Version: "1.0.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get:  &parser.Operation{OperationID: "listPets", Tags: []string{"pets"}},
				Post: &parser.Operation{OperationID: "createPet", Tags: []string{"pets"}},
			},
			"/users": &parser.PathItem{
				Get: &parser.Operation{OperationID: "listUsers", Tags: []string{"users"}},
			},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion303,
	}

	// Collect operations during walk, then use the document post handler
	// to add a summary based on collected data. This is the "single-walk" pattern.
	var operationCount int
	tagCounts := make(map[string]int)

	_ = walker.Walk(result,
		walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
			operationCount++
			for _, tag := range op.Tags {
				tagCounts[tag]++
			}
			return walker.Continue
		}),
		walker.WithOAS3DocumentPostHandler(func(wc *walker.WalkContext, d *parser.OAS3Document) {
			// Called after ALL children have been processed
			// Perfect for aggregating collected data
			fmt.Printf("Document: %s\n", d.Info.Title)
			fmt.Printf("Total operations: %d\n", operationCount)
			fmt.Printf("Operations by tag: pets=%d, users=%d\n", tagCounts["pets"], tagCounts["users"])
		}),
	)
	// Output:
	// Document: Pet Store
	// Total operations: 3
	// Operations by tag: pets=2, users=1
}

func ExampleWithOAS2DocumentPostHandler() {
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Legacy API", Version: "1.0.0"},
		Paths: parser.Paths{
			"/items": &parser.PathItem{
				Get: &parser.Operation{OperationID: "listItems"},
				Put: &parser.Operation{OperationID: "updateItems"},
			},
		},
		Definitions: map[string]*parser.Schema{
			"Item":  {Type: "object"},
			"Error": {Type: "object"},
		},
	}

	result := &parser.ParseResult{
		Document:   doc,
		OASVersion: parser.OASVersion20,
	}

	// Collect statistics during walk, then summarize in post handler
	var schemaCount, operationCount int

	_ = walker.Walk(result,
		walker.WithSchemaHandler(func(wc *walker.WalkContext, schema *parser.Schema) walker.Action {
			schemaCount++
			return walker.Continue
		}),
		walker.WithOperationHandler(func(wc *walker.WalkContext, op *parser.Operation) walker.Action {
			operationCount++
			return walker.Continue
		}),
		walker.WithOAS2DocumentPostHandler(func(wc *walker.WalkContext, d *parser.OAS2Document) {
			// Called after ALL children have been processed
			fmt.Printf("Swagger %s: %s\n", d.Swagger, d.Info.Title)
			fmt.Printf("Schemas: %d, Operations: %d\n", schemaCount, operationCount)
		}),
	)
	// Output:
	// Swagger 2.0: Legacy API
	// Schemas: 2, Operations: 2
}
