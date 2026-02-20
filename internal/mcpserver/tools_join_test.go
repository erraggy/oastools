package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const joinSpecA = `openapi: "3.0.0"
info:
  title: Spec A
  version: "1.0.0"
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        "200":
          description: OK
components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
`

const joinSpecB = `openapi: "3.0.0"
info:
  title: Spec B
  version: "1.0.0"
paths:
  /orders:
    get:
      operationId: listOrders
      responses:
        "200":
          description: OK
components:
  schemas:
    Order:
      type: object
      properties:
        id:
          type: integer
        total:
          type: number
`

func TestJoinTool_TwoSpecs(t *testing.T) {
	input := joinInput{
		Specs: []specInput{
			{Content: joinSpecA},
			{Content: joinSpecB},
		},
	}
	_, output, err := handleJoin(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.Equal(t, 2, output.SpecCount)
	assert.Contains(t, output.Version, "3.0")
	assert.Equal(t, 2, output.PathCount, "merged document should have 2 paths")
	assert.Equal(t, 2, output.SchemaCount, "merged document should have 2 schemas")
	assert.NotEmpty(t, output.Document, "document should be returned inline")
	assert.Empty(t, output.WrittenTo)

	// The merged document should contain paths and schemas from both specs.
	assert.Contains(t, output.Document, "/users")
	assert.Contains(t, output.Document, "/orders")
	assert.Contains(t, output.Document, "User")
	assert.Contains(t, output.Document, "Order")

	assert.Contains(t, output.Summary, "Joined 2 specs")
	assert.Contains(t, output.Summary, "2 paths")
	assert.Contains(t, output.Summary, "2 schemas")
}

func TestJoinTool_OutputFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "merged.yaml")

	input := joinInput{
		Specs: []specInput{
			{Content: joinSpecA},
			{Content: joinSpecB},
		},
		Output: outPath,
	}
	_, output, err := handleJoin(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.Equal(t, outPath, output.WrittenTo)
	assert.Empty(t, output.Document, "document should not be inline when written to file")

	// Verify the file was written and contains the merged spec.
	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "/users")
	assert.Contains(t, content, "/orders")
	assert.Contains(t, content, "Spec A")
}

func TestJoinTool_TooFewSpecs(t *testing.T) {
	input := joinInput{
		Specs: []specInput{
			{Content: joinSpecA},
		},
	}
	result, output, err := handleJoin(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.Document)
}

func TestJoinTool_InvalidSpec(t *testing.T) {
	input := joinInput{
		Specs: []specInput{
			{Content: joinSpecA},
			{Content: "not valid yaml: ["},
		},
	}
	result, output, err := handleJoin(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.Document)
}

func TestJoinTool_PathCollisionFail(t *testing.T) {
	// Both specs define /users, which should fail with "fail" path strategy.
	specWithUsers := `openapi: "3.0.0"
info:
  title: Another Users Spec
  version: "1.0.0"
paths:
  /users:
    post:
      operationId: createUser
      responses:
        "201":
          description: Created
`
	input := joinInput{
		Specs: []specInput{
			{Content: joinSpecA},
			{Content: specWithUsers},
		},
		PathStrategy: "fail",
	}
	result, output, err := handleJoin(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.Document)
}

func TestJoinTool_PathCollisionAcceptLeft(t *testing.T) {
	specWithUsers := `openapi: "3.0.0"
info:
  title: Another Users Spec
  version: "1.0.0"
paths:
  /users:
    post:
      operationId: createUser
      responses:
        "201":
          description: Created
`
	input := joinInput{
		Specs: []specInput{
			{Content: joinSpecA},
			{Content: specWithUsers},
		},
		PathStrategy: "accept-left",
	}
	_, output, err := handleJoin(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.Equal(t, 2, output.SpecCount)
	assert.NotEmpty(t, output.Document)
	assert.Contains(t, output.Document, "/users")
}

func TestJoinTool_NoSpecs(t *testing.T) {
	input := joinInput{
		Specs: []specInput{},
	}
	result, output, err := handleJoin(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.Document)
}

func TestJoinTool_InvalidOutputPath(t *testing.T) {
	input := joinInput{
		Specs: []specInput{
			{Content: joinSpecA},
			{Content: joinSpecB},
		},
		Output: "/nonexistent/dir/file.yaml",
	}
	result, output, err := handleJoin(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.WrittenTo)
}

func TestHandleJoin_ConfigDefaults(t *testing.T) {
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
		JoinPathStrategy:   "accept-left",
		JoinSchemaStrategy: "accept-left",
	}
	t.Cleanup(func() { cfg = origCfg })

	// Use specs with a path collision (/users) to prove the config default
	// strategy (accept-left) is applied. Without it, the join would fail.
	specWithUsers := `openapi: "3.0.0"
info:
  title: Another Users Spec
  version: "1.0.0"
paths:
  /users:
    post:
      operationId: createUser
      responses:
        "201":
          description: Created
`
	input := joinInput{
		Specs: []specInput{
			{Content: joinSpecA},     // defines /users (GET)
			{Content: specWithUsers}, // also defines /users (POST)
		},
	}

	// With cfg.JoinPathStrategy="accept-left", the collision should resolve
	// using the left spec's /users rather than failing.
	_, output, err := handleJoin(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.Greater(t, output.PathCount, 0, "join should succeed with config default strategy")
	assert.NotEmpty(t, output.Document, "document should be returned inline")
	assert.Contains(t, output.Document, "/users")
}

func TestJoinTool_MissingSpecInput(t *testing.T) {
	input := joinInput{
		Specs: []specInput{
			{Content: joinSpecA},
			{}, // No file, url, or content
		},
	}
	result, output, err := handleJoin(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.Document)
}
