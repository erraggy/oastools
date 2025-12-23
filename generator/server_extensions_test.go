package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testPetstoreSpec = `openapi: "3.0.3"
info:
  title: Petstore API
  version: "1.0.0"
paths:
  /pets:
    get:
      operationId: listPets
      summary: List all pets
      parameters:
        - name: limit
          in: query
          required: false
          schema:
            type: integer
            format: int32
      responses:
        '200':
          description: A list of pets
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pets'
        default:
          description: unexpected error
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
    post:
      operationId: createPet
      summary: Create a pet
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/NewPet'
      responses:
        '201':
          description: Pet created
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
        default:
          description: unexpected error
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
  /pets/{petId}:
    get:
      operationId: showPetById
      summary: Info for a specific pet
      parameters:
        - name: petId
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Expected response to a valid request
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Pet'
        default:
          description: unexpected error
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'
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
    NewPet:
      type: object
      required:
        - name
      properties:
        name:
          type: string
        tag:
          type: string
    Pets:
      type: array
      items:
        $ref: '#/components/schemas/Pet'
    Error:
      type: object
      required:
        - code
        - message
      properties:
        code:
          type: integer
          format: int32
        message:
          type: string
`

func TestGenerateServerResponses(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petstore.yaml")
	err := os.WriteFile(tmpFile, []byte(testPetstoreSpec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithServer(true),
		WithServerResponses(true),
	)
	require.NoError(t, err)

	// Check server_responses.go was generated
	respFile := result.GetFile("server_responses.go")
	require.NotNil(t, respFile, "server_responses.go not generated")

	content := string(respFile.Content)

	// Check helper functions
	assert.Contains(t, content, "func WriteJSON(w http.ResponseWriter, statusCode int, body any)")
	assert.Contains(t, content, "func WriteError(w http.ResponseWriter, statusCode int, message string)")
	assert.Contains(t, content, "func WriteNoContent(w http.ResponseWriter)")

	// Check response types for each operation
	assert.Contains(t, content, "type ListPetsResponse struct")
	assert.Contains(t, content, "type CreatePetResponse struct")
	assert.Contains(t, content, "type ShowPetByIdResponse struct")

	// Check status methods
	assert.Contains(t, content, "func (ListPetsResponse) Status200(")
	assert.Contains(t, content, "func (ListPetsResponse) StatusDefault(")
	assert.Contains(t, content, "func (CreatePetResponse) Status201(")

	// Check WriteTo method
	assert.Contains(t, content, "func (r ListPetsResponse) WriteTo(w http.ResponseWriter) error")
}

func TestGenerateServerBinder(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petstore.yaml")
	err := os.WriteFile(tmpFile, []byte(testPetstoreSpec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithServer(true),
		WithServerBinder(true),
	)
	require.NoError(t, err)

	// Check server_binder.go was generated
	binderFile := result.GetFile("server_binder.go")
	require.NotNil(t, binderFile, "server_binder.go not generated")

	content := string(binderFile.Content)

	// Check RequestBinder type
	assert.Contains(t, content, "type RequestBinder struct")
	assert.Contains(t, content, "func NewRequestBinder(parsed *parser.ParseResult)")
	assert.Contains(t, content, "func NewRequestBinderFromValidator(v *httpvalidator.Validator)")

	// Check BindingError type
	assert.Contains(t, content, "type BindingError struct")
	assert.Contains(t, content, "func (e *BindingError) Error() string")
	assert.Contains(t, content, "func (e *BindingError) ValidationErrors()")

	// Check bind methods for each operation
	assert.Contains(t, content, "func (b *RequestBinder) BindListPetsRequest(r *http.Request)")
	assert.Contains(t, content, "func (b *RequestBinder) BindCreatePetRequest(r *http.Request)")
	assert.Contains(t, content, "func (b *RequestBinder) BindShowPetByIdRequest(r *http.Request)")

	// Check parameter binding
	assert.Contains(t, content, `result.QueryParams["limit"]`)
	assert.Contains(t, content, `result.PathParams["petId"]`)
}

func TestGenerateServerMiddleware(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petstore.yaml")
	err := os.WriteFile(tmpFile, []byte(testPetstoreSpec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithServer(true),
		WithServerMiddleware(true),
	)
	require.NoError(t, err)

	// Check server_middleware.go was generated
	mwFile := result.GetFile("server_middleware.go")
	require.NotNil(t, mwFile, "server_middleware.go not generated")

	content := string(mwFile.Content)

	// Check middleware functions
	assert.Contains(t, content, "func ValidationMiddleware(parsed *parser.ParseResult)")
	assert.Contains(t, content, "func ValidationMiddlewareWithConfig(parsed *parser.ParseResult, cfg ValidationConfig)")

	// Check ValidationConfig type
	assert.Contains(t, content, "type ValidationConfig struct")
	assert.Contains(t, content, "IncludeRequestValidation bool")
	assert.Contains(t, content, "IncludeResponseValidation bool")
	assert.Contains(t, content, "StrictMode bool")
	assert.Contains(t, content, "OnValidationError func(")

	// Check default config
	assert.Contains(t, content, "func DefaultValidationConfig() ValidationConfig")

	// Check error response types
	assert.Contains(t, content, "type ValidationErrorResponse struct")

	// Check response recorder for response validation
	assert.Contains(t, content, "type responseRecorder struct")
}

func TestGenerateServerRouter(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petstore.yaml")
	err := os.WriteFile(tmpFile, []byte(testPetstoreSpec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithServer(true),
		WithServerRouter("stdlib"),
	)
	require.NoError(t, err)

	// Check server_router.go was generated
	routerFile := result.GetFile("server_router.go")
	require.NotNil(t, routerFile, "server_router.go not generated")

	content := string(routerFile.Content)

	// Check ServerRouter type
	assert.Contains(t, content, "type ServerRouter struct")
	assert.Contains(t, content, "func NewServerRouter(server ServerInterface, parsed *parser.ParseResult")

	// Check RouterOption
	assert.Contains(t, content, "type RouterOption func(*ServerRouter)")
	assert.Contains(t, content, "func WithMiddleware(mw ...func(http.Handler) http.Handler)")

	// Check handler methods
	assert.Contains(t, content, "func (r *ServerRouter) Handler() http.Handler")
	assert.Contains(t, content, "func (r *ServerRouter) ServeHTTP(w http.ResponseWriter, req *http.Request)")

	// Check PathParam helper
	assert.Contains(t, content, "func PathParam(r *http.Request, name string) string")

	// Check route handling
	assert.Contains(t, content, `case "/pets:GET"`)
	assert.Contains(t, content, `case "/pets:POST"`)
	assert.Contains(t, content, `case "/pets/{petId}:GET"`)

	// Check handler methods for operations
	assert.Contains(t, content, "func (r *ServerRouter) handleListPets(")
	assert.Contains(t, content, "func (r *ServerRouter) handleCreatePet(")
	assert.Contains(t, content, "func (r *ServerRouter) handleShowPetById(")
}

func TestGenerateServerStubs(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petstore.yaml")
	err := os.WriteFile(tmpFile, []byte(testPetstoreSpec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithServer(true),
		WithServerStubs(true),
		WithServerResponses(true), // Stubs reference response types
	)
	require.NoError(t, err)

	// Check server_stubs.go was generated
	stubsFile := result.GetFile("server_stubs.go")
	require.NotNil(t, stubsFile, "server_stubs.go not generated")

	content := string(stubsFile.Content)

	// Check StubServer type
	assert.Contains(t, content, "type StubServer struct")
	assert.Contains(t, content, "func NewStubServer() *StubServer")

	// Check function fields for each operation
	assert.Contains(t, content, "ListPetsFunc func(ctx context.Context, req *ListPetsRequest)")
	assert.Contains(t, content, "CreatePetFunc func(ctx context.Context, req *CreatePetRequest)")
	assert.Contains(t, content, "ShowPetByIdFunc func(ctx context.Context, req *ShowPetByIdRequest)")

	// Check method implementations
	assert.Contains(t, content, "func (s *StubServer) ListPets(ctx context.Context, req *ListPetsRequest)")
	assert.Contains(t, content, "func (s *StubServer) CreatePet(ctx context.Context, req *CreatePetRequest)")
	assert.Contains(t, content, "func (s *StubServer) ShowPetById(ctx context.Context, req *ShowPetByIdRequest)")

	// Check Reset method
	assert.Contains(t, content, "func (s *StubServer) Reset()")

	// Check With* options
	assert.Contains(t, content, "func WithListPets(fn func(")
	assert.Contains(t, content, "func WithCreatePet(fn func(")
	assert.Contains(t, content, "func WithShowPetById(fn func(")

	// Check NewStubServerWithOptions
	assert.Contains(t, content, "func NewStubServerWithOptions(opts ...StubServerOption) *StubServer")
}

func TestGenerateServerAll(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petstore.yaml")
	err := os.WriteFile(tmpFile, []byte(testPetstoreSpec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithServer(true),
		WithServerAll(), // Enable all server extensions
	)
	require.NoError(t, err)

	// Check all server extension files are generated
	assert.NotNil(t, result.GetFile("server.go"), "server.go not generated")
	assert.NotNil(t, result.GetFile("server_responses.go"), "server_responses.go not generated")
	assert.NotNil(t, result.GetFile("server_binder.go"), "server_binder.go not generated")
	assert.NotNil(t, result.GetFile("server_middleware.go"), "server_middleware.go not generated")
	assert.NotNil(t, result.GetFile("server_router.go"), "server_router.go not generated")
	assert.NotNil(t, result.GetFile("server_stubs.go"), "server_stubs.go not generated")

	// Verify we have 7 files total (types + server + 5 extensions)
	var goFileCount int
	for _, f := range result.Files {
		if strings.HasSuffix(f.Name, ".go") {
			goFileCount++
		}
	}
	assert.GreaterOrEqual(t, goFileCount, 7, "Expected at least 7 .go files")
}

func TestGenerateServerResponsesWithNoOperations(t *testing.T) {
	spec := `openapi: "3.0.0"
info:
  title: Empty API
  version: "1.0.0"
paths: {}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("emptyapi"),
		WithServer(true),
		WithServerResponses(true),
	)
	require.NoError(t, err)

	// With no paths, server_responses.go should not be generated
	respFile := result.GetFile("server_responses.go")
	assert.Nil(t, respFile, "server_responses.go should not be generated for empty paths")
}

func TestGenerateServerBinderPointerTypes(t *testing.T) {
	// Ensure optional query params are properly typed as pointers
	spec := `openapi: "3.0.3"
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
          required: false
          schema:
            type: integer
            format: int32
        - name: limit
          in: query
          required: true
          schema:
            type: integer
            format: int32
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
	require.NotNil(t, serverFile)

	content := string(serverFile.Content)

	// Optional param should be pointer (using regex to handle varying whitespace)
	assert.Regexp(t, `Page\s+\*int32`, content, "Optional param should be pointer type")
	// Required param should NOT be pointer
	assert.Regexp(t, `Limit\s+int32`, content, "Required param should not be pointer type")
	assert.NotRegexp(t, `Limit\s+\*int32`, content, "Required param should not be pointer type")
	// Ensure no double pointers
	assert.NotContains(t, content, "**int32", "Should not have double pointers")
}

func TestServerRouterWithValidation(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petstore.yaml")
	err := os.WriteFile(tmpFile, []byte(testPetstoreSpec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithServer(true),
		WithServerRouter("stdlib"),
		WithServerMiddleware(true),
	)
	require.NoError(t, err)

	// Both router and middleware should be generated
	assert.NotNil(t, result.GetFile("server_router.go"))
	assert.NotNil(t, result.GetFile("server_middleware.go"))

	// Middleware should have validation functions
	mwContent := string(result.GetFile("server_middleware.go").Content)
	assert.Contains(t, mwContent, "v.ValidateRequest(r)")
}
