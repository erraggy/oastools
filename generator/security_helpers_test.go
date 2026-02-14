package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/erraggy/oastools/parser"
)

func TestSanitizeSecurityFunctionName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"api_key", "ApiKey"},
		{"API_KEY", "APIKEY"},
		{"bearer-auth", "BearerAuth"},
		{"bearerAuth", "BearerAuth"},
		{"OAuth2", "OAuth2"},
		{"oauth2", "Oauth2"},
		{"api.key", "ApiKey"},
		{"api key", "ApiKey"},
		{"123auth", "123auth"},
		{"auth123", "Auth123"},
		{"", "Default"},
		{"___", "Default"},
		{"my-api-key", "MyApiKey"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeSecurityFunctionName(tt.input)
			assert.Equal(t, tt.want, got, "sanitizeSecurityFunctionName(%q)", tt.input)
		})
	}
}

func TestSecurityHelperGenerator_APIKey_Header(t *testing.T) {
	g := NewSecurityHelperGenerator("api")

	schemes := map[string]*parser.SecurityScheme{
		"api_key": {
			Type:        "apiKey",
			Name:        "X-API-Key",
			In:          "header",
			Description: "API key in header",
		},
	}

	result := g.GenerateSecurityHelpers(schemes)

	// Verify function signature
	assert.Contains(t, result, "func WithApiKeyAPIKey(key string) ClientOption", "expected WithApiKeyAPIKey function")

	// Verify header setting
	assert.Contains(t, result, `req.Header.Set("X-API-Key", key)`, "expected header set code")
}

func TestSecurityHelperGenerator_APIKey_Query(t *testing.T) {
	g := NewSecurityHelperGenerator("api")

	schemes := map[string]*parser.SecurityScheme{
		"api_key": {
			Type: "apiKey",
			Name: "api_key",
			In:   "query",
		},
	}

	result := g.GenerateSecurityHelpers(schemes)

	// Verify function signature
	assert.Contains(t, result, "func WithApiKeyAPIKeyQuery(key string) ClientOption", "expected WithApiKeyAPIKeyQuery function")

	// Verify query parameter setting
	assert.Contains(t, result, `q.Set("api_key", key)`, "expected query set code")
}

func TestSecurityHelperGenerator_APIKey_Cookie(t *testing.T) {
	g := NewSecurityHelperGenerator("api")

	schemes := map[string]*parser.SecurityScheme{
		"session": {
			Type: "apiKey",
			Name: "session_id",
			In:   "cookie",
		},
	}

	result := g.GenerateSecurityHelpers(schemes)

	// Verify function signature
	assert.Contains(t, result, "func WithSessionAPIKeyCookie(key string) ClientOption", "expected WithSessionAPIKeyCookie function")

	// Verify cookie setting
	assert.Contains(t, result, `req.AddCookie(&http.Cookie{Name: "session_id", Value: key})`, "expected cookie add code")
}

func TestSecurityHelperGenerator_HTTP_Basic(t *testing.T) {
	g := NewSecurityHelperGenerator("api")

	schemes := map[string]*parser.SecurityScheme{
		"basic": {
			Type:   "http",
			Scheme: "basic",
		},
	}

	result := g.GenerateSecurityHelpers(schemes)

	// Verify function signature
	assert.Contains(t, result, "func WithBasicBasicAuth(username, password string) ClientOption", "expected WithBasicBasicAuth function")

	// Verify SetBasicAuth call
	assert.Contains(t, result, "req.SetBasicAuth(username, password)", "expected SetBasicAuth code")
}

func TestSecurityHelperGenerator_HTTP_Bearer(t *testing.T) {
	g := NewSecurityHelperGenerator("api")

	schemes := map[string]*parser.SecurityScheme{
		"bearer_auth": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
		},
	}

	result := g.GenerateSecurityHelpers(schemes)

	// Verify function signature
	assert.Contains(t, result, "func WithBearerAuthBearerToken(token string) ClientOption", "expected WithBearerAuthBearerToken function")

	// Verify bearer format comment
	assert.Contains(t, result, "Bearer format: JWT", "expected bearer format comment")

	// Verify Authorization header
	assert.Contains(t, result, `req.Header.Set("Authorization", "Bearer "+token)`, "expected Authorization header code")
}

func TestSecurityHelperGenerator_OAuth2(t *testing.T) {
	g := NewSecurityHelperGenerator("api")

	schemes := map[string]*parser.SecurityScheme{
		"oauth2": {
			Type: "oauth2",
			Flows: &parser.OAuthFlows{
				AuthorizationCode: &parser.OAuthFlow{
					AuthorizationURL: "https://example.com/oauth/authorize",
					TokenURL:         "https://example.com/oauth/token",
					Scopes: map[string]string{
						"read":  "Read access",
						"write": "Write access",
					},
				},
			},
		},
	}

	result := g.GenerateSecurityHelpers(schemes)

	// Verify function signature
	assert.Contains(t, result, "func WithOauth2OAuth2Token(token string) ClientOption", "expected WithOauth2OAuth2Token function")

	// Verify scopes documentation
	assert.Contains(t, result, "Available scopes:", "expected scopes documentation")

	// Verify Authorization header
	assert.Contains(t, result, `req.Header.Set("Authorization", "Bearer "+token)`, "expected Authorization header code")
}

