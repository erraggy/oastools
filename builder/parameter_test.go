package builder

import (
	"net/http"
	"testing"

	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWithParamMinimum tests the WithParamMinimum option.
func TestWithParamMinimum(t *testing.T) {
	cfg := &paramConfig{}
	WithParamMinimum(1.5)(cfg)
	require.NotNil(t, cfg.minimum)
	assert.Equal(t, 1.5, *cfg.minimum)
}

// TestWithParamMaximum tests the WithParamMaximum option.
func TestWithParamMaximum(t *testing.T) {
	cfg := &paramConfig{}
	WithParamMaximum(100.0)(cfg)
	require.NotNil(t, cfg.maximum)
	assert.Equal(t, 100.0, *cfg.maximum)
}

// TestWithParamExclusiveMinimum tests the WithParamExclusiveMinimum option.
func TestWithParamExclusiveMinimum(t *testing.T) {
	cfg := &paramConfig{}
	WithParamExclusiveMinimum(true)(cfg)
	assert.True(t, cfg.exclusiveMinimum)
}

// TestWithParamExclusiveMaximum tests the WithParamExclusiveMaximum option.
func TestWithParamExclusiveMaximum(t *testing.T) {
	cfg := &paramConfig{}
	WithParamExclusiveMaximum(true)(cfg)
	assert.True(t, cfg.exclusiveMaximum)
}

// TestWithParamMultipleOf tests the WithParamMultipleOf option.
func TestWithParamMultipleOf(t *testing.T) {
	cfg := &paramConfig{}
	WithParamMultipleOf(5.0)(cfg)
	require.NotNil(t, cfg.multipleOf)
	assert.Equal(t, 5.0, *cfg.multipleOf)
}

// TestWithParamMinLength tests the WithParamMinLength option.
func TestWithParamMinLength(t *testing.T) {
	cfg := &paramConfig{}
	WithParamMinLength(1)(cfg)
	require.NotNil(t, cfg.minLength)
	assert.Equal(t, 1, *cfg.minLength)
}

// TestWithParamMaxLength tests the WithParamMaxLength option.
func TestWithParamMaxLength(t *testing.T) {
	cfg := &paramConfig{}
	WithParamMaxLength(100)(cfg)
	require.NotNil(t, cfg.maxLength)
	assert.Equal(t, 100, *cfg.maxLength)
}

// TestWithParamPattern tests the WithParamPattern option.
func TestWithParamPattern(t *testing.T) {
	cfg := &paramConfig{}
	WithParamPattern("^[a-zA-Z]+$")(cfg)
	assert.Equal(t, "^[a-zA-Z]+$", cfg.pattern)
}

// TestWithParamMinItems tests the WithParamMinItems option.
func TestWithParamMinItems(t *testing.T) {
	cfg := &paramConfig{}
	WithParamMinItems(1)(cfg)
	require.NotNil(t, cfg.minItems)
	assert.Equal(t, 1, *cfg.minItems)
}

// TestWithParamMaxItems tests the WithParamMaxItems option.
func TestWithParamMaxItems(t *testing.T) {
	cfg := &paramConfig{}
	WithParamMaxItems(10)(cfg)
	require.NotNil(t, cfg.maxItems)
	assert.Equal(t, 10, *cfg.maxItems)
}

// TestWithParamUniqueItems tests the WithParamUniqueItems option.
func TestWithParamUniqueItems(t *testing.T) {
	cfg := &paramConfig{}
	WithParamUniqueItems(true)(cfg)
	assert.True(t, cfg.uniqueItems)
}

// TestWithParamEnum tests the WithParamEnum option.
func TestWithParamEnum(t *testing.T) {
	cfg := &paramConfig{}
	WithParamEnum("available", "pending", "sold")(cfg)
	require.Len(t, cfg.enum, 3)
	assert.Equal(t, "available", cfg.enum[0])
	assert.Equal(t, "pending", cfg.enum[1])
	assert.Equal(t, "sold", cfg.enum[2])
}

// TestWithParamDefault tests the WithParamDefault option.
func TestWithParamDefault(t *testing.T) {
	cfg := &paramConfig{}
	WithParamDefault(20)(cfg)
	assert.Equal(t, 20, cfg.defaultValue)
}

// TestHasParamConstraints tests the hasParamConstraints helper.
func TestHasParamConstraints(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *paramConfig
		expected bool
	}{
		{
			name:     "no constraints",
			cfg:      &paramConfig{},
			expected: false,
		},
		{
			name:     "with minimum",
			cfg:      &paramConfig{minimum: ptrFloat64(1.0)},
			expected: true,
		},
		{
			name:     "with maximum",
			cfg:      &paramConfig{maximum: ptrFloat64(100.0)},
			expected: true,
		},
		{
			name:     "with exclusiveMinimum",
			cfg:      &paramConfig{exclusiveMinimum: true},
			expected: true,
		},
		{
			name:     "with exclusiveMaximum",
			cfg:      &paramConfig{exclusiveMaximum: true},
			expected: true,
		},
		{
			name:     "with multipleOf",
			cfg:      &paramConfig{multipleOf: ptrFloat64(5.0)},
			expected: true,
		},
		{
			name:     "with minLength",
			cfg:      &paramConfig{minLength: ptrInt(1)},
			expected: true,
		},
		{
			name:     "with maxLength",
			cfg:      &paramConfig{maxLength: ptrInt(100)},
			expected: true,
		},
		{
			name:     "with pattern",
			cfg:      &paramConfig{pattern: "^[a-z]+$"},
			expected: true,
		},
		{
			name:     "with minItems",
			cfg:      &paramConfig{minItems: ptrInt(1)},
			expected: true,
		},
		{
			name:     "with maxItems",
			cfg:      &paramConfig{maxItems: ptrInt(10)},
			expected: true,
		},
		{
			name:     "with uniqueItems",
			cfg:      &paramConfig{uniqueItems: true},
			expected: true,
		},
		{
			name:     "with enum",
			cfg:      &paramConfig{enum: []any{"a", "b"}},
			expected: true,
		},
		{
			name:     "with default",
			cfg:      &paramConfig{defaultValue: 10},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, hasParamConstraints(tt.cfg))
		})
	}
}

