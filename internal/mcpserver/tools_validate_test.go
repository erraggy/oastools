package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTool_ValidSpec(t *testing.T) {
	content := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
`
	input := validateInput{
		Spec: specInput{Content: content},
	}
	_, output, err := handleValidate(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.True(t, output.Valid)
	assert.Empty(t, output.Errors)
}

func TestValidateTool_InvalidSpec(t *testing.T) {
	content := `openapi: "3.0.0"
info:
  title: Test API
paths: {}
`
	input := validateInput{
		Spec: specInput{Content: content},
	}
	_, output, err := handleValidate(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.False(t, output.Valid)
	assert.NotEmpty(t, output.Errors)
}
