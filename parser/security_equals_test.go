package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// equalOAuthFlow tests
// =============================================================================

func TestEqualOAuthFlow(t *testing.T) {
	tests := []struct {
		name string
		a    *OAuthFlow
		b    *OAuthFlow
		want bool
	}{
		// Nil handling
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil, b non-nil",
			a:    nil,
			b:    &OAuthFlow{AuthorizationURL: "https://auth.example.com"},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &OAuthFlow{AuthorizationURL: "https://auth.example.com"},
			b:    nil,
			want: false,
		},
		// Empty flows
		{
			name: "both empty",
			a:    &OAuthFlow{},
			b:    &OAuthFlow{},
			want: true,
		},
		// AuthorizationURL field
		{
			name: "same AuthorizationURL",
			a:    &OAuthFlow{AuthorizationURL: "https://auth.example.com/authorize"},
			b:    &OAuthFlow{AuthorizationURL: "https://auth.example.com/authorize"},
			want: true,
		},
		{
			name: "different AuthorizationURL",
			a:    &OAuthFlow{AuthorizationURL: "https://auth.example.com/authorize"},
			b:    &OAuthFlow{AuthorizationURL: "https://auth.other.com/authorize"},
			want: false,
		},
		// TokenURL field
		{
			name: "same TokenURL",
			a:    &OAuthFlow{TokenURL: "https://auth.example.com/token"},
			b:    &OAuthFlow{TokenURL: "https://auth.example.com/token"},
			want: true,
		},
		{
			name: "different TokenURL",
			a:    &OAuthFlow{TokenURL: "https://auth.example.com/token"},
			b:    &OAuthFlow{TokenURL: "https://auth.other.com/token"},
			want: false,
		},
		// RefreshURL field
		{
			name: "same RefreshURL",
			a:    &OAuthFlow{RefreshURL: "https://auth.example.com/refresh"},
			b:    &OAuthFlow{RefreshURL: "https://auth.example.com/refresh"},
			want: true,
		},
		{
			name: "different RefreshURL",
			a:    &OAuthFlow{RefreshURL: "https://auth.example.com/refresh"},
			b:    &OAuthFlow{RefreshURL: "https://auth.other.com/refresh"},
			want: false,
		},
		// Scopes field
		{
			name: "same Scopes",
			a: &OAuthFlow{
				Scopes: map[string]string{
					"read:users":  "Read user data",
					"write:users": "Modify user data",
				},
			},
			b: &OAuthFlow{
				Scopes: map[string]string{
					"read:users":  "Read user data",
					"write:users": "Modify user data",
				},
			},
			want: true,
		},
		{
			name: "different Scopes values",
			a: &OAuthFlow{
				Scopes: map[string]string{
					"read:users": "Read user data",
				},
			},
			b: &OAuthFlow{
				Scopes: map[string]string{
					"read:users": "Read all user information",
				},
			},
			want: false,
		},
		{
			name: "different Scopes keys",
			a: &OAuthFlow{
				Scopes: map[string]string{
					"read:users": "Read user data",
				},
			},
			b: &OAuthFlow{
				Scopes: map[string]string{
					"read:accounts": "Read user data",
				},
			},
			want: false,
		},
		{
			name: "Scopes nil vs empty",
			a:    &OAuthFlow{Scopes: nil},
			b:    &OAuthFlow{Scopes: map[string]string{}},
			want: true,
		},
		// Extra field (extensions)
		{
			name: "same Extra",
			a:    &OAuthFlow{Extra: map[string]any{"x-custom": "value"}},
			b:    &OAuthFlow{Extra: map[string]any{"x-custom": "value"}},
			want: true,
		},
		{
			name: "different Extra",
			a:    &OAuthFlow{Extra: map[string]any{"x-custom": "value1"}},
			b:    &OAuthFlow{Extra: map[string]any{"x-custom": "value2"}},
			want: false,
		},
		{
			name: "Extra nil vs empty",
			a:    &OAuthFlow{Extra: nil},
			b:    &OAuthFlow{Extra: map[string]any{}},
			want: true,
		},
		// Complete OAuth flow - Authorization Code
		{
			name: "complete authorization code flow equal",
			a: &OAuthFlow{
				AuthorizationURL: "https://auth.example.com/authorize",
				TokenURL:         "https://auth.example.com/token",
				RefreshURL:       "https://auth.example.com/refresh",
				Scopes: map[string]string{
					"read:users":  "Read user data",
					"write:users": "Modify user data",
					"admin":       "Full admin access",
				},
				Extra: map[string]any{"x-token-lifetime": 3600},
			},
			b: &OAuthFlow{
				AuthorizationURL: "https://auth.example.com/authorize",
				TokenURL:         "https://auth.example.com/token",
				RefreshURL:       "https://auth.example.com/refresh",
				Scopes: map[string]string{
					"read:users":  "Read user data",
					"write:users": "Modify user data",
					"admin":       "Full admin access",
				},
				Extra: map[string]any{"x-token-lifetime": 3600},
			},
			want: true,
		},
		// Complete OAuth flow - Implicit (no TokenURL)
		{
			name: "complete implicit flow equal",
			a: &OAuthFlow{
				AuthorizationURL: "https://auth.example.com/authorize",
				Scopes: map[string]string{
					"read:users": "Read user data",
				},
			},
			b: &OAuthFlow{
				AuthorizationURL: "https://auth.example.com/authorize",
				Scopes: map[string]string{
					"read:users": "Read user data",
				},
			},
			want: true,
		},
		// Complete OAuth flow - Client Credentials (no AuthorizationURL)
		{
			name: "complete client credentials flow equal",
			a: &OAuthFlow{
				TokenURL: "https://auth.example.com/token",
				Scopes: map[string]string{
					"read:data": "Read data",
				},
			},
			b: &OAuthFlow{
				TokenURL: "https://auth.example.com/token",
				Scopes: map[string]string{
					"read:data": "Read data",
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalOAuthFlow(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalOAuthFlows tests
// =============================================================================

func TestEqualOAuthFlows(t *testing.T) {
	tests := []struct {
		name string
		a    *OAuthFlows
		b    *OAuthFlows
		want bool
	}{
		// Nil handling
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil, b non-nil",
			a:    nil,
			b:    &OAuthFlows{},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &OAuthFlows{},
			b:    nil,
			want: false,
		},
		// Empty flows
		{
			name: "both empty",
			a:    &OAuthFlows{},
			b:    &OAuthFlows{},
			want: true,
		},
		// Implicit flow
		{
			name: "same Implicit",
			a:    &OAuthFlows{Implicit: &OAuthFlow{AuthorizationURL: "https://auth.example.com"}},
			b:    &OAuthFlows{Implicit: &OAuthFlow{AuthorizationURL: "https://auth.example.com"}},
			want: true,
		},
		{
			name: "different Implicit",
			a:    &OAuthFlows{Implicit: &OAuthFlow{AuthorizationURL: "https://auth.example.com"}},
			b:    &OAuthFlows{Implicit: &OAuthFlow{AuthorizationURL: "https://auth.other.com"}},
			want: false,
		},
		{
			name: "Implicit nil vs non-nil",
			a:    &OAuthFlows{Implicit: nil},
			b:    &OAuthFlows{Implicit: &OAuthFlow{AuthorizationURL: "https://auth.example.com"}},
			want: false,
		},
		// Password flow
		{
			name: "same Password",
			a:    &OAuthFlows{Password: &OAuthFlow{TokenURL: "https://auth.example.com/token"}},
			b:    &OAuthFlows{Password: &OAuthFlow{TokenURL: "https://auth.example.com/token"}},
			want: true,
		},
		{
			name: "different Password",
			a:    &OAuthFlows{Password: &OAuthFlow{TokenURL: "https://auth.example.com/token"}},
			b:    &OAuthFlows{Password: &OAuthFlow{TokenURL: "https://auth.other.com/token"}},
			want: false,
		},
		// ClientCredentials flow
		{
			name: "same ClientCredentials",
			a:    &OAuthFlows{ClientCredentials: &OAuthFlow{TokenURL: "https://auth.example.com/token"}},
			b:    &OAuthFlows{ClientCredentials: &OAuthFlow{TokenURL: "https://auth.example.com/token"}},
			want: true,
		},
		{
			name: "different ClientCredentials",
			a:    &OAuthFlows{ClientCredentials: &OAuthFlow{TokenURL: "https://auth.example.com/token"}},
			b:    &OAuthFlows{ClientCredentials: &OAuthFlow{TokenURL: "https://auth.other.com/token"}},
			want: false,
		},
		// AuthorizationCode flow
		{
			name: "same AuthorizationCode",
			a: &OAuthFlows{
				AuthorizationCode: &OAuthFlow{
					AuthorizationURL: "https://auth.example.com/authorize",
					TokenURL:         "https://auth.example.com/token",
				},
			},
			b: &OAuthFlows{
				AuthorizationCode: &OAuthFlow{
					AuthorizationURL: "https://auth.example.com/authorize",
					TokenURL:         "https://auth.example.com/token",
				},
			},
			want: true,
		},
		{
			name: "different AuthorizationCode",
			a: &OAuthFlows{
				AuthorizationCode: &OAuthFlow{
					AuthorizationURL: "https://auth.example.com/authorize",
					TokenURL:         "https://auth.example.com/token",
				},
			},
			b: &OAuthFlows{
				AuthorizationCode: &OAuthFlow{
					AuthorizationURL: "https://auth.other.com/authorize",
					TokenURL:         "https://auth.other.com/token",
				},
			},
			want: false,
		},
		// Extra field (extensions)
		{
			name: "same Extra",
			a:    &OAuthFlows{Extra: map[string]any{"x-custom": "value"}},
			b:    &OAuthFlows{Extra: map[string]any{"x-custom": "value"}},
			want: true,
		},
		{
			name: "different Extra",
			a:    &OAuthFlows{Extra: map[string]any{"x-custom": "value1"}},
			b:    &OAuthFlows{Extra: map[string]any{"x-custom": "value2"}},
			want: false,
		},
		// Complete OAuth flows with multiple flow types
		{
			name: "complete flows with multiple types equal",
			a: &OAuthFlows{
				Implicit: &OAuthFlow{
					AuthorizationURL: "https://auth.example.com/authorize",
					Scopes:           map[string]string{"read": "Read access"},
				},
				AuthorizationCode: &OAuthFlow{
					AuthorizationURL: "https://auth.example.com/authorize",
					TokenURL:         "https://auth.example.com/token",
					Scopes:           map[string]string{"read": "Read access", "write": "Write access"},
				},
				Extra: map[string]any{"x-provider": "example"},
			},
			b: &OAuthFlows{
				Implicit: &OAuthFlow{
					AuthorizationURL: "https://auth.example.com/authorize",
					Scopes:           map[string]string{"read": "Read access"},
				},
				AuthorizationCode: &OAuthFlow{
					AuthorizationURL: "https://auth.example.com/authorize",
					TokenURL:         "https://auth.example.com/token",
					Scopes:           map[string]string{"read": "Read access", "write": "Write access"},
				},
				Extra: map[string]any{"x-provider": "example"},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalOAuthFlows(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalSecurityScheme tests
// =============================================================================

func TestEqualSecurityScheme(t *testing.T) {
	tests := []struct {
		name string
		a    *SecurityScheme
		b    *SecurityScheme
		want bool
	}{
		// Nil handling
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil, b non-nil",
			a:    nil,
			b:    &SecurityScheme{Type: "apiKey"},
			want: false,
		},
		{
			name: "a non-nil, b nil",
			a:    &SecurityScheme{Type: "apiKey"},
			b:    nil,
			want: false,
		},
		// Empty schemes
		{
			name: "both empty",
			a:    &SecurityScheme{},
			b:    &SecurityScheme{},
			want: true,
		},
		// Ref field
		{
			name: "same Ref",
			a:    &SecurityScheme{Ref: "#/components/securitySchemes/api_key"},
			b:    &SecurityScheme{Ref: "#/components/securitySchemes/api_key"},
			want: true,
		},
		{
			name: "different Ref",
			a:    &SecurityScheme{Ref: "#/components/securitySchemes/api_key"},
			b:    &SecurityScheme{Ref: "#/components/securitySchemes/oauth"},
			want: false,
		},
		// Type field
		{
			name: "same Type",
			a:    &SecurityScheme{Type: "apiKey"},
			b:    &SecurityScheme{Type: "apiKey"},
			want: true,
		},
		{
			name: "different Type",
			a:    &SecurityScheme{Type: "apiKey"},
			b:    &SecurityScheme{Type: "http"},
			want: false,
		},
		// Description field
		{
			name: "different Description",
			a:    &SecurityScheme{Type: "apiKey", Description: "API Key auth"},
			b:    &SecurityScheme{Type: "apiKey", Description: "API key authentication"},
			want: false,
		},
		// Name field (apiKey)
		{
			name: "different Name",
			a:    &SecurityScheme{Type: "apiKey", Name: "X-API-Key"},
			b:    &SecurityScheme{Type: "apiKey", Name: "Authorization"},
			want: false,
		},
		// In field (apiKey)
		{
			name: "different In",
			a:    &SecurityScheme{Type: "apiKey", Name: "api_key", In: "header"},
			b:    &SecurityScheme{Type: "apiKey", Name: "api_key", In: "query"},
			want: false,
		},
		// Scheme field (http)
		{
			name: "different Scheme",
			a:    &SecurityScheme{Type: "http", Scheme: "basic"},
			b:    &SecurityScheme{Type: "http", Scheme: "bearer"},
			want: false,
		},
		// BearerFormat field (http bearer)
		{
			name: "different BearerFormat",
			a:    &SecurityScheme{Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
			b:    &SecurityScheme{Type: "http", Scheme: "bearer", BearerFormat: "opaque"},
			want: false,
		},
		// Flow field (OAS 2.0 oauth2)
		{
			name: "different Flow (OAS 2.0)",
			a:    &SecurityScheme{Type: "oauth2", Flow: "implicit"},
			b:    &SecurityScheme{Type: "oauth2", Flow: "accessCode"},
			want: false,
		},
		// AuthorizationURL field (OAS 2.0)
		{
			name: "different AuthorizationURL (OAS 2.0)",
			a:    &SecurityScheme{Type: "oauth2", Flow: "implicit", AuthorizationURL: "https://auth.example.com"},
			b:    &SecurityScheme{Type: "oauth2", Flow: "implicit", AuthorizationURL: "https://auth.other.com"},
			want: false,
		},
		// TokenURL field (OAS 2.0)
		{
			name: "different TokenURL (OAS 2.0)",
			a:    &SecurityScheme{Type: "oauth2", Flow: "application", TokenURL: "https://auth.example.com/token"},
			b:    &SecurityScheme{Type: "oauth2", Flow: "application", TokenURL: "https://auth.other.com/token"},
			want: false,
		},
		// OpenIDConnectURL field
		{
			name: "different OpenIDConnectURL",
			a:    &SecurityScheme{Type: "openIdConnect", OpenIDConnectURL: "https://example.com/.well-known/openid"},
			b:    &SecurityScheme{Type: "openIdConnect", OpenIDConnectURL: "https://other.com/.well-known/openid"},
			want: false,
		},
		// Flows field (OAS 3.0+)
		{
			name: "same Flows",
			a: &SecurityScheme{
				Type: "oauth2",
				Flows: &OAuthFlows{
					AuthorizationCode: &OAuthFlow{
						AuthorizationURL: "https://auth.example.com/authorize",
						TokenURL:         "https://auth.example.com/token",
					},
				},
			},
			b: &SecurityScheme{
				Type: "oauth2",
				Flows: &OAuthFlows{
					AuthorizationCode: &OAuthFlow{
						AuthorizationURL: "https://auth.example.com/authorize",
						TokenURL:         "https://auth.example.com/token",
					},
				},
			},
			want: true,
		},
		{
			name: "different Flows",
			a: &SecurityScheme{
				Type: "oauth2",
				Flows: &OAuthFlows{
					AuthorizationCode: &OAuthFlow{
						AuthorizationURL: "https://auth.example.com/authorize",
						TokenURL:         "https://auth.example.com/token",
					},
				},
			},
			b: &SecurityScheme{
				Type: "oauth2",
				Flows: &OAuthFlows{
					ClientCredentials: &OAuthFlow{
						TokenURL: "https://auth.example.com/token",
					},
				},
			},
			want: false,
		},
		{
			name: "Flows nil vs non-nil",
			a:    &SecurityScheme{Type: "oauth2", Flows: nil},
			b:    &SecurityScheme{Type: "oauth2", Flows: &OAuthFlows{}},
			want: false,
		},
		// Scopes field (OAS 2.0)
		{
			name: "same Scopes (OAS 2.0)",
			a: &SecurityScheme{
				Type:   "oauth2",
				Scopes: map[string]string{"read": "Read access"},
			},
			b: &SecurityScheme{
				Type:   "oauth2",
				Scopes: map[string]string{"read": "Read access"},
			},
			want: true,
		},
		{
			name: "different Scopes (OAS 2.0)",
			a: &SecurityScheme{
				Type:   "oauth2",
				Scopes: map[string]string{"read": "Read access"},
			},
			b: &SecurityScheme{
				Type:   "oauth2",
				Scopes: map[string]string{"write": "Write access"},
			},
			want: false,
		},
		{
			name: "Scopes nil vs empty",
			a:    &SecurityScheme{Type: "oauth2", Scopes: nil},
			b:    &SecurityScheme{Type: "oauth2", Scopes: map[string]string{}},
			want: true,
		},
		// Extra field (extensions)
		{
			name: "same Extra",
			a:    &SecurityScheme{Type: "apiKey", Extra: map[string]any{"x-custom": "value"}},
			b:    &SecurityScheme{Type: "apiKey", Extra: map[string]any{"x-custom": "value"}},
			want: true,
		},
		{
			name: "different Extra",
			a:    &SecurityScheme{Type: "apiKey", Extra: map[string]any{"x-custom": "value1"}},
			b:    &SecurityScheme{Type: "apiKey", Extra: map[string]any{"x-custom": "value2"}},
			want: false,
		},
		// Complete security scheme - API Key
		{
			name: "complete apiKey scheme equal",
			a: &SecurityScheme{
				Type:        "apiKey",
				Description: "API Key authentication",
				Name:        "X-API-Key",
				In:          "header",
				Extra:       map[string]any{"x-rate-limited": true},
			},
			b: &SecurityScheme{
				Type:        "apiKey",
				Description: "API Key authentication",
				Name:        "X-API-Key",
				In:          "header",
				Extra:       map[string]any{"x-rate-limited": true},
			},
			want: true,
		},
		// Complete security scheme - HTTP Bearer
		{
			name: "complete http bearer scheme equal",
			a: &SecurityScheme{
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
				Description:  "JWT authentication",
			},
			b: &SecurityScheme{
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
				Description:  "JWT authentication",
			},
			want: true,
		},
		// Complete security scheme - OAuth2 (OAS 3.0+)
		{
			name: "complete oauth2 scheme (OAS 3.0+) equal",
			a: &SecurityScheme{
				Type:        "oauth2",
				Description: "OAuth 2.0 authentication",
				Flows: &OAuthFlows{
					AuthorizationCode: &OAuthFlow{
						AuthorizationURL: "https://auth.example.com/authorize",
						TokenURL:         "https://auth.example.com/token",
						Scopes: map[string]string{
							"read:users":  "Read user data",
							"write:users": "Modify user data",
						},
					},
				},
			},
			b: &SecurityScheme{
				Type:        "oauth2",
				Description: "OAuth 2.0 authentication",
				Flows: &OAuthFlows{
					AuthorizationCode: &OAuthFlow{
						AuthorizationURL: "https://auth.example.com/authorize",
						TokenURL:         "https://auth.example.com/token",
						Scopes: map[string]string{
							"read:users":  "Read user data",
							"write:users": "Modify user data",
						},
					},
				},
			},
			want: true,
		},
		// Complete security scheme - OpenID Connect
		{
			name: "complete openIdConnect scheme equal",
			a: &SecurityScheme{
				Type:             "openIdConnect",
				Description:      "OpenID Connect authentication",
				OpenIDConnectURL: "https://example.com/.well-known/openid-configuration",
			},
			b: &SecurityScheme{
				Type:             "openIdConnect",
				Description:      "OpenID Connect authentication",
				OpenIDConnectURL: "https://example.com/.well-known/openid-configuration",
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalSecurityScheme(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalSecuritySchemeMap tests
// =============================================================================

func TestEqualSecuritySchemeMap(t *testing.T) {
	tests := []struct {
		name string
		a    map[string]*SecurityScheme
		b    map[string]*SecurityScheme
		want bool
	}{
		// Nil and empty handling
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "both empty",
			a:    map[string]*SecurityScheme{},
			b:    map[string]*SecurityScheme{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    map[string]*SecurityScheme{},
			want: true,
		},
		// Same entries
		{
			name: "same single entry",
			a: map[string]*SecurityScheme{
				"api_key": {Type: "apiKey", Name: "X-API-Key", In: "header"},
			},
			b: map[string]*SecurityScheme{
				"api_key": {Type: "apiKey", Name: "X-API-Key", In: "header"},
			},
			want: true,
		},
		{
			name: "same multiple entries",
			a: map[string]*SecurityScheme{
				"api_key": {Type: "apiKey", Name: "X-API-Key", In: "header"},
				"bearer":  {Type: "http", Scheme: "bearer"},
			},
			b: map[string]*SecurityScheme{
				"api_key": {Type: "apiKey", Name: "X-API-Key", In: "header"},
				"bearer":  {Type: "http", Scheme: "bearer"},
			},
			want: true,
		},
		// Different entries
		{
			name: "different values same key",
			a: map[string]*SecurityScheme{
				"auth": {Type: "apiKey"},
			},
			b: map[string]*SecurityScheme{
				"auth": {Type: "http"},
			},
			want: false,
		},
		{
			name: "different keys",
			a: map[string]*SecurityScheme{
				"api_key": {Type: "apiKey"},
			},
			b: map[string]*SecurityScheme{
				"bearer": {Type: "apiKey"},
			},
			want: false,
		},
		{
			name: "a has extra key",
			a: map[string]*SecurityScheme{
				"api_key": {Type: "apiKey"},
				"bearer":  {Type: "http"},
			},
			b: map[string]*SecurityScheme{
				"api_key": {Type: "apiKey"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalSecuritySchemeMap(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalSecurityRequirement tests
// =============================================================================

func TestEqualSecurityRequirement(t *testing.T) {
	tests := []struct {
		name string
		a    SecurityRequirement
		b    SecurityRequirement
		want bool
	}{
		// Nil and empty handling
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "both empty",
			a:    SecurityRequirement{},
			b:    SecurityRequirement{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    SecurityRequirement{},
			want: true,
		},
		// Same entries
		{
			name: "same single entry with empty scopes",
			a:    SecurityRequirement{"api_key": {}},
			b:    SecurityRequirement{"api_key": {}},
			want: true,
		},
		{
			name: "same single entry with scopes",
			a:    SecurityRequirement{"oauth2": {"read:users", "write:users"}},
			b:    SecurityRequirement{"oauth2": {"read:users", "write:users"}},
			want: true,
		},
		{
			name: "same multiple entries",
			a:    SecurityRequirement{"api_key": {}, "oauth2": {"read"}},
			b:    SecurityRequirement{"api_key": {}, "oauth2": {"read"}},
			want: true,
		},
		// Different entries
		{
			name: "different scopes",
			a:    SecurityRequirement{"oauth2": {"read:users"}},
			b:    SecurityRequirement{"oauth2": {"write:users"}},
			want: false,
		},
		{
			name: "different scope order",
			a:    SecurityRequirement{"oauth2": {"read", "write"}},
			b:    SecurityRequirement{"oauth2": {"write", "read"}},
			want: false,
		},
		{
			name: "different keys",
			a:    SecurityRequirement{"api_key": {}},
			b:    SecurityRequirement{"bearer": {}},
			want: false,
		},
		{
			name: "a has extra key",
			a:    SecurityRequirement{"api_key": {}, "oauth2": {}},
			b:    SecurityRequirement{"api_key": {}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalSecurityRequirement(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}

// =============================================================================
// equalSecurityRequirementSlice tests
// =============================================================================

func TestEqualSecurityRequirementSlice(t *testing.T) {
	tests := []struct {
		name string
		a    []SecurityRequirement
		b    []SecurityRequirement
		want bool
	}{
		// Nil and empty handling
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "both empty",
			a:    []SecurityRequirement{},
			b:    []SecurityRequirement{},
			want: true,
		},
		{
			name: "nil vs empty",
			a:    nil,
			b:    []SecurityRequirement{},
			want: true,
		},
		// Same elements
		{
			name: "same single element",
			a:    []SecurityRequirement{{"api_key": {}}},
			b:    []SecurityRequirement{{"api_key": {}}},
			want: true,
		},
		{
			name: "same multiple elements",
			a: []SecurityRequirement{
				{"api_key": {}},
				{"oauth2": {"read"}},
			},
			b: []SecurityRequirement{
				{"api_key": {}},
				{"oauth2": {"read"}},
			},
			want: true,
		},
		// Different elements
		{
			name: "different elements",
			a:    []SecurityRequirement{{"api_key": {}}},
			b:    []SecurityRequirement{{"bearer": {}}},
			want: false,
		},
		{
			name: "different lengths",
			a: []SecurityRequirement{
				{"api_key": {}},
				{"oauth2": {}},
			},
			b: []SecurityRequirement{
				{"api_key": {}},
			},
			want: false,
		},
		{
			name: "different order",
			a: []SecurityRequirement{
				{"api_key": {}},
				{"oauth2": {}},
			},
			b: []SecurityRequirement{
				{"oauth2": {}},
				{"api_key": {}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalSecurityRequirementSlice(tt.a, tt.b)
			assert.Equal(t, tt.want, got)
		})
	}
}
