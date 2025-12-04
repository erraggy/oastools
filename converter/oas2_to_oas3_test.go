package converter

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConvertSecurityDefinitions tests the convertSecurityDefinitions method.
func TestConvertSecurityDefinitions(t *testing.T) {
	tests := []struct {
		name        string
		src         *parser.OAS2Document
		wantSchemes int
		checkIssues bool
		schemeName  string
	}{
		{
			name: "basic auth",
			src: &parser.OAS2Document{
				SecurityDefinitions: map[string]*parser.SecurityScheme{
					"basicAuth": {
						Type: "basic",
					},
				},
			},
			wantSchemes: 1,
			schemeName:  "basicAuth",
		},
		{
			name: "apiKey auth",
			src: &parser.OAS2Document{
				SecurityDefinitions: map[string]*parser.SecurityScheme{
					"apiKey": {
						Type: "apiKey",
						Name: "X-API-Key",
						In:   "header",
					},
				},
			},
			wantSchemes: 1,
			schemeName:  "apiKey",
		},
		{
			name: "oauth2 implicit flow",
			src: &parser.OAS2Document{
				SecurityDefinitions: map[string]*parser.SecurityScheme{
					"oauth2": {
						Type:             "oauth2",
						Flow:             "implicit",
						AuthorizationURL: "https://example.com/oauth/authorize",
						Scopes: map[string]string{
							"read":  "Read access",
							"write": "Write access",
						},
					},
				},
			},
			wantSchemes: 1,
			schemeName:  "oauth2",
		},
		{
			name: "oauth2 password flow",
			src: &parser.OAS2Document{
				SecurityDefinitions: map[string]*parser.SecurityScheme{
					"oauth2": {
						Type:     "oauth2",
						Flow:     "password",
						TokenURL: "https://example.com/oauth/token",
						Scopes: map[string]string{
							"admin": "Admin access",
						},
					},
				},
			},
			wantSchemes: 1,
			schemeName:  "oauth2",
		},
		{
			name: "oauth2 application flow",
			src: &parser.OAS2Document{
				SecurityDefinitions: map[string]*parser.SecurityScheme{
					"oauth2": {
						Type:     "oauth2",
						Flow:     "application",
						TokenURL: "https://example.com/oauth/token",
						Scopes: map[string]string{
							"read": "Read access",
						},
					},
				},
			},
			wantSchemes: 1,
			schemeName:  "oauth2",
		},
		{
			name: "oauth2 accessCode flow",
			src: &parser.OAS2Document{
				SecurityDefinitions: map[string]*parser.SecurityScheme{
					"oauth2": {
						Type:             "oauth2",
						Flow:             "accessCode",
						AuthorizationURL: "https://example.com/oauth/authorize",
						TokenURL:         "https://example.com/oauth/token",
						Scopes: map[string]string{
							"read":  "Read access",
							"write": "Write access",
						},
					},
				},
			},
			wantSchemes: 1,
			schemeName:  "oauth2",
		},
		{
			name: "oauth2 unknown flow",
			src: &parser.OAS2Document{
				SecurityDefinitions: map[string]*parser.SecurityScheme{
					"oauth2": {
						Type: "oauth2",
						Flow: "unknown",
					},
				},
			},
			wantSchemes: 1,
			checkIssues: true,
			schemeName:  "oauth2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			dst := &parser.OAS3Document{
				Components: &parser.Components{
					SecuritySchemes: make(map[string]*parser.SecurityScheme),
				},
			}
			result := &ConversionResult{}

			c.convertSecurityDefinitions(tt.src, dst, result)

			require.NotNil(t, dst.Components.SecuritySchemes, "Expected security schemes to be created")
			assert.Len(t, dst.Components.SecuritySchemes, tt.wantSchemes, "Expected %d security schemes", tt.wantSchemes)

			if tt.schemeName != "" {
				scheme, exists := dst.Components.SecuritySchemes[tt.schemeName]
				require.True(t, exists, "Expected scheme %s to exist", tt.schemeName)
				assert.NotNil(t, scheme, "Expected scheme to be non-nil")
			}

			if tt.checkIssues {
				assert.NotEmpty(t, result.Issues, "Expected conversion issues")
			}
		})
	}
}

