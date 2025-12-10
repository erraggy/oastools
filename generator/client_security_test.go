package generator

import (
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGenerateClientWithAPIKeyHeader tests API key header authentication generation.
func TestGenerateClientWithAPIKeyHeader(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: API Key Test
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
      responses:
        '200':
          description: OK
      security:
        - apiKeyHeader: []
components:
  securitySchemes:
    apiKeyHeader:
      type: apiKey
      in: header
      name: X-API-Key
      description: API key passed in header
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithSecurity(true),
	)
	require.NoError(t, err)

	securityFile := result.GetFile("security_helpers.go")
	require.NotNil(t, securityFile, "security_helpers.go not generated")

	content := string(securityFile.Content)
	assert.Contains(t, content, "WithApiKeyHeaderAPIKey")
	assert.Contains(t, content, "X-API-Key")
	assert.Contains(t, content, "req.Header.Set")
	assert.Contains(t, content, "ClientOption")

	// Verify it compiles
	fset := token.NewFileSet()
	_, err = goparser.ParseFile(fset, "security_helpers.go", securityFile.Content, goparser.AllErrors)
	assert.NoError(t, err, "security_helpers.go should be valid Go syntax")
}

// TestGenerateClientWithAPIKeyQuery tests API key query parameter authentication generation.
func TestGenerateClientWithAPIKeyQuery(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: API Key Query Test
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
      responses:
        '200':
          description: OK
components:
  securitySchemes:
    apiKeyQuery:
      type: apiKey
      in: query
      name: api_key
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithSecurity(true),
	)
	require.NoError(t, err)

	securityFile := result.GetFile("security_helpers.go")
	require.NotNil(t, securityFile)

	content := string(securityFile.Content)
	assert.Contains(t, content, "APIKeyQuery")
	assert.Contains(t, content, "api_key")
	assert.Contains(t, content, "req.URL.Query()")

	// Verify it compiles
	fset := token.NewFileSet()
	_, err = goparser.ParseFile(fset, "security_helpers.go", securityFile.Content, goparser.AllErrors)
	assert.NoError(t, err)
}

// TestGenerateClientWithAPIKeyCookie tests API key cookie authentication generation.
func TestGenerateClientWithAPIKeyCookie(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: API Key Cookie Test
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
      responses:
        '200':
          description: OK
components:
  securitySchemes:
    apiKeyCookie:
      type: apiKey
      in: cookie
      name: session_id
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithSecurity(true),
	)
	require.NoError(t, err)

	securityFile := result.GetFile("security_helpers.go")
	require.NotNil(t, securityFile)

	content := string(securityFile.Content)
	assert.Contains(t, content, "APIKeyCookie")
	assert.Contains(t, content, "session_id")
	assert.Contains(t, content, "AddCookie")

	// Verify it compiles
	fset := token.NewFileSet()
	_, err = goparser.ParseFile(fset, "security_helpers.go", securityFile.Content, goparser.AllErrors)
	assert.NoError(t, err)
}

// TestGenerateClientWithBasicAuth tests HTTP Basic authentication generation.
func TestGenerateClientWithBasicAuth(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Basic Auth Test
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
      responses:
        '200':
          description: OK
components:
  securitySchemes:
    basicAuth:
      type: http
      scheme: basic
      description: Basic HTTP authentication
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithSecurity(true),
	)
	require.NoError(t, err)

	securityFile := result.GetFile("security_helpers.go")
	require.NotNil(t, securityFile)

	content := string(securityFile.Content)
	assert.Contains(t, content, "BasicAuth")
	assert.Contains(t, content, "username")
	assert.Contains(t, content, "password")
	assert.Contains(t, content, "SetBasicAuth")

	// Verify it compiles
	fset := token.NewFileSet()
	_, err = goparser.ParseFile(fset, "security_helpers.go", securityFile.Content, goparser.AllErrors)
	assert.NoError(t, err)
}

