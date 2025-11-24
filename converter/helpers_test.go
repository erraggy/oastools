package converter

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConvertOAS2RequestBody tests the convertOAS2RequestBody method.
func TestConvertOAS2RequestBody(t *testing.T) {
	tests := []struct {
		name      string
		operation *parser.Operation
		document  *parser.OAS2Document
		hasBody   bool
	}{
		{
			name: "operation with body parameter",
			operation: &parser.Operation{
				Parameters: []*parser.Parameter{
					{
						Name:        "body",
						In:          "body",
						Description: "User object",
						Required:    true,
						Schema:      &parser.Schema{Type: "object"},
					},
				},
			},
			document: &parser.OAS2Document{
				Consumes: []string{"application/json"},
			},
			hasBody: true,
		},
		{
			name: "operation without body parameter",
			operation: &parser.Operation{
				Parameters: []*parser.Parameter{
					{
						Name: "id",
						In:   "path",
					},
				},
			},
			document: &parser.OAS2Document{},
			hasBody:  false,
		},
		{
			name: "operation with body and operation-level consumes",
			operation: &parser.Operation{
				Consumes: []string{"application/xml"},
				Parameters: []*parser.Parameter{
					{
						Name:   "body",
						In:     "body",
						Schema: &parser.Schema{Type: "object"},
					},
				},
			},
			document: &parser.OAS2Document{
				Consumes: []string{"application/json"},
			},
			hasBody: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			result := c.convertOAS2RequestBody(tt.operation, tt.document)

			if tt.hasBody {
				require.NotNil(t, result, "convertOAS2RequestBody should return non-nil for operation with body parameter")
				assert.NotNil(t, result.Content, "RequestBody should have content")
				assert.NotEmpty(t, result.Content, "RequestBody content should not be empty")
			} else {
				assert.Nil(t, result, "convertOAS2RequestBody should return nil for operation without body parameter")
			}
		})
	}
}

// TestGetConsumes tests the getConsumes method.
func TestGetConsumes(t *testing.T) {
	tests := []struct {
		name      string
		operation *parser.Operation
		document  *parser.OAS2Document
		expected  []string
	}{
		{
			name: "operation-level consumes",
			operation: &parser.Operation{
				Consumes: []string{"application/xml", "application/json"},
			},
			document: &parser.OAS2Document{
				Consumes: []string{"text/plain"},
			},
			expected: []string{"application/xml", "application/json"},
		},
		{
			name:      "document-level consumes",
			operation: &parser.Operation{},
			document: &parser.OAS2Document{
				Consumes: []string{"application/json"},
			},
			expected: []string{"application/json"},
		},
		{
			name:      "no consumes defined",
			operation: &parser.Operation{},
			document:  &parser.OAS2Document{},
			expected:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			result := c.getConsumes(tt.operation, tt.document)
			assert.Equal(t, tt.expected, result, "getConsumes should return expected consumes array")
		})
	}
}

// TestConvertOAS3ParameterToOAS2 tests the convertOAS3ParameterToOAS2 method.
func TestConvertOAS3ParameterToOAS2(t *testing.T) {
	tests := []struct {
		name      string
		param     *parser.Parameter
		expectNil bool
	}{
		{
			name: "simple query parameter",
			param: &parser.Parameter{
				Name:   "limit",
				In:     "query",
				Schema: &parser.Schema{Type: "integer"},
			},
			expectNil: false,
		},
		{
			name: "path parameter",
			param: &parser.Parameter{
				Name:     "id",
				In:       "path",
				Required: true,
				Schema:   &parser.Schema{Type: "string"},
			},
			expectNil: false,
		},
		{
			name: "header parameter",
			param: &parser.Parameter{
				Name:   "X-API-Key",
				In:     "header",
				Schema: &parser.Schema{Type: "string"},
			},
			expectNil: false,
		},
		{
			name: "cookie parameter (OAS 3.x only)",
			param: &parser.Parameter{
				Name:   "session",
				In:     "cookie",
				Schema: &parser.Schema{Type: "string"},
			},
			expectNil: true, // Should be nil - not supported in OAS 2.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			result := &ConversionResult{Issues: []ConversionIssue{}}
			converted := c.convertOAS3ParameterToOAS2(tt.param, result, "test.path")

			if tt.expectNil {
				assert.Nil(t, converted, "convertOAS3ParameterToOAS2 should return nil")
			} else {
				require.NotNil(t, converted, "convertOAS3ParameterToOAS2 should return non-nil")
				assert.Equal(t, tt.param.Name, converted.Name, "Name should be preserved")
			}
		})
	}
}

