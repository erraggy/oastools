package builder

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildOAS3 is a test helper that builds and asserts OAS3 document.
func buildOAS3(t *testing.T, b *Builder) *parser.OAS3Document {
	t.Helper()
	doc, err := b.BuildOAS3()
	require.NoError(t, err)
	return doc
}

func TestNew(t *testing.T) {
	b := New(parser.OASVersion320)

	assert.Equal(t, parser.OASVersion320, b.version)
	assert.NotNil(t, b.paths)
	assert.NotNil(t, b.schemas)
	assert.NotNil(t, b.schemaCache)
	assert.NotNil(t, b.operationIDs)
}

func TestNew_OAS20(t *testing.T) {
	// OAS 2.0 should be supported
	b := New(parser.OASVersion20)

	assert.Equal(t, parser.OASVersion20, b.version)
	assert.Empty(t, b.errors, "No errors should be recorded for OAS 2.0")
}

func TestBuild_OAS20Document(t *testing.T) {
	type User struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	b := New(parser.OASVersion20).
		SetTitle("Test API").
		SetVersion("1.0.0")

	b.AddOperation("GET", "/users",
		WithOperationID("listUsers"),
		WithResponse(200, []User{}),
	)

	// Use typed method - no type assertion needed
	doc, err := b.BuildOAS2()
	require.NoError(t, err)

	assert.Equal(t, "2.0", doc.Swagger)
	assert.Equal(t, "Test API", doc.Info.Title)
	assert.Equal(t, "1.0.0", doc.Info.Version)

	// Check that schemas are in definitions, not components
	assert.NotNil(t, doc.Definitions, "Schemas should be in definitions")
	assert.Contains(t, doc.Definitions, "builder.User")
}

func TestBuildOAS2_WrongVersion(t *testing.T) {
	// BuildOAS2() should error when builder is OAS 3.x
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0")

	_, err := b.BuildOAS2()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BuildOAS2()")
	assert.Contains(t, err.Error(), "use BuildOAS3()")
}

func TestBuildOAS3_WrongVersion(t *testing.T) {
	// BuildOAS3() should error when builder is OAS 2.0
	b := New(parser.OASVersion20).
		SetTitle("Test API").
		SetVersion("1.0.0")

	_, err := b.BuildOAS3()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BuildOAS3()")
	assert.Contains(t, err.Error(), "use BuildOAS2()")
}

func TestBuild_OAS3Document(t *testing.T) {
	type User struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0")

	b.AddOperation("GET", "/users",
		WithOperationID("listUsers"),
		WithResponse(200, []User{}),
	)

	// Use typed method - no type assertion needed
	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	assert.Equal(t, "3.2.0", doc.OpenAPI)
	assert.Equal(t, "Test API", doc.Info.Title)
	assert.Equal(t, "1.0.0", doc.Info.Version)

	// Check that schemas are in components, not definitions
	assert.NotNil(t, doc.Components, "Should have components")
	assert.NotNil(t, doc.Components.Schemas, "Schemas should be in components.schemas")
	assert.Contains(t, doc.Components.Schemas, "builder.User")
}

func TestSchemaRef_OAS20(t *testing.T) {
	b := New(parser.OASVersion20)
	ref := b.SchemaRef("User")
	assert.Equal(t, "#/definitions/User", ref)
}

func TestSchemaRef_OAS3(t *testing.T) {
	b := New(parser.OASVersion320)
	ref := b.SchemaRef("User")
	assert.Equal(t, "#/components/schemas/User", ref)
}

func TestParameterRef_OAS20(t *testing.T) {
	b := New(parser.OASVersion20)
	ref := b.ParameterRef("limitParam")
	assert.Equal(t, "#/parameters/limitParam", ref)
}

func TestParameterRef_OAS3(t *testing.T) {
	b := New(parser.OASVersion320)
	ref := b.ParameterRef("limitParam")
	assert.Equal(t, "#/components/parameters/limitParam", ref)
}

func TestResponseRef_OAS20(t *testing.T) {
	b := New(parser.OASVersion20)
	ref := b.ResponseRef("NotFound")
	assert.Equal(t, "#/responses/NotFound", ref)
}

func TestResponseRef_OAS3(t *testing.T) {
	b := New(parser.OASVersion320)
	ref := b.ResponseRef("NotFound")
	assert.Equal(t, "#/components/responses/NotFound", ref)
}

func TestNewWithInfo(t *testing.T) {
	info := &parser.Info{
		Title:   "Test API",
		Version: "1.0.0",
	}
	b := NewWithInfo(parser.OASVersion320, info)

	assert.Equal(t, parser.OASVersion320, b.version)
	assert.Equal(t, info, b.info)
}

func TestBuilder_SetInfo(t *testing.T) {
	b := New(parser.OASVersion320)
	info := &parser.Info{
		Title:   "Test API",
		Version: "1.0.0",
	}
	result := b.SetInfo(info)

	assert.Same(t, b, result) // Fluent API
	assert.Equal(t, info, b.info)
}

func TestBuilder_SetTitle(t *testing.T) {
	b := New(parser.OASVersion320)
	result := b.SetTitle("My API")

	assert.Same(t, b, result)
	assert.Equal(t, "My API", b.info.Title)
}

