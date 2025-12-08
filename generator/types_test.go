package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateOAS3Types(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
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
	tmpFile := filepath.Join(tmpDir, "test-api.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithTypes(true),
	)
	require.NoError(t, err)

	assert.Equal(t, "testapi", result.PackageName)
	assert.Equal(t, "3.0.0", result.SourceVersion)
	assert.Equal(t, 2, result.GeneratedTypes)

	typesFile := result.GetFile("types.go")
	require.NotNil(t, typesFile, "types.go not generated")

	content := string(typesFile.Content)
	assert.Contains(t, content, "package testapi")
	assert.Contains(t, content, "type Pet struct")
	assert.Contains(t, content, "Id")
	assert.Contains(t, content, "int64")
	assert.Contains(t, content, "Name")
	assert.Contains(t, content, "string")
	assert.Contains(t, content, "type Error struct")
}

func TestGenerateOAS2Types(t *testing.T) {
	spec := `swagger: "2.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
definitions:
  Pet:
    type: object
    required:
      - name
    properties:
      id:
        type: integer
        format: int64
      name:
        type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "swagger.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithTypes(true),
	)
	require.NoError(t, err)

	assert.Equal(t, "2.0", result.SourceVersion)
	assert.Equal(t, 1, result.GeneratedTypes)

	typesFile := result.GetFile("types.go")
	require.NotNil(t, typesFile, "types.go not generated")

	content := string(typesFile.Content)
	assert.Contains(t, content, "type Pet struct")
}

func TestGenerateSchemaWithRef(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Pet:
      $ref: '#/components/schemas/Animal'
    Animal:
      type: object
      properties:
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
	)
	require.NoError(t, err)

	typesFile := result.GetFile("types.go")
	require.NotNil(t, typesFile)

	content := string(typesFile.Content)
	assert.Contains(t, content, "type Pet = Animal")
}

func TestGenerateSchemaWithAllOf(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Pet:
      allOf:
        - $ref: '#/components/schemas/Animal'
        - type: object
          properties:
            petId:
              type: integer
    Animal:
      type: object
      properties:
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
	)
	require.NoError(t, err)

	typesFile := result.GetFile("types.go")
	content := string(typesFile.Content)

	assert.Contains(t, content, "type Pet struct")
	assert.Contains(t, content, "Animal")
	assert.Contains(t, content, "PetId")
}

func TestGenerateSchemaWithEnum(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Status:
      type: string
      enum:
        - available
        - pending
        - sold
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
	)
	require.NoError(t, err)

	typesFile := result.GetFile("types.go")
	content := string(typesFile.Content)

	assert.Contains(t, content, "type Status string")
	assert.Contains(t, content, "StatusAvailable")
	assert.Contains(t, content, "StatusPending")
	assert.Contains(t, content, "StatusSold")
}

func TestGenerateSchemaWithArray(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Pets:
      type: array
      items:
        $ref: '#/components/schemas/Pet'
    Pet:
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
	)
	require.NoError(t, err)

	typesFile := result.GetFile("types.go")
	content := string(typesFile.Content)

	assert.Contains(t, content, "type Pets []")
}

func TestGenerateSchemaWithAdditionalProperties(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Metadata:
      type: object
      properties:
        name:
          type: string
      additionalProperties: true
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
	)
	require.NoError(t, err)

	typesFile := result.GetFile("types.go")
	content := string(typesFile.Content)

	assert.Contains(t, content, "type Metadata struct")
	assert.Contains(t, content, "Name")
}

func TestGenerateSchemaWithNullable(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        name:
          type: string
          nullable: true
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithPointers(true),
	)
	require.NoError(t, err)

	typesFile := result.GetFile("types.go")
	content := string(typesFile.Content)

	assert.Contains(t, content, "*string")
}

func TestGenerateSchemaWithFormats(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Data:
      type: object
      properties:
        created:
          type: string
          format: date-time
        data:
          type: string
          format: byte
        id32:
          type: integer
          format: int32
        id64:
          type: integer
          format: int64
        score:
          type: number
          format: float
        value:
          type: number
          format: double
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
	)
	require.NoError(t, err)

	typesFile := result.GetFile("types.go")
	content := string(typesFile.Content)

	assert.Contains(t, content, "time.Time")
	assert.Contains(t, content, "[]byte")
	assert.Contains(t, content, "int32")
	assert.Contains(t, content, "int64")
	assert.Contains(t, content, "float32")
	assert.Contains(t, content, "float64")
}

func TestGenerateWithNoPointers(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        name:
          type: string
        tag:
          type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithPointers(false),
	)
	require.NoError(t, err)

	typesFile := result.GetFile("types.go")
	content := string(typesFile.Content)

	assert.NotContains(t, content, "*string `json:\"tag")
}

