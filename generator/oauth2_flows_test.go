package generator

import (
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOAuth2Generator(t *testing.T) {
	// Test with nil scheme
	g := NewOAuth2Generator("test", nil)
	assert.Nil(t, g)

	// Test with non-oauth2 scheme
	g = NewOAuth2Generator("test", &parser.SecurityScheme{Type: "apiKey"})
	assert.Nil(t, g)

	// Test with valid oauth2 scheme
	scheme := &parser.SecurityScheme{
		Type: "oauth2",
		Flows: &parser.OAuthFlows{
			AuthorizationCode: &parser.OAuthFlow{
				AuthorizationURL: "https://example.com/authorize",
				TokenURL:         "https://example.com/token",
			},
		},
	}
	g = NewOAuth2Generator("test", scheme)
	require.NotNil(t, g)
	assert.Equal(t, "test", g.Name)
}

func TestOAuth2Generator_HasFlows(t *testing.T) {
	// OAS 3.0+ style
	scheme := &parser.SecurityScheme{
		Type: "oauth2",
		Flows: &parser.OAuthFlows{
			AuthorizationCode: &parser.OAuthFlow{
				AuthorizationURL: "https://example.com/authorize",
				TokenURL:         "https://example.com/token",
			},
		},
	}
	g := NewOAuth2Generator("test", scheme)

	assert.True(t, g.hasAuthorizationCodeFlow())
	assert.False(t, g.hasClientCredentialsFlow())
	assert.False(t, g.hasPasswordFlow())
	assert.False(t, g.hasImplicitFlow())
}

func TestOAuth2Generator_HasFlows_OAS2(t *testing.T) {
	tests := []struct {
		flow            string
		wantAuthCode    bool
		wantClientCreds bool
		wantPassword    bool
		wantImplicit    bool
	}{
		{"accessCode", true, false, false, false},
		{"authorizationCode", true, false, false, false},
		{"application", false, true, false, false},
		{"clientCredentials", false, true, false, false},
		{"password", false, false, true, false},
		{"implicit", false, false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.flow, func(t *testing.T) {
			scheme := &parser.SecurityScheme{
				Type: "oauth2",
				Flow: tt.flow,
			}
			g := NewOAuth2Generator("test", scheme)

			assert.Equal(t, tt.wantAuthCode, g.hasAuthorizationCodeFlow())
			assert.Equal(t, tt.wantClientCreds, g.hasClientCredentialsFlow())
			assert.Equal(t, tt.wantPassword, g.hasPasswordFlow())
			assert.Equal(t, tt.wantImplicit, g.hasImplicitFlow())
		})
	}
}

func TestOAuth2Generator_GetURLs(t *testing.T) {
	// OAS 3.0+ style
	scheme := &parser.SecurityScheme{
		Type: "oauth2",
		Flows: &parser.OAuthFlows{
			AuthorizationCode: &parser.OAuthFlow{
				AuthorizationURL: "https://example.com/authorize",
				TokenURL:         "https://example.com/token",
			},
		},
	}
	g := NewOAuth2Generator("test", scheme)
	authURL, tokenURL := g.getURLs()

	assert.Equal(t, "https://example.com/authorize", authURL)
	assert.Equal(t, "https://example.com/token", tokenURL)

	// OAS 2.0 style
	scheme2 := &parser.SecurityScheme{
		Type:             "oauth2",
		Flow:             "accessCode",
		AuthorizationURL: "https://example2.com/authorize",
		TokenURL:         "https://example2.com/token",
	}
	g2 := NewOAuth2Generator("test", scheme2)
	authURL2, tokenURL2 := g2.getURLs()

	assert.Equal(t, "https://example2.com/authorize", authURL2)
	assert.Equal(t, "https://example2.com/token", tokenURL2)
}

func TestOAuth2Generator_CollectScopes(t *testing.T) {
	// OAS 3.0+ style
	scheme := &parser.SecurityScheme{
		Type: "oauth2",
		Flows: &parser.OAuthFlows{
			AuthorizationCode: &parser.OAuthFlow{
				Scopes: map[string]string{
					"read":  "Read access",
					"write": "Write access",
				},
			},
		},
	}
	g := NewOAuth2Generator("test", scheme)
	scopes := g.collectScopes()

	assert.Len(t, scopes, 2)

	// OAS 2.0 style
	scheme2 := &parser.SecurityScheme{
		Type: "oauth2",
		Flow: "accessCode",
		Scopes: map[string]string{
			"read":  "Read access",
			"admin": "Admin access",
		},
	}
	g2 := NewOAuth2Generator("test", scheme2)
	scopes2 := g2.collectScopes()

	assert.Len(t, scopes2, 2)
}

func TestOAuth2Generator_GenerateOAuth2File(t *testing.T) {
	scheme := &parser.SecurityScheme{
		Type: "oauth2",
		Flows: &parser.OAuthFlows{
			AuthorizationCode: &parser.OAuthFlow{
				AuthorizationURL: "https://example.com/authorize",
				TokenURL:         "https://example.com/token",
				Scopes: map[string]string{
					"read":  "Read access",
					"write": "Write access",
				},
			},
		},
	}
	g := NewOAuth2Generator("oauth2", scheme)
	result := g.GenerateOAuth2File("api")

	// Check package declaration
	assert.Contains(t, result, "package api")

	// Check imports
	assert.Contains(t, result, `"context"`)
	assert.Contains(t, result, `"net/http"`)

	// Check config struct
	assert.Contains(t, result, "type Oauth2OAuth2Config struct")

	// Check token struct
	assert.Contains(t, result, "type OAuth2Token struct")

	// Check client struct
	assert.Contains(t, result, "type Oauth2OAuth2Client struct")

	// Check constructor
	assert.Contains(t, result, "func NewOauth2OAuth2Client")

	// Check authorization code flow methods
	assert.Contains(t, result, "func (c *Oauth2OAuth2Client) GetAuthorizationURL")
	assert.Contains(t, result, "func (c *Oauth2OAuth2Client) ExchangeCode")

	// Check refresh token
	assert.Contains(t, result, "func (c *Oauth2OAuth2Client) RefreshToken")

	// Check auto-refresh option
	assert.Contains(t, result, "func WithOauth2OAuth2AutoRefresh")
}

func TestOAuth2Generator_GenerateClientCredentials(t *testing.T) {
	scheme := &parser.SecurityScheme{
		Type: "oauth2",
		Flows: &parser.OAuthFlows{
			ClientCredentials: &parser.OAuthFlow{
				TokenURL: "https://example.com/token",
			},
		},
	}
	g := NewOAuth2Generator("oauth2", scheme)
	result := g.GenerateOAuth2File("api")

	// Should have client credentials method
	assert.Contains(t, result, "func (c *Oauth2OAuth2Client) GetClientCredentialsToken")

	// Should NOT have authorization code methods
	assert.NotContains(t, result, "GetAuthorizationURL")
}

func TestOAuth2Generator_GeneratePassword(t *testing.T) {
	scheme := &parser.SecurityScheme{
		Type: "oauth2",
		Flows: &parser.OAuthFlows{
			Password: &parser.OAuthFlow{
				TokenURL: "https://example.com/token",
			},
		},
	}
	g := NewOAuth2Generator("oauth2", scheme)
	result := g.GenerateOAuth2File("api")

	// Should have password method
	assert.Contains(t, result, "func (c *Oauth2OAuth2Client) GetPasswordToken")

	// Should have warning about trusted applications
	assert.Contains(t, result, "trusted first-party applications")
}

func TestOAuth2Generator_GenerateImplicit(t *testing.T) {
	scheme := &parser.SecurityScheme{
		Type: "oauth2",
		Flows: &parser.OAuthFlows{
			Implicit: &parser.OAuthFlow{
				AuthorizationURL: "https://example.com/authorize",
			},
		},
	}
	g := NewOAuth2Generator("oauth2", scheme)
	result := g.GenerateOAuth2File("api")

	// Should have implicit method
	assert.Contains(t, result, "GetImplicitAuthorizationURL")

	// Should have deprecation warning
	assert.Contains(t, result, "Deprecated:")
}

func TestOAuth2Generator_HasAnyFlow(t *testing.T) {
	// No flows
	scheme := &parser.SecurityScheme{
		Type: "oauth2",
	}
	g := NewOAuth2Generator("test", scheme)
	assert.False(t, g.HasAnyFlow())

	// With authorization code flow
	scheme2 := &parser.SecurityScheme{
		Type: "oauth2",
		Flows: &parser.OAuthFlows{
			AuthorizationCode: &parser.OAuthFlow{
				AuthorizationURL: "https://example.com/authorize",
				TokenURL:         "https://example.com/token",
			},
		},
	}
	g2 := NewOAuth2Generator("test", scheme2)
	assert.True(t, g2.HasAnyFlow())
}

func TestOAuth2Generator_GeneratePKCEHelpers(t *testing.T) {
	scheme := &parser.SecurityScheme{
		Type: "oauth2",
		Flows: &parser.OAuthFlows{
			AuthorizationCode: &parser.OAuthFlow{
				AuthorizationURL: "https://example.com/authorize",
				TokenURL:         "https://example.com/token",
			},
		},
	}
	g := NewOAuth2Generator("oauth2", scheme)
	result := g.GenerateOAuth2File("api")

	// Check PKCE struct
	assert.Contains(t, result, "type PKCEChallenge struct")
	assert.Contains(t, result, "CodeVerifier string")
	assert.Contains(t, result, "CodeChallenge string")
	assert.Contains(t, result, "CodeChallengeMethod string")

	// Check PKCE generator function
	assert.Contains(t, result, "func GeneratePKCEChallenge()")

	// Check crypto imports for PKCE
	assert.Contains(t, result, `"crypto/rand"`)
	assert.Contains(t, result, `"crypto/sha256"`)
	assert.Contains(t, result, `"encoding/base64"`)

	// Check PKCE-enabled authorization URL method
	assert.Contains(t, result, "func (c *Oauth2OAuth2Client) GetAuthorizationURLWithPKCE")
	assert.Contains(t, result, "code_challenge")
	assert.Contains(t, result, "code_challenge_method")

	// Check PKCE-enabled exchange method
	assert.Contains(t, result, "func (c *Oauth2OAuth2Client) ExchangeCodeWithPKCE")
	assert.Contains(t, result, "code_verifier")
}

func TestOAuth2Generator_NoPKCEForClientCredentials(t *testing.T) {
	// PKCE should NOT be generated for client credentials flow (no auth code)
	scheme := &parser.SecurityScheme{
		Type: "oauth2",
		Flows: &parser.OAuthFlows{
			ClientCredentials: &parser.OAuthFlow{
				TokenURL: "https://example.com/token",
			},
		},
	}
	g := NewOAuth2Generator("oauth2", scheme)
	result := g.GenerateOAuth2File("api")

	// PKCE should not be present for client credentials only
	assert.NotContains(t, result, "type PKCEChallenge struct")
	assert.NotContains(t, result, "GetAuthorizationURLWithPKCE")
}