// TestConvertOAS2OperationToOAS3_EdgeCases tests edge cases in convertOAS2OperationToOAS3.
func TestConvertOAS2OperationToOAS3_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		operation   *parser.Operation
		doc         *parser.OAS2Document
		checkResult func(*testing.T, *parser.Operation)
	}{
		{
			name: "operation without body parameter",
			operation: &parser.Operation{
				Summary: "Get user",
				Parameters: []*parser.Parameter{
					{
						Name:     "id",
						In:       "path",
						Required: true,
						Type:     "string",
					},
				},
				Responses: &parser.Responses{},
			},
			doc: &parser.OAS2Document{},
			checkResult: func(t *testing.T, op *parser.Operation) {
				assert.NotNil(t, op)
				assert.Nil(t, op.RequestBody, "Should not have request body")
				assert.Len(t, op.Parameters, 1, "Should have 1 parameter")
			},
		},
		{
			name: "operation with body parameter",
			operation: &parser.Operation{
				Summary: "Create user",
				Parameters: []*parser.Parameter{
					{
						Name:     "body",
						In:       "body",
						Required: true,
						Schema:   &parser.Schema{Type: "object"},
					},
					{
						Name: "X-Request-ID",
						In:   "header",
						Type: "string",
					},
				},
				Consumes: []string{"application/json"},
				Responses: &parser.Responses{},
			},
			doc: &parser.OAS2Document{},
			checkResult: func(t *testing.T, op *parser.Operation) {
				assert.NotNil(t, op)
				assert.NotNil(t, op.RequestBody, "Should have request body")
				assert.Len(t, op.Parameters, 1, "Should have 1 non-body parameter")
				assert.Equal(t, "header", op.Parameters[0].In, "Remaining parameter should be header")
			},
		},
		{
			name: "operation with responses",
			operation: &parser.Operation{
				Summary: "Get data",
				Responses: &parser.Responses{
					Default: &parser.Response{
						Description: "Default response",
						Schema:      &parser.Schema{Type: "object"},
					},
					Codes: map[string]*parser.Response{
						"200": {
							Description: "Success",
							Schema:      &parser.Schema{Type: "object"},
						},
						"404": {
							Description: "Not found",
						},
					},
				},
				Produces: []string{"application/json", "application/xml"},
			},
			doc: &parser.OAS2Document{},
			checkResult: func(t *testing.T, op *parser.Operation) {
				assert.NotNil(t, op)
				assert.NotNil(t, op.Responses, "Should have responses")
				assert.NotNil(t, op.Responses.Default, "Should have default response")
				assert.Len(t, op.Responses.Codes, 2, "Should have 2 status code responses")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			result := &ConversionResult{}
			converted := c.convertOAS2OperationToOAS3(tt.operation, tt.doc, result, "paths./test.get")

			tt.checkResult(t, converted)
		})
	}
}

// TestConvertOAS2PathItemToOAS3_AllMethods tests all HTTP methods in convertOAS2PathItemToOAS3.
func TestConvertOAS2PathItemToOAS3_AllMethods(t *testing.T) {
	c := New()
	
	pathItem := &parser.PathItem{
		Summary:     "Test path",
		Description: "Test description",
		Get: &parser.Operation{
			Summary:   "Get operation",
			Responses: &parser.Responses{},
		},
		Put: &parser.Operation{
			Summary:   "Put operation",
			Responses: &parser.Responses{},
		},
		Post: &parser.Operation{
			Summary:   "Post operation",
			Responses: &parser.Responses{},
		},
		Delete: &parser.Operation{
			Summary:   "Delete operation",
			Responses: &parser.Responses{},
		},
		Options: &parser.Operation{
			Summary:   "Options operation",
			Responses: &parser.Responses{},
		},
		Head: &parser.Operation{
			Summary:   "Head operation",
			Responses: &parser.Responses{},
		},
		Patch: &parser.Operation{
			Summary:   "Patch operation",
			Responses: &parser.Responses{},
		},
	}
	
	doc := &parser.OAS2Document{}
	result := &ConversionResult{}

	converted := c.convertOAS2PathItemToOAS3(pathItem, doc, result, "paths./test")

	assert.NotNil(t, converted)
	assert.Equal(t, pathItem.Summary, converted.Summary)
	assert.Equal(t, pathItem.Description, converted.Description)
	assert.NotNil(t, converted.Get, "Should have GET operation")
	assert.NotNil(t, converted.Put, "Should have PUT operation")
	assert.NotNil(t, converted.Post, "Should have POST operation")
	assert.NotNil(t, converted.Delete, "Should have DELETE operation")
	assert.NotNil(t, converted.Options, "Should have OPTIONS operation")
	assert.NotNil(t, converted.Head, "Should have HEAD operation")
	assert.NotNil(t, converted.Patch, "Should have PATCH operation")
}

// TestConvertOAS2PathItemToOAS3_NilInput tests nil handling in convertOAS2PathItemToOAS3.
func TestConvertOAS2PathItemToOAS3_NilInput(t *testing.T) {
	c := New()
	doc := &parser.OAS2Document{}
	result := &ConversionResult{}

	converted := c.convertOAS2PathItemToOAS3(nil, doc, result, "paths./test")

	assert.Nil(t, converted, "Should return nil for nil input")
}