func TestSecurityHelperGenerator_OpenIDConnect(t *testing.T) {
	g := NewSecurityHelperGenerator("api")

	schemes := map[string]*parser.SecurityScheme{
		"oidc": {
			Type:             "openIdConnect",
			OpenIDConnectURL: "https://example.com/.well-known/openid-configuration",
		},
	}

	result := g.GenerateSecurityHelpers(schemes)

	// Verify function signature
	assert.Contains(t, result, "func WithOidcToken(token string) ClientOption", "expected WithOidcToken function")

	// Verify discovery URL comment
	assert.Contains(t, result, "OpenID Connect Discovery URL:", "expected discovery URL comment")
}

func TestSecurityHelperGenerator_EmptySchemes(t *testing.T) {
	g := NewSecurityHelperGenerator("api")

	result := g.GenerateSecurityHelpers(nil)
	assert.Equal(t, "", result, "expected empty result for nil schemes")

	result = g.GenerateSecurityHelpers(map[string]*parser.SecurityScheme{})
	assert.Equal(t, "", result, "expected empty result for empty schemes")
}

func TestSecurityHelperGenerator_MultipleSchemes(t *testing.T) {
	g := NewSecurityHelperGenerator("api")

	schemes := map[string]*parser.SecurityScheme{
		"api_key": {
			Type: "apiKey",
			Name: "X-API-Key",
			In:   "header",
		},
		"bearer": {
			Type:   "http",
			Scheme: "bearer",
		},
	}

	result := g.GenerateSecurityHelpers(schemes)

	// Both functions should be present
	assert.Contains(t, result, "WithApiKeyAPIKey", "expected WithApiKeyAPIKey function")
	assert.Contains(t, result, "WithBearerBearerToken", "expected WithBearerBearerToken function")
}

func TestSecurityHelperGenerator_GenerateSecurityImports(t *testing.T) {
	g := NewSecurityHelperGenerator("api")
	imports := g.GenerateSecurityImports()

	require.Len(t, imports, 2, "expected 2 imports")

	assert.Contains(t, imports, "context", "expected context import")
	assert.Contains(t, imports, "net/http", "expected net/http import")
}

func TestGetSecuritySchemeInfo(t *testing.T) {
	schemes := map[string]*parser.SecurityScheme{
		"api_key": {
			Type:        "apiKey",
			Name:        "X-API-Key",
			In:          "header",
			Description: "API key auth",
		},
		"oauth2": {
			Type: "oauth2",
			Flows: &parser.OAuthFlows{
				AuthorizationCode: &parser.OAuthFlow{
					AuthorizationURL: "https://example.com/oauth/authorize",
					TokenURL:         "https://example.com/oauth/token",
					Scopes: map[string]string{
						"read":  "Read access",
						"write": "Write access",
					},
				},
			},
		},
	}

	info := GetSecuritySchemeInfo(schemes)

	require.Len(t, info, 2, "expected 2 scheme infos")

	// Verify api_key info
	apiKeyInfo := info[0] // Sorted alphabetically
	assert.Equal(t, "api_key", apiKeyInfo.Name, "expected first scheme to be api_key")
	assert.Equal(t, "apiKey", apiKeyInfo.Type, "expected type apiKey")
	assert.Equal(t, "header", apiKeyInfo.In, "expected in header")

	// Verify oauth2 info
	oauth2Info := info[1]
	assert.Equal(t, "oauth2", oauth2Info.Name, "expected second scheme to be oauth2")
	assert.Len(t, oauth2Info.Scopes, 2, "expected 2 scopes")
	assert.Len(t, oauth2Info.FlowTypes, 1, "expected 1 flow type")
}

func TestGetSecuritySchemeInfo_Empty(t *testing.T) {
	info := GetSecuritySchemeInfo(nil)
	assert.Nil(t, info, "expected nil for nil schemes")

	info = GetSecuritySchemeInfo(map[string]*parser.SecurityScheme{})
	assert.Nil(t, info, "expected nil for empty schemes")
}

func TestCollectOAuth2Scopes_OAS2Style(t *testing.T) {
	g := NewSecurityHelperGenerator("api")

	scheme := &parser.SecurityScheme{
		Type: "oauth2",
		Flow: "implicit",
		Scopes: map[string]string{
			"read":  "Read access",
			"write": "Write access",
		},
	}

	scopes := g.collectOAuth2Scopes(scheme)

	assert.Len(t, scopes, 2, "expected 2 scopes")
}

func TestGetOAuth2FlowTypes(t *testing.T) {
	// OAS 2.0 style
	scheme2 := &parser.SecurityScheme{
		Type: "oauth2",
		Flow: "implicit",
	}
	flows2 := getOAuth2FlowTypes(scheme2)
	require.Len(t, flows2, 1, "expected 1 flow for OAS 2.0")
	assert.Equal(t, "implicit", flows2[0], "expected implicit for OAS 2.0")

	// OAS 3.0 style
	scheme3 := &parser.SecurityScheme{
		Type: "oauth2",
		Flows: &parser.OAuthFlows{
			AuthorizationCode: &parser.OAuthFlow{
				AuthorizationURL: "https://example.com/oauth/authorize",
				TokenURL:         "https://example.com/oauth/token",
			},
			ClientCredentials: &parser.OAuthFlow{
				TokenURL: "https://example.com/oauth/token",
			},
		},
	}
	flows3 := getOAuth2FlowTypes(scheme3)
	assert.Len(t, flows3, 2, "expected 2 flows for OAS 3.0")
}
