package converter

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConvertParametersToOAS2 tests the convertParametersToOAS2 method.
func TestConvertParametersToOAS2(t *testing.T) {
	tests := []struct {
		name      string
		params    []*parser.Parameter
		wantCount int
		wantNil   bool
	}{
		{
			name:      "nil parameters",
			params:    nil,
			wantNil:   true,
			wantCount: 0,
		},
		{
			name:      "empty parameters",
			params:    []*parser.Parameter{},
			wantNil:   true,
			wantCount: 0,
		},
		{
			name: "single parameter",
			params: []*parser.Parameter{
				{
					Name:        "id",
					In:          "path",
					Description: "User ID",
					Required:    true,
					Schema: &parser.Schema{
						Type: "string",
					},
				},
			},
			wantNil:   false,
			wantCount: 1,
		},
		{
			name: "multiple parameters",
			params: []*parser.Parameter{
				{
					Name: "id",
					In:   "path",
					Schema: &parser.Schema{
						Type: "integer",
					},
				},
				{
					Name: "filter",
					In:   "query",
					Schema: &parser.Schema{
						Type: "string",
					},
				},
			},
			wantNil:   false,
			wantCount: 2,
		},
		{
			name: "parameters with nil entries",
			params: []*parser.Parameter{
				{
					Name: "id",
					In:   "path",
					Schema: &parser.Schema{
						Type: "string",
					},
				},
				nil,
				{
					Name: "filter",
					In:   "query",
					Schema: &parser.Schema{
						Type: "string",
					},
				},
			},
			wantNil:   false,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			result := &ConversionResult{}
			converted := c.convertParametersToOAS2(tt.params, result, "test.path")

			if tt.wantNil {
				assert.Nil(t, converted, "Expected nil result for empty parameters")
			} else {
				require.NotNil(t, converted, "Expected non-nil result")
				assert.Len(t, converted, tt.wantCount, "Expected %d converted parameters", tt.wantCount)
			}
		})
	}
}

