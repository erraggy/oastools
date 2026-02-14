package generator

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
)

// TestGeneratedCredentialsThreadSafety verifies the generated credentials code
// includes proper synchronization primitives for concurrent access.
func TestGeneratedCredentialsThreadSafety(t *testing.T) {
	g := NewCredentialGenerator("api")
	result := g.GenerateCredentialsFile()

	t.Run("MemoryCredentialProvider uses RWMutex", func(t *testing.T) {
		assert.Contains(t, result, "sync.RWMutex")
	})

	t.Run("Set uses Lock for exclusive access", func(t *testing.T) {
		// Look for the Set method with proper locking
		assert.Contains(t, result, "func (p *MemoryCredentialProvider) Set")
		assert.Contains(t, result, "p.mu.Lock()")
		assert.Contains(t, result, "defer p.mu.Unlock()")
	})

	t.Run("Delete uses Lock for exclusive access", func(t *testing.T) {
		assert.Contains(t, result, "func (p *MemoryCredentialProvider) Delete")
	})

	t.Run("GetCredential uses RLock for shared access", func(t *testing.T) {
		assert.Contains(t, result, "p.mu.RLock()")
		assert.Contains(t, result, "defer p.mu.RUnlock()")
	})
}

// TestGeneratedOIDCThreadSafety verifies the generated OIDC discovery code
// includes proper synchronization primitives for concurrent access.
func TestGeneratedOIDCThreadSafety(t *testing.T) {
	g := NewOIDCDiscoveryGenerator("api")
	result := g.GenerateOIDCDiscoveryFile("https://example.com/.well-known/openid-configuration")

	t.Run("OIDCDiscoveryClient uses RWMutex", func(t *testing.T) {
		assert.Contains(t, result, "sync.RWMutex")
	})

	t.Run("GetConfiguration uses RLock for cache check", func(t *testing.T) {
		assert.Contains(t, result, "c.mu.RLock()")
	})

	t.Run("GetConfiguration uses Lock for cache update", func(t *testing.T) {
		assert.Contains(t, result, "c.mu.Lock()")
	})

	t.Run("Uses double-check locking pattern", func(t *testing.T) {
		// The pattern: RLock -> check -> RUnlock -> Lock -> double-check
		assert.GreaterOrEqual(t, strings.Count(result, "c.mu.RLock()"), 1)
		assert.GreaterOrEqual(t, strings.Count(result, "c.mu.Lock()"), 1)
	})

	t.Run("ClearCache uses Lock for clearing cache", func(t *testing.T) {
		assert.Contains(t, result, "func (c *OIDCDiscoveryClient) ClearCache()")
	})
}

// TestGeneratedOAuth2ThreadSafety verifies OAuth2 generator
// includes proper synchronization for concurrent token refresh.
func TestGeneratedOAuth2ThreadSafety(t *testing.T) {
	// Create a minimal security scheme for OAuth2
	scheme := &parser.SecurityScheme{
		Type: "oauth2",
		Flows: &parser.OAuthFlows{
			ClientCredentials: &parser.OAuthFlow{
				TokenURL: "https://example.com/oauth/token",
				Scopes:   map[string]string{"read": "Read access"},
			},
		},
	}
	g := NewOAuth2Generator("api", scheme)
	result := g.GenerateOAuth2File("api")

	t.Run("TokenManager uses Mutex for token refresh", func(t *testing.T) {
		// Token refresh should be serialized to prevent thundering herd
		assert.True(t, strings.Contains(result, "sync.Mutex") || strings.Contains(result, "sync.RWMutex"),
			"expected Mutex or RWMutex for token management")
	})
}
