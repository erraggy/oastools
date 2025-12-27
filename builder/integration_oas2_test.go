package builder

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOAS2Integration_RequestBodyAndResponse verifies the complete OAS 2.0 generation flow
func TestOAS2Integration_RequestBodyAndResponse(t *testing.T) {
	type CreateUserRequest struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	type CreateUserResponse struct {
		ID      int    `json:"id"`
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	b := New(parser.OASVersion20).
		SetTitle("User API").
		SetVersion("1.0.0").
		SetDescription("API for user management").
		AddOperation("POST", "/users",
			WithOperationID("createUser"),
			WithSummary("Create a new user"),
			WithRequestBody("application/json", CreateUserRequest{},
				WithRequired(true),
				WithRequestDescription("User creation data"),
			),
			WithResponse(200, CreateUserResponse{},
				WithResponseDescription("User created successfully"),
			),
		)

	doc, err := b.BuildOAS2()
	require.NoError(t, err)

	// Verify document structure
	assert.Equal(t, "2.0", doc.Swagger)
	assert.Equal(t, "User API", doc.Info.Title)
	assert.Equal(t, "1.0.0", doc.Info.Version)

	// Verify operation exists
	require.Contains(t, doc.Paths, "/users")
	require.NotNil(t, doc.Paths["/users"].Post)
	op := doc.Paths["/users"].Post

	// Verify operation details
	assert.Equal(t, "createUser", op.OperationID)
	assert.Equal(t, "Create a new user", op.Summary)

	// Verify NO requestBody field (OAS 3.x only)
	assert.Nil(t, op.RequestBody, "OAS 2.0 should not have requestBody field")

	// Verify body parameter exists
	require.Len(t, op.Parameters, 1, "OAS 2.0 should have body parameter")
	bodyParam := op.Parameters[0]
	assert.Equal(t, "body", bodyParam.Name)
	assert.Equal(t, parser.ParamInBody, bodyParam.In)
	assert.Equal(t, "User creation data", bodyParam.Description)
	assert.True(t, bodyParam.Required)
	require.NotNil(t, bodyParam.Schema)
	assert.Contains(t, bodyParam.Schema.Ref, "CreateUserRequest")

	// Verify response
	require.NotNil(t, op.Responses)
	require.Contains(t, op.Responses.Codes, "200")
	resp := op.Responses.Codes["200"]
	assert.Equal(t, "User created successfully", resp.Description)

	// Verify NO content field (OAS 3.x only)
	assert.Nil(t, resp.Content, "OAS 2.0 should not have content field in responses")

	// Verify direct schema field
	require.NotNil(t, resp.Schema, "OAS 2.0 should have direct schema field in responses")
	assert.Contains(t, resp.Schema.Ref, "CreateUserResponse")

	// Verify definitions are created
	require.Contains(t, doc.Definitions, "builder.CreateUserRequest")
	require.Contains(t, doc.Definitions, "builder.CreateUserResponse")

	// Verify request schema
	reqSchema := doc.Definitions["builder.CreateUserRequest"]
	assert.Equal(t, "object", reqSchema.Type)
	require.Contains(t, reqSchema.Properties, "name")
	require.Contains(t, reqSchema.Properties, "email")

	// Verify response schema
	respSchema := doc.Definitions["builder.CreateUserResponse"]
	assert.Equal(t, "object", respSchema.Type)
	require.Contains(t, respSchema.Properties, "id")
	require.Contains(t, respSchema.Properties, "success")
	require.Contains(t, respSchema.Properties, "message")

	// Verify document can be marshaled to JSON without errors
	jsonBytes, err := json.MarshalIndent(doc, "", "  ")
	require.NoError(t, err)
	assert.NotEmpty(t, jsonBytes)

	// Verify key JSON structure (no requestBody or content fields)
	var docMap map[string]any
	err = json.Unmarshal(jsonBytes, &docMap)
	require.NoError(t, err)

	paths := docMap["paths"].(map[string]any)
	usersPath := paths["/users"].(map[string]any)
	postOp := usersPath["post"].(map[string]any)

	// Should NOT have requestBody
	_, hasRequestBody := postOp["requestBody"]
	assert.False(t, hasRequestBody, "OAS 2.0 JSON should not contain requestBody field")

	// Should have parameters with body parameter
	params := postOp["parameters"].([]any)
	require.Len(t, params, 1)
	bodyParamMap := params[0].(map[string]any)
	assert.Equal(t, "body", bodyParamMap["in"])

	// Response should NOT have content
	responses := postOp["responses"].(map[string]any)
	response200 := responses["200"].(map[string]any)
	_, hasContent := response200["content"]
	assert.False(t, hasContent, "OAS 2.0 JSON should not contain content field in responses")

	// Response should have direct schema
	_, hasSchema := response200["schema"]
	assert.True(t, hasSchema, "OAS 2.0 JSON should have direct schema field in responses")
}

// TestOAS2Integration_RawSchema verifies WithRequestBodyRawSchema and WithResponseRawSchema for OAS 2.0
func TestOAS2Integration_RawSchema(t *testing.T) {
	uploadSchema := &parser.Schema{
		Type:   "string",
		Format: "binary",
	}

	downloadSchema := &parser.Schema{
		Type:   "string",
		Format: "binary",
	}

	b := New(parser.OASVersion20).
		SetTitle("File API").
		SetVersion("1.0.0").
		AddOperation("POST", "/upload",
			WithRequestBodyRawSchema("application/octet-stream", uploadSchema,
				WithRequired(true),
				WithRequestDescription("Binary file data"),
			),
			WithResponse(200, struct{ Success bool }{},
				WithResponseDescription("Upload successful"),
			),
		).
		AddOperation("GET", "/download",
			WithResponseRawSchema(200, "application/octet-stream", downloadSchema,
				WithResponseDescription("Binary file download"),
			),
		)

	doc, err := b.BuildOAS2()
	require.NoError(t, err)

	// Verify upload operation
	uploadOp := doc.Paths["/upload"].Post
	assert.Nil(t, uploadOp.RequestBody)
	require.Len(t, uploadOp.Parameters, 1)
	assert.Equal(t, parser.ParamInBody, uploadOp.Parameters[0].In)
	assert.Equal(t, "Binary file data", uploadOp.Parameters[0].Description)
	require.NotNil(t, uploadOp.Parameters[0].Schema)
	assert.Equal(t, "string", uploadOp.Parameters[0].Schema.Type)
	assert.Equal(t, "binary", uploadOp.Parameters[0].Schema.Format)

	// Verify download operation
	downloadOp := doc.Paths["/download"].Get
	require.NotNil(t, downloadOp.Responses)
	downloadResp := downloadOp.Responses.Codes["200"]
	assert.Nil(t, downloadResp.Content)
	require.NotNil(t, downloadResp.Schema)
	assert.Equal(t, "string", downloadResp.Schema.Type)
	assert.Equal(t, "binary", downloadResp.Schema.Format)
}

// TestOAS2ServerBuilder_EndToEnd verifies the complete OAS 2.0 server builder flow
// with HTTP request/response handling.
func TestOAS2ServerBuilder_EndToEnd(t *testing.T) {
	t.Parallel()

	type Pet struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
		Tag  string `json:"tag,omitempty"`
	}

	// In-memory pet store
	pets := map[int64]*Pet{
		1: {ID: 1, Name: "Fluffy", Tag: "cat"},
		2: {ID: 2, Name: "Spot", Tag: "dog"},
	}

	srv := NewServerBuilder(parser.OASVersion20, WithoutValidation()).
		SetTitle("Pet Store API (OAS 2.0)").
		SetVersion("1.0.0")

	// List pets operation
	srv.AddOperation(http.MethodGet, "/pets",
		WithOperationID("listPets"),
		WithResponse(http.StatusOK, []Pet{}),
	)
	srv.Handle(http.MethodGet, "/pets", func(_ context.Context, _ *Request) Response {
		result := make([]Pet, 0, len(pets))
		for _, p := range pets {
			result = append(result, *p)
		}
		return JSON(http.StatusOK, result)
	})

	// Get pet operation
	srv.AddOperation(http.MethodGet, "/pets/{petId}",
		WithOperationID("getPet"),
		WithPathParam("petId", int64(0)),
		WithResponse(http.StatusOK, Pet{}),
		WithResponse(http.StatusNotFound, struct {
			Error string `json:"error"`
		}{}),
	)
	srv.Handle(http.MethodGet, "/pets/{petId}", func(_ context.Context, req *Request) Response {
		petIDStr := req.PathParams["petId"]
		var petID int64
		switch v := petIDStr.(type) {
		case string:
			// Parse from string
			if _, err := json.Marshal(v); err == nil {
				// Simple string to int conversion for tests
				for id := range pets {
					if string(rune('0'+id)) == v || v == "1" && id == 1 || v == "2" && id == 2 {
						petID = id
						break
					}
				}
			}
		case int64:
			petID = v
		case float64:
			petID = int64(v)
		}

		pet, ok := pets[petID]
		if !ok {
			return Error(http.StatusNotFound, "pet not found")
		}
		return JSON(http.StatusOK, pet)
	})

	// Create pet operation
	srv.AddOperation(http.MethodPost, "/pets",
		WithOperationID("createPet"),
		WithRequestBody("application/json", Pet{}),
		WithResponse(http.StatusCreated, Pet{}),
	)
	srv.Handle(http.MethodPost, "/pets", func(_ context.Context, req *Request) Response {
		var newPet Pet
		if bodyMap, ok := req.Body.(map[string]any); ok {
			if name, ok := bodyMap["name"].(string); ok {
				newPet.Name = name
			}
			if tag, ok := bodyMap["tag"].(string); ok {
				newPet.Tag = tag
			}
		}
		newPet.ID = int64(len(pets) + 1)
		pets[newPet.ID] = &newPet
		return JSON(http.StatusCreated, newPet)
	})

	result, err := srv.BuildServer()
	require.NoError(t, err)
	require.NotNil(t, result.Handler)

	// Verify it's an OAS 2.0 document
	oas2Doc, ok := result.Spec.(*parser.OAS2Document)
	require.True(t, ok, "Expected OAS 2.0 document")
	assert.Equal(t, "2.0", oas2Doc.Swagger)

	// Test list pets
	t.Run("list pets", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/pets", nil)
		result.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")

		var petsList []Pet
		err := json.NewDecoder(rec.Body).Decode(&petsList)
		require.NoError(t, err)
		assert.Len(t, petsList, 2)
	})

	// Test get pet by ID
	t.Run("get pet", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/pets/1", nil)
		result.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var pet Pet
		err := json.NewDecoder(rec.Body).Decode(&pet)
		require.NoError(t, err)
		assert.Equal(t, int64(1), pet.ID)
		assert.Equal(t, "Fluffy", pet.Name)
	})

	// Test get non-existent pet
	t.Run("get non-existent pet", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/pets/999", nil)
		result.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	// Test create pet
	t.Run("create pet", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := NewTestRequest(http.MethodPost, "/pets").
			JSONBody(Pet{Name: "Max", Tag: "dog"}).
			Build()
		result.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)

		var pet Pet
		err := json.NewDecoder(rec.Body).Decode(&pet)
		require.NoError(t, err)
		assert.Equal(t, "Max", pet.Name)
		assert.Equal(t, "dog", pet.Tag)
		assert.Greater(t, pet.ID, int64(0))
	})

	// Test 404 for unknown path
	t.Run("unknown path returns 404", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
		result.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	// Test 405 for wrong method
	t.Run("wrong method returns 405", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodDelete, "/pets", nil)
		result.Handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	})
}