func TestGenerateWithAdditionalProperties(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Pet API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Tags:
      type: object
      additionalProperties:
        type: string
    Config:
      type: object
      additionalProperties:
        type: object
        properties:
          value:
            type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	g := New()
	g.GenerateTypes = true
	result, err := g.Generate(tmpFile)
	require.NoError(t, err)

	content := string(result.GetFile("types.go").Content)
	assert.Contains(t, content, "map[string]string")
}

func TestGenerateWithDiscriminator(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Pet API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Animal:
      oneOf:
        - $ref: '#/components/schemas/Dog'
        - $ref: '#/components/schemas/Cat'
      discriminator:
        propertyName: petType
        mapping:
          dog: '#/components/schemas/Dog'
          cat: '#/components/schemas/Cat'
    Dog:
      type: object
      properties:
        petType:
          type: string
        breed:
          type: string
    Cat:
      type: object
      properties:
        petType:
          type: string
        color:
          type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	g := New()
	g.GenerateTypes = true
	result, err := g.Generate(tmpFile)
	require.NoError(t, err)

	content := string(result.GetFile("types.go").Content)
	assert.Contains(t, content, "UnmarshalJSON")
	assert.Contains(t, content, "petType")
	assert.Contains(t, content, "switch")
}

func TestGenerateWithValidationTags(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Pet API
  version: "1.0.0"
paths: {}
components:
  schemas:
    User:
      type: object
      required:
        - name
        - age
      properties:
        name:
          type: string
          minLength: 1
          maxLength: 100
        email:
          type: string
          format: email
        age:
          type: integer
          minimum: 0
          maximum: 150
        tags:
          type: array
          minItems: 1
          maxItems: 10
        status:
          type: string
          enum:
            - active
            - inactive
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	g := New()
	g.GenerateTypes = true
	g.IncludeValidation = true
	result, err := g.Generate(tmpFile)
	require.NoError(t, err)

	content := string(result.GetFile("types.go").Content)
	assert.Contains(t, content, "validate:")
	assert.Contains(t, content, "required")
	assert.Contains(t, content, "email")
	assert.Contains(t, content, "min=")
}

func TestGenerateOAS2WithAllOf(t *testing.T) {
	spec := `swagger: "2.0"
info:
  title: Pet API
  version: "1.0.0"
paths: {}
definitions:
  Pet:
    type: object
    properties:
      id:
        type: integer
      name:
        type: string
  Dog:
    allOf:
      - $ref: '#/definitions/Pet'
      - type: object
        properties:
          breed:
            type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	g := New()
	g.GenerateTypes = true
	result, err := g.Generate(tmpFile)
	require.NoError(t, err)

	content := string(result.GetFile("types.go").Content)
	assert.Contains(t, content, "type Dog struct")
	assert.Contains(t, content, "Pet")
}

func TestGenerateOAS3WithContentTypeDetection(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Pet API
  version: "1.0.0"
paths:
  /pets:
    post:
      operationId: createPet
      requestBody:
        required: true
        content:
          application/xml:
            schema:
              $ref: '#/components/schemas/Pet'
      responses:
        '201':
          description: Created
          content:
            application/xml:
              schema:
                $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
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

	g := New()
	g.GenerateClient = true
	result, err := g.Generate(tmpFile)
	require.NoError(t, err)

	content := string(result.GetFile("client.go").Content)
	assert.Contains(t, content, "application/xml")
}

func TestGenerateOAS3ServerWithCookieParams(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Pet API
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      parameters:
        - name: session_id
          in: cookie
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Success
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Pet'
components:
  schemas:
    Pet:
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

	g := New()
	g.GenerateServer = true
	result, err := g.Generate(tmpFile)
	require.NoError(t, err)

	content := string(result.GetFile("server.go").Content)
	assert.Contains(t, content, "SessionId")
}

func TestGenerateTypesWithNestedObjects(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Order:
      type: object
      properties:
        id:
          type: integer
        customer:
          type: object
          properties:
            name:
              type: string
            address:
              type: object
              properties:
                street:
                  type: string
                city:
                  type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
	)
	require.NoError(t, err)

	typesFile := result.GetFile("types.go")
	require.NotNil(t, typesFile)

	content := string(typesFile.Content)
	assert.Contains(t, content, "type Order struct")
}

func TestGenerateTypesWithMapType(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    StringMap:
      type: object
      additionalProperties:
        type: string
    IntMap:
      type: object
      additionalProperties:
        type: integer
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
	)
	require.NoError(t, err)

	typesFile := result.GetFile("types.go")
	content := string(typesFile.Content)

	assert.Contains(t, content, "map[string]string")
	assert.Contains(t, content, "map[string]int64")
}

func TestGenerateOAS31TypeArrayWithNull(t *testing.T) {
	spec := `openapi: "3.1.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Pet:
      type: object
      properties:
        name:
          type:
            - string
            - "null"
        age:
          type: integer
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithPointers(true),
	)
	require.NoError(t, err)

	typesFile := result.GetFile("types.go")
	content := string(typesFile.Content)

	// OAS 3.1 type array with null should be handled as nullable
	assert.Contains(t, content, "*string")
}

