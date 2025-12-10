package generator

import (
	"strings"
	"testing"
)

func TestNewOIDCDiscoveryGenerator(t *testing.T) {
	g := NewOIDCDiscoveryGenerator("api")

	if g.PackageName != "api" {
		t.Errorf("expected PackageName 'api', got %s", g.PackageName)
	}
}

func TestOIDCDiscoveryGenerator_GenerateOIDCDiscoveryFile(t *testing.T) {
	g := NewOIDCDiscoveryGenerator("api")
	result := g.GenerateOIDCDiscoveryFile("https://auth.example.com")

	// Check package declaration
	if !strings.Contains(result, "package api") {
		t.Error("expected package declaration")
	}

	// Check imports
	if !strings.Contains(result, `"context"`) {
		t.Error("expected context import")
	}
	if !strings.Contains(result, `"encoding/json"`) {
		t.Error("expected encoding/json import")
	}
	if !strings.Contains(result, `"net/http"`) {
		t.Error("expected net/http import")
	}
	if !strings.Contains(result, `"sync"`) {
		t.Error("expected sync import")
	}
	if !strings.Contains(result, `"time"`) {
		t.Error("expected time import")
	}

	// Check OIDCConfiguration struct
	if !strings.Contains(result, "type OIDCConfiguration struct") {
		t.Error("expected OIDCConfiguration struct")
	}
	if !strings.Contains(result, "Issuer string") {
		t.Error("expected Issuer field")
	}
	if !strings.Contains(result, "AuthorizationEndpoint string") {
		t.Error("expected AuthorizationEndpoint field")
	}
	if !strings.Contains(result, "TokenEndpoint string") {
		t.Error("expected TokenEndpoint field")
	}
	if !strings.Contains(result, "JwksURI string") {
		t.Error("expected JwksURI field")
	}
	if !strings.Contains(result, "ScopesSupported []string") {
		t.Error("expected ScopesSupported field")
	}
	if !strings.Contains(result, "CodeChallengeMethodsSupported []string") {
		t.Error("expected CodeChallengeMethodsSupported field")
	}

	// Check default discovery URL constant
	if !strings.Contains(result, "DefaultOIDCDiscoveryURL") {
		t.Error("expected DefaultOIDCDiscoveryURL constant")
	}
	if !strings.Contains(result, `"https://auth.example.com"`) {
		t.Error("expected discovery URL value")
	}

	// Check OIDCDiscoveryClient struct
	if !strings.Contains(result, "type OIDCDiscoveryClient struct") {
		t.Error("expected OIDCDiscoveryClient struct")
	}
	if !strings.Contains(result, "discoveryURL string") {
		t.Error("expected discoveryURL field")
	}
	if !strings.Contains(result, "httpClient   *http.Client") {
		t.Error("expected httpClient field")
	}
	if !strings.Contains(result, "cacheTTL     time.Duration") {
		t.Error("expected cacheTTL field")
	}

	// Check constructor
	if !strings.Contains(result, "func NewOIDCDiscoveryClient(discoveryURL string") {
		t.Error("expected NewOIDCDiscoveryClient function")
	}

	// Check options
	if !strings.Contains(result, "type OIDCDiscoveryOption func(*OIDCDiscoveryClient)") {
		t.Error("expected OIDCDiscoveryOption type")
	}
	if !strings.Contains(result, "func WithOIDCHTTPClient(client *http.Client) OIDCDiscoveryOption") {
		t.Error("expected WithOIDCHTTPClient function")
	}
	if !strings.Contains(result, "func WithOIDCCacheTTL(ttl time.Duration) OIDCDiscoveryOption") {
		t.Error("expected WithOIDCCacheTTL function")
	}

	// Check methods
	if !strings.Contains(result, "func (c *OIDCDiscoveryClient) GetConfiguration(ctx context.Context)") {
		t.Error("expected GetConfiguration method")
	}
	if !strings.Contains(result, "func (c *OIDCDiscoveryClient) ClearCache()") {
		t.Error("expected ClearCache method")
	}
	if !strings.Contains(result, "func (c *OIDCDiscoveryClient) SupportsScope(ctx context.Context, scope string)") {
		t.Error("expected SupportsScope method")
	}
	if !strings.Contains(result, "func (c *OIDCDiscoveryClient) SupportsGrantType(ctx context.Context, grantType string)") {
		t.Error("expected SupportsGrantType method")
	}
	if !strings.Contains(result, "func (c *OIDCDiscoveryClient) SupportsPKCE(ctx context.Context)") {
		t.Error("expected SupportsPKCE method")
	}

	// Check helper functions
	if !strings.Contains(result, "func GetOAuth2ConfigFromOIDC(ctx context.Context, discoveryURL string)") {
		t.Error("expected GetOAuth2ConfigFromOIDC function")
	}
	if !strings.Contains(result, "func WithOIDCDiscovery(discoveryURL string, tokenFunc func(ctx context.Context) (string, error)) ClientOption") {
		t.Error("expected WithOIDCDiscovery function")
	}

	// Check caching implementation
	if !strings.Contains(result, "time.Since(c.lastFetched) < c.cacheTTL") {
		t.Error("expected cache TTL check")
	}

	// Check well-known path handling
	if !strings.Contains(result, ".well-known/openid-configuration") {
		t.Error("expected well-known path")
	}
}

