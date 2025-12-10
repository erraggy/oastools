package generator

import (
	"strings"
	"testing"
)

func TestCredentialGenerator_GenerateCredentialsFile(t *testing.T) {
	g := NewCredentialGenerator("api")
	result := g.GenerateCredentialsFile()

	// Check package declaration
	if !strings.Contains(result, "package api") {
		t.Error("expected package declaration")
	}

	// Check imports
	if !strings.Contains(result, `"context"`) {
		t.Error("expected context import")
	}
	if !strings.Contains(result, `"os"`) {
		t.Error("expected os import")
	}
	if !strings.Contains(result, `"sync"`) {
		t.Error("expected sync import")
	}

	// Check CredentialProvider interface
	if !strings.Contains(result, "type CredentialProvider interface") {
		t.Error("expected CredentialProvider interface")
	}
	if !strings.Contains(result, "GetCredential(ctx context.Context, name string) (string, error)") {
		t.Error("expected GetCredential method signature")
	}

	// Check MemoryCredentialProvider
	if !strings.Contains(result, "type MemoryCredentialProvider struct") {
		t.Error("expected MemoryCredentialProvider struct")
	}
	if !strings.Contains(result, "func NewMemoryCredentialProvider()") {
		t.Error("expected NewMemoryCredentialProvider function")
	}
	if !strings.Contains(result, "func (p *MemoryCredentialProvider) Set(name, value string)") {
		t.Error("expected Set method")
	}
	if !strings.Contains(result, "func (p *MemoryCredentialProvider) Delete(name string)") {
		t.Error("expected Delete method")
	}
	if !strings.Contains(result, "func (p *MemoryCredentialProvider) GetCredential") {
		t.Error("expected MemoryCredentialProvider.GetCredential method")
	}

	// Check EnvCredentialProvider
	if !strings.Contains(result, "type EnvCredentialProvider struct") {
		t.Error("expected EnvCredentialProvider struct")
	}
	if !strings.Contains(result, "func NewEnvCredentialProvider(prefix string)") {
		t.Error("expected NewEnvCredentialProvider function")
	}
	if !strings.Contains(result, "func (p *EnvCredentialProvider) GetCredential") {
		t.Error("expected EnvCredentialProvider.GetCredential method")
	}
	if !strings.Contains(result, "os.Getenv(envName)") {
		t.Error("expected os.Getenv usage")
	}

	// Check CredentialChain
	if !strings.Contains(result, "type CredentialChain struct") {
		t.Error("expected CredentialChain struct")
	}
	if !strings.Contains(result, "func NewCredentialChain(providers ...CredentialProvider)") {
		t.Error("expected NewCredentialChain function")
	}
	if !strings.Contains(result, "func (c *CredentialChain) GetCredential") {
		t.Error("expected CredentialChain.GetCredential method")
	}

	// Check WithCredentialProvider option
	if !strings.Contains(result, "func WithCredentialProvider(provider CredentialProvider, credentialName string) ClientOption") {
		t.Error("expected WithCredentialProvider function")
	}

	// Check WithCredentialProviderHeader option
	if !strings.Contains(result, "func WithCredentialProviderHeader(provider CredentialProvider, credentialName, headerName string) ClientOption") {
		t.Error("expected WithCredentialProviderHeader function")
	}
}

func TestCredentialGenerator_EnvConversion(t *testing.T) {
	g := NewCredentialGenerator("api")
	result := g.GenerateCredentialsFile()

	// Check that env conversion handles hyphens
	if !strings.Contains(result, `strings.ReplaceAll(name, "-", "_")`) {
		t.Error("expected hyphen to underscore conversion")
	}

	// Check uppercase conversion
	if !strings.Contains(result, "strings.ToUpper") {
		t.Error("expected uppercase conversion")
	}
}

func TestCredentialGenerator_ThreadSafety(t *testing.T) {
	g := NewCredentialGenerator("api")
	result := g.GenerateCredentialsFile()

	// Memory provider should use mutex
	if !strings.Contains(result, "mu.Lock()") {
		t.Error("expected Lock() for write operations")
	}
	if !strings.Contains(result, "mu.RLock()") {
		t.Error("expected RLock() for read operations")
	}
}
