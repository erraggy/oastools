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

// minimalSpecWithSchemaAndOp is a minimal OAS 3.0 spec with one schema and one
// operation, giving the generator something to produce types and client code from.
const minimalSpecWithSchemaAndOp = `openapi: "3.0.0"
info:
  title: Pet API
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      summary: List all pets
      responses:
        "200":
          description: A list of pets
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: "#/components/schemas/Pet"
components:
  schemas:
    Pet:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: integer
          format: int64
        name:
          type: string
`

func TestGenerateTool_TypesFromSpec(t *testing.T) {
	dir := t.TempDir()

	input := generateInput{
		Spec:      specInput{Content: minimalSpecWithSchemaAndOp},
		Types:     true,
		OutputDir: dir,
	}
	_, output, err := handleGenerate(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.True(t, output.Success)
	assert.Equal(t, dir, output.OutputDir)
	assert.Equal(t, "api", output.PackageName)
	assert.GreaterOrEqual(t, output.FileCount, 1)
	assert.GreaterOrEqual(t, output.GeneratedTypes, 1)
	assert.NotEmpty(t, output.Files)

	// Verify at least one .go file was written to disk.
	found := false
	for _, f := range output.Files {
		path := filepath.Join(dir, f.Name)
		info, statErr := os.Stat(path)
		if statErr == nil && info.Size() > 0 {
			found = true
			break
		}
	}
	assert.True(t, found, "expected at least one generated file on disk")
}

func TestGenerateTool_ClientGeneration(t *testing.T) {
	dir := t.TempDir()

	input := generateInput{
		Spec:        specInput{Content: minimalSpecWithSchemaAndOp},
		Client:      true,
		PackageName: "petstore",
		OutputDir:   dir,
	}
	_, output, err := handleGenerate(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.True(t, output.Success)
	assert.Equal(t, "petstore", output.PackageName)
	assert.GreaterOrEqual(t, output.FileCount, 2, "expect types + client files")
	assert.GreaterOrEqual(t, output.GeneratedOperations, 1)
}

func TestGenerateTool_MissingOutputDir(t *testing.T) {
	input := generateInput{
		Spec:  specInput{Content: minimalSpecWithSchemaAndOp},
		Types: true,
	}
	result, output, err := handleGenerate(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.OutputDir)
}

func TestGenerateTool_InvalidSpec(t *testing.T) {
	dir := t.TempDir()

	input := generateInput{
		Spec:      specInput{Content: "not valid yaml: ["},
		Types:     true,
		OutputDir: dir,
	}
	result, output, err := handleGenerate(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.OutputDir)
}

func TestGenerateTool_NoInputProvided(t *testing.T) {
	dir := t.TempDir()

	input := generateInput{
		Spec:      specInput{},
		Types:     true,
		OutputDir: dir,
	}
	result, output, err := handleGenerate(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsError)
	assert.Empty(t, output.OutputDir)
}

func TestGenerateTool_CustomPackageName(t *testing.T) {
	dir := t.TempDir()

	input := generateInput{
		Spec:        specInput{Content: minimalSpecWithSchemaAndOp},
		Types:       true,
		PackageName: "myapi",
		OutputDir:   dir,
	}
	_, output, err := handleGenerate(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.True(t, output.Success)
	assert.Equal(t, "myapi", output.PackageName)

	// Verify the generated file contains the correct package name.
	require.NotEmpty(t, output.Files, "expected at least one generated file")
	data, readErr := os.ReadFile(filepath.Join(dir, output.Files[0].Name))
	require.NoError(t, readErr)
	assert.Contains(t, string(data), "package myapi")
}

func TestGenerateTool_FileInput(t *testing.T) {
	dir := t.TempDir()

	input := generateInput{
		Spec:      specInput{File: "../../testdata/petstore-3.0.yaml"},
		Types:     true,
		OutputDir: dir,
	}
	_, output, err := handleGenerate(context.Background(), &mcp.CallToolRequest{}, input)
	require.NoError(t, err)

	assert.True(t, output.Success)
	assert.GreaterOrEqual(t, output.FileCount, 1)
	assert.GreaterOrEqual(t, output.GeneratedTypes, 1)
}
