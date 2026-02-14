package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const overlayTestSpec = `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
`

const overlayTestOverlay = `overlay: "1.0.0"
info:
  title: Update Title
  version: "1.0"
actions:
  - target: "$.info.title"
    update: "Updated API"
`

const overlayTestSpecWithDescription = `openapi: "3.0.0"
info:
  title: Test API
  description: "This should be removed"
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        "200":
          description: OK
`

const overlayTestRemoveOverlay = `overlay: "1.0.0"
info:
  title: Remove Description
  version: "1.0"
actions:
  - target: "$.info.description"
    remove: true
`

const overlayTestInvalidOverlay = `overlay: "2.0.0"
info:
  title: ""
  version: ""
actions: []
`

const overlayTestMissingFieldsOverlay = `overlay: ""
info:
  title: ""
  version: ""
actions: []
`

func TestOverlayApplyTool_UpdateTitle(t *testing.T) {
	input := overlayApplyInput{
		Spec:    specInput{Content: overlayTestSpec},
		Overlay: specInput{Content: overlayTestOverlay},
	}
	_, output, err := handleOverlayApply(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.Equal(t, 1, output.ActionsApplied)
	assert.Equal(t, 0, output.ActionsSkipped)
	assert.NotEmpty(t, output.Changes)
	assert.NotEmpty(t, output.Document)
	assert.Contains(t, output.Document, "Updated API")
	assert.Contains(t, output.Summary, "1 action applied")
}

func TestOverlayApplyTool_RemoveField(t *testing.T) {
	input := overlayApplyInput{
		Spec:    specInput{Content: overlayTestSpecWithDescription},
		Overlay: specInput{Content: overlayTestRemoveOverlay},
	}
	_, output, err := handleOverlayApply(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.Equal(t, 1, output.ActionsApplied)
	assert.NotEmpty(t, output.Document)
	// The description field should be removed from the output.
	assert.NotContains(t, output.Document, "This should be removed")

	// Verify the change record.
	require.Len(t, output.Changes, 1)
	assert.Equal(t, "remove", output.Changes[0].Operation)
	assert.Equal(t, "$.info.description", output.Changes[0].Target)
}

func TestOverlayApplyTool_DryRun(t *testing.T) {
	input := overlayApplyInput{
		Spec:    specInput{Content: overlayTestSpec},
		Overlay: specInput{Content: overlayTestOverlay},
		DryRun:  true,
	}
	_, output, err := handleOverlayApply(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.Equal(t, 1, output.ActionsApplied)
	assert.NotEmpty(t, output.Changes)
	// Dry run should not return a document.
	assert.Empty(t, output.Document)
	assert.Empty(t, output.WrittenTo)
	assert.Contains(t, output.Summary, "dry run")
}

func TestOverlayApplyTool_OutputFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "result.yaml")

	input := overlayApplyInput{
		Spec:    specInput{Content: overlayTestSpec},
		Overlay: specInput{Content: overlayTestOverlay},
		Output:  outPath,
	}
	_, output, err := handleOverlayApply(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.Equal(t, outPath, output.WrittenTo)
	assert.Empty(t, output.Document, "document should not be inline when written to file")

	// Verify the file was written and contains the updated title.
	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "Updated API")
}

func TestOverlayApplyTool_InvalidSpec(t *testing.T) {
	input := overlayApplyInput{
		Spec:    specInput{Content: "not valid yaml: ["},
		Overlay: specInput{Content: overlayTestOverlay},
	}
	result, output, err := handleOverlayApply(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.Document)
}

func TestOverlayApplyTool_InvalidOverlay(t *testing.T) {
	input := overlayApplyInput{
		Spec:    specInput{Content: overlayTestSpec},
		Overlay: specInput{Content: "not valid yaml: ["},
	}
	result, output, err := handleOverlayApply(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.Document)
}

func TestOverlayApplyTool_MissingInput(t *testing.T) {
	input := overlayApplyInput{
		Spec:    specInput{},
		Overlay: specInput{Content: overlayTestOverlay},
	}
	result, output, err := handleOverlayApply(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.Document)
}

func TestOverlayApplyTool_MissingOverlayInput(t *testing.T) {
	input := overlayApplyInput{
		Spec:    specInput{Content: overlayTestSpec},
		Overlay: specInput{},
	}
	result, output, err := handleOverlayApply(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.Document)
}

func TestOverlayApplyTool_InvalidOutputPath(t *testing.T) {
	input := overlayApplyInput{
		Spec:    specInput{Content: overlayTestSpec},
		Overlay: specInput{Content: overlayTestOverlay},
		Output:  "/nonexistent/dir/file.yaml",
	}
	result, output, err := handleOverlayApply(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.WrittenTo)
}

// overlay_validate tests

func TestOverlayValidateTool_ValidOverlay(t *testing.T) {
	input := overlayValidateInput{
		Overlay: specInput{Content: overlayTestOverlay},
	}
	_, output, err := handleOverlayValidate(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.True(t, output.Valid)
	assert.Equal(t, 0, output.ErrorCount)
	assert.Empty(t, output.Errors)
}

func TestOverlayValidateTool_InvalidVersion(t *testing.T) {
	input := overlayValidateInput{
		Overlay: specInput{Content: overlayTestInvalidOverlay},
	}
	_, output, err := handleOverlayValidate(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.False(t, output.Valid)
	assert.Greater(t, output.ErrorCount, 0)
	assert.NotEmpty(t, output.Errors)

	// Should report unsupported version.
	found := false
	for _, e := range output.Errors {
		if e.Field == "overlay" {
			found = true
			assert.Contains(t, e.Message, "unsupported version")
			break
		}
	}
	assert.True(t, found, "expected a version validation error")
}

func TestOverlayValidateTool_MissingFields(t *testing.T) {
	input := overlayValidateInput{
		Overlay: specInput{Content: overlayTestMissingFieldsOverlay},
	}
	_, output, err := handleOverlayValidate(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.False(t, output.Valid)
	// Should have errors for: overlay version, info.title, info.version, actions.
	assert.Equal(t, 4, output.ErrorCount)
}

func TestOverlayValidateTool_MissingInput(t *testing.T) {
	input := overlayValidateInput{
		Overlay: specInput{},
	}
	result, output, err := handleOverlayValidate(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.Errors)
}

func TestOverlayValidateTool_InvalidYAML(t *testing.T) {
	input := overlayValidateInput{
		Overlay: specInput{Content: "not valid yaml: ["},
	}
	result, output, err := handleOverlayValidate(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.Errors)
}