// TestConvertOAS3RequestBodyToOAS2 tests the convertOAS3RequestBodyToOAS2 method.
func TestConvertOAS3RequestBodyToOAS2(t *testing.T) {
	tests := []struct {
		name        string
		requestBody *parser.RequestBody
		expectParam bool
	}{
		{
			name: "request body with JSON content",
			requestBody: &parser.RequestBody{
				Description: "User object",
				Required:    true,
				Content: map[string]*parser.MediaType{
					"application/json": {
						Schema: &parser.Schema{Type: "object"},
					},
				},
			},
			expectParam: true,
		},
		{
			name: "request body with multiple media types",
			requestBody: &parser.RequestBody{
				Description: "Pet object",
				Content: map[string]*parser.MediaType{
					"application/json": {
						Schema: &parser.Schema{Type: "object"},
					},
					"application/xml": {
						Schema: &parser.Schema{Type: "object"},
					},
				},
			},
			expectParam: true,
		},
		{
			name:        "nil request body",
			requestBody: nil,
			expectParam: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			result := &ConversionResult{Issues: []ConversionIssue{}}
			converted, mediaTypes := c.convertOAS3RequestBodyToOAS2(tt.requestBody, result, "test.path")

			if tt.expectParam {
				require.NotNil(t, converted, "convertOAS3RequestBodyToOAS2 should return non-nil parameter")
				assert.Equal(t, "body", converted.In, "Converted parameter should be in body")
				assert.NotNil(t, mediaTypes, "Media types should be returned")
			} else {
				assert.Nil(t, converted, "convertOAS3RequestBodyToOAS2 should return nil for nil request body")
			}
		})
	}
}

// TestRewriteParameterRefsOAS2ToOAS3 tests the rewriteParameterRefsOAS2ToOAS3 function.
func TestRewriteParameterRefsOAS2ToOAS3(t *testing.T) {
	tests := []struct {
		name     string
		param    *parser.Parameter
		hasRef   bool
		expected string
	}{
		{
			name: "parameter with OAS 2.0 reference",
			param: &parser.Parameter{
				Ref: "#/parameters/UserId",
			},
			hasRef:   true,
			expected: "#/components/parameters/UserId",
		},
		{
			name: "parameter without reference",
			param: &parser.Parameter{
				Name: "limit",
				In:   "query",
			},
			hasRef: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rewriteParameterRefsOAS2ToOAS3(tt.param)

			if tt.hasRef {
				assert.Equal(t, tt.expected, tt.param.Ref, "Reference should be rewritten to OAS 3.x format")
			}
		})
	}
}

// TestRewriteParameterRefsOAS3ToOAS2 tests the rewriteParameterRefsOAS3ToOAS2 function.
func TestRewriteParameterRefsOAS3ToOAS2(t *testing.T) {
	tests := []struct {
		name     string
		param    *parser.Parameter
		hasRef   bool
		expected string
	}{
		{
			name: "parameter with OAS 3.x reference",
			param: &parser.Parameter{
				Ref: "#/components/parameters/UserId",
			},
			hasRef:   true,
			expected: "#/parameters/UserId",
		},
		{
			name: "parameter without reference",
			param: &parser.Parameter{
				Name: "limit",
				In:   "query",
			},
			hasRef: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rewriteParameterRefsOAS3ToOAS2(tt.param)

			if tt.hasRef {
				assert.Equal(t, tt.expected, tt.param.Ref, "Reference should be rewritten to OAS 2.0 format")
			}
		})
	}
}

// TestRewriteRequestBodyRefsOAS2ToOAS3 tests the rewriteRequestBodyRefsOAS2ToOAS3 function.
func TestRewriteRequestBodyRefsOAS2ToOAS3(t *testing.T) {
	tests := []struct {
		name        string
		requestBody *parser.RequestBody
		hasRefs     bool
		expected    string
	}{
		{
			name: "request body with schema reference",
			requestBody: &parser.RequestBody{
				Content: map[string]*parser.MediaType{
					"application/json": {
						Schema: &parser.Schema{
							Ref: "#/definitions/User",
						},
					},
				},
			},
			hasRefs:  true,
			expected: "#/components/schemas/User",
		},
		{
			name: "request body without reference",
			requestBody: &parser.RequestBody{
				Content: map[string]*parser.MediaType{
					"application/json": {
						Schema: &parser.Schema{
							Type: "object",
						},
					},
				},
			},
			hasRefs: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rewriteRequestBodyRefsOAS2ToOAS3(tt.requestBody)

			if tt.hasRefs {
				for _, mediaType := range tt.requestBody.Content {
					if mediaType.Schema != nil && mediaType.Schema.Ref != "" {
						assert.Equal(t, tt.expected, mediaType.Schema.Ref, "Reference should be rewritten to OAS 3.x format")
					}
				}
			}
		})
	}
}
