package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erraggy/oastools/parser"
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

func TestGenerateServerExtensionsOAS2(t *testing.T) {
	// OAS 2.0 spec - server extensions are stubs but should not error
	spec := `swagger: "2.0"
info:
  title: OAS2 API
  version: "1.0.0"
basePath: /api
paths:
  /pets:
    get:
      operationId: listPets
      responses:
        200:
          description: OK
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "oas2.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	// Enable all server extensions - should succeed without error
	// even though OAS2 stubs don't generate output
	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("oas2api"),
		WithServer(true),
		WithServerAll(),
	)
	require.NoError(t, err)
	assert.True(t, result.Success)

	// Server.go should still be generated
	assert.NotNil(t, result.GetFile("server.go"))

	// Extension files won't be generated for OAS2 (stubs return nil)
	// This is expected behavior - no error, just no output
}

func TestGenerateServerRouterChi(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petstore.yaml")
	err := os.WriteFile(tmpFile, []byte(testPetstoreSpec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithServer(true),
		WithServerRouter("chi"),
	)
	require.NoError(t, err)

	// Check server_router.go was generated
	routerFile := result.GetFile("server_router.go")
	require.NotNil(t, routerFile, "server_router.go not generated")

	content := string(routerFile.Content)

	// Check chi imports
	assert.Contains(t, content, `"github.com/go-chi/chi/v5"`)

	// Check NewChiRouter function
	assert.Contains(t, content, "func NewChiRouter(server ServerInterface, opts ...RouterOption) chi.Router")

	// Check RouterOption type for chi
	assert.Contains(t, content, "type RouterOption func(chi.Router)")
	assert.Contains(t, content, "func WithMiddleware(mw ...func(http.Handler) http.Handler)")

	// Check chi route registration
	assert.Contains(t, content, `r.Get("/pets"`)
	assert.Contains(t, content, `r.Post("/pets"`)
	assert.Contains(t, content, `r.Get("/pets/{petId}"`)

	// Check handler functions for operations (chi uses Chi suffix)
	assert.Contains(t, content, "func handleListPetsChi(")
	assert.Contains(t, content, "func handleCreatePetChi(")
	assert.Contains(t, content, "func handleShowPetByIdChi(")

	// Check chi.URLParam usage for path params
	assert.Contains(t, content, `chi.URLParam(req, "petId")`)

	// Check error handler support
	assert.Contains(t, content, "type ErrorHandler func(r *http.Request, err error)")
	assert.Contains(t, content, "func WithErrorHandler(handler ErrorHandler)")
}

func TestGenerateServerRouterChiWithErrorHandler(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petstore.yaml")
	err := os.WriteFile(tmpFile, []byte(testPetstoreSpec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithServer(true),
		WithServerRouter("chi"),
	)
	require.NoError(t, err)

	routerFile := result.GetFile("server_router.go")
	require.NotNil(t, routerFile)

	content := string(routerFile.Content)

	// Check error handler middleware pattern
	assert.Contains(t, content, "context.WithValue")
	assert.Contains(t, content, "errorHandlerKey{}")
	assert.Contains(t, content, "getErrorHandler(req)")

	// Check generic error message is used
	assert.Contains(t, content, `"internal server error"`)
}

func TestGenerateServerRouterChiQueryMethod(t *testing.T) {
	// OAS 3.2+ spec with QUERY method
	spec := `openapi: "3.2.0"
info:
  title: Query API
  version: "1.0.0"
paths:
  /search:
    query:
      operationId: searchData
      responses:
        '200':
          description: Success
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "query.yaml")
	err := os.WriteFile(tmpFile, []byte(spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("queryapi"),
		WithServer(true),
		WithServerRouter("chi"),
	)
	require.NoError(t, err)

	routerFile := result.GetFile("server_router.go")
	require.NotNil(t, routerFile, "server_router.go not generated")

	content := string(routerFile.Content)

	// Check that QUERY method uses chi's generic Method() function
	assert.Contains(t, content, `r.Method("QUERY", "/search"`)
	assert.Contains(t, content, "handleSearchDataChi")
}

func TestGenerateServerInvalidRouter(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petstore.yaml")
	err := os.WriteFile(tmpFile, []byte(testPetstoreSpec), 0600)
	require.NoError(t, err)

	// Invalid router value should error
	_, err = GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithServer(true),
		WithServerRouter("invalid"),
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid server router")
}

func TestGenerateServerRouterEmptyPaths(t *testing.T) {
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
		WithServerRouter("stdlib"),
	)
	require.NoError(t, err)

	// Router should not be generated for empty paths
	routerFile := result.GetFile("server_router.go")
	assert.Nil(t, routerFile, "server_router.go should not be generated for empty paths")
}

func TestGenerateServerRouterChiEmptyPaths(t *testing.T) {
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
		WithServerRouter("chi"),
	)
	require.NoError(t, err)

	// Chi router should not be generated for empty paths
	routerFile := result.GetFile("server_router.go")
	assert.Nil(t, routerFile, "server_router.go should not be generated for empty paths with chi router")
}

func TestGenerateServerStubsEmptyPaths(t *testing.T) {
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
		WithServerStubs(true),
	)
	require.NoError(t, err)

	// Stubs should not be generated for empty paths
	stubsFile := result.GetFile("server_stubs.go")
	assert.Nil(t, stubsFile, "server_stubs.go should not be generated for empty paths")
}

