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
			AddParameter("NameParam", "query", "name", string(""),
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
			AddParameter("StatusParam", "query", "status", string(""),
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
			AddParameter("NameParam", "query", "name", string(""),
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
			AddParameter("StatusParam", "query", "status", string(""),
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
				WithQueryParam("name", string(""),
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
				WithQueryParam("status", string(""),
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
				WithHeaderParam("X-Request-ID", string(""),
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
				WithCookieParam("session_id", string(""),
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
				WithQueryParam("status", string(""),
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
			WithQueryParam("q", string(""),
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

// Helper functions for pointer creation
func ptrFloat64(v float64) *float64 {
	return &v
}

func ptrInt(v int) *int {
	return &v
}
