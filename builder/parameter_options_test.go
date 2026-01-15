package builder

import (
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

// TestWithParamType tests the WithParamType option.
func TestWithParamType(t *testing.T) {
	cfg := &paramConfig{}
	WithParamType("string")(cfg)
	assert.Equal(t, "string", cfg.typeOverride)
}

// TestWithParamFormat tests the WithParamFormat option.
func TestWithParamFormat(t *testing.T) {
	cfg := &paramConfig{}
	WithParamFormat("uuid")(cfg)
	assert.Equal(t, "uuid", cfg.formatOverride)
}

// TestWithParamSchema tests the WithParamSchema option.
func TestWithParamSchema(t *testing.T) {
	schema := &parser.Schema{Type: "array", Items: &parser.Schema{Type: "string"}}
	cfg := &paramConfig{}
	WithParamSchema(schema)(cfg)
	assert.Same(t, schema, cfg.schemaOverride)
}
