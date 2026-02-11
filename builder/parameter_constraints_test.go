package builder

import (
	"testing"

	"github.com/erraggy/oastools/internal/testutil"
	"github.com/erraggy/oastools/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			cfg:      &paramConfig{minimum: testutil.Ptr(1.0)},
			expected: true,
		},
		{
			name:     "with maximum",
			cfg:      &paramConfig{maximum: testutil.Ptr(100.0)},
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
			cfg:      &paramConfig{multipleOf: testutil.Ptr(5.0)},
			expected: true,
		},
		{
			name:     "with minLength",
			cfg:      &paramConfig{minLength: testutil.Ptr(1)},
			expected: true,
		},
		{
			name:     "with maxLength",
			cfg:      &paramConfig{maxLength: testutil.Ptr(100)},
			expected: true,
		},
		{
			name:     "with pattern",
			cfg:      &paramConfig{pattern: "^[a-z]+$"},
			expected: true,
		},
		{
			name:     "with minItems",
			cfg:      &paramConfig{minItems: testutil.Ptr(1)},
			expected: true,
		},
		{
			name:     "with maxItems",
			cfg:      &paramConfig{maxItems: testutil.Ptr(10)},
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
		result := applyParamConstraintsToSchema(nil, &paramConfig{minimum: testutil.Ptr(1.0)})
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
			minimum:          testutil.Ptr(1.0),
			maximum:          testutil.Ptr(100.0),
			exclusiveMinimum: true,
			exclusiveMaximum: true,
			multipleOf:       testutil.Ptr(5.0),
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
			minLength: testutil.Ptr(1),
			maxLength: testutil.Ptr(50),
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
			minItems:    testutil.Ptr(1),
			maxItems:    testutil.Ptr(10),
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
			minimum:          testutil.Ptr(1.0),
			maximum:          testutil.Ptr(100.0),
			exclusiveMinimum: true,
			exclusiveMaximum: true,
			multipleOf:       testutil.Ptr(5.0),
			minLength:        testutil.Ptr(1),
			maxLength:        testutil.Ptr(50),
			pattern:          "^[a-z]+$",
			minItems:         testutil.Ptr(1),
			maxItems:         testutil.Ptr(10),
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

// TestValidateParamConstraints tests the validateParamConstraints helper.
func TestValidateParamConstraints(t *testing.T) {
	t.Run("valid constraints", func(t *testing.T) {
		cfg := &paramConfig{
			minimum:    testutil.Ptr(1.0),
			maximum:    testutil.Ptr(100.0),
			minLength:  testutil.Ptr(1),
			maxLength:  testutil.Ptr(50),
			minItems:   testutil.Ptr(0),
			maxItems:   testutil.Ptr(10),
			pattern:    "^[a-z]+$",
			multipleOf: testutil.Ptr(5.0),
		}
		err := validateParamConstraints(cfg)
		assert.NoError(t, err)
	})

	t.Run("minimum greater than maximum", func(t *testing.T) {
		cfg := &paramConfig{
			minimum: testutil.Ptr(100.0),
			maximum: testutil.Ptr(1.0),
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "minimum")
		assert.Contains(t, err.Error(), "maximum")
	})

	t.Run("minLength greater than maxLength", func(t *testing.T) {
		cfg := &paramConfig{
			minLength: testutil.Ptr(100),
			maxLength: testutil.Ptr(10),
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "minLength")
		assert.Contains(t, err.Error(), "maxLength")
	})

	t.Run("negative minLength", func(t *testing.T) {
		cfg := &paramConfig{
			minLength: testutil.Ptr(-1),
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "minLength")
		assert.Contains(t, err.Error(), "negative")
	})

	t.Run("negative maxLength", func(t *testing.T) {
		cfg := &paramConfig{
			maxLength: testutil.Ptr(-1),
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maxLength")
		assert.Contains(t, err.Error(), "negative")
	})

	t.Run("minItems greater than maxItems", func(t *testing.T) {
		cfg := &paramConfig{
			minItems: testutil.Ptr(10),
			maxItems: testutil.Ptr(1),
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "minItems")
		assert.Contains(t, err.Error(), "maxItems")
	})

	t.Run("negative minItems", func(t *testing.T) {
		cfg := &paramConfig{
			minItems: testutil.Ptr(-1),
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "minItems")
		assert.Contains(t, err.Error(), "negative")
	})

	t.Run("negative maxItems", func(t *testing.T) {
		cfg := &paramConfig{
			maxItems: testutil.Ptr(-1),
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "maxItems")
		assert.Contains(t, err.Error(), "negative")
	})

	t.Run("zero multipleOf", func(t *testing.T) {
		cfg := &paramConfig{
			multipleOf: testutil.Ptr(0.0),
		}
		err := validateParamConstraints(cfg)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "multipleOf")
		assert.Contains(t, err.Error(), "greater than 0")
	})

	t.Run("negative multipleOf", func(t *testing.T) {
		cfg := &paramConfig{
			multipleOf: testutil.Ptr(-5.0),
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
			minimum: testutil.Ptr(1.0), // Only minimum set, no maximum
		}
		err := validateParamConstraints(cfg)
		assert.NoError(t, err)
	})

	t.Run("multiple errors joined", func(t *testing.T) {
		cfg := &paramConfig{
			minimum:    testutil.Ptr(100.0),
			maximum:    testutil.Ptr(1.0), // min > max
			minLength:  testutil.Ptr(-1),  // negative
			maxLength:  testutil.Ptr(-2),  // negative
			multipleOf: testutil.Ptr(0.0), // not positive
			pattern:    "[invalid",        // invalid regex
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

// TestApplyTypeFormatOverrides tests the applyTypeFormatOverrides helper.
func TestApplyTypeFormatOverrides(t *testing.T) {
	t.Run("no overrides returns original", func(t *testing.T) {
		schema := &parser.Schema{Type: "string"}
		cfg := &paramConfig{}
		result := applyTypeFormatOverrides(schema, cfg)
		assert.Same(t, schema, result)
	})

	t.Run("type override only", func(t *testing.T) {
		schema := &parser.Schema{Type: "integer", Format: "int32"}
		cfg := &paramConfig{typeOverride: "string"}
		result := applyTypeFormatOverrides(schema, cfg)
		assert.Equal(t, "string", result.Type)
		assert.Equal(t, "int32", result.Format) // Format preserved
	})

	t.Run("format override only", func(t *testing.T) {
		schema := &parser.Schema{Type: "string"}
		cfg := &paramConfig{formatOverride: "uuid"}
		result := applyTypeFormatOverrides(schema, cfg)
		assert.Equal(t, "string", result.Type) // Type preserved
		assert.Equal(t, "uuid", result.Format)
	})

	t.Run("both type and format override", func(t *testing.T) {
		schema := &parser.Schema{Type: "integer", Format: "int32"}
		cfg := &paramConfig{
			typeOverride:   "string",
			formatOverride: "byte",
		}
		result := applyTypeFormatOverrides(schema, cfg)
		assert.Equal(t, "string", result.Type)
		assert.Equal(t, "byte", result.Format)
	})

	t.Run("schema override takes precedence", func(t *testing.T) {
		schema := &parser.Schema{Type: "integer", Format: "int32"}
		overrideSchema := &parser.Schema{Type: "number", Format: "decimal"}
		cfg := &paramConfig{
			typeOverride:   "string",       // Should be ignored
			formatOverride: "uuid",         // Should be ignored
			schemaOverride: overrideSchema, // Should be used
		}
		result := applyTypeFormatOverrides(schema, cfg)
		assert.Same(t, overrideSchema, result)
	})

	t.Run("nil schema with overrides", func(t *testing.T) {
		cfg := &paramConfig{
			typeOverride:   "string",
			formatOverride: "uuid",
		}
		result := applyTypeFormatOverrides(nil, cfg)
		assert.Equal(t, "string", result.Type)
		assert.Equal(t, "uuid", result.Format)
	})

	t.Run("nil schema with schema override", func(t *testing.T) {
		overrideSchema := &parser.Schema{Type: "array", Items: &parser.Schema{Type: "string"}}
		cfg := &paramConfig{schemaOverride: overrideSchema}
		result := applyTypeFormatOverrides(nil, cfg)
		assert.Same(t, overrideSchema, result)
	})
}

// TestApplyTypeFormatOverridesToOAS2Param tests the helper function directly.
func TestApplyTypeFormatOverridesToOAS2Param(t *testing.T) {
	t.Run("inferred type from schema only", func(t *testing.T) {
		param := &parser.Parameter{}
		schema := &parser.Schema{Type: "string", Format: ""}
		cfg := &paramConfig{}
		applyTypeFormatOverridesToOAS2Param(param, schema, cfg)
		assert.Equal(t, "string", param.Type)
		assert.Empty(t, param.Format)
	})

	t.Run("inferred type with format override", func(t *testing.T) {
		param := &parser.Parameter{}
		schema := &parser.Schema{Type: "string", Format: ""}
		cfg := &paramConfig{
			formatOverride: "uuid",
		}
		applyTypeFormatOverridesToOAS2Param(param, schema, cfg)
		assert.Equal(t, "string", param.Type)
		assert.Equal(t, "uuid", param.Format)
	})

	t.Run("schema override with string type", func(t *testing.T) {
		param := &parser.Parameter{}
		schema := &parser.Schema{Type: "string", Format: ""}
		cfg := &paramConfig{
			schemaOverride: &parser.Schema{Type: "number", Format: "decimal"},
		}
		applyTypeFormatOverridesToOAS2Param(param, schema, cfg)
		assert.Equal(t, "number", param.Type)
		assert.Equal(t, "decimal", param.Format)
	})

	t.Run("schema override with array type is ignored", func(t *testing.T) {
		param := &parser.Parameter{}
		schema := &parser.Schema{Type: "string", Format: ""}
		cfg := &paramConfig{
			schemaOverride: &parser.Schema{Type: []string{"string", "null"}, Format: ""},
		}
		applyTypeFormatOverridesToOAS2Param(param, schema, cfg)
		// Array type in schemaOverride cannot be assigned to string, so param.Type stays from inferred schema
		assert.Equal(t, "string", param.Type)
	})

	t.Run("type and format override without schema override", func(t *testing.T) {
		param := &parser.Parameter{}
		schema := &parser.Schema{Type: "integer", Format: "int32"}
		cfg := &paramConfig{
			typeOverride:   "number",
			formatOverride: "int64",
		}
		applyTypeFormatOverridesToOAS2Param(param, schema, cfg)
		assert.Equal(t, "number", param.Type)
		assert.Equal(t, "int64", param.Format)
	})

	t.Run("schema override takes precedence over type/format", func(t *testing.T) {
		param := &parser.Parameter{}
		schema := &parser.Schema{Type: "string", Format: ""}
		cfg := &paramConfig{
			typeOverride:   "string",
			formatOverride: "uuid",
			schemaOverride: &parser.Schema{Type: "boolean", Format: ""},
		}
		applyTypeFormatOverridesToOAS2Param(param, schema, cfg)
		assert.Equal(t, "boolean", param.Type)
		assert.Empty(t, param.Format)
	})

	t.Run("nil schema with overrides", func(t *testing.T) {
		param := &parser.Parameter{}
		cfg := &paramConfig{
			typeOverride:   "string",
			formatOverride: "uuid",
		}
		applyTypeFormatOverridesToOAS2Param(param, nil, cfg)
		assert.Equal(t, "string", param.Type)
		assert.Equal(t, "uuid", param.Format)
	})
}
