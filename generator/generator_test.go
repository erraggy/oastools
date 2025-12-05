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
