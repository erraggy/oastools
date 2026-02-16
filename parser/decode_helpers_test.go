package parser

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractExtensionsFromMap(t *testing.T) {
	m := map[string]any{
		"type":     "object",
		"x-custom": "value",
		"x-flag":   true,
		"title":    "Test",
	}
	ext := extractExtensionsFromMap(m)
	require.Len(t, ext, 2)
	assert.Equal(t, "value", ext["x-custom"])
	assert.Equal(t, true, ext["x-flag"])
}

func TestExtractExtensionsFromMap_NoExtensions(t *testing.T) {
	m := map[string]any{"type": "string"}
	assert.Nil(t, extractExtensionsFromMap(m))
}

func TestExtractExtensionsFromMap_EmptyMap(t *testing.T) {
	assert.Nil(t, extractExtensionsFromMap(map[string]any{}))
}

func TestExtractExtensionsFromMap_NilMap(t *testing.T) {
	assert.Nil(t, extractExtensionsFromMap(nil))
}

func TestMapGetStringSlice(t *testing.T) {
	m := map[string]any{
		"tags": []any{"pet", "store"},
	}
	assert.Equal(t, []string{"pet", "store"}, mapGetStringSlice(m, "tags"))
	assert.Nil(t, mapGetStringSlice(m, "missing"))
}

func TestMapGetStringSlice_WrongType(t *testing.T) {
	m := map[string]any{
		"tags": "not-an-array",
	}
	assert.Nil(t, mapGetStringSlice(m, "tags"))
}

func TestMapGetStringSlice_MixedTypes(t *testing.T) {
	m := map[string]any{
		"tags": []any{"valid", 42, "also-valid"},
	}
	// Only string items are included
	assert.Equal(t, []string{"valid", "also-valid"}, mapGetStringSlice(m, "tags"))
}

func TestMapGetFloat64Ptr(t *testing.T) {
	m := map[string]any{
		"fromJSON":   float64(3.14),
		"fromYAML":   int(42),
		"fromInt64":  int64(99),
		"fromUint64": uint64(18446744073709551615),
		"fromUint":   uint(500),
		"fromUint32": uint32(1000),
		"fromUint16": uint16(2000),
		"fromUint8":  uint8(255),
		"notANum":    "hello",
	}
	f := mapGetFloat64Ptr(m, "fromJSON")
	require.NotNil(t, f)
	assert.InDelta(t, 3.14, *f, 0.001)

	i := mapGetFloat64Ptr(m, "fromYAML")
	require.NotNil(t, i)
	assert.InDelta(t, 42.0, *i, 0.001)

	i64 := mapGetFloat64Ptr(m, "fromInt64")
	require.NotNil(t, i64)
	assert.InDelta(t, 99.0, *i64, 0.001)

	u64 := mapGetFloat64Ptr(m, "fromUint64")
	require.NotNil(t, u64)
	assert.InDelta(t, float64(uint64(18446744073709551615)), *u64, 1.0)

	u := mapGetFloat64Ptr(m, "fromUint")
	require.NotNil(t, u)
	assert.InDelta(t, 500.0, *u, 0.001)

	u32 := mapGetFloat64Ptr(m, "fromUint32")
	require.NotNil(t, u32)
	assert.InDelta(t, 1000.0, *u32, 0.001)

	u16 := mapGetFloat64Ptr(m, "fromUint16")
	require.NotNil(t, u16)
	assert.InDelta(t, 2000.0, *u16, 0.001)

	u8 := mapGetFloat64Ptr(m, "fromUint8")
	require.NotNil(t, u8)
	assert.InDelta(t, 255.0, *u8, 0.001)

	assert.Nil(t, mapGetFloat64Ptr(m, "notANum"))
	assert.Nil(t, mapGetFloat64Ptr(m, "missing"))
}

