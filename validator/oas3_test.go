package validator

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// validateOAS3Servers Tests
// =============================================================================

func TestValidateOAS3Servers_MissingURL(t *testing.T) {
	// Test: server without URL should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Servers: []*parser.Server{
			{
				// URL intentionally missing
				Description: "Test server",
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about server missing URL
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "servers[0]") &&
			strings.Contains(e.Message, "Server must have a url") {
			foundError = true
			assert.Equal(t, "url", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about server missing URL")
}

func TestValidateOAS3Servers_VariableMissingDefault(t *testing.T) {
	// Test: server variable without default should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Servers: []*parser.Server{
			{
				URL: "https://{host}.example.com",
				Variables: map[string]parser.ServerVariable{
					"host": {
						// Default intentionally missing
						Description: "Host name",
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about server variable missing default
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "servers[0].variables.host") &&
			strings.Contains(e.Message, "Server variable must have a default value") {
			foundError = true
			assert.Equal(t, "default", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about server variable missing default")
}

func TestValidateOAS3Servers_VariableDefaultNotInEnum(t *testing.T) {
	// Test: server variable with default not in enum should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Servers: []*parser.Server{
			{
				URL: "https://{env}.example.com",
				Variables: map[string]parser.ServerVariable{
					"env": {
						Default: "staging", // Not in enum
						Enum:    []string{"dev", "prod"},
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about default not in enum
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "servers[0].variables.env") &&
			strings.Contains(e.Message, "must be one of the enum values") {
			foundError = true
			assert.Equal(t, "default", e.Field)
			assert.Equal(t, "staging", e.Value)
			break
		}
	}
	assert.True(t, foundError, "Should have error about default value not in enum")
}

func TestValidateOAS3Servers_ValidServer(t *testing.T) {
	// Test: valid server should not error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Servers: []*parser.Server{
			{
				URL: "https://{env}.example.com",
				Variables: map[string]parser.ServerVariable{
					"env": {
						Default: "prod",
						Enum:    []string{"dev", "staging", "prod"},
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Should not have errors about servers
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "servers[") {
			t.Errorf("Unexpected server error: %s at %s", e.Message, e.Path)
		}
	}
}

func TestValidateOAS3Servers_MultipleServers(t *testing.T) {
	// Test: multiple servers with various issues
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Servers: []*parser.Server{
			{URL: "https://api.example.com"}, // Valid
			{Description: "Missing URL"},     // Invalid - missing URL
			{
				URL: "https://{port}.example.com",
				Variables: map[string]parser.ServerVariable{
					"port": {}, // Invalid - missing default
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Should have error about servers[1] missing URL
	var foundURLError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "servers[1]") &&
			strings.Contains(e.Message, "Server must have a url") {
			foundURLError = true
			break
		}
	}
	assert.True(t, foundURLError, "Should have error about servers[1] missing URL")

	// Should have error about servers[2].variables.port missing default
	var foundDefaultError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "servers[2].variables.port") &&
			strings.Contains(e.Message, "Server variable must have a default value") {
			foundDefaultError = true
			break
		}
	}
	assert.True(t, foundDefaultError, "Should have error about servers[2].variables.port missing default")
}

// =============================================================================
// validateOAS3SecurityScheme Tests
// =============================================================================

func TestValidateOAS3SecurityScheme_MissingType(t *testing.T) {
	// Test: security scheme without type should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"noType": {
					// Type intentionally missing
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing type
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "components.securitySchemes.noType") &&
			strings.Contains(e.Message, "Security scheme must have a type") {
			foundError = true
			assert.Equal(t, "type", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about security scheme missing type")
}

func TestValidateOAS3SecurityScheme_ApiKeyMissingName(t *testing.T) {
	// Test: apiKey security scheme without name should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"apiKey": {
					Type: "apiKey",
					In:   "header",
					// Name intentionally missing
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing name
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "components.securitySchemes.apiKey") &&
			strings.Contains(e.Message, "API key security scheme must have a name") {
			foundError = true
			assert.Equal(t, "name", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about apiKey missing name")
}

func TestValidateOAS3SecurityScheme_ApiKeyMissingIn(t *testing.T) {
	// Test: apiKey security scheme without 'in' should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"apiKey": {
					Type: "apiKey",
					Name: "X-API-Key",
					// In intentionally missing
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing 'in'
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "components.securitySchemes.apiKey") &&
			strings.Contains(e.Message, "API key security scheme must specify 'in'") {
			foundError = true
			assert.Equal(t, "in", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about apiKey missing 'in'")
}

func TestValidateOAS3SecurityScheme_HttpMissingScheme(t *testing.T) {
	// Test: http security scheme without scheme should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"httpAuth": {
					Type: "http",
					// Scheme intentionally missing
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing scheme
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "components.securitySchemes.httpAuth") &&
			strings.Contains(e.Message, "HTTP security scheme must have a scheme") {
			foundError = true
			assert.Equal(t, "scheme", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about http missing scheme")
}

func TestValidateOAS3SecurityScheme_OAuth2MissingFlows(t *testing.T) {
	// Test: oauth2 security scheme without flows should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"oauth2": {
					Type: "oauth2",
					// Flows intentionally missing
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing flows
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "components.securitySchemes.oauth2") &&
			strings.Contains(e.Message, "OAuth2 security scheme must have flows") {
			foundError = true
			assert.Equal(t, "flows", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about oauth2 missing flows")
}

func TestValidateOAS3SecurityScheme_OpenIdConnectMissingUrl(t *testing.T) {
	// Test: openIdConnect security scheme without openIdConnectUrl should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"oidc": {
					Type: "openIdConnect",
					// OpenIDConnectURL intentionally missing
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing openIdConnectUrl
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "components.securitySchemes.oidc") &&
			strings.Contains(e.Message, "OpenID Connect security scheme must have openIdConnectUrl") {
			foundError = true
			assert.Equal(t, "openIdConnectUrl", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about openIdConnect missing url")
}

func TestValidateOAS3SecurityScheme_ValidSecuritySchemes(t *testing.T) {
	// Test: valid security schemes should not error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"apiKeyAuth": {
					Type: "apiKey",
					Name: "X-API-Key",
					In:   "header",
				},
				"bearerAuth": {
					Type:   "http",
					Scheme: "bearer",
				},
				"basicAuth": {
					Type:   "http",
					Scheme: "basic",
				},
				"oidc": {
					Type:             "openIdConnect",
					OpenIDConnectURL: "https://example.com/.well-known/openid-configuration",
				},
				"oauth2": {
					Type: "oauth2",
					Flows: &parser.OAuthFlows{
						AuthorizationCode: &parser.OAuthFlow{
							AuthorizationURL: "https://example.com/oauth/authorize",
							TokenURL:         "https://example.com/oauth/token",
							Scopes:           map[string]string{"read": "Read access"},
						},
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Should not have errors about security schemes
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "components.securitySchemes.") {
			t.Errorf("Unexpected security scheme error: %s at %s", e.Message, e.Path)
		}
	}
}

func TestValidateOAS3SecurityScheme_NilScheme(t *testing.T) {
	// Test: nil security scheme should be skipped
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"nilScheme": nil,
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Should not panic and should not error on nil scheme
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "components.securitySchemes.nilScheme") {
			t.Errorf("Unexpected error for nil security scheme: %s", e.Message)
		}
	}
}

// =============================================================================
// validateOAuth2Flows Tests
// =============================================================================

func TestValidateOAuth2Flows_ImplicitMissingAuthorizationUrl(t *testing.T) {
	// Test: implicit flow without authorizationUrl should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"oauth2": {
					Type: "oauth2",
					Flows: &parser.OAuthFlows{
						Implicit: &parser.OAuthFlow{
							// AuthorizationURL intentionally missing
							Scopes: map[string]string{},
						},
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing authorizationUrl
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "flows.implicit") &&
			strings.Contains(e.Message, "Implicit flow must have authorizationUrl") {
			foundError = true
			assert.Equal(t, "authorizationUrl", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about implicit flow missing authorizationUrl")
}

func TestValidateOAuth2Flows_ImplicitInvalidAuthorizationUrl(t *testing.T) {
	// Test: implicit flow with invalid authorizationUrl should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"oauth2": {
					Type: "oauth2",
					Flows: &parser.OAuthFlows{
						Implicit: &parser.OAuthFlow{
							AuthorizationURL: "not-a-valid-url",
							Scopes:           map[string]string{},
						},
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about invalid authorizationUrl
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "flows.implicit") &&
			strings.Contains(e.Message, "Invalid URL format for authorizationUrl") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about invalid authorizationUrl in implicit flow")
}

func TestValidateOAuth2Flows_PasswordMissingTokenUrl(t *testing.T) {
	// Test: password flow without tokenUrl should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"oauth2": {
					Type: "oauth2",
					Flows: &parser.OAuthFlows{
						Password: &parser.OAuthFlow{
							// TokenURL intentionally missing
							Scopes: map[string]string{},
						},
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing tokenUrl
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "flows.password") &&
			strings.Contains(e.Message, "Password flow must have tokenUrl") {
			foundError = true
			assert.Equal(t, "tokenUrl", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about password flow missing tokenUrl")
}

func TestValidateOAuth2Flows_PasswordInvalidTokenUrl(t *testing.T) {
	// Test: password flow with invalid tokenUrl should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"oauth2": {
					Type: "oauth2",
					Flows: &parser.OAuthFlows{
						Password: &parser.OAuthFlow{
							TokenURL: "not-a-valid-url",
							Scopes:   map[string]string{},
						},
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about invalid tokenUrl
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "flows.password") &&
			strings.Contains(e.Message, "Invalid URL format for tokenUrl") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about invalid tokenUrl in password flow")
}

func TestValidateOAuth2Flows_ClientCredentialsMissingTokenUrl(t *testing.T) {
	// Test: clientCredentials flow without tokenUrl should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"oauth2": {
					Type: "oauth2",
					Flows: &parser.OAuthFlows{
						ClientCredentials: &parser.OAuthFlow{
							// TokenURL intentionally missing
							Scopes: map[string]string{},
						},
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing tokenUrl
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "flows.clientCredentials") &&
			strings.Contains(e.Message, "Client credentials flow must have tokenUrl") {
			foundError = true
			assert.Equal(t, "tokenUrl", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about clientCredentials flow missing tokenUrl")
}

func TestValidateOAuth2Flows_ClientCredentialsInvalidTokenUrl(t *testing.T) {
	// Test: clientCredentials flow with invalid tokenUrl should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"oauth2": {
					Type: "oauth2",
					Flows: &parser.OAuthFlows{
						ClientCredentials: &parser.OAuthFlow{
							TokenURL: "not-a-valid-url",
							Scopes:   map[string]string{},
						},
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about invalid tokenUrl
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "flows.clientCredentials") &&
			strings.Contains(e.Message, "Invalid URL format for tokenUrl") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about invalid tokenUrl in clientCredentials flow")
}

func TestValidateOAuth2Flows_AuthorizationCodeMissingAuthorizationUrl(t *testing.T) {
	// Test: authorizationCode flow without authorizationUrl should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"oauth2": {
					Type: "oauth2",
					Flows: &parser.OAuthFlows{
						AuthorizationCode: &parser.OAuthFlow{
							TokenURL: "https://example.com/oauth/token",
							// AuthorizationURL intentionally missing
							Scopes: map[string]string{},
						},
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing authorizationUrl
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "flows.authorizationCode") &&
			strings.Contains(e.Message, "Authorization code flow must have authorizationUrl") {
			foundError = true
			assert.Equal(t, "authorizationUrl", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about authorizationCode flow missing authorizationUrl")
}

func TestValidateOAuth2Flows_AuthorizationCodeInvalidAuthorizationUrl(t *testing.T) {
	// Test: authorizationCode flow with invalid authorizationUrl should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"oauth2": {
					Type: "oauth2",
					Flows: &parser.OAuthFlows{
						AuthorizationCode: &parser.OAuthFlow{
							AuthorizationURL: "not-a-valid-url",
							TokenURL:         "https://example.com/oauth/token",
							Scopes:           map[string]string{},
						},
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about invalid authorizationUrl
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "flows.authorizationCode") &&
			strings.Contains(e.Message, "Invalid URL format for authorizationUrl") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about invalid authorizationUrl in authorizationCode flow")
}

func TestValidateOAuth2Flows_AuthorizationCodeMissingTokenUrl(t *testing.T) {
	// Test: authorizationCode flow without tokenUrl should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"oauth2": {
					Type: "oauth2",
					Flows: &parser.OAuthFlows{
						AuthorizationCode: &parser.OAuthFlow{
							AuthorizationURL: "https://example.com/oauth/authorize",
							// TokenURL intentionally missing
							Scopes: map[string]string{},
						},
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about missing tokenUrl
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "flows.authorizationCode") &&
			strings.Contains(e.Message, "Authorization code flow must have tokenUrl") {
			foundError = true
			assert.Equal(t, "tokenUrl", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about authorizationCode flow missing tokenUrl")
}

func TestValidateOAuth2Flows_AuthorizationCodeInvalidTokenUrl(t *testing.T) {
	// Test: authorizationCode flow with invalid tokenUrl should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"oauth2": {
					Type: "oauth2",
					Flows: &parser.OAuthFlows{
						AuthorizationCode: &parser.OAuthFlow{
							AuthorizationURL: "https://example.com/oauth/authorize",
							TokenURL:         "not-a-valid-url",
							Scopes:           map[string]string{},
						},
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about invalid tokenUrl
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "flows.authorizationCode") &&
			strings.Contains(e.Message, "Invalid URL format for tokenUrl") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about invalid tokenUrl in authorizationCode flow")
}

func TestValidateOAuth2Flows_AllFlowsValid(t *testing.T) {
	// Test: all valid flows should not error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			SecuritySchemes: map[string]*parser.SecurityScheme{
				"oauth2": {
					Type: "oauth2",
					Flows: &parser.OAuthFlows{
						Implicit: &parser.OAuthFlow{
							AuthorizationURL: "https://example.com/oauth/authorize",
							Scopes:           map[string]string{"read": "Read access"},
						},
						Password: &parser.OAuthFlow{
							TokenURL: "https://example.com/oauth/token",
							Scopes:   map[string]string{"read": "Read access"},
						},
						ClientCredentials: &parser.OAuthFlow{
							TokenURL: "https://example.com/oauth/token",
							Scopes:   map[string]string{"read": "Read access"},
						},
						AuthorizationCode: &parser.OAuthFlow{
							AuthorizationURL: "https://example.com/oauth/authorize",
							TokenURL:         "https://example.com/oauth/token",
							Scopes:           map[string]string{"read": "Read access"},
						},
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Should not have errors about oauth2 flows
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "flows.") {
			t.Errorf("Unexpected OAuth2 flow error: %s at %s", e.Message, e.Path)
		}
	}
}

// =============================================================================
// Additional OAS3 Components Tests
// =============================================================================

func TestValidateOAS3Components_ResponseMissingDescription(t *testing.T) {
	// Test: component response without description should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			Responses: map[string]*parser.Response{
				"NotFound": {
					// Description intentionally missing
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about response missing description
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "components.responses.NotFound") &&
			strings.Contains(e.Message, "Response must have a description") {
			foundError = true
			assert.Equal(t, "description", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about response missing description")
}

func TestValidateOAS3Components_ParameterMissingSchemaAndContent(t *testing.T) {
	// Test: component parameter without schema or content should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			Parameters: map[string]*parser.Parameter{
				"QueryParam": {
					Name: "filter",
					In:   "query",
					// Schema and Content intentionally missing
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about parameter missing schema or content
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "components.parameters.QueryParam") &&
			strings.Contains(e.Message, "Parameter must have either a schema or content") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about parameter missing schema or content")
}

func TestValidateOAS3Components_ParameterBothSchemaAndContent(t *testing.T) {
	// Test: component parameter with both schema and content should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			Parameters: map[string]*parser.Parameter{
				"QueryParam": {
					Name:   "filter",
					In:     "query",
					Schema: &parser.Schema{Type: "string"},
					Content: map[string]*parser.MediaType{
						"application/json": {},
					},
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about parameter having both schema and content
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "components.parameters.QueryParam") &&
			strings.Contains(e.Message, "Parameter must not have both schema and content") {
			foundError = true
			break
		}
	}
	assert.True(t, foundError, "Should have error about parameter having both schema and content")
}

func TestValidateOAS3Components_PathParameterNotRequired(t *testing.T) {
	// Test: component path parameter without required: true should error
	doc := &parser.OAS3Document{
		OpenAPI:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
		Components: &parser.Components{
			Parameters: map[string]*parser.Parameter{
				"PathId": {
					Name:     "id",
					In:       "path",
					Schema:   &parser.Schema{Type: "string"},
					Required: false, // Should be true for path params
				},
			},
		},
	}

	parseResult := parser.ParseResult{
		Version:    "3.0.3",
		OASVersion: parser.OASVersion303,
		Document:   doc,
	}

	v := New()
	result, err := v.ValidateParsed(parseResult)
	require.NoError(t, err)

	// Find error about path parameter not having required: true
	var foundError bool
	for _, e := range result.Errors {
		if strings.Contains(e.Path, "components.parameters.PathId") &&
			strings.Contains(e.Message, "Path parameters must have required: true") {
			foundError = true
			assert.Equal(t, "required", e.Field)
			break
		}
	}
	assert.True(t, foundError, "Should have error about path parameter not having required: true")
}

// =============================================================================
// RequestBody Media Type Validation Tests
// =============================================================================

func TestValidateOAS3RequestBody_InvalidMediaType(t *testing.T) {
	tests := []struct {
		name      string
		mediaType string
		wantError bool
	}{
		{
			name:      "invalid media type with leading question mark",
			mediaType: "?invalid",
			wantError: true,
		},
		{
			name:      "invalid media type with leading slash",
			mediaType: "/json",
			wantError: true,
		},
		{
			name:      "valid media type application/json",
			mediaType: "application/json",
			wantError: false,
		},
		{
			name:      "valid media type with vendor prefix",
			mediaType: "application/vnd.api+json",
			wantError: false,
		},
		{
			name:      "valid wildcard media type",
			mediaType: "*/*",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &parser.OAS3Document{
				OpenAPI:    "3.0.3",
				OASVersion: parser.OASVersion303,
				Info:       &parser.Info{Title: "Test API", Version: "1.0.0"},
				Paths: map[string]*parser.PathItem{
					"/test": {
						Post: &parser.Operation{
							OperationID: "testOp",
							RequestBody: &parser.RequestBody{
								Content: map[string]*parser.MediaType{
									tt.mediaType: {
										Schema: &parser.Schema{Type: "object"},
									},
								},
							},
							Responses: &parser.Responses{
								Codes: map[string]*parser.Response{
									"200": {Description: "OK"},
								},
							},
						},
					},
				},
			}

			parseResult := parser.ParseResult{
				Version:    "3.0.3",
				OASVersion: parser.OASVersion303,
				Document:   doc,
			}

			v := New()
			result, err := v.ValidateParsed(parseResult)
			require.NoError(t, err)

			// Look for "Invalid media type" error on the requestBody content path
			var foundMediaTypeError bool
			for _, e := range result.Errors {
				if strings.Contains(e.Path, "requestBody.content.") &&
					strings.Contains(e.Message, "Invalid media type") {
					foundMediaTypeError = true
					break
				}
			}

			if tt.wantError {
				assert.True(t, foundMediaTypeError,
					"Expected 'Invalid media type' error for media type %q", tt.mediaType)
			} else {
				assert.False(t, foundMediaTypeError,
					"Did not expect 'Invalid media type' error for media type %q", tt.mediaType)
			}
		})
	}
}
