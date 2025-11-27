package jsonhelpers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalWithExtras(t *testing.T) {
	t.Run("without extras", func(t *testing.T) {
		base := map[string]any{
			"name":  "test",
			"value": 42,
		}
		data, err := MarshalWithExtras(base, nil)
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "test", result["name"])
		assert.Equal(t, float64(42), result["value"])
		assert.Len(t, result, 2)
	})

	t.Run("with extras", func(t *testing.T) {
		base := map[string]any{
			"name": "test",
		}
		extras := map[string]any{
			"x-custom": "value",
			"x-count":  10,
		}
		data, err := MarshalWithExtras(base, extras)
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "test", result["name"])
		assert.Equal(t, "value", result["x-custom"])
		assert.Equal(t, float64(10), result["x-count"])
		assert.Len(t, result, 3)
	})

	t.Run("extras override base", func(t *testing.T) {
		base := map[string]any{
			"name": "original",
		}
		extras := map[string]any{
			"name": "overridden",
		}
		data, err := MarshalWithExtras(base, extras)
		require.NoError(t, err)

		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)

		assert.Equal(t, "overridden", result["name"])
	})

	t.Run("empty base and extras", func(t *testing.T) {
		data, err := MarshalWithExtras(map[string]any{}, nil)
		require.NoError(t, err)
		assert.Equal(t, "{}", string(data))
	})
}

func TestUnmarshalExtras(t *testing.T) {
	t.Run("all known fields", func(t *testing.T) {
		data := map[string]any{
			"name":  "test",
			"value": 42,
		}
		knownFields := map[string]bool{
			"name":  true,
			"value": true,
		}
		extras := UnmarshalExtras(data, knownFields)
		assert.Nil(t, extras)
	})

	t.Run("with extension fields", func(t *testing.T) {
		data := map[string]any{
			"name":     "test",
			"x-custom": "value",
			"x-count":  10,
		}
		knownFields := map[string]bool{
			"name": true,
		}
		extras := UnmarshalExtras(data, knownFields)
		require.NotNil(t, extras)
		assert.Equal(t, "value", extras["x-custom"])
		assert.Equal(t, 10, extras["x-count"])
		assert.Len(t, extras, 2)
	})

	t.Run("empty data", func(t *testing.T) {
		data := map[string]any{}
		knownFields := map[string]bool{"name": true}
		extras := UnmarshalExtras(data, knownFields)
		assert.Nil(t, extras)
	})

	t.Run("no known fields", func(t *testing.T) {
		data := map[string]any{
			"x-custom": "value",
		}
		knownFields := map[string]bool{}
		extras := UnmarshalExtras(data, knownFields)
		require.NotNil(t, extras)
		assert.Equal(t, "value", extras["x-custom"])
	})
}

func TestGetString(t *testing.T) {
	t.Run("existing string field", func(t *testing.T) {
		m := map[string]any{
			"name": "test",
		}
		result := GetString(m, "name")
		assert.Equal(t, "test", result)
		assert.NotContains(t, m, "name", "field should be deleted")
	})

	t.Run("non-existent field", func(t *testing.T) {
		m := map[string]any{}
		result := GetString(m, "name")
		assert.Equal(t, "", result)
	})

	t.Run("wrong type field", func(t *testing.T) {
		m := map[string]any{
			"name": 42,
		}
		result := GetString(m, "name")
		assert.Equal(t, "", result)
		assert.NotContains(t, m, "name", "field should be deleted even if wrong type")
	})
}

func TestGetBool(t *testing.T) {
	t.Run("existing true field", func(t *testing.T) {
		m := map[string]any{
			"flag": true,
		}
		result := GetBool(m, "flag")
		assert.True(t, result)
		assert.NotContains(t, m, "flag")
	})

	t.Run("existing false field", func(t *testing.T) {
		m := map[string]any{
			"flag": false,
		}
		result := GetBool(m, "flag")
		assert.False(t, result)
		assert.NotContains(t, m, "flag")
	})

	t.Run("non-existent field", func(t *testing.T) {
		m := map[string]any{}
		result := GetBool(m, "flag")
		assert.False(t, result)
	})

	t.Run("wrong type field", func(t *testing.T) {
		m := map[string]any{
			"flag": "true",
		}
		result := GetBool(m, "flag")
		assert.False(t, result)
		assert.NotContains(t, m, "flag")
	})
}