// TestConvertSecuritySchemes tests the convertSecuritySchemes method.
func TestConvertSecuritySchemes(t *testing.T) {
	tests := []struct {
		name         string
		src          *parser.OAS3Document
		wantSchemes  int
		checkIssues  bool
		expectedType string
		schemeName   string
	}{
		{
			name: "no security schemes",
			src: &parser.OAS3Document{
				Components: nil,
			},
			wantSchemes: 0,
		},
		{
			name: "basic http auth",
			src: &parser.OAS3Document{
				Components: &parser.Components{
					SecuritySchemes: map[string]*parser.SecurityScheme{
						"basicAuth": {
							Type:   "http",
							Scheme: "basic",
						},
					},
				},
			},
			wantSchemes:  1,
			expectedType: "basic",
			schemeName:   "basicAuth",
		},
		{
			name: "non-basic http auth",
			src: &parser.OAS3Document{
				Components: &parser.Components{
					SecuritySchemes: map[string]*parser.SecurityScheme{
						"bearerAuth": {
							Type:   "http",
							Scheme: "bearer",
						},
					},
				},
			},
			wantSchemes:  1,
			checkIssues:  true,
			expectedType: "basic", // Converted to basic
			schemeName:   "bearerAuth",
		},
		{
			name: "oauth2 implicit flow",
			src: &parser.OAS3Document{
				Components: &parser.Components{
					SecuritySchemes: map[string]*parser.SecurityScheme{
						"oauth2": {
							Type: "oauth2",
							Flows: &parser.OAuthFlows{
								Implicit: &parser.OAuthFlow{
									AuthorizationURL: "https://example.com/oauth/authorize",
									Scopes: map[string]string{
										"read":  "Read access",
										"write": "Write access",
									},
								},
							},
						},
					},
				},
			},
			wantSchemes:  1,
			expectedType: "oauth2",
			schemeName:   "oauth2",
		},
		{
			name: "oauth2 password flow",
			src: &parser.OAS3Document{
				Components: &parser.Components{
					SecuritySchemes: map[string]*parser.SecurityScheme{
						"oauth2": {
							Type: "oauth2",
							Flows: &parser.OAuthFlows{
								Password: &parser.OAuthFlow{
									TokenURL: "https://example.com/oauth/token",
									Scopes: map[string]string{
										"read": "Read access",
									},
								},
							},
						},
					},
				},
			},
			wantSchemes:  1,
			expectedType: "oauth2",
			schemeName:   "oauth2",
		},
		{
			name: "oauth2 client credentials flow",
			src: &parser.OAS3Document{
				Components: &parser.Components{
					SecuritySchemes: map[string]*parser.SecurityScheme{
						"oauth2": {
							Type: "oauth2",
							Flows: &parser.OAuthFlows{
								ClientCredentials: &parser.OAuthFlow{
									TokenURL: "https://example.com/oauth/token",
									Scopes: map[string]string{
										"admin": "Admin access",
									},
								},
							},
						},
					},
				},
			},
			wantSchemes:  1,
			expectedType: "oauth2",
			schemeName:   "oauth2",
		},
		{
			name: "oauth2 authorization code flow",
			src: &parser.OAS3Document{
				Components: &parser.Components{
					SecuritySchemes: map[string]*parser.SecurityScheme{
						"oauth2": {
							Type: "oauth2",
							Flows: &parser.OAuthFlows{
								AuthorizationCode: &parser.OAuthFlow{
									AuthorizationURL: "https://example.com/oauth/authorize",
									TokenURL:         "https://example.com/oauth/token",
									Scopes: map[string]string{
										"read": "Read access",
									},
								},
							},
						},
					},
				},
			},
			wantSchemes:  1,
			expectedType: "oauth2",
			schemeName:   "oauth2",
		},
		{
			name: "oauth2 multiple flows",
			src: &parser.OAS3Document{
				Components: &parser.Components{
					SecuritySchemes: map[string]*parser.SecurityScheme{
						"oauth2": {
							Type: "oauth2",
							Flows: &parser.OAuthFlows{
								Implicit: &parser.OAuthFlow{
									AuthorizationURL: "https://example.com/oauth/authorize",
									Scopes:           map[string]string{"read": "Read"},
								},
								Password: &parser.OAuthFlow{
									TokenURL: "https://example.com/oauth/token",
									Scopes:   map[string]string{"write": "Write"},
								},
							},
						},
					},
				},
			},
			wantSchemes:  1,
			checkIssues:  true, // Should warn about multiple flows
			expectedType: "oauth2",
			schemeName:   "oauth2",
		},
		{
			name: "oauth2 no flows",
			src: &parser.OAS3Document{
				Components: &parser.Components{
					SecuritySchemes: map[string]*parser.SecurityScheme{
						"oauth2": {
							Type:  "oauth2",
							Flows: &parser.OAuthFlows{},
						},
					},
				},
			},
			wantSchemes:  1,
			checkIssues:  true, // Should warn about no flows
			expectedType: "oauth2",
			schemeName:   "oauth2",
		},
		{
			name: "openIdConnect",
			src: &parser.OAS3Document{
				Components: &parser.Components{
					SecuritySchemes: map[string]*parser.SecurityScheme{
						"openId": {
							Type:             "openIdConnect",
							OpenIDConnectURL: "https://example.com/.well-known/openid-configuration",
						},
					},
				},
			},
			wantSchemes: 0, // Should be skipped with critical issue
			checkIssues: true,
		},
		{
			name: "apiKey scheme",
			src: &parser.OAS3Document{
				Components: &parser.Components{
					SecuritySchemes: map[string]*parser.SecurityScheme{
						"apiKey": {
							Type: "apiKey",
							Name: "X-API-Key",
							In:   "header",
						},
					},
				},
			},
			wantSchemes:  1,
			expectedType: "apiKey",
			schemeName:   "apiKey",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			dst := &parser.OAS2Document{}
			result := &ConversionResult{}

			c.convertSecuritySchemes(tt.src, dst, result)

			if tt.wantSchemes == 0 {
				if dst.SecurityDefinitions != nil {
					assert.Len(t, dst.SecurityDefinitions, 0, "Expected no security definitions")
				}
			} else {
				require.NotNil(t, dst.SecurityDefinitions, "Expected security definitions to be created")
				require.Len(t, dst.SecurityDefinitions, tt.wantSchemes, "Expected %d security schemes", tt.wantSchemes)

				if tt.schemeName != "" && tt.expectedType != "" {
					scheme, exists := dst.SecurityDefinitions[tt.schemeName]
					require.True(t, exists, "Expected scheme %s to exist", tt.schemeName)
					assert.Equal(t, tt.expectedType, scheme.Type, "Expected type %s", tt.expectedType)
				}
			}

			if tt.checkIssues {
				assert.NotEmpty(t, result.Issues, "Expected conversion issues")
			}
		})
	}
}

