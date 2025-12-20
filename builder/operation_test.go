package builder

import (
	"net/http"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithOperationID(t *testing.T) {
	cfg := &operationConfig{}
	WithOperationID("testOp")(cfg)
	assert.Equal(t, "testOp", cfg.operationID)
}

func TestWithSummary(t *testing.T) {
	cfg := &operationConfig{}
	WithSummary("Test summary")(cfg)
	assert.Equal(t, "Test summary", cfg.summary)
}

func TestWithDescription(t *testing.T) {
	cfg := &operationConfig{}
	WithDescription("Test description")(cfg)
	assert.Equal(t, "Test description", cfg.description)
}

func TestWithTags(t *testing.T) {
	cfg := &operationConfig{}
	WithTags("tag1", "tag2")(cfg)
	assert.Equal(t, []string{"tag1", "tag2"}, cfg.tags)
}

func TestWithDeprecated(t *testing.T) {
	cfg := &operationConfig{}
	WithDeprecated(true)(cfg)
	assert.True(t, cfg.deprecated)
}

func TestWithParameter(t *testing.T) {
	cfg := &operationConfig{}
	param := &parser.Parameter{Name: "test"}
	WithParameter(param)(cfg)
	require.Len(t, cfg.parameters, 1)
	assert.Equal(t, "test", cfg.parameters[0].param.Name)
}

func TestWithSecurity(t *testing.T) {
	cfg := &operationConfig{}
	reqs := []parser.SecurityRequirement{
		{"api_key": []string{}},
	}
	WithSecurity(reqs...)(cfg)
	assert.Equal(t, reqs, cfg.security)
}

func TestWithNoSecurity(t *testing.T) {
	cfg := &operationConfig{}
	WithNoSecurity()(cfg)
	assert.True(t, cfg.noSecurity)
}

func TestWithRequired(t *testing.T) {
	cfg := &requestBodyConfig{}
	WithRequired(true)(cfg)
	assert.True(t, cfg.required)
}

func TestWithRequestDescription(t *testing.T) {
	cfg := &requestBodyConfig{}
	WithRequestDescription("Body description")(cfg)
	assert.Equal(t, "Body description", cfg.description)
}

func TestWithRequestExample(t *testing.T) {
	cfg := &requestBodyConfig{}
	example := map[string]any{"key": "value"}
	WithRequestExample(example)(cfg)
	assert.Equal(t, example, cfg.example)
}

func TestWithResponseDescription(t *testing.T) {
	cfg := &responseConfig{}
	WithResponseDescription("Response description")(cfg)
	assert.Equal(t, "Response description", cfg.description)
}

func TestWithResponseExample(t *testing.T) {
	cfg := &responseConfig{}
	example := map[string]any{"key": "value"}
	WithResponseExample(example)(cfg)
	assert.Equal(t, example, cfg.example)
}

func TestWithResponseHeader(t *testing.T) {
	cfg := &responseConfig{}
	header := &parser.Header{Description: "Test header"}
	WithResponseHeader("X-Test", header)(cfg)
	require.NotNil(t, cfg.headers)
	assert.Equal(t, header, cfg.headers["X-Test"])
}

func TestWithParamDescription(t *testing.T) {
	cfg := &paramConfig{}
	WithParamDescription("Param description")(cfg)
	assert.Equal(t, "Param description", cfg.description)
}

func TestWithParamRequired(t *testing.T) {
	cfg := &paramConfig{}
	WithParamRequired(true)(cfg)
	assert.True(t, cfg.required)
}

func TestWithParamExample(t *testing.T) {
	cfg := &paramConfig{}
	WithParamExample("example")(cfg)
	assert.Equal(t, "example", cfg.example)
}

func TestWithParamDeprecated(t *testing.T) {
	cfg := &paramConfig{}
	WithParamDeprecated(true)(cfg)
	assert.True(t, cfg.deprecated)
}

func TestWithRequestBody(t *testing.T) {
	type Body struct {
		Field string `json:"field"`
	}

	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/test",
			WithRequestBody("application/json", Body{},
				WithRequired(true),
				WithRequestDescription("Test body"),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	require.NotNil(t, b.paths["/test"].Post.RequestBody)
	rb := b.paths["/test"].Post.RequestBody
	assert.True(t, rb.Required)
	assert.Equal(t, "Test body", rb.Description)
	require.Contains(t, rb.Content, "application/json")
	require.NotNil(t, rb.Content["application/json"].Schema)
}

func TestWithResponse(t *testing.T) {
	type Response struct {
		Success bool `json:"success"`
	}

	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithResponse(http.StatusOK, Response{},
				WithResponseDescription("Success response"),
			),
		)

	require.NotNil(t, b.paths["/test"].Get.Responses)
	require.Contains(t, b.paths["/test"].Get.Responses.Codes, "200")
	resp := b.paths["/test"].Get.Responses.Codes["200"]
	assert.Equal(t, "Success response", resp.Description)
	require.Contains(t, resp.Content, "application/json")
}

func TestWithResponseRef(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithResponseRef(http.StatusOK, "#/components/responses/Success"),
		)

	require.NotNil(t, b.paths["/test"].Get.Responses)
	require.Contains(t, b.paths["/test"].Get.Responses.Codes, "200")
	assert.Equal(t, "#/components/responses/Success", b.paths["/test"].Get.Responses.Codes["200"].Ref)
}