func TestGetInt(t *testing.T) {
	t.Run("existing int field", func(t *testing.T) {
		m := map[string]any{
			"count": float64(42),
		}
		result := GetInt(m, "count")
		assert.Equal(t, 42, result)
		assert.NotContains(t, m, "count")
	})

	t.Run("non-existent field", func(t *testing.T) {
		m := map[string]any{}
		result := GetInt(m, "count")
		assert.Equal(t, 0, result)
	})

	t.Run("wrong type field", func(t *testing.T) {
		m := map[string]any{
			"count": "42",
		}
		result := GetInt(m, "count")
		assert.Equal(t, 0, result)
		assert.NotContains(t, m, "count")
	})

	t.Run("negative int", func(t *testing.T) {
		m := map[string]any{
			"count": float64(-10),
		}
		result := GetInt(m, "count")
		assert.Equal(t, -10, result)
	})
}

func TestGetFloat64(t *testing.T) {
	t.Run("existing float field", func(t *testing.T) {
		m := map[string]any{
			"value": 3.14,
		}
		result := GetFloat64(m, "value")
		assert.Equal(t, 3.14, result)
		assert.NotContains(t, m, "value")
	})

	t.Run("non-existent field", func(t *testing.T) {
		m := map[string]any{}
		result := GetFloat64(m, "value")
		assert.Equal(t, 0.0, result)
	})

	t.Run("integer as float", func(t *testing.T) {
		m := map[string]any{
			"value": float64(42),
		}
		result := GetFloat64(m, "value")
		assert.Equal(t, 42.0, result)
	})
}