func TestBuilder_SetVersion(t *testing.T) {
	b := New(parser.OASVersion320)
	result := b.SetVersion("2.0.0")

	assert.Same(t, b, result)
	assert.Equal(t, "2.0.0", b.info.Version)
}

func TestBuilder_SetDescription(t *testing.T) {
	b := New(parser.OASVersion320)
	result := b.SetDescription("API description")

	assert.Same(t, b, result)
	assert.Equal(t, "API description", b.info.Description)
}

func TestBuilder_SetTermsOfService(t *testing.T) {
	b := New(parser.OASVersion320)
	result := b.SetTermsOfService("https://example.com/tos")

	assert.Same(t, b, result)
	assert.Equal(t, "https://example.com/tos", b.info.TermsOfService)
}

func TestBuilder_SetContact(t *testing.T) {
	b := New(parser.OASVersion320)
	contact := &parser.Contact{
		Name:  "API Support",
		Email: "support@example.com",
	}
	result := b.SetContact(contact)

	assert.Same(t, b, result)
	assert.Equal(t, contact, b.info.Contact)
}

func TestBuilder_SetLicense(t *testing.T) {
	b := New(parser.OASVersion320)
	license := &parser.License{
		Name: "MIT",
		URL:  "https://opensource.org/licenses/MIT",
	}
	result := b.SetLicense(license)

	assert.Same(t, b, result)
	assert.Equal(t, license, b.info.License)
}

func TestBuilder_BuildOAS3(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0")

		doc, err := b.BuildOAS3()
		require.NoError(t, err)
		assert.Equal(t, "3.2.0", doc.OpenAPI)
		assert.Equal(t, "Test API", doc.Info.Title)
		assert.Equal(t, "1.0.0", doc.Info.Version)
	})

	t.Run("without info - no validation", func(t *testing.T) {
		// Builder does not perform OAS validation, that's the job of validator package
		b := New(parser.OASVersion320)
		doc, err := b.BuildOAS3()
		// No error because builder doesn't validate
		require.NoError(t, err)
		assert.Nil(t, doc.Info)
	})

	t.Run("with accumulated errors", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0")
		b.errors = append(b.errors, assert.AnError)
		b.errors = append(b.errors, fmt.Errorf("another error"))

		_, err := b.BuildOAS3()
		assert.Error(t, err)
		// Verify all errors are included in the message
		assert.Contains(t, err.Error(), "2 error(s)")
		assert.Contains(t, err.Error(), "assert.AnError")
		assert.Contains(t, err.Error(), "another error")
	})
}

func TestBuilder_BuildResult(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0")

	result, err := b.BuildResult()
	require.NoError(t, err)
	assert.Equal(t, "builder", result.SourcePath)
	assert.Equal(t, parser.SourceFormatYAML, result.SourceFormat)
	assert.Equal(t, "3.2.0", result.Version)
	assert.Equal(t, parser.OASVersion320, result.OASVersion)
	assert.NotNil(t, result.Document)
}

func TestBuilder_MarshalYAML(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0")

	data, err := b.MarshalYAML()
	require.NoError(t, err)
	assert.Contains(t, string(data), "openapi: 3.2.0")
	assert.Contains(t, string(data), "title: Test API")
}

func TestBuilder_MarshalYAML_Error(t *testing.T) {
	b := New(parser.OASVersion320)
	// Add accumulated error to cause build to fail
	b.errors = append(b.errors, fmt.Errorf("test error"))
	_, err := b.MarshalYAML()
	assert.Error(t, err)
}

func TestBuilder_MarshalJSON(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0")

	data, err := b.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(data), `"openapi": "3.2.0"`)
	assert.Contains(t, string(data), `"title": "Test API"`)
}

func TestBuilder_MarshalJSON_Error(t *testing.T) {
	b := New(parser.OASVersion320)
	// Add accumulated error to cause build to fail
	b.errors = append(b.errors, fmt.Errorf("test error"))
	_, err := b.MarshalJSON()
	assert.Error(t, err)
}

func TestBuilder_BuildResult_Error(t *testing.T) {
	b := New(parser.OASVersion320)
	// Add accumulated error to cause build to fail
	b.errors = append(b.errors, fmt.Errorf("test error"))
	_, err := b.BuildResult()
	assert.Error(t, err)
}

func TestBuilder_WriteFile(t *testing.T) {
	t.Run("yaml", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0")

		path := t.TempDir() + "/test.yaml"
		err := b.WriteFile(path)
		require.NoError(t, err)
	})

	t.Run("json", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0")

		path := t.TempDir() + "/test.json"
		err := b.WriteFile(path)
		require.NoError(t, err)
	})

	t.Run("default extension", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0")

		path := t.TempDir() + "/test"
		err := b.WriteFile(path)
		require.NoError(t, err)
	})

	t.Run("yml extension", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0")

		path := t.TempDir() + "/test.yml"
		err := b.WriteFile(path)
		require.NoError(t, err)
	})

	t.Run("error on build", func(t *testing.T) {
		b := New(parser.OASVersion320)
		// Add accumulated error to cause build to fail
		b.errors = append(b.errors, fmt.Errorf("test error"))

		path := t.TempDir() + "/test.yaml"
		err := b.WriteFile(path)
		assert.Error(t, err)
	})
}

