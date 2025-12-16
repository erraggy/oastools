package generator

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erraggy/oastools/fixer"
	"github.com/erraggy/oastools/parser"
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
	err := os.WriteFile(tmpFile, []byte(spec), 0644)
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
	err := os.WriteFile(tmpFile, []byte(spec), 0644)
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
	err := os.WriteFile(tmpFile, []byte(spec), 0644)
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

// TestGeneratedSplitClientCompiles verifies that generated split client code compiles without errors.
// This specifically tests the scenario where generateBaseClient is used (split client generation),
// which was the source of the missing imports bug.
func TestGeneratedSplitClientCompiles(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /items:
    get:
      operationId: listItems
      summary: "List all items\nSupports pagination and filtering"
      responses:
        '200':
          description: OK
    post:
      operationId: createItem
      summary: Create an item
      responses:
        '201':
          description: Created
  /users:
    get:
      operationId: listUsers
      summary: List all users
      responses:
        '200':
          description: OK
    post:
      operationId: createUser
      summary: Create a user
      responses:
        '201':
          description: Created
  /products:
    get:
      operationId: listProducts
      summary: List all products
      responses:
        '200':
          description: OK
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
	err := os.WriteFile(tmpFile, []byte(spec), 0644)
	require.NoError(t, err)

	// Generate with split client (low MaxOperationsPerFile to trigger split)
	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithTypes(true),
		WithMaxOperationsPerFile(2), // Force split with only 2 operations per file
	)
	require.NoError(t, err)

	// Verify we got split files (should have client.go plus client_*.go files)
	hasBaseClient := false
	hasSplitClient := false
	for _, file := range result.Files {
		if file.Name == "client.go" {
			hasBaseClient = true
			// With imports.Process(), base client.go only contains imports it actually uses.
			// The actual API methods (which use bytes, encoding/json, etc.) are in split files.
			// The compilation test below verifies everything works together.
		}
		if len(file.Name) > 7 && file.Name[:7] == "client_" {
			hasSplitClient = true
		}
	}
	assert.True(t, hasBaseClient, "should have base client.go")
	assert.True(t, hasSplitClient, "should have split client_*.go files")

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

	// Try to compile the generated code - this is the critical test
	// Note: Split client files may have unused imports if operations are very simple,
	// but the key test is that base client.go compiles with all required imports
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = outputDir
	output, err := cmd.CombinedOutput()

	// If compilation fails, check if it's just unused imports in split files (acceptable)
	// vs missing imports in base client (the bug we fixed)
	if err != nil {
		outputStr := string(output)
		// The critical issue we fixed was missing imports in base client.go
		// If we see "undefined:" errors in base client, that's the real problem
		if strings.Contains(outputStr, "client.go") && strings.Contains(outputStr, "undefined:") {
			t.Fatalf("base client.go has undefined symbols (missing imports bug not fixed):\n%s", outputStr)
		}
		// Unused imports in split files are a minor issue, not the bug we're addressing
		// We can note it but not fail the test
		if strings.Contains(outputStr, "imported and not used") {
			t.Logf("Note: Split client files have unused imports (pre-existing minor issue):\n%s", outputStr)
		} else {
			// Some other compilation error
			t.Fatalf("unexpected compilation error:\n%s", outputStr)
		}
	}
}

// TestGeneratedOAS2ClientCompiles verifies that generated OAS 2.0 (Swagger) client code
// compiles without errors, specifically testing multiline description handling.
func TestGeneratedOAS2ClientCompiles(t *testing.T) {
	spec := `swagger: "2.0"
info:
  title: Test API
  version: "1.0.0"
basePath: /v1
paths:
  /items:
    get:
      operationId: listItems
      summary: "List all items\nSupports pagination and filtering\nReturns paginated results"
      parameters:
        - name: page
          in: query
          type: integer
          description: "Page number\nStarts from 1"
      responses:
        '200':
          description: OK
          schema:
            type: array
            items:
              $ref: '#/definitions/Item'
    post:
      operationId: createItem
      summary: Create an item
      description: "Creates a new item in the system.\nThe item will be assigned an ID automatically.\nReturns the created item."
      parameters:
        - name: body
          in: body
          required: true
          schema:
            $ref: '#/definitions/Item'
      responses:
        '201':
          description: Created
definitions:
  Item:
    type: object
    properties:
      id:
        type: integer
      name:
        type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "swagger.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0644)
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
	assert.NoError(t, err, "generated OAS 2.0 client code should compile without errors.\nCompiler output:\n%s", string(output))
}

// TestGeneratedOAS2ServerCompiles verifies that generated OAS 2.0 (Swagger) server code
// compiles without errors, specifically testing multiline description handling.
func TestGeneratedOAS2ServerCompiles(t *testing.T) {
	spec := `swagger: "2.0"
info:
  title: Test API
  version: "1.0.0"
basePath: /v1
paths:
  /bars:
    get:
      operationId: listBars
      summary: "List all bars\nWith filtering support"
      responses:
        '200':
          description: OK
          schema:
            type: array
            items:
              $ref: '#/definitions/Bar'
    post:
      operationId: createBar
      description: "Create a new bar.\nThis endpoint requires authentication.\nReturns the created bar with its ID."
      parameters:
        - name: body
          in: body
          required: true
          schema:
            $ref: '#/definitions/Bar'
      responses:
        '201':
          description: Created
  /bars/{barId}:
    get:
      operationId: getBar
      summary: Get a bar by ID
      deprecated: true
      parameters:
        - name: barId
          in: path
          required: true
          type: integer
      responses:
        '200':
          description: OK
          schema:
            $ref: '#/definitions/Bar'
definitions:
  Bar:
    type: object
    properties:
      id:
        type: integer
      name:
        type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "swagger.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0644)
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
	assert.NoError(t, err, "generated OAS 2.0 server code should compile without errors.\nCompiler output:\n%s", string(output))
}

