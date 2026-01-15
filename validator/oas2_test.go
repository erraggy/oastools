package validator

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// validateOAS2Parameters Tests
// =============================================================================

func TestValidateOAS2Parameters_BodyParamMissingSchema(t *testing.T) {
	// Test: body parameter without schema should error
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		Parameters: map[string]*parser.Parameter{
			"BodyParam": {
				Name: "body",
				In:   "body",
				// Schema intentionally missing
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about body parameter missing schema
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "parameters.BodyParam") &&
			strings.Contains(e.Message, "Body parameter must have a schema") {
			foundError = true
			assert.Equal(t, "schema", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about body parameter missing schema")
}

func TestValidateOAS2Parameters_NonBodyParamMissingType(t *testing.T) {
	// Test: non-body parameter without type should error
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		Parameters: map[string]*parser.Parameter{
			"QueryParam": {
				Name: "filter",
				In:   "query",
				// Type intentionally missing
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about non-body parameter missing type
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "parameters.QueryParam") &&
			strings.Contains(e.Message, "Non-body parameter must have a type") {
			foundError = true
			assert.Equal(t, "type", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about non-body parameter missing type")
}

func TestValidateOAS2Parameters_ValidParameters(t *testing.T) {
	// Test: valid parameters should not error
	schema := &parser.Schema{Type: "object"}
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		Parameters: map[string]*parser.Parameter{
			"ValidBodyParam": {
				Name:   "body",
				In:     "body",
				Schema: schema,
			},
			"ValidQueryParam": {
				Name: "filter",
				In:   "query",
				Type: "string",
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Should not have errors about parameters
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "parameters.") {
			if strings.Contains(e.Message, "Body parameter must have a schema") ||
				strings.Contains(e.Message, "Non-body parameter must have a type") {
				t.Errorf("Unexpected parameter error: %s", e.Message)
			}
		}
	}
}

func TestValidateOAS2Parameters_NilParameter(t *testing.T) {
	// Test: nil parameter should be skipped
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		Parameters: map[string]*parser.Parameter{
			"NilParam": nil,
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Should not panic and should not error on nil parameter
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "parameters.NilParam") {
			t.Errorf("Unexpected error for nil parameter: %s", e.Message)
		}
	}
}

// =============================================================================
// validateOAS2Responses Tests
// =============================================================================

func TestValidateOAS2Responses_MissingDescription(t *testing.T) {
	// Test: response without description should error
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		Responses: map[string]*parser.Response{
			"NotFound": {
				// Description intentionally missing
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about response missing description
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "responses.NotFound") &&
			strings.Contains(e.Message, "Response must have a description") {
			foundError = true
			assert.Equal(t, "description", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about response missing description")
}

func TestValidateOAS2Responses_ValidResponse(t *testing.T) {
	// Test: response with description should not error
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		Responses: map[string]*parser.Response{
			"NotFound": {
				Description: "Resource not found",
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Should not have errors about responses
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "responses.NotFound") &&
			strings.Contains(e.Message, "Response must have a description") {
			t.Errorf("Unexpected response error: %s", e.Message)
		}
	}
}

func TestValidateOAS2Responses_NilResponse(t *testing.T) {
	// Test: nil response should be skipped
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		Responses: map[string]*parser.Response{
			"NilResponse": nil,
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Should not panic and should not error on nil response
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "responses.NilResponse") {
			t.Errorf("Unexpected error for nil response: %s", e.Message)
		}
	}
}

// =============================================================================
// validateOAS2Security Tests
// =============================================================================

func TestValidateOAS2Security_UndefinedSecurityScheme(t *testing.T) {
	// Test: security requirement referencing undefined scheme should error
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		Security: []parser.SecurityRequirement{
			{"undefined_scheme": []string{}},
		},
		SecurityDefinitions: map[string]*parser.SecurityScheme{},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about undefined security scheme
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "security[0].undefined_scheme") &&
			strings.Contains(e.Message, "references undefined security scheme") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about undefined security scheme")
}

func TestValidateOAS2Security_MissingType(t *testing.T) {
	// Test: security scheme without type should error
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		SecurityDefinitions: map[string]*parser.SecurityScheme{
			"noType": {
				// Type intentionally missing
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing type
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "securityDefinitions.noType") &&
			strings.Contains(e.Message, "Security scheme must have a type") {
			foundError = true
			assert.Equal(t, "type", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about security scheme missing type")
}

func TestValidateOAS2Security_ApiKeyMissingName(t *testing.T) {
	// Test: apiKey security scheme without name should error
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		SecurityDefinitions: map[string]*parser.SecurityScheme{
			"apiKey": {
				Type: "apiKey",
				In:   "header",
				// Name intentionally missing
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing name
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "securityDefinitions.apiKey") &&
			strings.Contains(e.Message, "API key security scheme must have a name") {
			foundError = true
			assert.Equal(t, "name", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about apiKey missing name")
}

func TestValidateOAS2Security_ApiKeyMissingIn(t *testing.T) {
	// Test: apiKey security scheme without 'in' should error
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		SecurityDefinitions: map[string]*parser.SecurityScheme{
			"apiKey": {
				Type: "apiKey",
				Name: "X-API-Key",
				// In intentionally missing
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing 'in'
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "securityDefinitions.apiKey") &&
			strings.Contains(e.Message, "API key security scheme must specify 'in'") {
			foundError = true
			assert.Equal(t, "in", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about apiKey missing 'in'")
}

func TestValidateOAS2Security_OAuth2MissingFlow(t *testing.T) {
	// Test: oauth2 security scheme without flow should error
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		SecurityDefinitions: map[string]*parser.SecurityScheme{
			"oauth2": {
				Type: "oauth2",
				// Flow intentionally missing
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing flow
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "securityDefinitions.oauth2") &&
			strings.Contains(e.Message, "OAuth2 security scheme must have a flow") {
			foundError = true
			assert.Equal(t, "flow", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about oauth2 missing flow")
}

func TestValidateOAS2Security_OAuth2ImplicitMissingAuthorizationUrl(t *testing.T) {
	// Test: oauth2 implicit flow without authorizationUrl should error
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		SecurityDefinitions: map[string]*parser.SecurityScheme{
			"oauth2Implicit": {
				Type: "oauth2",
				Flow: "implicit",
				// AuthorizationURL intentionally missing
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing authorizationUrl
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "securityDefinitions.oauth2Implicit") &&
			strings.Contains(e.Message, "requires authorizationUrl") {
			foundError = true
			assert.Equal(t, "authorizationUrl", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about implicit flow missing authorizationUrl")
}

func TestValidateOAS2Security_OAuth2ImplicitInvalidAuthorizationUrl(t *testing.T) {
	// Test: oauth2 implicit flow with invalid authorizationUrl should error
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		SecurityDefinitions: map[string]*parser.SecurityScheme{
			"oauth2Implicit": {
				Type:             "oauth2",
				Flow:             "implicit",
				AuthorizationURL: "not-a-valid-url",
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about invalid authorizationUrl
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "securityDefinitions.oauth2Implicit") &&
			strings.Contains(e.Message, "Invalid URL format for authorizationUrl") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about invalid authorizationUrl")
}

func TestValidateOAS2Security_OAuth2AccessCodeMissingAuthorizationUrl(t *testing.T) {
	// Test: oauth2 accessCode flow without authorizationUrl should error
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		SecurityDefinitions: map[string]*parser.SecurityScheme{
			"oauth2AccessCode": {
				Type:     "oauth2",
				Flow:     "accessCode",
				TokenURL: "https://example.com/oauth/token",
				// AuthorizationURL intentionally missing
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing authorizationUrl
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "securityDefinitions.oauth2AccessCode") &&
			strings.Contains(e.Message, "requires authorizationUrl") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about accessCode flow missing authorizationUrl")
}

func TestValidateOAS2Security_OAuth2PasswordMissingTokenUrl(t *testing.T) {
	// Test: oauth2 password flow without tokenUrl should error
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		SecurityDefinitions: map[string]*parser.SecurityScheme{
			"oauth2Password": {
				Type: "oauth2",
				Flow: "password",
				// TokenURL intentionally missing
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing tokenUrl
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "securityDefinitions.oauth2Password") &&
			strings.Contains(e.Message, "requires tokenUrl") {
			foundError = true
			assert.Equal(t, "tokenUrl", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about password flow missing tokenUrl")
}

func TestValidateOAS2Security_OAuth2PasswordInvalidTokenUrl(t *testing.T) {
	// Test: oauth2 password flow with invalid tokenUrl should error
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		SecurityDefinitions: map[string]*parser.SecurityScheme{
			"oauth2Password": {
				Type:     "oauth2",
				Flow:     "password",
				TokenURL: "not-a-valid-url",
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about invalid tokenUrl
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "securityDefinitions.oauth2Password") &&
			strings.Contains(e.Message, "Invalid URL format for tokenUrl") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about invalid tokenUrl")
}

func TestValidateOAS2Security_OAuth2ApplicationMissingTokenUrl(t *testing.T) {
	// Test: oauth2 application flow without tokenUrl should error
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		SecurityDefinitions: map[string]*parser.SecurityScheme{
			"oauth2Application": {
				Type: "oauth2",
				Flow: "application",
				// TokenURL intentionally missing
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing tokenUrl
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "securityDefinitions.oauth2Application") &&
			strings.Contains(e.Message, "requires tokenUrl") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about application flow missing tokenUrl")
}

func TestValidateOAS2Security_OAuth2AccessCodeMissingTokenUrl(t *testing.T) {
	// Test: oauth2 accessCode flow without tokenUrl should error
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		SecurityDefinitions: map[string]*parser.SecurityScheme{
			"oauth2AccessCode": {
				Type:             "oauth2",
				Flow:             "accessCode",
				AuthorizationURL: "https://example.com/oauth/authorize",
				// TokenURL intentionally missing
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing tokenUrl
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "securityDefinitions.oauth2AccessCode") &&
			strings.Contains(e.Message, "requires tokenUrl") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about accessCode flow missing tokenUrl")
}

func TestValidateOAS2Security_ValidSecuritySchemes(t *testing.T) {
	// Test: valid security schemes should not error
	doc := &parser.OAS2Document{
		Swagger: "2.0",
		Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
		Paths:   make(map[string]*parser.PathItem),
		Security: []parser.SecurityRequirement{
			{"apiKeyAuth": []string{}},
			{"oauth2Implicit": []string{"read", "write"}},
		},
		SecurityDefinitions: map[string]*parser.SecurityScheme{
			"apiKeyAuth": {
				Type: "apiKey",
				Name: "X-API-Key",
				In:   "header",
			},
			"basicAuth": {
				Type: "basic",
			},
			"oauth2Implicit": {
				Type:             "oauth2",
				Flow:             "implicit",
				AuthorizationURL: "https://example.com/oauth/authorize",
			},
			"oauth2Password": {
				Type:     "oauth2",
				Flow:     "password",
				TokenURL: "https://example.com/oauth/token",
			},
			"oauth2Application": {
				Type:     "oauth2",
				Flow:     "application",
				TokenURL: "https://example.com/oauth/token",
			},
			"oauth2AccessCode": {
				Type:             "oauth2",
				Flow:             "accessCode",
				AuthorizationURL: "https://example.com/oauth/authorize",
				TokenURL:         "https://example.com/oauth/token",
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "2.0",
		OASVersion: parser.OASVersion20,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Should not have errors about security definitions
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "securityDefinitions.") {
			t.Errorf("Unexpected security error: %s at %s", e.Message, e.Path)
		}
	}
}

// =============================================================================
// Additional OAS2 Edge Cases
// =============================================================================

func TestValidateOAS2Parameters_AllLocations(t *testing.T) {
	// Test all parameter locations (query, header, path, formData)
	tests := []struct {
		name     string
		in       string
		hasType  bool
		wantErr  bool
		errField string
	}{
		{"query param with type", "query", true, false, ""},
		{"query param without type", "query", false, true, "type"},
		{"header param with type", "header", true, false, ""},
		{"header param without type", "header", false, true, "type"},
		{"path param with type", "path", true, false, ""},
		{"path param without type", "path", false, true, "type"},
		{"formData param with type", "formData", true, false, ""},
		{"formData param without type", "formData", false, true, "type"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			param := &parser.Parameter{
				Name: "testParam",
				In:   tt.in,
			}
			if tt.hasType {
				param.Type = "string"
			}

			doc := &parser.OAS2Document{
				Swagger: "2.0",
				Info:    &parser.Info{Title: "Test API", Version: "1.0.0"},
				Paths:   make(map[string]*parser.PathItem),
				Parameters: map[string]*parser.Parameter{
					"testParam": param,
				},
			}

			parseResult := parser.ParseResult{
				Version:    "2.0",
				OASVersion: parser.OASVersion20,
				Document:   doc,
			}

			v := New()
			result, err := v.ValidateParsed(parseResult)
			require.NoError(t, err)

			// Check if expected error is present
			var foundError bool
			for _, e := range result.Errors {
				if strings.Contains(e.Path, "parameters.testParam") &&
					strings.Contains(e.Message, "Non-body parameter must have a type") {
					foundError = true
					break
				}
			}
			assert.Equal(t, tt.wantErr, foundError, "expected error=%v, got error=%v", tt.wantErr, foundError)
		})
	}
}
