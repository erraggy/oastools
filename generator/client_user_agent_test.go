package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erraggy/oastools"
	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratedClientDefaultUserAgent_OAS3(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: PetStore
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        '200':
          description: A list of pets
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petstore.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petstore"),
		WithClient(true),
	)
	require.NoError(t, err)

	clientFile := result.GetFile("client.go")
	require.NotNil(t, clientFile, "client.go not generated")

	content := string(clientFile.Content)

	// Check that Client struct has UserAgent field
	assert.Contains(t, content, "UserAgent string", "Client struct should have UserAgent field")

	// Check that NewClient sets default UserAgent
	expectedUserAgent := "oastools/" + oastools.Version() + "/generated/PetStore"
	assert.Contains(t, content, `UserAgent:  "`+expectedUserAgent+`"`, "NewClient should set default UserAgent")

	// Check that WithUserAgent option is generated
	assert.Contains(t, content, "func WithUserAgent(ua string) ClientOption", "WithUserAgent option should be generated")

	// Check that User-Agent header is set in requests
	assert.Contains(t, content, `req.Header.Set("User-Agent", c.UserAgent)`, "User-Agent header should be set in requests")
}

func TestGeneratedClientDefaultUserAgent_OAS2(t *testing.T) {
	spec := `swagger: "2.0"
info:
  title: PetAPI
  version: "2.0.0"
paths:
  /pets:
    get:
      operationId: getPets
      responses:
        '200':
          description: List of pets
definitions:
  Pet:
    type: object
    properties:
      id:
        type: integer
      name:
        type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petapi.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithClient(true),
	)
	require.NoError(t, err)

	clientFile := result.GetFile("client.go")
	require.NotNil(t, clientFile, "client.go not generated")

	content := string(clientFile.Content)

	// Check that Client struct has UserAgent field
	assert.Contains(t, content, "UserAgent string", "Client struct should have UserAgent field")

	// Check that NewClient sets default UserAgent
	expectedUserAgent := "oastools/" + oastools.Version() + "/generated/PetAPI"
	assert.Contains(t, content, `UserAgent:  "`+expectedUserAgent+`"`, "NewClient should set default UserAgent")

	// Check that WithUserAgent option is generated
	assert.Contains(t, content, "func WithUserAgent(ua string) ClientOption", "WithUserAgent option should be generated")

	// Check that User-Agent header is set in requests
	assert.Contains(t, content, `req.Header.Set("User-Agent", c.UserAgent)`, "User-Agent header should be set in requests")
}

func TestGeneratedClientUserAgentWithComplexTitle(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: My Complex API Name
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
	)
	require.NoError(t, err)

	clientFile := result.GetFile("client.go")
	require.NotNil(t, clientFile, "client.go not generated")

	content := string(clientFile.Content)

	// Check that NewClient uses the full title in UserAgent
	expectedUserAgent := "oastools/" + oastools.Version() + "/generated/My Complex API Name"
	assert.Contains(t, content, `UserAgent:  "`+expectedUserAgent+`"`, "NewClient should use full title in UserAgent")
}

func TestGeneratedClientUserAgentWithEmptyTitle(t *testing.T) {
	// Note: Empty title will trigger a parse error in validation,
	// so we skip this test and rely on the unit test for buildDefaultUserAgent
	// which tests the fallback behavior directly
	t.Skip("Empty title triggers parse validation error - tested in TestBuildDefaultUserAgent")
}

func TestGeneratedClientWithUserAgentOption(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
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
	)
	require.NoError(t, err)

	clientFile := result.GetFile("client.go")
	require.NotNil(t, clientFile, "client.go not generated")

	content := string(clientFile.Content)

	// Verify WithUserAgent option structure
	assert.Contains(t, content, "func WithUserAgent(ua string) ClientOption {")
	assert.Contains(t, content, "c.UserAgent = ua")
}

func TestGeneratedClientUserAgentSetInRequests(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Request Test API
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: testGet
      responses:
        '200':
          description: OK
    post:
      operationId: testPost
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
      responses:
        '201':
          description: Created
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
	)
	require.NoError(t, err)

	clientFile := result.GetFile("client.go")
	require.NotNil(t, clientFile, "client.go not generated")

	content := string(clientFile.Content)

	// Count how many times the User-Agent is set (should be once per method)
	userAgentCount := strings.Count(content, `req.Header.Set("User-Agent", c.UserAgent)`)
	assert.GreaterOrEqual(t, userAgentCount, 2, "User-Agent should be set in each generated method")

	// Verify it's set with conditional check
	assert.Contains(t, content, `if c.UserAgent != "" {`)
	assert.Contains(t, content, `req.Header.Set("User-Agent", c.UserAgent)`)
}

func TestBuildDefaultUserAgent(t *testing.T) {
	tests := []struct {
		name     string
		info     *parser.Info
		expected string
	}{
		{
			name:     "with title",
			info:     &parser.Info{Title: "PetStore"},
			expected: "oastools/" + oastools.Version() + "/generated/PetStore",
		},
		{
			name:     "with complex title",
			info:     &parser.Info{Title: "My Complex API"},
			expected: "oastools/" + oastools.Version() + "/generated/My Complex API",
		},
		{
			name:     "with empty title",
			info:     &parser.Info{Title: ""},
			expected: "oastools/" + oastools.Version() + "/generated/API Client",
		},
		{
			name:     "with nil info",
			info:     nil,
			expected: "oastools/" + oastools.Version() + "/generated/API Client",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildDefaultUserAgent(tt.info)
			assert.Equal(t, tt.expected, result)
		})
	}
}