// TestGenerateClientWithBearerToken tests HTTP Bearer token authentication generation.
func TestGenerateClientWithBearerToken(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Bearer Token Test
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
      responses:
        '200':
          description: OK
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: JWT Bearer token
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithSecurity(true),
	)
	require.NoError(t, err)

	securityFile := result.GetFile("security_helpers.go")
	require.NotNil(t, securityFile)

	content := string(securityFile.Content)
	assert.Contains(t, content, "BearerToken")
	assert.Contains(t, content, "Bearer")
	assert.Contains(t, content, "JWT")
	assert.Contains(t, content, `req.Header.Set("Authorization"`)

	// Verify it compiles
	fset := token.NewFileSet()
	_, err = goparser.ParseFile(fset, "security_helpers.go", securityFile.Content, goparser.AllErrors)
	assert.NoError(t, err)
}

// TestGenerateClientWithOAuth2 tests OAuth2 authentication generation.
func TestGenerateClientWithOAuth2(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: OAuth2 Test
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
      responses:
        '200':
          description: OK
components:
  securitySchemes:
    oauth2:
      type: oauth2
      description: OAuth2 authentication
      flows:
        authorizationCode:
          authorizationUrl: https://example.com/oauth/authorize
          tokenUrl: https://example.com/oauth/token
          scopes:
            read: Read access
            write: Write access
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithSecurity(true),
	)
	require.NoError(t, err)

	securityFile := result.GetFile("security_helpers.go")
	require.NotNil(t, securityFile)

	content := string(securityFile.Content)
	assert.Contains(t, content, "OAuth2Token")
	assert.Contains(t, content, "Bearer")
	assert.Contains(t, content, "read")
	assert.Contains(t, content, "write")

	// Verify it compiles
	fset := token.NewFileSet()
	_, err = goparser.ParseFile(fset, "security_helpers.go", securityFile.Content, goparser.AllErrors)
	assert.NoError(t, err)
}

// TestGenerateClientWithOpenIDConnect tests OpenID Connect authentication generation.
func TestGenerateClientWithOpenIDConnect(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: OIDC Test
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
      responses:
        '200':
          description: OK
components:
  securitySchemes:
    openIdConnect:
      type: openIdConnect
      openIdConnectUrl: https://example.com/.well-known/openid-configuration
      description: OpenID Connect authentication
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithSecurity(true),
	)
	require.NoError(t, err)

	securityFile := result.GetFile("security_helpers.go")
	require.NotNil(t, securityFile)

	content := string(securityFile.Content)
	assert.Contains(t, content, "Token")
	assert.Contains(t, content, "openid-configuration")
	assert.Contains(t, content, "Bearer")

	// Verify it compiles
	fset := token.NewFileSet()
	_, err = goparser.ParseFile(fset, "security_helpers.go", securityFile.Content, goparser.AllErrors)
	assert.NoError(t, err)
}

// TestGenerateClientWithMultipleSecuritySchemes tests generation with multiple security schemes.
func TestGenerateClientWithMultipleSecuritySchemes(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Multi Security Test
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
      responses:
        '200':
          description: OK
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
    bearerAuth:
      type: http
      scheme: bearer
    basicAuth:
      type: http
      scheme: basic
    oauth2:
      type: oauth2
      flows:
        clientCredentials:
          tokenUrl: https://example.com/token
          scopes:
            api: API access
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithSecurity(true),
	)
	require.NoError(t, err)

	securityFile := result.GetFile("security_helpers.go")
	require.NotNil(t, securityFile)

	content := string(securityFile.Content)

	// All security helpers should be present
	assert.Contains(t, content, "WithApiKeyAPIKey")
	assert.Contains(t, content, "WithBearerAuthBearerToken")
	assert.Contains(t, content, "WithBasicAuthBasicAuth")
	assert.Contains(t, content, "WithOauth2OAuth2Token")

	// Verify it compiles
	fset := token.NewFileSet()
	_, err = goparser.ParseFile(fset, "security_helpers.go", securityFile.Content, goparser.AllErrors)
	assert.NoError(t, err)
}