func TestMapGetIntPtr(t *testing.T) {
	m := map[string]any{
		"fromJSON":   float64(10),
		"fromYAML":   int(20),
		"fromInt64":  int64(30),
		"fromUint64": uint64(40),
		"fromUint":   uint(50),
		"fromUint32": uint32(60),
		"fromUint16": uint16(70),
		"fromUint8":  uint8(80),
		"notANum":    "hello",
	}
	f := mapGetIntPtr(m, "fromJSON")
	require.NotNil(t, f)
	assert.Equal(t, 10, *f)

	i := mapGetIntPtr(m, "fromYAML")
	require.NotNil(t, i)
	assert.Equal(t, 20, *i)

	i64 := mapGetIntPtr(m, "fromInt64")
	require.NotNil(t, i64)
	assert.Equal(t, 30, *i64)

	u64 := mapGetIntPtr(m, "fromUint64")
	require.NotNil(t, u64)
	assert.Equal(t, 40, *u64)

	u := mapGetIntPtr(m, "fromUint")
	require.NotNil(t, u)
	assert.Equal(t, 50, *u)

	u32 := mapGetIntPtr(m, "fromUint32")
	require.NotNil(t, u32)
	assert.Equal(t, 60, *u32)

	u16 := mapGetIntPtr(m, "fromUint16")
	require.NotNil(t, u16)
	assert.Equal(t, 70, *u16)

	u8 := mapGetIntPtr(m, "fromUint8")
	require.NotNil(t, u8)
	assert.Equal(t, 80, *u8)

	assert.Nil(t, mapGetIntPtr(m, "notANum"))
	assert.Nil(t, mapGetIntPtr(m, "missing"))

	// Overflow cases: values that exceed int range return nil
	overflow := map[string]any{
		"uint64Overflow": uint64(math.MaxUint64),
		"float64NaN":     math.NaN(),
		"float64PosInf":  math.Inf(1),
		"float64Large":   float64(1e19),
	}
	assert.Nil(t, mapGetIntPtr(overflow, "uint64Overflow"))
	assert.Nil(t, mapGetIntPtr(overflow, "float64NaN"))
	assert.Nil(t, mapGetIntPtr(overflow, "float64PosInf"))
	assert.Nil(t, mapGetIntPtr(overflow, "float64Large"))
}

func TestMapGetBoolPtr(t *testing.T) {
	m := map[string]any{
		"explode": true,
		"notBool": "true",
	}
	b := mapGetBoolPtr(m, "explode")
	require.NotNil(t, b)
	assert.True(t, *b)
	assert.Nil(t, mapGetBoolPtr(m, "notBool"))
	assert.Nil(t, mapGetBoolPtr(m, "missing"))
}

func TestMapGetBoolPtr_False(t *testing.T) {
	m := map[string]any{"flag": false}
	b := mapGetBoolPtr(m, "flag")
	require.NotNil(t, b)
	assert.False(t, *b)
}

func TestMapGetStringMap(t *testing.T) {
	m := map[string]any{
		"mapping": map[string]any{
			"dog": "#/components/schemas/Dog",
			"cat": "#/components/schemas/Cat",
		},
	}
	result := mapGetStringMap(m, "mapping")
	require.Len(t, result, 2)
	assert.Equal(t, "#/components/schemas/Dog", result["dog"])
	assert.Equal(t, "#/components/schemas/Cat", result["cat"])
	assert.Nil(t, mapGetStringMap(m, "missing"))
}

func TestMapGetStringMap_WrongType(t *testing.T) {
	m := map[string]any{
		"mapping": "not-a-map",
	}
	assert.Nil(t, mapGetStringMap(m, "mapping"))
}

func TestMapGetBoolMap(t *testing.T) {
	m := map[string]any{
		"vocab": map[string]any{
			"https://example.com/vocab/core":       true,
			"https://example.com/vocab/validation": false,
		},
	}
	result := mapGetBoolMap(m, "vocab")
	require.Len(t, result, 2)
	assert.True(t, result["https://example.com/vocab/core"])
	assert.False(t, result["https://example.com/vocab/validation"])
	assert.Nil(t, mapGetBoolMap(m, "missing"))
}

func TestMapGetBoolMap_WrongType(t *testing.T) {
	m := map[string]any{
		"vocab": "not-a-map",
	}
	assert.Nil(t, mapGetBoolMap(m, "vocab"))
}