func TestWithResponse_OAS20(t *testing.T) {
	// Test that OAS 2.0 converts response content to direct schema
	type Response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithResponse(http.StatusOK, Response{},
				WithResponseDescription("Success response"),
			),
		)

	require.NotNil(t, b.paths["/test"].Get.Responses)
	require.Contains(t, b.paths["/test"].Get.Responses.Codes, "200")
	resp := b.paths["/test"].Get.Responses.Codes["200"]
	assert.Equal(t, "Success response", resp.Description)

	// OAS 2.0 should NOT have Content
	assert.Nil(t, resp.Content)

	// OAS 2.0 should have direct Schema field with $ref
	require.NotNil(t, resp.Schema)
	assert.Contains(t, resp.Schema.Ref, "builder.Response")

	// Check the actual schema in definitions
	require.Contains(t, b.schemas, "builder.Response")
	actualSchema := b.schemas["builder.Response"]
	assert.Equal(t, "object", actualSchema.Type)
	require.Contains(t, actualSchema.Properties, "success")
	require.Contains(t, actualSchema.Properties, "message")
}

func TestWithResponseRawSchema_OAS20(t *testing.T) {
	// Test that OAS 2.0 converts response content to direct schema with raw schema
	schema := &parser.Schema{
		Type:   "string",
		Format: "binary",
	}

	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/download",
			WithResponseRawSchema(http.StatusOK, "application/octet-stream", schema,
				WithResponseDescription("File download"),
			),
		)

	require.NotNil(t, b.paths["/download"].Get.Responses)
	require.Contains(t, b.paths["/download"].Get.Responses.Codes, "200")
	resp := b.paths["/download"].Get.Responses.Codes["200"]
	assert.Equal(t, "File download", resp.Description)

	// OAS 2.0 should NOT have Content
	assert.Nil(t, resp.Content)

	// OAS 2.0 should have direct Schema field
	require.NotNil(t, resp.Schema)
	assert.Equal(t, "string", resp.Schema.Type)
	assert.Equal(t, "binary", resp.Schema.Format)
}

func TestWithDefaultResponse(t *testing.T) {
	type ErrorResponse struct {
		Error string `json:"error"`
	}

	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithResponse(http.StatusOK, struct{}{}),
			WithDefaultResponse(ErrorResponse{},
				WithResponseDescription("Error response"),
			),
		)

	require.NotNil(t, b.paths["/test"].Get.Responses)
	require.NotNil(t, b.paths["/test"].Get.Responses.Default)
	assert.Equal(t, "Error response", b.paths["/test"].Get.Responses.Default.Description)
}