// TestConvertOAS3OperationToOAS2_EdgeCases tests edge cases in convertOAS3OperationToOAS2.
func TestConvertOAS3OperationToOAS2_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		operation   *parser.Operation
		doc         *parser.OAS2Document
		checkResult func(*testing.T, *parser.Operation, *ConversionResult)
	}{
		{
			name: "operation with callbacks",
			operation: &parser.Operation{
				Summary:   "Create webhook",
				Responses: &parser.Responses{},
				Callbacks: map[string]*parser.Callback{
					"onData": {
						"{$request.body#/callbackUrl}": &parser.PathItem{
							Post: &parser.Operation{
								Summary: "Callback",
							},
						},
					},
				},
			},
			doc: &parser.OAS2Document{},
			checkResult: func(t *testing.T, op *parser.Operation, result *ConversionResult) {
				assert.NotNil(t, op)
				assert.NotEmpty(t, result.Issues, "Expected issue about callbacks")
				foundCallback := false
				for _, issue := range result.Issues {
					if issue.Severity == SeverityCritical && strings.Contains(issue.Message, "callbacks") {
						foundCallback = true
						break
					}
				}
				assert.True(t, foundCallback, "Expected critical issue about callbacks")
			},
		},
		{
			name: "operation with requestBody and consumes",
			operation: &parser.Operation{
				Summary: "Upload file",
				RequestBody: &parser.RequestBody{
					Content: map[string]*parser.MediaType{
						"application/json": {
							Schema: &parser.Schema{Type: "object"},
						},
						"application/xml": {
							Schema: &parser.Schema{Type: "object"},
						},
					},
				},
				Responses: &parser.Responses{},
			},
			doc: &parser.OAS2Document{},
			checkResult: func(t *testing.T, op *parser.Operation, result *ConversionResult) {
				assert.NotNil(t, op)
				assert.NotEmpty(t, op.Parameters, "Expected body parameter")
				assert.NotEmpty(t, op.Consumes, "Expected consumes to be set")
			},
		},
		{
			name: "operation with response produces",
			operation: &parser.Operation{
				Summary: "Get data",
				Responses: &parser.Responses{
					Default: &parser.Response{
						Description: "Default response",
						Content: map[string]*parser.MediaType{
							"application/json": {
								Schema: &parser.Schema{Type: "object"},
							},
						},
					},
					Codes: map[string]*parser.Response{
						"200": {
							Description: "Success",
							Content: map[string]*parser.MediaType{
								"application/xml": {
									Schema: &parser.Schema{Type: "object"},
								},
							},
						},
					},
				},
			},
			doc: &parser.OAS2Document{},
			checkResult: func(t *testing.T, op *parser.Operation, result *ConversionResult) {
				assert.NotNil(t, op)
				assert.NotEmpty(t, op.Produces, "Expected produces to be set")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			result := &ConversionResult{}
			converted := c.convertOAS3OperationToOAS2(tt.operation, tt.doc, result, "paths./test.get")

			tt.checkResult(t, converted, result)
		})
	}
}