// TestApplyParamConstraintsToSchema tests the applyParamConstraintsToSchema helper.
func TestApplyParamConstraintsToSchema(t *testing.T) {
	t.Run("nil schema", func(t *testing.T) {
		result := applyParamConstraintsToSchema(nil, &paramConfig{minimum: ptrFloat64(1.0)})
		assert.Nil(t, result)
	})

	t.Run("no constraints", func(t *testing.T) {
		schema := &parser.Schema{Type: "integer"}
		result := applyParamConstraintsToSchema(schema, &paramConfig{})
		// Should return the same schema (no copy needed)
		assert.Same(t, schema, result)
	})

	t.Run("all numeric constraints", func(t *testing.T) {
		schema := &parser.Schema{Type: "integer"}
		cfg := &paramConfig{
			minimum:          ptrFloat64(1.0),
			maximum:          ptrFloat64(100.0),
			exclusiveMinimum: true,
			exclusiveMaximum: true,
			multipleOf:       ptrFloat64(5.0),
		}
		result := applyParamConstraintsToSchema(schema, cfg)
		require.NotNil(t, result.Minimum)
		assert.Equal(t, 1.0, *result.Minimum)
		require.NotNil(t, result.Maximum)
		assert.Equal(t, 100.0, *result.Maximum)
		assert.True(t, result.ExclusiveMinimum.(bool))
		assert.True(t, result.ExclusiveMaximum.(bool))
		require.NotNil(t, result.MultipleOf)
		assert.Equal(t, 5.0, *result.MultipleOf)
	})

	t.Run("all string constraints", func(t *testing.T) {
		schema := &parser.Schema{Type: "string"}
		cfg := &paramConfig{
			minLength: ptrInt(1),
			maxLength: ptrInt(50),
			pattern:   "^[a-zA-Z]+$",
		}
		result := applyParamConstraintsToSchema(schema, cfg)
		require.NotNil(t, result.MinLength)
		assert.Equal(t, 1, *result.MinLength)
		require.NotNil(t, result.MaxLength)
		assert.Equal(t, 50, *result.MaxLength)
		assert.Equal(t, "^[a-zA-Z]+$", result.Pattern)
	})

	t.Run("all array constraints", func(t *testing.T) {
		schema := &parser.Schema{Type: "array"}
		cfg := &paramConfig{
			minItems:    ptrInt(1),
			maxItems:    ptrInt(10),
			uniqueItems: true,
		}
		result := applyParamConstraintsToSchema(schema, cfg)
		require.NotNil(t, result.MinItems)
		assert.Equal(t, 1, *result.MinItems)
		require.NotNil(t, result.MaxItems)
		assert.Equal(t, 10, *result.MaxItems)
		assert.True(t, result.UniqueItems)
	})

	t.Run("enum and default", func(t *testing.T) {
		schema := &parser.Schema{Type: "string"}
		cfg := &paramConfig{
			enum:         []any{"a", "b", "c"},
			defaultValue: "a",
		}
		result := applyParamConstraintsToSchema(schema, cfg)
		require.Len(t, result.Enum, 3)
		assert.Equal(t, "a", result.Enum[0])
		assert.Equal(t, "a", result.Default)
	})
}

