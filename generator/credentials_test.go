package generator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCredentialGenerator_GenerateCredentialsFile(t *testing.T) {
	g := NewCredentialGenerator("api")
	result := g.GenerateCredentialsFile()

	// Check package declaration
	assert.Contains(t, result, "package api")

	// Check imports
	assert.Contains(t, result, `"context"`)
	assert.Contains(t, result, `"os"`)
	assert.Contains(t, result, `"sync"`)

	// Check CredentialProvider interface
	assert.Contains(t, result, "type CredentialProvider interface")
	assert.Contains(t, result, "GetCredential(ctx context.Context, name string) (string, error)")

	// Check MemoryCredentialProvider
	assert.Contains(t, result, "type MemoryCredentialProvider struct")
	assert.Contains(t, result, "func NewMemoryCredentialProvider()")
	assert.Contains(t, result, "func (p *MemoryCredentialProvider) Set(name, value string)")
	assert.Contains(t, result, "func (p *MemoryCredentialProvider) Delete(name string)")
	assert.Contains(t, result, "func (p *MemoryCredentialProvider) GetCredential")

	// Check EnvCredentialProvider
	assert.Contains(t, result, "type EnvCredentialProvider struct")
	assert.Contains(t, result, "func NewEnvCredentialProvider(prefix string)")
	assert.Contains(t, result, "func (p *EnvCredentialProvider) GetCredential")
	assert.Contains(t, result, "os.Getenv(envName)")

	// Check CredentialChain
	assert.Contains(t, result, "type CredentialChain struct")
	assert.Contains(t, result, "func NewCredentialChain(providers ...CredentialProvider)")
	assert.Contains(t, result, "func (c *CredentialChain) GetCredential")

	// Check WithCredentialProvider option
	assert.Contains(t, result, "func WithCredentialProvider(provider CredentialProvider, credentialName string) ClientOption")

	// Check WithCredentialProviderHeader option
	assert.Contains(t, result, "func WithCredentialProviderHeader(provider CredentialProvider, credentialName, headerName string) ClientOption")
}

func TestCredentialGenerator_EnvConversion(t *testing.T) {
	g := NewCredentialGenerator("api")
	result := g.GenerateCredentialsFile()

	// Check that env conversion handles hyphens
	assert.Contains(t, result, `strings.ReplaceAll(name, "-", "_")`)

	// Check uppercase conversion
	assert.Contains(t, result, "strings.ToUpper")
}

func TestCredentialGenerator_ThreadSafety(t *testing.T) {
	g := NewCredentialGenerator("api")
	result := g.GenerateCredentialsFile()

	// Memory provider should use mutex
	assert.Contains(t, result, "mu.Lock()")
	assert.Contains(t, result, "mu.RLock()")
}
