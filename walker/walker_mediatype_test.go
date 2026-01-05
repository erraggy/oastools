// walker_mediatype_test.go - Tests for media type handler traversal
// Tests media type examples and flow control actions.

package walker

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalk_MediaTypeExamples(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
								Content: map[string]*parser.MediaType{
									"application/json": {
										Examples: map[string]*parser.Example{
											"cat":  {Summary: "A cat"},
											"dog":  {Summary: "A dog"},
											"bird": {Summary: "A bird"},
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

	// Should visit all 3 examples in the media type
	assert.Len(t, exampleNames, 3)
	assert.Contains(t, exampleNames, "cat")
	assert.Contains(t, exampleNames, "dog")
	assert.Contains(t, exampleNames, "bird")
}

func TestWalk_MediaTypeSkipChildren(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Paths: parser.Paths{
			"/pets": &parser.PathItem{
				Get: &parser.Operation{
					Responses: &parser.Responses{
						Codes: map[string]*parser.Response{
							"200": {
								Description: "OK",
								Content: map[string]*parser.MediaType{
									"application/json": {
										Schema: &parser.Schema{Type: "object"},
										Examples: map[string]*parser.Example{
											"example1": {Summary: "Test"},
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

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	schemaVisited := false
	exampleVisited := false
	err := Walk(result,
		WithMediaTypeHandler(func(wc *WalkContext, mt *parser.MediaType) Action {
			return SkipChildren
		}),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			if strings.Contains(wc.JSONPath, "content") {
				schemaVisited = true
			}
			return Continue
		}),
		WithExampleHandler(func(wc *WalkContext, example *parser.Example) Action {
			if strings.Contains(wc.JSONPath, "content") {
				exampleVisited = true
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.False(t, schemaVisited, "schema should not be visited when mediaType returns SkipChildren")
	assert.False(t, exampleVisited, "example should not be visited when mediaType returns SkipChildren")
}