// TestApplyParamConstraintsToParam tests the applyParamConstraintsToParam helper.
func TestApplyParamConstraintsToParam(t *testing.T) {
	t.Run("all constraints", func(t *testing.T) {
		param := &parser.Parameter{}
		cfg := &paramConfig{
			minimum:          ptrFloat64(1.0),
			maximum:          ptrFloat64(100.0),
			exclusiveMinimum: true,
			exclusiveMaximum: true,
			multipleOf:       ptrFloat64(5.0),
			minLength:        ptrInt(1),
			maxLength:        ptrInt(50),
			pattern:          "^[a-z]+$",
			minItems:         ptrInt(1),
			maxItems:         ptrInt(10),
			uniqueItems:      true,
			enum:             []any{"a", "b"},
			defaultValue:     "a",
		}
		applyParamConstraintsToParam(param, cfg)

		require.NotNil(t, param.Minimum)
		assert.Equal(t, 1.0, *param.Minimum)
		require.NotNil(t, param.Maximum)
		assert.Equal(t, 100.0, *param.Maximum)
		assert.True(t, param.ExclusiveMinimum)
		assert.True(t, param.ExclusiveMaximum)
		require.NotNil(t, param.MultipleOf)
		assert.Equal(t, 5.0, *param.MultipleOf)
		require.NotNil(t, param.MinLength)
		assert.Equal(t, 1, *param.MinLength)
		require.NotNil(t, param.MaxLength)
		assert.Equal(t, 50, *param.MaxLength)
		assert.Equal(t, "^[a-z]+$", param.Pattern)
		require.NotNil(t, param.MinItems)
		assert.Equal(t, 1, *param.MinItems)
		require.NotNil(t, param.MaxItems)
		assert.Equal(t, 10, *param.MaxItems)
		assert.True(t, param.UniqueItems)
		require.Len(t, param.Enum, 2)
		assert.Equal(t, "a", param.Enum[0])
		assert.Equal(t, "a", param.Default)
	})
}

// TestAddParameterWithConstraints_OAS3 tests AddParameter with constraints for OAS 3.x.
func TestAddParameterWithConstraints_OAS3(t *testing.T) {
	t.Run("numeric constraints", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddParameter("LimitParam", "query", "limit", int32(0),
				WithParamDescription("Max results"),
				WithParamMinimum(1),
				WithParamMaximum(100),
				WithParamDefault(20),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		require.NotNil(t, doc.Components.Parameters)
		param := doc.Components.Parameters["LimitParam"]
		require.NotNil(t, param)
		require.NotNil(t, param.Schema)

		// Constraints should be on schema for OAS 3.x
		require.NotNil(t, param.Schema.Minimum)
		assert.Equal(t, 1.0, *param.Schema.Minimum)
		require.NotNil(t, param.Schema.Maximum)
		assert.Equal(t, 100.0, *param.Schema.Maximum)
		assert.Equal(t, 20, param.Schema.Default)
	})

	t.Run("string constraints", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddParameter("NameParam", "query", "name", "",
				WithParamMinLength(1),
				WithParamMaxLength(50),
				WithParamPattern("^[a-zA-Z]+$"),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		param := doc.Components.Parameters["NameParam"]
		require.NotNil(t, param.Schema)
		require.NotNil(t, param.Schema.MinLength)
		assert.Equal(t, 1, *param.Schema.MinLength)
		require.NotNil(t, param.Schema.MaxLength)
		assert.Equal(t, 50, *param.Schema.MaxLength)
		assert.Equal(t, "^[a-zA-Z]+$", param.Schema.Pattern)
	})

	t.Run("enum constraint", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddParameter("StatusParam", "query", "status", "",
				WithParamEnum("available", "pending", "sold"),
				WithParamDefault("available"),
			)

		doc, err := b.BuildOAS3()
		require.NoError(t, err)

		param := doc.Components.Parameters["StatusParam"]
		require.NotNil(t, param.Schema)
		require.Len(t, param.Schema.Enum, 3)
		assert.Equal(t, "available", param.Schema.Enum[0])
		assert.Equal(t, "pending", param.Schema.Enum[1])
		assert.Equal(t, "sold", param.Schema.Enum[2])
		assert.Equal(t, "available", param.Schema.Default)
	})
}