func TestBuilder_AddServer(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddServer("https://api.example.com",
			WithServerDescription("Production server"),
		)

	require.Len(t, b.servers, 1)
	assert.Equal(t, "https://api.example.com", b.servers[0].URL)
	assert.Equal(t, "Production server", b.servers[0].Description)
}

func TestBuilder_AddServerWithVariables(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddServer("https://{environment}.api.example.com",
			WithServerDescription("Server with variables"),
			WithServerVariable("environment", "prod",
				WithServerVariableEnum("dev", "staging", "prod"),
				WithServerVariableDescription("Environment"),
			),
		)

	require.Len(t, b.servers, 1)
	assert.Contains(t, b.servers[0].Variables, "environment")
	assert.Equal(t, "prod", b.servers[0].Variables["environment"].Default)
	assert.Equal(t, []string{"dev", "staging", "prod"}, b.servers[0].Variables["environment"].Enum)
}

func TestBuilder_AddTag(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddTag("users",
			WithTagDescription("User operations"),
			WithTagExternalDocs("https://docs.example.com/users", "User docs"),
		)

	require.Len(t, b.tags, 1)
	assert.Equal(t, "users", b.tags[0].Name)
	assert.Equal(t, "User operations", b.tags[0].Description)
	require.NotNil(t, b.tags[0].ExternalDocs)
	assert.Equal(t, "https://docs.example.com/users", b.tags[0].ExternalDocs.URL)
}

func TestBuilder_AddOperation(t *testing.T) {
	type User struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/users",
			WithOperationID("listUsers"),
			WithSummary("List users"),
			WithDescription("Get all users"),
			WithTags("users"),
			WithResponse(http.StatusOK, []User{}),
		)

	require.Contains(t, b.paths, "/users")
	require.NotNil(t, b.paths["/users"].Get)
	assert.Equal(t, "listUsers", b.paths["/users"].Get.OperationID)
	assert.Equal(t, "List users", b.paths["/users"].Get.Summary)
	assert.Equal(t, []string{"users"}, b.paths["/users"].Get.Tags)
}

func TestBuilder_AddOperation_AllMethods(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0")

	methods := []string{
		http.MethodGet,
		http.MethodPut,
		http.MethodPost,
		http.MethodDelete,
		http.MethodOptions,
		http.MethodHead,
		http.MethodPatch,
		http.MethodTrace,
	}

	for _, method := range methods {
		b.AddOperation(method, "/test", WithOperationID(method+"Operation"))
	}

	pathItem := b.paths["/test"]
	assert.NotNil(t, pathItem.Get)
	assert.NotNil(t, pathItem.Put)
	assert.NotNil(t, pathItem.Post)
	assert.NotNil(t, pathItem.Delete)
	assert.NotNil(t, pathItem.Options)
	assert.NotNil(t, pathItem.Head)
	assert.NotNil(t, pathItem.Patch)
	assert.NotNil(t, pathItem.Trace)
}

func TestBuilder_AddOperation_DuplicateOperationID(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/users", WithOperationID("getUsers")).
		AddOperation(http.MethodGet, "/posts", WithOperationID("getUsers"))

	assert.Len(t, b.errors, 1)
	assert.Contains(t, b.errors[0].Error(), "duplicate operation ID")
}

func TestBuilder_AddOperation_WithRequestBody(t *testing.T) {
	type CreateUser struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/users",
			WithOperationID("createUser"),
			WithRequestBody("application/json", CreateUser{},
				WithRequired(true),
				WithRequestDescription("User to create"),
			),
			WithResponse(http.StatusCreated, CreateUser{}),
		)

	require.NotNil(t, b.paths["/users"].Post.RequestBody)
	assert.True(t, b.paths["/users"].Post.RequestBody.Required)
	assert.Equal(t, "User to create", b.paths["/users"].Post.RequestBody.Description)
}

func TestBuilder_AddOperation_WithParameters(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/users/{userId}",
			WithOperationID("getUser"),
			WithPathParam("userId", int64(0), WithParamDescription("User ID")),
			WithQueryParam("include", string(""), WithParamDescription("Include related")),
			WithHeaderParam("X-Request-ID", string(""), WithParamRequired(true)),
			WithResponse(http.StatusOK, struct{}{}),
		)

	params := b.paths["/users/{userId}"].Get.Parameters
	require.Len(t, params, 3)

	// Path param
	assert.Equal(t, "userId", params[0].Name)
	assert.Equal(t, "path", params[0].In)
	assert.True(t, params[0].Required)

	// Query param
	assert.Equal(t, "include", params[1].Name)
	assert.Equal(t, "query", params[1].In)

	// Header param
	assert.Equal(t, "X-Request-ID", params[2].Name)
	assert.Equal(t, "header", params[2].In)
	assert.True(t, params[2].Required)
}

