package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewOIDCDiscoveryGenerator(t *testing.T) {
	g := NewOIDCDiscoveryGenerator("api")

	assert.Equal(t, "api", g.PackageName)
}

func TestOIDCDiscoveryGenerator_GenerateOIDCDiscoveryFile(t *testing.T) {
	g := NewOIDCDiscoveryGenerator("api")
	result := g.GenerateOIDCDiscoveryFile("https://auth.example.com")

	// Check package declaration
	assert.Contains(t, result, "package api")

	// Check imports
	assert.Contains(t, result, `"context"`)
	assert.Contains(t, result, `"encoding/json"`)
	assert.Contains(t, result, `"net/http"`)
	assert.Contains(t, result, `"sync"`)
	assert.Contains(t, result, `"time"`)

	// Check OIDCConfiguration struct
	assert.Contains(t, result, "type OIDCConfiguration struct")
	assert.Contains(t, result, "Issuer string")
	assert.Contains(t, result, "AuthorizationEndpoint string")
	assert.Contains(t, result, "TokenEndpoint string")
	assert.Contains(t, result, "JwksURI string")
	assert.Contains(t, result, "ScopesSupported []string")
	assert.Contains(t, result, "CodeChallengeMethodsSupported []string")

	// Check default discovery URL constant
	assert.Contains(t, result, "DefaultOIDCDiscoveryURL")
	assert.Contains(t, result, `"https://auth.example.com"`)

	// Check OIDCDiscoveryClient struct
	assert.Contains(t, result, "type OIDCDiscoveryClient struct")
	assert.Contains(t, result, "discoveryURL string")
	assert.Contains(t, result, "httpClient   *http.Client")
	assert.Contains(t, result, "cacheTTL     time.Duration")

	// Check constructor
	assert.Contains(t, result, "func NewOIDCDiscoveryClient(discoveryURL string")

	// Check options
	assert.Contains(t, result, "type OIDCDiscoveryOption func(*OIDCDiscoveryClient)")
	assert.Contains(t, result, "func WithOIDCHTTPClient(client *http.Client) OIDCDiscoveryOption")
	assert.Contains(t, result, "func WithOIDCCacheTTL(ttl time.Duration) OIDCDiscoveryOption")

	// Check methods
	assert.Contains(t, result, "func (c *OIDCDiscoveryClient) GetConfiguration(ctx context.Context)")
	assert.Contains(t, result, "func (c *OIDCDiscoveryClient) ClearCache()")
	assert.Contains(t, result, "func (c *OIDCDiscoveryClient) SupportsScope(ctx context.Context, scope string)")
	assert.Contains(t, result, "func (c *OIDCDiscoveryClient) SupportsGrantType(ctx context.Context, grantType string)")
	assert.Contains(t, result, "func (c *OIDCDiscoveryClient) SupportsPKCE(ctx context.Context)")

	// Check helper functions
	assert.Contains(t, result, "func GetOAuth2ConfigFromOIDC(ctx context.Context, discoveryURL string)")
	assert.Contains(t, result, "func WithOIDCDiscovery(discoveryURL string, tokenFunc func(ctx context.Context) (string, error)) ClientOption")

	// Check caching implementation
	assert.Contains(t, result, "time.Since(c.lastFetched) < c.cacheTTL")

	// Check well-known path handling
	assert.Contains(t, result, ".well-known/openid-configuration")
}

func TestOIDCDiscoveryGenerator_NoDiscoveryURL(t *testing.T) {
	g := NewOIDCDiscoveryGenerator("api")
	result := g.GenerateOIDCDiscoveryFile("")

	// Should NOT have default discovery URL constant when empty
	assert.NotContains(t, result, "DefaultOIDCDiscoveryURL")

	// Should still have all other functionality
	assert.Contains(t, result, "type OIDCConfiguration struct")
	assert.Contains(t, result, "type OIDCDiscoveryClient struct")
}

func TestOIDCDiscoveryGenerator_ThreadSafety(t *testing.T) {
	g := NewOIDCDiscoveryGenerator("api")
	result := g.GenerateOIDCDiscoveryFile("https://auth.example.com")

	// Should use sync.RWMutex
	assert.Contains(t, result, "mu           sync.RWMutex")

	// Should use RLock for reads
	assert.Contains(t, result, "c.mu.RLock()")

	// Should use Lock for writes
	assert.Contains(t, result, "c.mu.Lock()")
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
		assert.Contains(t, result, tag)
	}
}

func TestOIDCDiscoveryGenerator_ErrorHandling(t *testing.T) {
	g := NewOIDCDiscoveryGenerator("api")
	result := g.GenerateOIDCDiscoveryFile("")

	// Should have error handling for HTTP request
	assert.Contains(t, result, "failed to create OIDC discovery request")
	assert.Contains(t, result, "failed to fetch OIDC configuration")
	assert.Contains(t, result, "failed to decode OIDC configuration")
	assert.Contains(t, result, "OIDC discovery returned status")
}