// TestGenerateClientWithoutSecurity tests that security.go is not generated when disabled.
func TestGenerateClientWithoutSecurity(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: No Security Test
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
      responses:
        '200':
          description: OK
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithSecurity(false), // Disable security generation
	)
	require.NoError(t, err)

	securityFile := result.GetFile("security_helpers.go")
	assert.Nil(t, securityFile, "security_helpers.go should not be generated when security is disabled")
}

// TestGenerateClientWithNoSecuritySchemes tests generation when no security schemes are defined.
func TestGenerateClientWithNoSecuritySchemes(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: No Security Schemes Test
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
      responses:
        '200':
          description: OK
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithSecurity(true),
	)
	require.NoError(t, err)

	securityFile := result.GetFile("security_helpers.go")
	assert.Nil(t, securityFile, "security_helpers.go should not be generated when no security schemes exist")
}

// TestGenerateOAS2ClientWithSecurity tests security generation for OAS 2.0 specs.
func TestGenerateOAS2ClientWithSecurity(t *testing.T) {
	spec := `swagger: "2.0"
info:
  title: OAS 2.0 Security Test
  version: "1.0.0"
basePath: /api/v1
paths:
  /test:
    get:
      operationId: test
      responses:
        '200':
          description: OK
securityDefinitions:
  api_key:
    type: apiKey
    in: header
    name: X-API-Key
  basic_auth:
    type: basic
  oauth2:
    type: oauth2
    flow: accessCode
    authorizationUrl: https://example.com/oauth/authorize
    tokenUrl: https://example.com/oauth/token
    scopes:
      read: Read access
      write: Write access
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "swagger.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithSecurity(true),
	)
	require.NoError(t, err)

	securityFile := result.GetFile("security_helpers.go")
	require.NotNil(t, securityFile, "security_helpers.go should be generated for OAS 2.0")

	content := string(securityFile.Content)
	assert.Contains(t, content, "ApiKey")
	assert.Contains(t, content, "BasicAuth")
	assert.Contains(t, content, "Oauth2")

	// Verify it compiles
	fset := token.NewFileSet()
	_, err = goparser.ParseFile(fset, "security_helpers.go", securityFile.Content, goparser.AllErrors)
	assert.NoError(t, err)
}

// TestGenerateClientWithOAuth2Flows tests OAuth2 flow helper generation.
func TestGenerateClientWithOAuth2Flows(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: OAuth2 Flows Test
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
      responses:
        '200':
          description: OK
components:
  securitySchemes:
    oauth2:
      type: oauth2
      flows:
        authorizationCode:
          authorizationUrl: https://example.com/oauth/authorize
          tokenUrl: https://example.com/oauth/token
          refreshUrl: https://example.com/oauth/refresh
          scopes:
            read: Read access
            write: Write access
        clientCredentials:
          tokenUrl: https://example.com/oauth/token
          scopes:
            api: API access
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithSecurity(true),
		WithOAuth2Flows(true),
	)
	require.NoError(t, err)

	// Check for oauth2_{scheme_name}.go - scheme is named "oauth2" so file is oauth2_oauth2.go
	oauth2File := result.GetFile("oauth2_oauth2.go")
	require.NotNil(t, oauth2File, "oauth2_oauth2.go should be generated")

	content := string(oauth2File.Content)
	assert.Contains(t, content, "OAuth2Config")
	assert.Contains(t, content, "OAuth2Token")
	assert.Contains(t, content, "OAuth2Client")
	assert.Contains(t, content, "GetAuthorizationURL")
	assert.Contains(t, content, "ExchangeCode")
	assert.Contains(t, content, "ClientCredentials")
	assert.Contains(t, content, "RefreshToken")

	// Verify it compiles
	fset := token.NewFileSet()
	_, err = goparser.ParseFile(fset, "oauth2_oauth2.go", oauth2File.Content, goparser.AllErrors)
	assert.NoError(t, err, "oauth2_oauth2.go should be valid Go syntax")
}

// TestGenerateClientWithCredentialMgmt tests credential management generation.
func TestGenerateClientWithCredentialMgmt(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Credential Mgmt Test
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
      responses:
        '200':
          description: OK
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithSecurity(true),
		WithCredentialMgmt(true),
	)
	require.NoError(t, err)

	// Check for credentials.go
	credFile := result.GetFile("credentials.go")
	require.NotNil(t, credFile, "credentials.go should be generated")

	content := string(credFile.Content)
	assert.Contains(t, content, "CredentialProvider")
	assert.Contains(t, content, "MemoryCredentialProvider")
	assert.Contains(t, content, "EnvCredentialProvider")
	assert.Contains(t, content, "CredentialChain")
	assert.Contains(t, content, "WithCredentialProvider")

	// Verify it compiles
	fset := token.NewFileSet()
	_, err = goparser.ParseFile(fset, "credentials.go", credFile.Content, goparser.AllErrors)
	assert.NoError(t, err, "credentials.go should be valid Go syntax")
}

// TestGenerateClientWithSecurityEnforce tests security enforcement generation.
func TestGenerateClientWithSecurityEnforce(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Security Enforce Test
  version: "1.0.0"
paths:
  /public:
    get:
      operationId: publicEndpoint
      responses:
        '200':
          description: OK
  /private:
    get:
      operationId: privateEndpoint
      security:
        - apiKey: []
      responses:
        '200':
          description: OK
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
security:
  - apiKey: []
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithSecurity(true),
		WithSecurityEnforce(true),
	)
	require.NoError(t, err)

	// Check for security_enforce.go
	enforceFile := result.GetFile("security_enforce.go")
	require.NotNil(t, enforceFile, "security_enforce.go should be generated")

	content := string(enforceFile.Content)
	assert.Contains(t, content, "SecurityRequirement")
	assert.Contains(t, content, "SecurityValidator")
	assert.Contains(t, content, "OperationSecurity")

	// Verify it compiles
	fset := token.NewFileSet()
	_, err = goparser.ParseFile(fset, "security_enforce.go", enforceFile.Content, goparser.AllErrors)
	assert.NoError(t, err, "security_enforce.go should be valid Go syntax")
}

// TestGenerateClientWithOIDCDiscovery tests OIDC discovery generation.
func TestGenerateClientWithOIDCDiscovery(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: OIDC Discovery Test
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
      responses:
        '200':
          description: OK
components:
  securitySchemes:
    oidc:
      type: openIdConnect
      openIdConnectUrl: https://auth.example.com/.well-known/openid-configuration
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithSecurity(true),
		WithOIDCDiscovery(true),
	)
	require.NoError(t, err)

	// Check for oidc_discovery.go
	oidcFile := result.GetFile("oidc_discovery.go")
	require.NotNil(t, oidcFile, "oidc_discovery.go should be generated")

	content := string(oidcFile.Content)
	assert.Contains(t, content, "OIDCConfiguration")
	assert.Contains(t, content, "OIDCDiscoveryClient")
	assert.Contains(t, content, "GetConfiguration")
	assert.Contains(t, content, "Issuer")
	assert.Contains(t, content, "TokenEndpoint")

	// Verify it compiles
	fset := token.NewFileSet()
	_, err = goparser.ParseFile(fset, "oidc_discovery.go", oidcFile.Content, goparser.AllErrors)
	assert.NoError(t, err, "oidc_discovery.go should be valid Go syntax")
}

// TestGenerateClientWithReadme tests README generation.
func TestGenerateClientWithReadme(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: README Test API
  version: "2.0.0"
  description: An API for testing README generation
paths:
  /items:
    get:
      operationId: listItems
      responses:
        '200':
          description: OK
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
    bearerAuth:
      type: http
      scheme: bearer
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithSecurity(true),
		WithReadme(true),
	)
	require.NoError(t, err)

	// Check for README.md
	readmeFile := result.GetFile("README.md")
	require.NotNil(t, readmeFile, "README.md should be generated")

	content := string(readmeFile.Content)
	assert.Contains(t, content, "README Test API")
	assert.Contains(t, content, "2.0.0")
	assert.Contains(t, content, "testapi")
	assert.Contains(t, content, "Security")
	assert.Contains(t, content, "apiKey")
	assert.Contains(t, content, "bearerAuth")
}

// TestGenerateClientWithoutReadme tests that README is not generated when disabled.
func TestGenerateClientWithoutReadme(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: No README Test
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
      responses:
        '200':
          description: OK
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithReadme(false),
	)
	require.NoError(t, err)

	readmeFile := result.GetFile("README.md")
	assert.Nil(t, readmeFile, "README.md should not be generated when disabled")
}

// TestGenerateAllSecurityFeatures tests generation with all security features enabled.
func TestGenerateAllSecurityFeatures(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Full Security Test
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
      security:
        - oauth2: [read]
      responses:
        '200':
          description: OK
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
    oauth2:
      type: oauth2
      flows:
        authorizationCode:
          authorizationUrl: https://example.com/oauth/authorize
          tokenUrl: https://example.com/oauth/token
          scopes:
            read: Read access
    oidc:
      type: openIdConnect
      openIdConnectUrl: https://example.com/.well-known/openid-configuration
security:
  - apiKey: []
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithServer(true),
		WithSecurity(true),
		WithOAuth2Flows(true),
		WithCredentialMgmt(true),
		WithSecurityEnforce(true),
		WithOIDCDiscovery(true),
		WithReadme(true),
	)
	require.NoError(t, err)

	// Verify all expected files are generated
	// Note: oauth2 file is named oauth2_{scheme_name}.go
	expectedFiles := []string{
		"types.go",
		"client.go",
		"server.go",
		"security_helpers.go",
		"oauth2_oauth2.go",
		"credentials.go",
		"security_enforce.go",
		"oidc_discovery.go",
		"README.md",
	}

	for _, filename := range expectedFiles {
		file := result.GetFile(filename)
		assert.NotNil(t, file, "%s should be generated", filename)

		// Verify Go files compile
		if strings.HasSuffix(filename, ".go") {
			fset := token.NewFileSet()
			_, err := goparser.ParseFile(fset, filename, file.Content, goparser.AllErrors)
			assert.NoError(t, err, "%s should be valid Go syntax", filename)
		}
	}
}

// TestGenerateSecurityWithSpecialCharacters tests security scheme names with special characters.
func TestGenerateSecurityWithSpecialCharacters(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Special Characters Test
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
      responses:
        '200':
          description: OK
components:
  securitySchemes:
    "api-key_v2":
      type: apiKey
      in: header
      name: X-API-Key
    "my.oauth.scheme":
      type: http
      scheme: bearer
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithSecurity(true),
	)
	require.NoError(t, err)

	securityFile := result.GetFile("security_helpers.go")
	require.NotNil(t, securityFile)

	content := string(securityFile.Content)
	// Should sanitize names to valid Go identifiers
	assert.Contains(t, content, "ApiKeyV2")
	assert.Contains(t, content, "MyOauthScheme")

	// Verify it compiles
	fset := token.NewFileSet()
	_, err = goparser.ParseFile(fset, "security_helpers.go", securityFile.Content, goparser.AllErrors)
	assert.NoError(t, err)
}

// TestGenerateSecurityServerOnly tests that security is not generated for server-only generation.
func TestGenerateSecurityServerOnly(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Server Only Test
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
      responses:
        '200':
          description: OK
components:
  securitySchemes:
    apiKey:
      type: apiKey
      in: header
      name: X-API-Key
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(false),
		WithServer(true),
		WithSecurity(true),
	)
	require.NoError(t, err)

	// Security helpers are for client-side authentication
	securityFile := result.GetFile("security_helpers.go")
	assert.Nil(t, securityFile, "security_helpers.go should not be generated for server-only")

	// Server file should still be generated
	serverFile := result.GetFile("server.go")
	assert.NotNil(t, serverFile, "server.go should be generated")
}