func TestBuilder_SecuritySchemes(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddAPIKeySecurityScheme("api_key", "header", "X-API-Key", "API key auth").
		AddHTTPSecurityScheme("bearer_auth", "bearer", "JWT", "Bearer auth").
		SetSecurity(
			SecurityRequirement("api_key"),
			SecurityRequirement("bearer_auth"),
		)

	doc := buildOAS3(t, b)

	require.NotNil(t, doc.Components)
	require.NotNil(t, doc.Components.SecuritySchemes)
	assert.Contains(t, doc.Components.SecuritySchemes, "api_key")
	assert.Contains(t, doc.Components.SecuritySchemes, "bearer_auth")
	assert.Len(t, doc.Security, 2)
}

func TestBuilder_AddSecurityScheme(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddSecurityScheme("custom", &parser.SecurityScheme{
			Type:        "apiKey",
			Name:        "custom-key",
			In:          "header",
			Description: "Custom scheme",
		})

	doc := buildOAS3(t, b)

	require.NotNil(t, doc.Components.SecuritySchemes)
	assert.Contains(t, doc.Components.SecuritySchemes, "custom")
	assert.Equal(t, "apiKey", doc.Components.SecuritySchemes["custom"].Type)
}

func TestBuilder_AddOAuth2SecurityScheme(t *testing.T) {
	flows := &parser.OAuthFlows{
		AuthorizationCode: &parser.OAuthFlow{
			AuthorizationURL: "https://example.com/oauth/authorize",
			TokenURL:         "https://example.com/oauth/token",
			Scopes: map[string]string{
				"read":  "Read access",
				"write": "Write access",
			},
		},
	}

	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddOAuth2SecurityScheme("oauth2", flows, "OAuth2 authentication")

	doc := buildOAS3(t, b)

	require.NotNil(t, doc.Components.SecuritySchemes)
	assert.Contains(t, doc.Components.SecuritySchemes, "oauth2")
	scheme := doc.Components.SecuritySchemes["oauth2"]
	assert.Equal(t, "oauth2", scheme.Type)
	assert.Equal(t, "OAuth2 authentication", scheme.Description)
	require.NotNil(t, scheme.Flows)
	require.NotNil(t, scheme.Flows.AuthorizationCode)
	assert.Equal(t, "https://example.com/oauth/authorize", scheme.Flows.AuthorizationCode.AuthorizationURL)
}

func TestBuilder_AddOpenIDConnectSecurityScheme(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddOpenIDConnectSecurityScheme("oidc", "https://example.com/.well-known/openid-configuration", "OpenID Connect")

	doc := buildOAS3(t, b)

	require.NotNil(t, doc.Components.SecuritySchemes)
	assert.Contains(t, doc.Components.SecuritySchemes, "oidc")
	scheme := doc.Components.SecuritySchemes["oidc"]
	assert.Equal(t, "openIdConnect", scheme.Type)
	assert.Equal(t, "https://example.com/.well-known/openid-configuration", scheme.OpenIDConnectURL)
	assert.Equal(t, "OpenID Connect", scheme.Description)
}

func TestBuilder_AddResponse(t *testing.T) {
	type ErrorResponse struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}

	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddResponse("Error", "Error response", ErrorResponse{},
			WithResponseExample(map[string]any{"code": 500, "message": "Internal error"}),
		)

	doc := buildOAS3(t, b)

	require.NotNil(t, doc.Components.Responses)
	assert.Contains(t, doc.Components.Responses, "Error")
	resp := doc.Components.Responses["Error"]
	assert.Equal(t, "Error response", resp.Description)
	require.Contains(t, resp.Content, "application/json")
	require.NotNil(t, resp.Content["application/json"].Schema)
}

func TestBuilder_AddResponse_WithContentType(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddResponse("XMLError", "XML Error response", struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{}, WithResponseContentType("application/xml"))

	doc := buildOAS3(t, b)

	require.NotNil(t, doc.Components.Responses)
	assert.Contains(t, doc.Components.Responses, "XMLError")
	resp := doc.Components.Responses["XMLError"]
	assert.Equal(t, "XML Error response", resp.Description)
	require.Contains(t, resp.Content, "application/xml")
}

func TestResponseRef(t *testing.T) {
	// Test OAS 3.x default
	b := New(parser.OASVersion320)
	ref := b.ResponseRef("Error")
	assert.Equal(t, "#/components/responses/Error", ref)
}

func TestBuilder_AddParameter(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddParameter("PageLimit", "query", "limit", int32(0),
			WithParamDescription("Maximum number of items to return"),
			WithParamRequired(false),
		)

	doc := buildOAS3(t, b)

	require.NotNil(t, doc.Components.Parameters)
	assert.Contains(t, doc.Components.Parameters, "PageLimit")
	param := doc.Components.Parameters["PageLimit"]
	assert.Equal(t, "limit", param.Name)
	assert.Equal(t, "query", param.In)
	assert.Equal(t, "Maximum number of items to return", param.Description)
	assert.False(t, param.Required)
}

