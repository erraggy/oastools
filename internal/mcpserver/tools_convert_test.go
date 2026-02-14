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

const oas30Spec = `openapi: "3.0.0"
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

func TestConvertTool_OAS30ToOAS31(t *testing.T) {
	input := convertInput{
		Spec:   specInput{Content: oas30Spec},
		Target: "3.1",
	}
	_, output, err := handleConvert(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.True(t, output.Success)
	assert.Equal(t, "3.0.0", output.SourceVersion)
	assert.Contains(t, output.TargetVersion, "3.1")
	assert.NotEmpty(t, output.Document)
	assert.Empty(t, output.WrittenTo)
	// The converted document should contain the 3.1 version marker.
	assert.Contains(t, output.Document, "3.1")
}

func TestConvertTool_OutputFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "converted.yaml")

	input := convertInput{
		Spec:   specInput{Content: oas30Spec},
		Target: "3.1",
		Output: outPath,
	}
	_, output, err := handleConvert(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.True(t, output.Success)
	assert.Equal(t, outPath, output.WrittenTo)
	assert.Empty(t, output.Document, "document should not be inline when written to file")

	// Verify the file was written and contains the converted spec.
	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "3.1")
	assert.Contains(t, string(data), "Test API")
}

func TestConvertTool_FileInput(t *testing.T) {
	input := convertInput{
		Spec:   specInput{File: "../../testdata/petstore-3.0.yaml"},
		Target: "3.1",
	}
	_, output, err := handleConvert(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.True(t, output.Success)
	assert.Contains(t, output.TargetVersion, "3.1")
	assert.NotEmpty(t, output.Document)
}

func TestConvertTool_MissingTarget(t *testing.T) {
	input := convertInput{
		Spec: specInput{Content: oas30Spec},
	}
	result, output, err := handleConvert(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.SourceVersion)
}

func TestConvertTool_InvalidSpec(t *testing.T) {
	input := convertInput{
		Spec:   specInput{Content: "not valid yaml: ["},
		Target: "3.1",
	}
	result, output, err := handleConvert(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.SourceVersion)
}

func TestConvertTool_NoInputProvided(t *testing.T) {
	input := convertInput{
		Spec:   specInput{},
		Target: "3.1",
	}
	result, output, err := handleConvert(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.SourceVersion)
}

func TestConvertTool_InvalidOutputPath(t *testing.T) {
	input := convertInput{
		Spec:   specInput{Content: oas30Spec},
		Target: "3.1",
		Output: "/nonexistent/dir/file.yaml",
	}
	result, output, err := handleConvert(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.WrittenTo)
}

func TestConvertTool_OAS20ToOAS30(t *testing.T) {
	input := convertInput{
		Spec:   specInput{File: "../../testdata/petstore-2.0.yaml"},
		Target: "3.0",
	}
	_, output, err := handleConvert(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.True(t, output.Success)
	assert.Equal(t, "2.0", output.SourceVersion)
	assert.Contains(t, output.TargetVersion, "3.0")
	assert.NotEmpty(t, output.Document)
}