func TestWithQueryParam(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithQueryParam("limit", int32(0),
				WithParamDescription("Max results"),
				WithParamRequired(true),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	params := b.paths["/test"].Get.Parameters
	require.Len(t, params, 1)
	assert.Equal(t, "limit", params[0].Name)
	assert.Equal(t, "query", params[0].In)
	assert.Equal(t, "Max results", params[0].Description)
	assert.True(t, params[0].Required)
	require.NotNil(t, params[0].Schema)
	assert.Equal(t, "integer", params[0].Schema.Type)
}

func TestWithPathParam(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/users/{id}",
			WithPathParam("id", int64(0),
				WithParamDescription("User ID"),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	params := b.paths["/users/{id}"].Get.Parameters
	require.Len(t, params, 1)
	assert.Equal(t, "id", params[0].Name)
	assert.Equal(t, "path", params[0].In)
	assert.True(t, params[0].Required) // Path params are always required
	require.NotNil(t, params[0].Schema)
	assert.Equal(t, "integer", params[0].Schema.Type)
	assert.Equal(t, "int64", params[0].Schema.Format)
}

func TestWithHeaderParam(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithHeaderParam("X-Request-ID", "",
				WithParamDescription("Request ID"),
				WithParamRequired(true),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	params := b.paths["/test"].Get.Parameters
	require.Len(t, params, 1)
	assert.Equal(t, "X-Request-ID", params[0].Name)
	assert.Equal(t, "header", params[0].In)
	assert.True(t, params[0].Required)
}

func TestWithCookieParam(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithCookieParam("session_id", "",
				WithParamDescription("Session cookie"),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	params := b.paths["/test"].Get.Parameters
	require.Len(t, params, 1)
	assert.Equal(t, "session_id", params[0].Name)
	assert.Equal(t, "cookie", params[0].In)
}

func TestWithParameterRef(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithParameterRef("#/components/parameters/PageLimit"),
			WithResponse(http.StatusOK, struct{}{}),
		)

	params := b.paths["/test"].Get.Parameters
	require.Len(t, params, 1)
	assert.Equal(t, "#/components/parameters/PageLimit", params[0].Ref)
}

func TestBuilder_AddOperation_UnsupportedMethod(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation("INVALID", "/test")

	require.Len(t, b.errors, 1)
	assert.Contains(t, b.errors[0].Error(), "unsupported HTTP method")
}

func TestBuilder_AddOperation_QueryMethod_OAS32(t *testing.T) {
	// QUERY method should work in OAS 3.2.0
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation("QUERY", "/search",
			WithOperationID("searchQuery"),
			WithResponse(http.StatusOK, struct{}{}),
		)

	require.Empty(t, b.errors, "QUERY method should be supported in OAS 3.2.0")
	require.NotNil(t, b.paths["/search"])
	assert.NotNil(t, b.paths["/search"].Query, "QUERY operation should be set")
	assert.Equal(t, "searchQuery", b.paths["/search"].Query.OperationID)
}

func TestBuilder_AddOperation_QueryMethod_OAS31_Error(t *testing.T) {
	// QUERY method should fail in OAS 3.1
	b := New(parser.OASVersion310).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation("QUERY", "/search",
			WithOperationID("searchQuery"),
			WithResponse(http.StatusOK, struct{}{}),
		)

	require.Len(t, b.errors, 1, "QUERY method should not be supported in OAS 3.1")
	assert.Contains(t, b.errors[0].Error(), "QUERY method is only supported in OAS 3.2.0+")
}

func TestBuilder_AddOperation_QueryMethod_OAS20_Error(t *testing.T) {
	// QUERY method should fail in OAS 2.0
	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation("QUERY", "/search",
			WithOperationID("searchQuery"),
			WithResponse(http.StatusOK, struct{}{}),
		)

	require.Len(t, b.errors, 1, "QUERY method should not be supported in OAS 2.0")
	assert.Contains(t, b.errors[0].Error(), "QUERY method is only supported in OAS 3.2.0+")
}

func TestBuilder_AddOperation_WithNoSecurity(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/public",
			WithNoSecurity(),
			WithResponse(http.StatusOK, struct{}{}),
		)

	require.NotNil(t, b.paths["/public"].Get.Security)
	require.Len(t, b.paths["/public"].Get.Security, 1)
	assert.Empty(t, b.paths["/public"].Get.Security[0])
}

func TestBuilder_AddOperation_WithExplicitSecurity(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddAPIKeySecurityScheme("api_key", "header", "X-API-Key", "API key").
		AddOperation(http.MethodGet, "/protected",
			WithSecurity(SecurityRequirement("api_key")),
			WithResponse(http.StatusOK, struct{}{}),
		)

	require.NotNil(t, b.paths["/protected"].Get.Security)
	require.Len(t, b.paths["/protected"].Get.Security, 1)
	assert.Contains(t, b.paths["/protected"].Get.Security[0], "api_key")
}

func TestBuilder_AddOperation_MultipleOnSamePath(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/users",
			WithOperationID("listUsers"),
			WithResponse(http.StatusOK, []struct{}{}),
		).
		AddOperation(http.MethodPost, "/users",
			WithOperationID("createUser"),
			WithResponse(http.StatusCreated, struct{}{}),
		)

	require.Contains(t, b.paths, "/users")
	require.NotNil(t, b.paths["/users"].Get)
	require.NotNil(t, b.paths["/users"].Post)
	assert.Equal(t, "listUsers", b.paths["/users"].Get.OperationID)
	assert.Equal(t, "createUser", b.paths["/users"].Post.OperationID)
}

func TestWithRequestBodyRawSchema(t *testing.T) {
	schema := &parser.Schema{
		Type:   "string",
		Format: "binary",
	}

	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/upload",
			WithRequestBodyRawSchema("application/octet-stream", schema,
				WithRequired(true),
				WithRequestDescription("File upload"),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	require.NotNil(t, b.paths["/upload"].Post.RequestBody)
	rb := b.paths["/upload"].Post.RequestBody
	assert.True(t, rb.Required)
	assert.Equal(t, "File upload", rb.Description)
	require.Contains(t, rb.Content, "application/octet-stream")
	require.NotNil(t, rb.Content["application/octet-stream"].Schema)
	assert.Equal(t, "string", rb.Content["application/octet-stream"].Schema.Type)
	assert.Equal(t, "binary", rb.Content["application/octet-stream"].Schema.Format)
}

func TestWithResponseRawSchema(t *testing.T) {
	schema := &parser.Schema{
		Type:   "string",
		Format: "binary",
	}

	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/download",
			WithResponseRawSchema(http.StatusOK, "application/octet-stream", schema,
				WithResponseDescription("File download"),
			),
		)

	require.NotNil(t, b.paths["/download"].Get.Responses)
	require.Contains(t, b.paths["/download"].Get.Responses.Codes, "200")
	resp := b.paths["/download"].Get.Responses.Codes["200"]
	assert.Equal(t, "File download", resp.Description)
	require.Contains(t, resp.Content, "application/octet-stream")
	require.NotNil(t, resp.Content["application/octet-stream"].Schema)
	assert.Equal(t, "string", resp.Content["application/octet-stream"].Schema.Type)
	assert.Equal(t, "binary", resp.Content["application/octet-stream"].Schema.Format)
}

func TestWithFileParam_OAS20(t *testing.T) {
	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/upload",
			WithFileParam("file",
				WithParamDescription("File to upload"),
				WithParamRequired(true),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	params := b.paths["/upload"].Post.Parameters
	require.Len(t, params, 1)
	assert.Equal(t, "file", params[0].Name)
	assert.Equal(t, parser.ParamInFormData, params[0].In)
	assert.Equal(t, "file", params[0].Type)
	assert.Equal(t, "File to upload", params[0].Description)
	assert.True(t, params[0].Required)
}

func TestWithFileParam_OAS3(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/upload",
			WithFileParam("file",
				WithParamDescription("File to upload"),
				WithParamRequired(true),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	require.NotNil(t, b.paths["/upload"].Post.RequestBody)
	rb := b.paths["/upload"].Post.RequestBody
	require.Contains(t, rb.Content, "multipart/form-data")
	schema := rb.Content["multipart/form-data"].Schema
	require.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
	require.Contains(t, schema.Properties, "file")
	fileSchema := schema.Properties["file"]
	assert.Equal(t, "string", fileSchema.Type)
	assert.Equal(t, "binary", fileSchema.Format)
	assert.Equal(t, "File to upload", fileSchema.Description)
	require.Contains(t, schema.Required, "file")
}

func TestWithFormParam_OAS3_NoFile(t *testing.T) {
	// Test that form params without files use application/x-www-form-urlencoded
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/login",
			WithFormParam("username", "",
				WithParamRequired(true),
				WithParamDescription("Username"),
			),
			WithFormParam("password", "",
				WithParamRequired(true),
				WithParamDescription("Password"),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	require.NotNil(t, b.paths["/login"].Post.RequestBody)
	rb := b.paths["/login"].Post.RequestBody
	require.Contains(t, rb.Content, "application/x-www-form-urlencoded")
	schema := rb.Content["application/x-www-form-urlencoded"].Schema
	require.NotNil(t, schema)
	assert.Equal(t, "object", schema.Type)
	require.Contains(t, schema.Properties, "username")
	require.Contains(t, schema.Properties, "password")
	require.Len(t, schema.Required, 2)
}

func TestWithFileParam_MultipleFiles(t *testing.T) {
	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/upload-multiple",
			WithFileParam("file1", WithParamRequired(true)),
			WithFileParam("file2", WithParamRequired(false)),
			WithFormParam("description", "", WithParamDescription("Upload description")),
			WithResponse(http.StatusOK, struct{}{}),
		)

	params := b.paths["/upload-multiple"].Post.Parameters
	require.Len(t, params, 3)

	// Check file1
	assert.Equal(t, "file1", params[0].Name)
	assert.Equal(t, "file", params[0].Type)
	assert.True(t, params[0].Required)

	// Check file2
	assert.Equal(t, "file2", params[1].Name)
	assert.Equal(t, "file", params[1].Type)
	assert.False(t, params[1].Required)

	// Check description
	assert.Equal(t, "description", params[2].Name)
	assert.Equal(t, "Upload description", params[2].Description)
}

func TestWithRequestBodyRawSchema_WithExample(t *testing.T) {
	schema := &parser.Schema{
		Type: "object",
		Properties: map[string]*parser.Schema{
			"name": {Type: "string"},
		},
	}
	example := map[string]any{"name": "test"}

	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/test",
			WithRequestBodyRawSchema("application/json", schema,
				WithRequestExample(example),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	rb := b.paths["/test"].Post.RequestBody
	require.NotNil(t, rb)
	require.Contains(t, rb.Content, "application/json")
	assert.Equal(t, example, rb.Content["application/json"].Example)
}

func TestWithRequestBodyRawSchema_OAS20(t *testing.T) {
	// Test that OAS 2.0 converts requestBody to body parameter
	schema := &parser.Schema{
		Type:   "string",
		Format: "binary",
	}

	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/upload",
			WithRequestBodyRawSchema("application/octet-stream", schema,
				WithRequired(true),
				WithRequestDescription("File upload"),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	// OAS 2.0 should NOT have RequestBody
	assert.Nil(t, b.paths["/upload"].Post.RequestBody)

	// OAS 2.0 should have a body parameter
	params := b.paths["/upload"].Post.Parameters
	require.Len(t, params, 1)
	assert.Equal(t, "body", params[0].Name)
	assert.Equal(t, parser.ParamInBody, params[0].In)
	assert.Equal(t, "File upload", params[0].Description)
	assert.True(t, params[0].Required)
	require.NotNil(t, params[0].Schema)
	assert.Equal(t, "string", params[0].Schema.Type)
	assert.Equal(t, "binary", params[0].Schema.Format)
}

func TestWithRequestBody_OAS20(t *testing.T) {
	// Test that OAS 2.0 converts requestBody to body parameter with reflection
	type RequestBody struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/users",
			WithRequestBody("application/json", RequestBody{},
				WithRequired(true),
				WithRequestDescription("User data"),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	// OAS 2.0 should NOT have RequestBody
	assert.Nil(t, b.paths["/users"].Post.RequestBody)

	// OAS 2.0 should have a body parameter
	params := b.paths["/users"].Post.Parameters
	require.Len(t, params, 1)
	assert.Equal(t, "body", params[0].Name)
	assert.Equal(t, parser.ParamInBody, params[0].In)
	assert.Equal(t, "User data", params[0].Description)
	assert.True(t, params[0].Required)
	require.NotNil(t, params[0].Schema)

	// Schema should be a $ref to the generated schema
	assert.Contains(t, params[0].Schema.Ref, "builder.RequestBody")

	// Check the actual schema in definitions
	require.Contains(t, b.schemas, "builder.RequestBody")
	actualSchema := b.schemas["builder.RequestBody"]
	assert.Equal(t, "object", actualSchema.Type)
	require.Contains(t, actualSchema.Properties, "name")
	require.Contains(t, actualSchema.Properties, "email")
}

func TestWithResponseRawSchema_WithHeaders(t *testing.T) {
	schema := &parser.Schema{
		Type:   "string",
		Format: "binary",
	}
	header := &parser.Header{
		Description: "Content disposition",
		Schema:      &parser.Schema{Type: "string"},
	}

	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/download",
			WithResponseRawSchema(http.StatusOK, "application/pdf", schema,
				WithResponseDescription("PDF download"),
				WithResponseHeader("Content-Disposition", header),
			),
		)

	resp := b.paths["/download"].Get.Responses.Codes["200"]
	require.NotNil(t, resp)
	require.Contains(t, resp.Headers, "Content-Disposition")
	assert.Equal(t, "Content disposition", resp.Headers["Content-Disposition"].Description)
}

func TestWithFileParam_IgnoresConstraints_OAS20(t *testing.T) {
	// Test that parameter constraints are ignored for file parameters in OAS 2.0
	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/upload",
			WithFileParam("file",
				WithParamDescription("File to upload"),
				WithParamRequired(true),
				WithParamMinLength(10),   // Should be ignored
				WithParamMaxLength(1000), // Should be ignored
				WithParamPattern(".*"),   // Should be ignored
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	params := b.paths["/upload"].Post.Parameters
	require.Len(t, params, 1)
	assert.Equal(t, "file", params[0].Name)
	assert.Equal(t, "file", params[0].Type)
	assert.Equal(t, "File to upload", params[0].Description)
	assert.True(t, params[0].Required)

	// Verify constraints are not applied to file parameters
	assert.Nil(t, params[0].MinLength, "minLength should not be set for file parameters")
	assert.Nil(t, params[0].MaxLength, "maxLength should not be set for file parameters")
	assert.Empty(t, params[0].Pattern, "pattern should not be set for file parameters")
	assert.Nil(t, params[0].Schema, "schema should not be set for file parameters in OAS 2.0")
}

func TestWithFileParam_IgnoresConstraints_OAS3(t *testing.T) {
	// Test that parameter constraints are ignored for file parameters in OAS 3.x
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/upload",
			WithFileParam("file",
				WithParamDescription("File to upload"),
				WithParamRequired(true),
				WithParamMinLength(10),   // Should be ignored
				WithParamMaxLength(1000), // Should be ignored
				WithParamPattern(".*"),   // Should be ignored
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	rb := b.paths["/upload"].Post.RequestBody
	require.NotNil(t, rb)
	require.Contains(t, rb.Content, "multipart/form-data")

	schema := rb.Content["multipart/form-data"].Schema
	require.NotNil(t, schema)
	require.Contains(t, schema.Properties, "file")

	fileSchema := schema.Properties["file"]
	assert.Equal(t, "string", fileSchema.Type)
	assert.Equal(t, "binary", fileSchema.Format)
	assert.Equal(t, "File to upload", fileSchema.Description)

	// Verify constraints are not applied to file schema
	assert.Nil(t, fileSchema.MinLength, "minLength should not be set for file parameters")
	assert.Nil(t, fileSchema.MaxLength, "maxLength should not be set for file parameters")
	assert.Empty(t, fileSchema.Pattern, "pattern should not be set for file parameters")
}

func TestWithFileParam_EmptyName(t *testing.T) {
	// Test that empty file parameter name is handled (though not recommended)
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/upload",
			WithFileParam("", WithParamRequired(true)),
			WithResponse(http.StatusOK, struct{}{}),
		)

	rb := b.paths["/upload"].Post.RequestBody
	require.NotNil(t, rb)
	require.Contains(t, rb.Content, "multipart/form-data")

	schema := rb.Content["multipart/form-data"].Schema
	require.NotNil(t, schema)

	// Empty name is allowed (though not valid OpenAPI), it creates a property with empty key
	require.Contains(t, schema.Properties, "")
	fileSchema := schema.Properties[""]
	assert.Equal(t, "string", fileSchema.Type)
	assert.Equal(t, "binary", fileSchema.Format)
}

// Tests for WithConsumes and WithProduces

func TestWithConsumes(t *testing.T) {
	cfg := &operationConfig{}
	WithConsumes("application/json", "application/xml")(cfg)
	assert.Equal(t, []string{"application/json", "application/xml"}, cfg.consumes)
}

func TestWithProduces(t *testing.T) {
	cfg := &operationConfig{}
	WithProduces("application/json", "text/plain")(cfg)
	assert.Equal(t, []string{"application/json", "text/plain"}, cfg.produces)
}

func TestWithConsumesProduces_OAS2(t *testing.T) {
	type User struct {
		Name string `json:"name"`
	}

	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/users",
			WithConsumes("application/json", "application/xml"),
			WithProduces("application/json"),
			WithRequestBody("application/json", User{}),
			WithResponse(http.StatusOK, User{}),
		)

	doc, err := b.BuildOAS2()
	require.NoError(t, err)

	op := doc.Paths["/users"].Post
	assert.Equal(t, []string{"application/json", "application/xml"}, op.Consumes)
	assert.Equal(t, []string{"application/json"}, op.Produces)
}

func TestWithConsumesProduces_IgnoredForOAS3Output(t *testing.T) {
	// Even if set, consumes/produces should be empty for OAS 3.x
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/test",
			WithConsumes("application/json"), // Should be stored but fields are omitempty
			WithProduces("application/json"), // Should be stored but fields are omitempty
			WithRequestBody("application/json", struct{}{}),
			WithResponse(http.StatusOK, struct{}{}),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	op := doc.Paths["/test"].Post
	// Fields are set but will be empty in serialized output due to omitempty
	assert.Equal(t, []string{"application/json"}, op.Consumes)
	assert.Equal(t, []string{"application/json"}, op.Produces)
}

// Tests for WithRequestBodyContentTypes

func TestWithRequestBodyContentTypes_OAS3(t *testing.T) {
	type User struct {
		Name string `json:"name"`
	}

	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/users",
			WithRequestBodyContentTypes(
				[]string{"application/json", "application/xml", "text/yaml"},
				User{},
				WithRequired(true),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	rb := doc.Paths["/users"].Post.RequestBody
	require.NotNil(t, rb)
	assert.True(t, rb.Required)

	// All content types should be present
	require.Contains(t, rb.Content, "application/json")
	require.Contains(t, rb.Content, "application/xml")
	require.Contains(t, rb.Content, "text/yaml")

	// All should reference the same schema
	for _, mediaType := range rb.Content {
		require.NotNil(t, mediaType.Schema)
		assert.Contains(t, mediaType.Schema.Ref, "User")
	}
}

func TestWithRequestBodyContentTypes_EmptySlice(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/test",
			WithRequestBodyContentTypes([]string{}, struct{}{}),
			WithResponse(http.StatusOK, struct{}{}),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	// Should have no request body when content types is empty
	assert.Nil(t, doc.Paths["/test"].Post.RequestBody)
}

func TestWithRequestBodyContentTypes_OAS2(t *testing.T) {
	// For OAS 2.0, should use only the first content type for the body parameter
	type User struct {
		Name string `json:"name"`
	}

	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/users",
			WithRequestBodyContentTypes(
				[]string{"application/json", "application/xml"},
				User{},
				WithRequired(true),
				WithRequestDescription("User data"),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	doc, err := b.BuildOAS2()
	require.NoError(t, err)

	// OAS 2.0 should have a body parameter
	params := doc.Paths["/users"].Post.Parameters
	require.Len(t, params, 1)
	assert.Equal(t, "body", params[0].Name)
	assert.Equal(t, parser.ParamInBody, params[0].In)
	assert.Equal(t, "User data", params[0].Description)
	assert.True(t, params[0].Required)
}

// Tests for WithResponseContentTypes

func TestWithResponseContentTypes_OAS3(t *testing.T) {
	type User struct {
		ID int `json:"id"`
	}

	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/users/{id}",
			WithPathParam("id", int64(0)),
			WithResponseContentTypes(
				http.StatusOK,
				[]string{"application/json", "application/xml"},
				User{},
				WithResponseDescription("User found"),
			),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	resp := doc.Paths["/users/{id}"].Get.Responses.Codes["200"]
	require.NotNil(t, resp)
	assert.Equal(t, "User found", resp.Description)

	require.Contains(t, resp.Content, "application/json")
	require.Contains(t, resp.Content, "application/xml")

	// Both should reference the same schema
	for _, mediaType := range resp.Content {
		require.NotNil(t, mediaType.Schema)
		assert.Contains(t, mediaType.Schema.Ref, "User")
	}
}

func TestWithResponseContentTypes_EmptySlice(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithResponseContentTypes(http.StatusOK, []string{}, struct{}{}),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	// Should have no responses when content types is empty
	assert.Nil(t, doc.Paths["/test"].Get.Responses)
}

func TestWithResponseContentTypes_OAS2(t *testing.T) {
	// For OAS 2.0, should extract first content type's schema
	type User struct {
		ID int `json:"id"`
	}

	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/users/{id}",
			WithPathParam("id", int64(0)),
			WithResponseContentTypes(
				http.StatusOK,
				[]string{"application/json", "application/xml"},
				User{},
				WithResponseDescription("User found"),
			),
		)

	doc, err := b.BuildOAS2()
	require.NoError(t, err)

	resp := doc.Paths["/users/{id}"].Get.Responses.Codes["200"]
	require.NotNil(t, resp)
	assert.Equal(t, "User found", resp.Description)

	// OAS 2.0 should have direct schema, not content map
	assert.Nil(t, resp.Content)
	require.NotNil(t, resp.Schema)
	assert.Contains(t, resp.Schema.Ref, "User")
}

func TestRequestBodyMethods_LastWins(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/test",
			WithRequestBody("text/plain", struct{}{}),
			WithRequestBodyContentTypes([]string{"application/json", "application/xml"}, struct{}{}),
			WithResponse(http.StatusOK, struct{}{}),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	rb := doc.Paths["/test"].Post.RequestBody
	require.NotNil(t, rb)

	// Should have the multi-content types, not text/plain
	assert.NotContains(t, rb.Content, "text/plain")
	assert.Contains(t, rb.Content, "application/json")
	assert.Contains(t, rb.Content, "application/xml")
}

func TestWithResponseContentTypes_WithHeaders(t *testing.T) {
	header := &parser.Header{
		Description: "Rate limit remaining",
		Schema:      &parser.Schema{Type: "integer"},
	}

	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithResponseContentTypes(
				http.StatusOK,
				[]string{"application/json", "application/xml"},
				struct{}{},
				WithResponseDescription("Success"),
				WithResponseHeader("X-Rate-Limit-Remaining", header),
			),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	resp := doc.Paths["/test"].Get.Responses.Codes["200"]
	require.NotNil(t, resp)
	require.Contains(t, resp.Headers, "X-Rate-Limit-Remaining")
	assert.Equal(t, "Rate limit remaining", resp.Headers["X-Rate-Limit-Remaining"].Description)
}

// Extension tests

func TestWithOperationExtension(t *testing.T) {
	cfg := &operationConfig{}
	WithOperationExtension("x-rate-limit", 100)(cfg)
	WithOperationExtension("x-internal", true)(cfg)

	require.NotNil(t, cfg.extensions)
	assert.Equal(t, 100, cfg.extensions["x-rate-limit"])
	assert.Equal(t, true, cfg.extensions["x-internal"])
}

func TestWithOperationExtension_OAS3Output(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithOperationExtension("x-rate-limit", 100),
			WithOperationExtension("x-custom-data", map[string]string{"key": "value"}),
			WithResponse(http.StatusOK, struct{}{}),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	op := doc.Paths["/test"].Get
	require.NotNil(t, op)
	require.NotNil(t, op.Extra)
	assert.Equal(t, 100, op.Extra["x-rate-limit"])
	assert.Equal(t, map[string]string{"key": "value"}, op.Extra["x-custom-data"])
}

func TestWithOperationExtension_OAS2Output(t *testing.T) {
	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithOperationExtension("x-rate-limit", 100),
			WithResponse(http.StatusOK, struct{}{}),
		)

	doc, err := b.BuildOAS2()
	require.NoError(t, err)

	op := doc.Paths["/test"].Get
	require.NotNil(t, op)
	require.NotNil(t, op.Extra)
	assert.Equal(t, 100, op.Extra["x-rate-limit"])
}

func TestWithParamExtension(t *testing.T) {
	cfg := &paramConfig{}
	WithParamExtension("x-example-values", []int{10, 25, 50})(cfg)

	require.NotNil(t, cfg.extensions)
	assert.Equal(t, []int{10, 25, 50}, cfg.extensions["x-example-values"])
}

func TestWithParamExtension_OAS3Output(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithQueryParam("limit", 0,
				WithParamExtension("x-example-values", []int{10, 25, 50}),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	params := doc.Paths["/test"].Get.Parameters
	require.Len(t, params, 1)
	require.NotNil(t, params[0].Extra)
	assert.Equal(t, []int{10, 25, 50}, params[0].Extra["x-example-values"])
}

func TestWithParamExtension_OAS2Output(t *testing.T) {
	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithQueryParam("limit", 0,
				WithParamExtension("x-example-values", []int{10, 25, 50}),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	doc, err := b.BuildOAS2()
	require.NoError(t, err)

	params := doc.Paths["/test"].Get.Parameters
	require.Len(t, params, 1)
	require.NotNil(t, params[0].Extra)
	assert.Equal(t, []int{10, 25, 50}, params[0].Extra["x-example-values"])
}

func TestWithParamAllowEmptyValue(t *testing.T) {
	cfg := &paramConfig{}
	WithParamAllowEmptyValue(true)(cfg)

	assert.True(t, cfg.allowEmptyValue)
}

func TestWithParamAllowEmptyValue_OAS2Output(t *testing.T) {
	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithQueryParam("filter", "",
				WithParamAllowEmptyValue(true),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	doc, err := b.BuildOAS2()
	require.NoError(t, err)

	params := doc.Paths["/test"].Get.Parameters
	require.Len(t, params, 1)
	assert.True(t, params[0].AllowEmptyValue)
}

func TestWithParamCollectionFormat(t *testing.T) {
	cfg := &paramConfig{}
	WithParamCollectionFormat("csv")(cfg)

	assert.Equal(t, "csv", cfg.collectionFormat)
}

func TestWithParamCollectionFormat_OAS2Output(t *testing.T) {
	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithQueryParam("tags", []string{},
				WithParamCollectionFormat("pipes"),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	doc, err := b.BuildOAS2()
	require.NoError(t, err)

	params := doc.Paths["/test"].Get.Parameters
	require.Len(t, params, 1)
	assert.Equal(t, "pipes", params[0].CollectionFormat)
}

func TestWithResponseExtension(t *testing.T) {
	cfg := &responseConfig{}
	WithResponseExtension("x-cache-ttl", 3600)(cfg)

	require.NotNil(t, cfg.extensions)
	assert.Equal(t, 3600, cfg.extensions["x-cache-ttl"])
}

func TestWithResponseExtension_OAS3Output(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithResponse(http.StatusOK, struct{}{},
				WithResponseExtension("x-cache-ttl", 3600),
			),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	resp := doc.Paths["/test"].Get.Responses.Codes["200"]
	require.NotNil(t, resp)
	require.NotNil(t, resp.Extra)
	assert.Equal(t, 3600, resp.Extra["x-cache-ttl"])
}

func TestWithResponseExtension_OAS2Output(t *testing.T) {
	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithResponse(http.StatusOK, struct{}{},
				WithResponseExtension("x-cache-ttl", 3600),
			),
		)

	doc, err := b.BuildOAS2()
	require.NoError(t, err)

	resp := doc.Paths["/test"].Get.Responses.Codes["200"]
	require.NotNil(t, resp)
	require.NotNil(t, resp.Extra)
	assert.Equal(t, 3600, resp.Extra["x-cache-ttl"])
}

func TestWithResponseExtension_DefaultResponse(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/test",
			WithDefaultResponse(struct{}{},
				WithResponseExtension("x-error-codes", []string{"E001", "E002"}),
			),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	resp := doc.Paths["/test"].Get.Responses.Default
	require.NotNil(t, resp)
	require.NotNil(t, resp.Extra)
	assert.Equal(t, []string{"E001", "E002"}, resp.Extra["x-error-codes"])
}

func TestWithRequestBodyExtension(t *testing.T) {
	cfg := &requestBodyConfig{}
	WithRequestBodyExtension("x-codegen-request-body-name", "user")(cfg)

	require.NotNil(t, cfg.extensions)
	assert.Equal(t, "user", cfg.extensions["x-codegen-request-body-name"])
}

func TestWithRequestBodyExtension_OAS3Output(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/users",
			WithRequestBody("application/json", struct{}{},
				WithRequestBodyExtension("x-codegen-request-body-name", "user"),
			),
			WithResponse(http.StatusCreated, struct{}{}),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	rb := doc.Paths["/users"].Post.RequestBody
	require.NotNil(t, rb)
	require.NotNil(t, rb.Extra)
	assert.Equal(t, "user", rb.Extra["x-codegen-request-body-name"])
}

func TestWithRequestBodyExtension_OAS2BodyParam(t *testing.T) {
	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/users",
			WithRequestBody("application/json", struct{}{},
				WithRequestBodyExtension("x-codegen-request-body-name", "user"),
			),
			WithResponse(http.StatusCreated, struct{}{}),
		)

	doc, err := b.BuildOAS2()
	require.NoError(t, err)

	params := doc.Paths["/users"].Post.Parameters
	var bodyParam *parser.Parameter
	for _, p := range params {
		if p.In == parser.ParamInBody {
			bodyParam = p
			break
		}
	}
	require.NotNil(t, bodyParam)
	require.NotNil(t, bodyParam.Extra)
	assert.Equal(t, "user", bodyParam.Extra["x-codegen-request-body-name"])
}

func TestWithRequestBodyExtension_RawSchema(t *testing.T) {
	schema := &parser.Schema{
		Type:   "string",
		Format: "binary",
	}

	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/upload",
			WithRequestBodyRawSchema("application/octet-stream", schema,
				WithRequestBodyExtension("x-max-size", "10MB"),
			),
			WithResponse(http.StatusCreated, struct{}{}),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	rb := doc.Paths["/upload"].Post.RequestBody
	require.NotNil(t, rb)
	require.NotNil(t, rb.Extra)
	assert.Equal(t, "10MB", rb.Extra["x-max-size"])
}

func TestWithRequestBodyExtension_MultipleContentTypes(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/users",
			WithRequestBodyContentTypes(
				[]string{"application/json", "application/xml"},
				struct{}{},
				WithRequestBodyExtension("x-supports-xml", true),
			),
			WithResponse(http.StatusCreated, struct{}{}),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	rb := doc.Paths["/users"].Post.RequestBody
	require.NotNil(t, rb)
	require.NotNil(t, rb.Extra)
	assert.Equal(t, true, rb.Extra["x-supports-xml"])
}

func TestCombinedExtensions_AllComponents(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddOperation(http.MethodPost, "/users/{id}",
			WithOperationExtension("x-operation-custom", "op-value"),
			WithPathParam("id", "",
				WithParamExtension("x-param-custom", "param-value"),
			),
			WithRequestBody("application/json", struct{}{},
				WithRequestBodyExtension("x-body-custom", "body-value"),
			),
			WithResponse(http.StatusOK, struct{}{},
				WithResponseExtension("x-response-custom", "response-value"),
			),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	op := doc.Paths["/users/{id}"].Post
	require.NotNil(t, op)

	// Check operation extension
	assert.Equal(t, "op-value", op.Extra["x-operation-custom"])

	// Check parameter extension
	require.Len(t, op.Parameters, 1)
	assert.Equal(t, "param-value", op.Parameters[0].Extra["x-param-custom"])

	// Check request body extension
	require.NotNil(t, op.RequestBody)
	assert.Equal(t, "body-value", op.RequestBody.Extra["x-body-custom"])

	// Check response extension
	resp := op.Responses.Codes["200"]
	require.NotNil(t, resp)
	assert.Equal(t, "response-value", resp.Extra["x-response-custom"])
}

func TestAddParameter_WithExtension(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddParameter("pageSize", parser.ParamInQuery, "page_size", 10,
			WithParamExtension("x-default-value", 20),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	param, ok := doc.Components.Parameters["pageSize"]
	require.True(t, ok)
	require.NotNil(t, param.Extra)
	assert.Equal(t, 20, param.Extra["x-default-value"])
}

func TestAddParameter_WithAllowEmptyValue_OAS2(t *testing.T) {
	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddParameter("filter", parser.ParamInQuery, "filter", "",
			WithParamAllowEmptyValue(true),
		)

	doc, err := b.BuildOAS2()
	require.NoError(t, err)

	param, ok := doc.Parameters["filter"]
	require.True(t, ok)
	assert.True(t, param.AllowEmptyValue)
}

func TestAddParameter_WithCollectionFormat_OAS2(t *testing.T) {
	b := New(parser.OASVersion20).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddParameter("tags", parser.ParamInQuery, "tags", []string{},
			WithParamCollectionFormat("multi"),
		)

	doc, err := b.BuildOAS2()
	require.NoError(t, err)

	param, ok := doc.Parameters["tags"]
	require.True(t, ok)
	assert.Equal(t, "multi", param.CollectionFormat)
}

func TestAddResponse_WithExtension(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test").
		SetVersion("1.0.0").
		AddResponse("NotFound", "Resource not found", struct{}{},
			WithResponseExtension("x-error-code", "NOT_FOUND"),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	resp, ok := doc.Components.Responses["NotFound"]
	require.True(t, ok)
	require.NotNil(t, resp.Extra)
	assert.Equal(t, "NOT_FOUND", resp.Extra["x-error-code"])
}
