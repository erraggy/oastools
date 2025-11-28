package builder

import (
	"net/http"
	"testing"
	"time"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	b := New(parser.OASVersion320)

	assert.Equal(t, parser.OASVersion320, b.version)
	assert.NotNil(t, b.paths)
	assert.NotNil(t, b.schemas)
	assert.NotNil(t, b.schemaCache)
	assert.NotNil(t, b.operationIDs)
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

func TestBuilder_Build(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0")

		doc, err := b.Build()
		require.NoError(t, err)
		assert.Equal(t, "3.2.0", doc.OpenAPI)
		assert.Equal(t, "Test API", doc.Info.Title)
		assert.Equal(t, "1.0.0", doc.Info.Version)
	})

	t.Run("missing info", func(t *testing.T) {
		b := New(parser.OASVersion320)
		_, err := b.Build()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "info is required")
	})

	t.Run("missing title", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetVersion("1.0.0")
		_, err := b.Build()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "info.title is required")
	})

	t.Run("missing version", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API")
		_, err := b.Build()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "info.version is required")
	})

	t.Run("with accumulated errors", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0")
		b.errors = append(b.errors, assert.AnError)

		_, err := b.Build()
		assert.Error(t, err)
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

func TestBuilder_MarshalJSON(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0")

	data, err := b.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(data), `"openapi": "3.2.0"`)
	assert.Contains(t, string(data), `"title": "Test API"`)
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

	doc, err := b.Build()
	require.NoError(t, err)

	require.NotNil(t, doc.Components)
	require.NotNil(t, doc.Components.SecuritySchemes)
	assert.Contains(t, doc.Components.SecuritySchemes, "api_key")
	assert.Contains(t, doc.Components.SecuritySchemes, "bearer_auth")
	assert.Len(t, doc.Security, 2)
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

	doc, err := b.Build()
	require.NoError(t, err)

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
	assert.Contains(t, doc.Components.Schemas, "Pet")
	assert.Contains(t, doc.Components.Schemas, "Error")
}

func TestFromDocument(t *testing.T) {
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

	doc, err := b.Build()
	require.NoError(t, err)

	// Verify original content preserved
	assert.Contains(t, doc.Paths, "/existing")
	assert.Contains(t, doc.Components.Schemas, "ExistingSchema")

	// Verify new content added
	assert.Contains(t, doc.Paths, "/new")
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
	assert.Contains(t, schema.Ref, "Event")

	// Check the actual schema
	require.Contains(t, b.schemas, "Event")
	eventSchema := b.schemas["Event"]
	require.Contains(t, eventSchema.Properties, "timestamp")

	// The timestamp property should point to a time type schema with date-time format
	// Since time.Time is not a named struct, it generates directly
	timestampProp := eventSchema.Properties["timestamp"]
	assert.Equal(t, "string", timestampProp.Type)
	assert.Equal(t, "date-time", timestampProp.Format)
}
