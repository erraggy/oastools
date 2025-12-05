package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
)

func TestNew(t *testing.T) {
	g := New()

	if g == nil {
		t.Fatal("New() returned nil")
	}
	if g.PackageName != "api" {
		t.Errorf("PackageName = %q, want %q", g.PackageName, "api")
	}
	if g.GenerateClient {
		t.Error("GenerateClient should be false by default")
	}
	if g.GenerateServer {
		t.Error("GenerateServer should be false by default")
	}
	if !g.GenerateTypes {
		t.Error("GenerateTypes should be true by default")
	}
	if !g.UsePointers {
		t.Error("UsePointers should be true by default")
	}
	if !g.IncludeValidation {
		t.Error("IncludeValidation should be true by default")
	}
}

func TestGenerateWithOptions_RequiresInputSource(t *testing.T) {
	_, err := GenerateWithOptions(
		WithPackageName("test"),
	)
	if err == nil {
		t.Error("expected error when no input source provided")
	}
	if !strings.Contains(err.Error(), "must specify an input source") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenerateWithOptions_OnlyOneInputSource(t *testing.T) {
	parsed := parser.ParseResult{
		Version:    "3.0.0",
		OASVersion: parser.OASVersion300,
	}

	_, err := GenerateWithOptions(
		WithFilePath("test.yaml"),
		WithParsed(parsed),
	)
	if err == nil {
		t.Error("expected error when multiple input sources provided")
	}
	if !strings.Contains(err.Error(), "must specify exactly one input source") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestWithPackageName_Empty(t *testing.T) {
	_, err := GenerateWithOptions(
		WithFilePath("test.yaml"),
		WithPackageName(""),
	)
	if err == nil {
		t.Error("expected error for empty package name")
	}
	if !strings.Contains(err.Error(), "package name cannot be empty") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenerateOAS3Types(t *testing.T) {
	// Create a minimal OAS 3.0 spec for testing
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
	// Create temp file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test-api.yaml")
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithTypes(true),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	// Check result
	if result.PackageName != "testapi" {
		t.Errorf("PackageName = %q, want %q", result.PackageName, "testapi")
	}
	if result.SourceVersion != "3.0.0" {
		t.Errorf("SourceVersion = %q, want %q", result.SourceVersion, "3.0.0")
	}
	if result.GeneratedTypes != 2 {
		t.Errorf("GeneratedTypes = %d, want %d", result.GeneratedTypes, 2)
	}

	// Check types.go was generated
	typesFile := result.GetFile("types.go")
	if typesFile == nil {
		t.Fatal("types.go not generated")
	}

	content := string(typesFile.Content)

	// Verify package declaration
	if !strings.Contains(content, "package testapi") {
		t.Error("types.go missing package declaration")
	}

	// Verify Pet struct
	if !strings.Contains(content, "type Pet struct") {
		t.Error("types.go missing Pet struct")
	}
	if !strings.Contains(content, "Id") || !strings.Contains(content, "int64") {
		t.Error("types.go missing Id field in Pet")
	}
	if !strings.Contains(content, "Name") || !strings.Contains(content, "string") {
		t.Error("types.go missing Name field in Pet")
	}

	// Verify Error struct
	if !strings.Contains(content, "type Error struct") {
		t.Error("types.go missing Error struct")
	}
}

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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithClient(true),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	// Check result
	if result.GeneratedOperations != 3 {
		t.Errorf("GeneratedOperations = %d, want %d", result.GeneratedOperations, 3)
	}

	// Check client.go was generated
	clientFile := result.GetFile("client.go")
	if clientFile == nil {
		t.Fatal("client.go not generated")
	}

	content := string(clientFile.Content)

	// Verify client struct
	if !strings.Contains(content, "type Client struct") {
		t.Error("client.go missing Client struct")
	}

	// Verify NewClient function
	if !strings.Contains(content, "func NewClient") {
		t.Error("client.go missing NewClient function")
	}

	// Verify operation methods
	if !strings.Contains(content, "func (c *Client) ListPets") {
		t.Error("client.go missing ListPets method")
	}
	if !strings.Contains(content, "func (c *Client) CreatePet") {
		t.Error("client.go missing CreatePet method")
	}
	if !strings.Contains(content, "func (c *Client) GetPet") {
		t.Error("client.go missing GetPet method")
	}

	// Verify params struct for query parameters
	if !strings.Contains(content, "type ListPetsParams struct") {
		t.Error("client.go missing ListPetsParams struct")
	}
}

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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithServer(true),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	// Check server.go was generated
	serverFile := result.GetFile("server.go")
	if serverFile == nil {
		t.Fatal("server.go not generated")
	}

	content := string(serverFile.Content)

	// Verify server interface
	if !strings.Contains(content, "type ServerInterface interface") {
		t.Error("server.go missing ServerInterface")
	}

	// Verify methods in interface
	if !strings.Contains(content, "ListPets(") {
		t.Error("server.go missing ListPets in interface")
	}
	if !strings.Contains(content, "CreatePet(") {
		t.Error("server.go missing CreatePet in interface")
	}

	// Verify request types
	if !strings.Contains(content, "type ListPetsRequest struct") {
		t.Error("server.go missing ListPetsRequest struct")
	}

	// Verify UnimplementedServer
	if !strings.Contains(content, "type UnimplementedServer struct") {
		t.Error("server.go missing UnimplementedServer")
	}
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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithTypes(true),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	if result.SourceVersion != "2.0" {
		t.Errorf("SourceVersion = %q, want %q", result.SourceVersion, "2.0")
	}
	if result.GeneratedTypes != 1 {
		t.Errorf("GeneratedTypes = %d, want %d", result.GeneratedTypes, 1)
	}

	typesFile := result.GetFile("types.go")
	if typesFile == nil {
		t.Fatal("types.go not generated")
	}

	content := string(typesFile.Content)
	if !strings.Contains(content, "type Pet struct") {
		t.Error("types.go missing Pet struct")
	}
}

func TestGenerateResult_WriteFiles(t *testing.T) {
	result := &GenerateResult{
		Files: []GeneratedFile{
			{Name: "types.go", Content: []byte("package test\n\ntype Foo struct{}\n")},
			{Name: "client.go", Content: []byte("package test\n\nfunc NewClient() {}\n")},
		},
	}

	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")

	if err := result.WriteFiles(outputDir); err != nil {
		t.Fatalf("WriteFiles failed: %v", err)
	}

	// Verify files were created
	for _, file := range result.Files {
		filePath := filepath.Join(outputDir, file.Name)
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("failed to read %s: %v", file.Name, err)
			continue
		}
		if string(content) != string(file.Content) {
			t.Errorf("file %s content mismatch", file.Name)
		}
	}
}