func TestBuilder_AddParameter_PathAlwaysRequired(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddParameter("UserId", "path", "userId", int64(0),
			WithParamRequired(false), // This should be ignored for path params
		)

	doc := buildOAS3(t, b)

	require.NotNil(t, doc.Components.Parameters)
	param := doc.Components.Parameters["UserId"]
	assert.True(t, param.Required) // Path parameters are always required
}

func TestParameterRef(t *testing.T) {
	// Test OAS 3.x default
	b := New(parser.OASVersion320)
	ref := b.ParameterRef("PageLimit")
	assert.Equal(t, "#/components/parameters/PageLimit", ref)
}

func TestBuilder_CompleteExample(t *testing.T) {
	type Pet struct {
		ID   int64  `json:"id" oas:"description=Unique pet identifier"`
		Name string `json:"name" oas:"minLength=1,description=Pet name"`
		Tag  string `json:"tag,omitempty" oas:"description=Optional tag"`
	}

	type Error struct {
		Code    int32  `json:"code"`
		Message string `json:"message"`
	}

	b := New(parser.OASVersion320).
		SetTitle("Pet Store API").
		SetVersion("1.0.0").
		SetDescription("A sample Pet Store API").
		AddServer("https://api.petstore.example.com/v1",
			WithServerDescription("Production server"),
		).
		AddTag("pets", WithTagDescription("Pet operations")).
		AddOperation(http.MethodGet, "/pets",
			WithOperationID("listPets"),
			WithSummary("List all pets"),
			WithTags("pets"),
			WithQueryParam("limit", int32(0), WithParamDescription("Max number to return")),
			WithResponse(http.StatusOK, []Pet{}, WithResponseDescription("A list of pets")),
			WithResponse(http.StatusInternalServerError, Error{}, WithResponseDescription("Error")),
		).
		AddOperation(http.MethodGet, "/pets/{petId}",
			WithOperationID("getPet"),
			WithSummary("Get a pet by ID"),
			WithTags("pets"),
			WithPathParam("petId", int64(0), WithParamDescription("Pet ID")),
			WithResponse(http.StatusOK, Pet{}),
			WithResponse(http.StatusNotFound, Error{}),
		)

	doc := buildOAS3(t, b)

	// Verify document structure
	assert.Equal(t, "3.2.0", doc.OpenAPI)
	assert.Equal(t, "Pet Store API", doc.Info.Title)
	require.Len(t, doc.Servers, 1)
	require.Len(t, doc.Tags, 1)
	assert.Contains(t, doc.Paths, "/pets")
	assert.Contains(t, doc.Paths, "/pets/{petId}")

	// Verify schemas were generated
	require.NotNil(t, doc.Components)
	require.NotNil(t, doc.Components.Schemas)
	assert.Contains(t, doc.Components.Schemas, "builder.Pet")
	assert.Contains(t, doc.Components.Schemas, "builder.Error")
}

func TestFromDocument(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		// Create an existing document
		original := &parser.OAS3Document{
			OpenAPI: "3.1.0",
			Info: &parser.Info{
				Title:   "Original API",
				Version: "1.0.0",
			},
			Paths: parser.Paths{
				"/existing": &parser.PathItem{
					Get: &parser.Operation{
						OperationID: "existingOp",
					},
				},
			},
			Components: &parser.Components{
				Schemas: map[string]*parser.Schema{
					"ExistingSchema": {Type: "string"},
				},
			},
		}

		// Create builder from document
		b := FromDocument(original)

		// Add new operation
		b.AddOperation(http.MethodPost, "/new",
			WithOperationID("newOp"),
			WithResponse(http.StatusOK, struct{}{}),
		)

		doc := buildOAS3(t, b)

		// Verify original content preserved
		assert.Contains(t, doc.Paths, "/existing")
		assert.Contains(t, doc.Components.Schemas, "ExistingSchema")

		// Verify new content added
		assert.Contains(t, doc.Paths, "/new")
	})

	t.Run("with responses and parameters", func(t *testing.T) {
		original := &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info: &parser.Info{
				Title:   "Original API",
				Version: "1.0.0",
			},
			Components: &parser.Components{
				Responses: map[string]*parser.Response{
					"NotFound": {Description: "Not found"},
				},
				Parameters: map[string]*parser.Parameter{
					"PageSize": {Name: "pageSize", In: "query"},
				},
			},
		}

		b := FromDocument(original)
		doc := buildOAS3(t, b)

		assert.Contains(t, doc.Components.Responses, "NotFound")
		assert.Contains(t, doc.Components.Parameters, "PageSize")
	})

	t.Run("with security schemes", func(t *testing.T) {
		original := &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info: &parser.Info{
				Title:   "Original API",
				Version: "1.0.0",
			},
			Components: &parser.Components{
				SecuritySchemes: map[string]*parser.SecurityScheme{
					"api_key": {Type: "apiKey", Name: "X-API-Key", In: "header"},
				},
			},
			Security: []parser.SecurityRequirement{
				{"api_key": []string{}},
			},
		}

		b := FromDocument(original)
		doc := buildOAS3(t, b)

		assert.Contains(t, doc.Components.SecuritySchemes, "api_key")
		assert.Len(t, doc.Security, 1)
	})

	t.Run("with servers and tags", func(t *testing.T) {
		original := &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info: &parser.Info{
				Title:   "Original API",
				Version: "1.0.0",
			},
			Servers: []*parser.Server{
				{URL: "https://api.example.com"},
			},
			Tags: []*parser.Tag{
				{Name: "users", Description: "User operations"},
			},
		}

		b := FromDocument(original)
		doc := buildOAS3(t, b)

		require.Len(t, doc.Servers, 1)
		assert.Equal(t, "https://api.example.com", doc.Servers[0].URL)
		require.Len(t, doc.Tags, 1)
		assert.Equal(t, "users", doc.Tags[0].Name)
	})

	t.Run("with invalid version defaults to 3.2.0", func(t *testing.T) {
		original := &parser.OAS3Document{
			OpenAPI: "invalid",
			Info: &parser.Info{
				Title:   "Original API",
				Version: "1.0.0",
			},
		}

		b := FromDocument(original)
		doc := buildOAS3(t, b)

		assert.Equal(t, "3.2.0", doc.OpenAPI)
	})

	t.Run("with nil components", func(t *testing.T) {
		original := &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info: &parser.Info{
				Title:   "Original API",
				Version: "1.0.0",
			},
			Components: nil,
		}

		b := FromDocument(original)
		doc := buildOAS3(t, b)
		assert.NotNil(t, doc)
	})

	t.Run("with external docs", func(t *testing.T) {
		original := &parser.OAS3Document{
			OpenAPI: "3.0.3",
			Info: &parser.Info{
				Title:   "Original API",
				Version: "1.0.0",
			},
			ExternalDocs: &parser.ExternalDocs{
				URL:         "https://docs.example.com",
				Description: "API documentation",
			},
		}

		b := FromDocument(original)
		doc := buildOAS3(t, b)

		require.NotNil(t, doc.ExternalDocs)
		assert.Equal(t, "https://docs.example.com", doc.ExternalDocs.URL)
		assert.Equal(t, "API documentation", doc.ExternalDocs.Description)
	})
}

