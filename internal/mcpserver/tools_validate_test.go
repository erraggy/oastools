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

func TestValidateTool_Pagination(t *testing.T) {
	// This spec has multiple validation errors (missing info fields and responses).
	content := `openapi: "3.0.0"
info: {}
paths:
  /a:
    get: {}
  /b:
    post: {}
  /c:
    put: {}
`
	// Baseline: get total error count without pagination.
	input := validateInput{
		Spec: specInput{Content: content},
	}
	_, baseline, err := handleValidate(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	require.False(t, baseline.Valid)
	require.Greater(t, baseline.ErrorCount, 2, "need at least 3 errors for pagination test")

	t.Run("limit", func(t *testing.T) {
		_, output, err := handleValidate(context.Background(), &mcp.CallToolRequest{}, validateInput{
			Spec:       specInput{Content: content},
			NoWarnings: true,
			Limit:      1,
		})
		require.NoError(t, err)
		assert.Equal(t, baseline.ErrorCount, output.ErrorCount)
		assert.Equal(t, 1, output.Returned)
		assert.Len(t, output.Errors, 1)
	})

	t.Run("offset", func(t *testing.T) {
		_, output, err := handleValidate(context.Background(), &mcp.CallToolRequest{}, validateInput{
			Spec:       specInput{Content: content},
			NoWarnings: true,
			Offset:     1,
		})
		require.NoError(t, err)
		assert.Equal(t, baseline.ErrorCount, output.ErrorCount)
		assert.Equal(t, baseline.ErrorCount-1, output.Returned)
	})

	t.Run("offset and limit", func(t *testing.T) {
		_, output, err := handleValidate(context.Background(), &mcp.CallToolRequest{}, validateInput{
			Spec:       specInput{Content: content},
			NoWarnings: true,
			Offset:     1,
			Limit:      2,
		})
		require.NoError(t, err)
		assert.Equal(t, baseline.ErrorCount, output.ErrorCount)
		assert.Equal(t, 2, output.Returned)
		assert.Len(t, output.Errors, 2)
	})

	t.Run("offset beyond total", func(t *testing.T) {
		_, output, err := handleValidate(context.Background(), &mcp.CallToolRequest{}, validateInput{
			Spec:       specInput{Content: content},
			NoWarnings: true,
			Offset:     baseline.ErrorCount,
		})
		require.NoError(t, err)
		assert.Equal(t, baseline.ErrorCount, output.ErrorCount)
		assert.Equal(t, 0, output.Returned)
		assert.Nil(t, output.Errors)
	})
}
