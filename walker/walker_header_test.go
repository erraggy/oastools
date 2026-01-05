// walker_header_test.go - Tests for header handler traversal
// Tests header content, examples, and flow control actions.

package walker

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalk_HeaderWithContent(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Headers: map[string]*parser.Header{
				"X-Custom-Header": {
					Description: "Custom header with content",
					Content: map[string]*parser.MediaType{
						"application/json": {
							Schema: &parser.Schema{Type: "object"},
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
		if strings.Contains(p, "headers") && strings.Contains(p, "content") {
			found = true
			break
		}
	}
	assert.True(t, found, "should visit header content media type")
}

func TestWalk_HeaderWithExamples(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Headers: map[string]*parser.Header{
				"X-Request-ID": {
					Description: "Request ID header",
					Schema:      &parser.Schema{Type: "string"},
					Examples: map[string]*parser.Example{
						"uuid1": {Summary: "UUID example", Value: "123e4567-e89b-12d3-a456-426614174000"},
						"uuid2": {Summary: "Another UUID", Value: "987fcdeb-51a2-43e6-b7c8-123456789abc"},
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
	assert.Contains(t, exampleNames, "uuid1")
	assert.Contains(t, exampleNames, "uuid2")
}

func TestWalk_HeaderSkipChildren(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Headers: map[string]*parser.Header{
				"X-Rate-Limit": {
					Description: "Rate limit header",
					Schema:      &parser.Schema{Type: "integer"},
					Examples: map[string]*parser.Example{
						"example1": {Summary: "Example"},
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
		WithHeaderHandler(func(wc *WalkContext, header *parser.Header) Action {
			return SkipChildren
		}),
		WithSchemaHandler(func(wc *WalkContext, schema *parser.Schema) Action {
			if strings.Contains(wc.JSONPath, "headers") {
				schemaVisited = true
			}
			return Continue
		}),
		WithExampleHandler(func(wc *WalkContext, example *parser.Example) Action {
			if strings.Contains(wc.JSONPath, "headers") {
				exampleVisited = true
			}
			return Continue
		}),
	)
	require.NoError(t, err)

	assert.False(t, schemaVisited, "schema should not be visited when header handler returns SkipChildren")
	assert.False(t, exampleVisited, "example should not be visited when header handler returns SkipChildren")
}

func TestWalk_HeaderStop(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.0.0",
		Info:    &parser.Info{Title: "Test", Version: "1.0"},
		Components: &parser.Components{
			Headers: map[string]*parser.Header{
				"A-Header": {Description: "First header"},
				"B-Header": {Description: "Second header"},
				"C-Header": {Description: "Third header"},
			},
		},
	}

	result := &parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document:   doc,
	}

	var visitedHeaders []string
	err := Walk(result,
		WithHeaderHandler(func(wc *WalkContext, header *parser.Header) Action {
			visitedHeaders = append(visitedHeaders, wc.Name)
			// Stop after first header
			return Stop
		}),
	)
	require.NoError(t, err)

	assert.Len(t, visitedHeaders, 1, "should stop after first header")
}