func TestFromOAS2Document(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		// Create an existing OAS 2.0 document
		original := &parser.OAS2Document{
			Swagger: "2.0",
			Info: &parser.Info{
				Title:   "Original API",
				Version: "1.0.0",
			},
			Paths: parser.Paths{
				"/existing": &parser.PathItem{
					Get: &parser.Operation{
						OperationID: "existingOp",
					},
				},
			},
			Definitions: map[string]*parser.Schema{
				"ExistingSchema": {Type: "string"},
			},
		}

		// Create builder from document
		b := FromOAS2Document(original)

		// Add new operation
		b.AddOperation(http.MethodPost, "/new",
			WithOperationID("newOp"),
			WithResponse(http.StatusOK, struct{}{}),
		)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		// Verify original content preserved
		assert.Contains(t, doc.Paths, "/existing")
		assert.Contains(t, doc.Definitions, "ExistingSchema")

		// Verify new content added
		assert.Contains(t, doc.Paths, "/new")
	})

	t.Run("with parameters and responses", func(t *testing.T) {
		original := &parser.OAS2Document{
			Swagger: "2.0",
			Info: &parser.Info{
				Title:   "Original API",
				Version: "1.0.0",
			},
			Parameters: map[string]*parser.Parameter{
				"PageSize": {Name: "pageSize", In: "query"},
			},
			Responses: map[string]*parser.Response{
				"NotFound": {Description: "Not found"},
			},
		}

		b := FromOAS2Document(original)
		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		assert.Contains(t, doc.Parameters, "PageSize")
		assert.Contains(t, doc.Responses, "NotFound")
	})

	t.Run("with security definitions", func(t *testing.T) {
		original := &parser.OAS2Document{
			Swagger: "2.0",
			Info: &parser.Info{
				Title:   "Original API",
				Version: "1.0.0",
			},
			SecurityDefinitions: map[string]*parser.SecurityScheme{
				"api_key": {Type: "apiKey", Name: "X-API-Key", In: "header"},
			},
			Security: []parser.SecurityRequirement{
				{"api_key": []string{}},
			},
		}

		b := FromOAS2Document(original)
		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		assert.Contains(t, doc.SecurityDefinitions, "api_key")
		assert.Len(t, doc.Security, 1)
	})

	t.Run("with tags and external docs", func(t *testing.T) {
		original := &parser.OAS2Document{
			Swagger: "2.0",
			Info: &parser.Info{
				Title:   "Original API",
				Version: "1.0.0",
			},
			Tags: []*parser.Tag{
				{Name: "users", Description: "User operations"},
			},
			ExternalDocs: &parser.ExternalDocs{
				URL:         "https://docs.example.com",
				Description: "API documentation",
			},
		}

		b := FromOAS2Document(original)
		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		require.Len(t, doc.Tags, 1)
		assert.Equal(t, "users", doc.Tags[0].Name)
		require.NotNil(t, doc.ExternalDocs)
		assert.Equal(t, "https://docs.example.com", doc.ExternalDocs.URL)
	})
}

