package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateOAS3Server(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Pet API
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        '200':
          description: A list of pets
    post:
      operationId: createPet
      responses:
        '201':
          description: Pet created
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: integer
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "pet-api.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithServer(true),
	)
	require.NoError(t, err)

	serverFile := result.GetFile("server.go")
	require.NotNil(t, serverFile, "server.go not generated")

	content := string(serverFile.Content)
	assert.Contains(t, content, "type ServerInterface interface")
	assert.Contains(t, content, "ListPets(")
	assert.Contains(t, content, "CreatePet(")
	assert.Contains(t, content, "type ListPetsRequest struct")
	assert.Contains(t, content, "type UnimplementedServer struct")
}

func TestGenerateOAS2Server(t *testing.T) {
	spec := `swagger: "2.0"
info:
  title: Pet API
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      parameters:
        - name: limit
          in: query
          type: integer
        - name: X-Request-ID
          in: header
          type: string
      responses:
        '200':
          description: A list of pets
    post:
      operationId: createPet
      parameters:
        - name: body
          in: body
          schema:
            $ref: '#/definitions/Pet'
      responses:
        '201':
          description: Pet created
definitions:
  Pet:
    type: object
    properties:
      id:
        type: integer
      name:
        type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "swagger.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithServer(true),
	)
	require.NoError(t, err)

	serverFile := result.GetFile("server.go")
	require.NotNil(t, serverFile, "server.go not generated")

	content := string(serverFile.Content)
	assert.Contains(t, content, "type ServerInterface interface")
	assert.Contains(t, content, "ListPets(")
	assert.Contains(t, content, "CreatePet(")
	assert.Contains(t, content, "ListPetsRequest")
	assert.Contains(t, content, "CreatePetRequest")
	assert.Contains(t, content, "UnimplementedServer")
}

func TestGenerateServerWithAllParameterTypes(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /items/{id}:
    get:
      operationId: getItem
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
        - name: filter
          in: query
          schema:
            type: string
        - name: X-Request-ID
          in: header
          schema:
            type: string
      responses:
        '200':
          description: OK
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithServer(true),
	)
	require.NoError(t, err)

	serverFile := result.GetFile("server.go")
	content := string(serverFile.Content)

	assert.Contains(t, content, "GetItemRequest")
	assert.Contains(t, content, "Id")
	assert.Contains(t, content, "Filter")
	assert.Contains(t, content, "XRequestID")
}

func TestGenerateServerWithRequestBody(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /items:
    post:
      operationId: createItem
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  type: string
                value:
                  type: number
      responses:
        '201':
          description: Created
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithServer(true),
	)
	require.NoError(t, err)

	serverFile := result.GetFile("server.go")
	content := string(serverFile.Content)

	assert.Contains(t, content, "CreateItem")
	assert.Contains(t, content, "CreateItemRequest")
	assert.Contains(t, content, "Body")
}

func TestGenerateServerWithMultipleOperations(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        '200':
          description: OK
    post:
      operationId: createUser
      responses:
        '201':
          description: Created
  /users/{id}:
    get:
      operationId: getUser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: OK
    put:
      operationId: updateUser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: OK
    delete:
      operationId: deleteUser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      responses:
        '204':
          description: Deleted
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithServer(true),
	)
	require.NoError(t, err)

	serverFile := result.GetFile("server.go")
	content := string(serverFile.Content)

	assert.Contains(t, content, "ListUsers")
	assert.Contains(t, content, "CreateUser")
	assert.Contains(t, content, "GetUser")
	assert.Contains(t, content, "UpdateUser")
	assert.Contains(t, content, "DeleteUser")
}

func TestGenerateServerWithResponseTypes(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /items/{id}:
    get:
      operationId: getItem
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Item'
        '404':
          description: Not found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
components:
  schemas:
    Item:
      type: object
      properties:
        id:
          type: integer
        name:
          type: string
    Error:
      type: object
      properties:
        code:
          type: integer
        message:
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
	)
	require.NoError(t, err)

	serverFile := result.GetFile("server.go")
	content := string(serverFile.Content)

	assert.Contains(t, content, "GetItem")
	assert.Contains(t, content, "GetItemRequest")
}

func TestGenerateServerWithArrayParameters(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /search:
    get:
      operationId: search
      parameters:
        - name: tags
          in: query
          schema:
            type: array
            items:
              type: string
        - name: ids
          in: query
          schema:
            type: array
            items:
              type: integer
      responses:
        '200':
          description: OK
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithServer(true),
	)
	require.NoError(t, err)

	serverFile := result.GetFile("server.go")
	content := string(serverFile.Content)

	assert.Contains(t, content, "Search")
	assert.Contains(t, content, "SearchRequest")
	assert.Contains(t, content, "[]string")
	assert.Contains(t, content, "[]int64")
}

func TestGenerateServerUnimplementedMethods(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: testOp
      responses:
        '200':
          description: OK
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithServer(true),
	)
	require.NoError(t, err)

	serverFile := result.GetFile("server.go")
	content := string(serverFile.Content)

	assert.Contains(t, content, "UnimplementedServer")
	assert.Contains(t, content, "func (s *UnimplementedServer) TestOp")
	assert.Contains(t, content, "ErrNotImplemented")
}

func TestGenerateServerWithDeprecatedOperation(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /old:
    get:
      operationId: oldEndpoint
      deprecated: true
      responses:
        '200':
          description: OK
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithServer(true),
	)
	require.NoError(t, err)

	serverFile := result.GetFile("server.go")
	content := string(serverFile.Content)

	assert.Contains(t, content, "Deprecated:")
}

func TestGenerateServerAndClient(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /items:
    get:
      operationId: listItems
      responses:
        '200':
          description: OK
    post:
      operationId: createItem
      requestBody:
        content:
          application/json:
            schema:
              type: object
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
	)
	require.NoError(t, err)

	assert.NotNil(t, result.GetFile("types.go"))
	assert.NotNil(t, result.GetFile("client.go"))
	assert.NotNil(t, result.GetFile("server.go"))
	assert.NotNil(t, result.GetFile("README.md"))
	assert.Equal(t, 4, len(result.Files))
}

func TestGenerateServerWithNoOperationId(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /pets:
    get:
      summary: List pets
      responses:
        '200':
          description: OK
  /pets/{id}:
    get:
      summary: Get pet
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
      responses:
        '200':
          description: OK
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithServer(true),
	)
	require.NoError(t, err)

	serverFile := result.GetFile("server.go")
	content := string(serverFile.Content)

	// Should generate method names from path + method
	assert.Contains(t, content, "GetPets")
	assert.Contains(t, content, "GetPetsById")
}

func TestGenerateServerWithOptionalParams(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /items:
    get:
      operationId: listItems
      parameters:
        - name: page
          in: query
          schema:
            type: integer
        - name: limit
          in: query
          schema:
            type: integer
        - name: sort
          in: query
          schema:
            type: string
      responses:
        '200':
          description: OK
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithServer(true),
		WithPointers(true),
	)
	require.NoError(t, err)

	serverFile := result.GetFile("server.go")
	content := string(serverFile.Content)

	assert.Contains(t, content, "ListItemsRequest")
	assert.Contains(t, content, "Page")
	assert.Contains(t, content, "Limit")
	assert.Contains(t, content, "Sort")
}

func TestGenerateServerWithRequiredParams(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /items:
    get:
      operationId: listItems
      parameters:
        - name: status
          in: query
          required: true
          schema:
            type: string
        - name: category
          in: query
          required: true
          schema:
            type: string
      responses:
        '200':
          description: OK
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithServer(true),
	)
	require.NoError(t, err)

	serverFile := result.GetFile("server.go")
	content := string(serverFile.Content)

	assert.Contains(t, content, "ListItemsRequest")
	assert.Contains(t, content, "Status")
	assert.Contains(t, content, "Category")
}

func TestGenerateServerWithMultilineDescription(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /bars:
    get:
      operationId: getBar
      summary: "Retrieves all Bars that match the specified criteria\nThis API is intended for retrieval of large amounts of results(>10k) using a pagination based on a after token.\nIf you need to use offset pagination, consider using GET /bar/queries/bar/* and POST /bar/entities/bar/* APIs."
      responses:
        '200':
          description: OK
  /deprecated:
    get:
      operationId: deprecatedOp
      summary: "Foo\nDeprecated: Please use version v2 of this endpoint."
      deprecated: true
      responses:
        '200':
          description: OK
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithServer(true),
	)
	require.NoError(t, err)

	serverFile := result.GetFile("server.go")
	require.NotNil(t, serverFile, "server.go not generated")
	
	content := string(serverFile.Content)

	// Verify GetBar has proper multiline comments
	assert.Contains(t, content, "// GetBar Retrieves all Bars")
	assert.Contains(t, content, "// This API is intended for retrieval")
	assert.Contains(t, content, "// If you need to use offset pagination")
	
	// Verify deprecated operation has proper comments
	assert.Contains(t, content, "// DeprecatedOp Foo")
	assert.Contains(t, content, "// Deprecated: Please use version v2")
	
	// Ensure no bare newlines in comments (would cause compile error)
	assert.NotContains(t, content, "criteria\nThis API")
	assert.NotContains(t, content, "token.\nIf you")
}