func TestOIDCDiscoveryGenerator_NoDiscoveryURL(t *testing.T) {
	g := NewOIDCDiscoveryGenerator("api")
	result := g.GenerateOIDCDiscoveryFile("")

	// Should NOT have default discovery URL constant when empty
	if strings.Contains(result, "DefaultOIDCDiscoveryURL") {
		t.Error("should not have DefaultOIDCDiscoveryURL when URL is empty")
	}

	// Should still have all other functionality
	if !strings.Contains(result, "type OIDCConfiguration struct") {
		t.Error("expected OIDCConfiguration struct")
	}
	if !strings.Contains(result, "type OIDCDiscoveryClient struct") {
		t.Error("expected OIDCDiscoveryClient struct")
	}
}

func TestOIDCDiscoveryGenerator_ThreadSafety(t *testing.T) {
	g := NewOIDCDiscoveryGenerator("api")
	result := g.GenerateOIDCDiscoveryFile("https://auth.example.com")

	// Should use sync.RWMutex
	if !strings.Contains(result, "mu           sync.RWMutex") {
		t.Error("expected sync.RWMutex for thread safety")
	}

	// Should use RLock for reads
	if !strings.Contains(result, "c.mu.RLock()") {
		t.Error("expected RLock for read operations")
	}

	// Should use Lock for writes
	if !strings.Contains(result, "c.mu.Lock()") {
		t.Error("expected Lock for write operations")
	}
}

func TestOIDCDiscoveryGenerator_JSONTags(t *testing.T) {
	g := NewOIDCDiscoveryGenerator("api")
	result := g.GenerateOIDCDiscoveryFile("")

	// Check JSON tags are present
	expectedTags := []string{
		`json:"issuer"`,
		`json:"authorization_endpoint"`,
		`json:"token_endpoint"`,
		`json:"jwks_uri"`,
		`json:"scopes_supported,omitempty"`,
		`json:"grant_types_supported,omitempty"`,
		`json:"code_challenge_methods_supported,omitempty"`,
	}

	for _, tag := range expectedTags {
		if !strings.Contains(result, tag) {
			t.Errorf("expected JSON tag %s", tag)
		}
	}
}

func TestOIDCDiscoveryGenerator_ErrorHandling(t *testing.T) {
	g := NewOIDCDiscoveryGenerator("api")
	result := g.GenerateOIDCDiscoveryFile("")

	// Should have error handling for HTTP request
	if !strings.Contains(result, "failed to create OIDC discovery request") {
		t.Error("expected error message for request creation")
	}
	if !strings.Contains(result, "failed to fetch OIDC configuration") {
		t.Error("expected error message for fetch failure")
	}
	if !strings.Contains(result, "failed to decode OIDC configuration") {
		t.Error("expected error message for decode failure")
	}
	if !strings.Contains(result, "OIDC discovery returned status") {
		t.Error("expected error message for non-200 status")
	}
}
