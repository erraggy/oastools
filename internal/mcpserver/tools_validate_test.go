package mcpserver

import (
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool { return &b }

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
			NoWarnings: boolPtr(true),
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
			NoWarnings: boolPtr(true),
			Offset:     1,
		})
		require.NoError(t, err)
		assert.Equal(t, baseline.ErrorCount, output.ErrorCount)
		assert.Equal(t, baseline.ErrorCount-1, output.Returned)
	})

	t.Run("offset and limit", func(t *testing.T) {
		_, output, err := handleValidate(context.Background(), &mcp.CallToolRequest{}, validateInput{
			Spec:       specInput{Content: content},
			NoWarnings: boolPtr(true),
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
			NoWarnings: boolPtr(true),
			Offset:     baseline.ErrorCount,
		})
		require.NoError(t, err)
		assert.Equal(t, baseline.ErrorCount, output.ErrorCount)
		assert.Equal(t, 0, output.Returned)
		assert.Nil(t, output.Errors)
	})
}

func TestHandleValidate_ConfigDefaults(t *testing.T) {
	specCache.reset()
	origCfg := cfg
	cfg = &serverConfig{
		CacheEnabled:       true,
		CacheMaxSize:       10,
		CacheFileTTL:       15 * time.Minute,
		CacheURLTTL:        5 * time.Minute,
		CacheContentTTL:    15 * time.Minute,
		CacheSweepInterval: 60 * time.Second,
		WalkLimit:          100,
		WalkDetailLimit:    25,
		MaxInlineSize:      10 * 1024 * 1024,
		ValidateStrict:     true,
		ValidateNoWarnings: true,
	}
	t.Cleanup(func() { cfg = origCfg })

	t.Run("config defaults apply when input omitted", func(t *testing.T) {
		input := validateInput{
			Spec: specInput{File: "../../testdata/petstore-3.0.yaml"},
		}
		_, output, err := handleValidate(context.Background(), &mcp.CallToolRequest{}, input)
		require.NoError(t, err)
		// With no_warnings=true from config, warnings should be suppressed.
		assert.Empty(t, output.Warnings)
		assert.Equal(t, 0, output.WarningCount)
	})

	t.Run("explicit false overrides config true", func(t *testing.T) {
		// Use a spec that produces warnings (missing description, etc.).
		specWithWarnings := `openapi: "3.0.0"
info:
  title: Warn Test
  version: "1.0"
paths:
  /a:
    get:
      operationId: getA
      responses:
        "200":
          description: OK
`
		// First, verify the spec actually has warnings with no_warnings=false.
		baseCfg := cfg
		cfg = &serverConfig{
			CacheEnabled:       false,
			WalkLimit:          100,
			WalkDetailLimit:    25,
			MaxInlineSize:      10 * 1024 * 1024,
			ValidateStrict:     true,
			ValidateNoWarnings: false,
		}
		_, baseOutput, err := handleValidate(context.Background(), &mcp.CallToolRequest{}, validateInput{
			Spec: specInput{Content: specWithWarnings},
		})
		require.NoError(t, err)
		cfg = baseCfg

		if baseOutput.WarningCount == 0 {
			t.Skip("test spec produces no warnings; cannot test override")
		}

		// Now test: cfg has NoWarnings=true, but explicit false should override.
		input := validateInput{
			Spec:       specInput{Content: specWithWarnings},
			NoWarnings: boolPtr(false),
		}
		_, output, err := handleValidate(context.Background(), &mcp.CallToolRequest{}, input)
		require.NoError(t, err)
		assert.Greater(t, output.WarningCount, 0, "explicit false should override config true")
	})
}
