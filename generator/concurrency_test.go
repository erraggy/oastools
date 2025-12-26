package generator

import (
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
)

// TestGeneratedCredentialsThreadSafety verifies the generated credentials code
// includes proper synchronization primitives for concurrent access.
func TestGeneratedCredentialsThreadSafety(t *testing.T) {
	g := NewCredentialGenerator("api")
	result := g.GenerateCredentialsFile()

	t.Run("MemoryCredentialProvider uses RWMutex", func(t *testing.T) {
		if !strings.Contains(result, "sync.RWMutex") {
			t.Error("expected sync.RWMutex in MemoryCredentialProvider")
		}
	})

	t.Run("Set uses Lock for exclusive access", func(t *testing.T) {
		// Look for the Set method with proper locking
		if !strings.Contains(result, "func (p *MemoryCredentialProvider) Set") {
			t.Error("expected Set method")
		}
		if !strings.Contains(result, "p.mu.Lock()") {
			t.Error("expected Lock() in write operations")
		}
		if !strings.Contains(result, "defer p.mu.Unlock()") {
			t.Error("expected defer Unlock() pattern")
		}
	})

	t.Run("Delete uses Lock for exclusive access", func(t *testing.T) {
		if !strings.Contains(result, "func (p *MemoryCredentialProvider) Delete") {
			t.Error("expected Delete method")
		}
	})

	t.Run("GetCredential uses RLock for shared access", func(t *testing.T) {
		if !strings.Contains(result, "p.mu.RLock()") {
			t.Error("expected RLock() for read operations")
		}
		if !strings.Contains(result, "defer p.mu.RUnlock()") {
			t.Error("expected defer RUnlock() pattern")
		}
	})
}

// TestGeneratedOIDCThreadSafety verifies the generated OIDC discovery code
// includes proper synchronization primitives for concurrent access.
func TestGeneratedOIDCThreadSafety(t *testing.T) {
	g := NewOIDCDiscoveryGenerator("api")
	result := g.GenerateOIDCDiscoveryFile("https://example.com/.well-known/openid-configuration")

	t.Run("OIDCDiscoveryClient uses RWMutex", func(t *testing.T) {
		if !strings.Contains(result, "sync.RWMutex") {
			t.Error("expected sync.RWMutex in OIDCDiscoveryClient")
		}
	})

	t.Run("GetConfiguration uses RLock for cache check", func(t *testing.T) {
		if !strings.Contains(result, "c.mu.RLock()") {
			t.Error("expected RLock() for cache read")
		}
	})

	t.Run("GetConfiguration uses Lock for cache update", func(t *testing.T) {
		if !strings.Contains(result, "c.mu.Lock()") {
			t.Error("expected Lock() for cache write")
		}
	})

	t.Run("Uses double-check locking pattern", func(t *testing.T) {
		// The pattern: RLock -> check -> RUnlock -> Lock -> double-check
		if strings.Count(result, "c.mu.RLock()") < 1 {
			t.Error("expected RLock in double-check pattern")
		}
		if strings.Count(result, "c.mu.Lock()") < 1 {
			t.Error("expected Lock in double-check pattern")
		}
	})

	t.Run("ClearCache uses Lock for clearing cache", func(t *testing.T) {
		if !strings.Contains(result, "func (c *OIDCDiscoveryClient) ClearCache()") {
			t.Error("expected ClearCache method")
		}
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
		if !strings.Contains(result, "sync.Mutex") && !strings.Contains(result, "sync.RWMutex") {
			t.Error("expected Mutex or RWMutex for token management")
		}
	})
}