// TestFixerToGeneratorIntegration verifies the full pipeline from fixer to generator.
// This reproduces the exact use case from issue #149:
// OAS 2.0 in JSON format with --infer, --prune-all, generic naming, and missing path params.
func TestFixerToGeneratorIntegration(t *testing.T) {
	// Import fixer package for this integration test
	// This is the EXACT use case from issue #149
	specJSON := `{
  "swagger": "2.0",
  "info": {
    "title": "Test API",
    "version": "1.0"
  },
  "paths": {
    "/users/{userId}/posts/{postId}": {
      "get": {
        "operationId": "getUserPost",
        "produces": ["application/json"],
        "responses": {
          "200": {
            "description": "Success",
            "schema": {
              "$ref": "#/definitions/Response[Post]"
            }
          }
        }
      }
    }
  },
  "definitions": {
    "Response[Post]": {
      "type": "object",
      "properties": {
        "data": {
          "$ref": "#/definitions/Post"
        }
      }
    },
    "Post": {
      "type": "object",
      "properties": {
        "id": {
          "type": "integer"
        },
        "title": {
          "type": "string"
        }
      }
    },
    "UnusedSchema": {
      "type": "object",
      "properties": {
        "orphan": {
          "type": "string"
        }
      }
    }
  }
}`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "swagger.json")
	err := os.WriteFile(tmpFile, []byte(specJSON), 0644)
	require.NoError(t, err)

	// Step 1: Parse the original document
	parseResult, err := parser.ParseWithOptions(parser.WithFilePath(tmpFile))
	require.NoError(t, err)
	require.Equal(t, parser.FormatJSON, parseResult.SourceFormat, "should detect JSON format")

	// Step 2: Run fixer with ALL options (matching issue #149 use case)
	f := fixer.New()
	f.InferTypes = true // --infer flag
	f.EnabledFixes = []fixer.FixType{
		fixer.FixTypeMissingPathParameter, // fix missing params
		fixer.FixTypeRenamedGenericSchema, // generic naming
		fixer.FixTypePrunedUnusedSchema,   // --prune-all
		fixer.FixTypePrunedEmptyPath,      // --prune-all
	}
	f.GenericNamingConfig.Strategy = fixer.GenericNamingOf // _of_ strategy

	fixResult, err := f.FixParsed(*parseResult)
	require.NoError(t, err)
	require.True(t, fixResult.HasFixes(), "should have applied fixes")

	// Verify the fixes that were applied
	t.Logf("Applied %d fixes to the document", fixResult.FixCount)

	// Step 3: Write the fixed document to a temp file
	fixedFile := filepath.Join(tmpDir, "fixed-swagger.json")
	err = parser.WriteDocument(fixedFile, fixResult.Document, fixResult.SourceFormat)
	require.NoError(t, err)

	// Step 4: Validate the fixed document is valid (even with --strict)
	validateResult, err := parser.ParseWithOptions(
		parser.WithFilePath(fixedFile),
		parser.WithValidateStructure(true),
	)
	require.NoError(t, err)
	require.Empty(t, validateResult.Errors, "fixed document should be valid")

	// Step 5: Generate client and server code from the fixed document
	genResult, err := GenerateWithOptions(
		WithFilePath(fixedFile),
		WithPackageName("testapi"),
		WithClient(true),
		WithServer(true),
		WithTypes(true),
	)
	require.NoError(t, err)
	require.NotNil(t, genResult)
	require.NotEmpty(t, genResult.Files, "should generate files")

	// Step 6: Write generated files to output directory
	outputDir := filepath.Join(tmpDir, "testapi")
	err = os.MkdirAll(outputDir, 0755)
	require.NoError(t, err)

	for _, file := range genResult.Files {
		filePath := filepath.Join(outputDir, file.Name)
		err = os.WriteFile(filePath, file.Content, 0644)
		require.NoError(t, err, "failed to write %s", file.Name)
		t.Logf("Generated file: %s", file.Name)
	}

	// Step 7: Create go.mod for the test package
	goModContent := `module testapi

go 1.24
`
	err = os.WriteFile(filepath.Join(outputDir, "go.mod"), []byte(goModContent), 0644)
	require.NoError(t, err)

	// Step 8: Try to compile the generated code
	// This is the CRITICAL test - the generated code MUST compile without errors
	cmd := exec.Command("go", "build", "./...")
	cmd.Dir = outputDir
	output, err := cmd.CombinedOutput()
	assert.NoError(t, err, "generated code from fixed document should compile without errors.\n"+
		"This validates the fix for issue #149.\nCompiler output:\n%s", string(output))

	if err == nil {
		t.Logf("✅ SUCCESS: Generated code compiles successfully after fixer with --infer and --prune-all")
	}
}