func TestGenerateTypesWithDescription(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Pet:
      type: object
      description: A pet in the store
      properties:
        id:
          type: integer
          description: The unique identifier
        name:
          type: string
          description: The pet's name
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
	)
	require.NoError(t, err)

	typesFile := result.GetFile("types.go")
	content := string(typesFile.Content)

	assert.Contains(t, content, "A pet in the store")
	assert.Contains(t, content, "The unique identifier")
	assert.Contains(t, content, "The pet's name")
}

func TestGenerateTypesWithBooleanAndDefault(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Settings:
      type: object
      properties:
        enabled:
          type: boolean
          default: true
        count:
          type: integer
          default: 10
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
	)
	require.NoError(t, err)

	typesFile := result.GetFile("types.go")
	content := string(typesFile.Content)

	assert.Contains(t, content, "Enabled")
	assert.Contains(t, content, "bool")
	assert.Contains(t, content, "Count")
}

func TestGenerateTypesWithMinMaxValidation(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Pagination:
      type: object
      properties:
        page:
          type: integer
          minimum: 1
        pageSize:
          type: integer
          minimum: 1
          maximum: 100
        query:
          type: string
          minLength: 1
          maxLength: 255
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	g := New()
	g.GenerateTypes = true
	g.IncludeValidation = true
	result, err := g.Generate(tmpFile)
	require.NoError(t, err)

	content := string(result.GetFile("types.go").Content)

	// Check for validation tags
	assert.True(t, strings.Contains(content, "min=") || strings.Contains(content, "validate:"))
}

func TestGenerateTypesWithDuplicateFieldNames(t *testing.T) {
	// Test that properties like @id and id (which both convert to Id) are handled
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    JsonLdResource:
      type: object
      properties:
        "@id":
          type: string
          description: JSON-LD identifier
        id:
          type: string
          description: Regular identifier
        "@type":
          type: string
        type:
          type: string
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
	)
	require.NoError(t, err)

	typesFile := result.GetFile("types.go")
	require.NotNil(t, typesFile)
	content := string(typesFile.Content)

	// Should have JsonLdResource struct
	assert.Contains(t, content, "type JsonLdResource struct")

	// Should have both Id and Id2 fields (or similar deduplication)
	assert.Contains(t, content, "Id ")
	assert.Contains(t, content, "Id2 ")

	// Should have proper JSON tags for both
	assert.Contains(t, content, `json:"@id`)
	assert.Contains(t, content, `json:"id`)

	// Should have Type_ and Type_2 fields (underscore from @type conversion)
	assert.Contains(t, content, "Type_ ")
	assert.Contains(t, content, "Type_2 ")
}

func TestGenerateTypesWithAdditionalPropertiesFalse(t *testing.T) {
	// Test that additionalProperties: false does not generate an AdditionalProperties field
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    StrictObject:
      type: object
      properties:
        name:
          type: string
      additionalProperties: false
    FlexibleObject:
      type: object
      properties:
        name:
          type: string
      additionalProperties: true
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
	)
	require.NoError(t, err)

	typesFile := result.GetFile("types.go")
	require.NotNil(t, typesFile)
	content := string(typesFile.Content)

	// Both structs should be generated
	assert.Contains(t, content, "type StrictObject struct")
	assert.Contains(t, content, "type FlexibleObject struct")

	// Find the StrictObject struct and verify it doesn't have AdditionalProperties
	strictStart := strings.Index(content, "type StrictObject struct")
	strictEnd := strings.Index(content[strictStart:], "}")
	strictStruct := content[strictStart : strictStart+strictEnd+1]
	assert.NotContains(t, strictStruct, "AdditionalProperties")

	// FlexibleObject should have AdditionalProperties
	flexStart := strings.Index(content, "type FlexibleObject struct")
	flexEnd := strings.Index(content[flexStart:], "}")
	flexStruct := content[flexStart : flexStart+flexEnd+1]
	assert.Contains(t, flexStruct, "AdditionalProperties")
}