// TestConvertServersToHostBasePath tests the convertServersToHostBasePath method.
func TestConvertServersToHostBasePath(t *testing.T) {
	tests := []struct {
		name           string
		src            *parser.OAS3Document
		expectedHost   string
		expectedBase   string
		expectedScheme []string
		checkIssues    bool
	}{
		{
			name: "no servers",
			src: &parser.OAS3Document{
				Servers: []*parser.Server{},
			},
			expectedHost:   "localhost",
			expectedBase:   "/",
			expectedScheme: []string{"https"},
			checkIssues:    true,
		},
		{
			name: "single server",
			src: &parser.OAS3Document{
				Servers: []*parser.Server{
					{URL: "https://api.example.com/v1"},
				},
			},
			expectedHost:   "api.example.com",
			expectedBase:   "/v1",
			expectedScheme: []string{"https"},
		},
		{
			name: "multiple servers",
			src: &parser.OAS3Document{
				Servers: []*parser.Server{
					{URL: "https://api.example.com/v1"},
					{URL: "http://staging.example.com/v1"},
				},
			},
			expectedHost:   "api.example.com",
			expectedBase:   "/v1",
			expectedScheme: []string{"https"},
			checkIssues:    true, // Should warn about multiple servers
		},
		{
			name: "server with variables",
			src: &parser.OAS3Document{
				Servers: []*parser.Server{
					{
						URL: "https://api.example.com/v1",
						Variables: map[string]parser.ServerVariable{
							"version": {
								Default: "v1",
							},
						},
					},
				},
			},
			expectedHost:   "api.example.com",
			expectedBase:   "/v1",
			expectedScheme: []string{"https"},
			checkIssues:    true, // Should warn about variables
		},
		{
			name: "invalid server URL",
			src: &parser.OAS3Document{
				Servers: []*parser.Server{
					{URL: "://invalid-url"},
				},
			},
			expectedHost:   "localhost",
			expectedBase:   "/",
			expectedScheme: []string{"https"},
			checkIssues:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			dst := &parser.OAS2Document{}
			result := &ConversionResult{}

			c.convertServersToHostBasePath(tt.src, dst, result)

			assert.Equal(t, tt.expectedHost, dst.Host, "Expected host %s", tt.expectedHost)
			assert.Equal(t, tt.expectedBase, dst.BasePath, "Expected basePath %s", tt.expectedBase)
			assert.Equal(t, tt.expectedScheme, dst.Schemes, "Expected schemes %v", tt.expectedScheme)

			if tt.checkIssues {
				assert.NotEmpty(t, result.Issues, "Expected conversion issues")
			}
		})
	}
}

// TestConvertOAS3PathItemToOAS2_Trace tests TRACE method handling in convertOAS3PathItemToOAS2.
func TestConvertOAS3PathItemToOAS2_Trace(t *testing.T) {
	c := New()
	pathItem := &parser.PathItem{
		Trace: &parser.Operation{
			Summary:   "Trace endpoint",
			Responses: &parser.Responses{},
		},
	}
	doc := &parser.OAS2Document{}
	result := &ConversionResult{}

	converted := c.convertOAS3PathItemToOAS2(pathItem, doc, result, "paths./test")

	assert.NotNil(t, converted)
	assert.Nil(t, converted.Trace, "TRACE should not be in OAS 2.0")
	require.NotEmpty(t, result.Issues, "Expected issue about TRACE method")

	foundTrace := false
	for _, issue := range result.Issues {
		if issue.Severity == SeverityCritical && strings.Contains(issue.Message, "TRACE") {
			foundTrace = true
			break
		}
	}
	assert.True(t, foundTrace, "Expected critical issue about TRACE method")
}

// TestConvertOAS3PathItemToOAS2_Query tests QUERY method handling in convertOAS3PathItemToOAS2.
func TestConvertOAS3PathItemToOAS2_Query(t *testing.T) {
	c := New()
	pathItem := &parser.PathItem{
		Query: &parser.Operation{
			Summary:   "Query endpoint",
			Responses: &parser.Responses{},
		},
	}
	doc := &parser.OAS2Document{}
	result := &ConversionResult{}

	converted := c.convertOAS3PathItemToOAS2(pathItem, doc, result, "paths./search")

	assert.NotNil(t, converted)
	assert.Nil(t, converted.Query, "QUERY should not be in OAS 2.0")
	require.NotEmpty(t, result.Issues, "Expected issue about QUERY method")

	foundQuery := false
	for _, issue := range result.Issues {
		if issue.Severity == SeverityCritical && strings.Contains(issue.Message, "QUERY") {
			foundQuery = true
			break
		}
	}
	assert.True(t, foundQuery, "Expected critical issue about QUERY method")
}

