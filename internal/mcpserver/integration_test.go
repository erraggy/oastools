package mcpserver

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minimalOAS31 is a minimal valid OpenAPI 3.1 spec used across integration tests.
const minimalOAS31 = `{
  "openapi": "3.1.0",
  "info": {"title": "Test API", "version": "1.0.0"},
  "paths": {
    "/pets": {
      "get": {
        "operationId": "listPets",
        "summary": "List all pets",
        "tags": ["pets"],
        "responses": {"200": {"description": "OK"}}
      },
      "post": {
        "operationId": "createPet",
        "summary": "Create a pet",
        "tags": ["pets"],
        "responses": {"201": {"description": "Created"}}
      }
    },
    "/pets/{petId}": {
      "get": {
        "operationId": "getPet",
        "summary": "Get a pet by ID",
        "tags": ["pets"],
        "parameters": [{"name": "petId", "in": "path", "required": true, "schema": {"type": "string"}}],
        "responses": {"200": {"description": "OK"}}
      }
    }
  },
  "components": {
    "schemas": {
      "Pet": {
        "type": "object",
        "properties": {
          "id": {"type": "integer"},
          "name": {"type": "string"}
        }
      }
    },
    "securitySchemes": {
      "bearerAuth": {
        "type": "http",
        "scheme": "bearer"
      }
    }
  }
}`

// startTestSession creates an in-process MCP server/client pair and returns
// the connected client session. The server is shut down when the test ends.
func startTestSession(t *testing.T) *mcp.ClientSession {
	t.Helper()

	server := mcp.NewServer(
		&mcp.Implementation{Name: "oastools-test", Version: "test"},
		nil,
	)
	registerAllTools(server)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	// Start server in background — it blocks until the connection closes.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	done := make(chan error, 1)
	go func() {
		done <- server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(
		&mcp.Implementation{Name: "test-client", Version: "test"},
		nil,
	)
	session, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = session.Close()
		cancel()
		<-done
	})

	return session
}

func TestIntegration_ListTools(t *testing.T) {
	session := startTestSession(t)

	result, err := session.ListTools(context.Background(), &mcp.ListToolsParams{})
	require.NoError(t, err)
	require.NotNil(t, result)

	// We expect 15 tools: 9 core + 6 walk.
	assert.Len(t, result.Tools, 15, "expected 15 registered tools")

	// Collect tool names and verify expected ones are present.
	names := make([]string, 0, len(result.Tools))
	for _, tool := range result.Tools {
		names = append(names, tool.Name)
	}

	expectedTools := []string{
		"validate",
		"parse",
		"fix",
		"convert",
		"diff",
		"join",
		"overlay_apply",
		"overlay_validate",
		"generate",
		"walk_operations",
		"walk_schemas",
		"walk_parameters",
		"walk_responses",
		"walk_security",
		"walk_paths",
	}

	for _, name := range expectedTools {
		assert.True(t, slices.Contains(names, name), "missing tool: %s", name)
	}

	// Every tool should have a non-empty description.
	for _, tool := range result.Tools {
		assert.NotEmpty(t, tool.Description, "tool %q has empty description", tool.Name)
	}
}

func TestIntegration_CallTool_Validate(t *testing.T) {
	session := startTestSession(t)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "validate",
		Arguments: map[string]any{
			"spec": map[string]any{
				"content": minimalOAS31,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "validate should succeed on valid spec")

	// The structured output should contain version and valid fields.
	structured := unmarshalStructured(t, result)
	assert.Equal(t, true, structured["valid"])
	assert.Equal(t, "3.1.0", structured["version"])
	assert.Equal(t, float64(0), structured["error_count"])
}

func TestIntegration_CallTool_Parse(t *testing.T) {
	session := startTestSession(t)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "parse",
		Arguments: map[string]any{
			"spec": map[string]any{
				"content": minimalOAS31,
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "parse should succeed on valid spec")

	structured := unmarshalStructured(t, result)
	assert.Equal(t, "3.1.0", structured["version"])
	assert.Equal(t, "Test API", structured["title"])
	assert.Equal(t, float64(2), structured["path_count"])
	assert.Equal(t, float64(3), structured["operation_count"])
	assert.Equal(t, float64(1), structured["schema_count"])
}

func TestIntegration_CallTool_WalkOperations(t *testing.T) {
	session := startTestSession(t)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "walk_operations",
		Arguments: map[string]any{
			"spec": map[string]any{
				"content": minimalOAS31,
			},
			"method": "get",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsError, "walk_operations should succeed")

	structured := unmarshalStructured(t, result)
	assert.Equal(t, float64(3), structured["total"])
	assert.Equal(t, float64(2), structured["matched"]) // 2 GET operations

	summaries, ok := structured["summaries"].([]any)
	require.True(t, ok, "summaries should be an array")
	assert.Len(t, summaries, 2)

	// Verify both GET operations are returned.
	operationIDs := make([]string, 0, len(summaries))
	for _, s := range summaries {
		m, ok := s.(map[string]any)
		require.True(t, ok, "expected summary to be map[string]any, got %T", s)
		opID, ok := m["operation_id"].(string)
		require.True(t, ok, "expected operation_id to be string, got %T", m["operation_id"])
		operationIDs = append(operationIDs, opID)
	}
	assert.Contains(t, operationIDs, "listPets")
	assert.Contains(t, operationIDs, "getPet")
}

func TestIntegration_CallTool_Error_InvalidSpec(t *testing.T) {
	session := startTestSession(t)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "validate",
		Arguments: map[string]any{
			"spec": map[string]any{
				"content": "this is not valid JSON or YAML for an OAS spec",
			},
		},
	})
	require.NoError(t, err, "MCP protocol call should succeed even on tool error")
	require.NotNil(t, result)
	assert.True(t, result.IsError, "validate should return IsError for unparseable input")

	// The error content should contain descriptive text.
	require.NotEmpty(t, result.Content)
	text, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok, "error content should be TextContent")
	assert.NotEmpty(t, text.Text)
}

func TestIntegration_CallTool_Error_MissingSpec(t *testing.T) {
	session := startTestSession(t)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "parse",
		Arguments: map[string]any{
			"spec": map[string]any{},
		},
	})
	require.NoError(t, err, "MCP protocol call should succeed even on tool error")
	require.NotNil(t, result)
	assert.True(t, result.IsError, "parse should return IsError when no spec source is provided")
}

// unmarshalStructured extracts the structured output from a CallToolResult.
// It first checks StructuredContent, then falls back to parsing the first TextContent.
func unmarshalStructured(t *testing.T, result *mcp.CallToolResult) map[string]any {
	t.Helper()

	// Prefer structured content if available.
	if result.StructuredContent != nil {
		data, err := json.Marshal(result.StructuredContent)
		require.NoError(t, err)
		var m map[string]any
		require.NoError(t, json.Unmarshal(data, &m))
		return m
	}

	// Fall back to parsing text content.
	require.NotEmpty(t, result.Content, "expected at least one content item")
	text, ok := result.Content[0].(*mcp.TextContent)
	require.True(t, ok, "expected TextContent, got %T", result.Content[0])

	var m map[string]any
	require.NoError(t, json.Unmarshal([]byte(text.Text), &m), "failed to parse text content as JSON")
	return m
}
