// walker_parameter_test.go - Tests for parameter handler traversal
// Tests parameter schemas, content, examples, and flow control actions.

package walker

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalk_ParameterWithSchema(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets/{id}": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPet",
					Parameters: []*parser.Parameter{
						{
							Name:   "id",
							In:     "path",
							Schema: &parser.Schema{Type: "integer"},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var schemaPaths []string
	err := Walk(result,
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			schemaPaths = append(schemaPaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range schemaPaths {
		if strings.Contains(p, "parameters[0].schema") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit parameter schema")
}

func TestWalk_ParameterWithContent(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Parameters: []*parser.Parameter{
						{
							Name: "filter",
							In:   "query",
							Content: map[string]*parser.MediaType{
								"application/json": {
									Schema: &parser.Schema{Type: "object"},
								},
							},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var mediaTypePaths []string
	err := Walk(result,
		WithMediaTypeHandler(func(wc *WalkContext, mt *parser.MediaType) Action {
			mediaTypePaths = append(mediaTypePaths, wc.JSONPath)
			return Continue
		}),
	)
	require.NoError(t, err)

	found := false
	for _, p := range mediaTypePaths {
		if strings.Contains(p, "parameters[0].content") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit parameter content media type")
}

func TestWalk_ParameterWithExamples(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets/{id}": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPet",
					Parameters: []*parser.Parameter{
						{
							Name: "id",
							In:   "path",
							Examples: map[string]*parser.Example{
								"petId1": {Summary: "First pet", Value: 1},
								"petId2": {Summary: "Second pet", Value: 2},
							},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var exampleNames []string
	err := Walk(result,
		WithExampleHandler(func(wc *WalkContext, example *parser.Example) Action {
			exampleNames = append(exampleNames, wc.Name)
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, exampleNames, 2)
	assert.Contains(t, exampleNames, "petId1")
	assert.Contains(t, exampleNames, "petId2")
}

func TestWalk_ParameterSkipChildren(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets/{id}": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "getPet",
					Parameters: []*parser.Parameter{
						{
							Name:   "id",
							In:     "path",
							Schema: &parser.Schema{Type: "integer"},
							Examples: map[string]*parser.Example{
								"example1": {Summary: "Example"},
							},
						},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	schemaVisited := false
	exampleVisited := false
	err := Walk(result,
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			return SkipChildren
		}),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			if strings.Contains(wc.JSONPath, "parameters") {
				schemaVisited = true
			}
			return Continue
		}),
		WithExampleHandler(func(wc *WalkContext, example *parser.Example) Action {
			if strings.Contains(wc.JSONPath, "parameters") {
				exampleVisited = true
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.False(t, schemaVisited, "schema should not be visited when parameter handler returns SkipChildren")
	assert.False(t, exampleVisited, "example should not be visited when parameter handler returns SkipChildren")
}

func TestWalk_ParameterStop(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Parameters: []*parser.Parameter{
						{Name: "limit", In: "query"},
						{Name: "offset", In: "query"},
						{Name: "filter", In: "query"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedParams []string
	err := Walk(result,
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			visitedParams = append(visitedParams, param.Name)
			if param.Name == "limit" {
				return Stop
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedParams, 1, "should stop after first parameter")
	assert.Equal(t, "limit", visitedParams[0])
}

func TestWalk_NilParameterInSlice(t *testing.T) {
	// Test that nil parameters in a slice are handled gracefully
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					OperationID: "listPets",
					Parameters: []*parser.Parameter{
						{Name: "valid", In: "query"},
						nil, // nil parameter
						{Name: "another", In: "query"},
					},
				},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedParams []string
	err := Walk(result,
		WithParameterHandler(func(wc *WalkContext, param *parser.Parameter) Action {
			visitedParams = append(visitedParams, param.Name)
			return Continue
		}),
	)
	require.NoError(t, err)

	// Should visit only non-nil parameters
	assert.Len(t, visitedParams, 2)
	assert.Contains(t, visitedParams, "valid")
	assert.Contains(t, visitedParams, "another")
}
