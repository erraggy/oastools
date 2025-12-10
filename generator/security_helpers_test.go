package generator

import (
	"strings"
	"testing"

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
			if got != tt.want {
				t.Errorf("sanitizeSecurityFunctionName(%q) = %q, want %q", tt.input, got, tt.want)
			}
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
	if !strings.Contains(result, "func WithApiKeyAPIKey(key string) ClientOption") {
		t.Error("expected WithApiKeyAPIKey function")
	}

	// Verify header setting
	if !strings.Contains(result, `req.Header.Set("X-API-Key", key)`) {
		t.Error("expected header set code")
	}
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
	if !strings.Contains(result, "func WithApiKeyAPIKeyQuery(key string) ClientOption") {
		t.Error("expected WithApiKeyAPIKeyQuery function")
	}

	// Verify query parameter setting
	if !strings.Contains(result, `q.Set("api_key", key)`) {
		t.Error("expected query set code")
	}
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
	if !strings.Contains(result, "func WithSessionAPIKeyCookie(key string) ClientOption") {
		t.Error("expected WithSessionAPIKeyCookie function")
	}

	// Verify cookie setting
	if !strings.Contains(result, `req.AddCookie(&http.Cookie{Name: "session_id", Value: key})`) {
		t.Error("expected cookie add code")
	}
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
	if !strings.Contains(result, "func WithBasicBasicAuth(username, password string) ClientOption") {
		t.Error("expected WithBasicBasicAuth function")
	}

	// Verify SetBasicAuth call
	if !strings.Contains(result, "req.SetBasicAuth(username, password)") {
		t.Error("expected SetBasicAuth code")
	}
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
	if !strings.Contains(result, "func WithBearerAuthBearerToken(token string) ClientOption") {
		t.Error("expected WithBearerAuthBearerToken function")
	}

	// Verify bearer format comment
	if !strings.Contains(result, "Bearer format: JWT") {
		t.Error("expected bearer format comment")
	}

	// Verify Authorization header
	if !strings.Contains(result, `req.Header.Set("Authorization", "Bearer "+token)`) {
		t.Error("expected Authorization header code")
	}
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
	if !strings.Contains(result, "func WithOauth2OAuth2Token(token string) ClientOption") {
		t.Error("expected WithOauth2OAuth2Token function")
	}

	// Verify scopes documentation
	if !strings.Contains(result, "Available scopes:") {
		t.Error("expected scopes documentation")
	}

	// Verify Authorization header
	if !strings.Contains(result, `req.Header.Set("Authorization", "Bearer "+token)`) {
		t.Error("expected Authorization header code")
	}
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
	if !strings.Contains(result, "func WithOidcToken(token string) ClientOption") {
		t.Error("expected WithOidcToken function")
	}

	// Verify discovery URL comment
	if !strings.Contains(result, "OpenID Connect Discovery URL:") {
		t.Error("expected discovery URL comment")
	}
}

func TestSecurityHelperGenerator_EmptySchemes(t *testing.T) {
	g := NewSecurityHelperGenerator("api")

	result := g.GenerateSecurityHelpers(nil)
	if result != "" {
		t.Errorf("expected empty result for nil schemes, got %q", result)
	}

	result = g.GenerateSecurityHelpers(map[string]*parser.SecurityScheme{})
	if result != "" {
		t.Errorf("expected empty result for empty schemes, got %q", result)
	}
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
	if !strings.Contains(result, "WithApiKeyAPIKey") {
		t.Error("expected WithApiKeyAPIKey function")
	}
	if !strings.Contains(result, "WithBearerBearerToken") {
		t.Error("expected WithBearerBearerToken function")
	}
}

func TestSecurityHelperGenerator_GenerateSecurityImports(t *testing.T) {
	g := NewSecurityHelperGenerator("api")
	imports := g.GenerateSecurityImports()

	if len(imports) != 2 {
		t.Errorf("expected 2 imports, got %d", len(imports))
	}

	hasContext := false
	hasHTTP := false
	for _, imp := range imports {
		if imp == "context" {
			hasContext = true
		}
		if imp == "net/http" {
			hasHTTP = true
		}
	}

	if !hasContext {
		t.Error("expected context import")
	}
	if !hasHTTP {
		t.Error("expected net/http import")
	}
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

	if len(info) != 2 {
		t.Errorf("expected 2 scheme infos, got %d", len(info))
	}

	// Verify api_key info
	apiKeyInfo := info[0] // Sorted alphabetically
	if apiKeyInfo.Name != "api_key" {
		t.Errorf("expected first scheme to be api_key, got %s", apiKeyInfo.Name)
	}
	if apiKeyInfo.Type != "apiKey" {
		t.Errorf("expected type apiKey, got %s", apiKeyInfo.Type)
	}
	if apiKeyInfo.In != "header" {
		t.Errorf("expected in header, got %s", apiKeyInfo.In)
	}

	// Verify oauth2 info
	oauth2Info := info[1]
	if oauth2Info.Name != "oauth2" {
		t.Errorf("expected second scheme to be oauth2, got %s", oauth2Info.Name)
	}
	if len(oauth2Info.Scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(oauth2Info.Scopes))
	}
	if len(oauth2Info.FlowTypes) != 1 {
		t.Errorf("expected 1 flow type, got %d", len(oauth2Info.FlowTypes))
	}
}

func TestGetSecuritySchemeInfo_Empty(t *testing.T) {
	info := GetSecuritySchemeInfo(nil)
	if info != nil {
		t.Errorf("expected nil for nil schemes, got %v", info)
	}

	info = GetSecuritySchemeInfo(map[string]*parser.SecurityScheme{})
	if info != nil {
		t.Errorf("expected nil for empty schemes, got %v", info)
	}
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

	if len(scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(scopes))
	}
}

func TestGetOAuth2FlowTypes(t *testing.T) {
	// OAS 2.0 style
	scheme2 := &parser.SecurityScheme{
		Type: "oauth2",
		Flow: "implicit",
	}
	flows2 := getOAuth2FlowTypes(scheme2)
	if len(flows2) != 1 || flows2[0] != "implicit" {
		t.Errorf("expected [implicit] for OAS 2.0, got %v", flows2)
	}

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
	if len(flows3) != 2 {
		t.Errorf("expected 2 flows for OAS 3.0, got %v", flows3)
	}
}
