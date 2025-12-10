package generator

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
)

func TestNewOAuth2Generator(t *testing.T) {
	// Test with nil scheme
	g := NewOAuth2Generator("test", nil)
	if g != nil {
		t.Error("expected nil for nil scheme")
	}

	// Test with non-oauth2 scheme
	g = NewOAuth2Generator("test", &parser.SecurityScheme{Type: "apiKey"})
	if g != nil {
		t.Error("expected nil for non-oauth2 scheme")
	}

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
	if g == nil {
		t.Fatal("expected generator for valid scheme")
	}
	if g.Name != "test" {
		t.Errorf("expected name 'test', got %s", g.Name)
	}
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

	if !g.hasAuthorizationCodeFlow() {
		t.Error("expected authorization code flow")
	}
	if g.hasClientCredentialsFlow() {
		t.Error("did not expect client credentials flow")
	}
	if g.hasPasswordFlow() {
		t.Error("did not expect password flow")
	}
	if g.hasImplicitFlow() {
		t.Error("did not expect implicit flow")
	}
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

			if got := g.hasAuthorizationCodeFlow(); got != tt.wantAuthCode {
				t.Errorf("hasAuthorizationCodeFlow() = %v, want %v", got, tt.wantAuthCode)
			}
			if got := g.hasClientCredentialsFlow(); got != tt.wantClientCreds {
				t.Errorf("hasClientCredentialsFlow() = %v, want %v", got, tt.wantClientCreds)
			}
			if got := g.hasPasswordFlow(); got != tt.wantPassword {
				t.Errorf("hasPasswordFlow() = %v, want %v", got, tt.wantPassword)
			}
			if got := g.hasImplicitFlow(); got != tt.wantImplicit {
				t.Errorf("hasImplicitFlow() = %v, want %v", got, tt.wantImplicit)
			}
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

	if authURL != "https://example.com/authorize" {
		t.Errorf("expected authURL 'https://example.com/authorize', got %s", authURL)
	}
	if tokenURL != "https://example.com/token" {
		t.Errorf("expected tokenURL 'https://example.com/token', got %s", tokenURL)
	}

	// OAS 2.0 style
	scheme2 := &parser.SecurityScheme{
		Type:             "oauth2",
		Flow:             "accessCode",
		AuthorizationURL: "https://example2.com/authorize",
		TokenURL:         "https://example2.com/token",
	}
	g2 := NewOAuth2Generator("test", scheme2)
	authURL2, tokenURL2 := g2.getURLs()

	if authURL2 != "https://example2.com/authorize" {
		t.Errorf("expected authURL 'https://example2.com/authorize', got %s", authURL2)
	}
	if tokenURL2 != "https://example2.com/token" {
		t.Errorf("expected tokenURL 'https://example2.com/token', got %s", tokenURL2)
	}
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

	if len(scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(scopes))
	}

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

	if len(scopes2) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(scopes2))
	}
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
	if !strings.Contains(result, "package api") {
		t.Error("expected package declaration")
	}

	// Check imports
	if !strings.Contains(result, `"context"`) {
		t.Error("expected context import")
	}
	if !strings.Contains(result, `"net/http"`) {
		t.Error("expected net/http import")
	}

	// Check config struct
	if !strings.Contains(result, "type Oauth2OAuth2Config struct") {
		t.Error("expected OAuth2Config struct")
	}

	// Check token struct
	if !strings.Contains(result, "type OAuth2Token struct") {
		t.Error("expected OAuth2Token struct")
	}

	// Check client struct
	if !strings.Contains(result, "type Oauth2OAuth2Client struct") {
		t.Error("expected OAuth2Client struct")
	}

	// Check constructor
	if !strings.Contains(result, "func NewOauth2OAuth2Client") {
		t.Error("expected NewOAuth2Client function")
	}

	// Check authorization code flow methods
	if !strings.Contains(result, "func (c *Oauth2OAuth2Client) GetAuthorizationURL") {
		t.Error("expected GetAuthorizationURL method")
	}
	if !strings.Contains(result, "func (c *Oauth2OAuth2Client) ExchangeCode") {
		t.Error("expected ExchangeCode method")
	}

	// Check refresh token
	if !strings.Contains(result, "func (c *Oauth2OAuth2Client) RefreshToken") {
		t.Error("expected RefreshToken method")
	}

	// Check auto-refresh option
	if !strings.Contains(result, "func WithOauth2OAuth2AutoRefresh") {
		t.Error("expected WithOAuth2AutoRefresh function")
	}
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
	if !strings.Contains(result, "func (c *Oauth2OAuth2Client) GetClientCredentialsToken") {
		t.Error("expected GetClientCredentialsToken method")
	}

	// Should NOT have authorization code methods
	if strings.Contains(result, "GetAuthorizationURL") {
		t.Error("did not expect GetAuthorizationURL method")
	}
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
	if !strings.Contains(result, "func (c *Oauth2OAuth2Client) GetPasswordToken") {
		t.Error("expected GetPasswordToken method")
	}

	// Should have warning about trusted applications
	if !strings.Contains(result, "trusted first-party applications") {
		t.Error("expected warning about trusted applications")
	}
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
	if !strings.Contains(result, "GetImplicitAuthorizationURL") {
		t.Error("expected GetImplicitAuthorizationURL method")
	}

	// Should have deprecation warning
	if !strings.Contains(result, "Deprecated:") {
		t.Error("expected deprecation warning")
	}
}

