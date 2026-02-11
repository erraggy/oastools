package builder

import (
	"net/http"
	"testing"

	"github.com/erraggy/oastools/internal/testutil"
	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInlineParamWithConstraints_OAS3 tests inline param helpers with constraints for OAS 3.x.
func TestInlineParamWithConstraints_OAS3(t *testing.T) {
	t.Run("query param with numeric constraints", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/pets",
				WithQueryParam("limit", int32(0),
					WithParamDescription("Max results"),
					WithParamMinimum(1),
					WithParamMaximum(100),
					WithParamDefault(20),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		params := doc.Paths["/pets"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		require.NotNil(t, param.Schema)
		require.NotNil(t, param.Schema.Minimum)
		assert.Equal(t, 1.0, *param.Schema.Minimum)
		require.NotNil(t, param.Schema.Maximum)
		assert.Equal(t, 100.0, *param.Schema.Maximum)
		assert.Equal(t, 20, param.Schema.Default)
	})

	t.Run("query param with string constraints", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/pets",
				WithQueryParam("name", "",
					WithParamMinLength(1),
					WithParamMaxLength(50),
					WithParamPattern("^[a-zA-Z]+$"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		params := doc.Paths["/pets"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		require.NotNil(t, param.Schema)
		require.NotNil(t, param.Schema.MinLength)
		assert.Equal(t, 1, *param.Schema.MinLength)
		require.NotNil(t, param.Schema.MaxLength)
		assert.Equal(t, 50, *param.Schema.MaxLength)
		assert.Equal(t, "^[a-zA-Z]+$", param.Schema.Pattern)
	})

	t.Run("query param with enum", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/pets",
				WithQueryParam("status", "",
					WithParamEnum("available", "pending", "sold"),
					WithParamDefault("available"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		params := doc.Paths["/pets"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		require.NotNil(t, param.Schema)
		require.Len(t, param.Schema.Enum, 3)
		assert.Equal(t, "available", param.Schema.Enum[0])
		assert.Equal(t, "available", param.Schema.Default)
	})

	t.Run("path param with constraints", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/pets/{petId}",
				WithPathParam("petId", int64(0),
					WithParamMinimum(1),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		params := doc.Paths["/pets/{petId}"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		assert.True(t, param.Required)
		require.NotNil(t, param.Schema)
		require.NotNil(t, param.Schema.Minimum)
		assert.Equal(t, 1.0, *param.Schema.Minimum)
	})

	t.Run("header param with constraints", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/pets",
				WithHeaderParam("X-Request-ID", "",
					WithParamPattern("^[a-f0-9-]+$"),
					WithParamRequired(true),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		params := doc.Paths["/pets"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		assert.True(t, param.Required)
		require.NotNil(t, param.Schema)
		assert.Equal(t, "^[a-f0-9-]+$", param.Schema.Pattern)
	})

	t.Run("cookie param with constraints", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/pets",
				WithCookieParam("session_id", "",
					WithParamMinLength(32),
					WithParamMaxLength(64),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		params := doc.Paths["/pets"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		require.NotNil(t, param.Schema)
		require.NotNil(t, param.Schema.MinLength)
		assert.Equal(t, 32, *param.Schema.MinLength)
		require.NotNil(t, param.Schema.MaxLength)
		assert.Equal(t, 64, *param.Schema.MaxLength)
	})
}

// TestInlineParamWithConstraints_OAS2 tests inline param helpers with constraints for OAS 2.0.
func TestInlineParamWithConstraints_OAS2(t *testing.T) {
	t.Run("query param with constraints", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/pets",
				WithQueryParam("limit", int32(0),
					WithParamMinimum(1),
					WithParamMaximum(100),
					WithParamDefault(20),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		params := doc.Paths["/pets"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]

		// OAS 2.0: constraints on parameter, not schema
		require.NotNil(t, param.Minimum)
		assert.Equal(t, 1.0, *param.Minimum)
		require.NotNil(t, param.Maximum)
		assert.Equal(t, 100.0, *param.Maximum)
		assert.Equal(t, 20, param.Default)
	})

	t.Run("query param with enum", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/pets",
				WithQueryParam("status", "",
					WithParamEnum("available", "pending", "sold"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		params := doc.Paths["/pets"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		require.Len(t, param.Enum, 3)
	})
}

// TestInlineParamWithInvalidConstraints tests that invalid constraints in inline params accumulate errors.
func TestInlineParamWithInvalidConstraints(t *testing.T) {
	t.Run("min greater than max in query param", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/test",
				WithQueryParam("bad", int32(0),
					WithParamMinimum(100),
					WithParamMaximum(1),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		_, err := b.BuildOAS3()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "minimum")
	})

	t.Run("invalid pattern in header param", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/test",
				WithHeaderParam("X-Bad", "",
					WithParamPattern("[invalid"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		_, err := b.BuildOAS3()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pattern")
	})
}

// TestExplicitFormat_OAS3 tests explicit format overrides for OAS 3.x.
func TestExplicitFormat_OAS3(t *testing.T) {
	t.Run("path param with uuid format", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/users/{user_id}",
				WithPathParam("user_id", "",
					WithParamFormat("uuid"),
					WithParamDescription("User UUID"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		params := doc.Paths["/users/{user_id}"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		require.NotNil(t, param.Schema)
		assert.Equal(t, "string", param.Schema.Type)
		assert.Equal(t, "uuid", param.Schema.Format)
		assert.Equal(t, "User UUID", param.Description)
	})

	t.Run("query param with email format", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/users",
				WithQueryParam("email", "",
					WithParamFormat("email"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		params := doc.Paths["/users"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		require.NotNil(t, param.Schema)
		assert.Equal(t, "string", param.Schema.Type)
		assert.Equal(t, "email", param.Schema.Format)
	})

	t.Run("query param with date format (overrides time.Time inference)", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/users",
				WithQueryParam("birth_date", "",
					WithParamFormat("date"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		params := doc.Paths["/users"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		require.NotNil(t, param.Schema)
		assert.Equal(t, "string", param.Schema.Type)
		assert.Equal(t, "date", param.Schema.Format)
	})
}

// TestExplicitType_OAS3 tests explicit type overrides for OAS 3.x.
func TestExplicitType_OAS3(t *testing.T) {
	t.Run("byte data type override", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/upload",
				WithQueryParam("data", []byte{},
					WithParamType("string"),
					WithParamFormat("byte"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		params := doc.Paths["/upload"].Post.Parameters
		require.Len(t, params, 1)
		param := params[0]
		require.NotNil(t, param.Schema)
		assert.Equal(t, "string", param.Schema.Type)
		assert.Equal(t, "byte", param.Schema.Format)
	})
}

// TestSchemaOverride_OAS3 tests full schema override for OAS 3.x.
func TestSchemaOverride_OAS3(t *testing.T) {
	t.Run("array schema override", func(t *testing.T) {
		schema := &parser.Schema{
			Type:     "array",
			Items:    &parser.Schema{Type: "string", Format: "uuid"},
			MinItems: testutil.Ptr(1),
			MaxItems: testutil.Ptr(10),
		}

		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/items",
				WithQueryParam("ids", nil,
					WithParamSchema(schema),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		params := doc.Paths["/items"].Get.Parameters
		require.Len(t, params, 1)
		idsParam := params[0]
		require.NotNil(t, idsParam.Schema)
		assert.Equal(t, "array", idsParam.Schema.Type)
		require.NotNil(t, idsParam.Schema.Items)
		items := idsParam.Schema.Items.(*parser.Schema)
		assert.Equal(t, "string", items.Type)
		assert.Equal(t, "uuid", items.Format)
		require.NotNil(t, idsParam.Schema.MinItems)
		assert.Equal(t, 1, *idsParam.Schema.MinItems)
		require.NotNil(t, idsParam.Schema.MaxItems)
		assert.Equal(t, 10, *idsParam.Schema.MaxItems)
	})
}

// TestExplicitFormat_OAS2 tests explicit format overrides for OAS 2.0.
func TestExplicitFormat_OAS2(t *testing.T) {
	t.Run("path param with uuid format", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/users/{user_id}",
				WithPathParam("user_id", "",
					WithParamFormat("uuid"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		params := doc.Paths["/users/{user_id}"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		// OAS 2.0: type/format are top-level parameter fields
		assert.Equal(t, "string", param.Type)
		assert.Equal(t, "uuid", param.Format)
	})

	t.Run("query param with email format", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/users",
				WithQueryParam("email", "",
					WithParamFormat("email"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		params := doc.Paths["/users"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		assert.Equal(t, "string", param.Type)
		assert.Equal(t, "email", param.Format)
	})
}

// TestExplicitType_OAS2 tests explicit type overrides for OAS 2.0.
func TestExplicitType_OAS2(t *testing.T) {
	t.Run("byte data type override", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodPost, "/upload",
				WithQueryParam("data", []byte{},
					WithParamType("string"),
					WithParamFormat("byte"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		params := doc.Paths["/upload"].Post.Parameters
		require.Len(t, params, 1)
		param := params[0]
		assert.Equal(t, "string", param.Type)
		assert.Equal(t, "byte", param.Format)
	})
}

// TestSchemaOverride_OAS2 tests full schema override for OAS 2.0.
func TestSchemaOverride_OAS2(t *testing.T) {
	t.Run("schema override uses type and format", func(t *testing.T) {
		schema := &parser.Schema{
			Type:   "number",
			Format: "decimal",
		}

		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/products",
				WithQueryParam("price", nil,
					WithParamSchema(schema),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		params := doc.Paths["/products"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		assert.Equal(t, "number", param.Type)
		assert.Equal(t, "decimal", param.Format)
	})
}

// TestPrecedenceRules tests the precedence of schema override over type/format overrides.
func TestPrecedenceRules(t *testing.T) {
	t.Run("schema takes precedence over type and format OAS3", func(t *testing.T) {
		schema := &parser.Schema{Type: "number", Format: "decimal"}

		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/test",
				WithQueryParam("amount", int64(0),
					WithParamType("string"),     // Should be ignored
					WithParamFormat("currency"), // Should be ignored
					WithParamSchema(schema),     // Should win
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		params := doc.Paths["/test"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		assert.Equal(t, "number", param.Schema.Type)
		assert.Equal(t, "decimal", param.Schema.Format)
	})

	t.Run("format without type uses inferred type OAS3", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/test",
				WithQueryParam("id", "", // Inferred type: string
					WithParamFormat("uuid"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		params := doc.Paths["/test"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		assert.Equal(t, "string", param.Schema.Type) // Preserved from inference
		assert.Equal(t, "uuid", param.Schema.Format) // From override
	})

	t.Run("format override clears existing format OAS3", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/test",
				WithQueryParam("count", int64(0), // Inferred: integer, int64
					WithParamFormat("int32"), // Override format
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		params := doc.Paths["/test"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		assert.Equal(t, "integer", param.Schema.Type) // Preserved
		assert.Equal(t, "int32", param.Schema.Format) // Overridden
	})
}

// TestCombinedOverridesAndConstraints tests combining overrides with constraints.
func TestCombinedOverridesAndConstraints(t *testing.T) {
	t.Run("format override with pattern constraint OAS3", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/test",
				WithQueryParam("id", "",
					WithParamFormat("uuid"),
					WithParamPattern("^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		params := doc.Paths["/test"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		require.NotNil(t, param.Schema)
		assert.Equal(t, "string", param.Schema.Type)
		assert.Equal(t, "uuid", param.Schema.Format)
		assert.Equal(t, "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", param.Schema.Pattern)
	})

	t.Run("type override with min/max constraints OAS3", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/test",
				WithQueryParam("amount", "",
					WithParamType("number"),
					WithParamFormat("decimal"),
					WithParamMinimum(0),
					WithParamMaximum(1000),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		params := doc.Paths["/test"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		require.NotNil(t, param.Schema)
		assert.Equal(t, "number", param.Schema.Type)
		assert.Equal(t, "decimal", param.Schema.Format)
		require.NotNil(t, param.Schema.Minimum)
		assert.Equal(t, 0.0, *param.Schema.Minimum)
		require.NotNil(t, param.Schema.Maximum)
		assert.Equal(t, 1000.0, *param.Schema.Maximum)
	})
}

// TestTypeOverrideAlone tests type override without format override.
func TestTypeOverrideAlone(t *testing.T) {
	t.Run("type override only OAS3 clears inferred format", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/test",
				WithQueryParam("amount", int32(0), // Inferred: integer, int32
					WithParamType("number"), // Override to number only
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		params := doc.Paths["/test"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		require.NotNil(t, param.Schema)
		assert.Equal(t, "number", param.Schema.Type)
		// Format should be preserved from inference (int32)
		assert.Equal(t, "int32", param.Schema.Format)
	})

	t.Run("type override only OAS2", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddOperation(http.MethodGet, "/test",
				WithQueryParam("amount", int32(0),
					WithParamType("number"),
				),
				WithResponse(http.StatusOK, struct{}{}),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		params := doc.Paths["/test"].Get.Parameters
		require.Len(t, params, 1)
		param := params[0]
		assert.Equal(t, "number", param.Type)
		// Format is preserved from inference (int32 -> int32)
		assert.Equal(t, "int32", param.Format)
	})
}