func TestBuilder_SetExternalDocs(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		SetExternalDocs(&parser.ExternalDocs{
			URL:         "https://docs.example.com",
			Description: "API documentation",
		})

	doc := buildOAS3(t, b)

	require.NotNil(t, doc.ExternalDocs)
	assert.Equal(t, "https://docs.example.com", doc.ExternalDocs.URL)
	assert.Equal(t, "API documentation", doc.ExternalDocs.Description)
}

func TestBuilder_AddWebhook(t *testing.T) {
	type WebhookPayload struct {
		Event string `json:"event"`
		Data  any    `json:"data"`
	}

	t.Run("basic webhook", func(t *testing.T) {
		b := New(parser.OASVersion310).
			SetTitle("Webhook API").
			SetVersion("1.0.0").
			AddWebhook("newUser", http.MethodPost,
				WithOperationID("userCreatedWebhook"),
				WithRequestBody("application/json", WebhookPayload{}),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc := buildOAS3(t, b)

		require.NotNil(t, doc.Webhooks)
		require.Contains(t, doc.Webhooks, "newUser")
		assert.NotNil(t, doc.Webhooks["newUser"].Post)
		assert.Equal(t, "userCreatedWebhook", doc.Webhooks["newUser"].Post.OperationID)
	})

	t.Run("multiple methods on same webhook", func(t *testing.T) {
		b := New(parser.OASVersion310).
			SetTitle("Webhook API").
			SetVersion("1.0.0").
			AddWebhook("events", http.MethodPost,
				WithOperationID("createEvent"),
				WithResponse(http.StatusOK, struct{}{}),
			).
			AddWebhook("events", http.MethodGet,
				WithOperationID("listEvents"),
				WithResponse(http.StatusOK, []struct{}{}),
			)

		doc := buildOAS3(t, b)

		require.NotNil(t, doc.Webhooks)
		require.Contains(t, doc.Webhooks, "events")
		assert.NotNil(t, doc.Webhooks["events"].Post)
		assert.NotNil(t, doc.Webhooks["events"].Get)
	})

	t.Run("with security", func(t *testing.T) {
		b := New(parser.OASVersion310).
			SetTitle("Webhook API").
			SetVersion("1.0.0").
			AddAPIKeySecurityScheme("api_key", "header", "X-API-Key", "API Key").
			AddWebhook("secure", http.MethodPost,
				WithOperationID("secureWebhook"),
				WithResponse(http.StatusOK, struct{}{}),
				WithSecurity(parser.SecurityRequirement{"api_key": []string{}}),
			)

		doc := buildOAS3(t, b)

		require.NotNil(t, doc.Webhooks)
		require.Contains(t, doc.Webhooks, "secure")
		require.NotNil(t, doc.Webhooks["secure"].Post)
		require.Len(t, doc.Webhooks["secure"].Post.Security, 1)
		assert.Contains(t, doc.Webhooks["secure"].Post.Security[0], "api_key")
	})
}

func TestBuilder_TimeType(t *testing.T) {
	type Event struct {
		Name      string    `json:"name"`
		Timestamp time.Time `json:"timestamp"`
	}

	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0")

	schema := b.generateSchema(Event{})

	// The schema should be a reference
	assert.Contains(t, schema.Ref, "builder.Event")

	// Check the actual schema
	require.Contains(t, b.schemas, "builder.Event")
	eventSchema := b.schemas["builder.Event"]
	require.Contains(t, eventSchema.Properties, "timestamp")

	// The timestamp property should point to a time type schema with date-time format
	// Since time.Time is not a named struct, it generates directly
	timestampProp := eventSchema.Properties["timestamp"]
	assert.Equal(t, "string", timestampProp.Type)
	assert.Equal(t, "date-time", timestampProp.Format)
}

func TestBuilder_FileUpload_OAS20_Integration(t *testing.T) {
	b := New(parser.OASVersion20).
		SetTitle("File Upload API").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/upload",
			WithOperationID("uploadFile"),
			WithFileParam("file", WithParamRequired(true), WithParamDescription("File to upload")),
			WithFormParam("description", "", WithParamDescription("File description")),
			WithResponse(http.StatusOK, struct {
				Success bool   `json:"success"`
				FileID  string `json:"file_id"`
			}{}),
		)

	doc, err := b.BuildOAS2()
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Verify operation exists
	require.Contains(t, doc.Paths, "/upload")
	require.NotNil(t, doc.Paths["/upload"].Post)

	// Verify file parameter
	params := doc.Paths["/upload"].Post.Parameters
	require.Len(t, params, 2)

	// Find file parameter
	var fileParam, descParam *parser.Parameter
	for _, p := range params {
		switch p.Name {
		case "file":
			fileParam = p
		case "description":
			descParam = p
		}
	}

	require.NotNil(t, fileParam, "file parameter should exist")
	assert.Equal(t, parser.ParamInFormData, fileParam.In)
	assert.Equal(t, "file", fileParam.Type)
	assert.True(t, fileParam.Required)

	require.NotNil(t, descParam, "description parameter should exist")
	assert.Equal(t, parser.ParamInFormData, descParam.In)
}

