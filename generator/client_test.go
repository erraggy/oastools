package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateOAS3Client(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Pet API
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      summary: List all pets
      parameters:
        - name: limit
          in: query
          schema:
            type: integer
      responses:
        '200':
          description: A list of pets
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Pet'
    post:
      operationId: createPet
      summary: Create a pet
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Pet'
      responses:
        '201':
          description: Pet created
  /pets/{petId}:
    get:
      operationId: getPet
      summary: Get a pet by ID
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: integer
            format: int64
      responses:
        '200':
          description: A pet
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
      type: object
      properties:
        id:
          type: integer
          format: int64
        name:
          type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "pet-api.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithClient(true),
	)
	require.NoError(t, err)

	assert.Equal(t, 3, result.GeneratedOperations)

	clientFile := result.GetFile("client.go")
	require.NotNil(t, clientFile, "client.go not generated")

	content := string(clientFile.Content)
	assert.Contains(t, content, "type Client struct")
	assert.Contains(t, content, "func NewClient")
	assert.Contains(t, content, "func (c *Client) ListPets")
	assert.Contains(t, content, "func (c *Client) CreatePet")
	assert.Contains(t, content, "func (c *Client) GetPet")
	assert.Contains(t, content, "type ListPetsParams struct")
}

func TestGenerateOAS2Client(t *testing.T) {
	spec := `swagger: "2.0"
info:
  title: Pet API
  version: "1.0.0"
basePath: /v1
paths:
  /pets:
    get:
      operationId: listPets
      summary: List all pets
      parameters:
        - name: limit
          in: query
          type: integer
          description: Maximum number of pets
        - name: status
          in: query
          type: string
          required: true
      produces:
        - application/json
      responses:
        '200':
          description: A list of pets
          schema:
            type: array
            items:
              $ref: '#/definitions/Pet'
    post:
      operationId: createPet
      summary: Create a pet
      consumes:
        - application/json
      parameters:
        - name: body
          in: body
          required: true
          schema:
            $ref: '#/definitions/NewPet'
      responses:
        '201':
          description: Pet created
          schema:
            $ref: '#/definitions/Pet'
  /pets/{petId}:
    get:
      operationId: getPet
      summary: Get a pet by ID
      parameters:
        - name: petId
          in: path
          required: true
          type: integer
          format: int64
      responses:
        '200':
          description: A pet
          schema:
            $ref: '#/definitions/Pet'
        default:
          description: Error
          schema:
            $ref: '#/definitions/Error'
definitions:
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
      tag:
        type: string
  NewPet:
    type: object
    required:
      - name
    properties:
      name:
        type: string
      tag:
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
	tmpFile := filepath.Join(tmpDir, "swagger.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithClient(true),
	)
	require.NoError(t, err)

	clientFile := result.GetFile("client.go")
	require.NotNil(t, clientFile, "client.go not generated")

	content := string(clientFile.Content)
	assert.Contains(t, content, "type Client struct")
	assert.Contains(t, content, "func (c *Client) ListPets")
	assert.Contains(t, content, "func (c *Client) CreatePet")
	assert.Contains(t, content, "func (c *Client) GetPet")
	assert.Contains(t, content, "ListPetsParams")
	assert.Equal(t, 3, result.GeneratedOperations)
}

func TestGenerateClientWithDeprecatedOperation(t *testing.T) {
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
		WithClient(true),
	)
	require.NoError(t, err)

	clientFile := result.GetFile("client.go")
	content := string(clientFile.Content)

	assert.Contains(t, content, "Deprecated:")
}

func TestGenerateClientWithUserAgent(t *testing.T) {
	// UserAgent is used for fetching remote references during parsing,
	// not for inclusion in the generated client code.
	// This test verifies the option is accepted without error.
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /test:
    get:
      operationId: test
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
		WithClient(true),
		WithUserAgent("my-custom-agent/1.0"),
	)
	require.NoError(t, err)

	clientFile := result.GetFile("client.go")
	require.NotNil(t, clientFile, "client.go should be generated")
}

func TestGenerateClientWithMultipleContentTypes(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /upload:
    post:
      operationId: upload
      requestBody:
        content:
          multipart/form-data:
            schema:
              type: object
              properties:
                file:
                  type: string
                  format: binary
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
		WithClient(true),
	)
	require.NoError(t, err)

	clientFile := result.GetFile("client.go")
	require.NotNil(t, clientFile)

	content := string(clientFile.Content)
	assert.Contains(t, content, "Upload")
}

func TestGenerateClientWithPathParameters(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /users/{userId}/posts/{postId}:
    get:
      operationId: getUserPost
      parameters:
        - name: userId
          in: path
          required: true
          schema:
            type: integer
        - name: postId
          in: path
          required: true
          schema:
            type: string
            format: uuid
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
		WithClient(true),
	)
	require.NoError(t, err)

	clientFile := result.GetFile("client.go")
	content := string(clientFile.Content)

	assert.Contains(t, content, "GetUserPost")
	assert.Contains(t, content, "userId")
	assert.Contains(t, content, "postId")
}

func TestGenerateClientWithHeaderParameters(t *testing.T) {
	// Note: Currently header parameters are handled via RequestEditors pattern
	// rather than explicit method parameters in the generated client.
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /protected:
    get:
      operationId: getProtected
      parameters:
        - name: X-API-Key
          in: header
          required: true
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
		WithClient(true),
	)
	require.NoError(t, err)

	clientFile := result.GetFile("client.go")
	content := string(clientFile.Content)

	assert.Contains(t, content, "GetProtected")
	// Header parameters are typically handled via RequestEditors
	assert.Contains(t, content, "RequestEditorFn")
}

func TestGenerateClientWithAllHTTPMethods(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths:
  /resource:
    get:
      operationId: getResource
      responses:
        '200':
          description: OK
    post:
      operationId: createResource
      responses:
        '201':
          description: Created
    put:
      operationId: updateResource
      responses:
        '200':
          description: OK
    patch:
      operationId: patchResource
      responses:
        '200':
          description: OK
    delete:
      operationId: deleteResource
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
		WithClient(true),
	)
	require.NoError(t, err)

	clientFile := result.GetFile("client.go")
	content := string(clientFile.Content)

	assert.Contains(t, content, "GetResource")
	assert.Contains(t, content, "CreateResource")
	assert.Contains(t, content, "UpdateResource")
	assert.Contains(t, content, "PatchResource")
	assert.Contains(t, content, "DeleteResource")
	// Verify HTTP methods are used (string literals)
	assert.Contains(t, content, `"GET"`)
	assert.Contains(t, content, `"POST"`)
	assert.Contains(t, content, `"PUT"`)
	assert.Contains(t, content, `"PATCH"`)
	assert.Contains(t, content, `"DELETE"`)
}

func TestGenerateClientWithArrayQueryParams(t *testing.T) {
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
		WithClient(true),
	)
	require.NoError(t, err)

	clientFile := result.GetFile("client.go")
	content := string(clientFile.Content)

	assert.Contains(t, content, "Search")
	assert.Contains(t, content, "SearchParams")
	assert.Contains(t, content, "[]string")
	assert.Contains(t, content, "[]int64")
}

func TestGenerateClientWithRequestBody(t *testing.T) {
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
		WithClient(true),
	)
	require.NoError(t, err)

	clientFile := result.GetFile("client.go")
	content := string(clientFile.Content)

	assert.Contains(t, content, "CreateItem")
	assert.Contains(t, content, "json.Marshal")
}

func TestGenerateClientWithResponseTypes(t *testing.T) {
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
		WithClient(true),
	)
	require.NoError(t, err)

	clientFile := result.GetFile("client.go")
	content := string(clientFile.Content)

	assert.Contains(t, content, "GetItem")
	assert.Contains(t, content, "*Item")
}

func TestGenerateClientWithNoOperationId(t *testing.T) {
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
		WithClient(true),
	)
	require.NoError(t, err)

	clientFile := result.GetFile("client.go")
	content := string(clientFile.Content)

	// Should generate method names from path + method
	assert.Contains(t, content, "GetPets")
	assert.Contains(t, content, "GetPetsById")
}