func TestOAuth2Generator_HasAnyFlow(t *testing.T) {
	// No flows
	scheme := &parser.SecurityScheme{
		Type: "oauth2",
	}
	g := NewOAuth2Generator("test", scheme)
	if g.HasAnyFlow() {
		t.Error("expected no flows")
	}

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
	if !g2.HasAnyFlow() {
		t.Error("expected flows to be detected")
	}
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
	if !strings.Contains(result, "type PKCEChallenge struct") {
		t.Error("expected PKCEChallenge struct")
	}
	if !strings.Contains(result, "CodeVerifier string") {
		t.Error("expected CodeVerifier field")
	}
	if !strings.Contains(result, "CodeChallenge string") {
		t.Error("expected CodeChallenge field")
	}
	if !strings.Contains(result, "CodeChallengeMethod string") {
		t.Error("expected CodeChallengeMethod field")
	}

	// Check PKCE generator function
	if !strings.Contains(result, "func GeneratePKCEChallenge()") {
		t.Error("expected GeneratePKCEChallenge function")
	}

	// Check crypto imports for PKCE
	if !strings.Contains(result, `"crypto/rand"`) {
		t.Error("expected crypto/rand import for PKCE")
	}
	if !strings.Contains(result, `"crypto/sha256"`) {
		t.Error("expected crypto/sha256 import for PKCE")
	}
	if !strings.Contains(result, `"encoding/base64"`) {
		t.Error("expected encoding/base64 import for PKCE")
	}

	// Check PKCE-enabled authorization URL method
	if !strings.Contains(result, "func (c *Oauth2OAuth2Client) GetAuthorizationURLWithPKCE") {
		t.Error("expected GetAuthorizationURLWithPKCE method")
	}
	if !strings.Contains(result, "code_challenge") {
		t.Error("expected code_challenge parameter in PKCE URL")
	}
	if !strings.Contains(result, "code_challenge_method") {
		t.Error("expected code_challenge_method parameter in PKCE URL")
	}

	// Check PKCE-enabled exchange method
	if !strings.Contains(result, "func (c *Oauth2OAuth2Client) ExchangeCodeWithPKCE") {
		t.Error("expected ExchangeCodeWithPKCE method")
	}
	if !strings.Contains(result, "code_verifier") {
		t.Error("expected code_verifier parameter in PKCE exchange")
	}
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
	if strings.Contains(result, "type PKCEChallenge struct") {
		t.Error("PKCE should not be generated for client credentials flow")
	}
	if strings.Contains(result, "GetAuthorizationURLWithPKCE") {
		t.Error("PKCE methods should not be generated for client credentials flow")
	}
}