// TestAddParameterWithConstraints_OAS2 tests AddParameter with constraints for OAS 2.0.
func TestAddParameterWithConstraints_OAS2(t *testing.T) {
	t.Run("numeric constraints", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddParameter("LimitParam", "query", "limit", int32(0),
				WithParamDescription("Max results"),
				WithParamMinimum(1),
				WithParamMaximum(100),
				WithParamDefault(20),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		require.NotNil(t, doc.Parameters)
		param := doc.Parameters["LimitParam"]
		require.NotNil(t, param)

		// Constraints should be directly on parameter for OAS 2.0
		require.NotNil(t, param.Minimum)
		assert.Equal(t, 1.0, *param.Minimum)
		require.NotNil(t, param.Maximum)
		assert.Equal(t, 100.0, *param.Maximum)
		assert.Equal(t, 20, param.Default)
	})

	t.Run("string constraints", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddParameter("NameParam", "query", "name", "",
				WithParamMinLength(1),
				WithParamMaxLength(50),
				WithParamPattern("^[a-zA-Z]+$"),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		param := doc.Parameters["NameParam"]
		require.NotNil(t, param.MinLength)
		assert.Equal(t, 1, *param.MinLength)
		require.NotNil(t, param.MaxLength)
		assert.Equal(t, 50, *param.MaxLength)
		assert.Equal(t, "^[a-zA-Z]+$", param.Pattern)
	})

	t.Run("enum constraint", func(t *testing.T) {
		b := New(parser.OASVersion20).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddParameter("StatusParam", "query", "status", "",
				WithParamEnum("available", "pending", "sold"),
				WithParamDefault("available"),
			)

		doc, err := b.BuildOAS2()
		require.NoError(t, err)

		param := doc.Parameters["StatusParam"]
		require.Len(t, param.Enum, 3)
		assert.Equal(t, "available", param.Enum[0])
		assert.Equal(t, "pending", param.Enum[1])
		assert.Equal(t, "sold", param.Enum[2])
		assert.Equal(t, "available", param.Default)
	})
}

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

