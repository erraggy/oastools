package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGeneratedClientCompiles verifies that generated client code compiles without errors.
// This is critical to catch issues like missing imports that would break user code.
func TestGeneratedClientCompiles(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /items:
    get:
      operationId: listItems
      summary: "List all items\nSupports pagination and filtering"
      parameters:
        - name: page
          in: query
          schema:
            type: integer
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Item'
    post:
      operationId: createItem
      summary: Create an item
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Item'
      responses:
        '201':
          description: Created
components:
  schemas:
    Item:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithTypes(true),
	)
	require.NoError(t, err)

	// Create output directory
	outputDir := filepath.Join(tmpDir, "testapi")
	err = os.MkdirAll(outputDir, 0755)
	require.NoError(t, err)

	// Write generated files
	for _, file := range result.Files {
		filePath := filepath.Join(outputDir, file.Name)
		err = os.WriteFile(filePath, file.Content, 0644)
		require.NoError(t, err, "failed to write %s", file.Name)
	}

	// Create go.mod for the test package
	goModContent := `module testapi

go 1.24
`
	err = os.WriteFile(filepath.Join(outputDir, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	// Try to compile the generated code
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = outputDir
	output, err := cmd.CombinedOutput()
	assert.NoError(t, err, "generated client code should compile without errors.\nCompiler output:\n%s", string(output))
}

// TestGeneratedServerCompiles verifies that generated server code compiles without errors.
// This specifically tests the multiline description handling that was causing compile errors.
func TestGeneratedServerCompiles(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /bars:
    get:
      operationId: getBar
      summary: "Retrieves all Bars that match the specified criteria\nThis API is intended for retrieval of large amounts of results(>10k) using a pagination based on an after token.\nIf you need to use offset pagination, consider using GET /bar/queries/bar/* and POST /bar/entities/bar/* APIs."
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Bar'
  /foo:
    get:
      operationId: getFoo
      summary: "Foo\nDeprecated: Please use version v2 of this endpoint."
      deprecated: true
      responses:
        '200':
          description: OK
components:
  schemas:
    Bar:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithServer(true),
		WithTypes(true),
	)
	require.NoError(t, err)

	// Create output directory
	outputDir := filepath.Join(tmpDir, "testapi")
	err = os.MkdirAll(outputDir, 0755)
	require.NoError(t, err)

	// Write generated files
	for _, file := range result.Files {
		filePath := filepath.Join(outputDir, file.Name)
		err = os.WriteFile(filePath, file.Content, 0644)
		require.NoError(t, err, "failed to write %s", file.Name)
	}

	// Create go.mod for the test package
	goModContent := `module testapi

go 1.24
`
	err = os.WriteFile(filepath.Join(outputDir, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	// Try to compile the generated code
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = outputDir
	output, err := cmd.CombinedOutput()
	assert.NoError(t, err, "generated server code should compile without errors.\nCompiler output:\n%s", string(output))
}

// TestGeneratedClientAndServerCompiles verifies that generated client+server code compiles without errors.
// This tests the full generation including both client and server with complex multiline descriptions.
func TestGeneratedClientAndServerCompiles(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Complex API
  version: "1.0.0"
paths:
  /items:
    get:
      operationId: listItems
      summary: "List all items in the catalog\nSupports advanced filtering and pagination\nReturns up to 100 items per page"
      responses:
        '200':
          description: OK
    post:
      operationId: createItem
      description: "Creates a new item\nThe item must have unique properties\nDuplicates will be rejected"
      responses:
        '201':
          description: Created
components:
  schemas:
    Item:
      type: object
      properties:
        id:
          type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithServer(true),
		WithTypes(true),
	)
	require.NoError(t, err)

	// Create output directory
	outputDir := filepath.Join(tmpDir, "testapi")
	err = os.MkdirAll(outputDir, 0755)
	require.NoError(t, err)

	// Write generated files
	for _, file := range result.Files {
		filePath := filepath.Join(outputDir, file.Name)
		err = os.WriteFile(filePath, file.Content, 0644)
		require.NoError(t, err, "failed to write %s", file.Name)
	}

	// Create go.mod for the test package
	goModContent := `module testapi

go 1.24
`
	err = os.WriteFile(filepath.Join(outputDir, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	// Try to compile the generated code
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = outputDir
	output, err := cmd.CombinedOutput()
	assert.NoError(t, err, "generated client+server code should compile without errors.\nCompiler output:\n%s", string(output))
}