func TestGenerateServerBinderEmptyPaths(t *testing.T) {
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
		WithServerBinder(true),
	)
	require.NoError(t, err)

	// Binder should not be generated for empty paths
	binderFile := result.GetFile("server_binder.go")
	assert.Nil(t, binderFile, "server_binder.go should not be generated for empty paths")
}

func TestGenerateServerRouterWithErrorHandler(t *testing.T) {
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

	routerFile := result.GetFile("server_router.go")
	require.NotNil(t, routerFile)

	content := string(routerFile.Content)

	// Check WithErrorHandler option is generated
	assert.Contains(t, content, "WithErrorHandler")
	assert.Contains(t, content, "func(r *http.Request, err error)")
	assert.Contains(t, content, "errorHandler")

	// Check error handler is called in handlers
	assert.Contains(t, content, "r.errorHandler(req, err)")

	// Verify generic error message is used
	assert.Contains(t, content, `"internal server error"`)
}

// OAS 2.0 Petstore test spec
const testPetstoreOAS2Spec = `swagger: "2.0"
info:
  title: Petstore API
  version: "1.0.0"
host: api.example.com
basePath: /v1
schemes:
  - https
consumes:
  - application/json
produces:
  - application/json
paths:
  /pets:
    get:
      operationId: listPets
      summary: List all pets
      parameters:
        - name: limit
          in: query
          required: false
          type: integer
          format: int32
      responses:
        '200':
          description: A list of pets
          schema:
            $ref: '#/definitions/Pets'
        default:
          description: unexpected error
          schema:
            $ref: '#/definitions/Error'
    post:
      operationId: createPet
      summary: Create a pet
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
        default:
          description: unexpected error
          schema:
            $ref: '#/definitions/Error'
  /pets/{petId}:
    get:
      operationId: showPetById
      summary: Info for a specific pet
      parameters:
        - name: petId
          in: path
          required: true
          type: string
      responses:
        '200':
          description: Expected response to a valid request
          schema:
            $ref: '#/definitions/Pet'
        default:
          description: unexpected error
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
  Pets:
    type: array
    items:
      $ref: '#/definitions/Pet'
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

func TestGenerateServerResponses_OAS2(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petstore.json")
	err := os.WriteFile(tmpFile, []byte(testPetstoreOAS2Spec), 0600)
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
	require.NotNil(t, respFile, "server_responses.go not generated for OAS 2.0")

	content := string(respFile.Content)

	// Check helper functions
	assert.Contains(t, content, "func WriteJSON(w http.ResponseWriter, statusCode int, body any)")
	assert.Contains(t, content, "func WriteError(w http.ResponseWriter, statusCode int, message string)")

	// Check response types for each operation
	assert.Contains(t, content, "type ListPetsResponse struct")
	assert.Contains(t, content, "type CreatePetResponse struct")
	assert.Contains(t, content, "type ShowPetByIdResponse struct")

	// Check status methods
	assert.Contains(t, content, "func (ListPetsResponse) Status200(")
	assert.Contains(t, content, "func (ListPetsResponse) StatusDefault(")
	assert.Contains(t, content, "func (CreatePetResponse) Status201(")
}

func TestGenerateServerBinder_OAS2(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petstore.json")
	err := os.WriteFile(tmpFile, []byte(testPetstoreOAS2Spec), 0600)
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
	require.NotNil(t, binderFile, "server_binder.go not generated for OAS 2.0")

	content := string(binderFile.Content)

	// Check RequestBinder type
	assert.Contains(t, content, "type RequestBinder struct")
	assert.Contains(t, content, "func NewRequestBinder(parsed *parser.ParseResult)")

	// Check bind methods for each operation
	assert.Contains(t, content, "func (b *RequestBinder) BindListPetsRequest(r *http.Request)")
	assert.Contains(t, content, "func (b *RequestBinder) BindCreatePetRequest(r *http.Request)")
	assert.Contains(t, content, "func (b *RequestBinder) BindShowPetByIdRequest(r *http.Request)")

	// Check parameter binding
	assert.Contains(t, content, `result.QueryParams["limit"]`)
	assert.Contains(t, content, `result.PathParams["petId"]`)
}

func TestGenerateServerMiddleware_OAS2(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petstore.json")
	err := os.WriteFile(tmpFile, []byte(testPetstoreOAS2Spec), 0600)
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
	require.NotNil(t, mwFile, "server_middleware.go not generated for OAS 2.0")

	content := string(mwFile.Content)

	// Check middleware functions
	assert.Contains(t, content, "func ValidationMiddleware(parsed *parser.ParseResult)")
	assert.Contains(t, content, "func ValidationMiddlewareWithConfig(parsed *parser.ParseResult, cfg ValidationConfig)")

	// Check ValidationConfig type
	assert.Contains(t, content, "type ValidationConfig struct")
}

func TestGenerateServerRouter_OAS2(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petstore.json")
	err := os.WriteFile(tmpFile, []byte(testPetstoreOAS2Spec), 0600)
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
	require.NotNil(t, routerFile, "server_router.go not generated for OAS 2.0")

	content := string(routerFile.Content)

	// Check ServerRouter type
	assert.Contains(t, content, "type ServerRouter struct")
	assert.Contains(t, content, "func NewServerRouter(server ServerInterface, parsed *parser.ParseResult")

	// Check route handling
	assert.Contains(t, content, `case "/pets:GET"`)
	assert.Contains(t, content, `case "/pets:POST"`)
	assert.Contains(t, content, `case "/pets/{petId}:GET"`)

	// Check handler methods for operations
	assert.Contains(t, content, "func (r *ServerRouter) handleListPets(")
	assert.Contains(t, content, "func (r *ServerRouter) handleCreatePet(")
	assert.Contains(t, content, "func (r *ServerRouter) handleShowPetById(")
}

func TestGenerateServerRouter_OAS2_Chi(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petstore.json")
	err := os.WriteFile(tmpFile, []byte(testPetstoreOAS2Spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithServer(true),
		WithServerRouter("chi"),
	)
	require.NoError(t, err)

	// Check server_router.go was generated
	routerFile := result.GetFile("server_router.go")
	require.NotNil(t, routerFile, "server_router.go not generated for OAS 2.0 with chi")

	content := string(routerFile.Content)

	// Check chi-specific imports and functions
	assert.Contains(t, content, `"github.com/go-chi/chi/v5"`)
	assert.Contains(t, content, "func NewChiRouter(server ServerInterface")

	// Check chi route registration
	assert.Contains(t, content, `r.Get("/pets"`)
	assert.Contains(t, content, `r.Post("/pets"`)

	// Check chi path parameter extraction
	assert.Contains(t, content, `chi.URLParam(req, "petId")`)
}

func TestGenerateServerStubs_OAS2(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petstore.json")
	err := os.WriteFile(tmpFile, []byte(testPetstoreOAS2Spec), 0600)
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
	require.NotNil(t, stubsFile, "server_stubs.go not generated for OAS 2.0")

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
}

func TestGenerateServerAll_OAS2(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "petstore.json")
	err := os.WriteFile(tmpFile, []byte(testPetstoreOAS2Spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("petapi"),
		WithServer(true),
		WithServerAll(), // Enable all server extensions
	)
	require.NoError(t, err)

	// Check all server extension files are generated
	assert.NotNil(t, result.GetFile("server.go"), "server.go not generated for OAS 2.0")
	assert.NotNil(t, result.GetFile("server_responses.go"), "server_responses.go not generated for OAS 2.0")
	assert.NotNil(t, result.GetFile("server_binder.go"), "server_binder.go not generated for OAS 2.0")
	assert.NotNil(t, result.GetFile("server_middleware.go"), "server_middleware.go not generated for OAS 2.0")
	assert.NotNil(t, result.GetFile("server_router.go"), "server_router.go not generated for OAS 2.0")
	assert.NotNil(t, result.GetFile("server_stubs.go"), "server_stubs.go not generated for OAS 2.0")

	// Verify we have at least 7 files (types + server + 5 extensions)
	var goFileCount int
	for _, f := range result.Files {
		if strings.HasSuffix(f.Name, ".go") {
			goFileCount++
		}
	}
	assert.GreaterOrEqual(t, goFileCount, 7, "Expected at least 7 .go files for OAS 2.0")
}

// Test spec with typed path parameters (integer, number, boolean)
const testTypedPathParamsSpec = `openapi: "3.0.3"
info:
  title: Typed Params API
  version: "1.0.0"
paths:
  /items/{itemId}:
    get:
      operationId: getItem
      parameters:
        - name: itemId
          in: path
          required: true
          schema:
            type: integer
            format: int64
      responses:
        '200':
          description: Item found
  /products/{price}:
    get:
      operationId: getProductByPrice
      parameters:
        - name: price
          in: path
          required: true
          schema:
            type: number
            format: float
      responses:
        '200':
          description: Product found
  /flags/{enabled}:
    get:
      operationId: getFlag
      parameters:
        - name: enabled
          in: path
          required: true
          schema:
            type: boolean
      responses:
        '200':
          description: Flag found
`

func TestServerRouterTypedPathParams(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "api.yaml")
	err := os.WriteFile(tmpFile, []byte(testTypedPathParamsSpec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("api"),
		WithServer(true),
		WithServerRouter("stdlib"),
	)
	require.NoError(t, err)

	routerFile := result.GetFile("server_router.go")
	require.NotNil(t, routerFile, "server_router.go not generated")
	content := string(routerFile.Content)

	// Check integer path parameter with error handling
	assert.Contains(t, content, `strconv.ParseInt(PathParam(req, "itemId"), 10, 64)`)
	assert.Contains(t, content, `WriteError(w, http.StatusBadRequest, "invalid path parameter: itemId")`)

	// Check number path parameter with error handling
	assert.Contains(t, content, `strconv.ParseFloat(PathParam(req, "price"), 64)`)
	assert.Contains(t, content, `WriteError(w, http.StatusBadRequest, "invalid path parameter: price")`)

	// Check boolean path parameter with error handling
	assert.Contains(t, content, `strconv.ParseBool(PathParam(req, "enabled"))`)
	assert.Contains(t, content, `WriteError(w, http.StatusBadRequest, "invalid path parameter: enabled")`)
}

func TestServerRouterTypedPathParams_Chi(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "api.yaml")
	err := os.WriteFile(tmpFile, []byte(testTypedPathParamsSpec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("api"),
		WithServer(true),
		WithServerRouter("chi"),
	)
	require.NoError(t, err)

	routerFile := result.GetFile("server_router.go")
	require.NotNil(t, routerFile, "server_router.go not generated")
	content := string(routerFile.Content)

	// Check chi-specific integer path parameter with error handling
	assert.Contains(t, content, `strconv.ParseInt(chi.URLParam(req, "itemId"), 10, 64)`)
	assert.Contains(t, content, `WriteError(w, http.StatusBadRequest, "invalid path parameter: itemId")`)

	// Check chi-specific number path parameter with error handling
	assert.Contains(t, content, `strconv.ParseFloat(chi.URLParam(req, "price"), 64)`)
	assert.Contains(t, content, `WriteError(w, http.StatusBadRequest, "invalid path parameter: price")`)

	// Check chi-specific boolean path parameter with error handling
	assert.Contains(t, content, `strconv.ParseBool(chi.URLParam(req, "enabled"))`)
	assert.Contains(t, content, `WriteError(w, http.StatusBadRequest, "invalid path parameter: enabled")`)
}

// OAS 2.0 spec with typed path parameters
const testTypedPathParamsOAS2Spec = `swagger: "2.0"
info:
  title: Typed Params API
  version: "1.0.0"
basePath: /v1
paths:
  /items/{itemId}:
    get:
      operationId: getItem
      parameters:
        - name: itemId
          in: path
          required: true
          type: integer
          format: int64
      responses:
        '200':
          description: Item found
  /products/{price}:
    get:
      operationId: getProductByPrice
      parameters:
        - name: price
          in: path
          required: true
          type: number
          format: float
      responses:
        '200':
          description: Product found
  /flags/{enabled}:
    get:
      operationId: getFlag
      parameters:
        - name: enabled
          in: path
          required: true
          type: boolean
      responses:
        '200':
          description: Flag found
`

func TestServerRouterTypedPathParams_OAS2(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "api.json")
	err := os.WriteFile(tmpFile, []byte(testTypedPathParamsOAS2Spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("api"),
		WithServer(true),
		WithServerRouter("stdlib"),
	)
	require.NoError(t, err)

	routerFile := result.GetFile("server_router.go")
	require.NotNil(t, routerFile, "server_router.go not generated for OAS 2.0")
	content := string(routerFile.Content)

	// Check integer path parameter with error handling
	assert.Contains(t, content, `strconv.ParseInt(PathParam(req, "itemId"), 10, 64)`)
	assert.Contains(t, content, `WriteError(w, http.StatusBadRequest, "invalid path parameter: itemId")`)

	// Check number path parameter with error handling
	assert.Contains(t, content, `strconv.ParseFloat(PathParam(req, "price"), 64)`)
	assert.Contains(t, content, `WriteError(w, http.StatusBadRequest, "invalid path parameter: price")`)

	// Check boolean path parameter with error handling
	assert.Contains(t, content, `strconv.ParseBool(PathParam(req, "enabled"))`)
	assert.Contains(t, content, `WriteError(w, http.StatusBadRequest, "invalid path parameter: enabled")`)
}

func TestServerRouterTypedPathParams_OAS2_Chi(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "api.json")
	err := os.WriteFile(tmpFile, []byte(testTypedPathParamsOAS2Spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("api"),
		WithServer(true),
		WithServerRouter("chi"),
	)
	require.NoError(t, err)

	routerFile := result.GetFile("server_router.go")
	require.NotNil(t, routerFile, "server_router.go not generated for OAS 2.0")
	content := string(routerFile.Content)

	// Check chi-specific integer path parameter with error handling
	assert.Contains(t, content, `strconv.ParseInt(chi.URLParam(req, "itemId"), 10, 64)`)
	assert.Contains(t, content, `WriteError(w, http.StatusBadRequest, "invalid path parameter: itemId")`)

	// Check chi-specific number path parameter with error handling
	assert.Contains(t, content, `strconv.ParseFloat(chi.URLParam(req, "price"), 64)`)
	assert.Contains(t, content, `WriteError(w, http.StatusBadRequest, "invalid path parameter: price")`)

	// Check chi-specific boolean path parameter with error handling
	assert.Contains(t, content, `strconv.ParseBool(chi.URLParam(req, "enabled"))`)
	assert.Contains(t, content, `WriteError(w, http.StatusBadRequest, "invalid path parameter: enabled")`)
}

// OAS 2.0 spec with header parameters
const testHeaderParamsOAS2Spec = `swagger: "2.0"
info:
  title: Header Params API
  version: "1.0.0"
basePath: /v1
paths:
  /resources:
    get:
      operationId: listResources
      parameters:
        - name: X-Request-ID
          in: header
          required: true
          type: string
        - name: X-Page-Size
          in: header
          required: false
          type: integer
      responses:
        '200':
          description: Resources found
`

func TestServerBinder_OAS2_HeaderParams(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "api.json")
	err := os.WriteFile(tmpFile, []byte(testHeaderParamsOAS2Spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("api"),
		WithServer(true),
		WithServerBinder(true),
	)
	require.NoError(t, err)

	binderFile := result.GetFile("server_binder.go")
	require.NotNil(t, binderFile, "server_binder.go not generated for OAS 2.0")
	content := string(binderFile.Content)

	// Check header parameter binding
	assert.Contains(t, content, `result.HeaderParams["X-Request-ID"]`)
	assert.Contains(t, content, `result.HeaderParams["X-Page-Size"]`)
}

// OAS 2.0 spec with wildcard responses (2XX, 4XX, 5XX)
const testWildcardResponsesOAS2Spec = `swagger: "2.0"
info:
  title: Wildcard Responses API
  version: "1.0.0"
basePath: /v1
paths:
  /resources:
    get:
      operationId: listResources
      responses:
        '200':
          description: Success
          schema:
            type: object
        '2XX':
          description: Other success
          schema:
            type: object
        '4XX':
          description: Client error
          schema:
            type: object
        '5XX':
          description: Server error
          schema:
            type: object
`

func TestServerResponses_OAS2_WildcardCodes(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "api.json")
	err := os.WriteFile(tmpFile, []byte(testWildcardResponsesOAS2Spec), 0600)
	require.NoError(t, err)

	result, err := GenerateWithOptions(
		WithFilePath(tmpFile),
		WithPackageName("api"),
		WithServer(true),
		WithServerResponses(true),
	)
	require.NoError(t, err)

	respFile := result.GetFile("server_responses.go")
	require.NotNil(t, respFile, "server_responses.go not generated for OAS 2.0")
	content := string(respFile.Content)

	// Check that wildcard response codes are handled
	// Note: The implementation may convert 2XX/4XX/5XX to StatusDefault or skip them
	// This test validates the generator doesn't crash on wildcard codes
	assert.Contains(t, content, "type ListResourcesResponse struct")
	assert.Contains(t, content, "func (ListResourcesResponse) Status200(")
}

func TestGetOAS2ParamSchemaType(t *testing.T) {
	// Create a minimal oas2CodeGenerator for testing
	cg := &oas2CodeGenerator{}

	tests := []struct {
		name     string
		param    *parser.Parameter
		expected string
	}{
		{
			name:     "nil parameter",
			param:    nil,
			expected: "string",
		},
		{
			name:     "direct Type field",
			param:    &parser.Parameter{Type: "integer"},
			expected: "integer",
		},
		{
			name:     "direct Type field - number",
			param:    &parser.Parameter{Type: "number"},
			expected: "number",
		},
		{
			name:     "direct Type field - boolean",
			param:    &parser.Parameter{Type: "boolean"},
			expected: "boolean",
		},
		{
			name:     "schema with string type",
			param:    &parser.Parameter{Schema: &parser.Schema{Type: "integer"}},
			expected: "integer",
		},
		{
			name:     "schema with []any type",
			param:    &parser.Parameter{Schema: &parser.Schema{Type: []any{"string", "null"}}},
			expected: "string",
		},
		{
			name:     "schema with []string type",
			param:    &parser.Parameter{Schema: &parser.Schema{Type: []string{"number", "null"}}},
			expected: "number",
		},
		{
			name:     "schema with empty []any type",
			param:    &parser.Parameter{Schema: &parser.Schema{Type: []any{}}},
			expected: "string",
		},
		{
			name:     "schema with empty []string type",
			param:    &parser.Parameter{Schema: &parser.Schema{Type: []string{}}},
			expected: "string",
		},
		{
			name:     "empty parameter",
			param:    &parser.Parameter{},
			expected: "string",
		},
		{
			name:     "schema with nil type",
			param:    &parser.Parameter{Schema: &parser.Schema{}},
			expected: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cg.getOAS2ParamSchemaType(tt.param)
			assert.Equal(t, tt.expected, result)
		})
	}
}
