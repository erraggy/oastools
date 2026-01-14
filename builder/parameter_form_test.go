package builder

import (
	"net/http"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWithFormParam_OAS2 tests form parameters for OAS 2.0.
func TestWithFormParam_OAS2(t *testing.T) {
	t.Run("single form parameter", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/login",
				WithFormParam("username", "",
					WithParamDescription("User's username"),
					WithParamRequired(true),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		params := doc.Paths["/login"].Post.Parameters
		require.Len(t, params, 1)
		param := params[0]
		assert.Equal(t, "username", param.Name)
		assert.Equal(t, parser.ParamInFormData, param.In)
		assert.True(t, param.Required)
		assert.Equal(t, "User's username", param.Description)
	})

	t.Run("multiple form parameters", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/login",
				WithFormParam("username", "",
					WithParamRequired(true),
				),
				WithFormParam("password", "",
					WithParamRequired(true),
					WithParamMinLength(8),
				),
				WithFormParam("remember", false,
					WithParamDefault(false),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		params := doc.Paths["/login"].Post.Parameters
		require.Len(t, params, 3)

		// Check username
		assert.Equal(t, "username", params[0].Name)
		assert.Equal(t, parser.ParamInFormData, params[0].In)
		assert.True(t, params[0].Required)

		// Check password
		assert.Equal(t, "password", params[1].Name)
		assert.Equal(t, parser.ParamInFormData, params[1].In)
		assert.True(t, params[1].Required)
		require.NotNil(t, params[1].MinLength)
		assert.Equal(t, 8, *params[1].MinLength)

		// Check remember
		assert.Equal(t, "remember", params[2].Name)
		assert.Equal(t, parser.ParamInFormData, params[2].In)
		assert.False(t, params[2].Required)
		assert.Equal(t, false, params[2].Default)
	})

	t.Run("form parameters with constraints", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/submit",
				WithFormParam("age", int32(0),
					WithParamMinimum(18),
					WithParamMaximum(100),
				),
				WithFormParam("email", "",
					WithParamPattern("^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"),
				),
				WithFormParam("status", "",
					WithParamEnum("active", "inactive", "pending"),
					WithParamDefault("pending"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		params := doc.Paths["/submit"].Post.Parameters
		require.Len(t, params, 3)

		// Check age constraints
		ageParam := params[0]
		assert.Equal(t, "age", ageParam.Name)
		require.NotNil(t, ageParam.Minimum)
		assert.Equal(t, 18.0, *ageParam.Minimum)
		require.NotNil(t, ageParam.Maximum)
		assert.Equal(t, 100.0, *ageParam.Maximum)

		// Check email pattern
		emailParam := params[1]
		assert.Equal(t, "email", emailParam.Name)
		assert.Equal(t, "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$", emailParam.Pattern)

		// Check status enum
		statusParam := params[2]
		assert.Equal(t, "status", statusParam.Name)
		require.Len(t, statusParam.Enum, 3)
		assert.Equal(t, "active", statusParam.Enum[0])
		assert.Equal(t, "inactive", statusParam.Enum[1])
		assert.Equal(t, "pending", statusParam.Enum[2])
		assert.Equal(t, "pending", statusParam.Default)
	})

	t.Run("form parameters mixed with other parameters", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/items/{id}",
				WithPathParam("id", int64(0),
					WithParamDescription("Item ID"),
				),
				WithQueryParam("format", "",
					WithParamEnum("json", "xml"),
					WithParamDefault("json"),
				),
				WithFormParam("name", "",
					WithParamRequired(true),
				),
				WithFormParam("description", "",
					WithParamMaxLength(500),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		params := doc.Paths["/items/{id}"].Post.Parameters
		require.Len(t, params, 4)

		// Verify parameter types
		assert.Equal(t, parser.ParamInPath, params[0].In)
		assert.Equal(t, parser.ParamInQuery, params[1].In)
		assert.Equal(t, parser.ParamInFormData, params[2].In)
		assert.Equal(t, parser.ParamInFormData, params[3].In)
	})
}

// TestWithFormParam_OAS3 tests form parameters for OAS 3.x.
func TestWithFormParam_OAS3(t *testing.T) {
	t.Run("single form parameter", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/login",
				WithFormParam("username", "",
					WithParamDescription("User's username"),
					WithParamRequired(true),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		// Check request body exists with form-urlencoded content
		rb := doc.Paths["/login"].Post.RequestBody
		require.NotNil(t, rb)
		require.Contains(t, rb.Content, "application/x-www-form-urlencoded")

		mediaType := rb.Content["application/x-www-form-urlencoded"]
		require.NotNil(t, mediaType.Schema)
		assert.Equal(t, "object", mediaType.Schema.Type)

		// Check properties
		require.Contains(t, mediaType.Schema.Properties, "username")
		usernameProp := mediaType.Schema.Properties["username"]
		assert.Equal(t, "User's username", usernameProp.Description)

		// Check required fields
		require.Len(t, mediaType.Schema.Required, 1)
		assert.Equal(t, "username", mediaType.Schema.Required[0])
	})

	t.Run("multiple form parameters", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/login",
				WithFormParam("username", "",
					WithParamRequired(true),
					WithParamMinLength(3),
				),
				WithFormParam("password", "",
					WithParamRequired(true),
					WithParamMinLength(8),
				),
				WithFormParam("remember", false,
					WithParamDefault(false),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		rb := doc.Paths["/login"].Post.RequestBody
		require.NotNil(t, rb)
		mediaType := rb.Content["application/x-www-form-urlencoded"]
		require.NotNil(t, mediaType.Schema)

		// Check all properties exist
		require.Contains(t, mediaType.Schema.Properties, "username")
		require.Contains(t, mediaType.Schema.Properties, "password")
		require.Contains(t, mediaType.Schema.Properties, "remember")

		// Check username constraints
		usernameProp := mediaType.Schema.Properties["username"]
		require.NotNil(t, usernameProp.MinLength)
		assert.Equal(t, 3, *usernameProp.MinLength)

		// Check password constraints
		passwordProp := mediaType.Schema.Properties["password"]
		require.NotNil(t, passwordProp.MinLength)
		assert.Equal(t, 8, *passwordProp.MinLength)

		// Check remember default
		rememberProp := mediaType.Schema.Properties["remember"]
		assert.Equal(t, false, rememberProp.Default)

		// Check required fields
		require.Len(t, mediaType.Schema.Required, 2)
		assert.Contains(t, mediaType.Schema.Required, "username")
		assert.Contains(t, mediaType.Schema.Required, "password")
	})

	t.Run("form parameters with constraints", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/submit",
				WithFormParam("age", int32(0),
					WithParamMinimum(18),
					WithParamMaximum(100),
				),
				WithFormParam("email", "",
					WithParamPattern("^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"),
				),
				WithFormParam("status", "",
					WithParamEnum("active", "inactive", "pending"),
					WithParamDefault("pending"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		mediaType := doc.Paths["/submit"].Post.RequestBody.Content["application/x-www-form-urlencoded"]
		require.NotNil(t, mediaType.Schema)

		// Check age constraints
		ageProp := mediaType.Schema.Properties["age"]
		require.NotNil(t, ageProp.Minimum)
		assert.Equal(t, 18.0, *ageProp.Minimum)
		require.NotNil(t, ageProp.Maximum)
		assert.Equal(t, 100.0, *ageProp.Maximum)

		// Check email pattern
		emailProp := mediaType.Schema.Properties["email"]
		assert.Equal(t, "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$", emailProp.Pattern)

		// Check status enum
		statusProp := mediaType.Schema.Properties["status"]
		require.Len(t, statusProp.Enum, 3)
		assert.Equal(t, "active", statusProp.Enum[0])
		assert.Equal(t, "inactive", statusProp.Enum[1])
		assert.Equal(t, "pending", statusProp.Enum[2])
		assert.Equal(t, "pending", statusProp.Default)
	})

	t.Run("form parameters mixed with other parameters", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/items/{id}",
				WithPathParam("id", int64(0),
					WithParamDescription("Item ID"),
				),
				WithQueryParam("format", "",
					WithParamEnum("json", "xml"),
					WithParamDefault("json"),
				),
				WithFormParam("name", "",
					WithParamRequired(true),
				),
				WithFormParam("description", "",
					WithParamMaxLength(500),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		// Check regular parameters
		params := doc.Paths["/items/{id}"].Post.Parameters
		require.Len(t, params, 2)
		assert.Equal(t, parser.ParamInPath, params[0].In)
		assert.Equal(t, parser.ParamInQuery, params[1].In)

		// Check form parameters in request body
		rb := doc.Paths["/items/{id}"].Post.RequestBody
		require.NotNil(t, rb)
		mediaType := rb.Content["application/x-www-form-urlencoded"]
		require.NotNil(t, mediaType.Schema)
		require.Contains(t, mediaType.Schema.Properties, "name")
		require.Contains(t, mediaType.Schema.Properties, "description")
	})

	t.Run("form parameters with existing request body", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/upload",
				WithRequestBody("application/json", struct {
					Metadata string `json:"metadata"`
				}{},
					WithRequired(true),
				),
				WithFormParam("file", "",
					WithParamRequired(true),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		rb := doc.Paths["/upload"].Post.RequestBody
		require.NotNil(t, rb)

		// Both content types should exist
		require.Contains(t, rb.Content, "application/json")
		require.Contains(t, rb.Content, "application/x-www-form-urlencoded")

		// Check form parameters
		formMediaType := rb.Content["application/x-www-form-urlencoded"]
		require.NotNil(t, formMediaType.Schema)
		require.Contains(t, formMediaType.Schema.Properties, "file")
	})
}

// TestWithFormParam_OAS31 tests form parameters for OAS 3.1.
func TestWithFormParam_OAS31(t *testing.T) {
	t.Run("form parameters in OAS 3.1", func(t *testing.T) {
		b := New(parser.OASVersion310).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/register",
				WithFormParam("username", "",
					WithParamRequired(true),
					WithParamMinLength(3),
					WithParamMaxLength(20),
				),
				WithFormParam("email", "",
					WithParamRequired(true),
					WithParamPattern("^[^@]+@[^@]+\\.[^@]+$"),
				),
				WithFormParam("age", int32(0),
					WithParamMinimum(13),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		rb := doc.Paths["/register"].Post.RequestBody
		require.NotNil(t, rb)
		mediaType := rb.Content["application/x-www-form-urlencoded"]
		require.NotNil(t, mediaType.Schema)

		// Verify properties
		require.Contains(t, mediaType.Schema.Properties, "username")
		require.Contains(t, mediaType.Schema.Properties, "email")
		require.Contains(t, mediaType.Schema.Properties, "age")

		// Verify required fields
		require.Len(t, mediaType.Schema.Required, 2)
		assert.Contains(t, mediaType.Schema.Required, "username")
		assert.Contains(t, mediaType.Schema.Required, "email")
	})
}

// TestWithFormParam_Webhooks tests form parameters in webhooks.
func TestWithFormParam_Webhooks(t *testing.T) {
	t.Run("form parameters in webhooks", func(t *testing.T) {
		b := New(parser.OASVersion310).
			SetTitle("Webhook API").
			SetVersion("1.0.0").
			AddWebhook("user-created", http.MethodPost,
				WithFormParam("user_id", int64(0),
					WithParamRequired(true),
				),
				WithFormParam("username", "",
					WithParamRequired(true),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		require.NotNil(t, doc.Webhooks)
		require.Contains(t, doc.Webhooks, "user-created")
		webhook := doc.Webhooks["user-created"]
		require.NotNil(t, webhook.Post)

		rb := webhook.Post.RequestBody
		require.NotNil(t, rb)
		mediaType := rb.Content["application/x-www-form-urlencoded"]
		require.NotNil(t, mediaType.Schema)

		// Check properties
		require.Contains(t, mediaType.Schema.Properties, "user_id")
		require.Contains(t, mediaType.Schema.Properties, "username")

		// Check required
		require.Len(t, mediaType.Schema.Required, 2)
	})
}

// TestWithFormParam_InvalidConstraints tests error handling for invalid constraints.
func TestWithFormParam_InvalidConstraints(t *testing.T) {
	t.Run("invalid constraint in form parameter OAS 2.0", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/test",
				WithFormParam("bad", int32(0),
					WithParamMinimum(100),
					WithParamMaximum(1),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		_, err := b.BuildOAS2()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "minimum")
	})

	t.Run("invalid constraint in form parameter OAS 3.x", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/test",
				WithFormParam("bad", "",
					WithParamPattern("[invalid"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		_, err := b.BuildOAS3()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pattern")
	})
}

// TestWithFormParam_EmptyName tests form parameters with empty names.
func TestWithFormParam_EmptyName(t *testing.T) {
	t.Run("empty form parameter name OAS 2.0", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/test",
				WithFormParam("", ""),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		// Parameter should still be added (validation is separate concern)
		params := doc.Paths["/test"].Post.Parameters
		require.Len(t, params, 1)
		assert.Equal(t, "", params[0].Name)
	})

	t.Run("empty form parameter name OAS 3.x", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/test",
				WithFormParam("", ""),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		// Property should still be added (validation is separate concern)
		mediaType := doc.Paths["/test"].Post.RequestBody.Content["application/x-www-form-urlencoded"]
		require.Contains(t, mediaType.Schema.Properties, "")
	})
}

// TestWithFormParam_DeprecatedField tests deprecated form parameters.
func TestWithFormParam_DeprecatedField(t *testing.T) {
	t.Run("deprecated form parameter OAS 2.0", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/test",
				WithFormParam("old_field", "",
					WithParamDeprecated(true),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		params := doc.Paths["/test"].Post.Parameters
		require.Len(t, params, 1)
		assert.True(t, params[0].Deprecated)
	})

	t.Run("deprecated form parameter OAS 3.x", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/test",
				WithFormParam("old_field", "",
					WithParamDeprecated(true),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		mediaType := doc.Paths["/test"].Post.RequestBody.Content["application/x-www-form-urlencoded"]
		oldFieldProp := mediaType.Schema.Properties["old_field"]
		assert.True(t, oldFieldProp.Deprecated)
	})
}

// TestFormParamWithTypeFormatOverride tests form parameters with type/format overrides.
func TestFormParamWithTypeFormatOverride(t *testing.T) {
	t.Run("OAS3 form param with format override", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/register",
				WithFormParam("user_id", "",
					WithParamFormat("uuid"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		reqBody := doc.Paths["/register"].Post.RequestBody
		require.NotNil(t, reqBody)
		mediaType := reqBody.Content["application/x-www-form-urlencoded"]
		require.NotNil(t, mediaType)
		require.NotNil(t, mediaType.Schema)
		require.NotNil(t, mediaType.Schema.Properties)
		userIdProp := mediaType.Schema.Properties["user_id"]
		require.NotNil(t, userIdProp)
		assert.Equal(t, "string", userIdProp.Type)
		assert.Equal(t, "uuid", userIdProp.Format)
	})

	t.Run("OAS3 form param with type and format override", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/upload",
				WithFormParam("file_size", int32(0),
					WithParamType("integer"),
					WithParamFormat("int64"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		reqBody := doc.Paths["/upload"].Post.RequestBody
		require.NotNil(t, reqBody)
		mediaType := reqBody.Content["application/x-www-form-urlencoded"]
		require.NotNil(t, mediaType)
		fileSizeProp := mediaType.Schema.Properties["file_size"]
		require.NotNil(t, fileSizeProp)
		assert.Equal(t, "integer", fileSizeProp.Type)
		assert.Equal(t, "int64", fileSizeProp.Format)
	})

	t.Run("OAS2 form param with format override", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/register",
				WithFormParam("email", "",
					WithParamFormat("email"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		params := doc.Paths["/register"].Post.Parameters
		require.NotEmpty(t, params)

		var emailParam *parser.Parameter
		for _, p := range params {
			if p.Name == "email" {
				emailParam = p
				break
			}
		}
		require.NotNil(t, emailParam)
		assert.Equal(t, "formData", emailParam.In)
		assert.Equal(t, "string", emailParam.Type)
		assert.Equal(t, "email", emailParam.Format)
	})
}