func TestGetStringSlice(t *testing.T) {
	t.Run("existing slice field", func(t *testing.T) {
		m := map[string]any{
			"items": []any{"a", "b", "c"},
		}
		result := GetStringSlice(m, "items")
		assert.Equal(t, []string{"a", "b", "c"}, result)
		assert.NotContains(t, m, "items")
	})

	t.Run("non-existent field", func(t *testing.T) {
		m := map[string]any{}
		result := GetStringSlice(m, "items")
		assert.Nil(t, result)
	})

	t.Run("empty slice", func(t *testing.T) {
		m := map[string]any{
			"items": []any{},
		}
		result := GetStringSlice(m, "items")
		assert.Equal(t, []string{}, result)
	})

	t.Run("mixed types in slice", func(t *testing.T) {
		m := map[string]any{
			"items": []any{"a", 42, "b", true, "c"},
		}
		result := GetStringSlice(m, "items")
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("wrong type field", func(t *testing.T) {
		m := map[string]any{
			"items": "not a slice",
		}
		result := GetStringSlice(m, "items")
		assert.Nil(t, result)
	})
}

func TestGetStringMap(t *testing.T) {
	t.Run("existing map field", func(t *testing.T) {
		m := map[string]any{
			"props": map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
		}
		result := GetStringMap(m, "props")
		assert.Equal(t, map[string]string{
			"key1": "value1",
			"key2": "value2",
		}, result)
		assert.NotContains(t, m, "props")
	})

	t.Run("non-existent field", func(t *testing.T) {
		m := map[string]any{}
		result := GetStringMap(m, "props")
		assert.Nil(t, result)
	})

	t.Run("empty map", func(t *testing.T) {
		m := map[string]any{
			"props": map[string]any{},
		}
		result := GetStringMap(m, "props")
		assert.Equal(t, map[string]string{}, result)
	})

	t.Run("mixed types in map", func(t *testing.T) {
		m := map[string]any{
			"props": map[string]any{
				"key1": "value1",
				"key2": 42,
				"key3": "value3",
			},
		}
		result := GetStringMap(m, "props")
		assert.Equal(t, map[string]string{
			"key1": "value1",
			"key3": "value3",
		}, result)
	})

	t.Run("wrong type field", func(t *testing.T) {
		m := map[string]any{
			"props": "not a map",
		}
		result := GetStringMap(m, "props")
		assert.Nil(t, result)
	})
}

func TestGetAny(t *testing.T) {
	t.Run("string value", func(t *testing.T) {
		m := map[string]any{
			"field": "value",
		}
		result := GetAny(m, "field")
		assert.Equal(t, "value", result)
		assert.NotContains(t, m, "field")
	})

	t.Run("complex object", func(t *testing.T) {
		obj := map[string]any{
			"nested": "value",
		}
		m := map[string]any{
			"field": obj,
		}
		result := GetAny(m, "field")
		assert.Equal(t, obj, result)
	})

	t.Run("nil value", func(t *testing.T) {
		m := map[string]any{
			"field": nil,
		}
		result := GetAny(m, "field")
		assert.Nil(t, result)
	})

	t.Run("non-existent field", func(t *testing.T) {
		m := map[string]any{}
		result := GetAny(m, "field")
		assert.Nil(t, result)
	})
}

func TestSetIfNotEmpty(t *testing.T) {
	t.Run("non-empty string", func(t *testing.T) {
		m := map[string]any{}
		SetIfNotEmpty(m, "name", "test")
		assert.Equal(t, "test", m["name"])
	})

	t.Run("empty string", func(t *testing.T) {
		m := map[string]any{}
		SetIfNotEmpty(m, "name", "")
		assert.NotContains(t, m, "name")
	})
}

func TestSetIfNotNil(t *testing.T) {
	t.Run("non-nil value", func(t *testing.T) {
		m := map[string]any{}
		SetIfNotNil(m, "field", "value")
		assert.Equal(t, "value", m["field"])
	})

	t.Run("nil value", func(t *testing.T) {
		m := map[string]any{}
		SetIfNotNil(m, "field", nil)
		assert.NotContains(t, m, "field")
	})

	t.Run("zero value is not nil", func(t *testing.T) {
		m := map[string]any{}
		SetIfNotNil(m, "field", 0)
		assert.Equal(t, 0, m["field"])
	})
}

func TestSetIfNotZero(t *testing.T) {
	t.Run("non-zero value", func(t *testing.T) {
		m := map[string]any{}
		SetIfNotZero(m, "count", 42)
		assert.Equal(t, 42, m["count"])
	})

	t.Run("zero value", func(t *testing.T) {
		m := map[string]any{}
		SetIfNotZero(m, "count", 0)
		assert.NotContains(t, m, "count")
	})

	t.Run("negative value", func(t *testing.T) {
		m := map[string]any{}
		SetIfNotZero(m, "count", -10)
		assert.Equal(t, -10, m["count"])
	})
}

func TestSetIfTrue(t *testing.T) {
	t.Run("true value", func(t *testing.T) {
		m := map[string]any{}
		SetIfTrue(m, "flag", true)
		assert.Equal(t, true, m["flag"])
	})

	t.Run("false value", func(t *testing.T) {
		m := map[string]any{}
		SetIfTrue(m, "flag", false)
		assert.NotContains(t, m, "flag")
	})
}

// Integration test: Round-trip through marshal/unmarshal
func TestMarshalUnmarshalRoundTrip(t *testing.T) {
	type TestStruct struct {
		Name  string
		Count int
		Extra map[string]any
	}

	original := TestStruct{
		Name:  "test",
		Count: 42,
		Extra: map[string]any{
			"x-custom": "value",
			"x-count":  10,
		},
	}

	// Marshal
	base := map[string]any{
		"name":  original.Name,
		"count": original.Count,
	}
	data, err := MarshalWithExtras(base, original.Extra)
	require.NoError(t, err)

	// Unmarshal
	var temp map[string]any
	err = json.Unmarshal(data, &temp)
	require.NoError(t, err)

	result := TestStruct{
		Name:  GetString(temp, "name"),
		Count: GetInt(temp, "count"),
	}
	result.Extra = UnmarshalExtras(temp, map[string]bool{
		"name":  true,
		"count": true,
	})

	// Verify
	assert.Equal(t, original.Name, result.Name)
	assert.Equal(t, original.Count, result.Count)
	// Note: Extra values that were ints become float64 after JSON round-trip
	assert.Equal(t, "value", result.Extra["x-custom"])
	assert.Equal(t, float64(10), result.Extra["x-count"])
}