// TestConvertOAS3ToOAS2_SchemaFeatureDetection tests end-to-end detection of OAS 3.x
// schema features during conversion to OAS 2.0 via ConvertParsed.
func TestConvertOAS3ToOAS2_SchemaFeatureDetection(t *testing.T) {
	tests := []struct {
		name            string
		schema          *parser.Schema
		expectedKeyword string
	}{
		{
			name:            "writeOnly in component schema",
			schema:          &parser.Schema{Type: "string", WriteOnly: true},
			expectedKeyword: "writeOnly",
		},
		{
			name:            "deprecated in component schema",
			schema:          &parser.Schema{Type: "object", Deprecated: true},
			expectedKeyword: "deprecated",
		},
		{
			name: "if in component schema",
			schema: &parser.Schema{
				Type: "object",
				If:   &parser.Schema{Properties: map[string]*parser.Schema{"x": {Type: "string"}}},
			},
			expectedKeyword: "if",
		},
		{
			name: "prefixItems in component schema",
			schema: &parser.Schema{
				Type:        "array",
				PrefixItems: []*parser.Schema{{Type: "string"}, {Type: "integer"}},
			},
			expectedKeyword: "prefixItems",
		},
		{
			name: "contains in component schema",
			schema: &parser.Schema{
				Type:     "array",
				Contains: &parser.Schema{Type: "integer"},
			},
			expectedKeyword: "contains",
		},
		{
			name: "propertyNames in component schema",
			schema: &parser.Schema{
				Type:          "object",
				PropertyNames: &parser.Schema{Pattern: "^[a-z]+$"},
			},
			expectedKeyword: "propertyNames",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &parser.OAS3Document{
				OpenAPI: "3.1.0",
				Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
				Paths:   map[string]*parser.PathItem{},
				Components: &parser.Components{
					Schemas: map[string]*parser.Schema{
						"TestSchema": tt.schema,
					},
				},
			}

			parseResult := parser.ParseResult{
				Document:   doc,
				Version:    "3.1.0",
				OASVersion: parser.OASVersion310,
				Data:       make(map[string]any),
				SourcePath: "test.yaml",
			}

			c := New()
			result, err := c.ConvertParsed(parseResult, "2.0")

			require.NoError(t, err)
			require.NotNil(t, result)

			found := false
			for _, issue := range result.Issues {
				if strings.Contains(issue.Message, tt.expectedKeyword) {
					found = true
					assert.Equal(t, SeverityWarning, issue.Severity,
						"Feature detection issues should be warnings")
					assert.Contains(t, issue.Path, "components.schemas.TestSchema",
						"Issue path should reference the schema")
					break
				}
			}
			assert.True(t, found, "Should have issue for %s", tt.expectedKeyword)
		})
	}
}

// TestConvertOAS3ParameterToOAS2_TypeFallback tests type inference from composite schemas.
func TestConvertOAS3ParameterToOAS2_TypeFallback(t *testing.T) {
	tests := []struct {
		name         string
		param        *parser.Parameter
		expectedType string
	}{
		{
			name: "allOf with concrete type",
			param: &parser.Parameter{
				Name: "filter",
				In:   "query",
				Schema: &parser.Schema{
					AllOf: []*parser.Schema{
						{Type: "string"},
						{Description: "filter constraint"},
					},
				},
			},
			expectedType: "string",
		},
		{
			name: "oneOf with concrete types",
			param: &parser.Parameter{
				Name: "id",
				In:   "query",
				Schema: &parser.Schema{
					OneOf: []*parser.Schema{
						{Type: "string"},
						{Type: "integer"},
					},
				},
			},
			expectedType: "string",
		},
		{
			name: "no type defaults to string",
			param: &parser.Parameter{
				Name:   "unknown",
				In:     "query",
				Schema: &parser.Schema{},
			},
			expectedType: "string",
		},
		{
			name: "anyOf with concrete type",
			param: &parser.Parameter{
				Name: "value",
				In:   "query",
				Schema: &parser.Schema{
					AnyOf: []*parser.Schema{
						{Type: "number"},
						{Type: "string"},
					},
				},
			},
			expectedType: "number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Converter{}
			result := &ConversionResult{}
			converted := c.convertOAS3ParameterToOAS2(tt.param, result, "parameters.test")

			require.NotNil(t, converted)
			assert.Equal(t, tt.expectedType, converted.Type)
			assert.NotEmpty(t, result.Issues, "expected conversion issue")
		})
	}
}