func TestMapGetDependentRequired(t *testing.T) {
	m := map[string]any{
		"dependentRequired": map[string]any{
			"creditCard": []any{"billingAddress"},
			"name":       []any{"firstName", "lastName"},
		},
	}
	result := mapGetDependentRequired(m, "dependentRequired")
	require.Len(t, result, 2)
	assert.Equal(t, []string{"billingAddress"}, result["creditCard"])
	assert.Equal(t, []string{"firstName", "lastName"}, result["name"])
	assert.Nil(t, mapGetDependentRequired(m, "missing"))
}

func TestMapGetDependentRequired_WrongType(t *testing.T) {
	m := map[string]any{
		"dependentRequired": "not-a-map",
	}
	assert.Nil(t, mapGetDependentRequired(m, "dependentRequired"))
}

func TestDecodeSecurityRequirements(t *testing.T) {
	arr := []any{
		map[string]any{
			"api_key": []any{},
			"oauth":   []any{"read", "write"},
		},
	}
	reqs := decodeSecurityRequirements(arr)
	require.Len(t, reqs, 1)
	assert.Empty(t, reqs[0]["api_key"])
	assert.Equal(t, []string{"read", "write"}, reqs[0]["oauth"])
}

func TestDecodeSecurityRequirements_Nil(t *testing.T) {
	assert.Nil(t, decodeSecurityRequirements(nil))
}

func TestDecodeSecurityRequirements_NonMapItems(t *testing.T) {
	arr := []any{"not-a-map", 42}
	reqs := decodeSecurityRequirements(arr)
	assert.Empty(t, reqs)
}

func TestDecodeSecurityRequirements_Multiple(t *testing.T) {
	arr := []any{
		map[string]any{
			"bearerAuth": []any{},
		},
		map[string]any{
			"oauth2": []any{"admin"},
		},
	}
	reqs := decodeSecurityRequirements(arr)
	require.Len(t, reqs, 2)
	assert.Empty(t, reqs[0]["bearerAuth"])
	assert.Equal(t, []string{"admin"}, reqs[1]["oauth2"])
}

func TestDecodeSchemaOrBool_ArrayOfSchemas(t *testing.T) {
	// OAS 2.0 tuple validation: items can be an array of schemas
	input := []any{
		map[string]any{"type": "string"},
		map[string]any{"type": "integer"},
	}
	result := decodeSchemaOrBool(input)
	schemas, ok := result.([]*Schema)
	require.True(t, ok, "Expected []*Schema, got %T", result)
	require.Len(t, schemas, 2)
	assert.Equal(t, "string", schemas[0].Type)
	assert.Equal(t, "integer", schemas[1].Type)
}

func TestDecodeSchemaOrBool_ArrayWithNonMapElements(t *testing.T) {
	// Non-map elements in the array should be skipped
	input := []any{
		map[string]any{"type": "string"},
		"not-a-map",
		42,
		map[string]any{"type": "boolean"},
	}
	result := decodeSchemaOrBool(input)
	schemas, ok := result.([]*Schema)
	require.True(t, ok, "Expected []*Schema, got %T", result)
	require.Len(t, schemas, 2, "Non-map elements should be skipped")
	assert.Equal(t, "string", schemas[0].Type)
	assert.Equal(t, "boolean", schemas[1].Type)
}

func TestDecodeSchemaOrBool_EmptyArray(t *testing.T) {
	input := []any{}
	result := decodeSchemaOrBool(input)
	schemas, ok := result.([]*Schema)
	require.True(t, ok, "Expected []*Schema, got %T", result)
	assert.Empty(t, schemas)
}

func TestIsExtensionKey(t *testing.T) {
	assert.True(t, isExtensionKey("x-custom"))
	assert.True(t, isExtensionKey("x-"))
	assert.False(t, isExtensionKey("x"))
	assert.False(t, isExtensionKey(""))
	assert.False(t, isExtensionKey("type"))
	assert.False(t, isExtensionKey("X-Custom")) // case-sensitive
}
