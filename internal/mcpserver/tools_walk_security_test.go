package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const walkSecurityTestSpec = `openapi: "3.0.0"
info:
  title: Walk Security Test
  version: "1.0.0"
paths:
  /pets:
    get:
      summary: List pets
      responses:
        "200":
          description: OK
components:
  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      name: X-API-Key
      in: header
      description: API key authentication
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: Bearer token authentication
    OAuth2:
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

func callWalkSecurity(t *testing.T, input walkSecurityInput) (*mcp.CallToolResult, walkSecurityOutput) {
	t.Helper()
	result, out, err := handleWalkSecurity(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	if out == nil {
		return result, walkSecurityOutput{}
	}
	wo, ok := out.(walkSecurityOutput)
	require.True(t, ok, "expected walkSecurityOutput, got %T", out)
	return result, wo
}

func TestWalkSecurity_AllSchemes(t *testing.T) {
	input := walkSecurityInput{
		Spec: specInput{Content: walkSecurityTestSpec},
	}
	_, output := callWalkSecurity(t, input)

	assert.Equal(t, 3, output.Total)
	assert.Equal(t, 3, output.Matched)
	assert.Equal(t, 3, output.Returned)
	require.Len(t, output.Summaries, 3)
}

func TestWalkSecurity_FilterByName(t *testing.T) {
	input := walkSecurityInput{
		Spec: specInput{Content: walkSecurityTestSpec},
		Name: "ApiKeyAuth",
	}
	_, output := callWalkSecurity(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "ApiKeyAuth", output.Summaries[0].Name)
	assert.Equal(t, "apiKey", output.Summaries[0].Type)
	assert.Equal(t, "header", output.Summaries[0].In)
	assert.Equal(t, "API key authentication", output.Summaries[0].Description)
}

func TestWalkSecurity_FilterByType(t *testing.T) {
	input := walkSecurityInput{
		Spec: specInput{Content: walkSecurityTestSpec},
		Type: "http",
	}
	_, output := callWalkSecurity(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "BearerAuth", output.Summaries[0].Name)
	assert.Equal(t, "http", output.Summaries[0].Type)
}

func TestWalkSecurity_FilterByTypeOAuth2(t *testing.T) {
	input := walkSecurityInput{
		Spec: specInput{Content: walkSecurityTestSpec},
		Type: "oauth2",
	}
	_, output := callWalkSecurity(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "OAuth2", output.Summaries[0].Name)
	assert.Equal(t, "oauth2", output.Summaries[0].Type)
}

func TestWalkSecurity_DetailMode(t *testing.T) {
	input := walkSecurityInput{
		Spec:   specInput{Content: walkSecurityTestSpec},
		Name:   "BearerAuth",
		Detail: true,
	}
	_, output := callWalkSecurity(t, input)

	assert.Equal(t, 1, output.Matched)
	assert.Nil(t, output.Summaries)
	require.Len(t, output.Schemes, 1)
	assert.Equal(t, "BearerAuth", output.Schemes[0].Name)
	assert.NotNil(t, output.Schemes[0].SecurityScheme)
	assert.Equal(t, "http", output.Schemes[0].SecurityScheme.Type)
	assert.Equal(t, "bearer", output.Schemes[0].SecurityScheme.Scheme)
	assert.Equal(t, "JWT", output.Schemes[0].SecurityScheme.BearerFormat)
}

func TestWalkSecurity_Limit(t *testing.T) {
	input := walkSecurityInput{
		Spec:  specInput{Content: walkSecurityTestSpec},
		Limit: 1,
	}
	_, output := callWalkSecurity(t, input)

	assert.Equal(t, 3, output.Total)
	assert.Equal(t, 3, output.Matched)
	assert.Equal(t, 1, output.Returned)
	assert.Len(t, output.Summaries, 1)
}

func TestWalkSecurity_InvalidSpec(t *testing.T) {
	input := walkSecurityInput{
		Spec: specInput{Content: "not valid yaml: ["},
	}
	result, _ := callWalkSecurity(t, input)
	require.NotNil(t, result)
	assert.True(t, result.IsError)
}

func TestWalkSecurity_NoMatches(t *testing.T) {
	input := walkSecurityInput{
		Spec: specInput{Content: walkSecurityTestSpec},
		Type: "openIdConnect",
	}
	_, output := callWalkSecurity(t, input)

	assert.Equal(t, 0, output.Matched)
	assert.Nil(t, output.Summaries)
}

func TestWalkSecurity_FilterByExtension(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Extension Test
  version: "1.0.0"
paths: {}
components:
  securitySchemes:
    InternalAuth:
      type: apiKey
      name: X-Internal
      in: header
      x-internal: true
    PublicAuth:
      type: http
      scheme: bearer
`
	input := walkSecurityInput{
		Spec:      specInput{Content: spec},
		Extension: "x-internal=true",
	}
	_, output := callWalkSecurity(t, input)

	assert.Equal(t, 1, output.Matched)
	require.Len(t, output.Summaries, 1)
	assert.Equal(t, "InternalAuth", output.Summaries[0].Name)
}