// TestWebhookParamWithConstraints tests webhooks with parameter constraints.
func TestWebhookParamWithConstraints(t *testing.T) {
	b := New(parser.OASVersion310).
		SetTitle("Webhook API").
		SetVersion("1.0.0").
		AddWebhook("events", http.MethodPost,
			WithQueryParam("limit", int32(0),
				WithParamMinimum(1),
				WithParamMaximum(100),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	require.NotNil(t, doc.Webhooks)
	require.Contains(t, doc.Webhooks, "events")
	params := doc.Webhooks["events"].Post.Parameters
	require.Len(t, params, 1)
	param := params[0]
	require.NotNil(t, param.Schema)
	require.NotNil(t, param.Schema.Minimum)
	assert.Equal(t, 1.0, *param.Schema.Minimum)
	require.NotNil(t, param.Schema.Maximum)
	assert.Equal(t, 100.0, *param.Schema.Maximum)
}

// TestCombinedConstraints tests combining multiple constraint types.
func TestCombinedConstraints(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0").
		AddOperation(http.MethodGet, "/search",
			WithQueryParam("q", "",
				WithParamDescription("Search query"),
				WithParamRequired(true),
				WithParamMinLength(1),
				WithParamMaxLength(100),
				WithParamPattern("^[a-zA-Z0-9\\s]+$"),
				WithParamExample("test query"),
			),
			WithQueryParam("page", int32(0),
				WithParamMinimum(1),
				WithParamDefault(1),
			),
			WithQueryParam("size", int32(0),
				WithParamMinimum(1),
				WithParamMaximum(100),
				WithParamMultipleOf(10),
				WithParamDefault(10),
			),
			WithResponse(http.StatusOK, struct{}{}),
		)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	params := doc.Paths["/search"].Get.Parameters
	require.Len(t, params, 3)

	// Check "q" parameter
	qParam := params[0]
	assert.Equal(t, "q", qParam.Name)
	assert.True(t, qParam.Required)
	require.NotNil(t, qParam.Schema)
	require.NotNil(t, qParam.Schema.MinLength)
	assert.Equal(t, 1, *qParam.Schema.MinLength)
	require.NotNil(t, qParam.Schema.MaxLength)
	assert.Equal(t, 100, *qParam.Schema.MaxLength)
	assert.Equal(t, "^[a-zA-Z0-9\\s]+$", qParam.Schema.Pattern)

	// Check "page" parameter
	pageParam := params[1]
	assert.Equal(t, "page", pageParam.Name)
	require.NotNil(t, pageParam.Schema.Minimum)
	assert.Equal(t, 1.0, *pageParam.Schema.Minimum)
	assert.Equal(t, 1, pageParam.Schema.Default)

	// Check "size" parameter
	sizeParam := params[2]
	assert.Equal(t, "size", sizeParam.Name)
	require.NotNil(t, sizeParam.Schema.Minimum)
	assert.Equal(t, 1.0, *sizeParam.Schema.Minimum)
	require.NotNil(t, sizeParam.Schema.Maximum)
	assert.Equal(t, 100.0, *sizeParam.Schema.Maximum)
	require.NotNil(t, sizeParam.Schema.MultipleOf)
	assert.Equal(t, 10.0, *sizeParam.Schema.MultipleOf)
	assert.Equal(t, 10, sizeParam.Schema.Default)
}

// TestValidateParamConstraints tests the validateParamConstraints helper.
func TestValidateParamConstraints(t *testing.T) {
	t.Run("valid constraints", func(t *testing.T) {
		cfg := &paramConfig{
			minimum:    ptrFloat64(1.0),
			maximum:    ptrFloat64(100.0),
			minLength:  ptrInt(1),
			maxLength:  ptrInt(50),
			minItems:   ptrInt(0),
			maxItems:   ptrInt(10),
			pattern:    "^[a-z]+$",
			multipleOf: ptrFloat64(5.0),
		}
		err := validateParamConstraints(cfg)
		assert.NoError(t, err)
	})

	t.Run("minimum greater than maximum", func(t *testing.T) {
		cfg := &paramConfig{
			minimum: ptrFloat64(100.0),
			maximum: ptrFloat64(1.0),
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "minimum")
		assert.Contains(t, err.Error(), "maximum")
	})

	t.Run("minLength greater than maxLength", func(t *testing.T) {
		cfg := &paramConfig{
			minLength: ptrInt(100),
			maxLength: ptrInt(10),
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "minLength")
		assert.Contains(t, err.Error(), "maxLength")
	})

	t.Run("negative minLength", func(t *testing.T) {
		cfg := &paramConfig{
			minLength: ptrInt(-1),
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "minLength")
		assert.Contains(t, err.Error(), "negative")
	})

	t.Run("negative maxLength", func(t *testing.T) {
		cfg := &paramConfig{
			maxLength: ptrInt(-1),
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maxLength")
		assert.Contains(t, err.Error(), "negative")
	})

	t.Run("minItems greater than maxItems", func(t *testing.T) {
		cfg := &paramConfig{
			minItems: ptrInt(10),
			maxItems: ptrInt(1),
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "minItems")
		assert.Contains(t, err.Error(), "maxItems")
	})

	t.Run("negative minItems", func(t *testing.T) {
		cfg := &paramConfig{
			minItems: ptrInt(-1),
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "minItems")
		assert.Contains(t, err.Error(), "negative")
	})

	t.Run("negative maxItems", func(t *testing.T) {
		cfg := &paramConfig{
			maxItems: ptrInt(-1),
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maxItems")
		assert.Contains(t, err.Error(), "negative")
	})

	t.Run("zero multipleOf", func(t *testing.T) {
		cfg := &paramConfig{
			multipleOf: ptrFloat64(0),
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "multipleOf")
		assert.Contains(t, err.Error(), "greater than 0")
	})

	t.Run("negative multipleOf", func(t *testing.T) {
		cfg := &paramConfig{
			multipleOf: ptrFloat64(-5),
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "multipleOf")
		assert.Contains(t, err.Error(), "greater than 0")
	})

	t.Run("invalid regex pattern", func(t *testing.T) {
		cfg := &paramConfig{
			pattern: "[invalid",
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pattern")
		assert.Contains(t, err.Error(), "invalid regex")
	})

	t.Run("valid empty constraints", func(t *testing.T) {
		cfg := &paramConfig{}
		err := validateParamConstraints(cfg)
		assert.NoError(t, err)
	})

	t.Run("nil config fields are valid", func(t *testing.T) {
		cfg := &paramConfig{
			minimum: ptrFloat64(1.0), // Only minimum set, no maximum
		}
		err := validateParamConstraints(cfg)
		assert.NoError(t, err)
	})

	t.Run("multiple errors joined", func(t *testing.T) {
		cfg := &paramConfig{
			minimum:    ptrFloat64(100.0),
			maximum:    ptrFloat64(1.0), // min > max
			minLength:  ptrInt(-1),      // negative
			maxLength:  ptrInt(-2),      // negative
			multipleOf: ptrFloat64(0),   // not positive
			pattern:    "[invalid",      // invalid regex
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		// All errors should be present in the joined error
		errStr := err.Error()
		assert.Contains(t, errStr, "minimum")
		assert.Contains(t, errStr, "maximum")
		assert.Contains(t, errStr, "minLength")
		assert.Contains(t, errStr, "maxLength")
		assert.Contains(t, errStr, "multipleOf")
		assert.Contains(t, errStr, "pattern")
	})
}

// TestConstraintError tests the ConstraintError type.
func TestConstraintError(t *testing.T) {
	err := &ConstraintError{
		Field:   "minimum",
		Message: "test message",
	}
	assert.Equal(t, "constraint error on minimum: test message", err.Error())
}

// TestAddParameterWithInvalidConstraints tests that invalid constraints accumulate errors.
func TestAddParameterWithInvalidConstraints(t *testing.T) {
	t.Run("min greater than max", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddParameter("BadParam", "query", "bad", int32(0),
				WithParamMinimum(100),
				WithParamMaximum(1),
			)

		_, err := b.BuildOAS3()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "minimum")
	})

	t.Run("invalid pattern", func(t *testing.T) {
		b := New(parser.OASVersion320).
			SetTitle("Test API").
			SetVersion("1.0.0").
			AddParameter("BadParam", "query", "bad", "",
				WithParamPattern("[invalid"),
			)

		_, err := b.BuildOAS3()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "pattern")
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

// TestSchemaCopying tests that schema copying works correctly to avoid mutations.
func TestSchemaCopying(t *testing.T) {
	b := New(parser.OASVersion320).
		SetTitle("Test API").
		SetVersion("1.0.0")

	// Create two parameters with the same type but different constraints
	b.AddParameter("Param1", "query", "p1", int32(0),
		WithParamMinimum(1),
		WithParamMaximum(10),
	)
	b.AddParameter("Param2", "query", "p2", int32(0),
		WithParamMinimum(100),
		WithParamMaximum(1000),
	)

	doc, err := b.BuildOAS3()
	require.NoError(t, err)

	param1 := doc.Components.Parameters["Param1"]
	param2 := doc.Components.Parameters["Param2"]

	// Verify constraints are independent
	require.NotNil(t, param1.Schema.Minimum)
	require.NotNil(t, param2.Schema.Minimum)
	assert.Equal(t, 1.0, *param1.Schema.Minimum)
	assert.Equal(t, 100.0, *param2.Schema.Minimum)
	assert.Equal(t, 10.0, *param1.Schema.Maximum)
	assert.Equal(t, 1000.0, *param2.Schema.Maximum)
}

// Helper functions for pointer creation
func ptrFloat64(v float64) *float64 {
	return &v
}

func ptrInt(v int) *int {
	return &v
}

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