func TestBuilder_FileUpload_OAS3_Integration(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("File Upload API").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/upload",
			WithOperationID("uploadFile"),
			WithFileParam("file", WithParamRequired(true), WithParamDescription("File to upload")),
			WithFormParam("description", "", WithParamDescription("File description")),
			WithResponse(http.StatusOK, struct {
				Success bool   `json:"success"`
				FileID  string `json:"file_id"`
			}{}),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Verify operation exists
	require.Contains(t, doc.Paths, "/upload")
	require.NotNil(t, doc.Paths["/upload"].Post)

	// Verify request body exists (OAS 3.x uses request body for form data)
	rb := doc.Paths["/upload"].Post.RequestBody
	require.NotNil(t, rb)
	require.Contains(t, rb.Content, "multipart/form-data")

	// Verify schema structure
	schema := rb.Content["multipart/form-data"].Schema
	require.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)

	// Verify file property
	require.Contains(t, schema.Properties, "file")
	fileSchema := schema.Properties["file"]
	assert.Equal(t, "string", fileSchema.Type)
	assert.Equal(t, "binary", fileSchema.Format)
	assert.Equal(t, "File to upload", fileSchema.Description)

	// Verify description property
	require.Contains(t, schema.Properties, "description")
	descSchema := schema.Properties["description"]
	assert.Equal(t, "string", descSchema.Type)
	assert.Equal(t, "File description", descSchema.Description)

	// Verify required fields
	require.Contains(t, schema.Required, "file")
}

func TestBuilder_RawSchema_BinaryDownload_Integration(t *testing.T) {
	schema := &parser.Schema{
		Type:   "string",
		Format: "binary",
	}

	b := New(parser.OASVersion320).
		SetTitle("Download API").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/download/{id}",
			WithOperationID("downloadFile"),
			WithPathParam("id", int64(0), WithParamDescription("File ID")),
			WithResponseRawSchema(http.StatusOK, "application/octet-stream", schema,
				WithResponseDescription("Binary file content"),
				WithResponseHeader("Content-Disposition", &parser.Header{
					Description: "Suggested filename",
					Schema:      &parser.Schema{Type: "string"},
				}),
			),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Verify operation exists
	require.Contains(t, doc.Paths, "/download/{id}")
	require.NotNil(t, doc.Paths["/download/{id}"].Get)

	// Verify response
	resp := doc.Paths["/download/{id}"].Get.Responses.Codes["200"]
	require.NotNil(t, resp)
	assert.Equal(t, "Binary file content", resp.Description)

	// Verify content type
	require.Contains(t, resp.Content, "application/octet-stream")
	mediaType := resp.Content["application/octet-stream"]
	require.NotNil(t, mediaType.Schema)
	assert.Equal(t, "string", mediaType.Schema.Type)
	assert.Equal(t, "binary", mediaType.Schema.Format)

	// Verify header
	require.Contains(t, resp.Headers, "Content-Disposition")
	assert.Equal(t, "Suggested filename", resp.Headers["Content-Disposition"].Description)
}

func TestBuilder_RawSchema_ComplexUpload_Integration(t *testing.T) {
	// Complex schema for multipart upload with file and metadata
	schema := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"file": {
				Type:        "string",
				Format:      "binary",
				Description: "The file data",
			},
			"metadata": {
				Type: "object",
				Properties: map[string]*parser.Schema{
					"filename": {Type: "string"},
					"tags":     {Type: "array", Items: &parser.Schema{Type: "string"}},
				},
			},
		},
		Required: []string{"file"},
	}

	b := New(parser.OASVersion320).
		SetTitle("Complex Upload API").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/upload-with-metadata",
			WithOperationID("uploadWithMetadata"),
			WithRequestBodyRawSchema("multipart/form-data", schema,
				WithRequired(true),
				WithRequestDescription("Upload file with metadata"),
			),
			WithResponse(http.StatusCreated, struct {
				ID string `json:"id"`
			}{}),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Verify operation exists
	require.Contains(t, doc.Paths, "/upload-with-metadata")
	require.NotNil(t, doc.Paths["/upload-with-metadata"].Post)

	// Verify request body
	rb := doc.Paths["/upload-with-metadata"].Post.RequestBody
	require.NotNil(t, rb)
	assert.True(t, rb.Required)
	assert.Equal(t, "Upload file with metadata", rb.Description)

	// Verify schema structure
	require.Contains(t, rb.Content, "multipart/form-data")
	reqSchema := rb.Content["multipart/form-data"].Schema
	require.NotNil(t, reqSchema)
	assert.Equal(t, "object", reqSchema.Type)

	// Verify file property
	require.Contains(t, reqSchema.Properties, "file")
	fileSchema := reqSchema.Properties["file"]
	assert.Equal(t, "string", fileSchema.Type)
	assert.Equal(t, "binary", fileSchema.Format)

	// Verify metadata property
	require.Contains(t, reqSchema.Properties, "metadata")
	metadataSchema := reqSchema.Properties["metadata"]
	assert.Equal(t, "object", metadataSchema.Type)
	require.Contains(t, metadataSchema.Properties, "filename")
	require.Contains(t, metadataSchema.Properties, "tags")

	// Verify required fields
	require.Contains(t, reqSchema.Required, "file")
}