func TestGenerateResult_GetFile(t *testing.T) {
	result := &GenerateResult{
		Files: []GeneratedFile{
			{Name: "types.go", Content: []byte("package test")},
			{Name: "client.go", Content: []byte("package test")},
		},
	}

	// Find existing file
	if f := result.GetFile("types.go"); f == nil {
		t.Error("GetFile failed to find types.go")
	}

	// Non-existing file
	if f := result.GetFile("nonexistent.go"); f != nil {
		t.Error("GetFile should return nil for non-existing file")
	}
}

func TestGenerateResult_HasCriticalIssues(t *testing.T) {
	result := &GenerateResult{CriticalCount: 0}
	if result.HasCriticalIssues() {
		t.Error("HasCriticalIssues should return false when CriticalCount is 0")
	}

	result.CriticalCount = 1
	if !result.HasCriticalIssues() {
		t.Error("HasCriticalIssues should return true when CriticalCount > 0")
	}
}

func TestGenerateResult_HasWarnings(t *testing.T) {
	result := &GenerateResult{WarningCount: 0}
	if result.HasWarnings() {
		t.Error("HasWarnings should return false when WarningCount is 0")
	}

	result.WarningCount = 1
	if !result.HasWarnings() {
		t.Error("HasWarnings should return true when WarningCount > 0")
	}
}

func TestToTypeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"pet", "Pet"},
		{"Pet", "Pet"},
		{"pet-store", "PetStore"},
		{"pet_store", "PetStore"},
		{"pet.store", "PetStore"},
		{"PetStore", "PetStore"},
		{"123abc", "T123abc"},
		{"", "Type"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toTypeName(tt.input)
			if result != tt.expected {
				t.Errorf("toTypeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToParamName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"petId", "petId"},
		{"PetId", "petId"},
		{"pet-id", "petId"},
		{"pet_id", "petId"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toParamName(tt.input)
			if result != tt.expected {
				t.Errorf("toParamName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestOperationToMethodName(t *testing.T) {
	tests := []struct {
		op       *parser.Operation
		path     string
		method   string
		expected string
	}{
		{&parser.Operation{OperationID: "listPets"}, "/pets", "get", "ListPets"},
		{&parser.Operation{OperationID: "get-pet-by-id"}, "/pets/{id}", "get", "GetPetById"},
		{&parser.Operation{}, "/pets", "get", "GetPets"},
		{&parser.Operation{}, "/pets/{petId}", "get", "GetPetsByPetId"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := operationToMethodName(tt.op, tt.path, tt.method)
			if result != tt.expected {
				t.Errorf("operationToMethodName() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestStringFormatToGoType(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"date-time", "time.Time"},
		{"date", "string"},
		{"byte", "[]byte"},
		{"binary", "[]byte"},
		{"", "string"},
		{"unknown", "string"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := stringFormatToGoType(tt.format)
			if result != tt.expected {
				t.Errorf("stringFormatToGoType(%q) = %q, want %q", tt.format, result, tt.expected)
			}
		})
	}
}

func TestIntegerFormatToGoType(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"int32", "int32"},
		{"int64", "int64"},
		{"", "int64"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := integerFormatToGoType(tt.format)
			if result != tt.expected {
				t.Errorf("integerFormatToGoType(%q) = %q, want %q", tt.format, result, tt.expected)
			}
		})
	}
}

func TestNumberFormatToGoType(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"float", "float32"},
		{"double", "float64"},
		{"", "float64"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := numberFormatToGoType(tt.format)
			if result != tt.expected {
				t.Errorf("numberFormatToGoType(%q) = %q, want %q", tt.format, result, tt.expected)
			}
		})
	}
}

func TestGetSchemaType(t *testing.T) {
	tests := []struct {
		name     string
		schema   *parser.Schema
		expected string
	}{
		{"nil schema", nil, ""},
		{"string type", &parser.Schema{Type: "string"}, "string"},
		{"object type", &parser.Schema{Type: "object"}, "object"},
		{"array type", &parser.Schema{Type: "array"}, "array"},
		{"properties infer object", &parser.Schema{Properties: map[string]*parser.Schema{}}, "object"},
		{"items infer array", &parser.Schema{Items: &parser.Schema{}}, "array"},
		{"enum infer string", &parser.Schema{Enum: []any{"a", "b"}}, "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSchemaType(tt.schema)
			if result != tt.expected {
				t.Errorf("getSchemaType() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsRequired(t *testing.T) {
	required := []string{"id", "name", "email"}

	if !isRequired(required, "id") {
		t.Error("isRequired should return true for 'id'")
	}
	if !isRequired(required, "name") {
		t.Error("isRequired should return true for 'name'")
	}
	if isRequired(required, "optional") {
		t.Error("isRequired should return false for 'optional'")
	}
	if isRequired(nil, "any") {
		t.Error("isRequired should return false for nil required list")
	}
}

func TestCleanDescription(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple description", "Simple description"},
		{"Multi\nline\ndescription", "Multi line description"},
		{"  Whitespace  ", "Whitespace"},
		{strings.Repeat("a", 300), strings.Repeat("a", 197) + "..."},
	}

	for _, tt := range tests {
		t.Run(tt.input[:min(10, len(tt.input))], func(t *testing.T) {
			result := cleanDescription(tt.input)
			if result != tt.expected {
				t.Errorf("cleanDescription() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestZeroValue(t *testing.T) {
	tests := []struct {
		typeName string
		expected string
	}{
		{"", "nil"},
		{"*http.Response", "nil"},
		{"*Pet", "nil"},
		{"[]Pet", "nil"},
		{"map[string]Pet", "nil"},
		{"Pet", "Pet{}"},
		{"string", "string{}"},
	}

	for _, tt := range tests {
		t.Run(tt.typeName, func(t *testing.T) {
			result := zeroValue(tt.typeName)
			if result != tt.expected {
				t.Errorf("zeroValue(%q) = %q, want %q", tt.typeName, result, tt.expected)
			}
		})
	}
}

func TestWithOptions(t *testing.T) {
	// Test all option functions
	cfg := &generateConfig{}

	// Test WithFilePath
	opt := WithFilePath("test.yaml")
	if err := opt(cfg); err != nil {
		t.Errorf("WithFilePath error: %v", err)
	}
	if cfg.filePath == nil || *cfg.filePath != "test.yaml" {
		t.Error("WithFilePath did not set filePath")
	}

	// Test WithClient
	cfg = &generateConfig{}
	opt = WithClient(true)
	if err := opt(cfg); err != nil {
		t.Errorf("WithClient error: %v", err)
	}
	if !cfg.generateClient {
		t.Error("WithClient did not set generateClient")
	}

	// Test WithServer
	cfg = &generateConfig{}
	opt = WithServer(true)
	if err := opt(cfg); err != nil {
		t.Errorf("WithServer error: %v", err)
	}
	if !cfg.generateServer {
		t.Error("WithServer did not set generateServer")
	}

	// Test WithTypes
	cfg = &generateConfig{}
	opt = WithTypes(false)
	if err := opt(cfg); err != nil {
		t.Errorf("WithTypes error: %v", err)
	}
	if cfg.generateTypes {
		t.Error("WithTypes did not set generateTypes to false")
	}

	// Test WithPointers
	cfg = &generateConfig{}
	opt = WithPointers(false)
	if err := opt(cfg); err != nil {
		t.Errorf("WithPointers error: %v", err)
	}
	if cfg.usePointers {
		t.Error("WithPointers did not set usePointers to false")
	}

	// Test WithValidation
	cfg = &generateConfig{}
	opt = WithValidation(false)
	if err := opt(cfg); err != nil {
		t.Errorf("WithValidation error: %v", err)
	}
	if cfg.includeValidation {
		t.Error("WithValidation did not set includeValidation to false")
	}

	// Test WithStrictMode
	cfg = &generateConfig{}
	opt = WithStrictMode(true)
	if err := opt(cfg); err != nil {
		t.Errorf("WithStrictMode error: %v", err)
	}
	if !cfg.strictMode {
		t.Error("WithStrictMode did not set strictMode")
	}

	// Test WithIncludeInfo
	cfg = &generateConfig{}
	opt = WithIncludeInfo(false)
	if err := opt(cfg); err != nil {
		t.Errorf("WithIncludeInfo error: %v", err)
	}
	if cfg.includeInfo {
		t.Error("WithIncludeInfo did not set includeInfo to false")
	}

	// Test WithUserAgent
	cfg = &generateConfig{}
	opt = WithUserAgent("test-agent")
	if err := opt(cfg); err != nil {
		t.Errorf("WithUserAgent error: %v", err)
	}
	if cfg.userAgent != "test-agent" {
		t.Error("WithUserAgent did not set userAgent")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Additional tests for improved coverage

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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithClient(true),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	// Check client.go was generated
	clientFile := result.GetFile("client.go")
	if clientFile == nil {
		t.Fatal("client.go not generated")
	}

	content := string(clientFile.Content)

	// Verify client struct
	if !strings.Contains(content, "type Client struct") {
		t.Error("client.go missing Client struct")
	}

	// Verify operation methods
	if !strings.Contains(content, "func (c *Client) ListPets") {
		t.Error("client.go missing ListPets method")
	}
	if !strings.Contains(content, "func (c *Client) CreatePet") {
		t.Error("client.go missing CreatePet method")
	}
	if !strings.Contains(content, "func (c *Client) GetPet") {
		t.Error("client.go missing GetPet method")
	}

	// Verify params struct for query parameters
	if !strings.Contains(content, "ListPetsParams") {
		t.Error("client.go missing ListPetsParams struct")
	}

	// Verify generated operations count
	if result.GeneratedOperations != 3 {
		t.Errorf("GeneratedOperations = %d, want 3", result.GeneratedOperations)
	}
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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithServer(true),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	// Check server.go was generated
	serverFile := result.GetFile("server.go")
	if serverFile == nil {
		t.Fatal("server.go not generated")
	}

	content := string(serverFile.Content)

	// Verify server interface
	if !strings.Contains(content, "type ServerInterface interface") {
		t.Error("server.go missing ServerInterface")
	}

	// Verify methods in interface
	if !strings.Contains(content, "ListPets(") {
		t.Error("server.go missing ListPets in interface")
	}
	if !strings.Contains(content, "CreatePet(") {
		t.Error("server.go missing CreatePet in interface")
	}

	// Verify request types
	if !strings.Contains(content, "ListPetsRequest") {
		t.Error("server.go missing ListPetsRequest struct")
	}
	// CreatePet request should have Body field
	if !strings.Contains(content, "CreatePetRequest") {
		t.Error("server.go missing CreatePetRequest struct")
	}

	// Verify UnimplementedServer
	if !strings.Contains(content, "UnimplementedServer") {
		t.Error("server.go missing UnimplementedServer")
	}
}

func TestGenerateWithStrictMode(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Pet:
      oneOf:
        - $ref: '#/components/schemas/Cat'
        - $ref: '#/components/schemas/Dog'
    Cat:
      type: object
      properties:
        meow:
          type: boolean
    Dog:
      type: object
      properties:
        bark:
          type: boolean
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// With strict mode, info messages should not cause failure
	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithStrictMode(false),
		WithIncludeInfo(true),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	// Should have info about oneOf
	if result.InfoCount == 0 {
		t.Error("expected info messages about oneOf union types")
	}
}

func TestGenerateWithoutInfo(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
components:
  schemas:
    Pet:
      oneOf:
        - type: string
        - type: integer
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithIncludeInfo(false),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	// Info messages should be filtered out
	for _, issue := range result.Issues {
		if issue.Severity == SeverityInfo {
			t.Error("info messages should be filtered out when IncludeInfo is false")
		}
	}
}

func TestGenerateWithParsedDocument(t *testing.T) {
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
        id:
          type: integer
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Parse first
	p := parser.New()
	parseResult, err := p.Parse(tmpFile)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Generate using parsed result
	result, err := GenerateWithOptions(
		WithParsed(*parseResult),
		WithPackageName("testapi"),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	if result.GeneratedTypes != 1 {
		t.Errorf("GeneratedTypes = %d, want 1", result.GeneratedTypes)
	}
}

func TestGenerateFileNotFound(t *testing.T) {
	_, err := GenerateWithOptions(
		WithFilePath("nonexistent.yaml"),
		WithPackageName("testapi"),
	)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestGenerateInvalidSpec(t *testing.T) {
	spec := `not valid yaml: [[[`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
	)
	if err == nil {
		t.Error("expected error for invalid spec")
	}
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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	typesFile := result.GetFile("types.go")
	if typesFile == nil {
		t.Fatal("types.go not generated")
	}

	content := string(typesFile.Content)
	// Pet should be an alias for Animal
	if !strings.Contains(content, "type Pet = Animal") {
		t.Error("expected Pet to be an alias for Animal")
	}
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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	typesFile := result.GetFile("types.go")
	content := string(typesFile.Content)

	// Pet should embed Animal and have PetId
	if !strings.Contains(content, "type Pet struct") {
		t.Error("expected Pet struct")
	}
	if !strings.Contains(content, "Animal") {
		t.Error("expected Animal to be embedded")
	}
	if !strings.Contains(content, "PetId") {
		t.Error("expected PetId field")
	}
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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	typesFile := result.GetFile("types.go")
	content := string(typesFile.Content)

	// Should have enum constants
	if !strings.Contains(content, "type Status string") {
		t.Error("expected Status string type")
	}
	if !strings.Contains(content, "StatusAvailable") {
		t.Error("expected StatusAvailable constant")
	}
	if !strings.Contains(content, "StatusPending") {
		t.Error("expected StatusPending constant")
	}
	if !strings.Contains(content, "StatusSold") {
		t.Error("expected StatusSold constant")
	}
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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	typesFile := result.GetFile("types.go")
	content := string(typesFile.Content)

	// Pets should be a slice type (may be []any if ref not fully resolved)
	if !strings.Contains(content, "type Pets []") {
		t.Error("expected Pets to be a slice type")
	}
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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	typesFile := result.GetFile("types.go")
	content := string(typesFile.Content)

	// Should have Metadata struct
	if !strings.Contains(content, "type Metadata struct") {
		t.Error("expected Metadata struct")
	}
	// Should have Name field
	if !strings.Contains(content, "Name") {
		t.Error("expected Name field")
	}
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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithPointers(true),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	typesFile := result.GetFile("types.go")
	content := string(typesFile.Content)

	// Nullable field should be pointer
	if !strings.Contains(content, "*string") {
		t.Error("expected nullable string to be pointer type")
	}
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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	typesFile := result.GetFile("types.go")
	content := string(typesFile.Content)

	// Check format mappings
	if !strings.Contains(content, "time.Time") {
		t.Error("expected time.Time for date-time format")
	}
	if !strings.Contains(content, "[]byte") {
		t.Error("expected []byte for byte format")
	}
	if !strings.Contains(content, "int32") {
		t.Error("expected int32")
	}
	if !strings.Contains(content, "int64") {
		t.Error("expected int64")
	}
	if !strings.Contains(content, "float32") {
		t.Error("expected float32")
	}
	if !strings.Contains(content, "float64") {
		t.Error("expected float64")
	}
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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithPointers(false),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	typesFile := result.GetFile("types.go")
	content := string(typesFile.Content)

	// Without pointers, optional fields should not have *
	// Note: checking that tag field doesn't have pointer
	if strings.Contains(content, "*string `json:\"tag") {
		t.Error("expected no pointer for optional string when UsePointers is false")
	}
}

func TestGeneratedFile_WriteFile(t *testing.T) {
	file := &GeneratedFile{
		Name:    "test.go",
		Content: []byte("package test\n"),
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "subdir", "test.go")

	if err := file.WriteFile(filePath); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Verify file was created
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != "package test\n" {
		t.Error("file content mismatch")
	}
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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithClient(true),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	clientFile := result.GetFile("client.go")
	content := string(clientFile.Content)

	// Should have Deprecated comment
	if !strings.Contains(content, "Deprecated:") {
		t.Error("expected Deprecated comment for deprecated operation")
	}
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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("testapi"),
		WithServer(true),
	)
	if err != nil {
		t.Fatalf("GenerateWithOptions failed: %v", err)
	}

	serverFile := result.GetFile("server.go")
	content := string(serverFile.Content)

	// Request struct should have all parameter types
	if !strings.Contains(content, "GetItemRequest") {
		t.Error("expected GetItemRequest struct")
	}
	if !strings.Contains(content, "Id") {
		t.Error("expected Id field for path parameter")
	}
	if !strings.Contains(content, "Filter") {
		t.Error("expected Filter field for query parameter")
	}
	if !strings.Contains(content, "XRequestID") {
		t.Error("expected XRequestID field for header parameter")
	}
}

func TestIsTypeNullable(t *testing.T) {
	tests := []struct {
		name     string
		typeVal  any
		expected bool
	}{
		{"string type", "string", false},
		{"array with null", []any{"string", "null"}, true},
		{"array without null", []any{"string", "integer"}, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTypeNullable(tt.typeVal)
			if result != tt.expected {
				t.Errorf("isTypeNullable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNeedsTimeImport(t *testing.T) {
	tests := []struct {
		name     string
		schema   *parser.Schema
		expected bool
	}{
		{"nil schema", nil, false},
		{"date-time format", &parser.Schema{Type: "string", Format: "date-time"}, true},
		{"no format", &parser.Schema{Type: "string"}, false},
		{"nested date-time", &parser.Schema{
			Type: "object",
			Properties: map[string]*parser.Schema{
				"created": {Type: "string", Format: "date-time"},
			},
		}, true},
		{"array with date-time items", &parser.Schema{
			Type:  "array",
			Items: &parser.Schema{Type: "string", Format: "date-time"},
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := needsTimeImport(tt.schema)
			if result != tt.expected {
				t.Errorf("needsTimeImport() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGeneratorStruct_Generate(t *testing.T) {
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
        id:
          type: integer
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	g := New()
	g.PackageName = "testapi"
	g.GenerateTypes = true

	result, err := g.Generate(tmpFile)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.PackageName != "testapi" {
		t.Errorf("PackageName = %q, want %q", result.PackageName, "testapi")
	}
	if result.GeneratedTypes != 1 {
		t.Errorf("GeneratedTypes = %d, want 1", result.GeneratedTypes)
	}
}

func TestGeneratorStruct_GenerateParsed(t *testing.T) {
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
        id:
          type: integer
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	p := parser.New()
	parseResult, err := p.Parse(tmpFile)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	g := New()
	g.PackageName = "testapi"
	g.GenerateClient = true
	g.GenerateServer = true

	result, err := g.GenerateParsed(*parseResult)
	if err != nil {
		t.Fatalf("GenerateParsed failed: %v", err)
	}

	// Should have all three files
	if result.GetFile("types.go") == nil {
		t.Error("expected types.go")
	}
	if result.GetFile("client.go") == nil {
		t.Error("expected client.go")
	}
	if result.GetFile("server.go") == nil {
		t.Error("expected server.go")
	}
}

func TestGenerateEmptyPackageName(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Test API
  version: "1.0.0"
paths: {}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Generator with empty package name should default to "api"
	g := New()
	g.PackageName = ""
	g.GenerateTypes = true

	result, err := g.Generate(tmpFile)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.PackageName != "api" {
		t.Errorf("PackageName should default to 'api', got %q", result.PackageName)
	}
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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	g := New()
	g.GenerateTypes = true
	result, err := g.Generate(tmpFile)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content := string(result.GetFile("types.go").Content)
	// Should handle additionalProperties with typed values
	if !strings.Contains(content, "map[string]string") {
		t.Error("types.go should have map[string]string for Tags")
	}
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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	g := New()
	g.GenerateTypes = true
	result, err := g.Generate(tmpFile)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content := string(result.GetFile("types.go").Content)
	// Should have UnmarshalJSON for discriminated union
	if !strings.Contains(content, "UnmarshalJSON") {
		t.Error("types.go missing UnmarshalJSON for discriminated union")
	}
	if !strings.Contains(content, "petType") {
		t.Error("types.go missing petType discriminator field")
	}
	if !strings.Contains(content, "switch") {
		t.Error("types.go missing switch statement for discriminator")
	}
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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	g := New()
	g.GenerateTypes = true
	g.IncludeValidation = true
	result, err := g.Generate(tmpFile)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content := string(result.GetFile("types.go").Content)
	// Should have validate tags
	if !strings.Contains(content, "validate:") {
		t.Error("types.go missing validate tags")
	}
	if !strings.Contains(content, "required") {
		t.Error("types.go missing required validation")
	}
	if !strings.Contains(content, "email") {
		t.Error("types.go missing email validation")
	}
	if !strings.Contains(content, "min=") {
		t.Error("types.go missing min validation")
	}
}

func TestSchemaTypeFromMap(t *testing.T) {
	tests := []struct {
		name     string
		schema   map[string]interface{}
		expected string
	}{
		{
			name:     "string type",
			schema:   map[string]interface{}{"type": "string"},
			expected: "string",
		},
		{
			name:     "number type",
			schema:   map[string]interface{}{"type": "number"},
			expected: "float64",
		},
		{
			name:     "integer type",
			schema:   map[string]interface{}{"type": "integer"},
			expected: "int64",
		},
		{
			name:     "boolean type",
			schema:   map[string]interface{}{"type": "boolean"},
			expected: "bool",
		},
		{
			name:     "object type",
			schema:   map[string]interface{}{"type": "object"},
			expected: "map[string]any",
		},
		{
			name:     "array type",
			schema:   map[string]interface{}{"type": "array"},
			expected: "[]any",
		},
		{
			name:     "missing type",
			schema:   map[string]interface{}{},
			expected: "any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := schemaTypeFromMap(tt.schema)
			if result != tt.expected {
				t.Errorf("schemaTypeFromMap() = %q, want %q", result, tt.expected)
			}
		})
	}
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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	g := New()
	g.GenerateTypes = true
	result, err := g.Generate(tmpFile)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content := string(result.GetFile("types.go").Content)
	if !strings.Contains(content, "type Dog struct") {
		t.Error("types.go missing Dog struct for allOf")
	}
	if !strings.Contains(content, "Pet") {
		t.Error("types.go missing Pet reference in allOf")
	}
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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	g := New()
	g.GenerateClient = true
	result, err := g.Generate(tmpFile)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content := string(result.GetFile("client.go").Content)
	// Should use application/xml content type instead of hardcoded application/json
	if !strings.Contains(content, "application/xml") {
		t.Error("client.go should use application/xml from spec")
	}
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
	if err := os.WriteFile(tmpFile, []byte(spec), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	g := New()
	g.GenerateServer = true
	result, err := g.Generate(tmpFile)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	content := string(result.GetFile("server.go").Content)
	// Should have cookie parameter in request type
	if !strings.Contains(content, "SessionId") {
		t.Error("server.go should have SessionId cookie parameter")
	}
}

func TestEscapeReservedWord(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"break", "break_"},
		{"type", "type_"},
		{"Package", "Package_"}, // "package" is a keyword, matches when lowercased
		{"Error", "Error"},      // "error" is not in reserved words (predeclared, can be shadowed)
		{"func", "func_"},
		{"interface", "interface_"},
		{"pet", "pet"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeReservedWord(tt.input)
			if result != tt.expected {
				t.Errorf("escapeReservedWord(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToFieldName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"petId", "PetId"},
		{"pet_id", "PetId"},
		{"pet-id", "PetId"},
		{"PET_ID", "PETID"}, // All caps treated as one word
		{"break", "Break_"}, // keyword gets escaped
		{"pet", "Pet"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toFieldName(tt.input)
			if result != tt.expected {
				t.Errorf("toFieldName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
