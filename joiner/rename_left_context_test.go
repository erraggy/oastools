package joiner

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// operationContextDoc gives its Response schema one operation, whose operationId
// is what an operation-aware rename template reads.
func operationContextDoc(name, operationID string, extra bool) parser.ParseResult {
	response := &parser.Schema{Type: "object", Properties: map[string]*parser.Schema{"id": {Type: "string"}}}
	if extra {
		response.Properties["note"] = &parser.Schema{Type: "string"}
	}
	return parser.ParseResult{
		Document: &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info:    &parser.Info{Title: name, Version: "1.0.0"},
			Paths: parser.Paths{
				"/" + name: &parser.PathItem{Get: &parser.Operation{
					OperationID: operationID,
					Responses: &parser.Responses{Codes: map[string]*parser.Response{
						"200": {Description: "ok", Content: map[string]*parser.MediaType{
							"application/json": {Schema: &parser.Schema{Ref: "#/components/schemas/Response"}},
						}},
					}},
				}},
			},
			Components: &parser.Components{Schemas: map[string]*parser.Schema{"Response": response}},
			OASVersion: parser.OASVersion303,
		},
		Version: "3.0.3", OASVersion: parser.OASVersion303,
		SourcePath: name, SourceFormat: parser.SourceFormatJSON,
	}
}

// TestRenameLeftUsesOperationContext covers #482. The left side of a rename-left
// collision was named with a nil reference graph, so an operation-aware template
// fell back to its empty-field output while the right side got a full name.
func TestRenameLeftUsesOperationContext(t *testing.T) {
	res, err := JoinWithOptions(
		WithParsed(
			operationContextDoc("orders", "listOrders", false),
			operationContextDoc("users", "listUsers", true),
		),
		WithSchemaStrategy(StrategyRenameLeft),
		WithPathStrategy(StrategyAcceptLeft),
		WithOperationContext(true),
		WithRenameTemplate(`{{pascalCase .OperationID}}{{.Name}}`),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS3Document)

	// orders was moved aside, and its name comes from its own operation rather
	// than from an empty context.
	assert.Contains(t, d.Components.Schemas, "ListOrdersResponse")
	assert.NotContains(t, d.Components.Schemas, "Response_orders",
		"the template fell back, so the left side had no operation context")

	// users took the original name.
	assert.Contains(t, d.Components.Schemas, "Response")
	assert.Contains(t, d.Components.Schemas["Response"].Properties, "note")

	// Each document still reads its own schema.
	ref := func(path string) string {
		return d.Paths[path].Get.Responses.Codes["200"].Content["application/json"].Schema.Ref
	}
	assert.Equal(t, "#/components/schemas/ListOrdersResponse", ref("/orders"))
	assert.Equal(t, "#/components/schemas/Response", ref("/users"))
}

// TestRenameLeftOperationContextOAS2 is the OAS 2 counterpart.
func TestRenameLeftOperationContextOAS2(t *testing.T) {
	doc := func(name, operationID string, extra bool) parser.ParseResult {
		response := &parser.Schema{Type: "object", Properties: map[string]*parser.Schema{"id": {Type: "string"}}}
		if extra {
			response.Properties["note"] = &parser.Schema{Type: "string"}
		}
		return parser.ParseResult{
			Document: &parser.OAS2Document{
				Swagger: "2.0",
				Info:    &parser.Info{Title: name, Version: "1.0.0"},
				Paths: parser.Paths{
					"/" + name: &parser.PathItem{Get: &parser.Operation{
						OperationID: operationID,
						Responses: &parser.Responses{Codes: map[string]*parser.Response{
							"200": {Description: "ok", Schema: &parser.Schema{Ref: "#/definitions/Response"}},
						}},
					}},
				},
				Definitions: map[string]*parser.Schema{"Response": response},
				OASVersion:  parser.OASVersion20,
			},
			Version: "2.0", OASVersion: parser.OASVersion20,
			SourcePath: name, SourceFormat: parser.SourceFormatJSON,
		}
	}

	res, err := JoinWithOptions(
		WithParsed(doc("orders", "listOrders", false), doc("users", "listUsers", true)),
		WithSchemaStrategy(StrategyRenameLeft),
		WithPathStrategy(StrategyAcceptLeft),
		WithOperationContext(true),
		WithRenameTemplate(`{{pascalCase .OperationID}}{{.Name}}`),
	)
	require.NoError(t, err)

	d := res.Document.(*parser.OAS2Document)
	assert.Contains(t, d.Definitions, "ListOrdersResponse")
	assert.Contains(t, d.Definitions, "Response")
	assert.Equal(t, "#/definitions/ListOrdersResponse", petResponseRef(t, d, "/orders"))
	assert.Equal(t, "#/definitions/Response", petResponseRef(t, d, "/users"))
}

func TestRefGraphsCachesPerDocument(t *testing.T) {
	built := map[int]int{}
	graphs := newRefGraphs(true, func(docIndex int) *RefGraph {
		built[docIndex]++
		return &RefGraph{}
	})

	for range 3 {
		graphs.forDoc(0)
		graphs.forDoc(1)
	}
	assert.Equal(t, map[int]int{0: 1, 1: 1}, built, "each document's graph is built once")

	// Disabled and nil caches never build, which is the operation context off path.
	assert.Nil(t, newRefGraphs(false, func(int) *RefGraph { t.Fatal("built while disabled"); return nil }))
	var absent *refGraphs
	assert.Nil(t, absent.forDoc(0))
	assert.Nil(t, graphs.forDoc(-1))
}