// TestGetProduces tests the getProduces method with different scenarios.
func TestGetProduces(t *testing.T) {
	tests := []struct {
		name      string
		operation *parser.Operation
		document  *parser.OAS2Document
		expected  []string
	}{
		{
			name: "operation-level produces",
			operation: &parser.Operation{
				Produces: []string{"application/xml", "application/json"},
			},
			document: &parser.OAS2Document{
				Produces: []string{"text/plain"},
			},
			expected: []string{"application/xml", "application/json"},
		},
		{
			name:      "document-level produces",
			operation: &parser.Operation{},
			document: &parser.OAS2Document{
				Produces: []string{"application/json", "text/html"},
			},
			expected: []string{"application/json", "text/html"},
		},
		{
			name:      "no produces",
			operation: &parser.Operation{},
			document:  &parser.OAS2Document{},
			expected:  nil,
		},
		{
			name: "empty operation produces, use document",
			operation: &parser.Operation{
				Produces: []string{},
			},
			document: &parser.OAS2Document{
				Produces: []string{"application/json"},
			},
			expected: []string{"application/json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			result := c.getProduces(tt.operation, tt.document)
			assert.Equal(t, tt.expected, result, "getProduces should return correct value")
		})
	}
}

// TestConvertServers tests the convertServers method with different scenarios.
func TestConvertServers(t *testing.T) {
	tests := []struct {
		name         string
		src          *parser.OAS2Document
		wantServers  int
		checkFirst   bool
		expectedURL  string
		checkIssues  bool
	}{
		{
			name: "no host",
			src: &parser.OAS2Document{
				Host: "",
			},
			wantServers: 1,
			checkFirst:  true,
			expectedURL: "/",
			checkIssues: true,
		},
		{
			name: "host with https scheme",
			src: &parser.OAS2Document{
				Host:     "api.example.com",
				BasePath: "/v1",
				Schemes:  []string{"https"},
			},
			wantServers: 1,
			checkFirst:  true,
			expectedURL: "https://api.example.com/v1",
		},
		{
			name: "host with multiple schemes",
			src: &parser.OAS2Document{
				Host:     "api.example.com",
				BasePath: "/v2",
				Schemes:  []string{"http", "https"},
			},
			wantServers: 2,
			checkFirst:  true,
			expectedURL: "http://api.example.com/v2",
		},
		{
			name: "host with no schemes",
			src: &parser.OAS2Document{
				Host:     "api.example.com",
				BasePath: "/api",
				Schemes:  []string{},
			},
			wantServers: 1,
			checkFirst:  true,
			expectedURL: "https://api.example.com/api",
		},
		{
			name: "host with no basePath",
			src: &parser.OAS2Document{
				Host:    "api.example.com",
				Schemes: []string{"https"},
			},
			wantServers: 1,
			checkFirst:  true,
			expectedURL: "https://api.example.com/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			result := &ConversionResult{}
			servers := c.convertServers(tt.src, result)

			assert.Len(t, servers, tt.wantServers, "Expected %d servers", tt.wantServers)

			if tt.checkFirst && len(servers) > 0 {
				assert.Equal(t, tt.expectedURL, servers[0].URL, "Expected URL %s", tt.expectedURL)
			}

			if tt.checkIssues {
				assert.NotEmpty(t, result.Issues, "Expected conversion issues")
			}
		})
	}
}

// TestConvertParameters tests the convertParameters method.
func TestConvertParameters(t *testing.T) {
	tests := []struct {
		name      string
		params    []*parser.Parameter
		wantCount int
		wantNil   bool
	}{
		{
			name:    "nil parameters",
			params:  nil,
			wantNil: true,
		},
		{
			name:    "empty parameters",
			params:  []*parser.Parameter{},
			wantNil: true,
		},
		{
			name: "single parameter",
			params: []*parser.Parameter{
				{
					Name:     "id",
					In:       "path",
					Required: true,
					Type:     "string",
				},
			},
			wantCount: 1,
		},
		{
			name: "multiple parameters",
			params: []*parser.Parameter{
				{
					Name: "id",
					In:   "path",
					Type: "integer",
				},
				{
					Name: "filter",
					In:   "query",
					Type: "string",
				},
			},
			wantCount: 2,
		},
		{
			name: "parameters with nil entry",
			params: []*parser.Parameter{
				{
					Name: "id",
					In:   "path",
					Type: "string",
				},
				nil,
			},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			result := &ConversionResult{}
			converted := c.convertParameters(tt.params, result, "test.path")

			if tt.wantNil {
				assert.Nil(t, converted, "Expected nil for empty parameters")
			} else {
				require.NotNil(t, converted, "Expected non-nil result")
				assert.Len(t, converted, tt.wantCount, "Expected %d parameters", tt.wantCount)
			}
		})
	}
}