// TestConvertOAS3ToOAS2_InlineHeaderRefs tests that component header $refs are inlined
// during 3.0-to-2.0 conversion.
func TestConvertOAS3ToOAS2_InlineHeaderRefs(t *testing.T) {
	c := &Converter{}
	result := &ConversionResult{}

	src := parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
		Document: &parser.OAS3Document{
			OpenAPI:    "3.0.0",
			OASVersion: parser.OASVersion300,
			Info:       &parser.Info{Title: "Test", Version: "1.0.0"},
			Paths: parser.Paths{
				"/test": &parser.PathItem{
					Get: &parser.Operation{
						OperationID: "getTest",
						Responses: &parser.Responses{
							Codes: map[string]*parser.Response{
								"200": {
									Description: "OK",
									Headers: map[string]*parser.Header{
										"X-Rate-Limit": {
											Ref: "#/components/headers/X-Rate-Limit",
										},
									},
								},
							},
						},
					},
				},
			},
			Components: &parser.Components{
				Headers: map[string]*parser.Header{
					"X-Rate-Limit": {
						Description: "Rate limit",
						Schema:      &parser.Schema{Type: "integer"},
					},
				},
			},
		},
	}

	err := c.convertOAS3ToOAS2(src, result)
	require.NoError(t, err)

	doc, ok := result.Document.(*parser.OAS2Document)
	require.True(t, ok)

	pathItem, ok := doc.Paths["/test"]
	require.True(t, ok, "path /test should exist")
	require.NotNil(t, pathItem.Get, "GET operation should exist")
	require.NotNil(t, pathItem.Get.Responses, "responses should exist")
	require.NotNil(t, pathItem.Get.Responses.Codes, "response codes should exist")
	resp, ok := pathItem.Get.Responses.Codes["200"]
	require.True(t, ok, "200 response should exist")
	require.NotNil(t, resp.Headers, "headers should exist")
	require.Contains(t, resp.Headers, "X-Rate-Limit")

	header := resp.Headers["X-Rate-Limit"]
	assert.Empty(t, header.Ref, "ref should be resolved")
	assert.Equal(t, "Rate limit", header.Description)
}

// TestConvertOAS3ToOAS2_NestedSchemaFeatureDetection tests end-to-end detection of OAS 3.x
// schema features in nested schemas during conversion to OAS 2.0 via ConvertParsed.
func TestConvertOAS3ToOAS2_NestedSchemaFeatureDetection(t *testing.T) {
	doc := &parser.OAS3Document{
		OpenAPI: "3.1.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   map[string]*parser.PathItem{},
		Components: &parser.Components{
			Schemas: map[string]*parser.Schema{
				"User": {
					Type: "object",
					Properties: map[string]*parser.Schema{
						"name":     {Type: "string"},
						"password": {Type: "string", WriteOnly: true},
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Document:   doc,
		Version:    "3.1.0",
		OASVersion: parser.OASVersion310,
		Data:       make(map[string]any),
		SourcePath: "test.yaml",
	}

	c := New()
	result, err := c.ConvertParsed(parseResult, "2.0")

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify that the nested writeOnly feature was detected
	found := false
	for _, issue := range result.Issues {
		if strings.Contains(issue.Message, "writeOnly") &&
			strings.Contains(issue.Path, "properties.password") {
			found = true
			assert.Equal(t, SeverityWarning, issue.Severity,
				"Nested feature detection issues should be warnings")
			break
		}
	}
	assert.True(t, found,
		"Should detect writeOnly in nested property User.properties.password")
}
